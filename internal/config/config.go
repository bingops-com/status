// Package config reads the status page configuration.
package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Service adds what Gatus does not know about an endpoint: the words and the
// public address shown to visitors. Name matches the Gatus endpoint name.
type Service struct {
	Name        string `yaml:"name" json:"-"`
	Description string `yaml:"description" json:"description,omitempty"`
	URL         string `yaml:"url" json:"url,omitempty"`
}

// Notice is a hand-written announcement: planned maintenance or a word about
// a past incident.
type Notice struct {
	Date  string `yaml:"date" json:"date"`
	Title string `yaml:"title" json:"title"`
	Body  string `yaml:"body" json:"body,omitempty"`
}

type Config struct {
	Title string `yaml:"title"`
	// Gatus is the base address of the Gatus API, never sent to visitors.
	Gatus string `yaml:"gatus"`
	// Groups lists the Gatus groups that are public. Endpoints of any other
	// group are never recorded nor shown.
	Groups []string `yaml:"groups"`
	// Interval between two readings of Gatus.
	Interval time.Duration `yaml:"interval"`
	// Days of history kept and shown.
	Days int `yaml:"days"`
	// Timezone in which a day starts and ends.
	Timezone string `yaml:"timezone"`
	// FailureThreshold is the number of failed checks in a row that opens an
	// incident, so one slow answer is not reported as an outage.
	FailureThreshold int       `yaml:"failureThreshold"`
	Services         []Service `yaml:"services"`
	Notices          []Notice  `yaml:"notices"`

	Location *time.Location `yaml:"-"`
}

func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	c := &Config{Title: "lab.bingo", Interval: time.Minute, Days: 90, Timezone: "Europe/Paris", FailureThreshold: 2}
	if err := yaml.Unmarshal(raw, c); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if len(c.Groups) == 0 {
		return nil, errors.New("groups: list at least one public Gatus group")
	}
	if c.Interval < 10*time.Second {
		return nil, errors.New("interval: at least 10s")
	}
	if c.Days < 1 || c.Days > 365 {
		return nil, errors.New("days: between 1 and 365")
	}
	if c.FailureThreshold < 1 {
		return nil, errors.New("failureThreshold: at least 1")
	}
	if c.Location, err = time.LoadLocation(c.Timezone); err != nil {
		return nil, fmt.Errorf("timezone: %w", err)
	}
	for _, n := range c.Notices {
		if _, err := time.Parse(time.DateOnly, n.Date); err != nil {
			return nil, fmt.Errorf("notice %q: date must be YYYY-MM-DD", n.Title)
		}
	}
	return c, nil
}

// Service returns the visitor-facing details of a Gatus endpoint, if any.
func (c *Config) Service(name string) Service {
	for _, s := range c.Services {
		if s.Name == name {
			return s
		}
	}
	return Service{}
}
