// Package server exposes the public status API and serves the embedded UI.
package server

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/bingops-com/status/internal/config"
	"github.com/bingops-com/status/internal/monitor"
)

type Server struct {
	Config   *config.Config
	Recorder *monitor.Recorder
	Assets   fs.FS
	Demo     bool
	Now      func() time.Time
}

type serviceView struct {
	monitor.ServiceView
	config.Service
}

type status struct {
	Title    string          `json:"title"`
	Demo     bool            `json:"demo,omitempty"`
	Interval int             `json:"interval"` // seconds between two readings
	Notices  []config.Notice `json:"notices"`
	monitor.Snapshot
	Services []serviceView `json:"services"`
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("GET /api/status", s.status)
	mux.Handle("GET /", s.static())
	return headers(mux)
}

func (s *Server) status(w http.ResponseWriter, _ *http.Request) {
	snap := s.Recorder.Snapshot(s.now())
	out := status{Title: s.Config.Title, Demo: s.Demo, Interval: int(s.Config.Interval.Seconds()), Notices: s.Config.Notices, Snapshot: snap, Services: []serviceView{}}
	if out.Notices == nil {
		out.Notices = []config.Notice{}
	}
	for _, v := range snap.Services {
		out.Services = append(out.Services, serviceView{ServiceView: v, Service: s.Config.Service(v.Name)})
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=15")
	json.NewEncoder(w).Encode(out)
}

// The page is public and read-only: nothing it loads comes from elsewhere.
func headers(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

// static serves the built UI; any unknown path gets the page itself.
func (s *Server) static() http.Handler {
	files := http.FileServerFS(s.Assets)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if name == "" {
			name = "index.html"
		}
		if _, err := fs.Stat(s.Assets, name); err != nil {
			name = "index.html"
		}
		if strings.HasPrefix(name, "assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}
		if name == "index.html" {
			// The file server would redirect /index.html to /.
			page, err := fs.ReadFile(s.Assets, name)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(s.withPreview(page, r.UserAgent()))
			return
		}
		r2 := r.Clone(r.Context())
		r2.URL.Path = "/" + name
		files.ServeHTTP(w, r2)
	})
}
