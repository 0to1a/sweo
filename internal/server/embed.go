package server

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed all:frontend_dist
var frontendFS embed.FS

// serveFrontend returns an http.Handler that serves the embedded frontend.
// Falls back to index.html for SPA client-side routing.
func serveFrontend() http.Handler {
	dist, err := fs.Sub(frontendFS, "frontend_dist")
	if err != nil {
		// No embedded frontend — serve a placeholder
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte("<html><body><h1>sweo</h1><p>Frontend not built. Run <code>make build-frontend</code></p></body></html>"))
		})
	}

	fileServer := http.FileServer(http.FS(dist))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Sanitize path to prevent traversal
		p := path.Clean("/" + strings.TrimPrefix(r.URL.Path, "/"))
		p = strings.TrimPrefix(p, "/")
		if p == "" || p == "." {
			p = "index.html"
		}

		// Check if file exists
		if f, err := dist.Open(p); err == nil {
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		// SPA fallback: serve index.html for all non-file routes
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}
