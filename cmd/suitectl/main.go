package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sort"

	sovereign "github.com/sempitern0/Sovereign"
	"github.com/sempitern0/Sovereign/internal/catalog"
	"github.com/sempitern0/Sovereign/internal/diagnostics"
	"github.com/sempitern0/Sovereign/internal/install"
	"github.com/sempitern0/Sovereign/internal/planner"
	"github.com/sempitern0/Sovereign/internal/state"
)

const version = "0.3.0-dev"

// Process boundaries, replaceable in tests so no test run reaches a real
// package manager or depends on the terminal it was started from.
var (
	newRunner       = func() install.Runner { return install.ExecRunner{} }
	isTerminal      = install.IsTerminal
	detectProviders = func() map[string]bool { return diagnostics.Detect().Providers }
	newStore        = func() (state.Store, error) {
		dir, err := state.DefaultDir()
		return state.Store{Dir: dir}, err
	}
)

func main() { os.Exit(run(os.Args[1:])) }

func run(args []string) int {
	if len(args) == 0 {
		usage()
		return 0
	}
	switch args[0] {
	case "version":
		if len(args) != 1 {
			return usageError("version takes no arguments")
		}
		fmt.Println("suitectl", version)
		return 0
	case "doctor":
		fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		machine := fs.Bool("json", false, "Print machine-readable diagnostics")
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}
		if fs.NArg() != 0 {
			return usageError("doctor: unexpected arguments")
		}
		report := diagnostics.Detect()
		if *machine {
			return printJSON(report)
		}
		fmt.Printf("Sovereign %s | %s/%s\n", version, report.Platform, report.Architecture)
		names := make([]string, 0, len(report.Providers))
		for name := range report.Providers {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			status := "not found"
			if report.Providers[name] {
				status = "available"
			}
			fmt.Printf("  %-10s %s\n", name, status)
		}
		fmt.Println("Multisuite: optional (no connection attempted)")
		return 0
	case "catalog", "profiles":
		if len(args) < 2 || args[1] != "list" {
			return usageError(args[0] + ": expected list subcommand")
		}
		fs := flag.NewFlagSet(args[0]+" list", flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		root := fs.String("root", "", "Development only: read catalog/profiles from this checkout instead of the embedded copy")
		platform := fs.String("platform", "", "Filter applications by platform")
		machine := fs.Bool("json", false, "JSON output")
		if err := fs.Parse(args[2:]); err != nil {
			return 2
		}
		if fs.NArg() != 0 {
			return usageError("unexpected positional arguments")
		}
		if *platform != "" && *platform != "windows" && *platform != "linux" && *platform != "android" {
			return usageError("unsupported platform")
		}
		c, err := loadCatalog(*root)
		if err != nil {
			return loadError(err)
		}
		if args[0] == "catalog" {
			type entry struct {
				ID     string `json:"id"`
				Name   string `json:"name"`
				Status string `json:"status,omitempty"`
			}
			list := []entry{}
			for _, id := range c.AppIDs() {
				app := c.Apps[id]
				status := ""
				if *platform != "" {
					p, exists := app.Platforms[*platform]
					if !exists {
						continue
					}
					status = p.Status
				}
				list = append(list, entry{ID: id, Name: app.Name, Status: status})
			}
			if *machine {
				return printJSON(list)
			}
			for _, e := range list {
				fmt.Printf("%s\t%s\t%s\n", e.ID, e.Name, e.Status)
			}
		} else {
			if *platform != "" {
				return usageError("--platform applies only to catalog list")
			}
			if *machine {
				profiles := []catalog.Profile{}
				for _, id := range c.ProfileIDs() {
					profiles = append(profiles, c.Profiles[id])
				}
				return printJSON(profiles)
			}
			for _, id := range c.ProfileIDs() {
				fmt.Printf("%s\t%s\n", id, c.Profiles[id].Name)
			}
		}
		return 0
	case "plan":
		fs := flag.NewFlagSet("plan", flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		root := fs.String("root", "", "Development only: read catalog/profiles from this checkout instead of the embedded copy")
		profile := fs.String("profile", "", "Required profile ID")
		platform := fs.String("platform", runtime.GOOS, "Target platform (windows, linux, android)")
		machine := fs.Bool("json", false, "Versioned JSON plan")
		noInspect := fs.Bool("no-inspect", false, "Skip local package inspection and preserve deterministic H1 output")
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}
		if fs.NArg() != 0 || *profile == "" {
			return usageError("plan requires --profile and no positional arguments")
		}
		c, err := loadCatalog(*root)
		if err != nil {
			return loadError(err)
		}
		available := map[string]bool{}
		if *platform == runtime.GOOS {
			available = detectProviders()
		}
		plan, err := planner.Build(c, *profile, planner.Device{Platform: *platform, Providers: available})
		if err != nil {
			return loadError(err)
		}
		if *platform == runtime.GOOS {
			plan = planner.LocalizeDependencies(plan, runtime.GOARCH)
		}
		if !*noInspect && *platform == runtime.GOOS {
			engine := install.Engine{Runner: newRunner(), Platform: runtime.GOOS}
			plan = inspectPlan(context.Background(), c, plan, engine)
		}
		if *machine {
			return printJSON(plan)
		}
		printPlan(plan, "Read-only plan")
		return 0
	case "apply", "verify":
		fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		root := fs.String("root", "", "Development only: read catalog/profiles from this checkout instead of the embedded copy")
		app := fs.String("app", "", "Single allowlisted application ID")
		profile := fs.String("profile", "", "apply only: install every required, missing application of this profile")
		yes := fs.Bool("yes", false, "Explicitly authorize installation")
		machine := fs.Bool("json", false, "Machine-readable verification result")
		nonInteractive := fs.Bool("non-interactive", false, "Never attach the package manager to the terminal; WinGet prompts are disabled")
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}
		if fs.NArg() != 0 {
			return usageError(args[0] + " takes no positional arguments")
		}
		if args[0] == "verify" && (*app == "" || *profile != "") {
			return usageError("verify requires --app")
		}
		if args[0] == "apply" && (*app == "") == (*profile == "") {
			return usageError("apply requires exactly one of --app or --profile")
		}
		if args[0] == "verify" && *yes {
			return usageError("--yes applies only to apply")
		}
		if args[0] == "apply" && *machine {
			return usageError("--json applies only to verify")
		}
		if args[0] == "verify" && *nonInteractive {
			return usageError("--non-interactive applies only to apply")
		}
		c, err := loadCatalog(*root)
		if err != nil {
			return loadError(err)
		}
		engine := install.Engine{Runner: newRunner(), Platform: runtime.GOOS, Interactive: !*nonInteractive && isTerminal()}
		if *profile != "" {
			return applyProfile(c, *profile, *yes, engine)
		}
		target, err := engine.Resolve(c, *app, runtime.GOOS)
		if err != nil {
			return loadError(err)
		}
		if args[0] == "verify" {
			state, err := engine.Status(context.Background(), target)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Verification failed:", err)
				printProviderHint(err, target.Provider)
				return 1
			}
			result := install.Result{SchemaVersion: 1, Application: target.AppID, Platform: runtime.GOOS, Provider: target.Provider, State: state, Detail: "Local package-manager inspection"}
			if *machine {
				if rc := printJSON(result); rc != 0 {
					return rc
				}
			} else {
				fmt.Printf("%s: %s\n", target.AppID, state)
			}
			if state != install.Installed {
				return 1
			}
			return 0
		}
		var release func()
		if *yes {
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
		result, err := engine.Apply(context.Background(), target, *yes)
		switch {
		case err == nil && result.Changed:
			recordJournal([]state.Entry{journalEntry(target, state.ResultInstalled, true, "")})
		case err == nil:
			recordJournal([]state.Entry{journalEntry(target, state.ResultAlreadyInstalled, false, "")})
		case !errors.Is(err, install.ErrConsentRequired):
			recordJournal([]state.Entry{journalEntry(target, state.ResultFailed, result.Changed, install.ErrorClass(err))})
		}
		if errors.Is(err, install.ErrConsentRequired) {
			fmt.Fprintf(os.Stderr, "Refusing to install %s via %s (%s). Review the plan and pass --yes to authorize this installation.\n", target.AppID, target.Provider, target.PackageID)
			if target.NeedsAdminApproval() {
				fmt.Fprintln(os.Stderr, "Note: this package installs for all users; Windows will ask for administrator approval.")
			}
			if target.Provenance == "community" {
				fmt.Fprintln(os.Stderr, "Note: community-built package, not endorsed by the upstream project.")
			}
			if len(target.ProviderDependencies) > 0 {
				deps := planner.DependenciesForArch(target.ProviderDependencies, runtime.GOARCH)
				fmt.Fprintln(os.Stderr, "Note: "+planner.DependencyNotice(target.Provider, deps)+".")
			}
			return 3
		}
		if errors.Is(err, install.ErrFlathubRemoteMissing) {
			fmt.Fprintln(os.Stderr, "Sovereign does not add package repositories. Add the official Flathub user remote yourself, then retry:")
			fmt.Fprintln(os.Stderr, "  "+install.FlathubRemoteCommand)
			return 1
		}
		if errors.Is(err, install.ErrAdminApprovalNeedsTerminal) {
			fmt.Fprintln(os.Stderr, "Not installed:", err)
			return 1
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "Installation/verification failed:", err)
			printProviderHint(err, target.Provider)
			return 1
		}
		fmt.Printf("%s: %s (%s)\n", result.Application, result.State, result.Detail)
		return 0
	default:
		return usageError("unknown command: " + args[0])
	}
}

// loadCatalog uses the catalog compiled into the binary unless a development
// checkout is given explicitly with --root.
// printPlan prints plan entries with their notices and next steps.
func printPlan(plan planner.Plan, title string) {
	fmt.Printf("%s: profile=%s platform=%s\n", title, plan.Profile, plan.Platform)
	for _, op := range plan.Operations {
		fmt.Printf("  %-16s %-22s %s\n", op.Application, op.Status, op.Reason)
		for _, notice := range op.Notices() {
			fmt.Printf("  %-16s %-22s !  %s\n", "", "", notice)
		}
		if op.NextStep != "" {
			fmt.Printf("  %-16s %-22s -> %s\n", "", "", op.NextStep)
		}
	}
}

func loadCatalog(root string) (catalog.Catalog, error) {
	if root == "" {
		return catalog.LoadFS(sovereign.Files)
	}
	return catalog.Load(root)
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

func usageError(message string) int {
	fmt.Fprintln(os.Stderr, message)
	usage()
	return 2
}

func loadError(err error) int {
	fmt.Fprintln(os.Stderr, "Catalog/plan error:", err)
	return 1
}

func usage() {
	fmt.Println("Sovereign — local-first environment orchestrator")
	fmt.Println("Usage: suitectl doctor [--json] | version")
	fmt.Println("       suitectl catalog list [--platform windows|linux|android] [--json]")
	fmt.Println("       suitectl profiles list [--json]")
	fmt.Println("       suitectl plan --profile essential [--platform linux] [--json] [--no-inspect]")
	fmt.Println("       suitectl verify --app localsend [--json]")
	fmt.Println("       suitectl apply --app <id> --yes [--non-interactive] (consent required)")
	fmt.Println("       suitectl apply --profile essential --yes [--non-interactive] (one consent for the shown plan)")
	fmt.Println("The catalog is embedded; --root <checkout> reads a development copy instead.")
}
