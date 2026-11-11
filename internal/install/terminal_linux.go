//go:build linux

package install

import (
	"syscall"
	"unsafe"
)

// isTerminalFD asks the kernel for terminal attributes. Character devices that
// are not terminals, such as /dev/null, fail this ioctl.
func isTerminalFD(fd uintptr) bool {
	var attrs syscall.Termios
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, syscall.TCGETS, uintptr(unsafe.Pointer(&attrs)))
	return errno == 0
}
