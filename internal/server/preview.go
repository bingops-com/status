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

// A square of the bar follows the day marks of the page: green without a
// noticeable interruption, orange under half an hour, red beyond, white when
// nothing was measured.
const (
	noticeableMin = 2
	longMin       = 30
)

// span is a period a link preview can show: its bar has `squares` squares,
// read from the hours of the last week or from the days kept.
type span struct {
	uptime  string // key of ServiceView.Uptime
	hours   int    // read that many hours; 0 reads days
	days    int    // read that many days; 0 reads all of them
	squares int
	caption string
}

// The page's ?range= picks the span, so that a shared link previews what its
// sender was looking at. The week is the default, as on the page.
var spans = map[string]span{
	"day":   {uptime: "1", hours: 24, squares: 12, caption: "24 dernières heures"},
	"week":  {uptime: "7", hours: 7 * 24, squares: 14, caption: "7 derniers jours"},
	"month": {uptime: "30", days: 30, squares: 15, caption: "30 derniers jours"},
	"all":   {uptime: "all", squares: 15},
}

func spanOf(name string, days int) span {
	sp, ok := spans[name]
	if !ok {
		sp = spans["week"]
	}
	if sp.hours == 0 && (sp.days == 0 || sp.days >= days) {
		sp = spans["all"]
		sp.caption = fmt.Sprintf("%d derniers jours", days)
	}
	return sp
}

// measure is the uptime and the minutes lost of one hour or one day.
type measure struct {
	uptime  *float64
	downMin int
}

func measures(s monitor.ServiceView, sp span) []measure {
	out := []measure{}
	if sp.hours > 0 {
		for _, h := range s.Hours[max(0, len(s.Hours)-sp.hours):] {
			out = append(out, measure{h.Uptime, h.DownMin})
		}
		return out
	}
	history := s.History
	if sp.days > 0 {
		history = history[max(0, len(history)-sp.days):]
	}
	for _, d := range history {
		out = append(out, measure{d.Uptime, d.DownMin})
	}
	return out
}

// bar draws the measures as n squares, oldest first, each one covering the
// same share of them.
func bar(ms []measure, n int) string {
	n = min(n, len(ms))
	var b strings.Builder
	for i := 0; i < n; i++ {
		measured, downMin := false, 0
		for _, m := range ms[i*len(ms)/n : (i+1)*len(ms)/n] {
			if m.uptime != nil {
				measured = true
				downMin += m.downMin
			}
		}
		switch {
		case !measured:
			b.WriteString("⬜")
		case downMin < noticeableMin:
			b.WriteString("🟩")
		case downMin < longMin:
			b.WriteString("🟧")
		default:
			b.WriteString("🟥")
		}
	}
	return b.String()
}

// preview summarises a snapshot for a link preview: a title carrying the
// overall state, then for each service its bar and availability over the
// span, the open incident if any, and the colour of the state. The bar comes
// first on its line: the names have no fixed width to align it after them.
func preview(snap monitor.Snapshot, sp span, loc *time.Location, now time.Time) (title, description, colour string) {
	down := 0
	lines := []string{}
	for _, s := range snap.Services {
		line := s.Name
		if b := bar(measures(s, sp), sp.squares); b != "" {
			line = b + "  " + line
		}
		if u := s.Uptime[sp.uptime]; u != nil {
			line += " · " + percent(*u)
		}
		if s.State == "down" {
			down++
			line += " · ne répond pas"
		}
		lines = append(lines, line)
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

	foot := sp.caption
	switch {
	case snap.State == "unknown":
		foot += " · aucune mesure récente"
	case openSince(snap) != nil:
		start := openSince(snap).In(loc)
		layout := "15:04"
		if start.Format(time.DateOnly) != now.In(loc).Format(time.DateOnly) {
			layout = "le 02/01 à 15:04"
		}
		foot += " · incident en cours depuis " + start.Format(layout)
	}
	lines = append(lines, foot)
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
func (s *Server) withPreview(page []byte, userAgent, shown string) []byte {
	snap := s.Recorder.Snapshot(s.now())
	title, description, colour := preview(snap, spanOf(shown, snap.Days), s.location(), s.now())
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
