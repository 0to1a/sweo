package engine

import (
	"testing"

	"github.com/0to1a/sweo/internal/github"
	"github.com/stretchr/testify/assert"
)

func TestBuildPrompt(t *testing.T) {
	issue := github.Issue{
		Number: 42,
		Title:  "Fix login bug",
		Body:   "Users can't log in when using SSO.",
		URL:    "https://github.com/org/repo/issues/42",
	}

	prompt := buildPrompt(issue)
	assert.Contains(t, prompt, "#42")
	assert.Contains(t, prompt, "Fix login bug")
	assert.Contains(t, prompt, "SSO")
	assert.Contains(t, prompt, "https://github.com/org/repo/issues/42")
	assert.Contains(t, prompt, "Create a PR")
}

func TestBuildPromptNoBody(t *testing.T) {
	issue := github.Issue{
		Number: 1,
		Title:  "Simple fix",
		URL:    "https://github.com/org/repo/issues/1",
	}

	prompt := buildPrompt(issue)
	assert.Contains(t, prompt, "#1")
	assert.Contains(t, prompt, "Simple fix")
	assert.NotContains(t, prompt, "Issue description:")
}
