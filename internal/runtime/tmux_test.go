package runtime

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateName(t *testing.T) {
	assert.NoError(t, ValidateName("abc-123_test"))
	assert.NoError(t, ValidateName("a"))
	assert.Error(t, ValidateName("has space"))
	assert.Error(t, ValidateName("../escape"))
	assert.Error(t, ValidateName(""))
	assert.Error(t, ValidateName("semi;colon"))
}

// Integration tests: these require tmux to be installed.
// They are skipped in environments without tmux.

func skipIfNoTmux(t *testing.T) {
	t.Helper()
	if !hasTmuxBinary() {
		t.Skip("tmux not available, skipping integration test")
	}
}

func hasTmuxBinary() bool {
	err := runTmux("list-sessions")
	// Even "no server running" is fine — it means tmux exists
	return err == nil || err.Error() != ""
}

func TestTmuxIntegration(t *testing.T) {
	skipIfNoTmux(t)

	sessionName := "sweo-test-integration"

	// Cleanup in case of previous failed test
	KillSession(sessionName)

	// Create session
	err := NewSession(sessionName, "/tmp", nil)
	require.NoError(t, err)

	// Verify it exists
	assert.True(t, HasSession(sessionName))

	// Send a command
	err = SendKeys(sessionName, "echo hello-sweo-test")
	require.NoError(t, err)

	// Capture output (give tmux a moment)
	// Note: in real tests we'd wait, but this is a smoke test
	output, err := CapturePane(sessionName, 50)
	require.NoError(t, err)
	assert.NotEmpty(t, output)

	// Get pane TTY
	tty, err := GetPaneTTY(sessionName)
	require.NoError(t, err)
	assert.Contains(t, tty, "/dev/")

	// Kill session
	err = KillSession(sessionName)
	require.NoError(t, err)

	// Verify it's gone
	assert.False(t, HasSession(sessionName))
}

func TestSendKeysLongMessage(t *testing.T) {
	skipIfNoTmux(t)

	sessionName := "sweo-test-long"
	KillSession(sessionName)

	err := NewSession(sessionName, "/tmp", nil)
	require.NoError(t, err)
	defer KillSession(sessionName)

	// Send a long message (>200 chars)
	longMsg := "echo " + string(make([]byte, 250))
	err = SendKeys(sessionName, longMsg)
	// Should not error even if the message is long
	require.NoError(t, err)
}

func TestNewSessionWithEnv(t *testing.T) {
	skipIfNoTmux(t)

	sessionName := "sweo-test-env"
	KillSession(sessionName)

	env := map[string]string{
		"SWEO_TEST_VAR": "hello",
		"AO_SESSION_ID": "test-1",
	}

	err := NewSession(sessionName, "/tmp", env)
	require.NoError(t, err)
	defer KillSession(sessionName)

	assert.True(t, HasSession(sessionName))
}
