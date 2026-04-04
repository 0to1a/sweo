package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseKeyValue(t *testing.T) {
	input := "project=myproject\nbranch=feat/issue-1\nstatus=working\n"
	got := ParseKeyValue(input)
	assert.Equal(t, "myproject", got["project"])
	assert.Equal(t, "feat/issue-1", got["branch"])
	assert.Equal(t, "working", got["status"])
}

func TestParseKeyValueEdgeCases(t *testing.T) {
	// Value with = sign
	got := ParseKeyValue("url=https://github.com/org/repo?a=b\n")
	assert.Equal(t, "https://github.com/org/repo?a=b", got["url"])

	// Empty lines and whitespace
	got = ParseKeyValue("\n  \nkey=value\n\n")
	assert.Equal(t, "value", got["key"])
	assert.Len(t, got, 1)

	// Empty input
	got = ParseKeyValue("")
	assert.Empty(t, got)
}

func TestSerializeKeyValue(t *testing.T) {
	data := map[string]string{
		"branch":  "feat/issue-1",
		"project": "myproject",
		"status":  "working",
	}
	got := SerializeKeyValue(data)
	// Keys should be sorted
	assert.Equal(t, "branch=feat/issue-1\nproject=myproject\nstatus=working\n", got)
}

func TestWriteAndReadMetadata(t *testing.T) {
	dir := t.TempDir()
	sessionsDir := filepath.Join(dir, "sessions")

	data := map[string]string{
		"project": "myproject",
		"status":  "working",
		"branch":  "feat/issue-1",
	}

	err := WriteMetadata(sessionsDir, "ao-1", data)
	require.NoError(t, err)

	got, err := ReadMetadata(sessionsDir, "ao-1")
	require.NoError(t, err)
	assert.Equal(t, data, got)
}

func TestReadMetadataNonExistent(t *testing.T) {
	dir := t.TempDir()
	got, err := ReadMetadata(dir, "nonexistent")
	assert.NoError(t, err)
	assert.Nil(t, got)
}

func TestUpdateMetadata(t *testing.T) {
	dir := t.TempDir()
	sessionsDir := filepath.Join(dir, "sessions")

	// Write initial
	err := WriteMetadata(sessionsDir, "ao-1", map[string]string{
		"status":  "working",
		"branch":  "feat/issue-1",
		"project": "myproject",
	})
	require.NoError(t, err)

	// Update: change status, delete branch (empty value), add pr
	err = UpdateMetadata(sessionsDir, "ao-1", map[string]string{
		"status": "pr_open",
		"branch": "", // delete
		"pr":     "https://github.com/org/repo/pull/42",
	})
	require.NoError(t, err)

	got, err := ReadMetadata(sessionsDir, "ao-1")
	require.NoError(t, err)
	assert.Equal(t, "pr_open", got["status"])
	assert.Equal(t, "https://github.com/org/repo/pull/42", got["pr"])
	assert.Equal(t, "myproject", got["project"])
	_, hasBranch := got["branch"]
	assert.False(t, hasBranch)
}

func TestListSessions(t *testing.T) {
	dir := t.TempDir()
	sessionsDir := filepath.Join(dir, "sessions")
	require.NoError(t, os.MkdirAll(sessionsDir, 0755))

	// Create some session files
	for _, name := range []string{"ao-1", "ao-2", "ao-3"} {
		require.NoError(t, os.WriteFile(filepath.Join(sessionsDir, name), []byte("status=working\n"), 0644))
	}
	// Create a .tmp file (should be ignored)
	require.NoError(t, os.WriteFile(filepath.Join(sessionsDir, "ao-4.tmp"), []byte{}, 0644))
	// Create a dotfile (should be ignored)
	require.NoError(t, os.WriteFile(filepath.Join(sessionsDir, ".hidden"), []byte{}, 0644))

	ids, err := ListSessions(sessionsDir)
	require.NoError(t, err)
	assert.Equal(t, []string{"ao-1", "ao-2", "ao-3"}, ids)
}

func TestListSessionsEmpty(t *testing.T) {
	ids, err := ListSessions("/nonexistent/path")
	assert.NoError(t, err)
	assert.Nil(t, ids)
}

func TestReserveSessionID(t *testing.T) {
	dir := t.TempDir()
	sessionsDir := filepath.Join(dir, "sessions")

	// First reservation succeeds
	err := ReserveSessionID(sessionsDir, "ao-1")
	require.NoError(t, err)

	// Second reservation fails (already exists)
	err = ReserveSessionID(sessionsDir, "ao-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestDeleteMetadata(t *testing.T) {
	dir := t.TempDir()
	sessionsDir := filepath.Join(dir, "sessions")

	err := WriteMetadata(sessionsDir, "ao-1", map[string]string{"status": "done"})
	require.NoError(t, err)

	err = DeleteMetadata(sessionsDir, "ao-1")
	require.NoError(t, err)

	got, err := ReadMetadata(sessionsDir, "ao-1")
	assert.NoError(t, err)
	assert.Nil(t, got)
}

func TestValidateSessionID(t *testing.T) {
	assert.NoError(t, ValidateSessionID("ao-1"))
	assert.NoError(t, ValidateSessionID("my_session_123"))
	assert.Error(t, ValidateSessionID("../escape"))
	assert.Error(t, ValidateSessionID("has space"))
	assert.Error(t, ValidateSessionID(""))
}

func TestAtomicWrite(t *testing.T) {
	dir := t.TempDir()
	sessionsDir := filepath.Join(dir, "sessions")

	// Write initial data
	err := WriteMetadata(sessionsDir, "ao-1", map[string]string{"status": "working"})
	require.NoError(t, err)

	// Overwrite atomically
	err = WriteMetadata(sessionsDir, "ao-1", map[string]string{"status": "pr_open"})
	require.NoError(t, err)

	got, err := ReadMetadata(sessionsDir, "ao-1")
	require.NoError(t, err)
	assert.Equal(t, "pr_open", got["status"])

	// No .tmp files left behind
	entries, _ := os.ReadDir(sessionsDir)
	for _, e := range entries {
		assert.False(t, filepath.Ext(e.Name()) == ".tmp", "temp file left behind: %s", e.Name())
	}
}
