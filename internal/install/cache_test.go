package install

import (
	"context"
	"errors"
	"testing"

	sovereign "github.com/sempitern0/Sovereign"
	"github.com/sempitern0/Sovereign/internal/catalog"
)

func embeddedLinuxTarget(t *testing.T, app string) Target {
	t.Helper()
	c, err := catalog.LoadFS(sovereign.Files)
	if err != nil {
		t.Fatal(err)
	}
	target, err := (Engine{}).Resolve(c, app, "linux")
	if err != nil {
		t.Fatal(err)
	}
	return target
}

func countCalls(calls []call, program, first string) int {
	n := 0
	for _, c := range calls {
		if c.program == program && len(c.args) > 0 && c.args[0] == first {
			n++
		}
	}
	return n
}

func TestCacheListsFlatpakAppsOncePerRun(t *testing.T) {
	f := &fakeRunner{output: []string{"org.videolan.VLC\n"}}
	e := Engine{Runner: f, Platform: "linux", Cache: NewCache()}
	for _, app := range []string{"vlc", "localsend", "libreoffice", "comaps"} {
		if _, err := e.Status(context.Background(), embeddedLinuxTarget(t, app)); err != nil {
			t.Fatal(err)
		}
	}
	if len(f.calls) != 1 {
		t.Fatalf("want one flatpak list for four inspections, got %+v", f.calls)
	}
}

func TestCacheDoesNotKeepFailedInspections(t *testing.T) {
	f := &fakeRunner{output: []string{"", "org.videolan.VLC\n"}, errors: []error{errors.New("boom"), nil}}
	e := Engine{Runner: f, Platform: "linux", Cache: NewCache()}
	target := embeddedLinuxTarget(t, "vlc")
	if _, err := e.Status(context.Background(), target); err == nil {
		t.Fatal("failure must surface, never become missing")
	}
	if state, err := e.Status(context.Background(), target); err != nil || state != Installed {
		t.Fatalf("state %v err %v", state, err)
	}
}

// Installing several Flatpak apps checks remotes once and re-lists only after
// each install (the mandatory verification), never before.
func TestCacheInvalidatesAfterInstall(t *testing.T) {
	f := &fakeRunner{output: []string{
		"", // list (inspection)
		"flathub\thttps://dl.flathub.org/repo/\n", // remotes
		"",                   // install vlc
		"org.videolan.VLC\n", // list (verify vlc)
		"",                   // install localsend
		"org.videolan.VLC\norg.localsend.localsend_app\n", // list (verify localsend)
	}}
	e := Engine{Runner: f, Platform: "linux", Cache: NewCache()}
	for _, app := range []string{"vlc", "localsend"} {
		result, err := e.Apply(context.Background(), embeddedLinuxTarget(t, app), true)
		if err != nil || !result.Changed {
			t.Fatalf("%s: result %+v err %v (calls %+v)", app, result, err, f.calls)
		}
	}
	if got := countCalls(f.calls, "flatpak", "list"); got != 3 {
		t.Fatalf("want 3 flatpak list calls (1 inspection + 1 verification per install), got %d: %+v", got, f.calls)
	}
	if got := countCalls(f.calls, "flatpak", "remotes"); got != 1 {
		t.Fatalf("want 1 remotes call, got %d", got)
	}
}

func TestNilCacheKeepsPreviousBehaviour(t *testing.T) {
	f := &fakeRunner{output: []string{"", ""}}
	e := Engine{Runner: f, Platform: "linux"}
	target := embeddedLinuxTarget(t, "vlc")
	for range 2 {
		if _, err := e.Status(context.Background(), target); err != nil {
			t.Fatal(err)
		}
	}
	if len(f.calls) != 2 {
		t.Fatalf("without a cache every inspection calls the provider: %+v", f.calls)
	}
}

// The plan's inspection is reused by Apply: one `winget list` before the
// install (not two) and the mandatory verification afterwards.
func TestCacheReusesWingetInspectionBeforeInstall(t *testing.T) {
	target := comapsWindowsTarget(t)
	f := &interactiveRunner{fakeRunner: fakeRunner{output: []string{
		"No installed package found matching input criteria.", // inspection
		"CoMaps CoMaps.CoMaps 2026.06.13 winget",              // verification
	}}}
	e := Engine{Runner: f, Platform: "windows", Interactive: true, Cache: NewCache()}
	if state, err := e.Status(context.Background(), target); err != nil || state != Missing {
		t.Fatalf("state %v err %v", state, err)
	}
	result, err := e.Apply(context.Background(), target, true)
	if err != nil || !result.Changed || result.State != Installed {
		t.Fatalf("result %+v err %v", result, err)
	}
	if got := countCalls(f.calls, "winget", "list"); got != 2 || len(f.interactive) != 1 {
		t.Fatalf("want 2 winget list calls and 1 install, got %d lists: %+v / %+v", got, f.calls, f.interactive)
	}
}

func TestCacheDoesNotKeepFailedWingetInspections(t *testing.T) {
	target := comapsWindowsTarget(t)
	f := &fakeRunner{output: []string{"", "CoMaps CoMaps.CoMaps 2026.06.13 winget"}, errors: []error{errors.New("boom"), nil}}
	e := Engine{Runner: f, Platform: "windows", Cache: NewCache()}
	if _, err := e.Status(context.Background(), target); err == nil {
		t.Fatal("failure must surface, never become missing")
	}
	if state, err := e.Status(context.Background(), target); err != nil || state != Installed || len(f.calls) != 2 {
		t.Fatalf("state %v err %v calls %d", state, err, len(f.calls))
	}
}
