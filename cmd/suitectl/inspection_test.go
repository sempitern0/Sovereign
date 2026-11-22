package main

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/sempitern0/Sovereign/internal/catalog"
	"github.com/sempitern0/Sovereign/internal/install"
	"github.com/sempitern0/Sovereign/internal/planner"
)

type inspectRunner struct {
	out   string
	err   error
	calls int
}

func (f *inspectRunner) Run(_ context.Context, _ string, _ ...string) (string, error) {
	f.calls++
	return f.out, f.err
}
func testPlanCatalog() catalog.Catalog {
	return catalog.Catalog{Apps: map[string]catalog.Application{
		"localsend": {ID: "localsend", Upstream: "https://localsend.org/", Platforms: map[string]catalog.Platform{
			"linux": {Status: "verified", Provider: "flatpak", PackageID: "org.localsend.localsend_app", Source: "https://flathub.org/apps/org.localsend.localsend_app", VerifiedAt: "2026-11-10", Scope: "user", Provenance: "upstream"},
		}},
	}}
}
func TestPlanInspectionInstalledAndMissing(t *testing.T) {
	c := testPlanCatalog()
	for _, tc := range []struct{ output, want string }{
		{"org.localsend.localsend_app\n", "installed"},
		{"", "not_installed"},
	} {
		f := &inspectRunner{out: tc.output}
		p := planner.Plan{SchemaVersion: 1, Platform: "linux", ReadOnly: true, Operations: []planner.Operation{{Application: "localsend", Status: "inspection_required", Action: "inspect"}}}
		got := inspectPlan(context.Background(), c, p, install.Engine{Runner: f, Platform: "linux"})
		if got.Operations[0].Status != tc.want || got.Operations[0].Action != "none" || f.calls != 1 {
			t.Fatalf("%+v calls=%d", got, f.calls)
		}
	}
}
func TestPlanInspectionFailsClosed(t *testing.T) {
	c := testPlanCatalog()
	f := &inspectRunner{err: errors.New("provider unavailable")}
	p := planner.Plan{SchemaVersion: 1, Platform: "linux", ReadOnly: true, Operations: []planner.Operation{{Application: "localsend", Status: "inspection_required", Action: "inspect"}, {Application: "libreoffice", Status: "pending_verification", Action: "none"}}}
	got := inspectPlan(context.Background(), c, p, install.Engine{Runner: f, Platform: "linux"})
	if got.Operations[0].Status != "inspection_failed" || got.Operations[0].Action != "none" || got.Operations[1].Status != "pending_verification" || f.calls != 1 {
		t.Fatalf("%+v", got)
	}
}
func TestPlanUnreviewedAppKeepsH1(t *testing.T) {
	c := testPlanCatalog()
	f := &inspectRunner{}
	p := planner.Plan{Platform: "linux", ReadOnly: true, Operations: []planner.Operation{{Application: "libreoffice", Status: "inspection_required", Action: "inspect"}}}
	got := inspectPlan(context.Background(), c, p, install.Engine{Runner: f, Platform: "linux"})
	if got.Operations[0].Status != "inspection_required" || f.calls != 0 {
		t.Fatalf("%+v", got)
	}
}

// concurrencyRunner answers `winget list` slowly and records how many queries
// overlapped.
type concurrencyRunner struct {
	mu            sync.Mutex
	active, peak  int
	calls         int
	release       chan struct{}
	installedOnly string
}

func (r *concurrencyRunner) Run(_ context.Context, _ string, args ...string) (string, error) {
	r.mu.Lock()
	r.calls++
	r.active++
	r.peak = max(r.peak, r.active)
	r.mu.Unlock()
	<-r.release
	r.mu.Lock()
	r.active--
	r.mu.Unlock()
	if args[2] == r.installedOnly {
		return "Name " + args[2] + " 1.0 winget\n", nil
	}
	return "No installed package found matching input criteria.", nil
}

func TestInspectionRunsBoundedInParallelAndKeepsPlanOrder(t *testing.T) {
	c, err := loadCatalog("")
	if err != nil {
		t.Fatal(err)
	}
	plan, err := planner.Build(c, "essential", planner.Device{Platform: "windows", Providers: map[string]bool{"winget": true}})
	if err != nil {
		t.Fatal(err)
	}
	want := 0
	for _, op := range plan.Operations {
		if op.Status == "inspection_required" {
			want++
		}
	}
	if want <= maxConcurrentInspections {
		t.Skipf("need more than %d WinGet apps in the setup, have %d", maxConcurrentInspections, want)
	}
	r := &concurrencyRunner{release: make(chan struct{}), installedOnly: "VideoLAN.VLC"}
	done := make(chan planner.Plan)
	go func() {
		done <- inspectPlan(context.Background(), c, plan, install.Engine{Runner: r, Platform: "windows", Cache: install.NewCache()})
	}()
	// Hold every query until the bound is reached, proving they overlap.
	deadline := time.Now().Add(5 * time.Second)
	for {
		r.mu.Lock()
		active := r.active
		r.mu.Unlock()
		if active == maxConcurrentInspections {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("only %d concurrent inspections, want %d", active, maxConcurrentInspections)
		}
		time.Sleep(time.Millisecond)
	}
	close(r.release)
	got := <-done
	if r.calls != want || r.peak != maxConcurrentInspections {
		t.Fatalf("calls=%d (want %d) peak=%d (want %d)", r.calls, want, r.peak, maxConcurrentInspections)
	}
	for i, op := range got.Operations {
		if op.Application != plan.Operations[i].Application {
			t.Fatalf("plan order changed at %d: %s", i, op.Application)
		}
		if op.Application == "vlc" && op.Status != "installed" {
			t.Fatalf("vlc %+v", op)
		}
	}
}
