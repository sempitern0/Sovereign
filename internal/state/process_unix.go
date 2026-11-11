//go:build !windows

package state

import (
	"errors"
	"syscall"
)

// processAlive sends signal 0, which checks existence without affecting it.
func processAlive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}
