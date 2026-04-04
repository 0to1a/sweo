package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// ClaudeCode implements the Agent interface for Claude Code.
type ClaudeCode struct{}

func (c *ClaudeCode) Name() string { return "claude-code" }

func (c *ClaudeCode) BuildLaunchCommand(cfg LaunchConfig) string {
	args := []string{"claude", "--dangerously-skip-permissions"}

	if cfg.AgentRules != "" {
		args = append(args, "--append-system-prompt", shellQuote(cfg.AgentRules))
	}

	return strings.Join(args, " ")
}

func (c *ClaudeCode) BuildEnvironment(cfg LaunchConfig) map[string]string {
	env := map[string]string{
		"AO_SESSION_ID": cfg.SessionID,
	}
	if cfg.IssueID != "" {
		env["AO_ISSUE_ID"] = cfg.IssueID
	}
	return env
}

func (c *ClaudeCode) IsProcessRunning(tmuxName string) (bool, error) {
	return isAgentOnTmux(tmuxName, `claude`)
}

// SetupHooks installs the PostToolUse hook for metadata tracking.
// Creates .claude/settings.json and .claude/metadata-updater.sh in the workspace.
func (c *ClaudeCode) SetupHooks(workspacePath, sessionsDir, sessionID string) error {
	claudeDir := filepath.Join(workspacePath, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		return fmt.Errorf("create .claude dir: %w", err)
	}

	// Write metadata updater script
	scriptPath := filepath.Join(claudeDir, "metadata-updater.sh")
	script := buildMetadataUpdaterScript(sessionsDir, sessionID)
	if err := os.WriteFile(scriptPath, []byte(script), 0700); err != nil {
		return fmt.Errorf("write metadata updater: %w", err)
	}

	// Write or update settings.json with PostToolUse hook
	settingsPath := filepath.Join(claudeDir, "settings.json")
	settings := buildClaudeSettings(scriptPath)
	if err := os.WriteFile(settingsPath, settings, 0644); err != nil {
		return fmt.Errorf("write settings.json: %w", err)
	}

	return nil
}

func buildMetadataUpdaterScript(sessionsDir, sessionID string) string {
	metadataFile := filepath.Join(sessionsDir, sessionID)

	return fmt.Sprintf(`#!/bin/bash
# sweo metadata updater — called by Claude Code PostToolUse hook
set -euo pipefail

METADATA_FILE="%s"

if [ ! -f "$METADATA_FILE" ]; then
  exit 0
fi

# Only process Bash tool calls
if [ "${TOOL_NAME:-}" != "Bash" ]; then
  exit 0
fi

# Skip failed commands
if [ "${EXIT_CODE:-0}" != "0" ]; then
  exit 0
fi

INPUT="${TOOL_INPUT:-}"

# Strip leading cd commands
CLEAN_INPUT=$(echo "$INPUT" | sed 's/^cd [^&]*&& *//')

# Detect gh pr create -> extract PR URL
if echo "$CLEAN_INPUT" | grep -q "gh pr create"; then
  PR_URL=$(echo "${TOOL_OUTPUT:-}" | grep -oE 'https://github\.com/[^/]+/[^/]+/pull/[0-9]+' | head -1)
  if [ -n "$PR_URL" ]; then
    PR_NUM=$(echo "$PR_URL" | grep -oE '[0-9]+$')
    sed -i "s|^pr=.*|pr=$PR_URL|" "$METADATA_FILE" 2>/dev/null || echo "pr=$PR_URL" >> "$METADATA_FILE"
    sed -i "s|^prNumber=.*|prNumber=$PR_NUM|" "$METADATA_FILE" 2>/dev/null || echo "prNumber=$PR_NUM" >> "$METADATA_FILE"
    sed -i "s|^status=.*|status=pr_open|" "$METADATA_FILE" 2>/dev/null
  fi
fi

# Detect git checkout -b / git switch -c -> extract branch
if echo "$CLEAN_INPUT" | grep -qE '(git checkout -b|git switch -c) '; then
  BRANCH=$(echo "$CLEAN_INPUT" | grep -oE '(checkout -b|switch -c) ([^ ]+)' | awk '{print $NF}')
  if [ -n "$BRANCH" ]; then
    sed -i "s|^branch=.*|branch=$BRANCH|" "$METADATA_FILE" 2>/dev/null || echo "branch=$BRANCH" >> "$METADATA_FILE"
  fi
fi

# Detect gh pr merge -> mark as merged
if echo "$CLEAN_INPUT" | grep -q "gh pr merge"; then
  sed -i "s|^status=.*|status=merged|" "$METADATA_FILE" 2>/dev/null
fi
`, metadataFile)
}

func buildClaudeSettings(scriptPath string) []byte {
	settings := map[string]any{
		"permissions": map[string]any{
			"allow": []string{},
			"deny":  []string{},
		},
		"hooks": map[string]any{
			"PostToolUse": []map[string]any{
				{
					"matcher": "Bash",
					"hooks": []map[string]string{
						{"type": "command", "command": scriptPath},
					},
				},
			},
		},
	}

	data, _ := json.MarshalIndent(settings, "", "  ")
	return data
}

// isAgentOnTmux checks if an agent process is running in a tmux session.
// It gets the pane TTY via tmux, then checks ps for the process pattern.
func isAgentOnTmux(tmuxName, processPattern string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get pane TTY
	cmd := exec.CommandContext(ctx, "tmux", "list-panes", "-t", tmuxName, "-F", "#{pane_tty}")
	out, err := cmd.Output()
	if err != nil {
		return false, nil // session doesn't exist
	}

	tty := strings.TrimSpace(string(out))
	tty = strings.TrimPrefix(tty, "/dev/")
	if tty == "" {
		return false, nil
	}

	// Check ps for the process on that TTY
	psCmd := exec.CommandContext(ctx, "ps", "-eo", "tty,args")
	psOut, err := psCmd.Output()
	if err != nil {
		return false, nil
	}

	re := regexp.MustCompile(processPattern)
	for _, line := range strings.Split(string(psOut), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if fields[0] == tty && re.MatchString(line) {
			return true, nil
		}
	}

	return false, nil
}

func shellQuote(s string) string {
	// Simple shell quoting: wrap in single quotes, escape single quotes
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
