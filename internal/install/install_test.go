package install

import (
	"context"
	"errors"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	sovereign "github.com/sempitern0/Sovereign"
	"github.com/sempitern0/Sovereign/internal/catalog"
)

type call struct {
	program string
	args    []string
}
type fakeRunner struct {
	output []string
	errors []error
	calls  []call
}

func (f *fakeRunner) Run(_ context.Context, program string, args ...string) (string, error) {
	f.calls = append(f.calls, call{program, append([]string(nil), args...)})
	i := len(f.calls) - 1
	if i >= len(f.output) {
		return "", errors.New("unexpected provider call")
	}
	if i < len(f.errors) {
		return f.output[i], f.errors[i]
	}
	return f.output[i], nil
}

const (
	localSendWin         = "LocalSend.LocalSend"
	localSendFlatpak     = "org.localsend.localsend_app"
	localSendWinSource   = "https://github.com/microsoft/winget-pkgs/blob/master/manifests/l/LocalSend/LocalSend/1.18.2/LocalSend.LocalSend.yaml"
	localSendLinuxSource = "https://flathub.org/apps/org.localsend.localsend_app"
)

func reviewedCatalog() catalog.Catalog {
	return catalog.Catalog{Apps: map[string]catalog.Application{
		"localsend": {ID: "localsend", Upstream: "https://localsend.org/", Platforms: map[string]catalog.Platform{
			"windows": {Status: "verified", Provider: "winget", PackageID: localSendWin, Source: localSendWinSource, VerifiedAt: "2026-11-10"},
			"linux":   {Status: "verified", Provider: "flatpak", PackageID: localSendFlatpak, Source: localSendLinuxSource, VerifiedAt: "2026-11-10"},
		}},
	}}
}

func TestResolvePinnedSourcesAndPlatforms(t *testing.T) {
	for _, platform := range []string{"linux", "windows"} {
		if _, err := Resolve(reviewedCatalog(), "localsend", platform); err != nil {
			t.Fatal(err)
		}
	}
	for _, platform := range []string{"android", "darwin"} {
		if _, err := Resolve(reviewedCatalog(), "localsend", platform); err == nil {
			t.Fatal("unreviewed platform allowed")
		}
	}
	c := reviewedCatalog()
	a := c.Apps["localsend"]
	entry := a.Platforms["linux"]
	entry.Source = "https://evil.example/mirror"
	a.Platforms["linux"] = entry
	c.Apps["localsend"] = a
	if _, err := Resolve(c, "localsend", "linux"); err == nil {
		t.Fatal("source substitution allowed")
	}
	if _, err := Resolve(reviewedCatalog(), "vlc", "linux"); err == nil {
		t.Fatal("other app allowed")
	}
}

func TestApplyFailClosedWithoutConsent(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux only test")
	}
	c := reviewedCatalog()
	target, _ := Resolve(c, "localsend", "linux")
	f := &fakeRunner{output: []string{""}}
	e := Engine{Runner: f, Platform: runtime.GOOS}
	_, err := e.Apply(context.Background(), target, false)
	if err == nil || !strings.Contains(err.Error(), "consent") {
		t.Fatalf("got %v", err)
	}
	if len(f.calls) != 1 || f.calls[0].program != "flatpak" || f.calls[0].args[0] != "list" {
		t.Fatal("unexpected mutation without consent")
	}
}

func TestApplyIdempotent(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux only test")
	}
	target, _ := Resolve(reviewedCatalog(), "localsend", "linux")
	f := &fakeRunner{output: []string{"org.localsend.localsend_app\n"}}
	result, err := (Engine{Runner: f, Platform: runtime.GOOS}).Apply(context.Background(), target, false)
	if err != nil || result.Changed || result.State != Installed || len(f.calls) != 1 {
		t.Fatalf("unexpected result %+v err=%v", result, err)
	}
}

func TestApplyVerifiesAfterInstall(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux only test")
	}
	target, _ := Resolve(reviewedCatalog(), "localsend", "linux")
	f := &fakeRunner{output: []string{"", "flathub\thttps://dl.flathub.org/repo/\n", "", "org.localsend.localsend_app\n"}}
	result, err := (Engine{Runner: f, Platform: runtime.GOOS}).Apply(context.Background(), target, true)
	if err != nil || !result.Changed || result.State != Installed {
		t.Fatalf("result %+v error %v", result, err)
	}
	if len(f.calls) != 4 || f.calls[2].program != "flatpak" || f.calls[2].args[0] != "install" {
		t.Fatal("did not install after preflight")
	}
}

func TestRejectUntrustedRemote(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux only test")
	}
	target, _ := Resolve(reviewedCatalog(), "localsend", "linux")
	f := &fakeRunner{output: []string{"", "flathub\thttps://untrusted.example/repo\n"}}
	_, err := (Engine{Runner: f, Platform: runtime.GOOS}).Apply(context.Background(), target, true)
	if !errors.Is(err, ErrFlathubRemoteMissing) || len(f.calls) != 2 {
		t.Fatalf("unexpected result %v", err)
	}
}

func TestFailedAndPartialInstall(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux only test")
	}
	target, _ := Resolve(reviewedCatalog(), "localsend", "linux")
	f := &fakeRunner{output: []string{"", "flathub https://dl.flathub.org/repo/", ""}, errors: []error{nil, nil, errors.New("network down")}}
	_, err := (Engine{Runner: f, Platform: runtime.GOOS}).Apply(context.Background(), target, true)
	if err == nil || !strings.Contains(err.Error(), "installation failed") {
		t.Fatalf("got %v", err)
	}
}

func TestCrossPlatformExecutionForbidden(t *testing.T) {
	f := &fakeRunner{}
	e := Engine{Runner: f, Platform: "android"}
	if _, err := e.Status(context.Background(), Target{AppID: "localsend", Provider: "flatpak", PackageID: localSendFlatpak}); err == nil || len(f.calls) != 0 {
		t.Fatal("cross platform accepted")
	}
}

func TestWindowsProviderSequenceAndIdempotence(t *testing.T) {
	target, err := Resolve(reviewedCatalog(), "localsend", "windows")
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeRunner{output: []string{
		"No installed package found matching input criteria.",
		"",
		"Name ID Version Source\nLocalSend LocalSend.LocalSend 1.18.2 winget\n",
	}}
	engine := Engine{Runner: f, Platform: "windows"}
	result, err := engine.Apply(context.Background(), target, true)
	if err != nil || result.State != Installed || !result.Changed {
		t.Fatalf("result %+v error %v", result, err)
	}
	if len(f.calls) != 3 || f.calls[0].program != "winget" || f.calls[1].program != "winget" || f.calls[1].args[0] != "install" {
		t.Fatalf("unexpected calls: %+v", f.calls)
	}
	// Non-interactive runs disable WinGet prompts so they fail instead of hanging.
	expected := []string{"install", "--id", localSendWin, "--exact", "--source", "winget", "--no-upgrade", "--scope", "user", "--disable-interactivity"}
	if !reflect.DeepEqual(f.calls[1].args, expected) {
		t.Fatalf("unexpected installation arguments: %v", f.calls[1].args)
	}
	already := &fakeRunner{output: []string{"LocalSend LocalSend.LocalSend 1.18.2 winget"}}
	result, err = (Engine{Runner: already, Platform: "windows"}).Apply(context.Background(), target, false)
	if err != nil || result.Changed || len(already.calls) != 1 {
		t.Fatalf("second apply should not reinstall: %+v %v", result, err)
	}
}

func TestWindowsAmbiguousStateFailsClosed(t *testing.T) {
	target, _ := Resolve(reviewedCatalog(), "localsend", "windows")
	f := &fakeRunner{output: []string{"unexpected package-manager output"}}
	state, err := (Engine{Runner: f, Platform: "windows"}).Status(context.Background(), target)
	if err == nil || state != "" {
		t.Fatalf("ambiguous state accepted: %s, %v", state, err)
	}
}

func TestNativeRunnerRejectsCrossPlatform(t *testing.T) {
	other := "windows"
	if runtime.GOOS == "windows" {
		other = "linux"
	}
	_, err := (Engine{Runner: ExecRunner{}, Platform: other}).Status(context.Background(), Target{AppID: "localsend"})
	if err == nil {
		t.Fatal("native provider invoked for a different operating system")
	}
}

type interactiveRunner struct {
	fakeRunner
	interactive []call
}

func (f *interactiveRunner) RunInteractive(_ context.Context, program string, args ...string) error {
	f.interactive = append(f.interactive, call{program, append([]string(nil), args...)})
	return nil
}

func TestInteractiveInstallStreamsWithoutDisablingPrompts(t *testing.T) {
	target, _ := Resolve(reviewedCatalog(), "localsend", "windows")
	f := &interactiveRunner{fakeRunner: fakeRunner{output: []string{
		"No installed package found matching input criteria.",
		"LocalSend LocalSend.LocalSend 1.18.2 winget",
	}}}
	result, err := (Engine{Runner: f, Platform: "windows", Interactive: true}).Apply(context.Background(), target, true)
	if err != nil || !result.Changed {
		t.Fatalf("result %+v error %v", result, err)
	}
	want := []string{"install", "--id", localSendWin, "--exact", "--source", "winget", "--no-upgrade", "--scope", "user"}
	if len(f.interactive) != 1 || !reflect.DeepEqual(f.interactive[0].args, want) {
		t.Fatalf("interactive install not used as expected: %+v", f.interactive)
	}
	if len(f.calls) != 2 || f.calls[0].args[0] != "list" || f.calls[1].args[0] != "list" {
		t.Fatalf("install must not run through the capturing runner: %+v", f.calls)
	}
}

func TestEmbeddedAllowlistMatchesEmbeddedCatalog(t *testing.T) {
	allow, err := Embedded()
	if err != nil {
		t.Fatal(err)
	}
	c, err := catalog.LoadFS(sovereign.Files)
	if err != nil {
		t.Fatal(err)
	}
	if len(allow.Entries) == 0 {
		t.Fatal("empty allowlist")
	}
	for _, entry := range allow.Entries {
		if _, err := ResolveWith(c, entry.Application, entry.Platform, allow); err != nil {
			t.Fatalf("allowlist entry %s/%s does not match the catalog: %v", entry.Application, entry.Platform, err)
		}
	}
}

func TestAllowlistGatesInstallationIndependentlyOfCatalog(t *testing.T) {
	c := reviewedCatalog()
	c.Apps["vlc"] = catalog.Application{ID: "vlc", Upstream: "https://www.videolan.org/vlc/", Platforms: map[string]catalog.Platform{
		"linux": {Status: "verified", Provider: "flatpak", PackageID: "org.videolan.VLC", Source: "https://flathub.org/apps/org.videolan.VLC", VerifiedAt: "2026-11-10"},
	}}
	if _, err := Resolve(c, "vlc", "linux"); err == nil {
		t.Fatal("verified catalog entry installed without an allowlist entry")
	}
	allow := Allowlist{SchemaVersion: 1, Entries: []AllowEntry{{Application: "vlc", Upstream: "https://www.videolan.org/vlc/", Platform: "linux", Provider: "flatpak", PackageID: "org.videolan.VLC", Source: "https://flathub.org/apps/org.videolan.VLC"}}}
	target, err := ResolveWith(c, "vlc", "linux", allow)
	if err != nil {
		t.Fatalf("reviewed data-only entry rejected: %v", err)
	}
	// The generic adapter handles a newly allowlisted package without new code.
	f := &fakeRunner{output: []string{"org.videolan.VLC\n"}}
	state, err := (Engine{Runner: f, Platform: "linux", Allow: &allow}).Status(context.Background(), target)
	if err != nil || state != Installed {
		t.Fatalf("state %s err %v", state, err)
	}
	// The embedded allowlist still refuses it.
	if _, err := (Engine{Runner: &fakeRunner{}, Platform: "linux"}).Status(context.Background(), target); err == nil {
		t.Fatal("engine accepted a target outside the embedded allowlist")
	}
}

func TestParseAllowlistRejectsInvalidEntries(t *testing.T) {
	for name, body := range map[string]string{
		"unknown field":     "schema_version: 1\nentries: []\nextra: 1\n",
		"bad version":       "schema_version: 2\nentries: []\n",
		"provider mismatch": "schema_version: 1\nentries:\n  - {application: a, upstream: https://a/, platform: linux, provider: winget, package_id: a.b, source: https://a/}\n",
		"empty field":       "schema_version: 1\nentries:\n  - {application: a, upstream: https://a/, platform: linux, provider: flatpak, package_id: '', source: https://a/}\n",
		"duplicate":         "schema_version: 1\nentries:\n  - {application: a, upstream: https://a/, platform: linux, provider: flatpak, package_id: a.b, source: https://a/}\n  - {application: a, upstream: https://a/, platform: linux, provider: flatpak, package_id: a.b, source: https://a/}\n",
	} {
		if _, err := ParseAllowlist([]byte(body)); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
}

func TestInstallTimeoutIsApplied(t *testing.T) {
	target, _ := Resolve(reviewedCatalog(), "localsend", "windows")
	f := &deadlineRunner{}
	_, _ = (Engine{Runner: f, Platform: "windows", InstallTimeout: time.Minute}).Apply(context.Background(), target, true)
	if len(f.deadlines) != 2 || !f.deadlines[0] || !f.deadlines[1] {
		t.Fatalf("every provider call must carry a deadline: %v", f.deadlines)
	}
}

type deadlineRunner struct{ deadlines []bool }

func (d *deadlineRunner) Run(ctx context.Context, _ string, args ...string) (string, error) {
	_, ok := ctx.Deadline()
	d.deadlines = append(d.deadlines, ok)
	if args[0] == "list" {
		return "No installed package found matching input criteria.", nil
	}
	return "", errors.New("stop")
}
