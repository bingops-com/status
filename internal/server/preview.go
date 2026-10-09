package server

import (
	"fmt"
	"html"
	"math"
	"strings"
	"time"

	"github.com/bingops-com/status/internal/monitor"
)

// Link previews (Discord, Slack, ...) only read the HTML head and never run
// the page's script, so the current state is written there for them.

var previewMarks = map[string]string{"up": "🟢", "down": "🔴", "unknown": "⚪"}

// preview summarises a snapshot for a link preview: a title carrying the
// overall state, one mark per service, then the open incident or the
// availability, and the colour of the state.
func preview(snap monitor.Snapshot, loc *time.Location, now time.Time) (title, description, colour string) {
	down := 0
	marks := make([]string, 0, len(snap.Services))
	for _, s := range snap.Services {
		if s.State == "down" {
			down++
		}
		marks = append(marks, previewMarks[s.State]+" "+s.Name)
	}
	switch snap.State {
	case "ok":
		title, colour = "🟢 Tous les services sont opérationnels", "#2FB56A"
	case "partial":
		title, colour = fmt.Sprintf("🟠 %d services perturbés", down), "#F0A020"
		if down == 1 {
			title = "🟠 1 service perturbé"
		}
	case "down":
		title, colour = "🔴 Tous les services sont en panne", "#E5484D"
	default:
		title, colour = "⚪ État inconnu", "#8690A3"
	}

	lines := []string{}
	if len(marks) > 0 {
		lines = append(lines, strings.Join(marks, "   "))
	}
	switch {
	case snap.State == "unknown":
		lines = append(lines, "Aucune mesure récente")
	case openSince(snap) != nil:
		start := openSince(snap).In(loc)
		layout := "15:04"
		if start.Format(time.DateOnly) != now.In(loc).Format(time.DateOnly) {
			layout = "le 02/01 à 15:04"
		}
		lines = append(lines, "Incident en cours depuis "+start.Format(layout))
	default:
		if ratio, ok := meanUptime(snap); ok {
			lines = append(lines, fmt.Sprintf("Disponibilité sur %d jours : %s", snap.Days, percent(ratio)))
		}
	}
	return title, strings.Join(lines, "\n"), colour
}

// openSince is the start of the oldest incident still open, if any.
func openSince(snap monitor.Snapshot) *time.Time {
	var start *time.Time
	for i := range snap.Incidents {
		in := &snap.Incidents[i]
		if in.End == nil && (start == nil || in.Start.Before(*start)) {
			start = &in.Start
		}
	}
	return start
}

func meanUptime(snap monitor.Snapshot) (float64, bool) {
	sum, n := 0.0, 0
	for _, s := range snap.Services {
		if u := s.Uptime["all"]; u != nil {
			sum += *u
			n++
		}
	}
	if n == 0 {
		return 0, false
	}
	return sum / float64(n), true
}

// percent writes a ratio as the page does: two decimals, and never 100 %
// unless nothing was missed.
func percent(ratio float64) string {
	if ratio < 1 {
		ratio = math.Min(ratio, 0.9999)
	}
	return strings.Replace(fmt.Sprintf("%.2f", ratio*100), ".", ",", 1) + " %"
}

const themeColour = `<meta name="theme-color" content="#1443D6" />`

// withPreview adds the preview tags to the page's head. Discord takes the
// colour of its embed from theme-color; browsers keep the page's own.
func (s *Server) withPreview(page []byte, userAgent string) []byte {
	title, description, colour := preview(s.Recorder.Snapshot(s.now()), s.location(), s.now())
	tags := fmt.Sprintf(`<meta property="og:type" content="website" />
    <meta property="og:locale" content="fr_FR" />
    <meta property="og:site_name" content="%s" />
    <meta property="og:title" content="%s" />
    <meta property="og:description" content="%s" />
    <meta name="twitter:card" content="summary" />
  </head>`, html.EscapeString(s.Config.Title), html.EscapeString(title), html.EscapeString(description))
	out := strings.Replace(string(page), "</head>", tags, 1)
	if strings.Contains(userAgent, "Discordbot") {
		out = strings.Replace(out, themeColour, `<meta name="theme-color" content="`+colour+`" />`, 1)
	}
	return []byte(out)
}

func (s *Server) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Server) location() *time.Location {
	if s.Config.Location != nil {
		return s.Config.Location
	}
	return time.UTC
}
