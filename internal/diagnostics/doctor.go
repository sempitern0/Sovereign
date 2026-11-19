package diagnostics

import (
	"os/exec"
	"runtime"
)

// Report is read-only: detecting a package manager never installs or modifies software.
type Report struct {
	SchemaVersion      int             `json:"schema_version"`
	Platform           string          `json:"platform"`
	Architecture       string          `json:"architecture"`
	Providers          map[string]bool `json:"providers"`
	MultisuiteRequired bool            `json:"multisuite_required"`
}

// Detect reports only package managers currently supported by the local OS.
// Future provider implementations should explicitly extend this list.
func Detect() Report {
	names := []string{}
	switch runtime.GOOS {
	case "windows":
		names = []string{"winget"}
	case "linux":
		names = []string{"flatpak"}
	}
	providers := make(map[string]bool, len(names))
	for _, name := range names {
		_, err := exec.LookPath(name)
		providers[name] = err == nil
	}
	return Report{SchemaVersion: 1, Platform: runtime.GOOS, Architecture: runtime.GOARCH, Providers: providers, MultisuiteRequired: false}
}
