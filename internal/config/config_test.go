package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func write(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "status.yaml")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestDefaultConfigLoads(t *testing.T) {
	c, err := Load("../../config/status.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if c.Interval != time.Minute || c.Days != 90 || len(c.Groups) != 1 || c.Service("RomM").URL == "" {
		t.Fatalf("unexpected default configuration: %+v", c)
	}
}

func TestInvalidConfigIsRefused(t *testing.T) {
	for name, body := range map[string]string{
		"no public group": "gatus: http://gatus\n",
		"interval":        "groups: [Public]\ninterval: 1s\n",
		"days":            "groups: [Public]\ndays: 0\n",
		"timezone":        "groups: [Public]\ntimezone: Mars/Olympus\n",
		"notice date":     "groups: [Public]\nnotices: [{date: tomorrow, title: Maintenance}]\n",
	} {
		if _, err := Load(write(t, body)); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}
