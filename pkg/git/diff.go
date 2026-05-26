package git

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

type DiffResult struct {
	BaseBranch    string
	CurrentBranch string
	ChangedFiles  []string
	AddedFiles    []string
	ModifiedFiles []string
	DeletedFiles  []string
}

type CommitResult struct {
	CommitHash   string
	Author       string
	Message      string
	ChangedFiles []string
}

func GetDiffFiles(rootDir string, baseBranch string) (*DiffResult, error) {
	currentBranch, err := getCurrentBranch(rootDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get current branch: %w", err)
	}

	if baseBranch == "" {
		baseBranch = "main"
	}

	mergedBase, err := getMergeBase(rootDir, baseBranch)
	if err != nil {
		mergedBase = baseBranch
	}

	changedFiles, err := getDiffFileList(rootDir, mergedBase)
	if err != nil {
		return nil, fmt.Errorf("failed to get diff files: %w", err)
	}

	addedFiles, modifiedFiles, deletedFiles := classifyDiffFiles(rootDir, mergedBase)

	goFiles := filterGoFiles(changedFiles)

	return &DiffResult{
		BaseBranch:    baseBranch,
		CurrentBranch: currentBranch,
		ChangedFiles:  goFiles,
		AddedFiles:    filterGoFiles(addedFiles),
		ModifiedFiles: filterGoFiles(modifiedFiles),
		DeletedFiles:  filterGoFiles(deletedFiles),
	}, nil
}

func GetCommitFiles(rootDir string, commitHash string) (*CommitResult, error) {
	if commitHash == "" {
		return nil, fmt.Errorf("commit hash is required")
	}

	author, err := getCommitAuthor(rootDir, commitHash)
	if err != nil {
		author = "unknown"
	}

	message, err := getCommitMessage(rootDir, commitHash)
	if err != nil {
		message = ""
	}

	changedFiles, err := getCommitFileList(rootDir, commitHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit files: %w", err)
	}

	goFiles := filterGoFiles(changedFiles)

	return &CommitResult{
		CommitHash:   commitHash,
		Author:       author,
		Message:      message,
		ChangedFiles: goFiles,
	}, nil
}

func getCurrentBranch(rootDir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = rootDir
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func getMergeBase(rootDir string, branch string) (string, error) {
	cmd := exec.Command("git", "merge-base", branch, "HEAD")
	cmd.Dir = rootDir
	output, err := cmd.Output()
	if err != nil {
		cmd2 := exec.Command("git", "merge-base", "origin/"+branch, "HEAD")
		cmd2.Dir = rootDir
		output2, err2 := cmd2.Output()
		if err2 != nil {
			return "", err
		}
		return strings.TrimSpace(string(output2)), nil
	}
	return strings.TrimSpace(string(output)), nil
}

func getDiffFileList(rootDir string, baseRef string) ([]string, error) {
	cmd := exec.Command("git", "diff", "--name-only", baseRef)
	cmd.Dir = rootDir
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return parseFileList(string(output), rootDir), nil
}

func classifyDiffFiles(rootDir string, baseRef string) (added, modified, deleted []string) {
	cmd := exec.Command("git", "diff", "--name-status", baseRef)
	cmd.Dir = rootDir
	output, err := cmd.Output()
	if err != nil {
		return
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if len(line) < 2 {
			continue
		}
		status := line[0]
		filePath := strings.TrimSpace(line[1:])
		absPath := filepath.Join(rootDir, filePath)

		switch status {
		case 'A':
			added = append(added, absPath)
		case 'M':
			modified = append(modified, absPath)
		case 'D':
			deleted = append(deleted, absPath)
		case 'R':
			parts := strings.SplitN(filePath, "\t", 2)
			if len(parts) == 2 {
				added = append(added, filepath.Join(rootDir, parts[1]))
				deleted = append(deleted, filepath.Join(rootDir, parts[0]))
			}
		case 'C':
			parts := strings.SplitN(filePath, "\t", 2)
			if len(parts) == 2 {
				added = append(added, filepath.Join(rootDir, parts[1]))
			}
		}
	}
	return
}

func getCommitFileList(rootDir string, commitHash string) ([]string, error) {
	cmd := exec.Command("git", "diff-tree", "--no-commit-id", "--name-only", "-r", commitHash)
	cmd.Dir = rootDir
	output, err := cmd.Output()
	if err != nil {
		cmd2 := exec.Command("git", "show", "--name-only", "--pretty=format:", commitHash)
		cmd2.Dir = rootDir
		output2, err2 := cmd2.Output()
		if err2 != nil {
			return nil, err
		}
		return parseFileList(string(output2), rootDir), nil
	}
	return parseFileList(string(output), rootDir), nil
}

func getCommitAuthor(rootDir string, commitHash string) (string, error) {
	cmd := exec.Command("git", "show", "-s", "--format=%an", commitHash)
	cmd.Dir = rootDir
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func getCommitMessage(rootDir string, commitHash string) (string, error) {
	cmd := exec.Command("git", "show", "-s", "--format=%s", commitHash)
	cmd.Dir = rootDir
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func parseFileList(output string, rootDir string) []string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var files []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		absPath := filepath.Join(rootDir, line)
		files = append(files, absPath)
	}
	return files
}

func filterGoFiles(files []string) []string {
	var goFiles []string
	for _, f := range files {
		if strings.HasSuffix(f, ".go") && !strings.HasSuffix(f, "_test.go") {
			goFiles = append(goFiles, f)
		}
	}
	return goFiles
}
