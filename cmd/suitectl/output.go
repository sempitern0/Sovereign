package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/sempitern0/Sovereign/internal/planner"
)

// Human-readable and JSON output shared by the subcommands.

// loadCatalog uses the catalog compiled into the binary unless a development
// checkout is given explicitly with --root.
// notAvailableHere reports a setup application that Sovereign cannot install on
// this platform yet (not verified or unsupported). It is skipped, not an error.
func notAvailableHere(op planner.Operation) bool {
	return op.Status == "pending_verification" || op.Status == "unsupported"
}

// printPlan prints plan entries with their notices and next steps.
func printPlan(plan planner.Plan, title string, verbose bool) {
	installed := 0
	pending := 0
	other := 0
	unavailable := 0
	for _, op := range plan.Operations {
		if op.Status == "installed" {
			installed++
		} else if notAvailableHere(op) {
			unavailable++
		} else if op.Required && (op.Status == "not_installed" || op.Status == "inspection_required") {
			pending++
		} else if op.Required {
			other++
		}
	}
	summary := fmt.Sprintf("%s %s (%s): %d installed, %d to review", title, plan.Profile, plan.Platform, installed, pending+other)
	if unavailable > 0 {
		summary += fmt.Sprintf(", %d not yet available on this platform", unavailable)
	}
	fmt.Println(summary)
	unavailableProviders := map[string]int{}
	for _, op := range plan.Operations {
		if op.Status == "provider_unavailable" {
			unavailableProviders[op.Provider]++
		}
	}
	mentionedProvider := map[string]bool{}
	for _, op := range plan.Operations {
		if !verbose && op.Status == "provider_unavailable" {
			if !mentionedProvider[op.Provider] {
				mentionedProvider[op.Provider] = true
				fmt.Printf("  %s missing (%d applications). Setup can prepare this dependency after consent.\n", op.Provider, unavailableProviders[op.Provider])
				if op.Provider == "flatpak" && plan.Platform == "linux" {
					fmt.Println("    Next: run Sovereign setup --yes to install Flatpak and configure Flathub.")
				} else if op.NextStep != "" {
					fmt.Println("    Next:", op.NextStep)
				}
			}
			continue
		}
		if !verbose && op.Status == "installed" {
			continue
		}
		// Nothing for the user to do: say it once, without the internal reason.
		if !verbose && notAvailableHere(op) {
			fmt.Printf("  %-16s not yet available on %s (skipped)\n", op.Application, plan.Platform)
			continue
		}
		fmt.Printf("  %-16s %s\n", op.Application, op.Status)
		if !verbose && op.Status != "not_installed" && op.Status != "inspection_failed" &&
			op.Status != "provider_unavailable" && op.Status != "pending_verification" && op.Status != "unsupported" {
			continue
		}
		if op.Reason != "" {
			fmt.Printf("    %s\n", op.Reason)
		}
		// Approval/UAC notices only matter when an actual installation is required.
		if op.Status == "not_installed" || (verbose && op.Status != "installed") {
			for _, notice := range op.Notices() {
				fmt.Printf("    ! %s\n", notice)
			}
		}
		if op.NextStep != "" {
			fmt.Printf("    Next: %s\n", op.NextStep)
		}
	}
}

func printProviderHint(err error, provider string) {
	if errors.Is(err, exec.ErrNotFound) {
		if hint := planner.ProviderSetupHint(provider); hint != "" {
			fmt.Fprintln(os.Stderr, "Next step:", hint)
		}
	}
}

func printJSON(value any) int {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}
