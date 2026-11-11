package planner

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/sempitern0/Sovereign/internal/catalog"
)

func sample() catalog.Catalog {
	return catalog.Catalog{
		Apps: map[string]catalog.Application{
			"alpha": {ID: "alpha", Platforms: map[string]catalog.Platform{
				"linux": {Status: "verified", Provider: "flatpak", PackageID: "com.example.Alpha", VerifiedAt: "2026-10-20"},
			}, DependsOn: []string{"beta"}},
			"beta": {ID: "beta", Platforms: map[string]catalog.Platform{
				"linux": {Status: "pending", Provider: "flatpak"},
			}},
			"gamma": {ID: "gamma", Platforms: map[string]catalog.Platform{
				"linux": {Status: "verified", Provider: "flatpak", PackageID: "com.example.Gamma", VerifiedAt: "2026-10-20"},
			}},
			"delta": {ID: "delta", Platforms: map[string]catalog.Platform{
				"linux": {Status: "unsupported", Provider: "none"},
			}},
		},
		Profiles: map[string]catalog.Profile{
			"essential": {ID: "essential", Applications: []string{"alpha"}, Optional: []string{"gamma"}, Alternatives: []catalog.Alternative{
				{ID: "reader", Candidates: []string{"delta", "gamma"}, Defaults: map[string]string{"linux": "delta"}},
			}},
		},
	}
}

func TestDeterminismAndDependencyOrder(t *testing.T) {
	c := sample()
	d := Device{Platform: "linux", Providers: map[string]bool{"flatpak": true}}
	a, err := Build(c, "essential", d)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Build(c, "essential", d)
	if err != nil {
		t.Fatal(err)
	}
	ja, _ := json.Marshal(a)
	jb, _ := json.Marshal(b)
	if string(ja) != string(jb) {
		t.Fatalf("nondeterministic output")
	}
	if a.Operations[0].Application != "beta" || a.Operations[1].Application != "alpha" {
		t.Fatalf("dependency not ordered first: %+v", a.Operations)
	}
	if a.Operations[1].Status != "inspection_required" || a.Operations[0].Status != "pending_verification" {
		t.Fatalf("unexpected statuses: %+v", a.Operations)
	}
	if !a.ReadOnly {
		t.Fatal("plan must remain read-only")
	}
}

func TestBlockedUnsupportedAndOptional(t *testing.T) {
	p, err := Build(sample(), "essential", Device{Platform: "linux"})
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, op := range p.Operations {
		got[op.Application] = op.Status
	}
	want := map[string]string{"alpha": "provider_unavailable", "beta": "pending_verification", "gamma": "optional", "delta": "unsupported"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestErrors(t *testing.T) {
	for _, tc := range []struct {
		profile, platform string
		substring         string
	}{
		{"missing", "linux", "unknown profile"},
		{"essential", "solaris", "unsupported platform"},
		{"essential", "android", "no default"},
	} {
		_, err := Build(sample(), tc.profile, Device{Platform: tc.platform})
		if err == nil || !strings.Contains(err.Error(), tc.substring) {
			t.Fatalf("%+v got %v", tc, err)
		}
	}
	c := sample()
	delete(c.Apps, "beta")
	if _, err := Build(c, "essential", Device{Platform: "linux"}); err == nil {
		t.Fatal("missing dependency accepted")
	}
}

func TestSelectedAlternativeIsRequired(t *testing.T) {
	c := sample()
	c.Profiles["essential"] = catalog.Profile{ID: "essential", Applications: []string{}, Alternatives: []catalog.Alternative{
		{ID: "choice", Candidates: []string{"alpha", "gamma"}, Defaults: map[string]string{"linux": "gamma"}},
	}}
	p, err := Build(c, "essential", Device{Platform: "linux", Providers: map[string]bool{"flatpak": true}})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Operations) != 1 || p.Operations[0].Application != "gamma" || !p.Operations[0].Required {
		t.Fatalf("wrong choice %+v", p.Operations)
	}
}

func TestProviderUnavailableExplainsNextStep(t *testing.T) {
	c := catalog.Catalog{Apps: map[string]catalog.Application{
		"alpha": {ID: "alpha", Platforms: map[string]catalog.Platform{"linux": {Status: "verified", Provider: "flatpak", PackageID: "org.example.Alpha", VerifiedAt: "2026-11-10"}}},
	}, Profiles: map[string]catalog.Profile{"p": {ID: "p", Applications: []string{"alpha"}}}}
	plan, err := Build(c, "p", Device{Platform: "linux"})
	if err != nil {
		t.Fatal(err)
	}
	op := plan.Operations[0]
	if op.Status != "provider_unavailable" || !strings.Contains(op.Reason, "flatpak") || !strings.Contains(op.NextStep, "flathub.org/setup") {
		t.Fatalf("unhelpful blocked state: %+v", op)
	}
}

func TestNoticesWarnBeforeConsent(t *testing.T) {
	op := Operation{Scope: "machine", Provenance: "community"}
	notices := op.Notices()
	if len(notices) != 2 || !strings.Contains(notices[0], "administrator") || !strings.Contains(notices[1], "community") {
		t.Fatalf("notices %v", notices)
	}
	if got := (Operation{Scope: "user", Provenance: "upstream"}).Notices(); len(got) != 0 {
		t.Fatalf("unexpected notices %v", got)
	}
}
