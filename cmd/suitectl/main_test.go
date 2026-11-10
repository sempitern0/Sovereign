package main

import "testing"

func TestUnsupportedCommandsDoNotExecute(t *testing.T) {
	for _, command := range []string{"apply", "plan", "verify"} {
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
