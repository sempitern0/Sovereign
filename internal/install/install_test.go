package install

import (
	"context"
	"errors"
	"runtime"
	"strings"
	"testing"

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
	result, err := (Engine{Runner: f, Platform: runtime.GOOS}).Apply(context.Background(), target, true)
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
	if err == nil || !strings.Contains(err.Error(), "official") || len(f.calls) != 2 {
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
	if _, err := e.Status(context.Background(), Target{AppID: "localsend"}); err == nil || len(f.calls) != 0 {
		t.Fatal("cross platform accepted")
	}
}
