package main

import (
	"context"
	"runtime"
	"testing"

	"github.com/sempitern0/Sovereign/internal/install"
	"github.com/sempitern0/Sovereign/internal/privacy"
)

func TestOfflineCatalogAndWindowsToolNeverAutoInstall(t *testing.T) {
	c, err := loadCatalog("")
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Profiles) != 1 {
		t.Fatalf("expected one personal setup definition, got %d", len(c.Profiles))
	}
	if _, ok := c.Profiles["essential"]; !ok {
		t.Fatal("internal essential setup definition is missing")
	}
	if _, ok := c.Profiles["offline"]; ok {
		t.Fatal("unused offline profile must not be user-selectable")
	}
	for _, id := range []string{"kiwix", "comaps", "winhance"} {
		if _, ok := c.Apps[id]; !ok {
			t.Fatalf("%s missing from catalog", id)
		}
		for _, platform := range []string{"windows", "linux"} {
			_, err := (install.Engine{Platform: platform}).Resolve(c, id, platform)
			if err == nil {
				t.Fatalf("unverified package %s unexpectedly installable on %s", id, platform)
			}
		}
	}
}

func TestPrivacyCLIRejectsMutationCommands(t *testing.T) {
	for _, args := range [][]string{
		{"privacy", "apply"}, {"privacy", "restore"}, {"privacy", "audit", "--yes"},
	} {
		if code := run(args); code == 0 {
			t.Fatalf("privacy command unexpectedly authorized mutation: %v", args)
		}
	}
	if runtime.GOOS != "windows" {
		if code := run([]string{"privacy", "audit"}); code != 3 {
			t.Fatalf("non-Windows privacy should be unavailable, got %d", code)
		}
	}
}

type dummyPrivacyRunner struct{}

func (dummyPrivacyRunner) Run(context.Context, string) ([]byte, error) {
	return []byte("{}"), nil
}

func TestPrivacyInspectionMalformedResponseFailsClosed(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only command")
	}
	before := newPrivacyRunner
	newPrivacyRunner = func() privacy.Runner { return dummyPrivacyRunner{} }
	t.Cleanup(func() { newPrivacyRunner = before })
	if code := run([]string{"privacy", "audit", "--json"}); code == 0 {
		t.Fatal("invalid privacy report accepted")
	}
}
