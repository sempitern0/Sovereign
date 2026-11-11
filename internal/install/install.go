// Package install implements a deliberately narrow, consent-gated H2 adapter.
// It neither invokes a shell nor changes remote repositories.
//
// Which packages may be installed is decided by a reviewed allowlist embedded in
// the binary (see allowlist.go), not by the catalog alone: a catalog entry must be
// verified *and* match an allowlist entry field by field. Provider behaviour is
// generic, so enabling another reviewed package is a data change, not new code.
package install

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/sempitern0/Sovereign/internal/catalog"
)

type Runner interface {
	Run(ctx context.Context, program string, args ...string) (string, error)
}

// InteractiveRunner runs a program attached to the user's terminal so that
// progress and package-manager prompts remain visible and answerable.
type InteractiveRunner interface {
	RunInteractive(ctx context.Context, program string, args ...string) error
}

type State string

const (
	Installed State = "installed"
	Missing   State = "missing"
)

// ErrConsentRequired is returned only if an installation would change the host.
// Checking an already-installed application does not require installation consent.
var ErrConsentRequired = errors.New("installation requires explicit consent (--yes)")

// ErrFlathubRemoteMissing means the user-scoped official Flathub remote is absent.
var ErrFlathubRemoteMissing = errors.New("official user Flathub remote missing; configure it separately before applying")

// ErrAdminApprovalNeedsTerminal stops a machine-scope install that would trigger
// an administrator prompt nobody is attending.
var ErrAdminApprovalNeedsTerminal = errors.New("this package installs for all users and Windows will ask for administrator approval; run apply from an interactive terminal")

// FlathubRemoteCommand is shown to the user; Sovereign never runs it itself.
const FlathubRemoteCommand = "flatpak remote-add --user --if-not-exists flathub https://dl.flathub.org/repo/flathub.flatpakrepo"

const flathubRepo = "https://dl.flathub.org/repo"

// Official WinGet NO_APPLICATIONS_FOUND HRESULT.
const wingetNoApplicationsFound = 0x8A150014

const (
	DefaultInspectTimeout = 2 * time.Minute
	DefaultInstallTimeout = 30 * time.Minute
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
	// Allow is the reviewed allowlist used by Resolve, Status and Apply. Nil means
	// the copy embedded in the binary; production code never sets it.
	Allow *Allowlist
	// Interactive streams installation output and lets the package manager prompt
	// when the runner supports it. Otherwise prompts are disabled so nothing can
	// block invisibly.
	Interactive    bool
	InspectTimeout time.Duration
	InstallTimeout time.Duration
}

type Target struct {
	AppID      string
	Provider   string
	PackageID  string
	Scope      string
	Provenance string
	// ProviderDependencies are installed by the package manager itself.
	ProviderDependencies []string
}

// NeedsAdminApproval reports whether installing triggers an OS elevation prompt.
func (t Target) NeedsAdminApproval() bool { return t.Scope == "machine" }

// Resolve fails closed on unsupported platforms, unverified catalog entries,
// unexpected sources, or unreviewed apps. It uses the same allowlist that Status
// and Apply re-check, so planning and installation can never disagree. No user
// input becomes a command.
func (e Engine) Resolve(c catalog.Catalog, appID, platform string) (Target, error) {
	allow, err := e.allowlist()
	if err != nil {
		return Target{}, err
	}
	if !allow.HasApplication(appID) {
		return Target{}, fmt.Errorf("application %q is not approved for H2", appID)
	}
	app, ok := c.Apps[appID]
	if !ok {
		return Target{}, fmt.Errorf("application %q is not in the catalog", appID)
	}
	if _, supported := providerPlatform[platform]; !supported {
		return Target{}, fmt.Errorf("H2 installs are unsupported on %s", platform)
	}
	spec, ok := app.Platforms[platform]
	if !ok || spec.Status != "verified" || spec.VerifiedAt == "" {
		return Target{}, fmt.Errorf("application %q is not verified for %s", appID, platform)
	}
	entry, ok := allow.Lookup(appID, platform)
	if !ok {
		return Target{}, fmt.Errorf("application %q is not approved for H2 on %s", appID, platform)
	}
	if app.Upstream != entry.Upstream || spec.Provider != entry.Provider || spec.PackageID != entry.PackageID || spec.Source != entry.Source ||
		spec.Scope != entry.Scope || spec.Provenance != entry.Provenance || !slices.Equal(spec.ProviderDependencies, entry.ProviderDependencies) {
		return Target{}, fmt.Errorf("%s package identity, source, scope, provenance or dependencies differ from the reviewed allowlist", entry.Provider)
	}
	return Target{AppID: appID, Provider: entry.Provider, PackageID: entry.PackageID, Scope: entry.Scope, Provenance: entry.Provenance,
		ProviderDependencies: entry.ProviderDependencies}, nil
}

// providerPlatform pins each implemented provider to its only supported OS.
var providerPlatform = map[string]string{"windows": "winget", "linux": "flatpak"}

func (e Engine) allowlist() (Allowlist, error) {
	if e.Allow != nil {
		return *e.Allow, nil
	}
	return Embedded()
}

func (e Engine) validate(t Target) error {
	if e.Runner == nil {
		return errors.New("missing package-manager runner")
	}
	if _, isNative := e.Runner.(ExecRunner); isNative && e.Platform != runtime.GOOS {
		return fmt.Errorf("installation must run on its own OS (%s != %s)", e.Platform, runtime.GOOS)
	}
	if providerPlatform[e.Platform] == "" {
		return errors.New("unsupported local platform")
	}
	allow, err := e.allowlist()
	if err != nil {
		return err
	}
	// Defense in depth: a Target must still be a reviewed one, even if it was
	// constructed without Resolve.
	entry, ok := allow.Lookup(t.AppID, e.Platform)
	if !ok || t.Provider != entry.Provider || t.PackageID != entry.PackageID || t.Scope != entry.Scope || providerPlatform[e.Platform] != t.Provider {
		return errors.New("unreviewed installation target")
	}
	return nil
}

func withTimeout(ctx context.Context, d, fallback time.Duration) (context.Context, context.CancelFunc) {
	if d <= 0 {
		d = fallback
	}
	return context.WithTimeout(ctx, d)
}

func (e Engine) Status(ctx context.Context, t Target) (State, error) {
	if err := e.validate(t); err != nil {
		return "", err
	}
	ctx, cancel := withTimeout(ctx, e.InspectTimeout, DefaultInspectTimeout)
	defer cancel()
	switch t.Provider {
	case "flatpak":
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
	case "winget":
		out, err := e.Runner.Run(ctx, "winget", "list", "--id", t.PackageID, "--exact", "--source", "winget", "--disable-interactivity")
		if err != nil {
			// Never turn an arbitrary failure into "missing".
			var exit *exec.ExitError
			if errors.As(err, &exit) && uint32(exit.ExitCode()) == wingetNoApplicationsFound {
				return Missing, nil
			}
			return "", fmt.Errorf("cannot reliably inspect WinGet state: %w", err)
		}
		for _, line := range strings.Split(out, "\n") {
			for _, field := range strings.Fields(line) {
				if field == t.PackageID {
					return Installed, nil
				}
			}
		}
		// Legacy WinGet builds may exit 0 with this English message. The exit
		// code above is the locale-independent signal; anything else unknown
		// still fails closed.
		if strings.Contains(out, "No installed package found matching input criteria.") {
			return Missing, nil
		}
		return "", errors.New("unrecognized WinGet list output; refusing installation")
	default:
		return "", errors.New("unsupported provider")
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
		return result, ErrConsentRequired
	}
	var program string
	var args []string
	switch t.Provider {
	case "flatpak":
		if err := e.requireFlathub(ctx); err != nil {
			return result, err
		}
		program, args = "flatpak", []string{"install", "--user", "--noninteractive", "flathub", t.PackageID}
	case "winget":
		program, args = "winget", []string{"install", "--id", t.PackageID, "--exact", "--source", "winget", "--no-upgrade", "--scope", t.Scope}
	default:
		return result, errors.New("unsupported provider")
	}
	if t.NeedsAdminApproval() && !e.attended() {
		return result, ErrAdminApprovalNeedsTerminal
	}
	if err := e.install(ctx, program, args); err != nil {
		return result, fmt.Errorf("%s installation failed: %w", providerName(t.Provider), err)
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

func (e Engine) requireFlathub(ctx context.Context) error {
	ctx, cancel := withTimeout(ctx, e.InspectTimeout, DefaultInspectTimeout)
	defer cancel()
	remote, err := e.Runner.Run(ctx, "flatpak", "remotes", "--user", "--columns=name,url")
	if err != nil {
		return fmt.Errorf("cannot inspect Flatpak remotes: %w", err)
	}
	for _, line := range strings.Split(remote, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "flathub" && strings.TrimRight(fields[1], "/") == flathubRepo {
			return nil
		}
	}
	return ErrFlathubRemoteMissing
}

// install shows progress and prompts on a terminal; otherwise it captures output
// and disables WinGet prompts so an unattended run fails instead of hanging.
func (e Engine) install(ctx context.Context, program string, args []string) error {
	ctx, cancel := withTimeout(ctx, e.InstallTimeout, DefaultInstallTimeout)
	defer cancel()
	if interactive, ok := e.Runner.(InteractiveRunner); ok && e.attended() {
		return interactive.RunInteractive(ctx, program, args...)
	}
	if program == "winget" {
		args = append(append([]string(nil), args...), "--disable-interactivity")
	}
	_, err := e.Runner.Run(ctx, program, args...)
	return err
}

// attended reports whether installation output and prompts reach a person.
func (e Engine) attended() bool {
	_, ok := e.Runner.(InteractiveRunner)
	return ok && e.Interactive
}

func providerName(provider string) string {
	switch provider {
	case "winget":
		return "WinGet"
	case "flatpak":
		return "Flatpak"
	}
	return provider
}
