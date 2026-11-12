<#
.SYNOPSIS
  uninstall.ps1 -- Windows counterpart to `make uninstall`.

.DESCRIPTION
  Removes apex.exe from -Prefix, and the Apex skills and output style from -ConfigDir -- only
  those apex installed (skill folders carrying the .apex-installed mark; an output style whose
  frontmatter is `name: Apex`). Runs on Windows PowerShell 5.1 and PowerShell 7+.
#>
[CmdletBinding()]
param([string]$Prefix, [string]$ConfigDir)

$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'apex-install-common.ps1')

$cfg = Get-ApexInstallConfig -Prefix $Prefix -ConfigDir $ConfigDir

Remove-Item (Join-Path $cfg.Prefix 'apex.exe') -Force -ErrorAction SilentlyContinue
if ((Test-Path $cfg.Style) -and (Test-ApexStyle $cfg.Style)) { Remove-Item $cfg.Style -Force }
foreach ($s in $cfg.Skills) {
  $d = Join-Path $cfg.SkillsDir $s
  if (Test-Path (Join-Path $d $cfg.Mark)) { Remove-Item $d -Recurse -Force }
}
Write-Host '[ok] apex removed. Restart Claude Code.' -ForegroundColor Green
