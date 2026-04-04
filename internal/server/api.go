package server

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func (s *Server) handleListSessions(w http.ResponseWriter, r *http.Request) {
	sessions, err := s.sm.ListAll()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, sessions)
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

func (s *Server) handleGitHubWebhook(w http.ResponseWriter, r *http.Request) {
	// Stub: endpoint exists but does not process events yet.
	w.WriteHeader(200)
	fmt.Fprintf(w, `{"status":"ok","message":"webhook received but not processed"}`)
}

func writeJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
