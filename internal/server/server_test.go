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

func TestPageCarriesLinkPreview(t *testing.T) {
	paris, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 9, 20, 30, 0, 0, time.UTC)
	rec, err := monitor.NewRecorder(monitor.Options{Days: 90, Location: paris, FailureThreshold: 2, Interval: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	read := func(at time.Time, rommUp bool) {
		rec.Record([]monitor.Endpoint{
			{Key: "public_authentik", Name: "Authentik", Checks: []monitor.Check{{Time: at, Up: true}}},
			{Key: "public_romm", Name: "Rom<M>", Checks: []monitor.Check{{Time: at, Up: rommUp}}},
		}, at)
	}
	read(now, true)
	srv := &Server{
		Config:   &config.Config{Title: "lab.bingo", Interval: time.Minute, Location: paris},
		Recorder: rec,
		Assets: fstest.MapFS{"index.html": {Data: []byte(`<html><head>
    <meta name="theme-color" content="#1443D6" />
  </head><body>page</body></html>`)}},
		Now: func() time.Time { return now },
	}
	path := "/"
	get := func(agent string) string {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("User-Agent", agent)
		res := httptest.NewRecorder()
		srv.Handler().ServeHTTP(res, req)
		return res.Body.String()
	}

	empty := strings.Repeat("⬜", 13)
	page := get("Mozilla/5.0 (compatible; Discordbot/2.0; +https://discordapp.com)")
	for _, want := range []string{
		`og:site_name" content="lab.bingo"`,
		`og:title" content="🟢 Tous les services sont opérationnels"`,
		// One reading: only the last square of the week is measured.
		empty + "🟩  Authentik · 100,00 %\n" + empty + "🟩  Rom&lt;M&gt; · 100,00 %\n7 derniers jours\"",
		`theme-color" content="#2FB56A"`,
	} {
		if !strings.Contains(page, want) {
			t.Fatalf("preview lacks %q:\n%s", want, page)
		}
	}
	if page := get("Mozilla/5.0 Firefox/130.0"); !strings.Contains(page, `theme-color" content="#1443D6"`) || !strings.Contains(page, "og:title") {
		t.Fatalf("a browser must keep the page colour and still get the preview:\n%s", page)
	}

	read(now.Add(time.Minute), false)
	read(now.Add(2*time.Minute), false)
	now = now.Add(2 * time.Minute)
	page = get("Discordbot/2.0")
	for _, want := range []string{
		`og:title" content="🟠 1 service perturbé"`,
		empty + "🟩  Authentik · 100,00 %\n" + empty + "🟧  Rom&lt;M&gt; · 33,33 % · ne répond pas\n7 derniers jours · incident en cours depuis 22:31",
		`theme-color" content="#F0A020"`,
	} {
		if !strings.Contains(page, want) {
			t.Fatalf("preview lacks %q:\n%s", want, page)
		}
	}

	// The range of the link decides the span of the bars.
	for query, want := range map[string]string{
		"?range=day":   strings.Repeat("⬜", 11) + "🟧  Rom&lt;M&gt; · 33,33 % · ne répond pas\n24 dernières heures · ",
		"?range=month": strings.Repeat("⬜", 14) + "🟧  Rom&lt;M&gt; · 33,33 % · ne répond pas\n30 derniers jours · ",
		"?range=all":   strings.Repeat("⬜", 14) + "🟧  Rom&lt;M&gt; · 33,33 % · ne répond pas\n90 derniers jours · ",
		"?range=never": empty + "🟧  Rom&lt;M&gt; · 33,33 % · ne répond pas\n7 derniers jours · ",
	} {
		path = "/" + query
		if page = get("Discordbot/2.0"); !strings.Contains(page, want) {
			t.Fatalf("preview of %s lacks %q:\n%s", query, want, page)
		}
	}
	path = "/"

	now = now.Add(time.Hour)
	if page = get("Discordbot/2.0"); !strings.Contains(page, `og:title" content="⚪ État inconnu"`) || !strings.Contains(page, "7 derniers jours · aucune mesure récente") {
		t.Fatalf("stale readings must not claim a state:\n%s", page)
	}
}
