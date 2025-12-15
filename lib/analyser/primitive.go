package analyzer

// Module represents a Go module from `go list -m -json`
type Module struct {
	Path      string `json:"Path"`
	Main      bool   `json:"Main"`
	Dir       string `json:"Dir"`
	GoMod     string `json:"GoMod"`
	GoVersion string `json:"GoVersion"`
}

// Package represents a Go package from `go list -json ./...`
type Package struct {
	Dir        string   `json:"Dir"`
	ImportPath string   `json:"ImportPath"`
	Name       string   `json:"Name"`
	Module     *Module  `json:"Module"`
	Imports    []string `json:"Imports"`
}
