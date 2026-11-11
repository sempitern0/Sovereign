package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestH2RejectsUnsupportedAndUnconsentedOperations(t *testing.T) {
	root := filepath.Join("..", "..")
	for _, args := range [][]string{

		{"apply", "--app", "vlc", "--yes", "--root", root},
		{"verify", "--app", "vlc", "--root", root},
	} {
		if got := run(args); got == 0 {
			t.Fatalf("%v unexpectedly succeeded", args)
		}
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
