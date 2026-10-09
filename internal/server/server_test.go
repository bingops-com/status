package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/bingops-com/status/internal/config"
	"github.com/bingops-com/status/internal/monitor"
)

func TestStatusJoinsConfigAndHidesGatus(t *testing.T) {
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	rec, err := monitor.NewRecorder(monitor.Options{Days: 7, Location: time.UTC, FailureThreshold: 2, Interval: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	rec.Record([]monitor.Endpoint{{Key: "public_romm", Name: "RomM", Checks: []monitor.Check{{Time: now, Up: true, Ms: 120}}}}, now)
	srv := &Server{
		Config: &config.Config{Title: "lab.bingo", Gatus: "http://gatus.internal:8080", Interval: time.Minute,
			Services: []config.Service{{Name: "RomM", Description: "Bibliothèque de jeux", URL: "https://rom.lab.bingo"}}},
		Recorder: rec,
		Assets:   fstest.MapFS{"index.html": {Data: []byte("<html>page</html>")}, "assets/app.js": {Data: []byte("js")}},
		Now:      func() time.Time { return now },
	}
	h := srv.Handler()

	res := httptest.NewRecorder()
	h.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/status", nil))
	if strings.Contains(res.Body.String(), "gatus.internal") {
		t.Fatalf("the API reveals the Gatus address: %s", res.Body)
	}
	var got struct {
		State    string
		Services []struct {
			Name, Description, URL, State string
			History                       []any
		}
	}
	if err := json.Unmarshal(res.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.State != "ok" || len(got.Services) != 1 {
		t.Fatalf("unexpected status: %s", res.Body)
	}
	s := got.Services[0]
	if s.Description != "Bibliothèque de jeux" || s.URL != "https://rom.lab.bingo" || s.State != "up" || len(s.History) != 7 {
		t.Fatalf("unexpected service: %+v", s)
	}

	for path, want := range map[string]string{"/": "page", "/anything": "page", "/assets/app.js": "js"} {
		res := httptest.NewRecorder()
		h.ServeHTTP(res, httptest.NewRequest(http.MethodGet, path, nil))
		if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), want) {
			t.Fatalf("GET %s = %d %q", path, res.Code, res.Body)
		}
		if res.Header().Get("Content-Security-Policy") == "" {
			t.Fatalf("GET %s has no content security policy", path)
		}
	}

	res = httptest.NewRecorder()
	h.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/status", nil))
	if res.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST /api/status = %d, want 405", res.Code)
	}
}
