package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/0to1a/sweo/internal/core"
)

func (s *Server) handleListSessions(w http.ResponseWriter, r *http.Request) {
	sessions, err := s.sm.ListAll()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, filterActive(sessions))
}

func (s *Server) handleGetSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	session, err := s.sm.Get(id)
	if err != nil {
		http.Error(w, err.Error(), 404)
		return
	}
	writeJSON(w, session)
}

func (s *Server) handleKillSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.sm.Kill(id); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, map[string]string{"status": "killed", "session_id": id})
}

func (s *Server) handleResumeSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.sm.Resume(id); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	writeJSON(w, map[string]string{"status": "resumed", "session_id": id})
}

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	type projectInfo struct {
		Name          string `json:"name"`
		Repo          string `json:"repo"`
		Agent         string `json:"agent"`
		DefaultBranch string `json:"defaultBranch"`
	}

	var projects []projectInfo
	for name, proj := range s.cfg.Projects {
		projects = append(projects, projectInfo{
			Name:          name,
			Repo:          proj.Repo,
			Agent:         proj.Agent,
			DefaultBranch: proj.DefaultBranch,
		})
	}
	writeJSON(w, projects)
}

func (s *Server) handleListPRs(w http.ResponseWriter, r *http.Request) {
	sessions, err := s.sm.ListAll()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	type prEntry struct {
		PRNumber  int    `json:"prNumber"`
		PRURL     string `json:"prUrl"`
		SessionID string `json:"sessionId"`
		ProjectID string `json:"projectId"`
		Branch    string `json:"branch"`
		Status    string `json:"status"`
		IssueID   string `json:"issueId"`
		IssueTitle string `json:"issueTitle"`
	}

	var prs []prEntry
	for _, s := range sessions {
		if s.PRNumber > 0 {
			prs = append(prs, prEntry{
				PRNumber:  s.PRNumber,
				PRURL:     s.PRURL,
				SessionID: s.ID,
				ProjectID: s.ProjectID,
				Branch:    s.Branch,
				Status:    string(s.Status),
				IssueID:   s.IssueID,
				IssueTitle: s.IssueTitle,
			})
		}
	}
	writeJSON(w, prs)
}

func (s *Server) handleGitHubWebhook(w http.ResponseWriter, r *http.Request) {
	// Stub: endpoint exists but does not process events yet.
	w.WriteHeader(200)
	fmt.Fprintf(w, `{"status":"ok","message":"webhook received but not processed"}`)
}

func filterActive(sessions []*core.Session) []*core.Session {
	var active []*core.Session
	for _, s := range sessions {
		if !s.Status.IsTerminal() {
			active = append(active, s)
		}
	}
	return active
}

func writeJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
