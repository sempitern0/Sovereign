package main

import (
	"strings"
	"testing"
)

func TestSetupRejectsUnexpectedArguments(t *testing.T) {
	for _, args := range [][]string{{"setup", "--profile", "offline"}, {"setup", "other"}, {"setup", "--json"}} {
		if got := run(args); got != 2 {
			t.Fatalf("%v returned %d; want 2", args, got)
		}
	}
}

func TestSetupWithoutConsentNeverInstalls(t *testing.T) {
	ls, _, _ := packageIDs()
	host := &fakeHost{installed: map[string]bool{ls: true}}
	runner := &scriptedRunner{respond: host.respond}
	withProcess(t, runner, true)
	dir := withStateDir(t)
	code, stderr := captureStderr(t, func() int { return run([]string{"setup"}) })
	if code != 3 || !strings.Contains(stderr, "--yes") {
		t.Fatalf("code=%d stderr=%s", code, stderr)
	}
	if len(runner.interactive) != 0 || len(readJournal(t, dir)) != 0 {
		t.Fatalf("setup changed host or journal without consent")
	}
	for _, call := range runner.calls {
		if call[1] == "install" {
			t.Fatalf("installed without consent: %v", call)
		}
	}
}

func TestSetupRepairsMissingSoftwareAndIsIdempotent(t *testing.T) {
	ls, vlc, lo := packageIDs()
	host := &fakeHost{installed: map[string]bool{ls: true}}
	runner := &scriptedRunner{respond: host.respond}
	withProcess(t, runner, true)
	withStateDir(t)
	if code := run([]string{"setup", "--yes"}); code != 0 {
		t.Fatalf("setup returned %d", code)
	}
	if !host.installed[vlc] || !host.installed[lo] {
		t.Fatalf("missing software was not installed: %v", host.installed)
	}
	before := len(runner.interactive)
	if code := run([]string{"setup"}); code != 0 {
		t.Fatalf("second setup returned %d", code)
	}
	if len(runner.interactive) != before {
		t.Fatalf("second setup installed apps again")
	}
}
