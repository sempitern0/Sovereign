package main

import (
	"bytes"
	"github.com/sempitern0/Sovereign/internal/catalog"
	"runtime"
	"strings"
	"testing"
)

func fixtureWizard() catalog.Catalog {
	return catalog.Catalog{Profiles: map[string]catalog.Profile{
		"essential": {ID: "essential", Name: "Esencial", Applications: []string{"localsend"}},
		"offline":   {ID: "offline", Name: "Offline"},
	}, Apps: map[string]catalog.Application{}}
}
func TestWizardExitAndInvalidInput(t *testing.T) {
	var out bytes.Buffer
	w := newWizard(strings.NewReader("abc\n99\n0\n"), &out, func(...string) int { t.Fatal("called command"); return 1 }, fixtureWizard())
	if rc := w.run(); rc != 0 || !strings.Contains(out.String(), "Elige un número") {
		t.Fatalf("%d %s", rc, out.String())
	}
}
func TestWizardPreviewAndCancellationNeverInstall(t *testing.T) {
	for _, input := range []string{"1\n1\n2\n0\n", "1\n1\n1\nNO\n0\n", "1\n2\n0\n"} {
		var out bytes.Buffer
		w := newWizard(strings.NewReader(input), &out, func(args ...string) int {
			if args[0] == "apply" {
				t.Fatalf("unapproved installation %v", args)
			}
			return 0
		}, fixtureWizard())
		if rc := w.run(); rc != 0 {
			t.Fatal(rc)
		}
	}
}
func TestWizardExplicitConfirmation(t *testing.T) {
	var out bytes.Buffer
	var calls []string
	w := newWizard(strings.NewReader("1\n1\n1\nSI\n0\n"), &out, func(args ...string) int {
		calls = append(calls, strings.Join(args, " "))
		return 0
	}, fixtureWizard())
	if rc := w.run(); rc != 0 {
		t.Fatal(rc)
	}
	if len(calls) != 2 || calls[0] != "plan --profile essential" || calls[1] != "apply --profile essential --yes" {
		t.Fatal(calls)
	}
}
func TestWizardReadOnlyMenus(t *testing.T) {
	var out bytes.Buffer
	var calls []string
	w := newWizard(strings.NewReader("2\n1\n5\n4\n0\n"), &out, func(args ...string) int {
		calls = append(calls, strings.Join(args, " "))
		return 0
	}, fixtureWizard())
	if rc := w.run(); rc != 0 {
		t.Fatal(rc)
	}
	expected := []string{"status --profile essential", "doctor", "catalog list --platform " + runtime.GOOS}
	if strings.Join(calls, ";") != strings.Join(expected, ";") {
		t.Fatalf("%v", calls)
	}
}
func TestWizardEOF(t *testing.T) {
	var out bytes.Buffer
	w := newWizard(strings.NewReader(""), &out, func(...string) int { return 0 }, fixtureWizard())
	if rc := w.run(); rc != 0 {
		t.Fatal(rc)
	}
}

func TestWizardOfflineAndPrivacyMenusStayReadOnly(t *testing.T) {
	var out bytes.Buffer
	var called []string
	w:=newWizard(strings.NewReader("7\n8\n0\n"),&out,func(args ...string)int{
		called=append(called,strings.Join(args," "))
		return 0
	},fixtureWizard())
	if rc:=w.run();rc!=0 {t.Fatal(rc)}
	if !strings.Contains(out.String(),"Kiwix") || !strings.Contains(out.String(),"CoMaps") {
		t.Fatal("offline library guidance missing")
	}
	if runtime.GOOS=="windows" {
		if len(called)!=1 || called[0]!="privacy audit" {t.Fatalf("wrong audit: %v",called)}
	} else if len(called)!=0 {
		t.Fatalf("unsupported system invoked privacy: %v",called)
	}
}
