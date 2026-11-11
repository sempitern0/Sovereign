package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"slices"
	"strings"

	"github.com/sempitern0/Sovereign/internal/catalog"
	"github.com/sempitern0/Sovereign/internal/install"
	"github.com/sempitern0/Sovereign/internal/planner"
	"github.com/sempitern0/Sovereign/internal/state"
)

// Outcomes reported per application by `apply --profile`.
const (
	outcomeInstalled = "installed"
	outcomeAlready   = "already installed"
	outcomeFailed    = "failed"
	outcomeBlocked   = "blocked"
	outcomeSkipped   = "skipped"
)

type profileOutcome struct {
	app    string
	result string
	detail string
}

func (o profileOutcome) incomplete() bool {
	return o.result == outcomeFailed || o.result == outcomeBlocked
}

// applyProfile installs every required, missing, allowlisted application of a
// profile after a single consent covering the plan it prints first. It keeps
// going after a failure, skips applications whose dependencies failed, and
// returns 1 if the profile is incomplete, 3 if consent is missing.
func applyProfile(c catalog.Catalog, profileID string, consent bool, engine install.Engine) int {
	ctx := context.Background()
	plan, err := planner.Build(c, profileID, planner.Device{Platform: runtime.GOOS, Providers: detectProviders()})
	if err != nil {
		return loadError(err)
	}
	plan = planner.LocalizeDependencies(plan, runtime.GOARCH)
	plan = inspectPlan(ctx, c, plan, engine)
	printPlan(plan, "Installation plan")

	var missing []string
	for _, op := range plan.Operations {
		if op.Required && op.Status == "not_installed" {
			missing = append(missing, op.Application)
		}
	}
	if len(missing) > 0 && !consent {
		fmt.Fprintf(os.Stderr, "Nothing was installed. %d application(s) would be installed: %s.\n", len(missing), strings.Join(missing, ", "))
		fmt.Fprintln(os.Stderr, "Review the plan and its notices above, then pass --yes to authorize all of them.")
		return 3
	}

	var release func()
	if len(missing) > 0 {
		store, err := newStore()
		if err == nil {
			release, err = store.Lock()
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "Not installed:", err)
			return 1
		}
		defer release()
	}

	failed := map[string]bool{}
	var outcomes []profileOutcome
	var journal []state.Entry
	for _, op := range plan.Operations {
		outcome := profileOutcome{app: op.Application}
		switch {
		case op.Status == "installed":
			outcome.result = outcomeAlready
			if target, err := engine.Resolve(c, op.Application, plan.Platform); err == nil {
				journal = append(journal, journalEntry(target, state.ResultAlreadyInstalled, false, ""))
			}
		case op.Status == "not_installed" && op.Required:
			if dep := failedDependency(c, op.Application, failed); dep != "" {
				outcome.result, outcome.detail = outcomeBlocked, "dependency "+dep+" was not installed"
				break
			}
			target, err := engine.Resolve(c, op.Application, plan.Platform)
			if err != nil {
				outcome.result, outcome.detail = outcomeFailed, err.Error()
				break
			}
			result, err := engine.Apply(ctx, target, true)
			switch {
			case err != nil:
				outcome.result, outcome.detail = outcomeFailed, install.ErrorClass(err)+": "+err.Error()
				journal = append(journal, journalEntry(target, state.ResultFailed, result.Changed, install.ErrorClass(err)))
			case result.Changed:
				outcome.result = outcomeInstalled
				journal = append(journal, journalEntry(target, state.ResultInstalled, true, ""))
			default:
				outcome.result = outcomeAlready
				journal = append(journal, journalEntry(target, state.ResultAlreadyInstalled, false, ""))
			}
		case op.Required && (op.Status == "provider_unavailable" || op.Status == "inspection_failed"):
			outcome.result, outcome.detail = outcomeFailed, op.Status+": "+op.Reason
		default:
			outcome.result, outcome.detail = outcomeSkipped, op.Status
		}
		if outcome.incomplete() {
			failed[op.Application] = true
		}
		outcomes = append(outcomes, outcome)
	}
	recordJournal(journal)
	return printProfileSummary(profileID, outcomes)
}

// failedDependency returns the first dependency of app that did not install.
func failedDependency(c catalog.Catalog, app string, failed map[string]bool) string {
	deps := slices.Clone(c.Apps[app].DependsOn)
	slices.Sort(deps)
	for _, dep := range deps {
		if failed[dep] {
			return dep
		}
	}
	return ""
}

func journalEntry(t install.Target, result string, changed bool, class string) state.Entry {
	return state.Entry{Application: t.AppID, Platform: runtime.GOOS, Provider: t.Provider, PackageID: t.PackageID,
		Scope: t.Scope, Result: result, Changed: changed, ErrorClass: class}
}

// recordJournal never changes the exit code: the package managers, not the
// journal, are the source of truth.
func recordJournal(entries []state.Entry) {
	if len(entries) == 0 {
		return
	}
	store, err := newStore()
	if err == nil {
		err = store.Record(entries...)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Warning: local journal not updated:", err)
	}
}

func printProfileSummary(profileID string, outcomes []profileOutcome) int {
	counts := map[string]int{}
	fmt.Printf("\nSummary for profile %s:\n", profileID)
	for _, o := range outcomes {
		counts[o.result]++
		line := fmt.Sprintf("  %-16s %s", o.app, o.result)
		if o.detail != "" {
			line += " (" + o.detail + ")"
		}
		fmt.Println(line)
	}
	fmt.Printf("Result: %d installed, %d already installed, %d failed, %d blocked, %d skipped\n",
		counts[outcomeInstalled], counts[outcomeAlready], counts[outcomeFailed], counts[outcomeBlocked], counts[outcomeSkipped])
	if counts[outcomeFailed]+counts[outcomeBlocked] > 0 {
		fmt.Fprintln(os.Stderr, "The profile is incomplete; fix the failures above and run the same command again. Installed applications are kept.")
		return 1
	}
	return 0
}
