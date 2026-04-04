package engine

import (
	_ "embed"
	"strings"
)

//go:embed prompt.md
var baseRules string

// buildFullRules combines the embedded base rules with user-defined agentRules.
func buildFullRules(userRules string) string {
	parts := []string{baseRules}
	if strings.TrimSpace(userRules) != "" {
		parts = append(parts, "\n## Project-Specific Rules\n\n"+strings.TrimSpace(userRules))
	}
	return strings.Join(parts, "\n")
}
