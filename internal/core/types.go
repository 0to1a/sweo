package core

import "time"

// SessionStatus represents the current state of a session.
type SessionStatus string

const (
	StatusSpawning         SessionStatus = "spawning"
	StatusWorking          SessionStatus = "working"
	StatusWaitingInput     SessionStatus = "waiting_input"
	StatusPROpen           SessionStatus = "pr_open"
	StatusCIFailed         SessionStatus = "ci_failed"
	StatusChangesRequested SessionStatus = "changes_requested"
	StatusReviewPending    SessionStatus = "review_pending"
	StatusMerged           SessionStatus = "merged"
	StatusCleanup          SessionStatus = "cleanup"
	StatusDone             SessionStatus = "done"
	StatusErrored          SessionStatus = "errored"
	StatusStuck            SessionStatus = "stuck"
)

// IsTerminal returns true if the status is a final state.
func (s SessionStatus) IsTerminal() bool {
	switch s {
	case StatusDone, StatusErrored, StatusCleanup, StatusMerged:
		return true
	}
	return false
}

// Session represents a running or completed agent session.
type Session struct {
	ID            string
	ProjectID     string
	Status        SessionStatus
	Branch        string
	IssueID       string
	IssueTitle    string
	PRURL         string
	PRNumber      int
	WorkspacePath string
	TmuxName      string
	Agent         string
	CreatedAt     time.Time
	Metadata      map[string]string
}
