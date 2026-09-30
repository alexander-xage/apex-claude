# backbone

## What it does
Go CLI binary named `apex` (version `0.3.0`, from `internal/version.Version`) implementing the deterministic safety and introspection layer for Apex Claude. Registered subcommands (per `./bin/apex help`): `signals`, `health`, `hooks`, `doctor`, `docs`, `followups`, `handoff`, `reminder`, `update`, `validate`, plus the built-in `version`/`help`.
Internal packages write project state to `.claude/project/` and surface nudges via the SessionStart hook; the model never touches this layer directly.
Module is `apexclaude` (go 1.26), built to `bin/apex` via `make build` (or `go build -trimpath -ldflags "-s -w" -o bin/apex ./cmd/apex`, the form CI uses); cross-compiled for darwin/arm64, darwin/amd64, linux/amd64, linux/arm64, windows/amd64 by `make release`.

## CLI code
- cmd/apex/main.go — entry point; handles `version`/`help`/`--version`/`-v`/`--help`/`-h` inline, prints `apex <version.Version>`; dispatches everything else through the `commands` registry
- cmd/apex/registry.go — `commands` map populated by each cmd_*.go `init()` function via `register(name, summary, run)`
- cmd/apex/cmd_signals.go — registers `signals` with sub-subcommands: scan, show, stale; delegates to internal/signals and internal/proj
- cmd/apex/cmd_health.go — registers `health` with sub-subcommands: show, set <0-100> [note]; delegates to internal/health and internal/proj
- cmd/apex/cmd_hooks.go — registers `hooks` with one sub-subcommand: session-start → hooks.SessionStart (no PreToolUse bash guard; Claude Code's own auto-mode owns command permissions)
- cmd/apex/cmd_doctor.go — registers `doctor`; delegates to doctor.Run
- cmd/apex/cmd_docs.go — registers `docs` with scan/stale/show; delegates to internal/docs
- cmd/apex/cmd_followups.go — registers `followups` with add/list/close/render/path; delegates to internal/followups
- cmd/apex/cmd_handoff.go — registers `handoff` with scan (read-only fact report)/status/archive; delegates to internal/handoff
- cmd/apex/cmd_reminder.go — registers `reminder` with add/list/show/rm/due; delegates to internal/reminder
- cmd/apex/cmd_update.go — registers `update`; `apex update check [--quiet]` refreshes the release cache and exits 0 (up to date)/1 (update available)/2 (check failed); `apex update [--to vX.Y.Z]` refuses on a dev/plugin layout (`layout.IsLooseInstall` false → exit 2), otherwise downloads, verifies, and applies via internal/update.Apply, warning (not failing) if hooks aren't wired post-update
- cmd/apex/cmd_validate.go — registers `validate` with artifacts/spec/all targets; delegates to internal/validate
- internal/signals/signals.go — Scan() fingerprints manifests + top-level non-dotted dirs via SHA-256 and writes .claude/project/deterministic-signals.md; Stale() exits 0 (fresh), 1 (stale/missing), 2 (error); Show() returns file contents
- internal/health/health.go — Show() returns score (-1 if unset) and file body; Set() writes score 0-100 with optional note to .claude/project/health.md using `apex-health-score: <N>` HTML comment as the machine-readable anchor
- internal/hooks/hooks.go — SessionStart() best-effort sweeps a stale Windows `apex.exe.old` (via update.RemoveStaleWindowsOld), checks the update cache and appends an "update available" nudge when a newer tag is cached (skippable via `APEX_NO_UPDATE_CHECK`), spawns a detached `apex update check --quiet` when the cache is stale, then checks signals staleness and due reminders; emits one additionalContext JSON block if any nudges fired; always returns 0 (never blocks session start)
- internal/doctor/doctor.go — Run() branches its checks by layout (dev/plugin vs. loose install via internal/layout): dev layout checks `.claude-plugin/plugin.json` + `hooks/hooks.json` validity; loose install checks `apexHooksWired` on settings.json instead. Both layouts check output-styles/ has .md, agents/ has .md, commands/ has .md, skills/ has a subdirectory containing SKILL.md (glob: skills/*/SKILL.md); reports signals freshness as info
- internal/proj/proj.go — Root() returns $APEX_REPO if set, else cwd; StateDir() returns <root>/.claude/project, creating it if absent
- internal/layout/layout.go — ArtifactRoot() resolves $CLAUDE_PLUGIN_ROOT, else infers root from the running binary's location (bin/apex → parent), else ~/.claude, else cwd; LooksLikeArtifactRoot() / IsLooseInstall() discriminate on presence of `.claude-plugin`; ApexHooksWired() reads settings.json (BOM-tolerant) and checks for an `apex hooks`/`apex.exe hooks` command
- internal/update/update.go — LatestTag() does a no-follow-redirect GET against GitHub's `/releases/latest` and parses the tag from the `Location` header; Compare() orders `vX.Y.Z` triples; Cache read/write at `os.UserCacheDir()/apex-claude/update-check.json` (blank path on a UserCacheDir error is a silent no-op); Stale() flags a cache unpopulated or older than the 24h TTL; Refresh() re-fetches regardless of TTL and always stamps `checked_at` even on fetch failure
- internal/update/apply.go — Apply(root, to) downloads the `apex-claude-<GOOS>-<GOARCH>.zip` asset + `SHA256SUMS` for the resolved tag, verifies the checksum, extracts via archive/zip with zip-slip guards, replaces artifacts with installer semantics, and swaps the binary — atomic rename on Unix, rename-aside/write/rollback dance on Windows (a running exe can't be opened for write there); on any failure before artifact writes begin, root is left untouched; RemoveStaleWindowsOld() is exported so both `apex update` and the SessionStart hook share one `.old` sweep
- internal/handoff/handoff.go — Scan() gathers deterministic State; Report() renders it as a stdout fact table; Status()/Archive() operate on an existing handoff.md; there is no Write()/writer — the MODEL authors .claude/project/handoff.md from Report()'s output
- go.mod — module apexclaude, go 1.26, no external dependencies
- Makefile — targets: build, test (go test ./...), fmt (gofmt -w cmd internal), vet, clean, release (cross-compile matrix), install (delegates to scripts/install.sh), uninstall (delegates to scripts/uninstall.sh)

## Coupling
- Changing the `.claude/project/deterministic-signals.md` format (written by internal/signals) requires updating Stale() fingerprint logic and any consumer that reads this file (e.g. the session-start hook and doctor info check)
- Changing the `apex-health-score: <N>` HTML comment pattern in internal/health requires updating any tooling or skill that parses health.md
- Adding a new subcommand requires a new cmd_*.go with an `init()` that registers into the `commands` map in registry.go
- internal/handoff has no writer: Scan() gathers State, Report() renders it as a stdout fact table, and the MODEL authors .claude/project/handoff.md. Status()/Archive() are the only binary operations on an existing doc
- `RepoSlug` in internal/update ("FNGApex/apex-claude") must equal `$Repo` in scripts/install.ps1 and `REPO` in scripts/install-release.sh — a drift breaks update/install URL resolution silently
- `internal/update.Apply`'s artifact-replace semantics (overwrite commands/agents md, wholesale-replace skills dirs) must stay in lockstep with what scripts/install.sh, install-release.sh, and install.ps1 do — a divergence means `apex update` and a fresh install produce different trees
- $APEX_REPO and $CLAUDE_PLUGIN_ROOT are the two env-var seams that let the CLI operate outside the canonical repo layout (CI, remote envs, tests); `APEX_UPDATE_BASE_URL` is the equivalent seam for internal/update's network calls (shared with install-release.sh and install.ps1 for local test servers)

## Conventions worth knowing
- Subcommands self-register via init() — adding a file is enough; no manual wiring in main.go
- SessionStart never blocks (always exits 0); it can only emit advisory context, and performs zero synchronous network I/O — the update check it spawns is detached and unwaited
- Stale() exit codes are semantic: 0 = fresh, 1 = stale or file missing, 2 = error — callers must distinguish 1 from 2
- Fingerprint covers manifest files (go.mod, package.json, Cargo.toml, etc.) hashed by name+size (not content) and non-dotted top-level dirs by name only; dotfiles, file content, and nested dirs are excluded by design
- doctor.Run() treats signals freshness as info, not a hard failure — all other checks are hard failures; the check set itself branches on layout (dev/plugin vs. loose)
- `apex update` refuses to run against a dev/plugin layout — it is a loose-install-only operation; `apex update check` is layout-agnostic
- Release tags are always `vMAJOR.MINOR.PATCH`; a malformed tag from any source (env override, `--to`, a fetched Location header) is rejected before it can reach a network call or comparison
- `.old` Windows binary swap residue is cleaned opportunistically in two places (apex update's own Apply, and the SessionStart hook) rather than being guaranteed cleanup — failure to remove it is silent by design
