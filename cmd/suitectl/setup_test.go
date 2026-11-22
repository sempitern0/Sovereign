package main

import (
	"strings"
	"testing"
	"context"
	"runtime"

	"github.com/sempitern0/Sovereign/internal/install"
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

func TestSetupMissingLinuxProviderIsPreparedOnlyWithConsent(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux only test")
	}
	beforeProviders := detectProviders
	beforePrepare := prepareFlatpak
	withStateDir(t)
	defer func(){ detectProviders = beforeProviders; prepareFlatpak = beforePrepare }()
	providerPresent := false
	detectProviders = func() map[string]bool { return map[string]bool{"flatpak": providerPresent} }
	prepareCalls := 0
	prepareFlatpak = func(ctx context.Context, runner install.Runner, interactive, consent bool)(bool,error){
		prepareCalls++
		if !consent { t.Fatal("missing consent") }
		providerPresent=true
		return true,nil
	}
	host := &fakeHost{installed: allInstalled(t)}
	runner := &scriptedRunner{respond: host.respond}
	withProcess(t, runner, false)
	if code:=run([]string{"setup"});code!=3 || prepareCalls!=0 {
		t.Fatalf("without consent: code=%d preparer=%d",code,prepareCalls)
	}
	if code:=run([]string{"setup","--yes"});code!=0 || prepareCalls!=1 {
		t.Fatalf("with consent: code=%d preparer=%d",code,prepareCalls)
	}
	if code:=run([]string{"setup","--yes"});code!=0 || prepareCalls!=1 {
		t.Fatalf("second call: code=%d preparer=%d",code,prepareCalls)
	}
}
