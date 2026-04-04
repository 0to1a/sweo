package workspace

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestRepo creates a temporary git repo with an initial commit.
func setupTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	cmds := [][]string{
		{"git", "init", "--initial-branch=main"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "setup cmd %v failed: %s", args, string(out))
	}

	// Create initial commit
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test\n"), 0644))
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = dir
	require.NoError(t, cmd.Run())
	cmd = exec.Command("git", "commit", "-m", "init")
	cmd.Dir = dir
	require.NoError(t, cmd.Run())

	return dir
}

func TestCreateAndDestroy(t *testing.T) {
	repoPath := setupTestRepo(t)
	worktreePath := filepath.Join(t.TempDir(), "wt-test")

	// Create worktree
	err := Create(repoPath, "main", "feat/issue-1", worktreePath)
	require.NoError(t, err)

	// Verify it exists
	assert.True(t, Exists(worktreePath))

	// Verify the branch was created
	cmd := exec.Command("git", "branch", "--list", "feat/issue-1")
	cmd.Dir = repoPath
	out, err := cmd.Output()
	require.NoError(t, err)
	assert.Contains(t, string(out), "feat/issue-1")

	// Verify README exists in worktree
	_, err = os.Stat(filepath.Join(worktreePath, "README.md"))
	assert.NoError(t, err)

	// Destroy worktree
	err = Destroy(repoPath, worktreePath)
	require.NoError(t, err)

	// Verify it's gone
	assert.False(t, Exists(worktreePath))
}

func TestCreateExistingBranch(t *testing.T) {
	repoPath := setupTestRepo(t)

	// Create the branch first
	cmd := exec.Command("git", "branch", "feat/existing")
	cmd.Dir = repoPath
	require.NoError(t, cmd.Run())

	worktreePath := filepath.Join(t.TempDir(), "wt-existing")

	// Create worktree for existing branch — should succeed with fallback
	err := Create(repoPath, "main", "feat/existing", worktreePath)
	require.NoError(t, err)

	assert.True(t, Exists(worktreePath))

	// Cleanup
	Destroy(repoPath, worktreePath)
}

func TestDestroyNonExistent(t *testing.T) {
	repoPath := setupTestRepo(t)
	err := Destroy(repoPath, "/nonexistent/worktree/path")
	// Should not error — the path doesn't exist, removal is a no-op
	assert.NoError(t, err)
}

func TestExistsNotAWorktree(t *testing.T) {
	dir := t.TempDir()
	assert.False(t, Exists(dir)) // dir exists but is not a git worktree
	assert.False(t, Exists("/nonexistent"))
}
