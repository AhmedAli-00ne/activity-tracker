# Installs the tools needed to work on this repo on a fresh Windows laptop,
# then checks the agent builds and its tests pass.
#
# Run from the repo root in PowerShell:
#   powershell -ExecutionPolicy Bypass -File scripts\setup-dev.ps1
#
# Safe to re-run: anything already installed is skipped.

$ErrorActionPreference = 'Stop'

$tools = @(
    @{ Name = 'Git';        Id = 'Git.Git';            Command = 'git';  Version = @('--version') },
    @{ Name = 'Go';         Id = 'GoLang.Go';          Command = 'go';   Version = @('version') },
    @{ Name = 'Node.js LTS'; Id = 'OpenJS.NodeJS.LTS'; Command = 'node'; Version = @('--version') }
)

function Update-SessionPath {
    # winget installs update the registry PATH, not this session's PATH.
    $machine = [Environment]::GetEnvironmentVariable('Path', 'Machine')
    $user = [Environment]::GetEnvironmentVariable('Path', 'User')
    $env:Path = "$machine;$user"
}

# winget's "already installed, no upgrade available" exit code.
$WingetNoUpgrade = -1978335189

if (-not (Get-Command winget -ErrorAction SilentlyContinue)) {
    Write-Error 'winget is missing. Install "App Installer" from the Microsoft Store, then re-run this script.'
}

# Pick up tools installed since this terminal was opened.
Update-SessionPath

foreach ($tool in $tools) {
    if (Get-Command $tool.Command -ErrorAction SilentlyContinue) {
        Write-Host "[ok]      $($tool.Name) already installed"
        continue
    }
    Write-Host "[install] $($tool.Name)..."
    winget install --id $tool.Id -e --silent --accept-package-agreements --accept-source-agreements
    if ($LASTEXITCODE -ne 0 -and $LASTEXITCODE -ne $WingetNoUpgrade) {
        Write-Error "winget could not install $($tool.Name) (exit $LASTEXITCODE)."
    }
    Update-SessionPath
    if (-not (Get-Command $tool.Command -ErrorAction SilentlyContinue)) {
        Write-Error "$($tool.Name) installed but '$($tool.Command)' is not on PATH. Open a new terminal and re-run."
    }
}

Write-Host ''
foreach ($tool in $tools) {
    $out = & $tool.Command @($tool.Version)
    Write-Host ("{0,-12} {1}" -f $tool.Name, $out)
}

Write-Host ''
Write-Host 'Checking the agent builds and its tests pass...'
Push-Location (Join-Path $PSScriptRoot '..\agent')
try {
    go test ./...
    if ($LASTEXITCODE -ne 0) { Write-Error 'Agent tests failed.' }
    go build -trimpath -ldflags '-H windowsgui -s -w' -o bin\ActivityTracker.exe ./cmd/agent
    if ($LASTEXITCODE -ne 0) { Write-Error 'Agent build failed.' }
    Write-Host 'Agent built: agent\bin\ActivityTracker.exe'
}
finally {
    Pop-Location
}

Write-Host ''
Write-Host 'Setup complete. Open a new terminal so every tool is on PATH.'
