package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"

	"github.com/sempitern0/Sovereign/internal/diagnostics"
)

const version = "0.1.0-dev"

func main() { os.Exit(run(os.Args[1:])) }

func run(args []string) int {
	if len(args) == 0 {
		usage()
		return 0
	}
	switch args[0] {
	case "version":
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
			fmt.Fprintln(os.Stderr, "doctor: unexpected arguments")
			return 2
		}
		report := diagnostics.Detect()
		if *machine {
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			if err := encoder.Encode(report); err != nil {
				return 1
			}
			return 0
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
	case "apply", "plan", "verify":
		fmt.Fprintf(os.Stderr, "%s is not implemented in H0. No changes were made.\n", args[0])
		return 3
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", args[0])
		usage()
		return 2
	}
}

func usage() {
	fmt.Println("Sovereign — local-first environment orchestrator")
	fmt.Println("Usage: suitectl doctor [--json] | version")
	fmt.Println("Reserved for H1/H2: plan, verify, apply (currently disabled)")
}
