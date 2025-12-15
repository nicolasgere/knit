package analyzer

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/dominikbraun/graph"
)

// ListModule discovers all modules in a Go workspace using `go list -m -json`
func ListModule(dir string) (modules []Module, err error) {
	output, err := runCommand(dir, "go list -m -json")
	if err != nil {
		return
	}

	d := json.NewDecoder(strings.NewReader(output))
	for d.More() {
		var m Module
		if err = d.Decode(&m); err != nil {
			return
		}
		modules = append(modules, m)
	}
	return
}

// ListPackages lists all packages in the workspace using `go list -json`
// For workspaces, it queries each module directory explicitly
func ListPackages(workspaceRoot string, modules []Module) (packages []Package, err error) {
	if len(modules) == 0 {
		return nil, nil
	}

	// Ensure workspaceRoot is absolute for consistent path handling
	absWorkspaceRoot, err := filepath.Abs(workspaceRoot)
	if err != nil {
		absWorkspaceRoot = workspaceRoot
	}

	// Build the list of module paths to query (relative to workspace root)
	var patterns []string
	for _, m := range modules {
		// Get relative path from workspace root to module
		relPath, err := filepath.Rel(absWorkspaceRoot, m.Dir)
		if err != nil {
			// Fallback: use last component of the path
			relPath = filepath.Base(m.Dir)
		}
		patterns = append(patterns, "./"+relPath+"/...")
	}

	// Query all modules in a single go list command
	cmd := "go list -json " + strings.Join(patterns, " ")
	output, err := runCommand(absWorkspaceRoot, cmd)
	if err != nil {
		return nil, err
	}

	d := json.NewDecoder(strings.NewReader(output))
	for d.More() {
		var p Package
		if err = d.Decode(&p); err != nil {
			return nil, err
		}
		packages = append(packages, p)
	}
	return packages, nil
}

// BuildDependencyGraph builds a directed acyclic graph of module dependencies
// by analyzing package imports across the workspace
func BuildDependencyGraph(modules []Module) (*graph.Graph[string, string], error) {
	g := graph.New(graph.StringHash, graph.Directed(), graph.Acyclic())

	// Build a set of workspace module paths for quick lookup
	workspaceModules := make(map[string]bool)
	for _, m := range modules {
		workspaceModules[m.Path] = true
		if err := g.AddVertex(m.Path); err != nil {
			// Vertex may already exist, ignore
		}
	}

	// We need a workspace root - use the first module's parent or find go.work
	// For now, use the directory of the first module (assumes it's the main module)
	if len(modules) == 0 {
		return &g, nil
	}

	// Find workspace root by looking for go.work or use the main module's dir
	workspaceRoot := findWorkspaceRoot(modules)

	// Get all packages in the workspace
	packages, err := ListPackages(workspaceRoot, modules)
	if err != nil {
		return nil, fmt.Errorf("failed to list packages: %w", err)
	}

	// Build a map: import path prefix -> module path
	// This helps us determine which module an import belongs to
	importToModule := make(map[string]string)
	for _, pkg := range packages {
		if pkg.Module != nil {
			importToModule[pkg.ImportPath] = pkg.Module.Path
		}
	}

	// Track dependencies: module -> set of dependent modules
	moduleDeps := make(map[string]map[string]bool)
	for _, m := range modules {
		moduleDeps[m.Path] = make(map[string]bool)
	}

	// Analyze each package's imports
	for _, pkg := range packages {
		if pkg.Module == nil {
			continue
		}
		srcModule := pkg.Module.Path

		// Skip if source module is not in our workspace
		if !workspaceModules[srcModule] {
			continue
		}

		for _, imp := range pkg.Imports {
			// Find which module this import belongs to
			depModule := findModuleForImport(imp, importToModule, workspaceModules)
			if depModule != "" && depModule != srcModule {
				moduleDeps[srcModule][depModule] = true
			}
		}
	}

	// Add edges to the graph
	for srcModule, deps := range moduleDeps {
		for depModule := range deps {
			if err := g.AddEdge(srcModule, depModule); err != nil {
				// Edge may already exist or would create cycle, ignore
			}
		}
	}

	return &g, nil
}

// findWorkspaceRoot finds the workspace root directory by looking for go.work
// It searches upward from the first module's directory
func findWorkspaceRoot(modules []Module) string {
	if len(modules) == 0 {
		return "."
	}

	// Start from the first module's directory and search upward for go.work
	dir := modules[0].Dir
	for {
		goWorkPath := filepath.Join(dir, "go.work")
		if _, err := os.Stat(goWorkPath); err == nil {
			return dir
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root, no go.work found
			break
		}
		dir = parent
	}

	// Fallback: use the first module's directory
	return modules[0].Dir
}

// findModuleForImport determines which workspace module an import path belongs to
func findModuleForImport(importPath string, importToModule map[string]string, workspaceModules map[string]bool) string {
	// Direct match from our package scan
	if mod, ok := importToModule[importPath]; ok && workspaceModules[mod] {
		return mod
	}

	// Check if import path starts with any workspace module path
	for modPath := range workspaceModules {
		if strings.HasPrefix(importPath, modPath+"/") || importPath == modPath {
			return modPath
		}
	}

	return ""
}

func GetDependencyPaths(g *graph.Graph[string, string], vertex string) ([]string, error) {
	var dependencyPaths []string

	// Define a visitor function for DFS
	visitor := func(v string) bool {
		if v != vertex { // Don't include the starting vertex itself
			dependencyPaths = append(dependencyPaths, v)
		}
		return false // Continue traversal
	}

	// Perform DFS
	err := graph.DFS(*g, vertex, visitor)
	if err != nil {
		return nil, fmt.Errorf("failed to perform DFS for vertex %s: %w", vertex, err)
	}

	return dependencyPaths, nil
}

func runCommand(dir, command string) (output string, err error) {
	cmd := exec.Command("sh", "-c", command)
	cmd.Dir = dir
	var outputBytes []byte
	outputBytes, err = cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("command execution failed: %w\nOutput: %s", err, outputBytes)
	}
	return string(outputBytes), nil
}
