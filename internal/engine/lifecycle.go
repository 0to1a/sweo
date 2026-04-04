package engine

import (
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/0to1a/sweo/internal/config"
	"github.com/0to1a/sweo/internal/core"
	"github.com/0to1a/sweo/internal/github"
	"github.com/0to1a/sweo/internal/runtime"
)

// Lifecycle manages the polling loops and state transitions.
type Lifecycle struct {
	sm  *SessionManager
	cfg *config.Config

	// State tracking: sessionID -> last known status
	mu     sync.Mutex
	states map[string]core.SessionStatus

	// Fingerprints for dedup: sessionID -> last sent fingerprint
	ciFP      map[string]string
	reviewFP  map[string]string

	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewLifecycle creates a new Lifecycle manager.
func NewLifecycle(sm *SessionManager, cfg *config.Config) *Lifecycle {
	return &Lifecycle{
		sm:       sm,
		cfg:      cfg,
		states:   make(map[string]core.SessionStatus),
		ciFP:     make(map[string]string),
		reviewFP: make(map[string]string),
		stopCh:   make(chan struct{}),
	}
}

// Start begins the polling loops.
func (lc *Lifecycle) Start() {
	lc.wg.Add(2)

	go lc.issueLoop()
	go lc.changesLoop()

	log.Printf("Lifecycle started: issue check every %ds, changes check every %ds",
		lc.cfg.DelayCheckIssue, lc.cfg.DelayCheckChanges)
}

// Stop signals the polling loops to stop and waits for them to finish.
func (lc *Lifecycle) Stop() {
	close(lc.stopCh)
	lc.wg.Wait()
	log.Println("Lifecycle stopped")
}

// issueLoop polls for new agent:todo issues.
func (lc *Lifecycle) issueLoop() {
	defer lc.wg.Done()

	ticker := time.NewTicker(time.Duration(lc.cfg.DelayCheckIssue) * time.Second)
	defer ticker.Stop()

	// Run immediately on start
	lc.checkIssues()

	for {
		select {
		case <-lc.stopCh:
			return
		case <-ticker.C:
			lc.checkIssues()
		}
	}
}

// changesLoop polls for PR/CI/review changes.
func (lc *Lifecycle) changesLoop() {
	defer lc.wg.Done()

	ticker := time.NewTicker(time.Duration(lc.cfg.DelayCheckChanges) * time.Second)
	defer ticker.Stop()

	// Run immediately on start
	lc.checkChanges()

	for {
		select {
		case <-lc.stopCh:
			return
		case <-ticker.C:
			lc.checkChanges()
		}
	}
}

func (lc *Lifecycle) checkIssues() {
	for projectID, proj := range lc.cfg.Projects {
		issues, err := github.ListTodoIssues(proj.Repo)
		if err != nil {
			log.Printf("WARN: failed to list issues for %s: %v", projectID, err)
			continue
		}

		for _, issue := range issues {
			issueNum := fmt.Sprintf("%d", issue.Number)

			// Skip if already assigned to a session
			if lc.hasSessionForIssue(projectID, issueNum) {
				continue
			}

			log.Printf("New issue found: #%d %s (project: %s)", issue.Number, issue.Title, projectID)

			// Spawn session
			session, err := lc.sm.Spawn(projectID, issue)
			if err != nil {
				log.Printf("ERROR: failed to spawn session for issue #%d: %v", issue.Number, err)
				continue
			}

			// Update labels: remove agent:todo, add agent:working
			if err := github.RemoveLabel(proj.Repo, issue.Number, "agent:todo"); err != nil {
				log.Printf("WARN: failed to remove agent:todo from #%d: %v", issue.Number, err)
			}
			if err := github.AddLabel(proj.Repo, issue.Number, "agent:working"); err != nil {
				log.Printf("WARN: failed to add agent:working to #%d: %v", issue.Number, err)
			}

			// Track state
			lc.mu.Lock()
			lc.states[session.ID] = core.StatusWorking
			lc.mu.Unlock()
		}
	}
}

func (lc *Lifecycle) checkChanges() {
	for projectID, proj := range lc.cfg.Projects {
		sessions, err := lc.sm.List(projectID)
		if err != nil {
			log.Printf("WARN: failed to list sessions for %s: %v", projectID, err)
			continue
		}

		for _, session := range sessions {
			if session.Status.IsTerminal() {
				continue
			}

			newStatus := lc.determineStatus(session, proj)
			if newStatus == "" || newStatus == session.Status {
				continue
			}

			oldStatus := session.Status
			log.Printf("Session %s: %s -> %s", session.ID, oldStatus, newStatus)

			// Update metadata
			sessionsDir := core.SessionsDir(lc.cfg.Hash, projectID)
			core.UpdateMetadata(sessionsDir, session.ID, map[string]string{
				"status": string(newStatus),
			})

			// Fire outbound webhook
			issueNum, _ := strconv.Atoi(session.IssueID)
			FireWebhook(lc.cfg.WebhookURL, projectID, WebhookPayload{
				Event:      "status_change",
				SessionID:  session.ID,
				IssueID:    session.IssueID,
				IssueTitle: session.IssueTitle,
				OldStatus:  string(oldStatus),
				NewStatus:  string(newStatus),
				PRURL:      session.PRURL,
				Timestamp:  time.Now().UTC().Format(time.RFC3339),
			})

			// Handle terminal state: merged -> cleanup -> done
			if newStatus == core.StatusMerged {
				lc.handleMerged(session, proj, issueNum)
			}

			// Track new state
			lc.mu.Lock()
			lc.states[session.ID] = newStatus
			lc.mu.Unlock()
		}
	}
}

func (lc *Lifecycle) determineStatus(session *core.Session, proj config.ProjectConfig) core.SessionStatus {
	switch session.Status {
	case core.StatusSpawning, core.StatusWorking, core.StatusWaitingInput:
		return lc.checkWorkingSession(session, proj)

	case core.StatusPROpen:
		return lc.checkPRSession(session, proj)

	case core.StatusCIFailed:
		return lc.checkCIFailed(session, proj)

	case core.StatusChangesRequested:
		return lc.checkChangesRequested(session, proj)

	case core.StatusReviewPending:
		return lc.checkReviewPending(session, proj)
	}

	return ""
}

func (lc *Lifecycle) checkWorkingSession(session *core.Session, proj config.ProjectConfig) core.SessionStatus {
	// Check if tmux is alive
	if session.TmuxName != "" && !runtime.HasSession(session.TmuxName) {
		return core.StatusErrored
	}

	// Check inactivity
	if lc.isInactive(session) {
		return core.StatusStuck
	}

	// Check if PR was created (detected via branch)
	if session.Branch != "" && session.PRURL == "" {
		pr, err := github.FindPR(proj.Repo, session.Branch)
		if err == nil && pr != nil {
			// Update metadata with PR info
			sessionsDir := core.SessionsDir(lc.cfg.Hash, session.ProjectID)
			core.UpdateMetadata(sessionsDir, session.ID, map[string]string{
				"pr":       pr.URL,
				"prNumber": fmt.Sprintf("%d", pr.Number),
			})
			return core.StatusPROpen
		}
	}

	return ""
}

func (lc *Lifecycle) checkPRSession(session *core.Session, proj config.ProjectConfig) core.SessionStatus {
	if session.PRNumber == 0 {
		return ""
	}

	// Check if merged
	state, err := github.GetPRState(proj.Repo, session.PRNumber)
	if err != nil {
		return ""
	}
	if state == github.PRMerged {
		return core.StatusMerged
	}
	if state == github.PRClosed {
		return core.StatusErrored
	}

	// Check CI (if configured)
	if proj.IsWaitingCI() {
		checks, err := github.GetCIChecks(proj.Repo, session.PRNumber)
		if err == nil {
			ci := github.SummarizeCI(checks)
			if ci == github.CIFailing {
				lc.maybeSendCIFailure(session, proj, checks)
				return core.StatusCIFailed
			}
		}
	}

	// Check review
	review, err := github.GetReviewDecision(proj.Repo, session.PRNumber)
	if err == nil {
		switch review {
		case github.ReviewChangesRequested:
			lc.maybeSendReviewComments(session, proj)
			return core.StatusChangesRequested
		case github.ReviewPending:
			return core.StatusReviewPending
		}
	}

	return ""
}

func (lc *Lifecycle) checkCIFailed(session *core.Session, proj config.ProjectConfig) core.SessionStatus {
	if session.PRNumber == 0 {
		return ""
	}

	// Check if CI is now passing
	checks, err := github.GetCIChecks(proj.Repo, session.PRNumber)
	if err != nil {
		return ""
	}

	ci := github.SummarizeCI(checks)
	if ci == github.CIPassing {
		return core.StatusPROpen
	}

	return ""
}

func (lc *Lifecycle) checkChangesRequested(session *core.Session, proj config.ProjectConfig) core.SessionStatus {
	if session.PRNumber == 0 {
		return ""
	}

	// Check if review is now approved
	review, err := github.GetReviewDecision(proj.Repo, session.PRNumber)
	if err != nil {
		return ""
	}

	switch review {
	case github.ReviewApproved:
		return core.StatusReviewPending // will advance to merged check next cycle
	case github.ReviewPending:
		return core.StatusReviewPending
	}

	return ""
}

func (lc *Lifecycle) checkReviewPending(session *core.Session, proj config.ProjectConfig) core.SessionStatus {
	if session.PRNumber == 0 {
		return ""
	}

	// Check if merged
	state, err := github.GetPRState(proj.Repo, session.PRNumber)
	if err != nil {
		return ""
	}
	if state == github.PRMerged {
		return core.StatusMerged
	}

	// Check if review decision changed
	review, err := github.GetReviewDecision(proj.Repo, session.PRNumber)
	if err != nil {
		return ""
	}

	if review == github.ReviewApproved {
		// Check if merged again (approved + merged in same cycle)
		state, _ = github.GetPRState(proj.Repo, session.PRNumber)
		if state == github.PRMerged {
			return core.StatusMerged
		}
	}
	if review == github.ReviewChangesRequested {
		return core.StatusChangesRequested
	}

	return ""
}

func (lc *Lifecycle) handleMerged(session *core.Session, proj config.ProjectConfig, issueNum int) {
	log.Printf("Session %s merged, cleaning up", session.ID)

	if err := lc.sm.Cleanup(session); err != nil {
		log.Printf("WARN: cleanup failed for %s: %v", session.ID, err)
	}

	// Update issue labels
	if issueNum > 0 {
		github.RemoveLabel(proj.Repo, issueNum, "agent:working")
		github.AddLabel(proj.Repo, issueNum, "agent:done")
	}
}

func (lc *Lifecycle) hasSessionForIssue(projectID, issueID string) bool {
	sessions, err := lc.sm.List(projectID)
	if err != nil {
		return false
	}
	for _, s := range sessions {
		if s.IssueID == issueID && !s.Status.IsTerminal() {
			return true
		}
	}
	return false
}

func (lc *Lifecycle) isInactive(session *core.Session) bool {
	threshold := time.Duration(lc.cfg.MaxInactiveMins) * time.Minute

	// Check tmux output for recent activity
	if session.TmuxName == "" || !runtime.HasSession(session.TmuxName) {
		return false
	}

	// Use creation time as a proxy (proper implementation would track last activity)
	return time.Since(session.CreatedAt) > threshold
}

// maybeSendCIFailure sends CI failure details to the agent (deduplicated).
func (lc *Lifecycle) maybeSendCIFailure(session *core.Session, proj config.ProjectConfig, checks []github.CICheck) {
	// Build fingerprint from failed check names
	var fp string
	for _, c := range checks {
		if github.SummarizeCI([]github.CICheck{c}) == github.CIFailing {
			fp += c.Name + ","
		}
	}

	lc.mu.Lock()
	if lc.ciFP[session.ID] == fp {
		lc.mu.Unlock()
		return
	}
	lc.ciFP[session.ID] = fp
	lc.mu.Unlock()

	// Build message with failure details
	msg := "CI checks are failing. Here are the details:\n\n"
	for _, c := range checks {
		state := github.SummarizeCI([]github.CICheck{c})
		if state == github.CIFailing {
			msg += fmt.Sprintf("- %s: FAILED", c.Name)
			if c.Link != "" {
				msg += fmt.Sprintf(" (%s)", c.Link)
			}
			msg += "\n"
		}
	}
	msg += "\nPlease fix these CI failures."

	if err := lc.sm.SendMessage(session.ID, msg); err != nil {
		log.Printf("WARN: failed to send CI failure to %s: %v", session.ID, err)
	}
}

// maybeSendReviewComments sends review comments to the agent (deduplicated).
func (lc *Lifecycle) maybeSendReviewComments(session *core.Session, proj config.ProjectConfig) {
	if session.PRNumber == 0 {
		return
	}

	comments, err := github.GetPendingComments(proj.Repo, session.PRNumber)
	if err != nil || len(comments) == 0 {
		return
	}

	// Build fingerprint
	var fp string
	for _, c := range comments {
		fp += c[:min(50, len(c))] + "|"
	}

	lc.mu.Lock()
	if lc.reviewFP[session.ID] == fp {
		lc.mu.Unlock()
		return
	}
	lc.reviewFP[session.ID] = fp
	lc.mu.Unlock()

	// Build message
	msg := "Review comments have been left on your PR. Please address them:\n\n"
	for i, c := range comments {
		msg += fmt.Sprintf("Comment %d:\n%s\n\n", i+1, c)
	}

	if err := lc.sm.SendMessage(session.ID, msg); err != nil {
		log.Printf("WARN: failed to send review comments to %s: %v", session.ID, err)
	}
}
