package main

import (
	"strings"
	"testing"
)

func TestConciseHumanOutputWhenProfileIsComplete(t *testing.T) {
	ls, vlc, lo := packageIDs()
	host := &fakeHost{installed: map[string]bool{ls: true, vlc: true, lo: true}}
	withProcess(t, &scriptedRunner{respond: host.respond}, true)
	withStateDir(t)
	for _, args := range [][]string{
		{"plan", "--profile", "essential"},
		{"status"},
		{"apply", "--profile", "essential", "--yes"},
	} {
		code, out := captureStdout(t, func() int { return run(args) })
		if code != 0 {
			t.Fatalf("%v returned %d", args, code)
		}
		if strings.Contains(out, "administrator approval") || strings.Contains(out, "VCRedist") ||
			strings.Contains(out, "Journal:") || strings.Contains(out, "outside Sovereign") {
			t.Fatalf("%v contains irrelevant noise: %s", args, out)
		}
		if !strings.Contains(out, "3 installed") && !strings.Contains(out, "3 already installed") {
			t.Fatalf("%v lacks concise success summary: %s", args, out)
		}
	}
}

func TestVerboseDisplaysRequestedDetail(t *testing.T) {
	ls, vlc, lo := packageIDs()
	host := &fakeHost{installed: map[string]bool{ls: true, vlc: true, lo: true}}
	withProcess(t, &scriptedRunner{respond: host.respond}, false)
	withStateDir(t)
	code, out := captureStdout(t, func() int { return run([]string{"status", "--verbose"}) })
	if code != 0 || !strings.Contains(out, "Journal:") || !strings.Contains(out, "localsend") {
		t.Fatalf("verbose status missing detail: code=%d output=%q", code, out)
	}
	code, out = captureStdout(t, func() int { return run([]string{"plan", "--profile", "essential", "--verbose"}) })
	if code != 0 || !strings.Contains(out, "libreoffice") || !strings.Contains(out, "installed") {
		t.Fatalf("verbose plan missing detail: code=%d output=%q", code, out)
	}
}

func TestMissingAppShowsActionableAdminNotice(t *testing.T) {
	ls, vlc, lo := packageIDs()
	host := &fakeHost{installed: map[string]bool{ls: true, vlc: false, lo: true}}
	withProcess(t, &scriptedRunner{respond: host.respond}, false)
	withStateDir(t)
	code, out := captureStdout(t, func() int { return run([]string{"plan", "--profile", "essential"}) })
	if code != 0 || !strings.Contains(out, "vlc") || !strings.Contains(out, "administrator approval") ||
		strings.Contains(out, "Microsoft.VCRedist") {
		t.Fatalf("missing app should show only its own notices: code=%d output=%q", code, out)
	}
}
