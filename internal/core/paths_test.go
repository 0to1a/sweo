package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateSessionPrefix(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"app", "app"},
		{"api", "api"},
		{"ao", "ao"},
		{"agent-orchestrator", "ao"},
		{"my_project", "mp"},
		{"my-cool-app", "mca"},
		{"PyTorch", "pt"},
		{"MyProject", "mp"},
		{"integrator", "int"},
		{"frontend", "fro"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := GenerateSessionPrefix(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGenerateTmuxName(t *testing.T) {
	name := GenerateTmuxName("abcdef123456", "ao", 3)
	assert.Equal(t, "abcdef123456-ao-3", name)
}

func TestGenerateSessionID(t *testing.T) {
	id := GenerateSessionID("ao", 5)
	assert.Equal(t, "ao-5", id)
}

func TestGetNextSessionNumber(t *testing.T) {
	existing := []string{"ao-1", "ao-2", "ao-5", "other-1"}
	assert.Equal(t, 6, GetNextSessionNumber(existing, "ao"))
	assert.Equal(t, 2, GetNextSessionNumber(existing, "other"))
	assert.Equal(t, 1, GetNextSessionNumber(existing, "new"))
	assert.Equal(t, 1, GetNextSessionNumber(nil, "ao"))
}
