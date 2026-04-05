package runtime

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

const (
	tmuxTimeout         = 5 * time.Second
	longMessageThreshold = 200
)

var safeNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// ValidateName checks that a tmux session name is safe.
func ValidateName(name string) error {
	if !safeNamePattern.MatchString(name) {
		return fmt.Errorf("invalid tmux session name %q: must match [a-zA-Z0-9_-]+", name)
	}
	return nil
}

// NewSession creates a new detached tmux session.
func NewSession(name, cwd string, env map[string]string) error {
	if err := ValidateName(name); err != nil {
		return err
	}

	args := []string{"new-session", "-d", "-s", name}
	if cwd != "" {
		args = append(args, "-c", cwd)
	}
	for k, v := range env {
		args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
	}

	return runTmux(args...)
}

// HasSession returns true if the tmux session exists.
func HasSession(name string) bool {
	if err := ValidateName(name); err != nil {
		return false
	}
	err := runTmux("has-session", "-t", name)
	return err == nil
}

// KillSession terminates a tmux session.
func KillSession(name string) error {
	if err := ValidateName(name); err != nil {
		return err
	}
	return runTmux("kill-session", "-t", name)
}

// SendKeys sends text input to a tmux session.
// For short text (<200 chars, no newlines): uses send-keys -l.
// For long/multiline text: writes to temp file, uses load-buffer + paste-buffer.
func SendKeys(name, text string) error {
	if err := ValidateName(name); err != nil {
		return err
	}

	if len(text) < longMessageThreshold && !strings.Contains(text, "\n") {
		// Short message: send-keys literal + Enter
		if err := runTmux("send-keys", "-t", name, "-l", text); err != nil {
			return fmt.Errorf("send-keys: %w", err)
		}
		return runTmux("send-keys", "-t", name, "Enter")
	}

	// Long/multiline: temp file + load-buffer + paste-buffer
	tmpFile, err := os.CreateTemp("", "sweo-send-*.txt")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.WriteString(text); err != nil {
		tmpFile.Close()
		return fmt.Errorf("write temp file: %w", err)
	}
	tmpFile.Close()

	bufferName := "sweo-" + name
	if err := runTmux("load-buffer", "-b", bufferName, tmpPath); err != nil {
		return fmt.Errorf("load-buffer: %w", err)
	}

	if err := runTmux("paste-buffer", "-b", bufferName, "-t", name, "-d"); err != nil {
		return fmt.Errorf("paste-buffer: %w", err)
	}

	// Small delay before pressing Enter (tmux needs time to process paste)
	time.Sleep(300 * time.Millisecond)

	return runTmux("send-keys", "-t", name, "Enter")
}

// SendRawKey sends a single key (e.g. "Enter", "C-j", "Escape") to a tmux session.
func SendRawKey(name, key string) error {
	if err := ValidateName(name); err != nil {
		return err
	}
	return runTmux("send-keys", "-t", name, key)
}

// CapturePane captures the last N lines of output from a tmux pane.
func CapturePane(name string, lines int) (string, error) {
	if err := ValidateName(name); err != nil {
		return "", err
	}
	if lines <= 0 {
		lines = 50
	}

	return runTmuxOutput("capture-pane", "-t", name, "-p", "-S", fmt.Sprintf("-%d", lines))
}

// GetPaneTTY returns the TTY device of the tmux pane.
func GetPaneTTY(name string) (string, error) {
	if err := ValidateName(name); err != nil {
		return "", err
	}

	output, err := runTmuxOutput("list-panes", "-t", name, "-F", "#{pane_tty}")
	if err != nil {
		return "", fmt.Errorf("get pane tty: %w", err)
	}
	return strings.TrimSpace(output), nil
}

func runTmux(args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), tmuxTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "/usr/bin/tmux", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("tmux %s: %w (output: %s)", args[0], err, strings.TrimSpace(string(output)))
	}
	return nil
}

func runTmuxOutput(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), tmuxTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "/usr/bin/tmux", args...)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("tmux %s: %w", args[0], err)
	}
	return string(output), nil
}
