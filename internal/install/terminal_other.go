//go:build !linux && !windows

package install

// isTerminalFD is conservative on platforms without a supported installer:
// treat the session as unattended.
func isTerminalFD(uintptr) bool { return false }
