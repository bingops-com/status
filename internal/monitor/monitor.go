// Package monitor keeps the availability history of the public services: it
// reads the checks made by Gatus, counts them per day and turns runs of
// failures into incidents.
package monitor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Check is one probe made by Gatus.
type Check struct {
	Time time.Time
	Up   bool
	Ms   int
}

// Endpoint is a monitored service with its recent checks.
type Endpoint struct {
	Key    string
	Name   string
	Checks []Check
}

type day struct {
	Checks int   `json:"checks"`
	Up     int   `json:"up"`
	Ms     int64 `json:"ms"`
}

type service struct {
	Name      string          `json:"name"`
	LastCheck time.Time       `json:"lastCheck"`
	Up        bool            `json:"up"`
	Ms        int             `json:"ms"`
	Fails     int             `json:"fails"`
	FailStart time.Time       `json:"failStart"`
	Open      int             `json:"open,omitempty"`
	Days      map[string]*day `json:"days"`
	Hours     map[string]*day `json:"hours,omitempty"`
}

// Incident is a period during which a service kept failing its checks.
type Incident struct {
	ID      int        `json:"id"`
	Key     string     `json:"key"`
	Service string     `json:"service"`
	Start   time.Time  `json:"start"`
	End     *time.Time `json:"end"`
}

type state struct {
	Services  map[string]*service `json:"services"`
	Incidents []*Incident         `json:"incidents"`
	NextID    int                 `json:"nextId"`
	ReadAt    time.Time           `json:"readAt"`
	Order     []string            `json:"order"`
}

// Options of a Recorder.
type Options struct {
	Path             string // file holding the history; empty keeps it in memory
	Days             int
	Location         *time.Location
	FailureThreshold int
	Interval         time.Duration
}

// Hours are kept for a week, for the short views; days for the whole history.
// They are keyed in UTC, where no hour is skipped or repeated.
const (
	hoursKept = 7 * 24
	hourKey   = "2006-01-02T15"
)

// Recorder accumulates checks. It is safe for concurrent use.
type Recorder struct {
	mu   sync.Mutex
	opts Options
	st   state
}

func NewRecorder(opts Options) (*Recorder, error) {
	r := &Recorder{opts: opts, st: state{Services: map[string]*service{}, NextID: 1}}
	if opts.Path == "" {
		return r, nil
	}
	raw, err := os.ReadFile(opts.Path)
	if os.IsNotExist(err) {
		return r, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &r.st); err != nil {
		return nil, fmt.Errorf("%s: %w", opts.Path, err)
	}
	if r.st.Services == nil {
		r.st.Services = map[string]*service{}
	}
	if r.st.NextID < 1 {
		r.st.NextID = 1
	}
	return r, nil
}

// Empty reports whether nothing has been recorded yet.
func (r *Recorder) Empty() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.st.Services) == 0
}

// Record adds the checks not seen yet and updates incidents. now is the time
// of the reading.
func (r *Recorder) Record(endpoints []Endpoint, now time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.st.Order = r.st.Order[:0]
	for _, e := range endpoints {
		r.st.Order = append(r.st.Order, e.Key)
		s := r.st.Services[e.Key]
		if s == nil {
			s = &service{Days: map[string]*day{}}
			r.st.Services[e.Key] = s
		}
		s.Name = e.Name
		checks := append([]Check(nil), e.Checks...)
		sort.Slice(checks, func(i, j int) bool { return checks[i].Time.Before(checks[j].Time) })
		for _, c := range checks {
			if !c.Time.After(s.LastCheck) {
				continue
			}
			r.add(e.Key, s, c)
		}
	}
	r.st.ReadAt = now
	r.prune(now)
}

func (r *Recorder) add(key string, s *service, c Check) {
	s.LastCheck, s.Up, s.Ms = c.Time, c.Up, c.Ms
	k := c.Time.In(r.opts.Location).Format(time.DateOnly)
	d := s.Days[k]
	if d == nil {
		d = &day{}
		s.Days[k] = d
	}
	if s.Hours == nil {
		s.Hours = map[string]*day{}
	}
	hk := c.Time.UTC().Format(hourKey)
	h := s.Hours[hk]
	if h == nil {
		h = &day{}
		s.Hours[hk] = h
	}
	d.Checks++
	h.Checks++
	if c.Up {
		d.Up++
		d.Ms += int64(c.Ms)
		h.Up++
		h.Ms += int64(c.Ms)
		s.Fails = 0
		if s.Open != 0 {
			end := c.Time
			if in := r.incident(s.Open); in != nil {
				in.End = &end
			}
			s.Open = 0
		}
		return
	}
	if s.Fails == 0 {
		s.FailStart = c.Time
	}
	s.Fails++
	if s.Open == 0 && s.Fails >= r.opts.FailureThreshold {
		in := &Incident{ID: r.st.NextID, Key: key, Service: s.Name, Start: s.FailStart}
		r.st.NextID++
		r.st.Incidents = append(r.st.Incidents, in)
		s.Open = in.ID
	}
}

func (r *Recorder) incident(id int) *Incident {
	for _, in := range r.st.Incidents {
		if in.ID == id {
			return in
		}
	}
	return nil
}

// prune drops what is older than the history kept.
func (r *Recorder) prune(now time.Time) {
	first := r.dayKey(now, r.opts.Days-1)
	firstHour := hourStart(now, hoursKept-1).Format(hourKey)
	for _, s := range r.st.Services {
		for k := range s.Days {
			if k < first {
				delete(s.Days, k)
			}
		}
		for k := range s.Hours {
			if k < firstHour {
				delete(s.Hours, k)
			}
		}
	}
	kept := r.st.Incidents[:0]
	for _, in := range r.st.Incidents {
		if in.End == nil || in.End.In(r.opts.Location).Format(time.DateOnly) >= first {
			kept = append(kept, in)
		}
	}
	r.st.Incidents = kept
}

// dayKey is the date `back` days before now, in the configured timezone.
func (r *Recorder) dayKey(now time.Time, back int) string {
	return now.In(r.opts.Location).AddDate(0, 0, -back).Format(time.DateOnly)
}

// hourStart is the beginning of the hour `back` hours before now.
func hourStart(now time.Time, back int) time.Time {
	return now.UTC().Truncate(time.Hour).Add(-time.Duration(back) * time.Hour)
}

// Save writes the history to its file, replacing it in one step.
func (r *Recorder) Save() error {
	if r.opts.Path == "" {
		return nil
	}
	r.mu.Lock()
	raw, err := json.Marshal(r.st)
	r.mu.Unlock()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(r.opts.Path), 0o755); err != nil {
		return err
	}
	tmp := r.opts.Path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, r.opts.Path)
}

// --- What visitors see ----------------------------------------------------

// DayView is one day of a service. Uptime is nil when nothing was measured.
type DayView struct {
	Date    string   `json:"date"`
	Uptime  *float64 `json:"uptime"`
	DownMin int      `json:"downMin"`
}

// HourView is one hour of a service, starting at Time.
type HourView struct {
	Time    time.Time `json:"time"`
	Uptime  *float64  `json:"uptime"`
	DownMin int       `json:"downMin"`
}

type ServiceView struct {
	ID         string              `json:"id"`
	Name       string              `json:"name"`
	State      string              `json:"state"` // up, down, unknown
	ResponseMs int                 `json:"responseMs"`
	Uptime     map[string]*float64 `json:"uptime"` // over 1, 7, 30 and all days kept
	History    []DayView           `json:"history"`
	Hours      []HourView          `json:"hours"` // the last week
}

type IncidentView struct {
	ID      int        `json:"id"`
	Service string     `json:"service"`
	Start   time.Time  `json:"start"`
	End     *time.Time `json:"end"`
}

type Snapshot struct {
	State     string         `json:"state"` // ok, partial, down, unknown
	ReadAt    *time.Time     `json:"readAt"`
	Stale     bool           `json:"stale"`
	Days      int            `json:"days"`
	Services  []ServiceView  `json:"services"`
	Incidents []IncidentView `json:"incidents"`
}

// Snapshot is the public view at `now`. Readings older than three intervals
// are stale: the current state of each service is then unknown.
func (r *Recorder) Snapshot(now time.Time) Snapshot {
	r.mu.Lock()
	defer r.mu.Unlock()
	snap := Snapshot{Days: r.opts.Days, Services: []ServiceView{}, Incidents: []IncidentView{}}
	if !r.st.ReadAt.IsZero() {
		t := r.st.ReadAt
		snap.ReadAt = &t
	}
	snap.Stale = r.st.ReadAt.IsZero() || now.Sub(r.st.ReadAt) > 3*r.opts.Interval

	up, down := 0, 0
	for _, key := range r.st.Order {
		s := r.st.Services[key]
		if s == nil {
			continue
		}
		v := ServiceView{ID: key, Name: s.Name, ResponseMs: s.Ms, Uptime: map[string]*float64{}}
		switch {
		case snap.Stale || s.LastCheck.IsZero():
			v.State = "unknown"
		case s.Up || s.Open == 0 && s.Fails < r.opts.FailureThreshold:
			v.State = "up"
			up++
		default:
			v.State = "down"
			down++
		}
		var checks, ups [3]int
		spans := [3]int{7, 30, r.opts.Days}
		for back := r.opts.Days - 1; back >= 0; back-- {
			k := r.dayKey(now, back)
			dv := DayView{Date: k}
			if d := s.Days[k]; d != nil && d.Checks > 0 {
				ratio := float64(d.Up) / float64(d.Checks)
				dv.Uptime = &ratio
				dv.DownMin = int(float64(d.Checks-d.Up)*r.opts.Interval.Minutes() + 0.5)
				for i, span := range spans {
					if back < span {
						checks[i] += d.Checks
						ups[i] += d.Up
					}
				}
			}
			v.History = append(v.History, dv)
		}
		for i, span := range spans {
			name := fmt.Sprint(span)
			if i == 2 {
				name = "all"
			}
			if checks[i] > 0 {
				ratio := float64(ups[i]) / float64(checks[i])
				v.Uptime[name] = &ratio
			} else {
				v.Uptime[name] = nil
			}
		}
		dayChecks, dayUps := 0, 0
		for back := hoursKept - 1; back >= 0; back-- {
			t := hourStart(now, back)
			hv := HourView{Time: t}
			if h := s.Hours[t.Format(hourKey)]; h != nil && h.Checks > 0 {
				ratio := float64(h.Up) / float64(h.Checks)
				hv.Uptime = &ratio
				hv.DownMin = int(float64(h.Checks-h.Up)*r.opts.Interval.Minutes() + 0.5)
				if back < 24 {
					dayChecks += h.Checks
					dayUps += h.Up
				}
			}
			v.Hours = append(v.Hours, hv)
		}
		v.Uptime["1"] = nil
		if dayChecks > 0 {
			ratio := float64(dayUps) / float64(dayChecks)
			v.Uptime["1"] = &ratio
		}
		snap.Services = append(snap.Services, v)
	}
	switch {
	case snap.Stale || up+down == 0:
		snap.State = "unknown"
	case down == 0:
		snap.State = "ok"
	case up == 0:
		snap.State = "down"
	default:
		snap.State = "partial"
	}

	shown := map[string]bool{}
	for _, key := range r.st.Order {
		shown[key] = true
	}
	for i := len(r.st.Incidents) - 1; i >= 0; i-- {
		in := r.st.Incidents[i]
		if shown[in.Key] {
			snap.Incidents = append(snap.Incidents, IncidentView{ID: in.ID, Service: in.Service, Start: in.Start, End: in.End})
		}
	}
	sort.SliceStable(snap.Incidents, func(i, j int) bool { return snap.Incidents[i].Start.After(snap.Incidents[j].Start) })
	return snap
}
