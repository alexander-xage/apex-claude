<#
.SYNOPSIS
  uninstall.ps1 -- remove the loose Apex Claude artifacts on native Windows.

.DESCRIPTION
  The Windows-native counterpart to scripts/uninstall.sh. Removes the ax-*
  commands/agents/skills, the apex output style, the binary, and the apex hook
  entries from ~/.claude/settings.json. Every other setting and any non-Apex
  artifact is left untouched. It does NOT touch ~/.claude/CLAUDE.md.

  Runs on Windows PowerShell 5.1 and PowerShell 7+.

.PARAMETER ConfigDir
  Install root. Default: $env:CLAUDE_CONFIG_DIR or $env:USERPROFILE\.claude.

.EXAMPLE
  pwsh scripts/uninstall.ps1
#>
[CmdletBinding()]
param([string]$ConfigDir)

$ErrorActionPreference = 'Stop'

if (-not $ConfigDir) { $ConfigDir = if ($env:CLAUDE_CONFIG_DIR) { $env:CLAUDE_CONFIG_DIR } else { Join-Path $env:USERPROFILE '.claude' } }

function Say { param($m) Write-Host "==> $m" -ForegroundColor Cyan }

Say "Removing Apex artifacts from $ConfigDir"
Remove-Item (Join-Path $ConfigDir 'commands/ax-*.md')      -Force -ErrorAction SilentlyContinue
Remove-Item (Join-Path $ConfigDir 'agents/ax-*.md')        -Force -ErrorAction SilentlyContinue
Remove-Item (Join-Path $ConfigDir 'skills/ax-*')           -Recurse -Force -ErrorAction SilentlyContinue
Remove-Item (Join-Path $ConfigDir 'output-styles/apex.md') -Force -ErrorAction SilentlyContinue
Remove-Item (Join-Path $ConfigDir 'bin/apex.exe')          -Force -ErrorAction SilentlyContinue

$settingsPath = Join-Path $ConfigDir 'settings.json'
if (Test-Path $settingsPath) {
  Say "Stripping apex hooks from settings.json"
  $raw = Get-Content $settingsPath -Raw
  if ($raw.Trim()) {
    try { $data = $raw | ConvertFrom-Json } catch { $data = $null }
    if ($data -and ($data.PSObject.Properties.Name -contains 'hooks')) {
      $hooks = $data.hooks
      function Test-IsApex($group) {
        foreach ($h in @($group.hooks)) {
          if ([string]$h.command -match 'apex(\.exe)? hooks') { return $true }
        }
        return $false
      }
      foreach ($event in 'PreToolUse','SessionStart') {
        if ($hooks.PSObject.Properties.Name -contains $event) {
          $kept = @($hooks.$event | Where-Object { -not (Test-IsApex $_) })
          if ($kept.Count -gt 0) { $hooks.$event = $kept }
          else { $hooks.PSObject.Properties.Remove($event) }
        }
      }
      if ($hooks.PSObject.Properties.Name.Count -eq 0) {
        $data.PSObject.Properties.Remove('hooks')
      }
      # UTF-8 WITHOUT a BOM. Windows PowerShell 5.1's `Set-Content -Encoding UTF8`
      # prepends one, and JSON parsers (Go's encoding/json, and so `apex doctor`)
      # reject it -- every 5.1 install then read as having no hooks wired.
      # [IO.File] resolves relative paths against the process cwd, not the
      # PowerShell location, so resolve first.
      $settingsFull = $ExecutionContext.SessionState.Path.GetUnresolvedProviderPathFromPSPath($settingsPath)
      [System.IO.File]::WriteAllText($settingsFull, ($data | ConvertTo-Json -Depth 20), (New-Object System.Text.UTF8Encoding $false))
    }
  }
}

Write-Host "[ok] Apex Claude removed. Restart Claude Code to drop /ax-*." -ForegroundColor Green
