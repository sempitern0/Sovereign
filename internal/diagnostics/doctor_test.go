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
	expected := ""
	switch runtime.GOOS {
	case "windows":
		expected = "winget"
	case "linux":
		expected = "flatpak"
	}
	if expected == "" {
		if len(d.Providers) != 0 {
			t.Fatalf("unsupported OS must have no providers: %+v", d.Providers)
		}
		return
	}
	if len(d.Providers) != 1 {
		t.Fatalf("expected only the implemented provider on %s, got %+v", runtime.GOOS, d.Providers)
	}
	if _, ok := d.Providers[expected]; !ok {
		t.Fatalf("missing expected provider %s", expected)
	}
}
