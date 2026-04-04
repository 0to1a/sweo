package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

const (
	sseSnapshotInterval  = 5 * time.Second
	sseHeartbeatInterval = 15 * time.Second
)

type sseSnapshot struct {
	Type      string `json:"type"`
	Timestamp string `json:"timestamp"`
	Sessions  any    `json:"sessions"`
}

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", 500)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// Send initial snapshot
	s.sendSnapshot(w, flusher)

	snapshotTicker := time.NewTicker(sseSnapshotInterval)
	heartbeatTicker := time.NewTicker(sseHeartbeatInterval)
	defer snapshotTicker.Stop()
	defer heartbeatTicker.Stop()

	ctx := r.Context()

	for {
		select {
		case <-ctx.Done():
			return
		case <-snapshotTicker.C:
			s.sendSnapshot(w, flusher)
		case <-heartbeatTicker.C:
			fmt.Fprintf(w, ": heartbeat\n\n")
			flusher.Flush()
		}
	}
}

func (s *Server) sendSnapshot(w http.ResponseWriter, flusher http.Flusher) {
	sessions, err := s.sm.ListAll()
	if err != nil {
		log.Printf("SSE snapshot error: %v", err)
		return
	}

	snapshot := sseSnapshot{
		Type:      "snapshot",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Sessions:  sessions,
	}

	data, err := json.Marshal(snapshot)
	if err != nil {
		return
	}

	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}
