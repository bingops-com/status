package monitor

import (
	"context"
	"hash/fnv"
	"time"
)

// Demo is a made-up source for working on the interface without Gatus. Its
// outages are derived from the clock, so two runs show the same history.
type Demo struct {
	Interval time.Duration
	// Outage keeps one service failing right now, to see the page in trouble.
	Outage bool
}

var demoServices = []string{"Authentik", "Portfolio", "RomM", "ROMarr"}

// check decides the outcome of a probe: a few fixed windows of failure per
// service, spread over the past weeks.
func (d Demo) check(name string, t time.Time) Check {
	h := fnv.New32a()
	h.Write([]byte(name))
	seed := int64(h.Sum32() % 97)
	hour := t.Unix() / 3600
	down := false
	// One outage every month or two, of 12 to 50 minutes.
	period := (31 + seed%20) * 24
	if (hour+seed*7)%period == 0 && int64(t.Minute()) < 12+seed%39 {
		down = true
	}
	ago := time.Since(t)
	// A long one, about 3 hours, 23 days ago for RomM.
	if name == "RomM" && ago > 23*24*time.Hour && ago < 23*24*time.Hour+3*time.Hour {
		down = true
	}
	if d.Outage && name == "RomM" && ago < 25*time.Minute {
		down = true
	}
	ms := 80 + int(seed) + int(t.Unix()/60%40)
	return Check{Time: t, Up: !down, Ms: ms}
}

func (d Demo) Fetch(context.Context) ([]Endpoint, error) {
	now := time.Now().Truncate(d.Interval)
	endpoints := []Endpoint{}
	for _, name := range demoServices {
		e := Endpoint{Key: "public_" + name, Name: name}
		for i := 5; i >= 0; i-- {
			e.Checks = append(e.Checks, d.check(name, now.Add(-time.Duration(i)*d.Interval)))
		}
		endpoints = append(endpoints, e)
	}
	return endpoints, nil
}

// Seed fills an empty recorder with the made-up history of the days kept.
func (d Demo) Seed(r *Recorder) {
	now := time.Now().Truncate(d.Interval)
	start := now.AddDate(0, 0, -r.opts.Days+12) // the first days stay unmeasured
	for _, name := range demoServices {
		e := Endpoint{Key: "public_" + name, Name: name}
		for t := start; t.Before(now); t = t.Add(d.Interval) {
			e.Checks = append(e.Checks, d.check(name, t))
		}
		r.Record([]Endpoint{e}, now)
	}
	r.st.Order = nil
}
