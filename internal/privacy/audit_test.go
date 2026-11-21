package privacy

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type fakeRunner struct {
	output []byte
	err    error
	called bool
	script string
}

func (f *fakeRunner) Run(_ context.Context, script string) ([]byte, error) {
	f.called = true
	f.script = script
	return f.output, f.err
}

const sample = `{"schema_version":1,"platform":"windows","read_only":true,"notice":"raw","items":[{"id":"advertising_id","label":"ID","value":"0"},{"id":"tailored_experiences","label":"Tailored","value":"unknown"},{"id":"diagnostics_policy","label":"Policy","value":"unknown"},{"id":"diagnostics_service","label":"DiagTrack","value":"Running"},{"id":"xbox_input_service","label":"Xbox","value":"Stopped"},{"id":"xbox_gamesave_service","label":"Game save","value":"unknown"}]}`

func TestAuditReadOnlyAndReport(t *testing.T) {
	runner := &fakeRunner{output: []byte(sample)}
	report, err := Audit(context.Background(), "windows", runner)
	if err != nil || !runner.called || !report.ReadOnly || len(report.Items) != 6 {
		t.Fatalf("%+v %v", report, err)
	}
	for _, danger := range []string{"Stop-Service", "Set-ItemProperty", "Remove-Item", "sc.exe config", "Disable-ScheduledTask"} {
		if strings.Contains(runner.script, danger) {
			t.Fatalf("mutating command in audit script: %s", danger)
		}
	}
}
func TestAuditRejectsUnsupportedOSWithoutExecution(t *testing.T) {
	r := &fakeRunner{}
	if _, err := Audit(context.Background(), "linux", r); err == nil || r.called {
		t.Fatal("unsupported OS was inspected")
	}
}
func TestAuditFailsClosedOnErrorsOrPartialOutput(t *testing.T) {
	for _, runner := range []*fakeRunner{
		{err: errors.New("denied")},
		{output: []byte("not json")},
		{output: []byte(`{"schema_version":1,"platform":"windows","read_only":true,"items":[]}`)},
		{output: []byte(strings.Replace(sample, "\"advertising_id\"", "\"other_key\"", 1))},
	} {
		if _, err := Audit(context.Background(), "windows", runner); err == nil {
			t.Fatal("accepted invalid audit")
		}
	}
}
