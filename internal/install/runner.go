package install

import (
	"bytes"
	"context"
	"fmt"
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

func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max] + "..."
}
