package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"runtime"
	"sort"

	"github.com/sempitern0/Sovereign/internal/catalog"
	"github.com/sempitern0/Sovereign/internal/diagnostics"
	"github.com/sempitern0/Sovereign/internal/planner"
)

const version = "0.2.0-dev"

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
		fmt.Fprintf(os.Stderr, "%s is not implemented. No changes were made.\n", args[0])
		return 3
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
	fmt.Println("Reserved for H2: verify, apply (disabled)")
}
