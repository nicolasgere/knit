package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var (
	binaryPath   string
	workspaceDir string
)

func TestMain(m *testing.M) {
	// Get absolute path to workspace
	wd, err := os.Getwd()
	if err != nil {
		panic("failed to get working directory: " + err.Error())
	}
	workspaceDir = filepath.Join(wd, "testdata", "workspace")

	// Build the binary once before all tests
	tmpDir, err := os.MkdirTemp("", "knit-e2e")
	if err != nil {
		panic("failed to create temp dir: " + err.Error())
	}
	binaryPath = filepath.Join(tmpDir, "knit")

	// Build from the parent directory (where main.go is)
	parentDir := filepath.Dir(wd)
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = parentDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		panic("failed to build binary: " + err.Error() + "\noutput: " + string(output))
	}

	// Run tests
	code := m.Run()

	// Cleanup
	os.RemoveAll(tmpDir)
	os.Exit(code)
}

// runKnit executes the knit binary with the given arguments
func runKnit(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func TestE2E_TestAllModules(t *testing.T) {
	output, err := runKnit(t, "test", "-p", workspaceDir)
	if err != nil {
		t.Fatalf("command failed: %v\noutput: %s", err, output)
	}

	// All 4 modules should be tested
	expectedModules := []string{
		"[example.com/core]",
		"[example.com/utils]",
		"[example.com/api]",
		"[example.com/app]",
	}

	for _, mod := range expectedModules {
		if !strings.Contains(output, mod) {
			t.Errorf("expected module %s in output, got:\n%s", mod, output)
		}
	}
}

func TestE2E_TestSingleTarget(t *testing.T) {
	output, err := runKnit(t, "test", "-p", workspaceDir, "-t", "example.com/core")
	if err != nil {
		t.Fatalf("command failed: %v\noutput: %s", err, output)
	}

	// Only core should be tested
	if !strings.Contains(output, "[example.com/core]") {
		t.Errorf("expected core module in output, got:\n%s", output)
	}

	// Other modules should NOT be tested
	unexpectedModules := []string{
		"[example.com/utils]",
		"[example.com/api]",
		"[example.com/app]",
	}

	for _, mod := range unexpectedModules {
		if strings.Contains(output, mod) {
			t.Errorf("unexpected module %s in output when targeting only core:\n%s", mod, output)
		}
	}
}

func TestE2E_TestTargetWithDependencies(t *testing.T) {
	t.Skip("Dependency flag removed - use 'knit affected --include-deps' instead")
}

func TestE2E_TestApiWithDependencies(t *testing.T) {
	t.Skip("Dependency flag removed - use 'knit affected --include-deps' instead")
}

func TestE2E_FmtAllModules(t *testing.T) {
	output, err := runKnit(t, "fmt", "-p", workspaceDir)
	if err != nil {
		t.Fatalf("command failed: %v\noutput: %s", err, output)
	}

	// All 4 modules should be formatted
	expectedModules := []string{
		"[example.com/core]",
		"[example.com/utils]",
		"[example.com/api]",
		"[example.com/app]",
	}

	for _, mod := range expectedModules {
		if !strings.Contains(output, mod) {
			t.Errorf("expected module %s in output, got:\n%s", mod, output)
		}
	}
}

func TestE2E_InstallAllModules(t *testing.T) {
	t.Skip("Install command removed - not useful for Go modules")
}

// setupGitRepo initializes a git repo, commits everything, then modifies specific files
func setupGitRepo(t *testing.T, dir string, filesToModify []string) func() {
	t.Helper()

	// Initialize git repo
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test")
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-m", "initial commit")

	// Modify the specified files
	for _, file := range filesToModify {
		filePath := filepath.Join(dir, file)
		f, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			t.Fatalf("failed to open file %s: %v", file, err)
		}
		_, err = f.WriteString("\n// modified for test\n")
		f.Close()
		if err != nil {
			t.Fatalf("failed to modify file %s: %v", file, err)
		}
	}

	// Return cleanup function
	return func() {
		os.RemoveAll(filepath.Join(dir, ".git"))
		// Restore modified files by removing the added comment
		for _, file := range filesToModify {
			filePath := filepath.Join(dir, file)
			data, _ := os.ReadFile(filePath)
			restored := strings.ReplaceAll(string(data), "\n// modified for test\n", "")
			os.WriteFile(filePath, []byte(restored), 0644)
		}
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\noutput: %s", args, err, output)
	}
}

func TestE2E_AffectedList(t *testing.T) {
	// Setup git repo with changes only in core and utils
	cleanup := setupGitRepo(t, workspaceDir, []string{
		"core/core.go",
		"utils/utils.go",
	})
	defer cleanup()

	output, err := runKnit(t, "affected", "-p", workspaceDir, "--base", "HEAD")
	if err != nil {
		t.Fatalf("command failed: %v\noutput: %s", err, output)
	}

	// Should list core and utils as affected
	if !strings.Contains(output, "example.com/core") {
		t.Errorf("expected example.com/core in output, got:\n%s", output)
	}
	if !strings.Contains(output, "example.com/utils") {
		t.Errorf("expected example.com/utils in output, got:\n%s", output)
	}

	// Should NOT list api or app (they weren't modified)
	if strings.Contains(output, "example.com/api") {
		t.Errorf("unexpected example.com/api in output:\n%s", output)
	}
	if strings.Contains(output, "example.com/app") {
		t.Errorf("unexpected example.com/app in output:\n%s", output)
	}
}

func TestE2E_AffectedGoArgsFormat(t *testing.T) {
	cleanup := setupGitRepo(t, workspaceDir, []string{
		"core/core.go",
	})
	defer cleanup()

	output, err := runKnit(t, "affected", "-p", workspaceDir, "--base", "HEAD", "-f", "go-args")
	if err != nil {
		t.Fatalf("command failed: %v\noutput: %s", err, output)
	}

	// Should be in go-args format: -p module1 -p module2
	if !strings.Contains(output, "-p example.com/core") {
		t.Errorf("expected '-p example.com/core' in output, got:\n%s", output)
	}
}

func TestE2E_AffectedGitHubMatrixFormat(t *testing.T) {
	cleanup := setupGitRepo(t, workspaceDir, []string{
		"api/api.go",
	})
	defer cleanup()

	output, err := runKnit(t, "affected", "-p", workspaceDir, "--base", "HEAD", "-f", "github-matrix")
	if err != nil {
		t.Fatalf("command failed: %v\noutput: %s", err, output)
	}

	// Should be JSON format
	if !strings.Contains(output, `"module"`) {
		t.Errorf("expected JSON with 'module' key in output, got:\n%s", output)
	}
	if !strings.Contains(output, "example.com/api") {
		t.Errorf("expected example.com/api in JSON output, got:\n%s", output)
	}
}

func TestE2E_AffectedWithDeps(t *testing.T) {
	// Modify only core - with --include-deps, should still only show core
	// since --include-deps shows dependencies OF affected modules, not dependents
	cleanup := setupGitRepo(t, workspaceDir, []string{
		"api/api.go",
	})
	defer cleanup()

	output, err := runKnit(t, "affected", "-p", workspaceDir, "--base", "HEAD", "--include-deps")
	if err != nil {
		t.Fatalf("command failed: %v\noutput: %s", err, output)
	}

	// api is affected, and it depends on utils and core
	// So with --include-deps we should see api, utils, core
	if !strings.Contains(output, "example.com/api") {
		t.Errorf("expected example.com/api in output, got:\n%s", output)
	}
	if !strings.Contains(output, "example.com/utils") {
		t.Errorf("expected example.com/utils (dependency of api) in output, got:\n%s", output)
	}
	if !strings.Contains(output, "example.com/core") {
		t.Errorf("expected example.com/core (dependency of api via utils) in output, got:\n%s", output)
	}

	// app should NOT be included (it's a dependent, not a dependency)
	if strings.Contains(output, "example.com/app") {
		t.Errorf("unexpected example.com/app in output:\n%s", output)
	}
}

func TestE2E_AffectedNoChanges(t *testing.T) {
	// Setup git repo with NO changes after commit
	cleanup := setupGitRepo(t, workspaceDir, []string{})
	defer cleanup()

	output, err := runKnit(t, "affected", "-p", workspaceDir, "--base", "HEAD", "-f", "github-matrix")
	if err != nil {
		t.Fatalf("command failed: %v\noutput: %s", err, output)
	}

	// Should output empty module array
	if !strings.Contains(output, `"module":[]`) {
		t.Errorf("expected empty module array in output, got:\n%s", output)
	}
}
