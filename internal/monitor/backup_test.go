package monitor

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestBackupRestoresMeasurementsAndIncidents(t *testing.T) {
	for _, missing := range []bool{false, true} {
		t.Run(map[bool]string{false: "corrupt", true: "missing"}[missing], func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "history.json")
			r := recorder(t, path)
			now := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
			r.Record([]Endpoint{{Key: "a", Name: "A", Checks: checks(now, "+--")}}, now)
			if err := r.Save(); err != nil {
				t.Fatal(err)
			}
			want := r.Snapshot(now)
			if missing {
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			} else if err := os.WriteFile(path, nil, 0o600); err != nil {
				t.Fatal(err)
			}
			restored := recorder(t, path)
			if restored.RestoredHistory() != path+".bak" || !reflect.DeepEqual(restored.Snapshot(now), want) {
				t.Fatal("backup lost measurements or incidents")
			}
			// Restore the primary even if Gatus is unavailable after startup.
			again := recorder(t, path)
			if again.RestoredHistory() != "" || !reflect.DeepEqual(again.Snapshot(now), want) {
				t.Fatal("repaired primary did not survive another startup")
			}
		})
	}
}

func TestBothCorruptFilesArePreserved(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")
	for _, name := range []string{path, path + ".bak"} {
		if err := os.WriteFile(name, []byte("{"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	r := recorder(t, path)
	if !r.Empty() || r.RestoredHistory() != "" || len(r.RecoveredHistories()) != 2 {
		t.Fatal("both invalid files must be preserved before starting fresh")
	}
	for _, archive := range r.RecoveredHistories() {
		raw, err := os.ReadFile(archive)
		if err != nil || string(raw) != "{" {
			t.Fatal("invalid bytes were not preserved")
		}
	}
}

func TestBackupReadErrorRemainsFatal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")
	if err := os.Mkdir(path+".bak", 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := NewRecorder(Options{Path: path}); err == nil {
		t.Fatal("backup I/O error was silently reset")
	}
}
