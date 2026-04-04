package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestParseValidConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(cfgPath, []byte(`
port: 4000
delayCheckIssue: 60
delayCheckChanges: 45
maximumInactiveMins: 15
webhookUrl: "https://example.com/hook"
projects:
  myproject:
    repo: "org/repo"
    path: "/tmp/myproject"
    defaultBranch: "develop"
    agent: "codex"
    agentRules: "Always write tests."
    waitingCI: false
`), 0644)
	require.NoError(t, err)

	cfg, err := loadFromPath(cfgPath)
	require.NoError(t, err)

	assert.Equal(t, 4000, cfg.Port)
	assert.Equal(t, 60, cfg.DelayCheckIssue)
	assert.Equal(t, 45, cfg.DelayCheckChanges)
	assert.Equal(t, 15, cfg.MaxInactiveMins)
	assert.Equal(t, "https://example.com/hook", cfg.WebhookURL)

	proj := cfg.Projects["myproject"]
	assert.Equal(t, "org/repo", proj.Repo)
	assert.Equal(t, "/tmp/myproject", proj.Path)
	assert.Equal(t, "develop", proj.DefaultBranch)
	assert.Equal(t, "codex", proj.Agent)
	assert.Equal(t, "Always write tests.", proj.AgentRules)
	assert.False(t, proj.IsWaitingCI())
}

func TestDefaults(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(cfgPath, []byte(`
projects:
  test:
    repo: "org/repo"
    path: "/tmp/test"
`), 0644)
	require.NoError(t, err)

	cfg, err := loadFromPath(cfgPath)
	require.NoError(t, err)

	assert.Equal(t, 3000, cfg.Port)
	assert.Equal(t, 30, cfg.DelayCheckIssue)
	assert.Equal(t, 30, cfg.DelayCheckChanges)
	assert.Equal(t, 10, cfg.MaxInactiveMins)
	assert.Equal(t, "", cfg.WebhookURL)

	proj := cfg.Projects["test"]
	assert.Equal(t, "main", proj.DefaultBranch)
	assert.Equal(t, "claude-code", proj.Agent)
	assert.True(t, proj.IsWaitingCI())
}

func TestValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr string
	}{
		{
			name:    "no projects",
			yaml:    `port: 3000`,
			wantErr: "at least one project",
		},
		{
			name: "missing repo",
			yaml: `
projects:
  test:
    path: "/tmp/test"`,
			wantErr: "repo is required",
		},
		{
			name: "bad repo format",
			yaml: `
projects:
  test:
    repo: "just-a-name"
    path: "/tmp/test"`,
			wantErr: "org/name",
		},
		{
			name: "missing path",
			yaml: `
projects:
  test:
    repo: "org/repo"`,
			wantErr: "path is required",
		},
		{
			name: "bad agent",
			yaml: `
projects:
  test:
    repo: "org/repo"
    path: "/tmp/test"
    agent: "gpt"`,
			wantErr: "claude-code' or 'codex",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			cfgPath := filepath.Join(dir, "config.yaml")
			err := os.WriteFile(cfgPath, []byte(tt.yaml), 0644)
			require.NoError(t, err)

			_, err = loadFromPath(cfgPath)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestComputeHash(t *testing.T) {
	h := computeHash("/home/user/.sweo")
	assert.Len(t, h, 12)
	assert.Regexp(t, `^[0-9a-f]{12}$`, h)

	// Deterministic
	h2 := computeHash("/home/user/.sweo")
	assert.Equal(t, h, h2)

	// Different input -> different hash
	h3 := computeHash("/other/path")
	assert.NotEqual(t, h, h3)
}

func TestExpandHome(t *testing.T) {
	home, _ := os.UserHomeDir()

	assert.Equal(t, filepath.Join(home, "projects"), expandHome("~/projects"))
	assert.Equal(t, "/absolute/path", expandHome("/absolute/path"))
	assert.Equal(t, "relative/path", expandHome("relative/path"))
}

func TestCreateDefaultConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")

	err := createDefaultConfig(dir, configPath)
	require.NoError(t, err)

	// File should exist
	_, err = os.Stat(configPath)
	require.NoError(t, err)

	// File should contain the template
	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "port: 3000")
	assert.Contains(t, content, "projects:")
	assert.Contains(t, content, "repo:")
	assert.Contains(t, content, "claude-code")
}

// loadFromPath is a test helper that loads config from an arbitrary path.
func loadFromPath(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	cfg.ConfigDir = filepath.Dir(path)
	cfg.applyDefaults()
	cfg.Hash = computeHash(cfg.ConfigDir)

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}
