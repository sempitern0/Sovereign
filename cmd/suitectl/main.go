package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"runtime"
	"sort"

	"github.com/sempitern0/Sovereign/internal/catalog"
	"github.com/sempitern0/Sovereign/internal/diagnostics"
	"github.com/sempitern0/Sovereign/internal/install"
	"github.com/sempitern0/Sovereign/internal/planner"
)

const version = "0.3.0-dev"

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
		root := fs.String("root", ".", "Repository root")
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
		c, err := catalog.Load(*root)
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
		root := fs.String("root", ".", "Repository root")
		profile := fs.String("profile", "", "Required profile ID")
		platform := fs.String("platform", runtime.GOOS, "Target platform (windows, linux, android)")
		machine := fs.Bool("json", false, "Versioned JSON plan")
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}
		if fs.NArg() != 0 || *profile == "" {
			return usageError("plan requires --profile and no positional arguments")
		}
		c, err := catalog.Load(*root)
		if err != nil {
			return loadError(err)
		}
		available := map[string]bool{}
		if *platform == runtime.GOOS {
			available = diagnostics.Detect().Providers
		}
		plan, err := planner.Build(c, *profile, planner.Device{Platform: *platform, Providers: available})
		if err != nil {
			return loadError(err)
		}
		if *machine {
			return printJSON(plan)
		}
		fmt.Printf("Read-only plan: profile=%s platform=%s (no package state inspected)\n", plan.Profile, plan.Platform)
		for _, op := range plan.Operations {
			fmt.Printf("  %-16s %-22s %s\n", op.Application, op.Status, op.Reason)
		}
		return 0
	case "apply", "verify":
		fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		root := fs.String("root", ".", "Repository root")
		app := fs.String("app", "", "Single application ID (H2: localsend only)")
		yes := fs.Bool("yes", false, "Explicitly authorize installation")
		machine := fs.Bool("json", false, "Machine-readable verification result")
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}
		if fs.NArg() != 0 || *app == "" {
			return usageError(args[0] + " requires --app and no positional arguments")
		}
		if args[0] == "verify" && *yes {
			return usageError("--yes applies only to apply")
		}
		if args[0] == "apply" && *machine {
			return usageError("--json applies only to verify")
		}
		c, err := catalog.Load(*root)
		if err != nil {
			return loadError(err)
		}
		target, err := install.Resolve(c, *app, runtime.GOOS)
		if err != nil {
			return loadError(err)
		}
		engine := install.Engine{Runner: install.ExecRunner{}, Platform: runtime.GOOS}
		if args[0] == "verify" {
			state, err := engine.Status(context.Background(), target)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Verification failed:", err)
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
		if !*yes {
			fmt.Fprintf(os.Stderr, "Refusing to install %s via %s (%s). Review the plan and pass --yes to authorize this installation.\n", target.AppID, target.Provider, target.PackageID)
			return 3
		}
		fmt.Fprintf(os.Stderr, "Authorized installation: %s via %s (%s); package manager may request additional confirmation.\n", target.AppID, target.Provider, target.PackageID)
		result, err := engine.Apply(context.Background(), target, true)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Installation/verification failed:", err)
			return 1
		}
		fmt.Printf("%s: %s (%s)\n", result.Application, result.State, result.Detail)
		return 0
	default:
		return usageError("unknown command: " + args[0])
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
	fmt.Println("       suitectl catalog list [--platform windows|linux|android] [--json] [--root .]")
	fmt.Println("       suitectl profiles list [--json] [--root .]")
	fmt.Println("       suitectl plan --profile essential [--platform linux] [--json] [--root .]")
	fmt.Println("       suitectl verify --app localsend [--json] [--root .]")
	fmt.Println("       suitectl apply --app localsend --yes [--root .] (consent required)")
}
