package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/sempitern0/Sovereign/internal/install"
)

// scriptedRunner simulates the host package manager; tests never run real ones.
type scriptedRunner struct {
	respond     func(program string, args []string) (string, error)
	calls       [][]string
	interactive [][]string
}

func (s *scriptedRunner) Run(_ context.Context, program string, args ...string) (string, error) {
	s.calls = append(s.calls, append([]string{program}, args...))
	return s.respond(program, args)
}

func (s *scriptedRunner) RunInteractive(_ context.Context, program string, args ...string) error {
	s.interactive = append(s.interactive, append([]string{program}, args...))
	_, err := s.respond(program, args)
	return err
}

func withProcess(t *testing.T, runner install.Runner, terminal bool) {
	t.Helper()
	if runtime.GOOS != "linux" && runtime.GOOS != "windows" {
		t.Skip("H2 providers exist only on Linux and Windows")
	}
	previousRunner, previousTerminal := newRunner, isTerminal
	newRunner = func() install.Runner { return runner }
	isTerminal = func() bool { return terminal }
	t.Cleanup(func() { newRunner, isTerminal = previousRunner, previousTerminal })
}

// captureStderr runs fn and returns what it wrote to os.Stderr.
func captureStderr(t *testing.T, fn func() int) (int, string) {
	t.Helper()
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	previous := os.Stderr
	os.Stderr = write
	code := fn()
	os.Stderr = previous
	write.Close()
	out, _ := io.ReadAll(read)
	return code, string(out)
}

// missingThenInstalled reports the package absent until an install happened.
func missingThenInstalled(installed *bool) func(string, []string) (string, error) {
	return func(program string, args []string) (string, error) {
		switch {
		case program == "flatpak" && args[0] == "remotes":
			return "flathub\thttps://dl.flathub.org/repo/\n", nil
		case args[0] == "install":
			*installed = true
			return "", nil
		case *installed && program == "flatpak":
			return "org.localsend.localsend_app\n", nil
		case *installed:
			return "LocalSend LocalSend.LocalSend 1.18.2 winget\n", nil
		case program == "winget":
			return "No installed package found matching input criteria.", nil
		}
		return "", nil
	}
}

func TestApplyExplainsMissingFlathubRemote(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Flathub applies to Linux")
	}
	runner := &scriptedRunner{respond: func(string, []string) (string, error) { return "", nil }}
	withProcess(t, runner, false)
	code, stderr := captureStderr(t, func() int { return run([]string{"apply", "--app", "localsend", "--yes"}) })
	if code != 1 || !strings.Contains(stderr, install.FlathubRemoteCommand) {
		t.Fatalf("code=%d stderr=%q", code, stderr)
	}
	for _, call := range runner.calls {
		if slices.Contains(call, "install") || slices.Contains(call, "remote-add") {
			t.Fatalf("Sovereign must not change remotes or install: %v", call)
		}
	}
}

func TestMissingPackageManagerShowsSetupHint(t *testing.T) {
	runner := &scriptedRunner{respond: func(program string, _ []string) (string, error) {
		return "", fmt.Errorf("%s failed: %w", program, exec.ErrNotFound)
	}}
	withProcess(t, runner, false)
	for _, args := range [][]string{{"verify", "--app", "localsend"}, {"apply", "--app", "localsend", "--yes"}} {
		code, stderr := captureStderr(t, func() int { return run(args) })
		if code != 1 || !strings.Contains(stderr, "Next step:") {
			t.Fatalf("%v: code=%d stderr=%q", args, code, stderr)
		}
	}
}

func TestInstallModeFollowsTerminalAndFlag(t *testing.T) {
	for _, tc := range []struct {
		name        string
		terminal    bool
		args        []string
		interactive bool
	}{
		{"terminal", true, nil, true},
		{"unattended", false, nil, false},
		{"flag overrides terminal", true, []string{"--non-interactive"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			installed := false
			runner := &scriptedRunner{}
			runner.respond = missingThenInstalled(&installed)
			withProcess(t, runner, tc.terminal)
			args := append([]string{"apply", "--app", "localsend", "--yes"}, tc.args...)
			code, stderr := captureStderr(t, func() int { return run(args) })
			if code != 0 {
				t.Fatalf("code=%d stderr=%q", code, stderr)
			}
			if tc.interactive {
				if len(runner.interactive) != 1 || slices.Contains(runner.interactive[0], "--disable-interactivity") {
					t.Fatalf("expected one interactive install: %v", runner.interactive)
				}
				return
			}
			if len(runner.interactive) != 0 {
				t.Fatalf("unattended run attached to the terminal: %v", runner.interactive)
			}
			var installs [][]string
			for _, call := range runner.calls {
				if call[1] == "install" {
					installs = append(installs, call)
				}
			}
			if len(installs) != 1 {
				t.Fatalf("expected one captured install: %v", runner.calls)
			}
			if runtime.GOOS == "windows" && !slices.Contains(installs[0], "--disable-interactivity") {
				t.Fatalf("WinGet prompts must be disabled when unattended: %v", installs[0])
			}
		})
	}
}

func TestNonInteractiveFlagIsApplyOnly(t *testing.T) {
	if got := run([]string{"verify", "--app", "localsend", "--non-interactive"}); got != 2 {
		t.Fatalf("got %d", got)
	}
}

// Refusing without --yes must tell the user what consenting would imply.
func TestRefusalExplainsScopeAndProvenance(t *testing.T) {
	missing := func(program string, args []string) (string, error) {
		if program == "winget" {
			return "No installed package found matching input criteria.", nil
		}
		return "", nil
	}
	withProcess(t, &scriptedRunner{respond: missing}, true)
	code, stderr := captureStderr(t, func() int { return run([]string{"apply", "--app", "vlc"}) })
	if code != 3 {
		t.Fatalf("code=%d stderr=%q", code, stderr)
	}
	want := map[string]string{"windows": "administrator approval", "linux": "community-built"}[runtime.GOOS]
	if !strings.Contains(stderr, want) {
		t.Fatalf("missing %q in %q", want, stderr)
	}
}

func TestMachineScopeUnattendedIsRefusedByCLI(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("machine scope exists only on Windows")
	}
	runner := &scriptedRunner{respond: func(string, []string) (string, error) {
		return "No installed package found matching input criteria.", nil
	}}
	withProcess(t, runner, false)
	code, stderr := captureStderr(t, func() int { return run([]string{"apply", "--app", "vlc", "--yes"}) })
	if code != 1 || !strings.Contains(stderr, "interactive terminal") || len(runner.interactive) != 0 {
		t.Fatalf("code=%d stderr=%q", code, stderr)
	}
	for _, call := range runner.calls {
		if call[1] == "install" {
			t.Fatalf("installed without an attended terminal: %v", call)
		}
	}
}
