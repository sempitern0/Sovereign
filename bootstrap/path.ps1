# User-scoped PATH activation for an already verified Go executable.
# Intentionally separate from installation to test idempotence and rollback.
function Test-GoPathSegment {
    param(
        [AllowNull()][string]$PathValue,
        [Parameter(Mandatory = $true)][string]$Directory
    )
    if ([string]::IsNullOrWhiteSpace($PathValue)) {
        return $false
    }
    $normalized = $Directory.TrimEnd([char[]]@('\', '/'))
    foreach ($segment in $PathValue.Split(';')) {
        if ($segment.Trim().Trim('"').TrimEnd([char[]]@('\', '/')) -ieq $normalized) {
            return $true
        }
    }
    return $false
}

function Add-GoToUserPath {
    [CmdletBinding()]
    param([Parameter(Mandatory = $true)][string]$Executable)

    if (-not (Test-Path -LiteralPath $Executable -PathType Leaf)) {
        throw "Refusing to publish a missing Go executable: $Executable"
    }

    $goDirectory = [IO.Path]::GetFullPath((Split-Path -Parent $Executable))
    $currentUserPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    if (-not (Test-GoPathSegment -PathValue $currentUserPath -Directory $goDirectory)) {
        # HKCU only: never write the machine PATH or require administrator rights.
        $newUserPath = if ([string]::IsNullOrWhiteSpace($currentUserPath)) {
            $goDirectory
        } else {
            $goDirectory + ';' + $currentUserPath.TrimStart(';')
        }
        [Environment]::SetEnvironmentVariable('Path', $newUserPath, 'User')
        $saved = [Environment]::GetEnvironmentVariable('Path', 'User')
        if (-not (Test-GoPathSegment -PathValue $saved -Directory $goDirectory)) {
            throw 'Unable to update the current-user PATH. Go was installed but not activated.'
        }
    }

    # Prepend in the calling PowerShell process, so "go" works immediately.
    # This does not change the parent shell when launched by pwsh -File.
    if ($env:Path -notlike "$goDirectory;*") {
        $segments = @($env:Path.Split(';') | Where-Object {
            $_.Trim().Trim('"').TrimEnd([char[]]@('\', '/')) -ine $goDirectory.TrimEnd([char[]]@('\', '/'))
        })
        $env:Path = (@($goDirectory) + $segments) -join ';'
    }
}
