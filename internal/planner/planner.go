// Package planner resolves a read-only, deterministic installation assessment.
// H1 never probes installed packages or changes the device.
package planner

import (
	"fmt"
	"sort"

	"github.com/sempitern0/Sovereign/internal/catalog"
)

const SchemaVersion = 1

type Device struct {
	Platform  string          `json:"platform"`
	Providers map[string]bool `json:"providers"`
}

type Operation struct {
	ID           string `json:"id"`
	Application  string `json:"application"`
	Required     bool   `json:"required"`
	Provider     string `json:"provider,omitempty"`
	PackageID    string `json:"package_id,omitempty"`
	Action       string `json:"action"`
	Status       string `json:"status"`
	Reason       string `json:"reason"`
	Verification string `json:"verification,omitempty"`
}

// Plan has no timestamps, host identifiers or local paths: identical inputs
// always produce byte-for-byte identical JSON.
type Plan struct {
	SchemaVersion int         `json:"schema_version"`
	Profile       string      `json:"profile"`
	Platform      string      `json:"platform"`
	ReadOnly      bool        `json:"read_only"`
	Operations    []Operation `json:"operations"`
}

func Build(c catalog.Catalog, profileID string, device Device) (Plan, error) {
	p, ok := c.Profiles[profileID]
	if !ok {
		return Plan{}, fmt.Errorf("unknown profile %q", profileID)
	}
	if device.Platform != "windows" && device.Platform != "linux" && device.Platform != "android" {
		return Plan{}, fmt.Errorf("unsupported platform %q", device.Platform)
	}
	if device.Providers == nil {
		device.Providers = map[string]bool{}
	}
	required := map[string]bool{}
	selected := map[string]bool{}
	for _, id := range p.Applications {
		selected[id] = true
		required[id] = true
	}
	// Alternatives require an explicit platform default. This avoids choosing
	// a random package or silently choosing a conflicting application.
	for _, group := range p.Alternatives {
		id, ok := group.Defaults[device.Platform]
		if !ok {
			return Plan{}, fmt.Errorf("profile %s: alternative %s has no default for %s", profileID, group.ID, device.Platform)
		}
		selected[id] = true
		required[id] = true
	}
	// Optional applications appear as advisory entries, never auto-installed.
	for _, id := range p.Optional {
		selected[id] = true
	}
	var includeDependencies func(string) error
	visiting := map[string]bool{}
	includeDependencies = func(id string) error {
		app, ok := c.Apps[id]
		if !ok {
			return fmt.Errorf("unknown application %q", id)
		}
		if visiting[id] {
			return fmt.Errorf("dependency cycle involving %s", id)
		}
		visiting[id] = true
		for _, dep := range app.DependsOn {
			required[dep] = required[dep] || required[id]
			selected[dep] = true
			if err := includeDependencies(dep); err != nil {
				return err
			}
		}
		delete(visiting, id)
		return nil
	}
	for _, id := range sortedKeys(selected) {
		if err := includeDependencies(id); err != nil {
			return Plan{}, err
		}
	}
	// A topological order makes dependencies appear before their consumers,
	// while lexical ordering breaks ties predictably.
	var order []string
	visited := map[string]bool{}
	var visit func(string)
	visit = func(id string) {
		if visited[id] {
			return
		}
		visited[id] = true
		deps := append([]string{}, c.Apps[id].DependsOn...)
		sort.Strings(deps)
		for _, dep := range deps {
			if selected[dep] {
				visit(dep)
			}
		}
		order = append(order, id)
	}
	for _, id := range sortedKeys(selected) {
		visit(id)
	}
	plan := Plan{SchemaVersion: SchemaVersion, Profile: profileID, Platform: device.Platform, ReadOnly: true, Operations: []Operation{}}
	for _, id := range order {
		app := c.Apps[id]
		op := Operation{ID: "inspect:" + id, Application: id, Required: required[id], Action: "none"}
		entry, ok := app.Platforms[device.Platform]
		switch {
		case !ok || entry.Status == "unsupported":
			op.Status, op.Reason = "unsupported", "Application not supported on target platform"
		case entry.Status != "verified":
			op.Status, op.Reason = "pending_verification", "Package source and identifier not verified"
		case !required[id]:
			op.Status, op.Reason = "optional", "Optional application; no automatic action"
		case !device.Providers[entry.Provider]:
			op.Status, op.Reason = "provider_unavailable", "Package manager unavailable"
			op.Provider, op.PackageID = entry.Provider, entry.PackageID
		default:
			op.Status, op.Reason = "inspection_required", "Package verified; installed state is not yet inspected (H2)"
			op.Action, op.Provider, op.PackageID = "inspect", entry.Provider, entry.PackageID
			op.Verification = entry.VerifiedAt
		}
		plan.Operations = append(plan.Operations, op)
	}
	return plan, nil
}

func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
