package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"sort"

	"github.com/sempitern0/Sovereign/internal/catalog"
	"github.com/sempitern0/Sovereign/internal/install"
	"github.com/sempitern0/Sovereign/internal/planner"
	"github.com/sempitern0/Sovereign/internal/state"
)

// Observed states reported by `status`.
const (
	observedInstalled       = "installed"
	observedMissing         = "missing"
	observedUnknown         = "unknown"
	observedProviderMissing = "provider_missing"
	observedNotInstallable  = "not_installable"
	observedNotChecked      = "not_checked"
)

// StatusReport is the versioned JSON contract of `suitectl status --json`.
type StatusReport struct {
	SchemaVersion int           `json:"schema_version"`
	Profile       string        `json:"profile"`
	Platform      string        `json:"platform"`
	Complete      bool          `json:"complete"`
	Journal       string        `json:"journal"`
	Applications  []AppStatus   `json:"applications"`
	OtherRecorded []state.Entry `json:"other_recorded,omitempty"`
	Warnings      []string      `json:"warnings,omitempty"`
}

// AppStatus compares the desired, observed and last recorded state of one app.
type AppStatus struct {
	Application    string `json:"application"`
	Required       bool   `json:"required"`
	Observed       string `json:"observed"`
	Detail         string `json:"detail,omitempty"`
	LastResult     string `json:"last_result,omitempty"`
	LastErrorClass string `json:"last_error_class,omitempty"`
	LastUpdated    string `json:"last_updated,omitempty"`
	Note           string `json:"note,omitempty"`
	NextStep       string `json:"next_step,omitempty"`
}

// buildStatus is read-only: it inspects package managers and peeks at the
// journal, but never writes, locks or moves anything.
func buildStatus(c catalog.Catalog, profileID string, engine install.Engine) (StatusReport, error) {
	plan, err := planner.Build(c, profileID, planner.Device{Platform: runtime.GOOS, Providers: detectProviders()})
	if err != nil {
		return StatusReport{}, err
	}
	plan = planner.LocalizeDependencies(plan, runtime.GOARCH)
	plan = inspectPlan(context.Background(), c, plan, engine)

	report := StatusReport{SchemaVersion: 1, Profile: plan.Profile, Platform: plan.Platform, Applications: []AppStatus{}}
	recorded := map[string]state.Entry{}
	store, err := newStore()
	if err == nil {
		report.Journal = store.Path()
		var journal state.Journal
		journal, err = store.Peek()
		for _, e := range journal.Entries {
			if e.Platform == plan.Platform {
				recorded[e.Application] = e
			}
		}
	}
	if err != nil {
		report.Warnings = append(report.Warnings, "journal not available: "+err.Error())
	}

	report.Complete = true
	inProfile := map[string]bool{}
	for _, op := range plan.Operations {
		inProfile[op.Application] = true
		app := AppStatus{Application: op.Application, Required: op.Required}
		switch op.Status {
		case "installed":
			app.Observed = observedInstalled
		case "not_installed":
			app.Observed = observedMissing
		case "inspection_failed":
			app.Observed, app.Detail = observedUnknown, op.Reason
		case "provider_unavailable":
			app.Observed, app.Detail = observedProviderMissing, op.Reason
		case "optional":
			app.Observed = observedNotChecked
		default:
			app.Observed, app.Detail = observedNotInstallable, op.Reason
		}
		if e, ok := recorded[op.Application]; ok {
			app.LastResult, app.LastErrorClass, app.LastUpdated = e.Result, e.ErrorClass, e.UpdatedAt
		}
		app.Note, app.NextStep = statusNote(app, op, profileID)
		if op.Required && app.Observed != observedInstalled && app.Observed != observedNotInstallable {
			report.Complete = false
		}
		report.Applications = append(report.Applications, app)
	}
	for app, e := range recorded {
		if !inProfile[app] {
			report.OtherRecorded = append(report.OtherRecorded, e)
		}
	}
	sort.Slice(report.OtherRecorded, func(i, j int) bool {
		return report.OtherRecorded[i].Application < report.OtherRecorded[j].Application
	})
	return report, nil
}

// statusNote explains drift between what Sovereign recorded and what the
// package manager reports now.
func statusNote(app AppStatus, op planner.Operation, profileID string) (string, string) {
	retry := "suitectl apply --profile " + profileID + " --yes"
	recordedOK := app.LastResult == state.ResultInstalled || app.LastResult == state.ResultAlreadyInstalled
	switch {
	case app.Observed == observedMissing && recordedOK:
		return "recorded as installed on " + app.LastUpdated + " but no longer present (removed outside Sovereign?)", retry
	case app.Observed == observedMissing && app.LastResult == state.ResultFailed:
		return "last attempt failed (" + app.LastErrorClass + ")", retry
	case app.Observed == observedMissing && op.Required:
		return "not installed yet", retry
	case app.Observed == observedInstalled && app.LastResult == state.ResultFailed:
		return "installed now, although the last recorded attempt failed", ""
	case app.Observed == observedInstalled && app.LastResult == "":
		return "installed outside Sovereign or before the journal existed", ""
	}
	return "", op.NextStep
}

func printStatus(r StatusReport) {
	fmt.Printf("Status: profile=%s platform=%s\n", r.Profile, r.Platform)
	for _, a := range r.Applications {
		required := "required"
		if !a.Required {
			required = "optional"
		}
		fmt.Printf("  %-16s %-9s %-17s", a.Application, required, a.Observed)
		if a.LastResult != "" {
			last := a.LastResult
			if a.LastErrorClass != "" {
				last += "/" + a.LastErrorClass
			}
			fmt.Printf(" last: %s (%s)", last, a.LastUpdated)
		}
		fmt.Println()
		for _, line := range []string{a.Detail, a.Note} {
			if line != "" {
				fmt.Printf("  %-16s %-9s %-17s %s\n", "", "", "", line)
			}
		}
		if a.NextStep != "" {
			fmt.Printf("  %-16s %-9s %-17s -> %s\n", "", "", "", a.NextStep)
		}
	}
	for _, e := range r.OtherRecorded {
		fmt.Printf("  (recorded, not in profile) %s: %s (%s)\n", e.Application, e.Result, e.UpdatedAt)
	}
	for _, w := range r.Warnings {
		fmt.Fprintln(os.Stderr, "Warning:", w)
	}
	if r.Complete {
		fmt.Println("Profile complete: every installable required application is present.")
	} else {
		fmt.Println("Profile incomplete: see the entries above.")
	}
	if r.Journal != "" {
		fmt.Println("Journal:", r.Journal)
	}
}
