package github

// Issue represents a GitHub issue.
type Issue struct {
	Number int      `json:"number"`
	Title  string   `json:"title"`
	Body   string   `json:"body"`
	URL    string   `json:"url"`
	State  string   `json:"state"`
	Labels []Label  `json:"labels"`
}

// Label represents a GitHub label.
type Label struct {
	Name string `json:"name"`
}

// PRInfo represents a GitHub pull request.
type PRInfo struct {
	Number     int    `json:"number"`
	URL        string `json:"url"`
	Title      string `json:"title"`
	Branch     string `json:"headRefName"`
	BaseBranch string `json:"baseRefName"`
	IsDraft    bool   `json:"isDraft"`
}

// CICheck represents a single CI check result.
type CICheck struct {
	Name  string `json:"name"`
	State string `json:"state"`
	Link  string `json:"link"`
}

// CISummary is the overall CI status.
type CISummary string

const (
	CIPassing CISummary = "passing"
	CIFailing CISummary = "failing"
	CIPending CISummary = "pending"
	CINone    CISummary = "none"
)

// ReviewDecision is the PR review decision.
type ReviewDecision string

const (
	ReviewApproved         ReviewDecision = "APPROVED"
	ReviewChangesRequested ReviewDecision = "CHANGES_REQUESTED"
	ReviewPending          ReviewDecision = "pending"
	ReviewNone             ReviewDecision = "none"
)

// PRState is the state of a PR.
type PRState string

const (
	PROpen   PRState = "OPEN"
	PRClosed PRState = "CLOSED"
	PRMerged PRState = "MERGED"
)
