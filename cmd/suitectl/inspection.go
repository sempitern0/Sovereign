package main

import (
	"context"
	"runtime"
	"sync"

	"github.com/sempitern0/Sovereign/internal/catalog"
	"github.com/sempitern0/Sovereign/internal/install"
	"github.com/sempitern0/Sovereign/internal/planner"
)

// maxConcurrentInspections bounds parallel read-only package-manager queries.
// WinGet `list` calls are slow (seconds each) and safe to run side by side;
// Flatpak queries are deduplicated by install.Cache, so they cost one call.
const maxConcurrentInspections = 3

// newEngine returns the production engine for one command, with a fresh
// read-only provider cache so repeated inspections reuse the same output.
func newEngine(interactive bool) install.Engine {
	return install.Engine{Runner: newRunner(), Platform: runtime.GOOS, Interactive: interactive, Cache: install.NewCache()}
}

// inspectPlan overlays local, read-only package-manager state on the deterministic
// H1 plan. It never guesses "missing" on inspection failure and never executes
// install. Independent inspections run concurrently (bounded); results are
// written back in plan order, so output stays deterministic.
func inspectPlan(ctx context.Context, c catalog.Catalog, plan planner.Plan, engine install.Engine) planner.Plan {
	sem := make(chan struct{}, maxConcurrentInspections)
	var wg sync.WaitGroup
	for i := range plan.Operations {
		op := &plan.Operations[i]
		if op.Status != "inspection_required" {
			continue
		}
		target, err := engine.Resolve(c, op.Application, plan.Platform)
		if err != nil {
			// Not installable through Sovereign here: keep the H1 status rather
			// than failing the whole profile.
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			state, err := engine.Status(ctx, target)
			applyInspection(op, target, state, err)
		}()
	}
	wg.Wait()
	return plan
}

// applyInspection records one inspection result on its plan operation. Each
// goroutine writes only to its own operation.
func applyInspection(op *planner.Operation, target install.Target, state install.State, err error) {
	op.Action = "none"
	if err != nil {
		op.Status = "inspection_failed"
		op.Reason = "Local package inspection failed: " + err.Error()
		op.NextStep = "Run for details: suitectl verify --app " + op.Application
		return
	}
	switch state {
	case install.Installed:
		op.Status, op.Reason = "installed", "Already installed on this device"
	case install.Missing:
		op.Status, op.Reason = "not_installed", "Package missing; explicit --yes required before installation"
		op.NextStep = "After reviewing the plan: suitectl apply --app " + op.Application + " --yes"
		if target.NeedsAdminApproval() {
			op.NextStep += " (from an interactive terminal; Windows may ask for administrator approval)"
		}
	default:
		op.Status, op.Reason = "inspection_failed", "Package manager returned an unknown state"
	}
}
