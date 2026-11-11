package install

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
)

// ExecRunner passes constant, separated arguments directly to OS executables.
type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, program string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, program, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return stdout.String(), fmt.Errorf("%s failed: %w (%s)", program, err, truncate(stderr.String(), 300))
	}
	return stdout.String(), nil
}

// RunInteractive attaches the package manager to the user's terminal so its
// progress is visible and its own prompts (e.g. agreements) stay under user control.
func (ExecRunner) RunInteractive(ctx context.Context, program string, args ...string) error {
	cmd := exec.CommandContext(ctx, program, args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s failed: %w", program, err)
	}
	return nil
}

// IsTerminal reports whether both standard input and output are character devices.
func IsTerminal() bool {
	for _, f := range []*os.File{os.Stdin, os.Stdout} {
		info, err := f.Stat()
		if err != nil || info.Mode()&os.ModeCharDevice == 0 {
			return false
		}
	}
	return true
}

func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max] + "..."
}
