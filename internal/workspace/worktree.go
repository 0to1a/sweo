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
func Create(repoPath, defaultBranch, branch, worktreePath string) error {
	// Fetch origin (best-effort, ignore errors)
	if hasRemote(repoPath, "origin") {
		_ = runGit(repoPath, "fetch", "origin", "--quiet")
	}

	// Resolve base ref
	baseRef, err := resolveBaseRef(repoPath, defaultBranch)
	if err != nil {
		return fmt.Errorf("resolve base ref: %w", err)
	}

	// Try creating worktree with new branch
	err = runGit(repoPath, "worktree", "add", "-b", branch, worktreePath, baseRef)
	if err != nil {
		// Branch might already exist — try without -b and checkout
		if strings.Contains(err.Error(), "already exists") {
			err2 := runGit(repoPath, "worktree", "add", worktreePath, baseRef)
			if err2 != nil {
				return fmt.Errorf("worktree add (fallback): %w", err2)
			}
			return runGit(worktreePath, "checkout", branch)
		}
		return fmt.Errorf("worktree add: %w", err)
	}

	return nil
}

// Destroy removes a git worktree. Falls back to os.RemoveAll if git fails.
func Destroy(repoPath, worktreePath string) error {
	err := runGit(repoPath, "worktree", "remove", "--force", worktreePath)
	if err != nil {
		// Fallback: just remove the directory
		if rmErr := os.RemoveAll(worktreePath); rmErr != nil {
			return fmt.Errorf("git worktree remove failed: %w; fallback rm also failed: %v", err, rmErr)
		}
		// Prune stale worktree entries
		_ = runGit(repoPath, "worktree", "prune")
	}
	return nil
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

	cmd := exec.CommandContext(ctx, "git", args...)
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

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = cwd
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", args[0], err)
	}
	return string(output), nil
}
