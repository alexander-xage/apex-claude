<#
.SYNOPSIS
  install.ps1 -- Windows counterpart to `make install`.

.DESCRIPTION
  Builds apex, copies it to -Prefix, and installs the Apex skills and output style into
  -ConfigDir as user skills (bare names, not a plugin). Same checks and ownership rules as the
  Makefile: stops if ponytail is missing, if a skill folder or output-styles/apex.md
  exists that apex did not install, or if a legacy Apex v0.3 binary would shadow the new one.

  Runs on Windows PowerShell 5.1 and PowerShell 7+. Undo with scripts/uninstall.ps1.

.PARAMETER Prefix
  Where apex.exe goes. Default: $env:USERPROFILE\.local\bin. Must be on PATH.

.PARAMETER ConfigDir
  Claude config root. Default: $env:CLAUDE_CONFIG_DIR or $env:USERPROFILE\.claude.
#>
[CmdletBinding()]
param([string]$Prefix, [string]$ConfigDir)

$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'apex-install-common.ps1')

$cfg = Get-ApexInstallConfig -Prefix $Prefix -ConfigDir $ConfigDir
$repo = Split-Path $PSScriptRoot -Parent

# check
$plugins = Join-Path $cfg.ConfigDir 'plugins/installed_plugins.json'
if (-not ((Test-Path $plugins) -and (Select-String -Path $plugins -Pattern '"ponytail@ponytail"' -SimpleMatch -Quiet))) {
  Fail 'ponytail plugin is not installed: /plugin marketplace add DietrichGebert/ponytail, then /plugin install ponytail@ponytail'
}
if (Test-Path $cfg.LegacyBin) {
  Fail "$($cfg.LegacyBin) is a legacy Apex v0.3 install and shadows the new apex on PATH; remove it first: pwsh example/scripts/uninstall.ps1"
}
foreach ($s in $cfg.Skills) {
  $d = Join-Path $cfg.SkillsDir $s
  if ((Test-Path $d) -and -not (Test-Path (Join-Path $d $cfg.Mark))) { Fail "$d exists and was not installed by apex; move it first" }
}
if ((Test-Path $cfg.Style) -and -not (Test-ApexStyle $cfg.Style)) { Fail "$($cfg.Style) exists and was not installed by apex; move it first" }

# build
Push-Location $repo
try {
  & go build -trimpath -o bin/apex.exe ./cmd/apex
  if ($LASTEXITCODE -ne 0) { Fail "go build failed ($LASTEXITCODE)" }
} finally { Pop-Location }

# install
foreach ($d in $cfg.Prefix, $cfg.SkillsDir, (Split-Path $cfg.Style -Parent)) { New-Item -ItemType Directory -Force $d | Out-Null }
Copy-Item (Join-Path $repo 'bin/apex.exe') (Join-Path $cfg.Prefix 'apex.exe') -Force
Copy-Item (Join-Path $repo 'output-styles/apex.md') $cfg.Style -Force
foreach ($s in $cfg.Skills) {
  $d = Join-Path $cfg.SkillsDir $s
  if (Test-Path $d) { Remove-Item $d -Recurse -Force }
  Copy-Item (Join-Path $repo "skills/$s") $d -Recurse
  New-Item -ItemType File (Join-Path $d $cfg.Mark) | Out-Null
}

$onPath = ($env:PATH -split ';') | Where-Object { $_ -and ($_.TrimEnd('\','/') -ieq $cfg.Prefix.TrimEnd('\','/')) }
if (-not $onPath) { Write-Warning "$($cfg.Prefix) is not on PATH; every skill calls a bare apex" }
Write-Host "[ok] apex -> $($cfg.Prefix); skills and output style -> $($cfg.ConfigDir). Restart Claude Code." -ForegroundColor Green
