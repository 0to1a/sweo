package config

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config is the top-level sweo configuration.
type Config struct {
	Port              int                      `yaml:"port"`
	DelayCheckIssue   int                      `yaml:"delayCheckIssue"`
	DelayCheckChanges int                      `yaml:"delayCheckChanges"`
	MaxInactiveMins   int                      `yaml:"maximumInactiveMins"`
	WebhookURL        string                   `yaml:"webhookUrl"`
	Projects          map[string]ProjectConfig `yaml:"projects"`

	// Computed at load time, not from YAML.
	Hash      string `yaml:"-"`
	ConfigDir string `yaml:"-"`
}

// ProjectConfig is per-project configuration.
type ProjectConfig struct {
	Repo          string `yaml:"repo"`
	Path          string `yaml:"path"`
	DefaultBranch string `yaml:"defaultBranch"`
	Agent         string `yaml:"agent"`
	AgentRules    string `yaml:"agentRules"`
	WaitingCI     *bool  `yaml:"waitingCI"`
}

// IsWaitingCI returns the effective waitingCI value (default true).
func (p *ProjectConfig) IsWaitingCI() bool {
	if p.WaitingCI == nil {
		return true
	}
	return *p.WaitingCI
}

var repoPattern = regexp.MustCompile(`^[a-zA-Z0-9._-]+/[a-zA-Z0-9._-]+$`)

// Load reads and validates the config from ~/.sweo/config.yaml.
// If the file does not exist, it creates a default config and returns ErrConfigCreated.
func Load() (*Config, error) {
	configDir := ConfigDir()
	configPath := filepath.Join(configDir, "config.yaml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			if createErr := createDefaultConfig(configDir, configPath); createErr != nil {
				return nil, fmt.Errorf("failed to create default config: %w", createErr)
			}
			return nil, ErrConfigCreated
		}
		return nil, fmt.Errorf("failed to read config %s: %w", configPath, err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	cfg.ConfigDir = configDir
	cfg.applyDefaults()
	cfg.Hash = computeHash(configDir)

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// ErrConfigCreated is returned when a default config was just created.
// The caller should notify the user and exit.
var ErrConfigCreated = fmt.Errorf("config created")

const defaultConfigTemplate = `# sweo — SWE-Orchestrator configuration
# Docs: https://github.com/0to1a/sweo

port: 3000                    # Web dashboard port
delayCheckIssue: 30           # Seconds between issue polling cycles
delayCheckChanges: 30         # Seconds between PR/CI/review polling cycles
maximumInactiveMins: 10       # Minutes of inactivity before marking session as stuck
webhookUrl: ""                # Outbound webhook URL (empty = disabled)

projects:
  # Add your project(s) here. Example:
  #
  # my-project:
  #   repo: "org/repo"                # GitHub org/repo (required)
  #   path: "/home/you/code/repo"     # Local checkout path (required)
  #   defaultBranch: "main"           # Base branch for worktrees
  #   agent: "claude-code"            # "claude-code" or "codex"
  #   agentRules: |                   # Custom instructions for the agent
  #     Always write tests.
  #   waitingCI: true                 # Wait for CI before advancing state
`

func createDefaultConfig(configDir, configPath string) error {
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	if err := os.WriteFile(configPath, []byte(defaultConfigTemplate), 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// ConfigDir returns ~/.sweo, expanding ~ to the user's home directory.
func ConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".sweo")
	}
	return filepath.Join(home, ".sweo")
}

func (c *Config) applyDefaults() {
	if c.Port == 0 {
		c.Port = 3000
	}
	if c.DelayCheckIssue == 0 {
		c.DelayCheckIssue = 30
	}
	if c.DelayCheckChanges == 0 {
		c.DelayCheckChanges = 30
	}
	if c.MaxInactiveMins == 0 {
		c.MaxInactiveMins = 10
	}

	for name, proj := range c.Projects {
		if proj.DefaultBranch == "" {
			proj.DefaultBranch = "main"
		}
		if proj.Agent == "" {
			proj.Agent = "claude-code"
		}
		proj.Path = expandHome(proj.Path)
		c.Projects[name] = proj
	}
}

// Validate checks the config for errors.
func (c *Config) Validate() error {
	if len(c.Projects) == 0 {
		return fmt.Errorf("config: at least one project is required")
	}

	for name, proj := range c.Projects {
		if proj.Repo == "" {
			return fmt.Errorf("config: project %q: repo is required", name)
		}
		if !repoPattern.MatchString(proj.Repo) {
			return fmt.Errorf("config: project %q: repo must be in 'org/name' format, got %q", name, proj.Repo)
		}
		if proj.Path == "" {
			return fmt.Errorf("config: project %q: path is required", name)
		}
		if proj.Agent != "claude-code" && proj.Agent != "codex" {
			return fmt.Errorf("config: project %q: agent must be 'claude-code' or 'codex', got %q", name, proj.Agent)
		}
	}

	return nil
}

func computeHash(dir string) string {
	h := sha256.Sum256([]byte(dir))
	return fmt.Sprintf("%x", h[:])[:12]
}

func expandHome(path string) string {
	if !strings.HasPrefix(path, "~/") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, path[2:])
}
