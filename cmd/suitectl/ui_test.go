package main

import (
	"os"
	"path/filepath"
	"runtime"
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
	ls, _, lo := packageIDs()
	host := &fakeHost{installed: map[string]bool{ls: true, lo: true}}
	withProcess(t, &scriptedRunner{respond: host.respond}, false)
	withStateDir(t)
	code, out := captureStdout(t, func() int { return run([]string{"plan", "--profile", "essential"}) })
	notice := "administrator approval"
	if runtime.GOOS == "linux" {
		notice = "community-built package"
	}
	if code != 0 || !strings.Contains(out, "vlc") || !strings.Contains(out, notice) ||
		strings.Contains(out, "Microsoft.VCRedist") {
		t.Fatalf("missing app should show only its own notices: code=%d output=%q", code, out)
	}
}

func TestPlanSummarizesAppsNotAvailableOnThisPlatform(t *testing.T) {
	root := catalogWithPendingApp(t)
	profile := filepath.Join(root, "profiles", "essential.yaml")
	data, err := os.ReadFile(profile)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(profile, append(data, []byte("  - beta\n")...), 0o600); err != nil {
		t.Fatal(err)
	}
	code, out := captureStdout(t, func() int {
		return run([]string{"plan", "--profile", "essential", "--platform", "linux", "--no-inspect", "--root", root})
	})
	if code != 0 || !strings.Contains(out, " not yet available on this platform") ||
		!strings.Contains(out, "beta             not yet available on linux (skipped)") ||
		strings.Contains(out, "Package source and identifier not verified") {
		t.Fatalf("code=%d out=%q", code, out)
	}
}
