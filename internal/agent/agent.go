package agent

// LaunchConfig contains the parameters needed to launch an agent.
type LaunchConfig struct {
	SessionID     string
	ProjectID     string
	IssueID       string
	WorkspacePath string
	AgentRules    string
	Prompt        string
}

// Agent is the interface for AI coding agent implementations.
type Agent interface {
	// Name returns the agent identifier.
	Name() string

	// BuildLaunchCommand returns the shell command to start the agent.
	BuildLaunchCommand(cfg LaunchConfig) string

	// BuildEnvironment returns environment variables for the agent process.
	BuildEnvironment(cfg LaunchConfig) map[string]string

	// IsProcessRunning checks if the agent process is running in the tmux session.
	IsProcessRunning(tmuxName string) (bool, error)

	// SetupHooks installs any hooks needed for metadata tracking.
	SetupHooks(workspacePath, sessionsDir, sessionID string) error
}

// New creates an Agent by name. Supported: "claude-code", "codex".
func New(name string) Agent {
	switch name {
	case "claude-code":
		return &ClaudeCode{}
	case "codex":
		return &Codex{}
	default:
		return &ClaudeCode{} // default
	}
}
