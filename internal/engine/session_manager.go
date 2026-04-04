package engine

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/0to1a/sweo/internal/agent"
	"github.com/0to1a/sweo/internal/config"
	"github.com/0to1a/sweo/internal/core"
	"github.com/0to1a/sweo/internal/github"
	"github.com/0to1a/sweo/internal/runtime"
	"github.com/0to1a/sweo/internal/workspace"
)

// SessionManager handles session CRUD operations.
type SessionManager struct {
	Cfg *config.Config
}

// NewSessionManager creates a new SessionManager.
func NewSessionManager(cfg *config.Config) *SessionManager {
	return &SessionManager{Cfg: cfg}
}

// Spawn creates a new agent session for an issue.
func (sm *SessionManager) Spawn(projectID string, issue github.Issue) (*core.Session, error) {
	proj, ok := sm.Cfg.Projects[projectID]
	if !ok {
		return nil, fmt.Errorf("project %q not found", projectID)
	}

	sessionsDir := core.SessionsDir(sm.Cfg.Hash, projectID)
	worktreesDir := core.WorktreesDir(sm.Cfg.Hash, projectID)

	// Generate session ID
	prefix := core.GenerateSessionPrefix(projectID)
	existing, _ := core.ListSessions(sessionsDir)
	num := core.GetNextSessionNumber(existing, prefix)
	sessionID := core.GenerateSessionID(prefix, num)
	tmuxName := core.GenerateTmuxName(sm.Cfg.Hash, prefix, num)
	branch := github.BranchName(issue.Number)
	worktreePath := fmt.Sprintf("%s/%s", worktreesDir, sessionID)

	// Reserve session ID atomically
	if err := core.ReserveSessionID(sessionsDir, sessionID); err != nil {
		return nil, fmt.Errorf("reserve session ID: %w", err)
	}

	// Write initial metadata
	issueNum := fmt.Sprintf("%d", issue.Number)
	meta := map[string]string{
		"project":   projectID,
		"status":    string(core.StatusSpawning),
		"issue":     issueNum,
		"issueTitle": issue.Title,
		"branch":    branch,
		"agent":     proj.Agent,
		"tmuxName":  tmuxName,
		"worktree":  worktreePath,
		"createdAt": time.Now().UTC().Format(time.RFC3339),
	}
	if err := core.WriteMetadata(sessionsDir, sessionID, meta); err != nil {
		return nil, fmt.Errorf("write metadata: %w", err)
	}

	// Clean up stale worktree if it exists (e.g. from a previously killed session)
	if workspace.Exists(worktreePath) {
		log.Printf("Removing stale worktree at %s", worktreePath)
		workspace.Destroy(proj.Path, worktreePath)
	}

	// Create worktree
	if err := workspace.Create(proj.Path, proj.DefaultBranch, branch, worktreePath); err != nil {
		core.UpdateMetadata(sessionsDir, sessionID, map[string]string{"status": string(core.StatusErrored)})
		return nil, fmt.Errorf("create worktree: %w", err)
	}

	// Create agent
	ag := agent.New(proj.Agent)

	// Setup hooks (Claude Code needs metadata hooks)
	if err := ag.SetupHooks(worktreePath, sessionsDir, sessionID); err != nil {
		log.Printf("WARN: setup hooks failed for %s: %v", sessionID, err)
	}

	// Build environment and create tmux session
	launchCfg := agent.LaunchConfig{
		SessionID:     sessionID,
		ProjectID:     projectID,
		IssueID:       issueNum,
		WorkspacePath: worktreePath,
		AgentRules:    buildFullRules(proj.AgentRules),
		Prompt:        buildPrompt(issue),
	}

	env := ag.BuildEnvironment(launchCfg)
	if err := runtime.NewSession(tmuxName, worktreePath, env); err != nil {
		core.UpdateMetadata(sessionsDir, sessionID, map[string]string{"status": string(core.StatusErrored)})
		return nil, fmt.Errorf("create tmux session: %w", err)
	}

	// Launch agent
	launchCmd := ag.BuildLaunchCommand(launchCfg)
	if err := runtime.SendKeys(tmuxName, launchCmd); err != nil {
		return nil, fmt.Errorf("send launch command: %w", err)
	}

	// For Claude Code, wait until the prompt is ready before sending the task
	if proj.Agent == "claude-code" {
		if err := waitForAgentReady(tmuxName, 60*time.Second); err != nil {
			log.Printf("WARN: agent not ready for %s, sending prompt anyway: %v", sessionID, err)
		}
		if err := runtime.SendKeys(tmuxName, launchCfg.Prompt); err != nil {
			log.Printf("WARN: failed to send prompt to %s: %v", sessionID, err)
		}
	}

	// Update status to working
	core.UpdateMetadata(sessionsDir, sessionID, map[string]string{"status": string(core.StatusWorking)})

	session := &core.Session{
		ID:            sessionID,
		ProjectID:     projectID,
		Status:        core.StatusWorking,
		Branch:        branch,
		IssueID:       issueNum,
		IssueTitle:    issue.Title,
		WorkspacePath: worktreePath,
		TmuxName:      tmuxName,
		Agent:         proj.Agent,
		CreatedAt:     time.Now(),
		Metadata:      meta,
	}

	log.Printf("Spawned session %s for issue #%d: %s", sessionID, issue.Number, issue.Title)
	return session, nil
}

// List returns all sessions for a project.
func (sm *SessionManager) List(projectID string) ([]*core.Session, error) {
	sessionsDir := core.SessionsDir(sm.Cfg.Hash, projectID)
	ids, err := core.ListSessions(sessionsDir)
	if err != nil {
		return nil, err
	}

	var sessions []*core.Session
	for _, id := range ids {
		s, err := sm.loadSession(sessionsDir, id, projectID)
		if err != nil {
			log.Printf("WARN: failed to load session %s: %v", id, err)
			continue
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}

// ListAll returns sessions across all projects.
func (sm *SessionManager) ListAll() ([]*core.Session, error) {
	var all []*core.Session
	for projectID := range sm.Cfg.Projects {
		sessions, err := sm.List(projectID)
		if err != nil {
			continue
		}
		all = append(all, sessions...)
	}
	return all, nil
}

// Get returns a single session by ID, searching across all projects.
func (sm *SessionManager) Get(sessionID string) (*core.Session, error) {
	for projectID := range sm.Cfg.Projects {
		sessionsDir := core.SessionsDir(sm.Cfg.Hash, projectID)
		meta, err := core.ReadMetadata(sessionsDir, sessionID)
		if err != nil || meta == nil {
			continue
		}
		return sm.loadSession(sessionsDir, sessionID, projectID)
	}
	return nil, fmt.Errorf("session %q not found", sessionID)
}

// Kill terminates a session's tmux process, destroys worktree, and removes metadata.
func (sm *SessionManager) Kill(sessionID string) error {
	session, err := sm.Get(sessionID)
	if err != nil {
		return err
	}

	proj := sm.Cfg.Projects[session.ProjectID]

	if session.TmuxName != "" {
		runtime.KillSession(session.TmuxName)
	}

	if session.WorkspacePath != "" {
		if err := workspace.Destroy(proj.Path, session.WorkspacePath); err != nil {
			log.Printf("WARN: failed to destroy worktree for %s: %v", sessionID, err)
		}
	}

	sessionsDir := core.SessionsDir(sm.Cfg.Hash, session.ProjectID)
	return core.DeleteMetadata(sessionsDir, sessionID)
}

// Cleanup destroys the tmux session and worktree, then marks as done.
func (sm *SessionManager) Cleanup(session *core.Session) error {
	sessionsDir := core.SessionsDir(sm.Cfg.Hash, session.ProjectID)
	proj := sm.Cfg.Projects[session.ProjectID]

	core.UpdateMetadata(sessionsDir, session.ID, map[string]string{
		"status": string(core.StatusCleanup),
	})

	if session.TmuxName != "" && runtime.HasSession(session.TmuxName) {
		runtime.KillSession(session.TmuxName)
	}

	if session.WorkspacePath != "" {
		if err := workspace.Destroy(proj.Path, session.WorkspacePath); err != nil {
			log.Printf("WARN: failed to destroy worktree for %s: %v", session.ID, err)
		}
	}

	return core.UpdateMetadata(sessionsDir, session.ID, map[string]string{
		"status": string(core.StatusDone),
	})
}

// SendMessage sends a message to a running session.
func (sm *SessionManager) SendMessage(sessionID, message string) error {
	session, err := sm.Get(sessionID)
	if err != nil {
		return err
	}

	if session.TmuxName == "" {
		return fmt.Errorf("session %s has no tmux session", sessionID)
	}

	if !runtime.HasSession(session.TmuxName) {
		return fmt.Errorf("tmux session %s not running", session.TmuxName)
	}

	return runtime.SendKeys(session.TmuxName, message)
}

// UpdateStatus updates the status of a session.
func (sm *SessionManager) UpdateStatus(sessionID string, status core.SessionStatus) error {
	session, err := sm.Get(sessionID)
	if err != nil {
		return err
	}
	sessionsDir := core.SessionsDir(sm.Cfg.Hash, session.ProjectID)
	return core.UpdateMetadata(sessionsDir, sessionID, map[string]string{
		"status": string(status),
	})
}

func (sm *SessionManager) loadSession(sessionsDir, sessionID, projectID string) (*core.Session, error) {
	meta, err := core.ReadMetadata(sessionsDir, sessionID)
	if err != nil {
		return nil, err
	}
	if meta == nil {
		return nil, fmt.Errorf("session %q not found", sessionID)
	}

	createdAt, _ := time.Parse(time.RFC3339, meta["createdAt"])
	prNumber, _ := strconv.Atoi(meta["prNumber"])

	s := &core.Session{
		ID:            sessionID,
		ProjectID:     projectID,
		Status:        core.SessionStatus(meta["status"]),
		Branch:        meta["branch"],
		IssueID:       meta["issue"],
		IssueTitle:    meta["issueTitle"],
		PRURL:         meta["pr"],
		PRNumber:      prNumber,
		WorkspacePath: meta["worktree"],
		TmuxName:      meta["tmuxName"],
		Agent:         meta["agent"],
		CreatedAt:     createdAt,
		Metadata:      meta,
	}

	return s, nil
}

// waitForAgentReady polls tmux output until the agent's input prompt appears.
func waitForAgentReady(tmuxName string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		output, err := runtime.CapturePane(tmuxName, 20)
		if err == nil {
			// Claude Code shows ❯ when ready, Codex shows >
			if strings.Contains(output, "❯") || strings.Contains(output, "\n> ") {
				return nil
			}
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("timeout waiting for agent ready in %s", tmuxName)
}

func buildPrompt(issue github.Issue) string {
	prompt := fmt.Sprintf("Fix GitHub issue #%d: %s\n\n", issue.Number, issue.Title)
	if issue.Body != "" {
		prompt += fmt.Sprintf("Issue description:\n%s\n\n", issue.Body)
	}
	prompt += fmt.Sprintf("Issue URL: %s\n", issue.URL)
	prompt += "\nPlease work on this issue. Create a PR when ready."
	return prompt
}
