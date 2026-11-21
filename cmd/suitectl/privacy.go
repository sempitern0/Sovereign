package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/sempitern0/Sovereign/internal/privacy"
)

var newPrivacyRunner = func() privacy.Runner { return privacy.PowerShellRunner{} }

func runPrivacy(args []string) int {
	if len(args) == 0 || args[0] != "audit" {
		return usageError("privacy currently supports only: privacy audit [--json]")
	}
	fs := flag.NewFlagSet("privacy audit", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	machine := fs.Bool("json", false, "Read-only audit in JSON format")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		return usageError("privacy audit takes no positional arguments")
	}
	if runtime.GOOS != "windows" {
		fmt.Fprintln(os.Stderr, "Privacy audit is currently Windows-only. No system modifications were made.")
		return 3
	}
	report, err := privacy.Audit(context.Background(), runtime.GOOS, newPrivacyRunner())
	if err != nil {
		fmt.Fprintln(os.Stderr, "Privacy audit failed:", err)
		return 1
	}
	if *machine {
		return printJSON(report)
	}
	fmt.Println("Windows privacy — local read-only inspection")
	for _, item := range report.Items {
		fmt.Printf("  %-38s %s\n", item.Label, item.Value)
	}
	fmt.Println("No system changes. Values can vary by Windows edition, GPO and MDM policy.")
	fmt.Println("For manual review: https://github.com/memstechtips/Winhance (PolyForm Shield licence).")
	return 0
}
