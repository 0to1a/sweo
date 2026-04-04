package core

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// DataDir returns the data directory for a project: ~/.sweo/data/{hash}-{projectID}
func DataDir(hash, projectID string) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".sweo", "data", fmt.Sprintf("%s-%s", hash, projectID))
}

// SessionsDir returns the sessions directory for a project.
func SessionsDir(hash, projectID string) string {
	return filepath.Join(DataDir(hash, projectID), "sessions")
}

// WorktreesDir returns the worktrees directory for a project.
func WorktreesDir(hash, projectID string) string {
	return filepath.Join(DataDir(hash, projectID), "worktrees")
}

// GenerateSessionPrefix generates a short prefix from a project ID.
// Rules: <=4 chars -> as-is, kebab/snake -> initials, CamelCase -> uppercase letters, else first 3.
func GenerateSessionPrefix(projectID string) string {
	id := strings.ToLower(projectID)

	if len(id) <= 4 {
		return id
	}

	// CamelCase: extract uppercase letters
	if hasCamelCase(projectID) {
		var initials []rune
		for _, r := range projectID {
			if unicode.IsUpper(r) {
				initials = append(initials, unicode.ToLower(r))
			}
		}
		if len(initials) >= 2 {
			return string(initials)
		}
	}

	// kebab-case or snake_case: first letter of each word
	if strings.ContainsAny(id, "-_") {
		parts := regexp.MustCompile(`[-_]+`).Split(id, -1)
		var initials []byte
		for _, p := range parts {
			if len(p) > 0 {
				initials = append(initials, p[0])
			}
		}
		if len(initials) >= 2 {
			return string(initials)
		}
	}

	// Fallback: first 3 chars
	if len(id) >= 3 {
		return id[:3]
	}
	return id
}

func hasCamelCase(s string) bool {
	upperCount := 0
	for _, r := range s {
		if unicode.IsUpper(r) {
			upperCount++
		}
	}
	return upperCount >= 2
}

// GenerateTmuxName creates a globally unique tmux session name: {hash}-{prefix}-{num}
func GenerateTmuxName(hash, prefix string, num int) string {
	return fmt.Sprintf("%s-%s-%d", hash, prefix, num)
}

// GenerateSessionID creates a session ID: {prefix}-{num}
func GenerateSessionID(prefix string, num int) string {
	return fmt.Sprintf("%s-%d", prefix, num)
}

// GetNextSessionNumber finds the next available session number for the given prefix.
func GetNextSessionNumber(existing []string, prefix string) int {
	pattern := regexp.MustCompile(fmt.Sprintf(`^%s-(\d+)$`, regexp.QuoteMeta(prefix)))
	max := 0
	for _, name := range existing {
		matches := pattern.FindStringSubmatch(name)
		if len(matches) == 2 {
			n, err := strconv.Atoi(matches[1])
			if err == nil && n > max {
				max = n
			}
		}
	}
	return max + 1
}
