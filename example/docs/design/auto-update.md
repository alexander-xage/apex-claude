# Auto-update — design

> **Status:** implemented and shipping in v0.3.0. This is the decision record as written on 2026-07-01 — version numbers, file locations and the installer description below reflect the tree at that time. For current behavior read `docs/spec/auto-update.md`, whose change log lists every later supersession (checksum verification in both installers, artifact pruning, Windows PowerShell 5.1 compatibility).

Goal: an installed (loose) copy of Apex Claude detects new GitHub Releases and can pull them —
binary AND markdown artifacts in lockstep — with zero LLM involvement, zero new Go dependencies,
and zero risk of blocking a Claude Code session.

## Current state (verified in-repo, 2026-07-01)

- Version lives as `const version = "0.2.0"` in `cmd/apex/main.go:16`; `apex version` prints it.
  It is NOT reachable from `internal/*` (package main), and `scripts/publish.ps1:68` regexes that
  exact const to derive the release tag. So version embedding exists but is mis-placed for reuse.
- `scripts/publish.ps1` builds the 5-target matrix with plain `go build -trimpath -ldflags '-s -w'`
  (no `-X` injection) and publishes per-platform bundles `apex-claude-<os>-<arch>.zip` containing
  `apex(.exe)` + `commands/ax-*.md` + `agents/ax-*.md` + `skills/ax-*/` + `output-styles/apex.md`,
  plus `install.ps1` as a release asset. No checksums file today.
- `scripts/install.ps1` downloads `releases/latest/download/apex-claude-windows-amd64.zip` (or a
  pinned `$env:APEX_VERSION` tag), copies artifacts into `$ConfigDir` (default `~/.claude`), puts
  the binary at `<ConfigDir>/bin/apex.exe`, and wires PreToolUse + SessionStart hooks into
  `settings.json` by absolute binary path. `scripts/install.sh` does the same from a source
  checkout (build + copy). No download verification today.
- `internal/hooks.SessionStart` emits nudges (stale signals, due reminders) as
  `additionalContext`; returns 0 always; silent when nothing pending. Natural nudge surface.
- `internal/doctor/doctor.go` owns `artifactRoot()` (env `CLAUDE_PLUGIN_ROOT` → exe-parent
  inference → `~/.claude` → cwd) and `isLooseInstall()` (absence of `.claude-plugin/`). The
  update path needs the same layout logic — it must be extracted, not duplicated.
- Subcommands self-register via `register()` in `cmd/apex/cmd_*.go` (`registry.go`); adding
  `apex update` touches no central switch.

## Evidence trail (external)

| Claim | Source | Verdict |
|---|---|---|
| `releases/latest` returns 302 with tag in `Location` | `curl -sI https://github.com/cli/cli/releases/latest` → `Location: .../releases/tag/v2.95.0` | supported |
| gh CLI checks `releases/latest` API with 24h TTL state file, disabled via `GH_NO_UPDATE_NOTIFIER`, TTY-only, never blocks command | `cli/cli` `internal/update/update.go` (fetched) | supported |
| GitHub REST API unauthenticated limit 60 req/hr/IP | GitHub REST docs (well-known; not re-fetched) | supported (common knowledge, low risk — and moot: chosen mechanism avoids the API) |
| Windows cannot overwrite a running exe but CAN rename it | minio/selfupdate rename-dance pattern (`.old` rename, write new, delete `.old` later) | supported (industry-standard pattern; not re-fetched) |
| `Compress-Archive` zips are readable by Go `archive/zip` | stdlib zip reader handles standard PKZIP; publish already relies on `Expand-Archive` interop | supported |

## Decisions

### 1. Version embedding — `internal/version`

Move the const to a new package: `internal/version/version.go` with `const Version = "0.2.0"`.
`cmd/apex/main.go` prints `version.Version`; `internal/update` compares against it;
`publish.ps1` regexes the new file and — when `-Version` is passed explicitly — FAILS on mismatch
with the const (today a mismatched `-Version` would ship a binary that self-reports the wrong
version, silently breaking update detection).

Rejected: ldflags `-X` injection. It splits truth between Makefile, publish.ps1, and CI; a dev
`make build` binary would report `dev`/empty and the const already works and is already the
publish source of truth. A const is greppable, testable, and identical across all build paths.

### 2. Check mechanism — redirect sniff, cached, never in the hook's critical path

`GET https://github.com/FNGApex/apex-claude/releases/latest` with
`CheckRedirect: return http.ErrUseLastResponse`, 5s timeout; parse the tag from the `Location`
header (`.../releases/tag/vX.Y.Z`). Properties: no API rate limit (web endpoint, verified), no
JSON parsing, no auth, one round trip, stdlib-only.

Cache: `<os.UserCacheDir()>/apex-claude/update-check.json`
(`%LocalAppData%\apex-claude\` on Windows, `~/.cache/apex-claude/` on Linux,
`~/Library/Caches/apex-claude/` on macOS):

```json
{"checked_at": "RFC3339", "latest": "vX.Y.Z"}
```

TTL 24h. A FAILED check also stamps `checked_at` (with `latest` left as previously known) so an
offline machine backs off for 24h instead of retrying every session. Cache is user-scoped, not
project-scoped — deliberately NOT under `.claude/project/` (an update is a property of the
install, not the repo) and not under `~/.claude` (keep Claude Code's config dir free of
apex-private state).

Version compare: parse `vMAJOR.MINOR.PATCH` into ints, lexicographic on the triple. Tags that
don't match `^v\d+\.\d+\.\d+$` are ignored (no nudge, no update) — `releases/latest` never points
at prereleases, so no prerelease ordering logic is needed.

Rejected: GitHub Releases REST API (rate-limited at 60/hr/IP unauthenticated; JSON parsing for
data we can reconstruct — asset URLs are fully determined by tag + naming contract). Rejected:
checked-in raw version file (second source of truth that can drift from the actual release; the
release tag IS the truth).

### 3. Update surface — `apex update`, native Go, artifacts in lockstep

New subcommand family (registered in `cmd/apex/cmd_update.go`, logic in `internal/update`):

- `apex update check` — foreground: refresh cache, print current vs latest. Exit 0 = up to date,
  1 = update available, 2 = check failed. This same invocation is what the session-start hook
  spawns detached; deterministic exit codes let anything gate on it.
- `apex update` — apply: resolve tag (latest via redirect sniff, or `--to vX.Y.Z` to pin),
  download `apex-claude-<os>-<arch>.zip` + `SHA256SUMS` from that tag, verify checksum, extract
  with `archive/zip` to a temp dir, copy `commands/agents/skills/output-styles` into the artifact
  root (same replace semantics as install.ps1: overwrite `ax-*.md` files, wholesale-replace each
  `ax-*` skill dir), then swap the binary (see §4). Prints old → new version. On success,
  rewrites the cache as up-to-date.

Lockstep is guaranteed by construction: the bundle publish.ps1 already ships contains binary AND
artifacts, so one download updates both. `apex update` never touches `settings.json` — the hook
command is an absolute path to `<root>/bin/apex(.exe)` which does not change across updates. It
DOES verify wiring post-update (reuse doctor's `apexHooksWired`) and warns if absent.

Guard: `apex update` refuses on a dev/plugin layout (`isLooseInstall == false`) with
"dev layout — use `git pull && make install`". Self-updating a git checkout from release zips
would clobber working-tree files. The layout logic (`artifactRoot`, `isLooseInstall`,
`apexHooksWired`) moves from `internal/doctor` to a new `internal/layout` package; doctor imports
it, behavior unchanged.

Rejected: `apex update` shelling out to install.ps1/install.sh — reintroduces the
PowerShell/bash/python3 runtime deps the prebuilt-bundle model exists to avoid, and puts update
logic outside the deterministic binary. Rejected: git-archive artifact fetch — second delivery
channel to keep consistent with releases; the zip bundle already exists.

### 4. Windows swap + mid-session semantics

Unix: write the new binary as `<root>/bin/.apex.new` (same filesystem), `chmod +x`,
`os.Rename` over `apex` — atomic.

Windows: a running exe cannot be opened for write but CAN be renamed. Dance:
`apex.exe → apex.exe.old`, write new `apex.exe`, keep running from the renamed image. On write
failure, rename `.old` back (rollback). `.old` cannot be deleted by the process it still backs —
best-effort `os.Remove` of `apex.exe.old` runs at the start of every `apex update` and
`apex hooks session-start` (a failed remove is silently ignored; it succeeds on the next fresh
process). This is the minio/selfupdate pattern implemented on stdlib.

Mid-session semantics: Claude Code holds no persistent handle on the binary — each hook event
spawns a fresh process from `<root>/bin/apex(.exe)`. A swap between invocations is therefore safe
on all platforms; the next hook fire simply runs the new version. The one live process during an
update is `apex update` itself, which the rename dance covers.

### 5. Integrity — SHA256SUMS, no signatures

`publish.ps1` computes SHA-256 for every zip (`Get-FileHash`) and writes `SHA256SUMS` in coreutils
format (`<lower-hex-hash>  <filename>`, two spaces), uploaded as a release asset. `apex update`
downloads `SHA256SUMS` from the SAME tag, finds its asset's line, and verifies the downloaded zip
(`crypto/sha256`) before extracting; mismatch or missing entry aborts with exit 1 and no files
touched. `install.ps1` gains the same verification (best-effort: warn-and-abort on mismatch,
proceed with a warning only if SHA256SUMS is absent — old releases don't have one).

Explicit punt: NO cryptographic signing (minisign/sigstore/GPG). There is no key infrastructure,
and any signing key would live in the same GitHub account that hosts the releases — it adds
ceremony, not a trust boundary. Checksums protect against corrupt/truncated downloads, which is
the realistic failure mode. Revisit if the project ever gets an out-of-band trust root.

### 6. Session-start nudge, opt-out, offline

`internal/hooks.SessionStart` additions — strictly ordered to keep the hook sub-second:

1. Best-effort `os.Remove` of `bin/apex.exe.old` (Windows leftover; no-op elsewhere).
2. Read the cache file (one small local read, no network EVER in the hook).
3. If cached `latest` > `version.Version` → append nudge:
   `Apex update available: v0.2.0 → v0.3.0 — run 'apex update'`.
4. If cache is missing or `checked_at` older than 24h → spawn `os.Executable()` with args
   `update check --quiet` fully detached (`exec.Cmd.Start()` then `Process.Release()`; stdio nil).
   The result lands in the cache for the NEXT session. The hook never waits.

Opt-out: `APEX_NO_UPDATE_CHECK=1` (any non-empty value) skips steps 3–4 entirely — no spawn, no
nudge. It does not disable explicit `apex update` / `apex update check`. Offline behavior: the
detached check times out in 5s, stamps `checked_at`, exits; the session saw nothing. A hook can
never fail or block a session regardless of network state (SessionStart already returns 0
unconditionally; that contract is preserved).

Rejected: networking in the hook with a short timeout — even 1s is a per-session tax and a tail
risk (DNS hangs beat timeouts to the punch on some stacks). Rejected: gh-style in-command
goroutine check — apex processes are millisecond-lived hook fires; there is no long-running
foreground command to piggyback on. Detached spawn + next-session nudge is the Homebrew/
update-notifier cadence model and fits the hook architecture exactly.

### 7. publish.ps1 additions (the design owns both sides)

- Read version const from `internal/version/version.go` (path change).
- Assert explicit `-Version` matches the const; die on mismatch.
- Emit + upload `SHA256SUMS` alongside the zips and `install.ps1`.

Asset naming contract (FROZEN — update, install.ps1, and publish.ps1 all depend on it):
`apex-claude-<os>-<arch>.zip` for `<os>-<arch>` in the Makefile matrix, `SHA256SUMS`,
`install.ps1`, under repo `FNGApex/apex-claude` (slug hardcoded in `internal/update`, matching
install.ps1:41).

## Risks / notes

- `os.UserCacheDir()` can fail (unset HOME/LocalAppData): treat as "no cache, no spawn" — degrade
  to silence, never to an error.
- Skills replace is destructive per `ax-*` dir (matches installer); user-local edits to installed
  artifacts are overwritten by design — installed `~/.claude` copies are not a source tree.
- Downgrade via `--to` is allowed (pin to any tagged release); no special handling.
- `uninstall.ps1` referenced by install.ps1 docs is not a release asset today; out of scope here.
