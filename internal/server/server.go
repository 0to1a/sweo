package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/0to1a/sweo/internal/config"
	"github.com/0to1a/sweo/internal/engine"
)

// Server is the HTTP server for the sweo dashboard and API.
type Server struct {
	cfg    *config.Config
	sm     *engine.SessionManager
	server *http.Server
}

// New creates a new Server.
func New(cfg *config.Config, sm *engine.SessionManager) *Server {
	return &Server{cfg: cfg, sm: sm}
}

// Start begins listening on the configured port.
func (s *Server) Start() error {
	mux := http.NewServeMux()
	s.registerRoutes(mux)

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.cfg.Port),
		Handler: corsMiddleware(mux),
	}

	log.Printf("HTTP server listening on :%d", s.cfg.Port)
	return s.server.ListenAndServe()
}

// Stop gracefully shuts down the server.
func (s *Server) Stop() {
	if s.server == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s.server.Shutdown(ctx)
}

func (s *Server) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/sessions", s.handleListSessions)
	mux.HandleFunc("GET /api/sessions/{id}", s.handleGetSession)
	mux.HandleFunc("POST /api/sessions/{id}/kill", s.handleKillSession)
	mux.HandleFunc("POST /api/sessions/{id}/resume", s.handleResumeSession)
	mux.HandleFunc("GET /api/events", s.handleSSE)
	mux.HandleFunc("GET /api/projects", s.handleListProjects)
	mux.HandleFunc("GET /api/prs", s.handleListPRs)
	mux.HandleFunc("POST /api/webhooks/github", s.handleGitHubWebhook)
	mux.HandleFunc("GET /ws/terminal", s.handleTerminalWS)

	// Serve embedded frontend (falls back to placeholder if not built)
	frontend := serveFrontend()
	mux.Handle("GET /", frontend)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(204)
			return
		}
		next.ServeHTTP(w, r)
	})
}
