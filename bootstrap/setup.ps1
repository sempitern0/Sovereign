# H0 developer diagnostic; no downloads, installations or elevation.
[CmdletBinding()]
param([switch]$Json)
$ErrorActionPreference = 'Stop'
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Error 'Go 1.23+ required for H0. Release binaries are planned for H2.'
    exit 3
}
Push-Location (Split-Path -Parent $PSScriptRoot)
try {
    $arguments = @('run', './cmd/suitectl', 'doctor')
    if ($Json) { $arguments += '--json' }
    & go @arguments
    exit $LASTEXITCODE
} finally {
    Pop-Location
}
