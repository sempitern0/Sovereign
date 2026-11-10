# Sovereign developer bootstrap: install only an official SHA256-pinned Go
# toolchain in the current user's profile. No MSI, elevation or persistent PATH.
[CmdletBinding()]
param(
    [switch]$Json,
    [switch]$Offline
)

$ErrorActionPreference = 'Stop'
$GoVersion = '1.27.3'
$MinimumMinor = 23

if ([Environment]::OSVersion.Platform -ne [PlatformID]::Win32NT) {
    throw 'This PowerShell bootstrap supports Windows only; use setup.sh on Linux.'
}

function Test-CompatibleGo {
    param([string]$Executable)
    try {
        $versionOutput = & $Executable version 2>$null
        if ($LASTEXITCODE -ne 0) { return $false }
        if ($versionOutput -match '^go version go1\.(\d+)(?:\.\d+)? windows/') {
            return ([int]$Matches[1] -ge $MinimumMinor)
        }
    } catch {
        return $false
    }
    return $false
}

$GoExecutable = $null
$systemGo = Get-Command go -CommandType Application -ErrorAction SilentlyContinue
if ($null -ne $systemGo -and (Test-CompatibleGo $systemGo.Source)) {
    $GoExecutable = $systemGo.Source
}

if (-not $GoExecutable) {
    $architecture = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString()
    switch ($architecture) {
        'X64' {
            $goArch = 'amd64'
            $expectedSha = 'f26081f438319757f8a1b46d7d0b2c5ec7583d4038e485e0aa725cfde7885fff'
        }
        'Arm64' {
            $goArch = 'arm64'
            $expectedSha = 'c507872ff21f8a7cd3c1c217b059c6f7271a4a9d8e692810ef9dd3e1f601197c'
        }
        default { throw "Unsupported Windows architecture: $architecture" }
    }

    $localRoot = [Environment]::GetFolderPath('LocalApplicationData')
    if ([string]::IsNullOrWhiteSpace($localRoot)) {
        throw 'Cannot find the current user local data folder.'
    }
    $toolchains = Join-Path $localRoot 'Sovereign\toolchains'
    $target = Join-Path $toolchains "go$GoVersion"
    $GoExecutable = Join-Path $target 'bin\go.exe'

    if (Test-Path -LiteralPath $target) {
        if (-not (Test-Path -LiteralPath $GoExecutable -PathType Leaf)) {
            throw "Incomplete Go toolchain at $target. Refusing to overwrite."
        }
    } else {
        if ($Offline) {
            throw 'Go is unavailable and -Offline forbids downloading it.'
        }

        New-Item -ItemType Directory -Path $toolchains -Force | Out-Null
        $stage = Join-Path $toolchains ('.go' + $GoVersion + '.' + [guid]::NewGuid().ToString('N'))
        New-Item -ItemType Directory -Path $stage | Out-Null
        $archive = Join-Path $stage 'go.zip'
        $url = "https://go.dev/dl/go$GoVersion.windows-$goArch.zip"
        try {
            Write-Host "Downloading Go $GoVersion for Windows/$goArch from go.dev (SHA256 verified)..."
            $ProgressPreference = 'SilentlyContinue'
            Invoke-WebRequest -Uri $url -OutFile $archive -UseBasicParsing -MaximumRedirection 5 -TimeoutSec 600
            $actualSha = (Get-FileHash -LiteralPath $archive -Algorithm SHA256).Hash.ToLowerInvariant()
            if ($actualSha -ne $expectedSha) {
                throw 'Go archive SHA256 mismatch. Nothing was installed.'
            }

            Expand-Archive -LiteralPath $archive -DestinationPath $stage -Force
            $unpacked = Join-Path $stage 'go'
            if (-not (Test-Path -LiteralPath (Join-Path $unpacked 'bin\go.exe') -PathType Leaf)) {
                throw 'Verified Go archive does not contain a Go executable.'
            }
            if (Test-Path -LiteralPath $target) {
                throw 'Go destination was created concurrently; refusing to overwrite.'
            }
            Move-Item -LiteralPath $unpacked -Destination $target -ErrorAction Stop
            Write-Host "Go $GoVersion installed locally at $target"
        } finally {
            Remove-Item -LiteralPath $stage -Recurse -Force -ErrorAction SilentlyContinue
        }
    }

    $pinned = & $GoExecutable version
    if ($LASTEXITCODE -ne 0 -or $pinned -notmatch ('^go version go' + [regex]::Escape($GoVersion) + ' windows/')) {
        throw 'Cached Go toolchain does not match the pinned version.'
    }
}

$originalToolchain = [Environment]::GetEnvironmentVariable('GOTOOLCHAIN', 'Process')
$originalProxy = [Environment]::GetEnvironmentVariable('GOPROXY', 'Process')
$exitCode = 1
Push-Location (Split-Path -Parent $PSScriptRoot)
try {
    $env:GOTOOLCHAIN = 'local'
    if ($Offline) { $env:GOPROXY = 'off' }
    $goArgs = @('run', './cmd/suitectl', 'doctor')
    if ($Json) { $goArgs += '--json' }
    & $GoExecutable @goArgs
    $exitCode = $LASTEXITCODE
} finally {
    [Environment]::SetEnvironmentVariable('GOTOOLCHAIN', $originalToolchain, 'Process')
    [Environment]::SetEnvironmentVariable('GOPROXY', $originalProxy, 'Process')
    Pop-Location
}
exit $exitCode
