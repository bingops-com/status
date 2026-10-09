package monitor

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func recorder(t *testing.T, path string) *Recorder {
	t.Helper()
	r, err := NewRecorder(Options{Path: path, Days: 30, Location: time.UTC, FailureThreshold: 2, Interval: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func checks(start time.Time, outcomes string) []Check {
	out := []Check{}
	for i, c := range outcomes {
		out = append(out, Check{Time: start.Add(time.Duration(i) * time.Minute), Up: c == '+', Ms: 100})
	}
	return out
}

func TestRecordCountsEachCheckOnce(t *testing.T) {
	r := recorder(t, "")
	start := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	all := checks(start, "++++")
	r.Record([]Endpoint{{Key: "a", Name: "A", Checks: all[:3]}}, start.Add(3*time.Minute))
	r.Record([]Endpoint{{Key: "a", Name: "A", Checks: all}}, start.Add(4*time.Minute))
	d := r.st.Services["a"].Days["2026-10-09"]
	if d.Checks != 4 || d.Up != 4 {
		t.Fatalf("got %+v, want 4 checks counted once", d)
	}
}

func TestIncidentOpensAfterThresholdAndCloses(t *testing.T) {
	r := recorder(t, "")
	start := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	now := start.Add(10 * time.Minute)

	// A single failure is not an incident.
	r.Record([]Endpoint{{Key: "a", Name: "A", Checks: checks(start, "+-+")}}, now)
	if len(r.st.Incidents) != 0 {
		t.Fatalf("one failed check opened an incident")
	}
	if got := r.Snapshot(now).Services[0].State; got != "up" {
		t.Fatalf("state after one failure = %s, want up", got)
	}

	r.Record([]Endpoint{{Key: "a", Name: "A", Checks: checks(start.Add(3*time.Minute), "--")}}, now)
	snap := r.Snapshot(now)
	if len(snap.Incidents) != 1 || snap.Incidents[0].End != nil {
		t.Fatalf("incidents = %+v, want one ongoing", snap.Incidents)
	}
	if !snap.Incidents[0].Start.Equal(start.Add(3 * time.Minute)) {
		t.Fatalf("incident starts at %s, want the first failure of the run", snap.Incidents[0].Start)
	}
	if snap.State != "down" || snap.Services[0].State != "down" {
		t.Fatalf("state = %s/%s, want down", snap.State, snap.Services[0].State)
	}

	r.Record([]Endpoint{{Key: "a", Name: "A", Checks: checks(start.Add(5*time.Minute), "+")}}, now)
	snap = r.Snapshot(now)
	if snap.Incidents[0].End == nil || !snap.Incidents[0].End.Equal(start.Add(5*time.Minute)) {
		t.Fatalf("incident not closed by the first success: %+v", snap.Incidents[0])
	}
	if snap.State != "ok" {
		t.Fatalf("state = %s, want ok", snap.State)
	}
}

func TestSnapshotUptimeAndHistory(t *testing.T) {
	r := recorder(t, "")
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	r.Record([]Endpoint{{Key: "a", Name: "A", Checks: checks(now.Add(-24*time.Hour), "++--")}}, now)
	r.Record([]Endpoint{{Key: "a", Name: "A", Checks: checks(now.Add(-10*time.Minute), "++++")}}, now)
	s := r.Snapshot(now).Services[0]
	if len(s.History) != 30 {
		t.Fatalf("history has %d days, want 30", len(s.History))
	}
	last, before := s.History[29], s.History[28]
	if last.Date != "2026-10-09" || *last.Uptime != 1 || before.Date != "2026-10-08" || *before.Uptime != 0.5 || before.DownMin != 2 {
		t.Fatalf("unexpected days: %+v %+v", before, last)
	}
	if s.History[0].Uptime != nil {
		t.Fatalf("a day without checks must have no uptime")
	}
	if got := *s.Uptime["7"]; got != 0.75 {
		t.Fatalf("7-day uptime = %v, want 0.75", got)
	}
}

func TestStaleReadingsAreUnknown(t *testing.T) {
	r := recorder(t, "")
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	r.Record([]Endpoint{{Key: "a", Name: "A", Checks: checks(now, "+")}}, now)
	snap := r.Snapshot(now.Add(10 * time.Minute))
	if !snap.Stale || snap.State != "unknown" || snap.Services[0].State != "unknown" {
		t.Fatalf("old readings must be unknown: %+v", snap)
	}
}

func TestHistoryOlderThanKeptIsDropped(t *testing.T) {
	r := recorder(t, "")
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	r.Record([]Endpoint{{Key: "a", Name: "A", Checks: checks(now.AddDate(0, 0, -40), "--+")}}, now)
	if len(r.st.Services["a"].Days) != 0 || len(r.st.Incidents) != 0 {
		t.Fatalf("history older than the days kept survived: %+v", r.st)
	}
}

func TestSaveAndReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")
	r := recorder(t, path)
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	r.Record([]Endpoint{{Key: "a", Name: "A", Checks: checks(now, "+--")}}, now)
	if err := r.Save(); err != nil {
		t.Fatal(err)
	}
	again := recorder(t, path)
	snap := again.Snapshot(now)
	if len(snap.Services) != 1 || len(snap.Incidents) != 1 {
		t.Fatalf("history lost on reload: %+v", snap)
	}
}

func TestGatusKeepsOnlyPublicGroupsAndNoAddress(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`[
		  {"name":"Argo CD","group":"Plateforme","key":"plateforme_argo-cd","results":[{"success":true,"duration":1000000,"timestamp":"2026-10-09T12:00:00Z","hostname":"internal.svc"}]},
		  {"name":"RomM","group":"Public","key":"public_romm","results":[{"success":false,"duration":250000000,"timestamp":"2026-10-09T12:00:00Z","hostname":"rom.lab.bingo","errors":["secret detail"]}]}
		]`))
	}))
	defer srv.Close()
	endpoints, err := (&Gatus{URL: srv.URL, Groups: []string{"Public"}}).Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(endpoints) != 1 || endpoints[0].Name != "RomM" || endpoints[0].Checks[0].Up || endpoints[0].Checks[0].Ms != 250 {
		t.Fatalf("unexpected endpoints: %+v", endpoints)
	}
	r := recorder(t, "")
	now := time.Date(2026, 10, 9, 12, 0, 30, 0, time.UTC)
	r.Record(endpoints, now)
	raw, _ := json.Marshal(r.Snapshot(now))
	for _, leak := range []string{"Argo", "internal.svc", "secret detail", "rom.lab.bingo"} {
		if strings.Contains(string(raw), leak) {
			t.Fatalf("snapshot leaks %q: %s", leak, raw)
		}
	}
}
