# Auto-update — spec

Contract for the Apex Claude auto-update mechanism. Design + rationale: `docs/design/auto-update.md`.

Scope: pure-Go (stdlib only) update pipeline in the `apex` binary — version embedding, cached
release check, session-start nudge, `apex update` self-update of binary + artifacts in lockstep,
checksum verification — plus the publish- and install-side changes that feed and consume it. No LLM anywhere in the path.

## Fixed contracts

- Repo slug: `FNGApex/apex-claude` (const in `internal/update`, must equal install.ps1's `$Repo`).
- Release assets per tag: `apex-claude-<os>-<arch>.zip` for each Makefile `RELEASE_TARGETS` entry,
  `SHA256SUMS`, `install.ps1`. Bundle layout: `apex(.exe)`, `commands/ax-*.md`, `agents/ax-*.md`,
  `skills/ax-*/…`, `output-styles/apex.md`.
- `SHA256SUMS` format: one line per zip, `<lower-hex-sha256>  <filename>` (two spaces, coreutils
  `sha256sum` compatible).
- Latest-tag discovery: `GET https://github.com/FNGApex/apex-claude/releases/latest` without
  following redirects; tag = last path segment of the `Location` header. Timeout 5s.
- Version format: `vMAJOR.MINOR.PATCH`; compare as int triple. Non-matching tags are ignored.
- Cache file: `<os.UserCacheDir()>/apex-claude/update-check.json`, schema
  `{"checked_at":"<RFC3339>","latest":"vX.Y.Z"}`. TTL 24h. Failed checks stamp `checked_at` too.
- Opt-out env var: `APEX_NO_UPDATE_CHECK` (non-empty disables the hook-side check + nudge only).
- No cryptographic signatures — checksums only (explicit punt, see design §5).
- Checksum consumers (`apex update`, `install.ps1`, `install-release.sh`) fetch `SHA256SUMS` from
  the same version as the bundle. HTTP 404 on `SHA256SUMS` = pre-checksum release. `apex update`
  refuses it (exit 1); the installers warn and install unverified. Any other fetch failure is fatal
  everywhere. A missing line for the asset, or a hash mismatch, is fatal everywhere.
- Windows PowerShell 5.1 compatibility: `scripts/*.ps1` are ASCII-only (5.1 reads BOM-less UTF-8
  as Windows-1252, and a mis-decoded em-dash yields a curly quote that breaks parsing);
  `settings.json` is written UTF-8 **without** BOM; `layout.ApexHooksWired` strips a leading BOM so
  files written by older installers still read as wired.

## Checkpoints

Each row is independently implementable and verifiable.

| # | Checkpoint | Contract | Verify |
|---|------------|----------|--------|
| 1 | Version package | New `internal/version/version.go`: `package version; const Version = "X.Y.Z"` (single source of truth; `.claude-plugin/plugin.json` and `marketplace.json` must carry the same version — enforced by `TestPluginManifestsMatchVersion`). `cmd/apex/main.go` drops its local const and prints `version.Version`. `scripts/publish.ps1` and `scripts/publish.sh` both read the const from the new path; an explicit `-Version` / `--version` that != `v` + const dies. | `apex version` prints `apex X.Y.Z`; `go test ./...` green; `publish.ps1 -DryRun -Version v9.9.9` and `publish.sh --dry-run --version v9.9.9` fail with a mismatch error; without a version both resolve `v` + const. |
| 2 | Layout package extraction | `artifactRoot()`, `looksLikeArtifactRoot()`, `isLooseInstall()`, `apexHooksWired()` move from `internal/doctor` to new `internal/layout` (exported). Doctor imports layout; doctor output and exit codes unchanged. | Existing doctor tests pass unchanged (relocated); `apex doctor` output identical before/after on both dev and loose layouts. |
| 3 | Release check + cache | `internal/update`: `LatestTag()` does the no-redirect GET (5s timeout) and parses the tag; `Compare(a, b)` orders `vX.Y.Z` triples; cache read/write at the fixed path via `os.UserCacheDir()` (dir created on demand; a UserCacheDir error degrades to no-cache, never an error). Failed network check stamps `checked_at`, preserves prior `latest`. Unit tests use `httptest` for the redirect and `t.TempDir` via an injectable cache path. | `go test ./internal/update/...` green: 302 parse, malformed tag ignored, compare table, TTL expiry, failure stamping. |
| 4 | `apex update check` | New `cmd/apex/cmd_update.go` registers `update`. `apex update check` refreshes the cache and prints `apex v0.2.0 — latest v0.3.0 (update available)` or `apex v0.2.0 — up to date`. Exit 0 = up to date, 1 = update available, 2 = check failed. `--quiet` suppresses stdout (same exit codes) — this is the hook-spawned form. Runs in any layout (check is layout-agnostic). | Run against a stubbed base URL (env or test seam `APEX_UPDATE_BASE_URL` override): all three exit codes reproducible; cache file written with fresh `checked_at`. |
| 5 | Session-start nudge | `internal/hooks.SessionStart` gains, before existing nudges: (a) best-effort remove of `<root>/bin/apex.exe.old`; (b) cache read — if cached `latest` > `version.Version`, append nudge `Apex update available: v<cur> → <latest> — run 'apex update'`; (c) if cache missing/older than 24h, spawn `os.Executable()` `update check --quiet` detached (Start + Release, nil stdio) and do not wait. `APEX_NO_UPDATE_CHECK` non-empty skips (b) and (c). Hook still returns 0 always, performs zero network I/O itself, and stays sub-second. | Unit: seeded cache with newer version → nudge line present in JSON `additionalContext`; equal/older version → absent; env var set → absent and no spawn. Manual: `time apex hooks session-start` < 100ms with no cache and no network. |
| 6 | SHA256SUMS on publish | Both publish tools write `dist/SHA256SUMS` in the fixed format for every bundle zip and include it in `gh release create`/`upload` asset lists: `publish.ps1` via `Get-FileHash` (LF, no BOM), `publish.sh` via `sha256sum` or `shasum -a 256`. | `publish.ps1 -DryRun` / `publish.sh --dry-run` each produce `dist/SHA256SUMS` with 5 lines; `cd dist && sha256sum -c SHA256SUMS` passes. |
| 7 | `apex update` happy path (Unix semantics) | `apex update [--to vX.Y.Z]`: refuse unless `layout.isLooseInstall(root)` (message: `dev layout — use 'git pull && make install'`, exit 2). Resolve tag (latest or `--to`); if tag == current and no `--to`, print up-to-date, exit 0. Download zip for `runtime.GOOS-GOARCH` + `SHA256SUMS` from that tag; verify hash (mismatch/missing line → exit 1, nothing touched); extract via `archive/zip` (zip-slip guarded: reject entries escaping the temp dir) to a temp dir; copy artifacts into root with installer semantics (overwrite `commands/ax-*.md`, `agents/ax-*.md`, `output-styles/apex.md`; wholesale-replace each `skills/ax-*` dir), then prune `ax-*` commands/agents/skills the bundle no longer ships (non-`ax-` files are the user's and untouched; a kind the bundle ships none of is never pruned); binary: write `<root>/bin/.apex.new`, chmod 0755, `os.Rename` over `apex`. Post-update: warn (not fail) if `layout.apexHooksWired(root)` is false; rewrite cache as up-to-date; print `updated v<old> → <new>`. Exit 0 success, 1 verify/download failure, 2 usage/layout. | Integration test against `httptest` server serving a crafted bundle + SHA256SUMS into a temp loose root: artifacts and binary replaced, an `ax-*` command absent from the bundle pruned while a non-`ax-` command survives, exit 0; corrupted hash → exit 1 and root untouched; dev-layout root → exit 2. Manual: `apex update --to v0.2.0` against the real release on a scratch `CLAUDE_CONFIG_DIR`. |
| 8 | Windows binary swap | On `runtime.GOOS == "windows"`: rename `apex.exe → apex.exe.old`, write new `apex.exe` (from the verified extract), on write failure rename `.old` back (rollback, exit 1). `.old` cleanup is best-effort at the start of `apex update` and in checkpoint 5(a); removal failure is silent. | Windows test (or CI windows runner): run installed `apex.exe update --to <same-tag>` — succeeds while running, `apex.exe.old` exists after, next `apex hooks session-start` removes it. Simulated write-failure test asserts rollback restores the original exe. |
| 9 | Installer checksum verify | `scripts/install.ps1` and `scripts/install-release.sh` fetch `SHA256SUMS` for the resolved version **before** downloading the bundle (after a multi-MB download, PS 5.1 reused a server-closed pooled connection); `install.ps1` retries that fetch once on a transport error. Verify the zip before extracting — `install.ps1` hashes via .NET `SHA256`, not `Get-FileHash` (unresolvable in 5.1 when a PowerShell 7 `PSModulePath` is inherited); `install-release.sh` via `sha256sum`/`shasum -a 256`. 404 → warn and proceed; other fetch failure, missing entry, or mismatch → die. Both honor `APEX_UPDATE_BASE_URL` as `<base>/<version>/<asset>` for tests. | Against a local mock release: valid → installs; tampered zip → exit 1, nothing installed; SHA256SUMS lacking the asset → exit 1; no SHA256SUMS → warns, installs. `install.ps1` under Windows PowerShell 5.1 via `-File` and via `irm \| iex`; `install-release.sh` via curl and via wget. Real `v0.2.0` (pre-checksum) → warns, installs, `apex version` runs. |
| 10 | Registry + docs sync | Root `CLAUDE.md` backbone registry line gains `update`; `apex help` lists `update` with summary (automatic via `register()`). README backbone block lists `apex update`, and its install section describes checksum verification and the update flow. `.claude/project/signals.md` build/test table row for the update check is added on the next signals refresh (not hand-edited here). | `apex help` shows `update`; `grep update CLAUDE.md` hits the backbone registry line; `apex validate` and `apex doctor` pass. |

Suggested build order: 1 → 2 → 3 → 4 → 6 → 7 → 8 → 5 → 9 → 10 (nudge last among Go work so it
can spawn a real, working `update check`).

## Non-goals

- No signature verification (checksums only — design §5).
- No auto-apply: updates are always user-initiated via `apex update`; the system only nudges.
- No settings.json mutation during update (hook paths are stable absolute paths).
- No prerelease channel; `releases/latest` semantics only, `--to` for pinning.
- No self-upgrade out of `v0.2.0` (pre-update-feature, no `SHA256SUMS`): those installs re-run the
  installer once. No upgrade of plugin-directory installs (`.claude-plugin/` present): refused as dev
  layout; they use `claude plugin update`.
- No changes to `scripts/install.sh`'s download path — it builds from source, so there is nothing to
  verify; source-checkout installs update via `git pull && make install`.

## Change log

- 2026-07-01 — Initial spec (ax-plan): version package, redirect-sniff check with 24h cache,
  detached background refresh + session-start nudge, `apex update` lockstep self-update with
  SHA256SUMS verification and Windows rename dance, publish.ps1 checksum emission.
- 2026-09-13 — CP5 landed (5b5a3a3). Its two pre-existing signals tests reached the real
  `spawnCheck`, which re-execs the test binary and re-runs the suite: a fork bomb. Fixed by `noSpawn`
  in those tests (75d437a). Rule: every test touching `SessionStart` substitutes `spawnCheck`.
- 2026-09-13 — CP9 + CP10 landed. Superseded: CP9 was `install.ps1` only, verifying via
  `Get-FileHash` after download. It now covers `install-release.sh` too, fetches `SHA256SUMS`
  before the bundle, and hashes via .NET. Superseded: CP6 was `publish.ps1` only; `publish.sh` now
  emits `SHA256SUMS` and reads the version const from `internal/version` (it had still been
  regexing `cmd/apex/main.go`, so a bare `publish.sh` failed after CP1). Added the Windows
  PowerShell 5.1 compatibility contract: the committed `install.ps1` did not parse under 5.1 via
  `-File`, and every 5.1 install wrote a BOM'd `settings.json` that `apex doctor` read as unwired.
- 2026-09-13 — Release readiness for v0.3.0 (the first release able to update itself). Version
  const and both plugin manifests bumped together; version-dependent tests now derive from
  `version.Version`. Verified the full user journey against two publish.sh-built release sets
  (v0.3.0, v0.3.1) behind a mock of the `releases/latest` redirect, on Linux (`install-release.sh`)
  and on Windows PowerShell 5.1 (`install.ps1`): install → first session primes the cache via the
  detached check → newer release published → after the TTL the next session refreshes → following
  session nudges → `apex update` swaps binary + artifacts (Windows: running `apex.exe` renamed to
  `.old`, swept by the next session start) → `doctor` passes → re-run reports up to date. Hook wall
  time 3–4ms Linux, 9–14ms Windows. CI matrix gains windows-latest and macos-latest so CP8's
  Windows-only paths and darwin run on every push.
- 2026-09-13 — Superseded: CP7 artifact semantics were overwrite-only, so an `ax-*` artifact dropped
  by a release lingered on every updated install. `apex update` and all three installers now prune
  `ax-*` commands/agents/skills the new bundle doesn't ship. Motivated by cutting commands 23 → 14;
  must ship in v0.3.0 because the *installed* binary performs the next update.
