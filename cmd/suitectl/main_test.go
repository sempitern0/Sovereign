package main

import (
	"path/filepath"
	"testing"
)

func TestDisabledCommands(t *testing.T) {
	for _, command := range []string{"apply", "verify"} {
		if got := run([]string{command}); got != 3 {
			t.Fatalf("%s: got %d, want 3", command, got)
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
