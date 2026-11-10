package diagnostics

import (
	"runtime"
	"testing"
)

func TestDetect(t *testing.T) {
	d := Detect()
	if d.Platform != runtime.GOOS || d.Architecture != runtime.GOARCH || d.SchemaVersion != 1 {
		t.Fatal("unexpected platform")
	}
	if d.MultisuiteRequired {
		t.Fatal("multisuite must be optional")
	}
	if len(d.Providers) != 4 {
		t.Fatal("missing provider checks")
	}
}
