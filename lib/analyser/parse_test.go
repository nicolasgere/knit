package analyzer

import (
	"fmt"
	"testing"
)

func TestListModules(t *testing.T) {
	modules, err := ListModule("./__playground__/workspace/")
	if err != nil {
		t.Error(err)
	}
	if len(modules) != 2 {
		t.Errorf("Expected 2 modules, got %d", len(modules))
	}
	for _, m := range modules {
		fmt.Printf("Module: %s at %s\n", m.Path, m.Dir)
	}
}

func TestListPackages(t *testing.T) {
	workspaceRoot := "./__playground__/workspace/"
	modules, err := ListModule(workspaceRoot)
	if err != nil {
		t.Fatal(err)
	}

	packages, err := ListPackages(workspaceRoot, modules)
	if err != nil {
		t.Error(err)
	}
	if len(packages) == 0 {
		t.Error("Expected at least one package")
	}
	for _, p := range packages {
		modPath := ""
		if p.Module != nil {
			modPath = p.Module.Path
		}
		fmt.Printf("Package: %s (module: %s) imports: %v\n", p.ImportPath, modPath, p.Imports)
	}
}

func TestBuildDependencyGraph(t *testing.T) {
	modules, err := ListModule("./__playground__/workspace/")
	if err != nil {
		t.Fatal(err)
	}

	graph, err := BuildDependencyGraph(modules)
	if err != nil {
		t.Fatal(err)
	}

	// Print the graph edges
	adjacencyMap, err := (*graph).AdjacencyMap()
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println("Dependency Graph:")
	for src, edges := range adjacencyMap {
		for dst := range edges {
			fmt.Printf("  %s -> %s\n", src, dst)
		}
	}
}
