package catalog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, dir := range []string{"catalog/apps", "profiles"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0700); err != nil {
			t.Fatal(err)
		}
	}
	write := func(path, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, path), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("catalog/apps/alpha.yaml", "schema_version: 1\nid: alpha\nname: Alpha\ncategory: media\nlicense: MIT\nupstream: https://example.org/\nplatforms:\n  linux:\n    status: pending\n    provider: flatpak\n")
	write("profiles/essential.yaml", "schema_version: 1\nid: essential\nname: Essential\napplications:\n  - alpha\n")
	return root
}

func TestLoadFixture(t *testing.T) {
	c, err := Load(fixture(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Apps) != 1 || len(c.Profiles) != 1 {
		t.Fatalf("wrong catalog %+v", c)
	}
}

func TestRejectMalformedInputs(t *testing.T) {
	for _, test := range []struct{ name, path, find, replace, message string }{
		{"unknown key", "catalog/apps/alpha.yaml", "license: MIT", "license: MIT\nsecret: 123", "field secret not found"},
		{"wrong ID", "catalog/apps/alpha.yaml", "id: alpha", "id: evil", "invalid application"},
		{"unverified package", "catalog/apps/alpha.yaml", "provider: flatpak", "provider: flatpak\n    package_id: org.example.Alpha", "unverified"},
		{"missing app", "profiles/essential.yaml", "  - alpha", "  - missing", "missing application"},
		{"duplicate key", "catalog/apps/alpha.yaml", "name: Alpha", "name: Alpha\nname: Duplicate", "already defined"},
		{"extra doc", "catalog/apps/alpha.yaml", "name: Alpha", "name: Alpha", "multiple YAML documents"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := fixture(t)
			p := filepath.Join(root, test.path)
			body, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			after := strings.Replace(string(body), test.find, test.replace, 1)
			if test.name == "extra doc" {
				after += "\n---\nfoo: bar\n"
			}
			if err := os.WriteFile(p, []byte(after), 0600); err != nil {
				t.Fatal(err)
			}
			_, err = Load(root)
			if err == nil || !strings.Contains(err.Error(), test.message) {
				t.Fatalf("got %v, want %s", err, test.message)
			}
		})
	}
}

func TestRejectCyclesAndAlternatives(t *testing.T) {
	root := fixture(t)
	p := filepath.Join(root, "catalog/apps/alpha.yaml")
	data, _ := os.ReadFile(p)
	if err := os.WriteFile(p, append(data, []byte("depends_on: [alpha]\n")...), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "self-dependency") {
		t.Fatalf("got %v", err)
	}
}

func TestRejectMissingAlternative(t *testing.T) {
	root := fixture(t)
	p := filepath.Join(root, "profiles/essential.yaml")
	data, _ := os.ReadFile(p)
	if err := os.WriteFile(p, append(data, []byte("alternatives:\n  - id: reading\n    candidates: [alpha, missing]\n    defaults:\n      linux: missing\n")...), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "missing application") {
		t.Fatalf("got %v", err)
	}
}
