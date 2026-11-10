// Package install implements a deliberately narrow, consent-gated H2 adapter.
// It neither invokes a shell nor changes remote repositories.
package install

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/sempitern0/Sovereign/internal/catalog"
)

type Runner interface {
	Run(ctx context.Context, program string, args ...string) (string, error)
}

type State string

const (
	Installed State = "installed"
	Missing   State = "missing"
)

type Result struct {
	SchemaVersion int    `json:"schema_version"`
	Application   string `json:"application"`
	Platform      string `json:"platform"`
	Provider      string `json:"provider"`
	State         State  `json:"state"`
	Changed       bool   `json:"changed"`
	Detail        string `json:"detail"`
}

type Engine struct {
	Runner   Runner
	Platform string
}

type Target struct {
	AppID     string
	Provider  string
	PackageID string
}

const localSendWin = "LocalSend.LocalSend"
const localSendFlatpak = "org.localsend.localsend_app"
const localSendWinSource = "https://github.com/microsoft/winget-pkgs/blob/master/manifests/l/LocalSend/LocalSend/1.18.2/LocalSend.LocalSend.yaml"
const localSendLinuxSource = "https://flathub.org/apps/org.localsend.localsend_app"

// Resolve fails closed on unsupported platforms, unverified catalog entries,
// unexpected sources, or unreviewed apps. No user input becomes a command.
func Resolve(c catalog.Catalog, appID, platform string) (Target, error) {
	app, ok := c.Apps[appID]
	if !ok || appID != "localsend" || app.Upstream != "https://localsend.org/" {
		return Target{}, fmt.Errorf("application %q is not approved for H2", appID)
	}
	spec, ok := app.Platforms[platform]
	if !ok || spec.Status != "verified" || spec.VerifiedAt == "" {
		return Target{}, fmt.Errorf("application %q is not verified for %s", appID, platform)
	}
	target := Target{AppID: appID, Provider: spec.Provider, PackageID: spec.PackageID}
	switch platform {
	case "windows":
		if spec.Provider != "winget" || spec.PackageID != localSendWin || spec.Source != localSendWinSource {
			return Target{}, errors.New("WinGet package identity or source differs from the reviewed manifest")
		}
	case "linux":
		if spec.Provider != "flatpak" || spec.PackageID != localSendFlatpak || spec.Source != localSendLinuxSource {
			return Target{}, errors.New("Flatpak identity or source differs from the reviewed manifest")
		}
	default:
		return Target{}, fmt.Errorf("H2 installs are unsupported on %s", platform)
	}
	return target, nil
}

func (e Engine) validate() error {
	if e.Runner == nil {
		return errors.New("missing package-manager runner")
	}
	if _, isNative := e.Runner.(ExecRunner); isNative && e.Platform != runtime.GOOS {
		return fmt.Errorf("installation must run on its own OS (%s != %s)", e.Platform, runtime.GOOS)
	}
	return nil
}

func (e Engine) Status(ctx context.Context, t Target) (State, error) {
	if err := e.validate(); err != nil {
		return "", err
	}
	if t.AppID != "localsend" || (e.Platform == "windows" && (t.Provider != "winget" || t.PackageID != localSendWin)) ||
		(e.Platform == "linux" && (t.Provider != "flatpak" || t.PackageID != localSendFlatpak)) {
		return "", errors.New("unreviewed installation target")
	}
	switch e.Platform {
	case "linux":
		out, err := e.Runner.Run(ctx, "flatpak", "list", "--app", "--columns=application")
		if err != nil {
			return "", fmt.Errorf("cannot list Flatpak applications: %w", err)
		}
		for _, line := range strings.Split(out, "\n") {
			if strings.TrimSpace(line) == t.PackageID {
				return Installed, nil
			}
		}
		return Missing, nil
	case "windows":
		out, err := e.Runner.Run(ctx, "winget", "list", "--id", t.PackageID, "--exact", "--source", "winget", "--disable-interactivity")
		if err != nil {
			// Official WinGet NO_APPLICATIONS_FOUND HRESULT: 0x8A150014.
			// Never turn an arbitrary failure into "missing".
			var exit *exec.ExitError
			if errors.As(err, &exit) && uint32(exit.ExitCode()) == 0x8A150014 {
				return Missing, nil
			}
			return "", fmt.Errorf("cannot reliably inspect WinGet state: %w", err)
		}
		for _, line := range strings.Split(out, "\n") {
			fields := strings.Fields(line)
			for _, field := range fields {
				if field == t.PackageID {
					return Installed, nil
				}
			}
		}
		if strings.Contains(out, "No installed package found matching input criteria.") {
			return Missing, nil
		}
		return "", errors.New("unrecognized WinGet list output; refusing installation")
	default:
		return "", errors.New("unsupported local platform")
	}
}

func (e Engine) Apply(ctx context.Context, t Target, consent bool) (Result, error) {
	result := Result{SchemaVersion: 1, Application: t.AppID, Platform: e.Platform, Provider: t.Provider}
	before, err := e.Status(ctx, t)
	if err != nil {
		return result, err
	}
	if before == Installed {
		result.State, result.Detail = Installed, "Already installed; no changes"
		return result, nil
	}
	if !consent {
		result.State, result.Detail = Missing, "Consent required; no changes"
		return result, errors.New("installation requires explicit consent (--yes)")
	}
	if e.Platform == "linux" {
		remote, err := e.Runner.Run(ctx, "flatpak", "remotes", "--user", "--columns=name,url")
		if err != nil {
			return result, fmt.Errorf("cannot inspect Flatpak remotes: %w", err)
		}
		found := false
		for _, line := range strings.Split(remote, "\n") {
			fields := strings.Fields(line)
			if len(fields) == 2 && fields[0] == "flathub" && strings.TrimRight(fields[1], "/") == "https://dl.flathub.org/repo" {
				found = true
			}
		}
		if !found {
			return result, errors.New("official user Flathub remote missing; configure it separately before applying")
		}
		if _, err := e.Runner.Run(ctx, "flatpak", "install", "--user", "--noninteractive", "flathub", t.PackageID); err != nil {
			return result, fmt.Errorf("Flatpak installation failed: %w", err)
		}
	} else {
		if _, err := e.Runner.Run(ctx, "winget", "install", "--id", t.PackageID, "--exact", "--source", "winget", "--no-upgrade", "--scope", "user"); err != nil {
			return result, fmt.Errorf("WinGet installation failed: %w", err)
		}
	}
	result.Changed = true
	after, err := e.Status(ctx, t)
	if err != nil {
		return result, fmt.Errorf("installed but verification failed: %w", err)
	}
	if after != Installed {
		return result, errors.New("package manager returned success but package was not detected")
	}
	result.State, result.Detail = Installed, "Installed and verified"
	return result, nil
}
