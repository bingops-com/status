package monitor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestCorruptHistoryIsPreservedAndRecordingResumes(t *testing.T) {
	for name, raw := range map[string]string{
		"empty":          "",
		"truncated":      `{"services":`,
		"syntax":         `not JSON`,
		"null":           `null`,
		"partial decode": `{"services":{"a":{"name":"A"}},"nextId":"broken"}`,
		"null service":   `{"services":{"a":null}}`,
		"null incident":  `{"incidents":[null]}`,
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "history.json")
			if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
				t.Fatal(err)
			}
			r := recorder(t, path)
			backup := r.RecoveredHistory()
			if backup == "" || filepath.Dir(backup) != filepath.Dir(path) {
				t.Fatalf("missing same-volume backup: %q", backup)
			}
			kept, err := os.ReadFile(backup)
			if err != nil || string(kept) != raw {
				t.Fatalf("corrupt bytes were not preserved: %v", err)
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatalf("invalid history still at active path: %v", err)
			}
			now := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
			if !r.Empty() || r.Snapshot(now).State != "unknown" {
				t.Fatal("partially decoded or invented history survived")
			}
			// Another restart before the next reading must not lose the archive.
			r = recorder(t, path)
			if r.RecoveredHistory() != "" || !r.Empty() {
				t.Fatal("recovery is not idempotent")
			}
			r.Record([]Endpoint{{Key: "new", Name: "New", Checks: checks(now, "+")}}, now)
			if err := r.Save(); err != nil {
				t.Fatal(err)
			}
			again := recorder(t, path)
			if got := again.Snapshot(now); got.State != "ok" || len(got.Services) != 1 {
				t.Fatalf("new measurements not persisted: %+v", got)
			}
			kept, err = os.ReadFile(backup)
			if err != nil || string(kept) != raw {
				t.Fatal("archive changed after saving new history")
			}
		})
	}
}

func TestRecoveryDoesNotOverwriteEarlierArchive(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")
	var previous string
	for _, raw := range []string{"first bad file", "second bad file"} {
		if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
			t.Fatal(err)
		}
		r := recorder(t, path)
		if r.RecoveredHistory() == previous {
			t.Fatal("archive reused")
		}
		previous = r.RecoveredHistory()
	}
	archives, err := filepath.Glob(path + ".corrupt-*")
	if err != nil || len(archives) != 2 {
		t.Fatalf("archives = %v, %v", archives, err)
	}
	contents := map[string]bool{}
	for _, file := range archives {
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		contents[string(raw)] = true
	}
	if !contents["first bad file"] || !contents["second bad file"] {
		t.Fatal("an earlier archive was overwritten")
	}
}

func TestReadAndPreservationErrorsStillFailStartup(t *testing.T) {
	t.Run("read error", func(t *testing.T) {
		if _, err := NewRecorder(Options{Path: t.TempDir()}); err == nil {
			t.Fatal("read error silently reset history")
		}
	})
	t.Run("archive cannot be created", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("root bypasses filesystem permissions")
		}
		dir := t.TempDir()
		path := filepath.Join(dir, "history.json")
		if err := os.WriteFile(path, []byte("broken"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(dir, 0o500); err != nil {
			t.Fatal(err)
		}
		defer os.Chmod(dir, 0o700)
		if _, err := NewRecorder(Options{Path: path}); err == nil {
			t.Fatal("failed preservation silently reset history")
		}
		raw, err := os.ReadFile(path)
		if err != nil || string(raw) != "broken" {
			t.Fatal("original file lost when preservation failed")
		}
	})
}

func TestValidOlderHistoryWithoutCountersStillRecords(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")
	raw := `{"services":{"a":{"name":"A"}},"order":["a"]}`
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	r := recorder(t, path)
	if r.RecoveredHistory() != "" {
		t.Fatal("valid older history was quarantined")
	}
	now := time.Now()
	r.Record([]Endpoint{{Key: "a", Name: "A", Checks: checks(now, "+")}}, now)
	if got := r.Snapshot(now).State; got != "ok" {
		t.Fatalf("state = %q", got)
	}
}

func TestConcurrentSavesStayValidAndKeepLatestSnapshot(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")
	r := recorder(t, path)
	now := time.Now().UTC()
	var wg sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 20; i++ {
				r.Record([]Endpoint{{Key: "a", Name: "A", Checks: checks(now, "+")}}, now)
				if err := r.Save(); err != nil {
					t.Error(err)
					return
				}
				raw, err := os.ReadFile(path)
				if err != nil || !json.Valid(raw) {
					t.Errorf("published history is invalid: %v", err)
					return
				}
			}
		}()
	}
	wg.Wait()
	if got := recorder(t, path).Snapshot(now); !reflect.DeepEqual(got, r.Snapshot(now)) {
		t.Fatal("final snapshot lost on disk")
	}
	left, err := filepath.Glob(path + ".tmp-*")
	if err != nil || len(left) != 0 {
		t.Fatalf("temporary files leaked: %v, %v", left, err)
	}
}

func TestFailedSaveCleansTemporaryFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatal(err)
	}
	r := recorder(t, "")
	r.opts.Path = path
	if err := r.Save(); err == nil {
		t.Fatal("replacement of a directory unexpectedly succeeded")
	}
	left, err := filepath.Glob(path + ".tmp-*")
	if err != nil || len(left) != 0 {
		t.Fatalf("temporary files leaked: %v, %v", left, err)
	}
	if info, err := os.Stat(path); err != nil || !info.IsDir() {
		t.Fatal("failed replacement destroyed original path")
	}
}
