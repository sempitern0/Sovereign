package main

import (
	"context"
	"errors"
	"testing"

	"github.com/sempitern0/Sovereign/internal/catalog"
	"github.com/sempitern0/Sovereign/internal/install"
	"github.com/sempitern0/Sovereign/internal/planner"
)

type inspectRunner struct {
	out string
	err error
	calls int
}
func (f *inspectRunner) Run(_ context.Context, _ string, _ ...string) (string,error){
	f.calls++
	return f.out, f.err
}
func testPlanCatalog() catalog.Catalog {
	return catalog.Catalog{Apps: map[string]catalog.Application{
		"localsend": {ID:"localsend", Upstream:"https://localsend.org/", Platforms:map[string]catalog.Platform{
			"linux":{Status:"verified",Provider:"flatpak",PackageID:"org.localsend.localsend_app",Source:"https://flathub.org/apps/org.localsend.localsend_app",VerifiedAt:"2026-11-10"},
		}},
	}}
}
func TestPlanInspectionInstalledAndMissing(t *testing.T) {
	c:=testPlanCatalog()
	for _,tc:=range []struct{output,want string}{
		{"org.localsend.localsend_app\n","installed"},
		{"","not_installed"},
	}{
		f:=&inspectRunner{out:tc.output}
		p:=planner.Plan{SchemaVersion:1,Platform:"linux",ReadOnly:true,Operations:[]planner.Operation{{Application:"localsend",Status:"inspection_required",Action:"inspect"}}}
		got:=inspectPlan(context.Background(),c,p,install.Engine{Runner:f,Platform:"linux"})
		if got.Operations[0].Status!=tc.want || got.Operations[0].Action!="none" || f.calls!=1 {t.Fatalf("%+v calls=%d",got,f.calls)}
	}
}
func TestPlanInspectionFailsClosed(t *testing.T) {
	c:=testPlanCatalog()
	f:=&inspectRunner{err:errors.New("provider unavailable")}
	p:=planner.Plan{SchemaVersion:1,Platform:"linux",ReadOnly:true,Operations:[]planner.Operation{{Application:"localsend",Status:"inspection_required",Action:"inspect"},{Application:"libreoffice",Status:"pending_verification",Action:"none"}}}
	got:=inspectPlan(context.Background(),c,p,install.Engine{Runner:f,Platform:"linux"})
	if got.Operations[0].Status!="inspection_failed" || got.Operations[0].Action!="none" || got.Operations[1].Status!="pending_verification" || f.calls!=1 {t.Fatalf("%+v",got)}
}
func TestPlanUnreviewedAppKeepsH1(t *testing.T) {
	c:=testPlanCatalog()
	f:=&inspectRunner{}
	p:=planner.Plan{Platform:"linux",ReadOnly:true,Operations:[]planner.Operation{{Application:"vlc",Status:"inspection_required",Action:"inspect"}}}
	got:=inspectPlan(context.Background(),c,p,install.Engine{Runner:f,Platform:"linux"})
	if got.Operations[0].Status!="inspection_required" || f.calls!=0 {t.Fatalf("%+v",got)}
}
