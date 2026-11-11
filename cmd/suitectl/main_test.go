package main

import (
	"os"
	"path/filepath"
	"testing"
)

// catalogWithPendingApp copies the repository catalog into a temporary root
// and adds "beta", an application that is not verified nor allowlisted.
func catalogWithPendingApp(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, dir := range []string{"catalog/apps", "profiles"} {
		entries, err := os.ReadDir(filepath.Join("..", "..", dir))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(root, dir), 0o700); err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			data, err := os.ReadFile(filepath.Join("..", "..", dir, entry.Name()))
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, dir, entry.Name()), data, 0o600); err != nil {
				t.Fatal(err)
			}
		}
	}
	beta := "schema_version: 1\nid: beta\nname: Beta\ncategory: media\nlicense: MIT\nupstream: https://example.org/\nplatforms:\n  windows:\n    status: pending\n    provider: winget\n  linux:\n    status: pending\n    provider: flatpak\n"
	if err := os.WriteFile(filepath.Join(root, "catalog/apps/beta.yaml"), []byte(beta), 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestH2RejectsUnsupportedAndUnconsentedOperations(t *testing.T) {
	root := catalogWithPendingApp(t)
	// Every call must fail before reaching the package manager.
	runner := &scriptedRunner{respond: func(string, []string) (string, error) { return "", nil }}
	withProcess(t, runner, false)
	for _, args := range [][]string{
		{"apply", "--app", "beta", "--yes", "--root", root},
		{"verify", "--app", "beta", "--root", root},
	} {
		if got := run(args); got != 1 {
			t.Fatalf("%v returned %d, want 1", args, got)
		}
	}
	if len(runner.calls) != 0 {
		t.Fatalf("unapproved app reached the package manager: %v", runner.calls)
	}
}

func TestUnknownCommand(t *testing.T) {
	if got := run([]string{"unknown"}); got != 2 {
		t.Fatalf("got %d", got)
	}
}

func TestDoctorReadOnly(t *testing.T) {
	if got := run([]string{"doctor", "--json"}); got != 0 {
		t.Fatalf("got %d", got)
	}
	if got := run([]string{"doctor", "extra"}); got != 2 {
		t.Fatalf("got %d", got)
	}
}

func TestReadOnlyCLI(t *testing.T) {
	root := filepath.Join("..", "..")
	for _, args := range [][]string{
		{"catalog", "list", "--root", root, "--json"},
		{"profiles", "list", "--root", root, "--json"},
		{"plan", "--profile", "essential", "--platform", "linux", "--root", root, "--json"},
		{"plan", "--profile", "essential", "--platform", "android", "--root", root},
		{"plan", "--profile", "essential", "--platform", "linux", "--no-inspect", "--root", root, "--json"},
	} {
		if got := run(args); got != 0 {
			t.Fatalf("%v returned %d", args, got)
		}
	}
	if got := run([]string{"plan", "--profile", "absent", "--root", root}); got != 1 {
		t.Fatalf("missing profile returned %d", got)
	}
	if got := run([]string{"plan", "--profile", "essential", "--platform", "invalid", "--root", root}); got != 1 {
		t.Fatalf("invalid platform returned %d", got)
	}
}

// Release binaries run outside a checkout: without --root the embedded catalog
// must be used, whatever the working directory.
func TestEmbeddedCatalogOutsideCheckout(t *testing.T) {
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })
	for _, args := range [][]string{
		{"catalog", "list", "--json"},
		{"profiles", "list"},
		{"plan", "--profile", "essential", "--platform", "linux", "--json"},
	} {
		if got := run(args); got != 0 {
			t.Fatalf("%v returned %d outside the checkout", args, got)
		}
	}
	if got := run([]string{"plan", "--profile", "essential", "--root", "."}); got != 1 {
		t.Fatalf("explicit --root to an empty directory should fail, got %d", got)
	}
}
