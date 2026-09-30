# Project signals

## Framework & runtime

- Language: Go 1.26, module `apexclaude` (no external dependencies)
- Version: `0.3.0` — single source of truth is `internal/version/version.go` (`const Version`); `.claude-plugin/plugin.json` and `.claude-plugin/marketplace.json` must carry the same string, enforced by a test (`TestPluginManifestsMatchVersion`)
- Install model: loose `~/.claude/` artifacts (commands, agents, skills, output-styles, binary) — NOT a Claude Code plugin; three install paths (see install domain)
- Hooks wired via `~/.claude/settings.json` (SessionStart only) by install scripts using an embedded Python (Unix) or native PowerShell (Windows) settings merge; no plugin enable/disable lifecycle. Installers strip any legacy Apex `PreToolUse` group so upgrades don't leave a hook pointing at a removed subcommand
- Binary: `bin/apex` (static, zero runtime deps); built with `make build` or `go build`
- Auto-update: `apex update` / `apex update check` (internal/update) — checks GitHub Releases for a newer `vX.Y.Z` tag, caches the result for 24h (`os.UserCacheDir()/apex-claude/update-check.json`), verifies downloaded bundles against `SHA256SUMS`, and swaps the binary in place (rename-dance on Windows, since a running exe can't be overwritten there). The SessionStart hook never touches the network: it nudges when the cache already names a newer release, and spawns a detached `apex update check --quiet` when the cache is missing or older than 24h; opt out with `APEX_NO_UPDATE_CHECK`.

## Build / test / lint

| Purpose | Command | Source |
|---------|---------|--------|
| Build binary | `make build` → `bin/apex` | Makefile |
| Run all tests | `make test` (go test ./...) | Makefile |
| Format Go | `make fmt` (gofmt -w cmd internal) | Makefile |
| Vet Go | `make vet` | Makefile |
| Cross-compile release matrix | `make release` → `bin/<os>-<arch>/apex` | Makefile |
| Install loose artifacts to ~/.claude (source) | `make install` → `scripts/install.sh` | Makefile / scripts/install.sh |
| Install loose artifacts to ~/.claude (prebuilt, Unix) | `scripts/install-release.sh` | scripts/install-release.sh |
| Install loose artifacts to ~/.claude (prebuilt, Windows) | `scripts/install.ps1` | scripts/install.ps1 |
| Remove loose artifacts from ~/.claude (Unix) | `make uninstall` → `scripts/uninstall.sh` | Makefile / scripts/uninstall.sh |
| Remove loose artifacts from ~/.claude (Windows) | `scripts/uninstall.ps1` | scripts/uninstall.ps1 |
| Publish prebuilt release bundles (Linux/macOS) | `scripts/publish.sh` | scripts/publish.sh |
| Publish prebuilt release bundles (Windows) | `scripts/publish.ps1` | scripts/publish.ps1 |
| Check for a newer release | `bin/apex update check [--quiet]` | internal/update |
| Apply an update in place (loose installs only) | `bin/apex update [--to vX.Y.Z]` | internal/update |
| Check signals freshness | `bin/apex signals stale` | internal/signals |
| Doctor check | `bin/apex doctor` | internal/doctor |
| Lint artifacts/specs | `bin/apex validate <artifacts\|spec\|all>` | internal/validate |
| Doc-surface staleness | `bin/apex docs stale` | internal/docs |
| Follow-up ledger | `bin/apex followups` | internal/followups |
| Reminders | `bin/apex reminder` | internal/reminder |
| Handoff report/route/archive | `bin/apex handoff <scan\|status\|archive>` | internal/handoff |
| Repo health score | `bin/apex health <show\|set>` | internal/health |

Release targets: darwin/arm64, darwin/amd64, linux/amd64, linux/arm64, windows/amd64.

**Install paths:**
1. Source build (Unix): `scripts/install.sh` — needs go + make + python3; flags: `--release`, `--no-build`, `--help`. Idempotent.
2. Prebuilt download (Linux/macOS): `scripts/install-release.sh` — downloads `apex-claude-<os>-<arch>.zip` from GitHub Releases via curl/wget; verifies against the release's `SHA256SUMS` (a 404 on that file means a pre-checksum release — warns and installs unverified; any other fetch failure or a mismatch dies); needs curl/wget + python3, NO go/make. Env overrides: `APEX_VERSION`, `CLAUDE_CONFIG_DIR`, `APEX_UPDATE_BASE_URL` (test seam). One-liner: `curl -fsSL .../install-release.sh | bash`.
3. Prebuilt download (Windows): `scripts/install.ps1` — PowerShell 5.1 and 7+; no external toolchain required; same `SHA256SUMS` verification (via .NET `SHA256`, not `Get-FileHash`, to dodge a `PSModulePath` shadowing bug in 5.1) and same 404-warns / mismatch-dies contract. Writes `settings.json` as UTF-8 **without** a BOM (5.1's `Set-Content -Encoding UTF8` would otherwise break Go's `encoding/json`, and so `apex doctor`).

Both prebuilt publish scripts (`scripts/publish.sh`, `scripts/publish.ps1`) read the version from `internal/version/version.go`'s const and refuse a `--version`/`-Version` override that disagrees with it; both emit `dist/SHA256SUMS` alongside the zips and upload it as a release asset.

## Language breakdown

| Language | LOC | Files | % |
|----------|-----|-------|---|
| Go | 5612 | 44 | 61% |
| Markdown | 2232 | 59 | 24% |
| Shell | 726 | 4 | 8% |
| PowerShell | 484 | 3 | 5% |
| JSON | 54 | 5 | 1% |
| YAML | 61 | 1 | 1% |

(Recomputed by direct `wc -l` per extension on 2026-09-13; no cloc/scc available in this environment.)

## DevOps & CI

CI: GitHub Actions (`.github/workflows/ci.yml`), triggers ONLY on release-candidate tags (`v*-rc*`, e.g. `v0.3.0-rc.1`) plus manual `workflow_dispatch` — pushes to `master` and pull requests run nothing. Release flow: tag `vX.Y.Z-rc.N` → CI green on all three OSes → publish `vX.Y.Z`. Matrix: `ubuntu-latest`, `windows-latest`, `macos-latest` — added so the Windows-only auto-update binary-swap/rollback path and the darwin release artifacts actually get exercised (a Ubuntu-only CI never ran either).
Gate order per OS: `go vet ./...` → `go test ./...` → `go build -trimpath -ldflags "-s -w" -o bin/apex[.exe] ./cmd/apex` → `./bin/apex doctor` → `./bin/apex validate all`. `gofmt` runs Linux-only (a Windows checkout converts line endings to CRLF, which gofmt would flag on every file even when unmodified). Build uses `go build` directly, not `make`, because `make` is not guaranteed present on Windows runners.
No deployment pipeline — release cross-compilation and GitHub Release publishing handled locally via `scripts/publish.sh` (Linux/macOS) or `scripts/publish.ps1` (Windows), each of which also writes `SHA256SUMS`. End-user install/update is one of three install paths (see above) plus `apex update` for in-place upgrades; none of this is CI-driven.

---

## Domains

| Domain | Repo paths | One-liner | Detail |
|--------|------------|-----------|--------|
| backbone | cmd/apex/, internal/, go.mod, Makefile | Go CLI (`apex` v0.3.0): signals scan, health score, session-start hook (+ update nudge), doctor, followups, reminders, validate, docs gate, handoff, and `apex update`/`update check` | .claude/project/signals/backbone.md |
| install | scripts/install.sh, scripts/install-release.sh, scripts/uninstall.sh, scripts/install.ps1, scripts/uninstall.ps1, scripts/publish.sh, scripts/publish.ps1 | Three install paths + two publish tools: (1) source build via install.sh/make (needs go+make+python3); (2) Unix prebuilt download via install-release.sh (needs curl/wget+python3, SHA256SUMS-verified); (3) Windows prebuilt via install.ps1 (PowerShell, no toolchain, SHA256SUMS-verified, BOM-free settings.json). publish.sh/publish.ps1 cross-compile, bundle, checksum, and upload GitHub Releases; both read the version from internal/version. | .claude/project/signals/install.md |
| plugin | .claude-plugin/, agents/, commands/, skills/, output-styles/, hooks/, CLAUDE.md | Claude Code artifact surface (`apex-claude` v0.3.0): 10 agents, 7 skills, 14 commands — full lifecycle roster (plan/implement/ship/diagnose/signals/handoff/resume/improve); hooks.json retained as reference but hooks now wired by install scripts into settings.json, not via plugin manifest | .claude/project/signals/plugin.md |

## Cross-cutting

- Test layout: `*_test.go` co-located with packages under `internal/` and `cmd/apex/`; no separate `test/` directory
- Project state files live in `.claude/project/`: `deterministic-signals.md` (scan output), `health.md` (integrity score), `doc-surfaces.md` (docs cache), `followups/` (ledger), `reminders/` (due nudges)
- Deterministic substrate: `.claude/project/deterministic-signals.md` (written by `apex signals scan`)
- Domain partitioning basis: backbone = Go packages + CLI (deterministic layer); install = all deploy/publish scripts (distribution mechanism); plugin = Claude Code artifact surface (what Claude reads and interprets)
- Install model: three distinct install paths — source build (install.sh, needs go+make), Unix prebuilt (install-release.sh, needs only curl/wget+python3), Windows prebuilt (install.ps1, bundled PowerShell). Both prebuilt paths verify `SHA256SUMS`; `install.sh` doesn't need to since it builds locally. Prior plugin manifest (`.claude-plugin/`) is retained in repo but is not the active install vehicle. Commands appear as bare `/ax-*` (not `/apex-claude:ax-*`).
- Publish tooling: two maintainer-facing publish scripts (publish.sh for Linux/macOS, publish.ps1 for Windows) cross-compile the full release matrix, bundle each platform as `apex-claude-<os>-<arch>.zip`, write `SHA256SUMS`, and upload everything to a GitHub Release via `gh`; release notes include both the Unix curl one-liner and the Windows irm one-liner. Both refuse an explicit version override that disagrees with `internal/version`'s const.
- In-place update: `apex update` (loose installs only — refuses on a dev/plugin layout with `dev layout — use 'git pull && make install'`) downloads the release zip + `SHA256SUMS` for the target tag, verifies the checksum, extracts via `archive/zip` with zip-slip guards, overwrites `commands/ax-*.md` + `agents/ax-*.md` + `output-styles/apex.md`, wholesale-replaces each `skills/ax-*` dir, and swaps the binary (atomic rename on Unix; rename-aside/write/rollback dance on Windows, since Windows won't let a running exe be opened for write). `apex update check` (network, cache-refreshing, exit 0/1/2) is layout-agnostic and is what the SessionStart hook spawns detached when the 24h cache goes stale.
- Cross-domain coupling: all install scripts (install domain) build/bundle from backbone and copy plugin artifacts; hooks wired into `~/.claude/settings.json` reference `~/.claude/bin/apex` (the installed backbone binary). `internal/doctor` (backbone) validates presence of plugin artifact directories relative to the resolved artifact root (`internal/layout.ArtifactRoot()`), branching its checks by dev/plugin layout vs. loose install.
- CI does not run install — CI gates are vet + test + build + doctor + validate; install/publish/update are local developer or end-user operations.
- `CLAUDE.md` at repo root carries the full Apex spine (principles, determinism boundary, lifecycle, agent/skill/command registries) plus `@.claude/project/signals.md` inside an `<apex-signals>` block — @-ref wiring verified active, unchanged this pass.
