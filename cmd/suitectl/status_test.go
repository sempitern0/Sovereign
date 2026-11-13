package main

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/sempitern0/Sovereign/internal/state"
)

func captureStdout(t *testing.T, fn func() int) (int, string) {
	t.Helper()
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	previous := os.Stdout
	os.Stdout = write
	code := fn()
	os.Stdout = previous
	write.Close()
	out, _ := io.ReadAll(read)
	return code, string(out)
}

func statusJSON(t *testing.T) StatusReport {
	t.Helper()
	code, out := captureStdout(t, func() int { return run([]string{"status", "--json"}) })
	if code != 0 {
		t.Fatalf("status exited %d: %s", code, out)
	}
	var r StatusReport
	if err := json.Unmarshal([]byte(out), &r); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	return r
}

func byApp(r StatusReport) map[string]AppStatus {
	m := map[string]AppStatus{}
	for _, a := range r.Applications {
		m[a.Application] = a
	}
	return m
}

func TestStatusReportsDriftAgainstJournal(t *testing.T) {
	ls, _, lo := packageIDs()
	host := &fakeHost{installed: map[string]bool{ls: true, lo: true}}
	withProcess(t, &scriptedRunner{respond: host.respond}, false)
	dir := withStateDir(t)
	store := state.Store{Dir: dir}
	if err := store.Record(
		state.Entry{Application: "vlc", Platform: runtime.GOOS, Result: state.ResultInstalled, Changed: true},
		state.Entry{Application: "libreoffice", Platform: runtime.GOOS, Result: state.ResultFailed, ErrorClass: "install_failed"},
		state.Entry{Application: "beta", Platform: runtime.GOOS, Result: state.ResultInstalled},
		state.Entry{Application: "vlc", Platform: "android", Result: state.ResultInstalled},
	); err != nil {
		t.Fatal(err)
	}
	r := statusJSON(t)
	apps := byApp(r)
	if r.SchemaVersion != 1 || r.Profile != "essential" || r.Platform != runtime.GOOS || r.Complete {
		t.Fatalf("report header %+v", r)
	}
	if a := apps["vlc"]; a.Observed != observedMissing || !strings.Contains(a.Note, "no longer present") || !strings.Contains(a.NextStep, "apply --profile essential --yes") {
		t.Fatalf("vlc %+v", a)
	}
	if a := apps["libreoffice"]; a.Observed != observedInstalled || a.LastErrorClass != "install_failed" || !strings.Contains(a.Note, "last recorded attempt failed") {
		t.Fatalf("libreoffice %+v", a)
	}
	if a := apps["localsend"]; a.Observed != observedInstalled || !strings.Contains(a.Note, "outside Sovereign") {
		t.Fatalf("localsend %+v", a)
	}
	if len(r.OtherRecorded) != 1 || r.OtherRecorded[0].Application != "beta" {
		t.Fatalf("other recorded %+v", r.OtherRecorded)
	}
}

func TestStatusCompleteWhenEverythingIsInstalled(t *testing.T) {
	ls, vlc, lo := packageIDs()
	host := &fakeHost{installed: map[string]bool{ls: true, vlc: true, lo: true}}
	withProcess(t, &scriptedRunner{respond: host.respond}, false)
	withStateDir(t)
	if r := statusJSON(t); !r.Complete {
		t.Fatalf("expected complete: %+v", r)
	}
	code, out := captureStdout(t, func() int { return run([]string{"status"}) })
	if code != 0 || !strings.Contains(out, "Profile complete") || !strings.Contains(out, "Journal:") {
		t.Fatalf("code=%d out=%q", code, out)
	}
}

func TestStatusIsStrictlyReadOnly(t *testing.T) {
	host := &fakeHost{installed: map[string]bool{}}
	runner := &scriptedRunner{respond: host.respond}
	withProcess(t, runner, true)
	dir := withStateDir(t)
	if err := os.WriteFile(filepath.Join(dir, "journal.json"), []byte("{corrupt"), 0o600); err != nil {
		t.Fatal(err)
	}
	code, stderr := captureStderr(t, func() int {
		c, _ := captureStdout(t, func() int { return run([]string{"status"}) })
		return c
	})
	if code != 0 || !strings.Contains(stderr, "journal not available") {
		t.Fatalf("code=%d stderr=%q", code, stderr)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 || entries[0].Name() != "journal.json" {
		t.Fatalf("status modified the state directory: %v", entries)
	}
	if data, _ := os.ReadFile(filepath.Join(dir, "journal.json")); string(data) != "{corrupt" {
		t.Fatal("status rewrote the journal")
	}
	if len(runner.interactive) != 0 {
		t.Fatal("status ran an interactive command")
	}
	for _, call := range runner.calls {
		if call[1] != "list" && call[1] != "remotes" {
			t.Fatalf("status ran a mutating command: %v", call)
		}
	}
}

func TestStatusRejectsPositionalArguments(t *testing.T) {
	if got := run([]string{"status", "extra"}); got != 2 {
		t.Fatalf("got %d", got)
	}
}
