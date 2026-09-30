# install

## What it does
Three idempotent install paths deploy Apex Claude as loose user-level artifacts in `~/.claude/` (not as a Claude Code plugin), plus two matching uninstallers and two maintainer-facing publish scripts. Every install/uninstall path wires or strips the SessionStart hook in `~/.claude/settings.json` directly — no plugin enable/disable lifecycle — and is safe to re-run.

## Install paths
- **`scripts/install.sh`** (Unix, source build) — build → migrate → copy artifacts → install binary → wire hooks. `make install` delegates to it. Needs `go` + `make` + `python3` (python3 does the settings.json merge). Flags: `--release` (cross-compile matrix first), `--no-build` (install the existing `bin/apex`), `--help`.
- **`scripts/install-release.sh`** (Linux/macOS, prebuilt) — downloads `apex-claude-<os>-<arch>.zip` from the resolved GitHub release, fetches and verifies `SHA256SUMS` **before** downloading the bundle, then copies artifacts/binary and wires hooks via the same python3 settings.json merge. Needs `curl` or `wget` + `python3`; no Go toolchain. Designed for `curl -fsSL .../install-release.sh | bash`.
- **`scripts/install.ps1`** (Windows, prebuilt) — native PowerShell, runs on 5.1 and 7+, no external toolchain. Same SHA256SUMS-before-download flow (retries the SHA256SUMS fetch once on a transport error — PS 5.1 was found to reuse a server-closed pooled connection after a multi-MB download); hashes via .NET `SHA256` directly rather than `Get-FileHash` (which fails to resolve under 5.1 when a PowerShell 7 `PSModulePath` is inherited). Writes `settings.json` as UTF-8 **without** a BOM via `[System.IO.File]::WriteAllText` (5.1's `Set-Content -Encoding UTF8` prepends one, which breaks Go's `encoding/json` and so `apex doctor`/`internal/layout.ApexHooksWired`). Designed for `irm .../install.ps1 | iex`.

All three: migrate away from a prior plugin install of `apex-claude@apex-claude` if `claude` CLI is present; copy `commands/ax-*.md`, `agents/ax-*.md`, `skills/ax-*/`, `output-styles/apex.md`; install the binary to `<config-dir>/bin/apex[.exe]`; strip any legacy Apex `PreToolUse` group from settings.json (never re-added — Apex ships no bash guard, Claude Code's own auto-mode owns that) and merge in a fresh `SessionStart` entry pointing at the installed binary. None of them touch `~/.claude/CLAUDE.md` — the Apex spine is opt-in.

## SHA256SUMS verification (prebuilt paths only)
Both `install-release.sh` and `install.ps1` treat SHA256SUMS fetch outcomes identically: HTTP 404 = pre-checksum release → warn and install unverified; any other fetch failure → die; a fetched-but-asset-missing entry → die; a hash mismatch → die ("corrupt or tampered"). `install.sh` doesn't need this — it builds the binary locally from source. All three honor `APEX_UPDATE_BASE_URL` as `<base>/<version>/<asset>`, the same test seam `internal/update` uses, so a local server can stand in for GitHub.

## Uninstall
- `scripts/uninstall.sh` (Unix; also detects `apex.exe` under MINGW/MSYS/CYGWIN) / `scripts/uninstall.ps1` (Windows) — remove `commands/ax-*.md`, `agents/ax-*.md`, `skills/ax-*`, `output-styles/apex.md`, the installed binary; strip apex hook entries from settings.json (deletes empty hook sections); every other setting and non-Apex artifact untouched. `make uninstall` delegates to `uninstall.sh`.

## Publish (maintainer-only, not end-user install)
- `scripts/publish.sh` (Linux/macOS) / `scripts/publish.ps1` (Windows) — cross-compile the full release matrix, bundle each platform as `apex-claude-<os>-<arch>.zip`, write `dist/SHA256SUMS` in coreutils two-space format (`sha256sum`/`shasum -a 256` on Unix, `Get-FileHash` on Windows), and `gh release create`/`upload` the zips + SHA256SUMS + installer scripts. Both read the version from `internal/version/version.go`'s `const Version` and die if an explicit `--version`/`-Version` override disagrees with it (`v` + const is the only accepted form). Release notes embed both the Unix curl one-liner and the Windows irm one-liner.

## Install path vs. repo path
The installed binary lives at `<config-dir>/bin/apex[.exe]` (default `~/.claude`); the repo binary lives at `bin/apex`. Hook commands reference the installed path. After code changes in this repo, re-run `make install` (or `apex update` against a published release) to refresh the installed binary; the repo binary is refreshed by `make build`.

## Prerequisites
- `install.sh`: `go`, `make`, `python3` (unless `--no-build`, which still needs an existing `bin/apex`)
- `install-release.sh`: `curl` or `wget`, `python3`, `sha256sum` or `shasum` (skippable only if the release predates SHA256SUMS)
- `install.ps1`: PowerShell 5.1+ only — no external toolchain
- `claude` CLI on PATH only needed if migrating from a prior plugin install

## Coupling
- Install/update scripts copy artifact files from the plugin domain (commands/, agents/, skills/, output-styles/); any rename or addition in those directories requires a corresponding release/re-run to propagate.
- The hook command is hardcoded (per script) to `<bin>/apex[.exe] hooks session-start`; renaming backbone subcommands requires updating all three install scripts, both uninstall scripts, and internal/layout's `isApexHookCmd` matcher in lockstep.
- `internal/update.RepoSlug` ("FNGApex/apex-claude") must match `$Repo` in install.ps1 and `REPO` in install-release.sh — these are three independently-maintained copies of the same string.
- `$CLAUDE_CONFIG_DIR` env var overrides the default `~/.claude` destination on every install/uninstall script (useful for testing in alternate installs); `APEX_VERSION`/`$env:APEX_VERSION` pins a release tag on the two prebuilt installers.
- `internal/update.Apply`'s artifact-replace semantics must stay in lockstep with what all three install scripts do — a divergence means `apex update` and a fresh install produce different trees.
