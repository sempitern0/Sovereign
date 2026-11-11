package install

import (
	"bytes"
	"errors"
	"fmt"
	"slices"
	"sync"

	sovereign "github.com/sempitern0/Sovereign"
	"github.com/sempitern0/Sovereign/internal/catalog"
)

// AllowEntry is one reviewed installation target.
type AllowEntry struct {
	Application string `yaml:"application"`
	Upstream    string `yaml:"upstream"`
	Platform    string `yaml:"platform"`
	Provider    string `yaml:"provider"`
	PackageID   string `yaml:"package_id"`
	Source      string `yaml:"source"`
	Scope       string `yaml:"scope"`
	Provenance  string `yaml:"provenance"`
	// ProviderDependencies must equal the catalog's list: packages the provider
	// installs on its own because the manifest declares them.
	ProviderDependencies []string `yaml:"provider_dependencies,omitempty"`
}

// providerScopes lists the install scopes each implemented provider supports.
// Flatpak installs are always per user; system installs would need root.
var providerScopes = map[string][]string{"winget": {"user", "machine"}, "flatpak": {"user"}}

// Allowlist is the set of packages H2 may install, independent of the catalog.
type Allowlist struct {
	SchemaVersion int          `yaml:"schema_version"`
	Entries       []AllowEntry `yaml:"entries"`
}

func (a Allowlist) Lookup(appID, platform string) (AllowEntry, bool) {
	for _, entry := range a.Entries {
		if entry.Application == appID && entry.Platform == platform {
			return entry, true
		}
	}
	return AllowEntry{}, false
}

func (a Allowlist) HasApplication(appID string) bool {
	for _, entry := range a.Entries {
		if entry.Application == appID {
			return true
		}
	}
	return false
}

// ParseAllowlist validates structure; it does not consult the catalog.
func ParseAllowlist(data []byte) (Allowlist, error) {
	var allow Allowlist
	if err := catalog.DecodeStrict(bytes.NewReader(data), sovereign.AllowlistPath, &allow); err != nil {
		return Allowlist{}, err
	}
	if allow.SchemaVersion != 1 {
		return Allowlist{}, errors.New("install allowlist: unsupported schema_version")
	}
	seen := map[string]bool{}
	for i, entry := range allow.Entries {
		if entry.Application == "" || entry.Upstream == "" || entry.PackageID == "" || entry.Source == "" {
			return Allowlist{}, fmt.Errorf("install allowlist: entry %d has empty fields", i)
		}
		if providerPlatform[entry.Platform] != entry.Provider {
			return Allowlist{}, fmt.Errorf("install allowlist: entry %d: provider %q is not implemented for %q", i, entry.Provider, entry.Platform)
		}
		if !slices.Contains(providerScopes[entry.Provider], entry.Scope) {
			return Allowlist{}, fmt.Errorf("install allowlist: entry %d: scope %q is not supported by %s", i, entry.Scope, entry.Provider)
		}
		if entry.Provenance != "upstream" && entry.Provenance != "community" {
			return Allowlist{}, fmt.Errorf("install allowlist: entry %d: provenance must be upstream or community", i)
		}
		seenDeps := map[string]bool{}
		for _, dep := range entry.ProviderDependencies {
			if dep == "" || dep == entry.PackageID || seenDeps[dep] {
				return Allowlist{}, fmt.Errorf("install allowlist: entry %d: invalid or duplicate provider dependency %q", i, dep)
			}
			seenDeps[dep] = true
		}
		key := entry.Application + "/" + entry.Platform
		if seen[key] {
			return Allowlist{}, fmt.Errorf("install allowlist: duplicate entry %s", key)
		}
		seen[key] = true
	}
	return allow, nil
}

var (
	embeddedOnce  sync.Once
	embeddedAllow Allowlist
	embeddedErr   error
)

// Embedded returns the allowlist compiled into the binary. It is never read
// from --root or any other runtime location.
func Embedded() (Allowlist, error) {
	embeddedOnce.Do(func() {
		data, err := sovereign.Files.ReadFile(sovereign.AllowlistPath)
		if err != nil {
			embeddedErr = err
			return
		}
		embeddedAllow, embeddedErr = ParseAllowlist(data)
	})
	return embeddedAllow, embeddedErr
}
