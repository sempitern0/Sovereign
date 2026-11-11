//go:build windows

package state

import "syscall"

const processQueryLimitedInformation = 0x1000

// processAlive opens the process with minimal rights. The lock lives in a
// per-user directory, so its holder runs as the same user and can be opened;
// failure to open means the process is gone.
func processAlive(pid int) bool {
	h, err := syscall.OpenProcess(processQueryLimitedInformation, false, uint32(pid))
	if err != nil {
		return false
	}
	defer syscall.CloseHandle(h)
	var code uint32
	if err := syscall.GetExitCodeProcess(h, &code); err != nil {
		return true
	}
	const stillActive = 259
	return code == stillActive
}
