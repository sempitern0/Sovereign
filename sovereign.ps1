# Single entry point for Sovereign in Windows PowerShell.
# Go is installed in the current user's profile only when needed.
[CmdletBinding()]
param(
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$CommandArgs = @()
)
$ErrorActionPreference = 'Stop'
$bootstrap = Join-Path $PSScriptRoot 'bootstrap/setup.ps1'
& $bootstrap -RunArguments $CommandArgs
exit $LASTEXITCODE
