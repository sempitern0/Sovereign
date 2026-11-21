// Package privacy provides a read-only Windows settings and service inventory.
// It does not alter telemetry, terminate processes or change system services.
package privacy

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"unicode/utf16"
)

type Item struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Value string `json:"value"`
}

type Report struct {
	SchemaVersion int    `json:"schema_version"`
	Platform      string `json:"platform"`
	ReadOnly      bool   `json:"read_only"`
	Items         []Item `json:"items"`
	Notice        string `json:"notice"`
}

type Runner interface {
	Run(ctx context.Context, command string) ([]byte, error)
}

// PowerShellRunner executes a constant, encoded PowerShell program. No user input
// is interpolated into commands or into registry/service paths.
type PowerShellRunner struct{}

func (PowerShellRunner) Run(ctx context.Context, command string) ([]byte, error) {
	runes := utf16.Encode([]rune(command))
	payload := make([]byte, 2*len(runes))
	for i, r := range runes {
		payload[2*i], payload[2*i+1] = byte(r), byte(r>>8)
	}
	encoded := base64.StdEncoding.EncodeToString(payload)
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoLogo", "-NoProfile", "-NonInteractive", "-EncodedCommand", encoded)
	var out, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("Windows privacy inspection failed: %w (%s)", err, strings.TrimSpace(stderr.String()))
	}
	return out.Bytes(), nil
}

// The values are deliberately raw/unknown; Group Policy, edition and OS
// version determine the effective policy. No services are deemed unnecessary.
const auditScript = `$ErrorActionPreference = 'Stop'
function Read-Setting([string]$key, [string]$name) {
  try {
    $v = Get-ItemPropertyValue -LiteralPath $key -Name $name -ErrorAction Stop
    return [string]$v
  } catch { return 'unknown' }
}
function Read-Service([string]$name) {
  try {
    $s = Get-Service -Name $name -ErrorAction Stop
    return [string]$s.Status
  } catch { return 'unknown' }
}
$items = @(
  @{id='advertising_id';label='Advertising ID (HKCU)';value=(Read-Setting 'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\AdvertisingInfo' 'Enabled')}
  @{id='tailored_experiences';label='Tailored experiences (HKCU)';value=(Read-Setting 'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Privacy' 'TailoredExperiencesWithDiagnosticDataEnabled')}
  @{id='diagnostics_policy';label='Diagnostics policy (HKLM)';value=(Read-Setting 'HKLM:\SOFTWARE\Policies\Microsoft\Windows\DataCollection' 'AllowTelemetry')}
  @{id='diagnostics_service';label='Diagnostics service DiagTrack';value=(Read-Service 'DiagTrack')}
  @{id='xbox_input_service';label='Xbox accessory service XboxGipSvc';value=(Read-Service 'XboxGipSvc')}
  @{id='xbox_gamesave_service';label='Xbox game-save service XblGameSave';value=(Read-Service 'XblGameSave')}
)
@{schema_version=1;platform='windows';read_only=$true;items=$items;notice='Raw local values; no settings or services were changed.'} | ConvertTo-Json -Depth 4 -Compress
`

func Audit(ctx context.Context, platform string, runner Runner) (Report, error) {
	if platform != "windows" {
		return Report{}, errors.New("privacy audit is supported only on Windows")
	}
	if runner == nil {
		return Report{}, errors.New("missing privacy inspection runner")
	}
	data, err := runner.Run(ctx, auditScript)
	if err != nil {
		return Report{}, err
	}
	var report Report
	if err := json.Unmarshal(bytes.TrimSpace(data), &report); err != nil {
		return Report{}, fmt.Errorf("invalid privacy inspection response: %w", err)
	}
	if report.SchemaVersion != 1 || report.Platform != "windows" || !report.ReadOnly || len(report.Items) != 6 {
		return Report{}, errors.New("incomplete privacy inspection response")
	}
	expected := []string{"advertising_id", "tailored_experiences", "diagnostics_policy", "diagnostics_service", "xbox_input_service", "xbox_gamesave_service"}
	for i, id := range expected {
		item := report.Items[i]
		if item.ID != id || item.Label == "" || item.Value == "" {
			return Report{}, fmt.Errorf("unexpected or missing privacy item %d", i)
		}
	}
	return report, nil
}
