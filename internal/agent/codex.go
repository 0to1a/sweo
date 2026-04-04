package agent

import (
	"os/exec"
	"strings"
)

// Codex implements the Agent interface for OpenAI Codex CLI.
type Codex struct{}

func (c *Codex) Name() string { return "codex" }

func (c *Codex) BuildLaunchCommand(cfg LaunchConfig) string {
	binary := resolveCodexBinary()

	args := []string{binary, "--ask-for-approval", "never"}

	if cfg.AgentRules != "" {
		args = append(args, "-c", "developer_instructions="+shellQuote(cfg.AgentRules))
	}

	// Prompt goes after --
	args = append(args, "--", shellQuote(cfg.Prompt))

	return strings.Join(args, " ")
}

func (c *Codex) BuildEnvironment(cfg LaunchConfig) map[string]string {
	env := map[string]string{
		"AO_SESSION_ID":            cfg.SessionID,
		"CODEX_DISABLE_UPDATE_CHECK": "1",
	}
	if cfg.IssueID != "" {
		env["AO_ISSUE_ID"] = cfg.IssueID
	}
	return env
}

func (c *Codex) IsProcessRunning(tmuxName string) (bool, error) {
	return isAgentOnTmux(tmuxName, `codex`)
}

// SetupHooks is a no-op for Codex — it uses PATH wrappers instead of hooks.
func (c *Codex) SetupHooks(workspacePath, sessionsDir, sessionID string) error {
	return nil
}

func resolveCodexBinary() string {
	// Try exec.LookPath first
	path, err := exec.LookPath("codex")
	if err == nil {
		return path
	}

	// Fallback paths
	candidates := []string{
		"/usr/local/bin/codex",
		"/opt/homebrew/bin/codex",
	}
	for _, p := range candidates {
		if _, err := exec.LookPath(p); err == nil {
			return p
		}
	}

	return "codex" // hope for the best
}

