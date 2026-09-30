<#
.SYNOPSIS
  publish.ps1 -- cut a GitHub Release of Apex Claude with prebuilt bundles.

.DESCRIPTION
  Cross-compiles the apex backbone for the release matrix and, for each
  platform, bundles the binary together with the loose artifacts
  (commands / agents / skills / output-style) into a single zip:

      apex-claude-<os>-<arch>.zip
        apex(.exe)
        commands/ax-*.md
        agents/ax-*.md
        skills/ax-*/...
        output-styles/apex.md

  The zips are uploaded to a GitHub Release via `gh`. The Windows bundle is
  what scripts/install.ps1 downloads for `irm ... | iex` installs, so a fresh
  Windows box needs no Go, make, bash, or python -- just the prebuilt zip.

  This is the maintainer "ship to prod" step. End users never run it.

.PARAMETER Version
  Release tag (e.g. v0.2.0). Defaults to "v" + the const in
  internal/version/version.go. When passed explicitly, it must match that
  const or the script dies (catches a forgotten version bump).

.PARAMETER DryRun
  Build + bundle into dist/ but do not create or upload a GitHub Release.

.EXAMPLE
  pwsh scripts/publish.ps1                 # tag from main.go, publish
  pwsh scripts/publish.ps1 -Version v0.3.0
  pwsh scripts/publish.ps1 -DryRun         # local bundles only
#>
[CmdletBinding()]
param(
  [string]$Version,
  [switch]$DryRun
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$RepoRoot = Split-Path $PSScriptRoot -Parent
$Dist     = Join-Path $RepoRoot 'dist'
$Stage    = Join-Path $Dist 'stage'

# Release matrix -- mirrors RELEASE_TARGETS in the Makefile.
$Targets = @(
  @{ os = 'darwin';  arch = 'arm64' },
  @{ os = 'darwin';  arch = 'amd64' },
  @{ os = 'linux';   arch = 'amd64' },
  @{ os = 'linux';   arch = 'arm64' },
  @{ os = 'windows'; arch = 'amd64' }
)

function Say  { param($m) Write-Host "==> $m" -ForegroundColor Cyan }
function Die  { param($m) Write-Host "error: $m" -ForegroundColor Red; exit 1 }

# --- preflight ---------------------------------------------------------------
if (-not (Get-Command go -ErrorAction SilentlyContinue)) { Die "'go' is not on PATH -- needed to build the release matrix" }
if (-not $DryRun -and -not (Get-Command gh -ErrorAction SilentlyContinue)) {
  Die "'gh' is not on PATH -- needed to create the GitHub Release (or pass -DryRun)"
}

# --- resolve version ---------------------------------------------------------
$versionGo = Join-Path $RepoRoot 'internal/version/version.go'
$m = Select-String -Path $versionGo -Pattern 'const Version = "([^"]+)"' | Select-Object -First 1
if (-not $m) { Die "could not read Version const from $versionGo -- pass -Version explicitly" }
$ConstVersion = 'v' + $m.Matches[0].Groups[1].Value

if (-not $Version) {
  $Version = $ConstVersion
} elseif ($Version -ne $ConstVersion) {
  Die "-Version $Version does not match the const in $versionGo ($ConstVersion) -- bump the const or drop -Version"
}
if ($Version -notmatch '^v\d+\.\d+\.\d+') { Die "version '$Version' should look like v1.2.3" }
Say "Publishing $Version"

# --- clean dist --------------------------------------------------------------
if (Test-Path $Dist) { Remove-Item $Dist -Recurse -Force }
New-Item -ItemType Directory -Path $Stage -Force | Out-Null

Push-Location $RepoRoot
try {
  foreach ($t in $Targets) {
    $os = $t.os; $arch = $t.arch
    $ext = if ($os -eq 'windows') { '.exe' } else { '' }
    Say "building $os/$arch"

    $sdir = Join-Path $Stage "$os-$arch"
    foreach ($sub in 'commands','agents','skills','output-styles') {
      New-Item -ItemType Directory -Path (Join-Path $sdir $sub) -Force | Out-Null
    }

    # Build straight with the Go toolchain -- no make dependency, so publish
    # works on a stock Windows box. Flags mirror the Makefile release target.
    $env:GOOS = $os; $env:GOARCH = $arch; $env:CGO_ENABLED = '0'
    & go build -trimpath -ldflags '-s -w' -o (Join-Path $sdir "apex$ext") ./cmd/apex
    if ($LASTEXITCODE -ne 0) { Die "go build failed for $os/$arch" }

    # Bundle the platform-independent artifacts alongside the binary.
    Copy-Item "$RepoRoot/commands/ax-*.md" (Join-Path $sdir 'commands')
    Copy-Item "$RepoRoot/agents/ax-*.md"   (Join-Path $sdir 'agents')
    Copy-Item "$RepoRoot/skills/ax-*"      (Join-Path $sdir 'skills') -Recurse
    Copy-Item "$RepoRoot/output-styles/protocol.md" (Join-Path $sdir 'output-styles/apex.md')

    $zip = Join-Path $Dist "apex-claude-$os-$arch.zip"
    Compress-Archive -Path (Join-Path $sdir '*') -DestinationPath $zip -Force
    Say "  bundled -> $zip"
  }
} finally {
  Remove-Item Env:GOOS, Env:GOARCH, Env:CGO_ENABLED -ErrorAction SilentlyContinue
  Pop-Location
}

$zips = Get-ChildItem (Join-Path $Dist 'apex-claude-*.zip')
Say "Built $($zips.Count) bundles into $Dist"

# SHA256SUMS in coreutils format -- lower-hex, two spaces, bare filename -- so
# both `sha256sum -c` and `apex update` can verify the bundles. LF-only, no
# BOM: sha256sum rejects CRLF/BOM'd checksum lines.
$Sums = Join-Path $Dist 'SHA256SUMS'
$sumLines = $zips | ForEach-Object {
  '{0}  {1}' -f (Get-FileHash $_.FullName -Algorithm SHA256).Hash.ToLowerInvariant(), $_.Name
}
[System.IO.File]::WriteAllText($Sums, (($sumLines -join "`n") + "`n"))
Say "checksums -> $Sums"

if ($DryRun) {
  Write-Host "`n[ok] Dry run -- bundles in $Dist, no release created." -ForegroundColor Green
  return
}

# --- create / update the GitHub Release --------------------------------------
$notes = "Apex Claude $Version`n`nLinux / macOS install:`n``````bash`ncurl -fsSL https://github.com/FNGApex/apex-claude/releases/latest/download/install-release.sh | bash`n```````n`nWindows install:`n``````powershell`nirm https://github.com/FNGApex/apex-claude/releases/latest/download/install.ps1 | iex`n```````n"

# Both one-line installers ride along as release assets: install.ps1 for the
# Windows `irm | iex` path, install-release.sh for the Linux/macOS `curl | bash`
# path. Their download URLs (releases/latest/download/<name>) depend on this.
$installers = @("$PSScriptRoot/install.ps1", "$PSScriptRoot/install-release.sh")

$exists = (& gh release view $Version 2>$null) -and ($LASTEXITCODE -eq 0)
if ($exists) {
  Say "Release $Version exists -- uploading assets (--clobber)"
  & gh release upload $Version $zips.FullName $Sums @installers --clobber
} else {
  Say "Creating release $Version"
  & gh release create $Version $zips.FullName $Sums @installers `
      --title "Apex Claude $Version" --notes $notes
}
if ($LASTEXITCODE -ne 0) { Die "gh release step failed" }

Write-Host "`n[ok] Published $Version." -ForegroundColor Green
Write-Host "  Linux/macOS: curl -fsSL https://github.com/FNGApex/apex-claude/releases/latest/download/install-release.sh | bash"
Write-Host "  Windows    : irm https://github.com/FNGApex/apex-claude/releases/latest/download/install.ps1 | iex"
