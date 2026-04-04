package workspace

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

const gitTimeout = 30 * time.Second

// Create creates a git worktree at worktreePath on the given branch.
// It resolves the base ref from the repo's default branch.
// If the branch or worktree already exists from a previous session, it cleans up first.
func Create(repoPath, defaultBranch, branch, worktreePath string) error {
	// Aggressively clean up stale worktrees that might hold the branch
	cleanupBranch(repoPath, branch)

	// If the target worktree path still exists, remove it
	if _, err := os.Stat(worktreePath); err == nil {
		os.RemoveAll(worktreePath)
	}

	// Final prune + branch delete
	_ = runGit(repoPath, "worktree", "prune")
	_ = runGit(repoPath, "branch", "-D", branch)

	// Fetch origin (best-effort, ignore errors)
	if hasRemote(repoPath, "origin") {
		_ = runGit(repoPath, "fetch", "origin", "--quiet")
	}

	// Resolve base ref
	baseRef, err := resolveBaseRef(repoPath, defaultBranch)
	if err != nil {
		return fmt.Errorf("resolve base ref: %w", err)
	}

	// Create worktree with new branch
	return runGit(repoPath, "worktree", "add", "-b", branch, worktreePath, baseRef)
}

// cleanupBranch removes all worktrees using the given branch and deletes the branch.
func cleanupBranch(repoPath, branch string) {
	// List all worktrees, find any using this branch, and remove them
	out, err := runGitOutput(repoPath, "worktree", "list", "--porcelain")
	if err != nil {
		return
	}

	var currentPath string
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "worktree ") {
			currentPath = strings.TrimPrefix(line, "worktree ")
		}
		if strings.TrimSpace(line) == "branch refs/heads/"+branch && currentPath != "" && currentPath != repoPath {
			// This worktree uses our branch — remove it
			_ = runGit(repoPath, "worktree", "remove", "--force", currentPath)
			os.RemoveAll(currentPath)
			currentPath = ""
		}
	}

	_ = runGit(repoPath, "worktree", "prune")
	_ = runGit(repoPath, "branch", "-D", branch)
}

// Destroy removes a git worktree and its branch. Falls back to os.RemoveAll if git fails.
func Destroy(repoPath, worktreePath string) error {
	// Get the branch name before removing (for cleanup)
	branch := getWorktreeBranch(worktreePath)

	err := runGit(repoPath, "worktree", "remove", "--force", worktreePath)
	if err != nil {
		// Fallback: just remove the directory
		os.RemoveAll(worktreePath)
	}

	// Always prune stale worktree entries
	_ = runGit(repoPath, "worktree", "prune")

	// Delete the branch so it can be recreated on re-spawn
	if branch != "" {
		_ = runGit(repoPath, "branch", "-D", branch)
	}

	return nil
}

// getWorktreeBranch returns the branch checked out in a worktree, or "".
func getWorktreeBranch(worktreePath string) string {
	out, err := runGitOutput(worktreePath, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return ""
	}
	branch := strings.TrimSpace(out)
	if branch == "HEAD" {
		return "" // detached HEAD
	}
	return branch
}

// Exists checks if a path is a valid git worktree.
func Exists(worktreePath string) bool {
	info, err := os.Stat(worktreePath)
	if err != nil || !info.IsDir() {
		return false
	}
	err = runGit(worktreePath, "rev-parse", "--is-inside-work-tree")
	return err == nil
}

// resolveBaseRef tries refs in priority order:
// 1. origin/<defaultBranch>
// 2. refs/heads/<defaultBranch>
func resolveBaseRef(repoPath, defaultBranch string) (string, error) {
	candidates := []string{
		"origin/" + defaultBranch,
		"refs/heads/" + defaultBranch,
	}

	for _, ref := range candidates {
		if refExists(repoPath, ref) {
			return ref, nil
		}
	}

	return "", fmt.Errorf("no valid base ref found for branch %q in %s", defaultBranch, repoPath)
}

func refExists(repoPath, ref string) bool {
	err := runGit(repoPath, "rev-parse", "--verify", "--quiet", ref)
	return err == nil
}

func hasRemote(repoPath, name string) bool {
	output, err := runGitOutput(repoPath, "remote")
	if err != nil {
		return false
	}
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if strings.TrimSpace(line) == name {
			return true
		}
	}
	return false
}

func runGit(cwd string, args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "/usr/bin/git", args...)
	cmd.Dir = cwd
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git %s: %w (output: %s)", args[0], err, strings.TrimSpace(string(output)))
	}
	return nil
}

func runGitOutput(cwd string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "/usr/bin/git", args...)
	cmd.Dir = cwd
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", args[0], err)
	}
	return string(output), nil
}
