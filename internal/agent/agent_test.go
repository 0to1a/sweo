package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAgent(t *testing.T) {
	cc := New("claude-code")
	assert.Equal(t, "claude-code", cc.Name())

	cx := New("codex")
	assert.Equal(t, "codex", cx.Name())

	def := New("unknown")
	assert.Equal(t, "claude-code", def.Name())
}

func TestClaudeCodeBuildCommand(t *testing.T) {
	cc := &ClaudeCode{}
	cfg := LaunchConfig{
		SessionID: "ao-1",
		AgentRules: "Always write tests.",
	}

	cmd := cc.BuildLaunchCommand(cfg)
	assert.Contains(t, cmd, "claude")
	assert.Contains(t, cmd, "--dangerously-skip-permissions")
	assert.Contains(t, cmd, "--append-system-prompt")
	assert.Contains(t, cmd, "Always write tests.")
}

func TestClaudeCodeBuildCommandNoRules(t *testing.T) {
	cc := &ClaudeCode{}
	cfg := LaunchConfig{SessionID: "ao-1"}

	cmd := cc.BuildLaunchCommand(cfg)
	assert.Contains(t, cmd, "claude")
	assert.NotContains(t, cmd, "--append-system-prompt")
}

func TestClaudeCodeEnvironment(t *testing.T) {
	cc := &ClaudeCode{}
	env := cc.BuildEnvironment(LaunchConfig{
		SessionID: "ao-1",
		IssueID:   "42",
	})

	assert.Equal(t, "ao-1", env["AO_SESSION_ID"])
	assert.Equal(t, "42", env["AO_ISSUE_ID"])
}

func TestCodexBuildCommand(t *testing.T) {
	cx := &Codex{}
	cfg := LaunchConfig{
		SessionID:  "ao-2",
		Prompt:     "Fix the login bug",
		AgentRules: "Use conventional commits",
	}

	cmd := cx.BuildLaunchCommand(cfg)
	assert.Contains(t, cmd, "codex")
	assert.Contains(t, cmd, "--ask-for-approval")
	assert.Contains(t, cmd, "never")
	assert.Contains(t, cmd, "Fix the login bug")
	assert.Contains(t, cmd, "developer_instructions")
}

func TestCodexEnvironment(t *testing.T) {
	cx := &Codex{}
	env := cx.BuildEnvironment(LaunchConfig{
		SessionID: "ao-2",
		IssueID:   "99",
	})

	assert.Equal(t, "ao-2", env["AO_SESSION_ID"])
	assert.Equal(t, "99", env["AO_ISSUE_ID"])
	assert.Equal(t, "1", env["CODEX_DISABLE_UPDATE_CHECK"])
}

func TestShellQuote(t *testing.T) {
	assert.Equal(t, "'hello'", shellQuote("hello"))
	assert.Equal(t, "'it'\\''s'", shellQuote("it's"))
	assert.Equal(t, "'hello world'", shellQuote("hello world"))
}

func TestClaudeCodeSetupHooks(t *testing.T) {
	dir := t.TempDir()
	cc := &ClaudeCode{}

	err := cc.SetupHooks(dir, "/tmp/sessions", "ao-1")
	assert.NoError(t, err)

	// Check files were created
	assert.FileExists(t, dir+"/.claude/metadata-updater.sh")
	assert.FileExists(t, dir+"/.claude/settings.json")
}

func TestCodexSetupHooksNoop(t *testing.T) {
	cx := &Codex{}
	err := cx.SetupHooks("/tmp", "/tmp/sessions", "ao-1")
	assert.NoError(t, err)
}
