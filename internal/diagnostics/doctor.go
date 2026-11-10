package diagnostics

import (
	"os"
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

func Detect() Report {
	names := []string{"winget", "flatpak", "apt-get", "dnf"}
	providers := make(map[string]bool, len(names))
	for _, name := range names {
		_, err := exec.LookPath(name)
		providers[name] = err == nil
	}
	return Report{SchemaVersion: 1, Platform: runtime.GOOS, Architecture: runtime.GOARCH, Providers: providers, MultisuiteRequired: false}
}

func IsOffline() bool { return os.Getenv("SOVEREIGN_OFFLINE") == "1" }
