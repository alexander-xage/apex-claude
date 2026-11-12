# Shared by install.ps1 and uninstall.ps1. Keep SKILLS and the marks in step with the Makefile.

function Fail { param($m) Write-Host "error: $m" -ForegroundColor Red; exit 1 }

function Get-ApexInstallConfig {
  param([string]$Prefix, [string]$ConfigDir)
  if (-not $Prefix) { $Prefix = Join-Path $env:USERPROFILE '.local\bin' }
  if (-not $ConfigDir) { $ConfigDir = if ($env:CLAUDE_CONFIG_DIR) { $env:CLAUDE_CONFIG_DIR } else { Join-Path $env:USERPROFILE '.claude' } }
  [pscustomobject]@{
    Prefix    = $Prefix
    ConfigDir = $ConfigDir
    SkillsDir = Join-Path $ConfigDir 'skills'
    Style     = Join-Path $ConfigDir 'output-styles\apex.md'
    LegacyBin = Join-Path $ConfigDir 'bin\apex.exe'
    Mark      = '.apex-installed'
    Skills    = @('agent-protocol', 'git-discipline', 'sign-off', 'apex-init', 'pr-review')
  }
}

# The output style is one file, so its own frontmatter name is the ownership mark.
function Test-ApexStyle { param($Path) [bool](Select-String -Path $Path -Pattern '^name: Apex$' -Quiet) }
