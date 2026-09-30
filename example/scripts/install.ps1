<#
.SYNOPSIS
  install.ps1 -- install Apex Claude on native Windows (no bash/python/go).

.DESCRIPTION
  The Windows-native counterpart to scripts/install.sh. Downloads the prebuilt
  release bundle (apex.exe + commands/agents/skills/output-style) published by
  scripts/publish.ps1, copies the loose artifacts into ~/.claude, installs the
  binary into ~/.claude/bin, and wires the SessionStart hook into
  ~/.claude/settings.json -- stripping any legacy Apex PreToolUse group and
  preserving every other setting.

  Designed for one-line install:

      irm https://github.com/FNGApex/apex-claude/releases/latest/download/install.ps1 | iex

  Because `iex` cannot pass parameters, overrides are read from env vars:
      $env:APEX_VERSION      pin a release tag (default: latest)
      $env:CLAUDE_CONFIG_DIR install root      (default: $env:USERPROFILE\.claude)

  Runs on Windows PowerShell 5.1 and PowerShell 7+. It does NOT touch
  ~/.claude/CLAUDE.md -- the Apex spine is opt-in. Remove with uninstall.ps1.

.PARAMETER Version
  Release tag to install (e.g. v0.2.0). Default: latest.

.PARAMETER ConfigDir
  Install root. Default: $env:CLAUDE_CONFIG_DIR or $env:USERPROFILE\.claude.

.EXAMPLE
  irm .../install.ps1 | iex
  pwsh scripts/install.ps1 -Version v0.2.0
#>
[CmdletBinding()]
param(
  [string]$Version,
  [string]$ConfigDir
)

$ErrorActionPreference = 'Stop'

$Repo = 'FNGApex/apex-claude'

if (-not $Version)   { $Version   = if ($env:APEX_VERSION) { $env:APEX_VERSION } else { 'latest' } }
if (-not $ConfigDir) { $ConfigDir = if ($env:CLAUDE_CONFIG_DIR) { $env:CLAUDE_CONFIG_DIR } else { Join-Path $env:USERPROFILE '.claude' } }

function Say { param($m) Write-Host "==> $m" -ForegroundColor Cyan }
function Die { param($m) Write-Host "error: $m" -ForegroundColor Red; exit 1 }

# --- 1. resolve download URL -------------------------------------------------
$asset = 'apex-claude-windows-amd64.zip'

# assetUrl builds a release-asset URL for $Version. APEX_UPDATE_BASE_URL
# overrides the host wholesale as <base>/<version>/<name> -- the same seam
# `apex update` honors -- so a local server can stand in for GitHub in tests.
function Get-AssetUrl { param($name)
  if ($env:APEX_UPDATE_BASE_URL) { return "$($env:APEX_UPDATE_BASE_URL.TrimEnd('/'))/$Version/$name" }
  if ($Version -eq 'latest') { return "https://github.com/$Repo/releases/latest/download/$name" }
  return "https://github.com/$Repo/releases/download/$Version/$name"
}
$url = Get-AssetUrl $asset

$tmp = Join-Path ([System.IO.Path]::GetTempPath()) ("apex-install-" + [System.IO.Path]::GetRandomFileName())
New-Item -ItemType Directory -Path $tmp -Force | Out-Null
$zip = Join-Path $tmp $asset

try {
  # --- 1a. fetch SHA256SUMS ----------------------------------------------------
  # Fetched BEFORE the bundle, from the same $Version. Fetching it after a
  # multi-MB download made Windows PowerShell 5.1 reuse a pooled github.com
  # connection the server had already closed ("connection was closed
  # unexpectedly") -- so order matters, and a transport error gets one retry.
  # Only an HTTP 404 means "pre-checksum release": warn and proceed. Any other
  # failure dies -- silently skipping verification on a flaky fetch would turn
  # the check into a suggestion. For 'latest', a release published between
  # this fetch and the bundle download yields a mismatch: loud, safe, re-runnable.
  $sumsUrl = Get-AssetUrl 'SHA256SUMS'
  $sums = $null
  $sumsMissing = $false
  foreach ($attempt in 1..2) {
    try {
      $sums = (Invoke-WebRequest -Uri $sumsUrl -UseBasicParsing).Content
      if ($sums -is [byte[]]) { $sums = [System.Text.Encoding]::UTF8.GetString($sums) }
      break
    } catch {
      $resp = $_.Exception.Response
      if ($resp -and [int]$resp.StatusCode -eq 404) { $sumsMissing = $true; break }
      if ($resp -or $attempt -eq 2) {
        Die "could not fetch SHA256SUMS from $sumsUrl ($($_.Exception.Message))"
      }
    }
  }

  Say "Downloading $Version bundle"
  $oldProgress = $ProgressPreference
  $ProgressPreference = 'SilentlyContinue'
  try {
    Invoke-WebRequest -Uri $url -OutFile $zip -UseBasicParsing
  } catch {
    Die "download failed from $url -- check the version tag and that a release exists ($($_.Exception.Message))"
  } finally {
    $ProgressPreference = $oldProgress
  }

  # --- 1b. verify checksum -----------------------------------------------------
  if ($sumsMissing) {
    Write-Host "warning: release $Version has no SHA256SUMS (pre-checksum release) -- installing unverified" -ForegroundColor Yellow
  } else {
    $expected = $null
    foreach ($line in ($sums -split "`n")) {
      $parts = $line.Trim() -split '\s+', 2
      if ($parts.Count -eq 2 -and $parts[1] -eq $asset) { $expected = $parts[0].ToLowerInvariant() }
    }
    if (-not $expected) { Die "SHA256SUMS for $Version has no entry for $asset -- refusing to install" }
    # .NET directly, not Get-FileHash: Windows PowerShell 5.1 launched from a
    # PowerShell 7 session inherits a PSModulePath that shadows its own
    # Microsoft.PowerShell.Utility, and Get-FileHash then fails to resolve.
    $stream = [System.IO.File]::OpenRead($zip)
    try {
      $sha = [System.Security.Cryptography.SHA256]::Create()
      $actual = -join ($sha.ComputeHash($stream) | ForEach-Object { $_.ToString('x2') })
    } finally { $stream.Dispose() }
    if ($actual -ne $expected) {
      Die "checksum mismatch for $asset (expected $expected, got $actual) -- download corrupt or tampered"
    }
    Say "Checksum verified"
  }

  Say "Extracting"
  $src = Join-Path $tmp 'bundle'
  Expand-Archive -Path $zip -DestinationPath $src -Force

  $exe = Join-Path $src 'apex.exe'
  if (-not (Test-Path $exe)) { Die "bundle is missing apex.exe -- corrupt or wrong asset" }

  # --- 2. drop any prior plugin install (migration) --------------------------
  if (Get-Command claude -ErrorAction SilentlyContinue) {
    $list = & claude plugin list 2>$null
    if ($LASTEXITCODE -eq 0 -and ($list -match 'apex-claude@apex-claude')) {
      Say "Removing prior plugin install of Apex (switching to loose artifacts)"
      & claude plugin uninstall 'apex-claude@apex-claude' 2>$null
      & claude plugin marketplace remove 'apex-claude' 2>$null
    }
  }

  # --- 3. copy artifacts -----------------------------------------------------
  Say "Installing artifacts into $ConfigDir"
  foreach ($sub in 'commands','agents','skills','output-styles','bin') {
    New-Item -ItemType Directory -Path (Join-Path $ConfigDir $sub) -Force | Out-Null
  }
  # Prune ax-* artifacts this release no longer ships, so a command cut from
  # Apex disappears instead of lingering in the slash menu. ax-* is Apex's
  # namespace (uninstall removes the whole prefix); other files are the user's.
  # A kind the bundle doesn't ship at all is skipped, so a malformed bundle
  # can never wipe the installed set.
  function Remove-Unshipped($srcDir, $dstDir, $filter) {
    $shipped = @(Get-ChildItem -Path $srcDir -Filter $filter -ErrorAction SilentlyContinue | ForEach-Object { $_.Name })
    if ($shipped.Count -eq 0) { return }
    Get-ChildItem -Path $dstDir -Filter $filter -ErrorAction SilentlyContinue |
      Where-Object { $shipped -notcontains $_.Name } |
      ForEach-Object { Remove-Item $_.FullName -Recurse -Force }
  }
  Remove-Unshipped (Join-Path $src 'commands') (Join-Path $ConfigDir 'commands') 'ax-*.md'
  Remove-Unshipped (Join-Path $src 'agents')   (Join-Path $ConfigDir 'agents')   'ax-*.md'
  Remove-Unshipped (Join-Path $src 'skills')   (Join-Path $ConfigDir 'skills')   'ax-*'

  Copy-Item (Join-Path $src 'commands/ax-*.md') (Join-Path $ConfigDir 'commands') -Force
  Copy-Item (Join-Path $src 'agents/ax-*.md')   (Join-Path $ConfigDir 'agents')   -Force
  Copy-Item (Join-Path $src 'output-styles/apex.md') (Join-Path $ConfigDir 'output-styles/apex.md') -Force

  # Skills are dir/SKILL.md -- replace each ax-* skill dir wholesale.
  Get-ChildItem (Join-Path $src 'skills') -Directory | ForEach-Object {
    $dest = Join-Path $ConfigDir "skills/$($_.Name)"
    if (Test-Path $dest) { Remove-Item $dest -Recurse -Force }
    Copy-Item $_.FullName $dest -Recurse -Force
  }

  # --- 4. binary -------------------------------------------------------------
  $apexBin = Join-Path $ConfigDir 'bin/apex.exe'
  Say "Installing binary into $apexBin"
  Copy-Item $exe $apexBin -Force

  # --- 5. wire hooks (preserve all other settings) ---------------------------
  $settingsPath = Join-Path $ConfigDir 'settings.json'
  Say "Wiring hooks into $settingsPath"

  # Hook command: forward-slash full path to the .exe (Claude Code on Windows
  # invokes the executable directly; forward slashes are the documented form).
  # Quote only if the path contains a space.
  $binPath = ($apexBin -replace '\\','/')
  $cmdBin  = if ($binPath -match ' ') { '"' + $binPath + '"' } else { $binPath }

  $data = [pscustomobject]@{}
  if (Test-Path $settingsPath) {
    $raw = Get-Content $settingsPath -Raw
    if ($raw.Trim()) {
      try { $data = $raw | ConvertFrom-Json } catch {
        Die "settings.json is not valid JSON -- fix it by hand and re-run ($($_.Exception.Message))"
      }
    }
  }

  function Has-Prop($obj, $name) { $obj.PSObject.Properties.Name -contains $name }
  function Test-IsApex($group) {
    foreach ($h in @($group.hooks)) {
      if ([string]$h.command -match 'apex(\.exe)? hooks') { return $true }
    }
    return $false
  }

  if (-not (Has-Prop $data 'hooks')) {
    $data | Add-Member -NotePropertyName 'hooks' -NotePropertyValue ([pscustomobject]@{})
  }
  $hooks = $data.hooks

  $newGroups = @{
    SessionStart = [pscustomobject]@{ hooks = @([pscustomobject]@{ type = 'command'; command = "$cmdBin hooks session-start" }) }
  }

  # Strip any prior apex group so re-runs don't stack duplicates. PreToolUse is
  # stripped but never re-added: Apex no longer ships a bash guard (Claude Code's
  # own auto-mode owns that), so an upgrade from an older Apex must clean the
  # stale PreToolUse entry rather than leave it pointing at a removed subcommand.
  foreach ($event in 'PreToolUse','SessionStart') {
    $kept = @()
    if (Has-Prop $hooks $event) {
      $kept = @($hooks.$event | Where-Object { -not (Test-IsApex $_) })
    }
    $merged = @($kept)
    if ($newGroups.ContainsKey($event)) { $merged = @($kept) + @($newGroups[$event]) }
    if (Has-Prop $hooks $event) { $hooks.$event = $merged }
    elseif ($merged.Count -gt 0) { $hooks | Add-Member -NotePropertyName $event -NotePropertyValue $merged }
  }

  # UTF-8 WITHOUT a BOM. Windows PowerShell 5.1's `Set-Content -Encoding UTF8`
  # prepends one, and JSON parsers (Go's encoding/json, and so `apex doctor`)
  # reject it -- every 5.1 install then read as having no hooks wired.
  # [IO.File] resolves relative paths against the process cwd, not the
  # PowerShell location, so resolve first.
  $settingsFull = $ExecutionContext.SessionState.Path.GetUnresolvedProviderPathFromPSPath($settingsPath)
  [System.IO.File]::WriteAllText($settingsFull, ($data | ConvertTo-Json -Depth 20), (New-Object System.Text.UTF8Encoding $false))
  Write-Host "  hooks wired -> $binPath"

  # --- done ------------------------------------------------------------------
  $cmdCount   = (Get-ChildItem (Join-Path $ConfigDir 'commands/ax-*.md')).Count
  $agentCount = (Get-ChildItem (Join-Path $ConfigDir 'agents/ax-*.md')).Count
  $skillCount = (Get-ChildItem (Join-Path $ConfigDir 'skills') -Directory -Filter 'ax-*').Count

  Write-Host ""
  Write-Host "[ok] Apex Claude installed (loose artifacts)." -ForegroundColor Green
  Write-Host "  commands : $cmdCount  -> $ConfigDir\commands"
  Write-Host "  agents   : $agentCount  -> $ConfigDir\agents"
  Write-Host "  skills   : $skillCount   -> $ConfigDir\skills"
  Write-Host "  style    : Apex -> $ConfigDir\output-styles\apex.md"
  Write-Host ""
  Write-Host "Next steps:"
  Write-Host "  - Restart Claude Code so /ax-* commands, agents, skills, and hooks load."
  Write-Host "  - Activate the output style: /output-style Apex"

  # Trim trailing separators so entries like "C:\x\bin\" still match
  # (-contains is already case-insensitive for strings).
  $binDir = Join-Path $ConfigDir 'bin'
  $onPath = ($env:PATH -split ';' | Where-Object { $_ } | ForEach-Object { $_.TrimEnd('\', '/') }) -contains $binDir.TrimEnd('\', '/')
  if (-not $onPath) {
    Write-Host ""
    Write-Host "! $binDir is not on your PATH" -ForegroundColor Yellow
    Write-Host "  Claude Code's hooks call apex by full path, so they work regardless."
    Write-Host "  To run 'apex' yourself, add it to PATH (persists for new shells):"
    Write-Host "    [Environment]::SetEnvironmentVariable('Path', `"$binDir;`$env:Path`", 'User')"
  }
} finally {
  Remove-Item $tmp -Recurse -Force -ErrorAction SilentlyContinue
}
