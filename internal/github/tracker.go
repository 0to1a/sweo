package github

import (
	"context"
	"encoding/json"
	"fmt"
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

// BranchName generates a branch name for an issue number.
func BranchName(issueNumber int) string {
	return fmt.Sprintf("feat/issue-%d", issueNumber)
}

func runGH(args ...string) (string, error) {
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
