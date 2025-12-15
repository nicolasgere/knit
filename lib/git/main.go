package git

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// GetChangedFiles returns a list of files changed compared to a reference.
// If useMergeBase is true, it compares against the merge-base (common ancestor),
// which is useful in CI to detect changes in a PR/branch.
func GetChangedFiles(compareRef string, useMergeBase bool, dir string) ([]string, error) {
	var cmd *exec.Cmd

	if useMergeBase {
		// Find the merge-base (common ancestor) and compare against it
		// This is what you want in CI for PRs
		mergeBase, err := getMergeBase(compareRef, dir)
		if err != nil {
			return nil, fmt.Errorf("failed to get merge-base: %w", err)
		}
		cmd = exec.Command("git", "diff", "--name-only", mergeBase)
	} else {
		// Direct comparison against the reference
		cmd = exec.Command("git", "diff", "--name-only", compareRef)
	}

	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("error executing git diff: %w", err)
	}

	// Split the output into individual file paths
	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" {
		return []string{}, nil
	}

	return strings.Split(trimmed, "\n"), nil
}

// getMergeBase finds the common ancestor between HEAD and the given ref
func getMergeBase(ref string, dir string) (string, error) {
	cmd := exec.Command("git", "merge-base", ref, "HEAD")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git merge-base failed: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// GetAffectedRootDirectories returns root directories that have changed files.
// Deprecated: Use GetChangedFiles + FindAffectedModules instead.
func GetAffectedRootDirectories(compareBranch string, dir string) ([]string, error) {
	changedFiles, err := GetChangedFiles(compareBranch, false, dir)
	if err != nil {
		return nil, err
	}

	// Create a map to store unique root directories
	affectedRootDirs := make(map[string]bool)

	// Extract root directories from file paths
	for _, file := range changedFiles {
		rootDir := extractRootDirectory(file)
		if rootDir != "" {
			affectedRootDirs[rootDir] = true
		}
	}

	// Convert map keys to slice
	result := make([]string, 0, len(affectedRootDirs))
	for dir := range affectedRootDirs {
		result = append(result, dir)
	}

	return result, nil
}

// FindAffectedModuleDirs determines which module directories contain changed files.
// It takes the list of changed files and the list of module directories (absolute paths),
// and returns the directories of modules that have changes.
func FindAffectedModuleDirs(changedFiles []string, moduleDirs []string, workspaceRoot string) []string {
	// Sort module directories by length (longest first) so more specific paths match first
	// This prevents the root module from matching files in submodules
	sortedDirs := make([]string, len(moduleDirs))
	copy(sortedDirs, moduleDirs)
	sortByLengthDesc(sortedDirs)

	affectedDirs := make(map[string]bool)

	for _, file := range changedFiles {
		// Convert to absolute path
		absFile := filepath.Join(workspaceRoot, file)

		// Check which module this file belongs to (most specific first)
		for _, modDir := range sortedDirs {
			if strings.HasPrefix(absFile, modDir+string(filepath.Separator)) || absFile == modDir {
				affectedDirs[modDir] = true
				break
			}
		}
	}

	result := make([]string, 0, len(affectedDirs))
	for dir := range affectedDirs {
		result = append(result, dir)
	}
	return result
}

// sortByLengthDesc sorts strings by length in descending order
func sortByLengthDesc(strs []string) {
	for i := 0; i < len(strs)-1; i++ {
		for j := i + 1; j < len(strs); j++ {
			if len(strs[j]) > len(strs[i]) {
				strs[i], strs[j] = strs[j], strs[i]
			}
		}
	}
}

func extractRootDirectory(filePath string) string {
	parts := strings.SplitN(filePath, string(filepath.Separator), 2)
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}
