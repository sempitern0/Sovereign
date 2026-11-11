package main

import (
	"context"

	"github.com/sempitern0/Sovereign/internal/catalog"
	"github.com/sempitern0/Sovereign/internal/install"
	"github.com/sempitern0/Sovereign/internal/planner"
)

// inspectPlan overlays local, read-only package-manager state on the deterministic
// H1 plan. It never guesses "missing" on inspection failure and never executes install.
func inspectPlan(ctx context.Context, c catalog.Catalog, plan planner.Plan, engine install.Engine) planner.Plan {
	for i := range plan.Operations {
		op := &plan.Operations[i]
		if op.Status != "inspection_required" {
			continue
		}
		target, err := engine.Resolve(c, op.Application, plan.Platform)
		if err != nil {
			// This is an H2 capability not implemented for this package.
			// Retain the H1 status rather than failing the whole profile.
			continue
		}
		state, err := engine.Status(ctx, target)
		if err != nil {
			op.Status = "inspection_failed"
			op.Reason = "Local package inspection failed: " + err.Error()
			op.Action = "none"
			op.NextStep = "Run for details: suitectl verify --app " + op.Application
			continue
		}
		op.Action = "none"
		switch state {
		case install.Installed:
			op.Status, op.Reason = "installed", "Already installed on this device"
		case install.Missing:
			op.Status, op.Reason = "not_installed", "Package missing; explicit --yes required before installation"
			op.NextStep = "After reviewing the plan: suitectl apply --app " + op.Application + " --yes"
		default:
			op.Status, op.Reason = "inspection_failed", "Package manager returned an unknown state"
		}
	}
	return plan
}
