package github

import (
	"encoding/json"
	"fmt"
	"strings"
)

// FindPR finds a PR for the given branch.
// Returns nil, nil if no PR exists.
func FindPR(repo, branch string) (*PRInfo, error) {
	out, err := runGH("pr", "list",
		"--repo", repo,
		"--head", branch,
		"--json", "number,url,title,headRefName,baseRefName,isDraft",
		"--limit", "1",
	)
	if err != nil {
		return nil, fmt.Errorf("find PR: %w", err)
	}

	var prs []PRInfo
	if err := json.Unmarshal([]byte(out), &prs); err != nil {
		return nil, fmt.Errorf("parse PR list: %w", err)
	}
	if len(prs) == 0 {
		return nil, nil
	}
	return &prs[0], nil
}

// GetPRState returns the state of a PR (OPEN, CLOSED, MERGED).
func GetPRState(repo string, prNumber int) (PRState, error) {
	out, err := runGH("pr", "view",
		fmt.Sprintf("%d", prNumber),
		"--repo", repo,
		"--json", "state",
	)
	if err != nil {
		return "", fmt.Errorf("get PR state: %w", err)
	}

	var result struct {
		State string `json:"state"`
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		return "", fmt.Errorf("parse PR state: %w", err)
	}
	return PRState(result.State), nil
}

// GetCIChecks returns CI check results for a PR.
func GetCIChecks(repo string, prNumber int) ([]CICheck, error) {
	out, err := runGH("pr", "checks",
		fmt.Sprintf("%d", prNumber),
		"--repo", repo,
		"--json", "name,state,link",
	)
	if err != nil {
		// gh pr checks may fail for various reasons (no checks, etc.)
		if strings.Contains(err.Error(), "no checks") {
			return nil, nil
		}
		return nil, fmt.Errorf("get CI checks: %w", err)
	}

	var checks []CICheck
	if err := json.Unmarshal([]byte(out), &checks); err != nil {
		return nil, fmt.Errorf("parse CI checks: %w", err)
	}
	return checks, nil
}

// SummarizeCI aggregates CI check states into a single summary.
func SummarizeCI(checks []CICheck) CISummary {
	if len(checks) == 0 {
		return CINone
	}

	hasFailed := false
	hasPending := false

	for _, check := range checks {
		state := normalizeCheckState(check.State)
		switch state {
		case "failed":
			hasFailed = true
		case "pending", "running":
			hasPending = true
		}
	}

	if hasFailed {
		return CIFailing
	}
	if hasPending {
		return CIPending
	}
	return CIPassing
}

// GetReviewDecision returns the review decision for a PR.
func GetReviewDecision(repo string, prNumber int) (ReviewDecision, error) {
	out, err := runGH("pr", "view",
		fmt.Sprintf("%d", prNumber),
		"--repo", repo,
		"--json", "reviewDecision",
	)
	if err != nil {
		return ReviewNone, fmt.Errorf("get review decision: %w", err)
	}

	var result struct {
		ReviewDecision string `json:"reviewDecision"`
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		return ReviewNone, fmt.Errorf("parse review decision: %w", err)
	}

	switch result.ReviewDecision {
	case "APPROVED":
		return ReviewApproved, nil
	case "CHANGES_REQUESTED":
		return ReviewChangesRequested, nil
	case "REVIEW_REQUIRED":
		return ReviewPending, nil
	default:
		return ReviewNone, nil
	}
}

// GetPendingComments returns unresolved review comment bodies for a PR.
func GetPendingComments(repo string, prNumber int) ([]string, error) {
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid repo format: %s", repo)
	}
	owner, name := parts[0], parts[1]

	query := fmt.Sprintf(`query($owner: String!, $name: String!, $number: Int!) {
  repository(owner: $owner, name: $name) {
    pullRequest(number: $number) {
      reviewThreads(first: 100) {
        nodes {
          isResolved
          comments(first: 1) {
            nodes {
              body
            }
          }
        }
      }
    }
  }
}`)

	out, err := runGH("api", "graphql",
		"-f", fmt.Sprintf("owner=%s", owner),
		"-f", fmt.Sprintf("name=%s", name),
		"-F", fmt.Sprintf("number=%d", prNumber),
		"-f", fmt.Sprintf("query=%s", query),
	)
	if err != nil {
		return nil, fmt.Errorf("get pending comments: %w", err)
	}

	var result struct {
		Data struct {
			Repository struct {
				PullRequest struct {
					ReviewThreads struct {
						Nodes []struct {
							IsResolved bool `json:"isResolved"`
							Comments   struct {
								Nodes []struct {
									Body string `json:"body"`
								} `json:"nodes"`
							} `json:"comments"`
						} `json:"nodes"`
					} `json:"reviewThreads"`
				} `json:"pullRequest"`
			} `json:"repository"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		return nil, fmt.Errorf("parse pending comments: %w", err)
	}

	var comments []string
	for _, thread := range result.Data.Repository.PullRequest.ReviewThreads.Nodes {
		if !thread.IsResolved && len(thread.Comments.Nodes) > 0 {
			comments = append(comments, thread.Comments.Nodes[0].Body)
		}
	}
	return comments, nil
}

// PRComment represents a comment on a PR (issue comment, not review thread).
type PRComment struct {
	Author string `json:"author"`
	Body   string `json:"body"`
	ID     int    `json:"id"`
}

// GetPRComments returns all human comments on a PR from three sources:
// 1. Issue comments (general comments below the PR description)
// 2. Review bodies (submitted via "Comment" or "Request Changes")
// 3. Review thread comments (inline code comments)
// This ensures we catch feedback even from the repo owner who can't "request changes" on their own PR.
func GetPRComments(repo string, prNumber int) ([]PRComment, error) {
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid repo format: %s", repo)
	}
	owner, name := parts[0], parts[1]

	var all []PRComment

	// 1. Issue comments
	if out, err := runGH("api",
		fmt.Sprintf("repos/%s/%s/issues/%d/comments", owner, name, prNumber),
		"--jq", `.[] | {id: .id, author: .user.login, body: .body}`,
	); err == nil {
		all = append(all, parseNDJSON(out)...)
	}

	// 2. Review bodies (the text submitted with approve/comment/request-changes)
	if out, err := runGH("api",
		fmt.Sprintf("repos/%s/%s/pulls/%d/reviews", owner, name, prNumber),
		"--jq", `.[] | select(.body != "") | {id: .id, author: .user.login, body: .body}`,
	); err == nil {
		all = append(all, parseNDJSON(out)...)
	}

	// 3. Inline review comments (code line comments)
	if out, err := runGH("api",
		fmt.Sprintf("repos/%s/%s/pulls/%d/comments", owner, name, prNumber),
		"--jq", `.[] | {id: .id, author: .user.login, body: (.body + " (file: " + .path + ")")}`,
	); err == nil {
		all = append(all, parseNDJSON(out)...)
	}

	if len(all) == 0 {
		return nil, nil
	}
	return all, nil
}

func parseNDJSON(out string) []PRComment {
	var comments []PRComment
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		var c PRComment
		if err := json.Unmarshal([]byte(line), &c); err != nil {
			continue
		}
		if c.Body != "" {
			comments = append(comments, c)
		}
	}
	return comments
}

// normalizeCheckState maps GitHub check states to a simple set.
func normalizeCheckState(state string) string {
	switch strings.ToUpper(state) {
	case "SUCCESS":
		return "passed"
	case "FAILURE", "TIMED_OUT", "CANCELLED", "ACTION_REQUIRED", "ERROR":
		return "failed"
	case "IN_PROGRESS":
		return "running"
	case "PENDING", "QUEUED", "REQUESTED", "WAITING", "EXPECTED":
		return "pending"
	case "SKIPPED", "NEUTRAL", "STALE":
		return "skipped"
	default:
		return "skipped"
	}
}
