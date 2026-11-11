package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/sempitern0/Sovereign/internal/state"
)

// TestMain keeps every test away from the user's real state directory and from
// real provider detection.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "suitectl-state-")
	if err != nil {
		panic(err)
	}
	newStore = func() (state.Store, error) { return state.Store{Dir: dir}, nil }
	detectProviders = func() map[string]bool { return map[string]bool{"winget": true, "flatpak": true} }
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// withStateDir gives one test its own journal directory.
func withStateDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	previous := newStore
	newStore = func() (state.Store, error) { return state.Store{Dir: dir}, nil }
	t.Cleanup(func() { newStore = previous })
	return dir
}

// fakeHost simulates WinGet or Flatpak with a set of installed package IDs.
// Packages listed in failing make their install command fail.
type fakeHost struct {
	installed map[string]bool
	failing   map[string]bool
}

func (h *fakeHost) respond(program string, args []string) (string, error) {
	switch {
	case program == "flatpak" && args[0] == "remotes":
		return "flathub\thttps://dl.flathub.org/repo/\n", nil
	case args[0] == "install":
		id := args[len(args)-1]
		if program == "winget" {
			id = args[2] // install --id <id> ...
		}
		if h.failing[id] {
			return "", errors.New(program + " failed: exit status 1 (C:\\Users\\someone\\log.txt)")
		}
		h.installed[id] = true
		return "", nil
	case program == "flatpak":
		var out strings.Builder
		for id := range h.installed {
			out.WriteString(id + "\n")
		}
		return out.String(), nil
	default: // winget list --id <id>
		if h.installed[args[2]] {
			return "Name " + args[2] + " 1.0 winget\n", nil
		}
		return "No installed package found matching input criteria.", nil
	}
}

func packageIDs() (localsend, vlc, libreoffice string) {
	if runtime.GOOS == "windows" {
		return "LocalSend.LocalSend", "VideoLAN.VLC", "TheDocumentFoundation.LibreOffice"
	}
	return "org.localsend.localsend_app", "org.videolan.VLC", "org.libreoffice.LibreOffice"
}

func readJournal(t *testing.T, dir string) map[string]state.Entry {
	t.Helper()
	j, err := (state.Store{Dir: dir}).Load()
	if err != nil {
		t.Fatal(err)
	}
	byApp := map[string]state.Entry{}
	for _, e := range j.Entries {
		byApp[e.Application] = e
	}
	return byApp
}

func TestApplyProfileNeedsOneConsentAndChangesNothingWithout(t *testing.T) {
	ls, _, _ := packageIDs()
	host := &fakeHost{installed: map[string]bool{ls: true}}
	runner := &scriptedRunner{respond: host.respond}
	withProcess(t, runner, true)
	dir := withStateDir(t)
	code, stderr := captureStderr(t, func() int { return run([]string{"apply", "--profile", "essential"}) })
	if code != 3 || !strings.Contains(stderr, "2 application(s) would be installed: ") || !strings.Contains(stderr, "--yes") {
		t.Fatalf("code=%d stderr=%q", code, stderr)
	}
	if len(runner.interactive) != 0 {
		t.Fatalf("installed without consent: %v", runner.interactive)
	}
	for _, call := range runner.calls {
		if call[1] == "install" {
			t.Fatalf("installed without consent: %v", call)
		}
	}
	if entries := readJournal(t, dir); len(entries) != 0 {
		t.Fatalf("journal written without consent: %v", entries)
	}
}

func TestApplyProfileInstallsMissingAndKeepsGoingAfterFailure(t *testing.T) {
	ls, vlc, lo := packageIDs()
	host := &fakeHost{installed: map[string]bool{ls: true}, failing: map[string]bool{lo: true}}
	runner := &scriptedRunner{respond: host.respond}
	withProcess(t, runner, true)
	dir := withStateDir(t)
	code, stderr := captureStderr(t, func() int { return run([]string{"apply", "--profile", "essential", "--yes"}) })
	if code != 1 || !strings.Contains(stderr, "profile is incomplete") {
		t.Fatalf("code=%d stderr=%q", code, stderr)
	}
	if !host.installed[vlc] || host.installed[lo] {
		t.Fatalf("VLC must install despite the LibreOffice failure: %v", host.installed)
	}
	entries := readJournal(t, dir)
	if entries["localsend"].Result != state.ResultAlreadyInstalled || entries["vlc"].Result != state.ResultInstalled ||
		entries["libreoffice"].Result != state.ResultFailed || entries["libreoffice"].ErrorClass != "install_failed" {
		t.Fatalf("journal %+v", entries)
	}
	data, err := os.ReadFile(filepath.Join(dir, "journal.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "Users") || strings.Contains(string(data), "exit status") {
		t.Fatalf("journal leaks provider output: %s", data)
	}
}

func TestApplyProfileSecondRunChangesNothing(t *testing.T) {
	ls, vlc, lo := packageIDs()
	host := &fakeHost{installed: map[string]bool{ls: true, vlc: true, lo: true}}
	runner := &scriptedRunner{respond: host.respond}
	withProcess(t, runner, true)
	withStateDir(t)
	if code := run([]string{"apply", "--profile", "essential", "--yes"}); code != 0 {
		t.Fatalf("code=%d", code)
	}
	if len(runner.interactive) != 0 {
		t.Fatalf("reinstalled: %v", runner.interactive)
	}
	for _, call := range runner.calls {
		if call[1] == "install" {
			t.Fatalf("reinstalled: %v", call)
		}
	}
}

func TestApplyProfileBlocksDependentsOfFailedApps(t *testing.T) {
	root := catalogWithPendingApp(t)
	vlcPath := filepath.Join(root, "catalog/apps/vlc.yaml")
	data, err := os.ReadFile(vlcPath)
	if err != nil {
		t.Fatal(err)
	}
	withDep := strings.Replace(string(data), "capabilities:", "depends_on:\n  - libreoffice\ncapabilities:", 1)
	if err := os.WriteFile(vlcPath, []byte(withDep), 0o600); err != nil {
		t.Fatal(err)
	}
	ls, vlc, lo := packageIDs()
	host := &fakeHost{installed: map[string]bool{ls: true}, failing: map[string]bool{lo: true}}
	withProcess(t, &scriptedRunner{respond: host.respond}, true)
	withStateDir(t)
	read, write, _ := os.Pipe()
	stdout := os.Stdout
	os.Stdout = write
	code := run([]string{"apply", "--profile", "essential", "--yes", "--root", root})
	os.Stdout = stdout
	write.Close()
	raw, _ := io.ReadAll(read)
	out := string(raw)
	if code != 1 || host.installed[vlc] || !strings.Contains(out, "blocked (dependency libreoffice was not installed)") {
		t.Fatalf("code=%d installed=%v out=%q", code, host.installed, out)
	}
}

func TestApplyProfileRefusesWhileAnotherApplyRuns(t *testing.T) {
	_, vlc, _ := packageIDs()
	host := &fakeHost{installed: map[string]bool{}}
	runner := &scriptedRunner{respond: host.respond}
	withProcess(t, runner, true)
	dir := withStateDir(t)
	release, err := (state.Store{Dir: dir}).Lock()
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	for _, args := range [][]string{{"apply", "--profile", "essential", "--yes"}, {"apply", "--app", "vlc", "--yes"}} {
		code, stderr := captureStderr(t, func() int { return run(args) })
		if code != 1 || !strings.Contains(stderr, "another suitectl apply is running") {
			t.Fatalf("%v: code=%d stderr=%q", args, code, stderr)
		}
	}
	if host.installed[vlc] || len(runner.interactive) != 0 {
		t.Fatal("installed while locked")
	}
}

func TestApplyAppRecordsJournal(t *testing.T) {
	host := &fakeHost{installed: map[string]bool{}}
	withProcess(t, &scriptedRunner{respond: host.respond}, true)
	dir := withStateDir(t)
	if code := run([]string{"apply", "--app", "localsend", "--yes"}); code != 0 {
		t.Fatalf("code=%d", code)
	}
	if e := readJournal(t, dir)["localsend"]; e.Result != state.ResultInstalled || !e.Changed || e.UpdatedAt == "" {
		t.Fatalf("journal entry %+v", e)
	}
}

func TestApplyFlagsAreExclusive(t *testing.T) {
	for _, args := range [][]string{
		{"apply"},
		{"apply", "--app", "vlc", "--profile", "essential"},
		{"verify", "--profile", "essential"},
	} {
		if got := run(args); got != 2 {
			t.Fatalf("%v returned %d", args, got)
		}
	}
}
