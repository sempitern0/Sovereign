package install

import (
	"os"
	"testing"
)

// A null device is a character device but never a terminal.
func TestNullDeviceIsNotATerminal(t *testing.T) {
	null, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer null.Close()
	if isTerminalFD(null.Fd()) {
		t.Fatalf("%s reported as a terminal", os.DevNull)
	}
	file, err := os.CreateTemp(t.TempDir(), "out")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if isTerminalFD(file.Fd()) {
		t.Fatal("regular file reported as a terminal")
	}
}
