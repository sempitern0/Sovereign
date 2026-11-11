package state

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func fixedStore(t *testing.T) Store {
	t.Helper()
	return Store{Dir: filepath.Join(t.TempDir(), "state"), Now: func() time.Time { return time.Date(2026, 11, 11, 10, 0, 0, 0, time.UTC) }}
}

func TestRecordUpsertsAndSorts(t *testing.T) {
	s := fixedStore(t)
	if err := s.Record(Entry{Application: "vlc", Platform: "linux", Result: ResultFailed, ErrorClass: "install_failed"},
		Entry{Application: "localsend", Platform: "linux", Result: ResultInstalled, Changed: true}); err != nil {
		t.Fatal(err)
	}
	if err := s.Record(Entry{Application: "vlc", Platform: "linux", Result: ResultInstalled, Changed: true}); err != nil {
		t.Fatal(err)
	}
	j, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(j.Entries) != 2 || j.Entries[0].Application != "localsend" || j.Entries[1].Result != ResultInstalled || j.Entries[1].ErrorClass != "" {
		t.Fatalf("journal %+v", j)
	}
	if j.Entries[0].UpdatedAt != "2026-11-11T10:00:00Z" {
		t.Fatalf("timestamp %q", j.Entries[0].UpdatedAt)
	}
	info, err := os.Stat(filepath.Join(s.Dir, journalFile))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o077 != 0 && os.PathSeparator == '/' {
		t.Fatalf("journal readable by others: %v", info.Mode())
	}
	leftovers, _ := filepath.Glob(filepath.Join(s.Dir, ".journal.json.tmp-*"))
	if len(leftovers) != 0 {
		t.Fatalf("temporary files left: %v", leftovers)
	}
}

func TestCorruptJournalIsMovedAsideNotDeleted(t *testing.T) {
	s := fixedStore(t)
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s.Dir, journalFile), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := s.Record(Entry{Application: "vlc", Platform: "linux", Result: ResultInstalled})
	if err == nil || !strings.Contains(err.Error(), "moved to") {
		t.Fatalf("expected warning, got %v", err)
	}
	aside, _ := filepath.Glob(filepath.Join(s.Dir, journalFile+".corrupt-*"))
	if len(aside) != 1 {
		t.Fatalf("corrupt journal not preserved: %v", aside)
	}
	data, _ := os.ReadFile(filepath.Join(s.Dir, journalFile))
	var j Journal
	if json.Unmarshal(data, &j) != nil || len(j.Entries) != 1 {
		t.Fatalf("new journal not written: %s", data)
	}
}

func TestLockExcludesConcurrentRuns(t *testing.T) {
	s := fixedStore(t)
	release, err := s.Lock()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Lock(); !errors.Is(err, ErrLocked) {
		t.Fatalf("second lock: %v", err)
	}
	release()
	again, err := s.Lock()
	if err != nil {
		t.Fatalf("lock after release: %v", err)
	}
	again()
}

func TestStaleLockFromDeadProcessIsReplaced(t *testing.T) {
	s := fixedStore(t)
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		t.Fatal(err)
	}
	// PIDs this large are not assigned on Linux (pid_max <= 4194304) or Windows in practice.
	if err := os.WriteFile(filepath.Join(s.Dir, lockFile), []byte("2147483646\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	release, err := s.Lock()
	if err != nil {
		t.Fatalf("stale lock not replaced: %v", err)
	}
	release()
	if _, err := os.Stat(filepath.Join(s.Dir, lockFile)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("lock not removed on release")
	}
}

func TestDefaultDirIsAbsolute(t *testing.T) {
	dir, err := DefaultDir()
	if err != nil {
		t.Skip("no state directory in this environment:", err)
	}
	if !filepath.IsAbs(dir) {
		t.Fatalf("relative state dir %q", dir)
	}
}

func TestPeekNeverModifiesFiles(t *testing.T) {
	s := fixedStore(t)
	if j, err := s.Peek(); err != nil || len(j.Entries) != 0 {
		t.Fatalf("missing journal: %v %v", j, err)
	}
	if _, err := os.Stat(s.Dir); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("Peek created the state directory")
	}
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(s.Path(), []byte("garbage"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Peek(); !errors.Is(err, ErrUnreadableJournal) {
		t.Fatalf("got %v", err)
	}
	entries, _ := os.ReadDir(s.Dir)
	if len(entries) != 1 {
		t.Fatalf("Peek touched the directory: %v", entries)
	}
}
