// Package catalog loads and validates the declarative inputs used by the planner.
// It never contacts the network or executes package managers.
package catalog

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const SchemaVersion = 1
const maxManifestSize = 128 << 10

var identifier = regexp.MustCompile(`^[a-z][a-z0-9-]{1,63}$`)
var capabilityIdentifier = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
var packageIdentifier = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._+-]{1,127}$`)

type Platform struct {
	Status     string `yaml:"status" json:"status"`
	Provider   string `yaml:"provider" json:"provider"`
	PackageID  string `yaml:"package_id,omitempty" json:"package_id,omitempty"`
	VerifiedAt string `yaml:"verified_at,omitempty" json:"verified_at,omitempty"`
	Source     string `yaml:"source,omitempty" json:"source,omitempty"`
	// Scope is where the package installs: "user" or "machine" (Windows only;
	// machine scope means the OS asks for administrator approval).
	Scope string `yaml:"scope,omitempty" json:"scope,omitempty"`
	// Provenance records who produced the installed artifact: "upstream" when the
	// channel installs binaries published by the upstream project, "community"
	// when a third party builds or maintains the package from upstream sources.
	Provenance string `yaml:"provenance,omitempty" json:"provenance,omitempty"`
	Notes      string `yaml:"notes,omitempty" json:"notes,omitempty"`
}

var scopes = []string{"user", "machine"}
var provenances = []string{"upstream", "community"}

type Application struct {
	SchemaVersion int                 `yaml:"schema_version" json:"schema_version"`
	ID            string              `yaml:"id" json:"id"`
	Name          string              `yaml:"name" json:"name"`
	Category      string              `yaml:"category" json:"category"`
	License       string              `yaml:"license" json:"license"`
	Upstream      string              `yaml:"upstream" json:"upstream"`
	Description   string              `yaml:"description,omitempty" json:"description,omitempty"`
	Capabilities  []string            `yaml:"capabilities,omitempty" json:"capabilities,omitempty"`
	DependsOn     []string            `yaml:"depends_on,omitempty" json:"depends_on,omitempty"`
	Platforms     map[string]Platform `yaml:"platforms" json:"platforms"`
}

type Alternative struct {
	ID         string            `yaml:"id" json:"id"`
	Candidates []string          `yaml:"candidates" json:"candidates"`
	Defaults   map[string]string `yaml:"defaults,omitempty" json:"defaults,omitempty"`
}

type Profile struct {
	SchemaVersion int           `yaml:"schema_version" json:"schema_version"`
	ID            string        `yaml:"id" json:"id"`
	Name          string        `yaml:"name" json:"name"`
	Description   string        `yaml:"description,omitempty" json:"description,omitempty"`
	Applications  []string      `yaml:"applications" json:"applications"`
	Optional      []string      `yaml:"optional,omitempty" json:"optional,omitempty"`
	Alternatives  []Alternative `yaml:"alternatives,omitempty" json:"alternatives,omitempty"`
}

type Catalog struct {
	Apps     map[string]Application
	Profiles map[string]Profile
}

func (c Catalog) AppIDs() []string {
	keys := make([]string, 0, len(c.Apps))
	for id := range c.Apps {
		keys = append(keys, id)
	}
	sort.Strings(keys)
	return keys
}

func (c Catalog) ProfileIDs() []string {
	keys := make([]string, 0, len(c.Profiles))
	for id := range c.Profiles {
		keys = append(keys, id)
	}
	sort.Strings(keys)
	return keys
}

// Load reads manifests from a directory on disk (development and --root).
func Load(root string) (Catalog, error) {
	return LoadFS(os.DirFS(root))
}

// LoadFS reads manifests from any filesystem, including the catalog embedded in
// the suitectl binary. Paths use forward slashes as required by io/fs.
func LoadFS(fsys fs.FS) (Catalog, error) {
	c := Catalog{Apps: map[string]Application{}, Profiles: map[string]Profile{}}
	for _, kind := range []string{"catalog/apps", "profiles"} {
		entries, err := fs.ReadDir(fsys, kind)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return c, err
		}
		var names []string
		for _, entry := range entries {
			if strings.HasSuffix(entry.Name(), ".yaml") {
				names = append(names, entry.Name())
			}
		}
		if len(names) == 0 {
			return c, fmt.Errorf("%s: no YAML manifests", kind)
		}
		sort.Strings(names)
		byName := map[string]fs.DirEntry{}
		for _, entry := range entries {
			byName[entry.Name()] = entry
		}
		for _, name := range names {
			path := kind + "/" + name
			id := strings.TrimSuffix(name, ".yaml")
			if !identifier.MatchString(id) {
				return c, fmt.Errorf("%s: invalid filename", path)
			}
			if kind == "catalog/apps" {
				var a Application
				if err := decode(fsys, path, byName[name], &a); err != nil {
					return c, err
				}
				if err := validateApp(a, id); err != nil {
					return c, fmt.Errorf("%s: %w", path, err)
				}
				c.Apps[id] = a
			} else {
				var p Profile
				if err := decode(fsys, path, byName[name], &p); err != nil {
					return c, err
				}
				if err := validateProfile(p, id); err != nil {
					return c, fmt.Errorf("%s: %w", path, err)
				}
				c.Profiles[id] = p
			}
		}
	}
	if err := c.validateReferences(); err != nil {
		return c, err
	}
	return c, nil
}

// decode rejects symlinks, non-regular and oversized files before parsing.
// fs.DirEntry.Type reports the entry itself, without following symlinks.
func decode(fsys fs.FS, path string, entry fs.DirEntry, target any) error {
	info, err := entry.Info()
	if err != nil {
		return err
	}
	if !entry.Type().IsRegular() || info.Size() > maxManifestSize {
		return fmt.Errorf("%s: only regular YAML files of at most %d bytes allowed", path, maxManifestSize)
	}
	f, err := fsys.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return DecodeStrict(io.LimitReader(f, maxManifestSize+1), path, target)
}

// DecodeStrict decodes exactly one YAML document, rejecting unknown fields and
// duplicate keys.
func DecodeStrict(r io.Reader, path string, target any) error {
	decoder := yaml.NewDecoder(r)
	decoder.KnownFields(true)
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return fmt.Errorf("%s: multiple YAML documents or invalid trailing content", path)
	}
	return nil
}

var categories = []string{"transfer", "media", "documents", "reading", "security", "offline", "utilities"}
var providers = map[string][]string{
	"windows": {"winget", "manual", "none"},
	"linux":   {"flatpak", "apt", "dnf", "manual", "none"},
	"android": {"fdroid", "obtainium", "manual", "none"},
}

func validateApp(a Application, filename string) error {
	if a.SchemaVersion != SchemaVersion || a.ID != filename || len(a.Name) < 2 || len(a.License) < 2 || !slices.Contains(categories, a.Category) || !httpsURL(a.Upstream) || len(a.Platforms) == 0 {
		return errors.New("invalid application identity, schema, category, upstream or required fields")
	}
	for _, values := range []struct {
		ids     []string
		pattern *regexp.Regexp
	}{{a.DependsOn, identifier}, {a.Capabilities, capabilityIdentifier}} {
		seen := map[string]bool{}
		for _, id := range values.ids {
			if !values.pattern.MatchString(id) || seen[id] {
				return fmt.Errorf("invalid or duplicate capability/dependency %q", id)
			}
			seen[id] = true
		}
	}
	for platform, entry := range a.Platforms {
		allowed, ok := providers[platform]
		if !ok || !slices.Contains(allowed, entry.Provider) {
			return fmt.Errorf("%s: unsupported platform/provider %s", a.ID, platform)
		}
		if !slices.Contains([]string{"verified", "pending", "unsupported"}, entry.Status) {
			return fmt.Errorf("%s: invalid status", a.ID)
		}
		if entry.Status == "unsupported" && entry.Provider != "none" {
			return fmt.Errorf("%s: unsupported requires provider none", a.ID)
		}
		if entry.Status != "verified" && (entry.PackageID != "" || entry.Scope != "" || entry.Provenance != "") {
			return fmt.Errorf("%s: unverified entries cannot advertise a package ID, scope or provenance", a.ID)
		}
		if entry.Status == "verified" {
			if entry.Provider == "none" || !packageIdentifier.MatchString(entry.PackageID) || !httpsURL(entry.Source) || entry.VerifiedAt == "" {
				return fmt.Errorf("%s: verified platform requires package ID, trusted HTTPS source and date", a.ID)
			}
			if !slices.Contains(scopes, entry.Scope) || !slices.Contains(provenances, entry.Provenance) {
				return fmt.Errorf("%s: verified platform requires scope (user|machine) and provenance (upstream|community)", a.ID)
			}
			if entry.Scope == "machine" && platform != "windows" {
				return fmt.Errorf("%s: machine scope is only supported on windows", a.ID)
			}
			date, err := time.Parse("2006-01-02", entry.VerifiedAt)
			if err != nil || date.After(time.Now().UTC().Add(24*time.Hour)) {
				return fmt.Errorf("%s: invalid verification date", a.ID)
			}
		}
	}
	return nil
}

func httpsURL(raw string) bool {
	value, err := url.Parse(raw)
	return err == nil && value.Scheme == "https" && value.Hostname() != "" && value.User == nil && value.Fragment == "" && !strings.ContainsAny(raw, " \t\r\n")
}

func validateProfile(p Profile, filename string) error {
	if p.SchemaVersion != SchemaVersion || p.ID != filename || len(p.Name) < 2 || p.Applications == nil {
		return errors.New("invalid profile identity, schema or required fields")
	}
	seen := map[string]bool{}
	for _, id := range append(append([]string{}, p.Applications...), p.Optional...) {
		if !identifier.MatchString(id) || seen[id] {
			return fmt.Errorf("duplicate or invalid profile application %q", id)
		}
		seen[id] = true
	}
	groups := map[string]bool{}
	for _, alt := range p.Alternatives {
		if !identifier.MatchString(alt.ID) || groups[alt.ID] || len(alt.Candidates) < 2 {
			return fmt.Errorf("invalid or duplicate alternative group %q", alt.ID)
		}
		groups[alt.ID] = true
		candidates := map[string]bool{}
		for _, id := range alt.Candidates {
			if !identifier.MatchString(id) || candidates[id] || seen[id] {
				return fmt.Errorf("invalid or overlapping alternative candidate %q", id)
			}
			candidates[id] = true
		}
		for platform, id := range alt.Defaults {
			if _, ok := providers[platform]; !ok || !candidates[id] {
				return fmt.Errorf("alternative %s: invalid default for %s", alt.ID, platform)
			}
		}
	}
	return nil
}

func (c Catalog) validateReferences() error {
	for _, id := range c.AppIDs() {
		for _, dep := range c.Apps[id].DependsOn {
			if _, ok := c.Apps[dep]; !ok {
				return fmt.Errorf("application %s: missing dependency %s", id, dep)
			}
			if dep == id {
				return fmt.Errorf("application %s: self-dependency", id)
			}
		}
	}
	visited := map[string]int{}
	var visit func(string) error
	visit = func(id string) error {
		if visited[id] == 1 {
			return fmt.Errorf("dependency cycle involving %s", id)
		}
		if visited[id] == 2 {
			return nil
		}
		visited[id] = 1
		for _, dep := range c.Apps[id].DependsOn {
			if err := visit(dep); err != nil {
				return err
			}
		}
		visited[id] = 2
		return nil
	}
	for _, id := range c.AppIDs() {
		if err := visit(id); err != nil {
			return err
		}
	}
	for _, id := range c.ProfileIDs() {
		p := c.Profiles[id]
		for _, name := range append(append([]string{}, p.Applications...), p.Optional...) {
			if _, ok := c.Apps[name]; !ok {
				return fmt.Errorf("profile %s: missing application %s", id, name)
			}
		}
		for _, alt := range p.Alternatives {
			for _, name := range alt.Candidates {
				if _, ok := c.Apps[name]; !ok {
					return fmt.Errorf("profile %s: alternative %s missing application %s", id, alt.ID, name)
				}
			}
		}
	}
	return nil
}
