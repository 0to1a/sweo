package github

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"
)

const ghTimeout = 30 * time.Second

// ListTodoIssues returns open issues tagged with "agent:todo".
func ListTodoIssues(repo string) ([]Issue, error) {
	out, err := runGH("issue", "list",
		"--repo", repo,
		"--label", "agent:todo",
		"--state", "open",
		"--json", "number,title,body,url,state,labels",
		"--limit", "100",
	)
	if err != nil {
		return nil, fmt.Errorf("list todo issues: %w", err)
	}

	var issues []Issue
	if err := json.Unmarshal([]byte(out), &issues); err != nil {
		return nil, fmt.Errorf("parse issues: %w", err)
	}
	return issues, nil
}

// GetIssue fetches a single issue by number.
func GetIssue(repo string, number int) (*Issue, error) {
	out, err := runGH("issue", "view",
		fmt.Sprintf("%d", number),
		"--repo", repo,
		"--json", "number,title,body,url,state,labels",
	)
	if err != nil {
		return nil, fmt.Errorf("get issue: %w", err)
	}

	var issue Issue
	if err := json.Unmarshal([]byte(out), &issue); err != nil {
		return nil, fmt.Errorf("parse issue: %w", err)
	}
	return &issue, nil
}

// AddLabel adds a label to an issue.
func AddLabel(repo string, number int, label string) error {
	_, err := runGH("issue", "edit",
		fmt.Sprintf("%d", number),
		"--repo", repo,
		"--add-label", label,
	)
	if err != nil {
		return fmt.Errorf("add label %q: %w", label, err)
	}
	return nil
}

// RemoveLabel removes a label from an issue.
func RemoveLabel(repo string, number int, label string) error {
	_, err := runGH("issue", "edit",
		fmt.Sprintf("%d", number),
		"--repo", repo,
		"--remove-label", label,
	)
	if err != nil {
		// Ignore errors if label doesn't exist
		if strings.Contains(err.Error(), "not found") {
			return nil
		}
		return fmt.Errorf("remove label %q: %w", label, err)
	}
	return nil
}

// CommentOnIssue adds a comment to an issue.
func CommentOnIssue(repo string, number int, body string) error {
	_, err := runGH("issue", "comment",
		fmt.Sprintf("%d", number),
		"--repo", repo,
		"--body", body,
	)
	if err != nil {
		return fmt.Errorf("comment on issue: %w", err)
	}
	return nil
}

// sweoLabels defines the labels sweo manages.
var sweoLabels = []struct {
	Name        string
	Color       string
	Description string
}{
	{"agent:todo", "0075ca", "Issue assigned to sweo agent"},
	{"agent:working", "e4e669", "Agent is working on this issue"},
	{"agent:done", "0e8a16", "Agent completed this issue"},
}

// EnsureLabels creates the sweo labels in the repo if they don't exist.
func EnsureLabels(repo string) error {
	// Get existing labels
	out, err := runGH("label", "list", "--repo", repo, "--json", "name", "--limit", "200")
	if err != nil {
		return fmt.Errorf("list labels: %w", err)
	}

	var existing []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(out), &existing); err != nil {
		return fmt.Errorf("parse labels: %w", err)
	}

	existingSet := make(map[string]bool)
	for _, l := range existing {
		existingSet[l.Name] = true
	}

	for _, label := range sweoLabels {
		if existingSet[label.Name] {
			continue
		}
		_, err := runGH("label", "create", label.Name,
			"--repo", repo,
			"--color", label.Color,
			"--description", label.Description,
		)
		if err != nil {
			// Ignore "already exists" race condition
			if strings.Contains(err.Error(), "already exists") {
				continue
			}
			return fmt.Errorf("create label %q: %w", label.Name, err)
		}
	}

	return nil
}

// BranchName generates a branch name for an issue number.
func BranchName(issueNumber int) string {
	return fmt.Sprintf("feat/issue-%d", issueNumber)
}

var retryDelays = []time.Duration{15 * time.Second, 30 * time.Second, 60 * time.Second}

func runGH(args ...string) (string, error) {
	output, err := runGHOnce(args...)
	if err == nil {
		return output, nil
	}

	// Retry on network errors (connection refused, no route, timeout)
	if !isNetworkError(err) {
		return "", err
	}

	for i, delay := range retryDelays {
		log.Printf("gh %s: network error, retry %d/%d in %s", args[0], i+1, len(retryDelays), delay)
		time.Sleep(delay)
		output, err = runGHOnce(args...)
		if err == nil {
			return output, nil
		}
		if !isNetworkError(err) {
			return "", err
		}
	}

	return "", err
}

func runGHOnce(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), ghTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "gh", args...)
	output, err := cmd.Output()
	if err != nil {
		var stderr string
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr = string(exitErr.Stderr)
		}
		return "", fmt.Errorf("gh %s: %w (stderr: %s)", args[0], err, strings.TrimSpace(stderr))
	}
	return strings.TrimSpace(string(output)), nil
}

func isNetworkError(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "no route to host") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "dial tcp") ||
		strings.Contains(msg, "i/o timeout") ||
		strings.Contains(msg, "network is unreachable")
}
