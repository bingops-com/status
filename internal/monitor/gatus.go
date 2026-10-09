package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Source provides the latest checks of the public services.
type Source interface {
	Fetch(ctx context.Context) ([]Endpoint, error)
}

// Gatus reads the endpoints of the public groups from a Gatus instance. Only
// the name, the outcome and the duration of a check are kept: addresses,
// conditions and error messages never leave this function.
type Gatus struct {
	URL    string
	Groups []string
	Client *http.Client
}

type gatusStatus struct {
	Name    string `json:"name"`
	Group   string `json:"group"`
	Key     string `json:"key"`
	Results []struct {
		Success   bool      `json:"success"`
		Duration  int64     `json:"duration"` // nanoseconds
		Timestamp time.Time `json:"timestamp"`
	} `json:"results"`
}

func (g *Gatus) Fetch(ctx context.Context) ([]Endpoint, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(g.URL, "/")+"/api/v1/endpoints/statuses?page=1&pageSize=100", nil)
	if err != nil {
		return nil, err
	}
	client := g.Client
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gatus answered %d", res.StatusCode)
	}
	var statuses []gatusStatus
	if err := json.NewDecoder(res.Body).Decode(&statuses); err != nil {
		return nil, err
	}
	public := map[string]bool{}
	for _, name := range g.Groups {
		public[name] = true
	}
	endpoints := []Endpoint{}
	for _, s := range statuses {
		if !public[s.Group] {
			continue
		}
		e := Endpoint{Key: s.Key, Name: s.Name}
		for _, r := range s.Results {
			e.Checks = append(e.Checks, Check{Time: r.Timestamp, Up: r.Success, Ms: int(r.Duration / int64(time.Millisecond))})
		}
		endpoints = append(endpoints, e)
	}
	return endpoints, nil
}

// Watch reads the source at every interval until the context ends, and saves
// the history after each reading.
func (r *Recorder) Watch(ctx context.Context, src Source, onError func(error)) {
	read := func() {
		fetch, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
		endpoints, err := src.Fetch(fetch)
		if err != nil {
			onError(err)
			return
		}
		r.Record(endpoints, time.Now())
		if err := r.Save(); err != nil {
			onError(err)
		}
	}
	read()
	tick := time.NewTicker(r.opts.Interval)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			read()
		}
	}
}
