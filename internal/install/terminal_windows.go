//go:build windows

package install

import "syscall"

// isTerminalFD succeeds only for console handles; NUL and redirected files fail.
func isTerminalFD(fd uintptr) bool {
	var mode uint32
	return syscall.GetConsoleMode(syscall.Handle(fd), &mode) == nil
}
