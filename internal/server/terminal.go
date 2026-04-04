package server

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sync"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type resizeMsg struct {
	Type string `json:"type"`
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}

func (s *Server) handleTerminalWS(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session")
	if sessionID == "" {
		http.Error(w, "session parameter required", 400)
		return
	}

	// Look up tmux name from session metadata
	session, err := s.sm.Get(sessionID)
	if err != nil {
		http.Error(w, "session not found: "+err.Error(), 404)
		return
	}
	if session.TmuxName == "" {
		http.Error(w, "session has no tmux session", 400)
		return
	}

	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	// Disable tmux mouse mode so xterm.js handles scroll natively
	disableMouseCmd := exec.Command("/usr/bin/tmux", "set-option", "-t", session.TmuxName, "mouse", "off")
	disableMouseCmd.Run() // best-effort, ignore errors

	// Spawn PTY attached to tmux session
	cmd := exec.Command("/usr/bin/tmux", "attach-session", "-t", session.TmuxName)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	ptmx, err := pty.Start(cmd)
	if err != nil {
		log.Printf("PTY start failed for %s: %v", sessionID, err)
		conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseInternalServerErr, "failed to start terminal"))
		return
	}
	defer func() {
		ptmx.Close()
		cmd.Process.Kill()
		cmd.Wait()
	}()

	// Set initial size
	pty.Setsize(ptmx, &pty.Winsize{Cols: 120, Rows: 40})

	var wg sync.WaitGroup

	// PTY stdout -> WebSocket
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 4096)
		for {
			n, err := ptmx.Read(buf)
			if err != nil {
				if err != io.EOF {
					log.Printf("PTY read error: %v", err)
				}
				conn.WriteMessage(websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				return
			}
			if err := conn.WriteMessage(websocket.TextMessage, buf[:n]); err != nil {
				return
			}
		}
	}()

	// WebSocket -> PTY stdin
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}

			// Check for resize message
			if len(msg) > 0 && msg[0] == '{' {
				var resize resizeMsg
				if json.Unmarshal(msg, &resize) == nil && resize.Type == "resize" {
					if resize.Cols > 0 && resize.Rows > 0 {
						pty.Setsize(ptmx, &pty.Winsize{Cols: resize.Cols, Rows: resize.Rows})
						continue
					}
				}
			}

			// Regular input
			if _, err := ptmx.Write(msg); err != nil {
				return
			}
		}
	}()

	wg.Wait()
}
