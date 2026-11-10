# Tests update the temporary GitHub Actions user's PATH and restore it.
$ErrorActionPreference = 'Stop'
. (Join-Path (Split-Path -Parent $PSScriptRoot) 'bootstrap/path.ps1')

$originalUserPath = [Environment]::GetEnvironmentVariable('Path', 'User')
$originalProcessPath = $env:Path
$temporary = Join-Path ([IO.Path]::GetTempPath()) ('sovereign-path-test-' + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $temporary -Force | Out-Null
$fakeExecutable = Join-Path $temporary 'go.exe'

try {
    if (Test-GoPathSegment -PathValue 'C:\Other;C:\Go\bin' -Directory 'C:\Go\bin') {
        # Existing segment: useful preflight check.
    } else {
        throw 'Existing segments were not recognized.'
    }

    try {
        Add-GoToUserPath -Executable $fakeExecutable
        throw 'Missing executable was accepted.'
    } catch {
        if ($_.Exception.Message -notlike 'Refusing to publish*') {
            throw
        }
    }

    Set-Content -LiteralPath $fakeExecutable -Value 'test placeholder'
    Add-GoToUserPath -Executable $fakeExecutable
    Add-GoToUserPath -Executable $fakeExecutable

    $nowUserPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    if (-not (Test-GoPathSegment -PathValue $nowUserPath -Directory $temporary)) {
        throw 'Current-user PATH missing Go directory.'
    }
    if (-not $env:Path.StartsWith("$temporary;", [StringComparison]::OrdinalIgnoreCase)) {
        throw 'Current process PATH does not prioritize Go.'
    }
    $count = @($nowUserPath.Split(';') | Where-Object {
        $_.TrimEnd([char[]]@('\', '/')) -ieq $temporary.TrimEnd([char[]]@('\', '/'))
    }).Count
    if ($count -ne 1) {
        throw "PATH modification is not idempotent: $count duplicates"
    }
    Write-Output 'Go PATH activation: idempotence and user-scope checks passed.'
} finally {
    [Environment]::SetEnvironmentVariable('Path', $originalUserPath, 'User')
    $env:Path = $originalProcessPath
    Remove-Item -LiteralPath $temporary -Recurse -Force -ErrorAction SilentlyContinue
}
