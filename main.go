package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	analyzer "github.com/nicolasgere/knit/lib/analyser"

	"github.com/nicolasgere/knit/lib/git"
	"github.com/nicolasgere/knit/lib/runner"
	"github.com/nicolasgere/knit/lib/utils"
	"github.com/urfave/cli/v2"
)

var defaultDir = "."

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	setupSignalHandling(cancel)

	r := runner.NewRunner(ctx, 3)
	app := createCliApp(&r)

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func setupSignalHandling(cancel context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()
}

func createCliApp(r *runner.Runner) *cli.App {
	return &cli.App{
		Commands: []*cli.Command{
			createCommand("fmt", "Format every modules", "go fmt ./...", r),
			createCommand("test", "Test every modules", "go test ./...", r),
			createAffectedCommand(),
			createGraphCommand(),
		},
	}
}

// OutputFormat defines the format for the affected command output
type OutputFormat string

const (
	FormatList         OutputFormat = "list"
	FormatGoArgs       OutputFormat = "go-args"
	FormatGitHubMatrix OutputFormat = "github-matrix"
)

// createAffectedCommand creates the 'affected' command
func createAffectedCommand() *cli.Command {
	var (
		path         string
		base         string
		useMergeBase bool
		format       string
		includeDeps  bool
	)

	return &cli.Command{
		Name:  "affected",
		Usage: "List modules affected by changes since a git reference",
		Description: `Detect which modules have changed compared to a git reference.

Examples:
  knit affected                        # Compare against 'main' branch
  knit affected --base origin/main     # Compare against origin/main
  knit affected --merge-base           # Use merge-base (recommended for CI)
  knit affected -f go-args             # Output: -p module1 -p module2
  knit affected -f github-matrix       # Output: JSON matrix for GitHub Actions
  knit affected --include-deps         # Include dependencies of affected modules`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "path",
				Usage:       "Path to the workspace root",
				Aliases:     []string{"p"},
				Value:       ".",
				Destination: &path,
			},
			&cli.StringFlag{
				Name:        "base",
				Usage:       "Git reference to compare against (branch, tag, or commit)",
				Aliases:     []string{"b"},
				Value:       "main",
				Destination: &base,
			},
			&cli.BoolFlag{
				Name:        "merge-base",
				Usage:       "Compare against merge-base (common ancestor) - recommended for CI/PRs",
				Aliases:     []string{"m"},
				Destination: &useMergeBase,
			},
			&cli.StringFlag{
				Name:        "format",
				Usage:       "Output format: list (default), go-args, github-matrix",
				Aliases:     []string{"f"},
				Value:       "list",
				Destination: &format,
			},
			&cli.BoolFlag{
				Name:        "include-deps",
				Usage:       "Include dependencies of affected modules",
				Aliases:     []string{"d"},
				Destination: &includeDeps,
			},
		},
		Action: func(c *cli.Context) error {
			return runAffected(path, base, useMergeBase, OutputFormat(format), includeDeps)
		},
	}
}

func runAffected(path, base string, useMergeBase bool, format OutputFormat, includeDeps bool) error {
	// Get absolute path to workspace
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// List all modules in the workspace
	modules, err := analyzer.ListModule(absPath)
	if err != nil {
		return fmt.Errorf("failed to list modules: %w", err)
	}

	if len(modules) == 0 {
		return fmt.Errorf("no modules found in workspace")
	}

	// Get changed files
	changedFiles, err := git.GetChangedFiles(base, useMergeBase, absPath)
	if err != nil {
		return fmt.Errorf("failed to get changed files: %w", err)
	}

	// Get module directories
	moduleDirs := make([]string, len(modules))
	moduleDirToPath := make(map[string]string)
	for i, m := range modules {
		moduleDirs[i] = m.Dir
		moduleDirToPath[m.Dir] = m.Path
	}

	// Find affected module directories
	affectedDirs := git.FindAffectedModuleDirs(changedFiles, moduleDirs, absPath)

	// Convert to module paths
	affectedPaths := make([]string, 0, len(affectedDirs))
	for _, dir := range affectedDirs {
		if path, ok := moduleDirToPath[dir]; ok {
			affectedPaths = append(affectedPaths, path)
		}
	}

	// Include dependencies if requested
	if includeDeps && len(affectedPaths) > 0 {
		graph, err := analyzer.BuildDependencyGraph(modules)
		if err != nil {
			return fmt.Errorf("failed to build dependency graph: %w", err)
		}

		allAffected := make(map[string]bool)
		for _, p := range affectedPaths {
			allAffected[p] = true
		}

		// For each affected module, find its dependencies
		for _, p := range affectedPaths {
			deps, err := analyzer.GetDependencyPaths(graph, p)
			if err != nil {
				// Module might not have dependencies, continue
				continue
			}
			for _, dep := range deps {
				allAffected[dep] = true
			}
		}

		// Convert back to slice
		affectedPaths = make([]string, 0, len(allAffected))
		for p := range allAffected {
			affectedPaths = append(affectedPaths, p)
		}
	}

	// Output in the requested format
	return outputAffected(affectedPaths, format)
}

func outputAffected(modules []string, format OutputFormat) error {
	switch format {
	case FormatList:
		for _, m := range modules {
			fmt.Println(m)
		}

	case FormatGoArgs:
		// Output: -p module1 -p module2 ...
		var args []string
		for _, m := range modules {
			args = append(args, "-p", m)
		}
		fmt.Println(strings.Join(args, " "))

	case FormatGitHubMatrix:
		// Output: JSON for GitHub Actions matrix
		type MatrixOutput struct {
			Module []string `json:"module"`
		}
		matrix := MatrixOutput{Module: modules}
		if len(modules) == 0 {
			matrix.Module = []string{} // Ensure empty array, not null
		}
		data, err := json.Marshal(matrix)
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(data))

	default:
		return fmt.Errorf("unknown format: %s (use list, go-args, or github-matrix)", format)
	}

	return nil
}

// createGraphCommand creates the 'graph' command to visualize module dependencies
func createGraphCommand() *cli.Command {
	var (
		path   string
		format string
	)

	return &cli.Command{
		Name:  "graph",
		Usage: "Display the dependency graph of all modules in the workspace",
		Description: `Show all modules and their dependencies within the monorepo.

Examples:
  knit graph                    # Show dependency graph
  knit graph -f dot             # Output in DOT format (for Graphviz)
  knit graph -f json            # Output in JSON format`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "path",
				Usage:       "Path to the workspace root",
				Aliases:     []string{"p"},
				Value:       ".",
				Destination: &path,
			},
			&cli.StringFlag{
				Name:        "format",
				Usage:       "Output format: tree (default), dot, json",
				Aliases:     []string{"f"},
				Value:       "tree",
				Destination: &format,
			},
		},
		Action: func(c *cli.Context) error {
			return runGraph(path, format)
		},
	}
}

func runGraph(path, format string) error {
	// Get absolute path to workspace
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// List all modules in the workspace
	modules, err := analyzer.ListModule(absPath)
	if err != nil {
		return fmt.Errorf("failed to list modules: %w", err)
	}

	if len(modules) == 0 {
		return fmt.Errorf("no modules found in workspace")
	}

	// Build dependency graph
	g, err := analyzer.BuildDependencyGraph(modules)
	if err != nil {
		return fmt.Errorf("failed to build dependency graph: %w", err)
	}

	// Get adjacency map
	adjMap, err := (*g).AdjacencyMap()
	if err != nil {
		return fmt.Errorf("failed to get adjacency map: %w", err)
	}

	// Output in requested format
	switch format {
	case "tree":
		return outputGraphTree(modules, adjMap)
	case "dot":
		return outputGraphDot(modules, adjMap)
	case "json":
		return outputGraphJSON(modules, adjMap)
	default:
		return fmt.Errorf("unknown format: %s (use tree, dot, or json)", format)
	}
}

func outputGraphTree[T any](modules []analyzer.Module, adjMap map[string]map[string]T) error {
	fmt.Println("Module Dependency Graph")
	fmt.Println("=======================")
	fmt.Println()

	for _, m := range modules {
		deps := adjMap[m.Path]
		if len(deps) == 0 {
			fmt.Printf("📦 %s\n", m.Path)
			fmt.Println("   (no workspace dependencies)")
		} else {
			fmt.Printf("📦 %s\n", m.Path)
			depList := make([]string, 0, len(deps))
			for dep := range deps {
				depList = append(depList, dep)
			}
			for i, dep := range depList {
				if i == len(depList)-1 {
					fmt.Printf("   └── %s\n", dep)
				} else {
					fmt.Printf("   ├── %s\n", dep)
				}
			}
		}
		fmt.Println()
	}

	return nil
}

func outputGraphDot[T any](modules []analyzer.Module, adjMap map[string]map[string]T) error {
	fmt.Println("digraph dependencies {")
	fmt.Println("  rankdir=TB;")
	fmt.Println("  node [shape=box, style=rounded];")
	fmt.Println()

	// Add all nodes
	for _, m := range modules {
		// Use short name for display
		shortName := m.Path
		if idx := strings.LastIndex(m.Path, "/"); idx != -1 {
			shortName = m.Path[idx+1:]
		}
		fmt.Printf("  \"%s\" [label=\"%s\"];\n", m.Path, shortName)
	}
	fmt.Println()

	// Add edges
	for _, m := range modules {
		deps := adjMap[m.Path]
		for dep := range deps {
			fmt.Printf("  \"%s\" -> \"%s\";\n", m.Path, dep)
		}
	}

	fmt.Println("}")
	return nil
}

func outputGraphJSON[T any](modules []analyzer.Module, adjMap map[string]map[string]T) error {
	type ModuleNode struct {
		Path         string   `json:"path"`
		Dir          string   `json:"dir"`
		Dependencies []string `json:"dependencies"`
	}

	type GraphOutput struct {
		Modules []ModuleNode `json:"modules"`
	}

	output := GraphOutput{
		Modules: make([]ModuleNode, 0, len(modules)),
	}

	for _, m := range modules {
		deps := adjMap[m.Path]
		depList := make([]string, 0, len(deps))
		for dep := range deps {
			depList = append(depList, dep)
		}

		output.Modules = append(output.Modules, ModuleNode{
			Path:         m.Path,
			Dir:          m.Dir,
			Dependencies: depList,
		})
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

func createCommand(name, usage, cmd string, r *runner.Runner) *cli.Command {
	var target string
	var useColor bool
	var affected bool
	var base string

	return &cli.Command{
		Name:  name,
		Usage: usage,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "Path",
				Usage:       "Path to the root directory of the project",
				Aliases:     []string{"p"},
				Destination: &defaultDir,
			},
			&cli.StringFlag{
				Name:        "target",
				Usage:       "Targeted module",
				Aliases:     []string{"t"},
				Destination: &target,
			},
			&cli.BoolFlag{
				Name:        "affected",
				Usage:       "Run only on affected modules (since merge-base)",
				Aliases:     []string{"a"},
				Destination: &affected,
				Value:       false,
			},
			&cli.StringFlag{
				Name:        "base",
				Usage:       "Git reference to compare against when using --affected (default: main)",
				Aliases:     []string{"b"},
				Value:       "main",
				Destination: &base,
			},
			&cli.BoolFlag{
				Name:        "color",
				Usage:       "Enable colored output for better readability",
				Aliases:     []string{"c"},
				Destination: &useColor,
				Value:       false,
			},
		},
		Action: func(*cli.Context) error {
			// Enable color output if requested
			utils.SetColorEnabled(useColor)

			// Get absolute path to workspace
			absPath, err := filepath.Abs(defaultDir)
			if err != nil {
				return fmt.Errorf("failed to get absolute path: %w", err)
			}

			modules, err := analyzer.ListModule(absPath)
			if err != nil {
				return err
			}
			modulesToRun := modules

			// Filter by affected modules if requested
			if affected {
				changedFiles, err := git.GetChangedFiles(base, true, absPath)
				if err != nil {
					return fmt.Errorf("failed to get changed files: %w", err)
				}

				// Get module directories
				moduleDirs := make([]string, len(modules))
				moduleDirToPath := make(map[string]string)
				for i, m := range modules {
					moduleDirs[i] = m.Dir
					moduleDirToPath[m.Dir] = m.Path
				}

				// Find affected module directories
				affectedDirs := git.FindAffectedModuleDirs(changedFiles, moduleDirs, absPath)

				// Convert to module list
				affectedModules := make([]analyzer.Module, 0)
				affectedPaths := make(map[string]bool)
				for _, dir := range affectedDirs {
					if path, ok := moduleDirToPath[dir]; ok {
						affectedPaths[path] = true
					}
				}

				for _, m := range modules {
					if affectedPaths[m.Path] {
						affectedModules = append(affectedModules, m)
					}
				}
				modulesToRun = affectedModules

				if len(modulesToRun) == 0 {
					fmt.Println("No affected modules found")
					return nil
				}
			}

			// Filter by target if specified
			if target != "" {
				filteredModule := make([]analyzer.Module, 0)
				for _, m := range modulesToRun {
					if m.Path == target {
						filteredModule = append(filteredModule, m)
					}
				}
				modulesToRun = filteredModule
			}

			runOnModules(defaultDir, cmd, r, modulesToRun)
			return nil
		},
	}
}

func runOnModules(dir, cmd string, r *runner.Runner, modules []analyzer.Module) {

	tasks := createTasks(modules, cmd)
	tfs := r.RunTasks(tasks)

	var wg sync.WaitGroup
	wg.Add(len(tfs))

	for _, tf := range tfs {
		go handleTaskFuture(tf, &wg)
	}

	wg.Wait()
	return
}

func createTasks(modules []analyzer.Module, cmd string) []runner.Task {
	tasks := make([]runner.Task, len(modules))
	for i, module := range modules {
		tasks[i] = runner.Task{
			Id:   module.Path,
			Cmd:  cmd,
			Root: module.Dir,
		}
	}
	return tasks
}

func handleTaskFuture(tf *runner.TaskFuture, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case stdout, ok := <-tf.Stdout:
			handleOutput(tf.Id, stdout, ok, &tf.Stdout)
		case stderr, ok := <-tf.Stderr:
			handleOutput(tf.Id, stderr, ok, &tf.Stderr)
		case result := <-tf.Done:
			isSuccess := result.Status == 0
			statusMsg := fmt.Sprintf("Done with status %d", result.Status)
			if isSuccess {
				statusMsg = "✓ Done"
			} else {
				statusMsg = fmt.Sprintf("✗ Failed (exit %d)", result.Status)
			}
			utils.LogStatus(tf.Id, statusMsg, isSuccess)
			return
		}
	}
}

func handleOutput(id string, output []byte, ok bool, channel *chan []byte) {
	if !ok {
		*channel = nil
		return
	}
	if len(output) > 0 {
		utils.LogWithTaskId(id, string(output), utils.INFO)
	}
}
