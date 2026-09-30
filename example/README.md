# Apex Claude

Apex Claude is a Claude Code workflow layer that encodes a full coding loop: plan, build with
tests, review, ship, and improve. It pairs a set of agents, skills, slash commands, an output
style, and a hook with a small Go backbone binary (`apex`) that does the deterministic work. It
installs as loose `~/.claude/` artifacts (bare `/ax-*` commands); a plugin manifest is also
included for marketplace installs.

The guiding split runs through everything here. The binary owns determinism: scanning the repo,
linting artifacts, gating on staleness, detecting state, scoring health. The model owns judgment:
classification, drafting, design, and the decision about whether a change is actually correct.
When code can answer a question, code answers it. The model is reserved for the calls that need
reasoning.

## How the layer is organized

| Surface | Mechanism | Job |
|---------|-----------|-----|
| `.claude-plugin/plugin.json` | Plugin manifest | Names and versions the bundle |
| `output-styles/protocol.md` | Output style (Apex) | How Claude talks: article-dropping, signal-first, tiered |
| `agents/ax-*.md` | Subagents | Locate, build, review, reason, plan, debug, write, run |
| `skills/ax-*/SKILL.md` | Skills | Test-first, commit, verify, review, documentation, prose voice |
| `commands/ax-*.md` | Slash commands | The lifecycle verbs, from setup through ship and improve |
| `hooks/hooks.json` | Hooks | SessionStart context, wired to the binary |
| `cmd/apex/` to `bin/apex` | Go CLI | The deterministic backbone |

## The lifecycle

The commands follow one loop. Each verb is small and self-describing in the slash listing.

1. **Plan.** `/ax-plan` researches the task across the codebase and online, then writes a design
   doc and a checkpoint spec. Trivial work gets an inline spec instead. `/ax-pressure-test`
   challenges a design before you commit to it.
2. **Implement.** `/ax-implement` runs the implement and review loop: it briefs a fresh-context
   builder, gates each iteration on the reviewer's confidence score, and commits per green pass.
   `/ax-diagnose` covers failure-driven work, starting from a broken test or a symptom.
3. **Ship.** Pick the verb that matches how far you want to go. `/ax-ship` reviews the change and
   commits it, and `--push` sends it to the remote as well. `/ax-pr` pushes the branch and opens a
   pull request. `/ax-merge` lands the branch on base, and `--squash` collapses it into a single
   commit on the way. Branch-landing verbs run the reviewer first if the work never went through
   `/ax-ship`.
4. **Sync docs.** The `ax-documentation` skill keeps the human-facing surfaces current. Ship verbs
   run its maintenance mode automatically; invoke `/ax-documentation` directly for a full pass.
5. **Improve.** `/ax-improve` mines a session for friction and turns it into concrete fixes.

A few housekeeping verbs sit outside the loop. `/ax-follow-up` works the ledger of loose ends and
files reminders that surface at the next session start. `/ax-refresh-signals` regenerates the
project map, and on its first run in a new repo it also proposes the `.gitignore` entries and
`docs/` folders Apex expects. `/ax-git-cleanup` finds stale branches and worktrees and removes the
ones you confirm. Everything else you might reach for — reviewing a branch, watching CI, opening an
issue, scheduling a timed reminder — is already covered by Claude Code itself (`/code-review`,
`/loop`, `/schedule`) or by simply asking.

`/ax-autopilot` runs the whole loop hands-off, with one human decision: how to merge.

When a session has to stop mid-stream, `/ax-handoff` captures where you are — the binary scans the
deterministic state (branch, HEAD, dirty files, open follow-ups, health) and you compose the intent
narrative on top. `/ax-resume` picks it back up in a fresh session: it reads the staleness verdict
from the recorded commit, reconciles the plan against current reality, and archives the handoff once
you confirm the resume point.

## The orchestrator owns the truth

Subagents never claim a task is done. They gather evidence and report it up. The orchestrator,
the main loop you talk to, owns final verification, finds the fix when something is wrong, and
makes every gate decision.

Reviewers and documentation checks reflect this. Instead of a binary pass or fail verdict, they
emit a confidence score from 0 to 100 and flag problems without prescribing the fix. Findings
carry four severity tiers: blocker, risk, nit, and uncertain. The orchestrator aggregates those
scores into a persisted repo health signal that `apex health` reads and `apex doctor` checks.

## Two voices

Claude's replies in the terminal use the Apex output style: article-dropping, structural forms
by default, with intensity tiers (lite, full, ultra) you can switch mid-session. Security
warnings and irreversible-action confirmations always revert to full prose so nothing critical
gets compressed away.

Enduring narrative documentation, like this README and anything under `docs/guides/`, uses the
opposite voice through the `ax-explainer` module: clear, expansive, and detailed. Everything
else, including specs, design docs, and signals files, stays terse and technical.

## The backbone

The deterministic work lives in a single static Go binary built from the standard library with
no external dependencies. It starts in milliseconds, which is what lets it sit inside a hook,
and it returns real exit codes that commands branch on.

```bash
make build      # -> bin/apex (static binary)
make test       # go test ./...
make vet        # go vet ./...
make fmt        # gofmt -w cmd internal
make release    # cross-compile bin/<os>-<arch>/apex for mac, linux, windows
```

The subcommand groups:

```bash
apex signals scan|show|stale     # deterministic project map + staleness gate
apex health show|set             # repo health/integrity score
apex followups list|add|close|render|path
apex reminder add|list|show|rm|due
apex hooks session-start
apex doctor                      # integrity check on the plugin layout + project state
apex validate                    # lint artifacts and specs (exit 1 on issues)
apex docs scan|stale             # documentation-surface cache and staleness gate
apex update [check] [--to vX.Y.Z] # check for a newer release, or apply one
apex version
```

Apex ships no Bash guard. Earlier versions wired a `PreToolUse` hook that pattern-matched
destructive commands, but Claude Code's own auto-mode already arbitrates what runs, and a second
regex layer underneath it bought false blocks rather than safety. Command permissions belong to the
harness; the backbone sticks to scanning, gating, and state.

> The compiled binary is gitignored. Run `make build` after cloning so the hook has something to
> call, and ship per-platform binaries through `make release` plus GitHub Releases.

## Installing and adopting

Install Apex as loose user-level artifacts with one command:

```bash
make install      # or: scripts/install.sh
```

This builds the `apex` binary, copies the commands, agents, skills, and output style into
`~/.claude/`, drops the binary at `~/.claude/bin/apex`, and wires the `SessionStart` hook into
`~/.claude/settings.json` without disturbing any of your other settings. Upgrading from a version
that installed the `PreToolUse` guard? The installer strips that stale entry for you.
Restart Claude Code afterward, then activate the voice with `/output-style Apex`.

That source build needs a Go toolchain. On a Linux or macOS box without one, install from a
published release instead with a single line — it downloads the prebuilt `apex` binary for your
platform rather than compiling it:

```bash
curl -fsSL https://github.com/FNGApex/apex-claude/releases/latest/download/install-release.sh | bash
```

`install-release.sh` detects your OS and architecture (Linux and macOS, amd64 and arm64), downloads
the matching `apex-claude-<os>-<arch>.zip` bundle, copies the same artifacts into `~/.claude/`, and
wires the hooks exactly as the source installer does — it just skips the build, so it needs no `go`
or `make`, only `curl` (or `wget`) and the `python3` that ships on essentially every Linux and macOS
system. Pin a specific release with `APEX_VERSION=v0.2.0` and redirect the install root with
`CLAUDE_CONFIG_DIR`. Remove it later with `scripts/uninstall.sh`, the same as a source install.

On native Windows, where there is no bash, `python3`, or build toolchain to lean on, install from a
published release with a single PowerShell line:

```powershell
irm https://github.com/FNGApex/apex-claude/releases/latest/download/install.ps1 | iex
```

`install.ps1` downloads the prebuilt `apex.exe` bundle, copies the same artifacts into
`~/.claude\`, and wires the hooks using PowerShell's native JSON handling — no dependencies beyond
the Windows PowerShell that ships with the OS (5.1 and 7+ both work). Remove it later with
`scripts/uninstall.ps1`. The release bundles themselves are cut by the maintainer with one of two
mirrored publish tools — `scripts/publish.sh` from Linux or macOS, `scripts/publish.ps1` from
Windows. Both cross-compile the same matrix, zip the binary together with the artifacts per
platform, write a `SHA256SUMS` file covering every bundle, and upload the bundles, the checksums,
and both one-line installers to a GitHub Release via `gh`, so whichever platform the maintainer
ships from, all three install paths stay in sync. Before publishing, push a release-candidate tag such as
`v0.3.0-rc.1`: that is what runs CI, on Linux, Windows and macOS. Ordinary pushes and pull requests
don't run it, so a green release-candidate run is the check that a release is safe to cut. The Actions
tab's "Run workflow" button runs the same checks on demand.

Both prebuilt installers verify what they download. Each one fetches the release's `SHA256SUMS`
first and refuses to install a bundle whose hash does not match, or one the file does not list at
all. Releases cut before checksums existed, such as `v0.2.0`, have no `SHA256SUMS`; for those the
installers print a warning and install unverified rather than refusing outright.

Once installed, a prebuilt install keeps itself current without re-running the installer. At the
start of each Claude Code session the hook checks a cached record of the latest release — it never
touches the network itself — and, when a newer version exists, adds a one-line note to the session
suggesting `apex update`. Refreshing that cache happens in a detached background process at most
once a day, so the hook stays fast. Running `apex update` downloads the new bundle, verifies it
against `SHA256SUMS`, and replaces the binary and the artifacts together, so the two never drift
apart; `apex update check` only reports, and `--to v1.2.3` pins a specific release. Set
`APEX_NO_UPDATE_CHECK=1` to silence the session note and the background check. Updates are always
something you run — Apex only ever tells you one is available.

Coming from `v0.2.0`? That release predates `apex update` and has no checksums, so it cannot upgrade
itself. Re-run the one-line installer for your platform once to move to the current release; every
update after that goes through `apex update`. The same holds for an install done through the
`.claude-plugin/` marketplace route — `apex update` refuses to touch a plugin directory, so plugin
installs keep upgrading through `claude plugin update`. [`CHANGELOG.md`](CHANGELOG.md) lists everything that changed in
each release, including the commands that were removed and what replaced them.

Apex installs as loose files rather than as a Claude Code plugin on purpose. Plugin commands are
namespaced by the harness, so a plugin install surfaces them as `/apex-claude:ax-plan`; loose
user-level artifacts are not namespaced, so the same command is just `/ax-plan`. The cost of the
loose model is that there is no plugin enable/disable lifecycle: the installers own installation,
`apex update` owns upgrades of a prebuilt install, and `scripts/uninstall.sh` (or `make uninstall`)
owns removal. `apex update` deliberately refuses to run against a source checkout — there,
`git pull && make install` is the upgrade path.

The repo still carries a `.claude-plugin/` manifest, so you can register it as a marketplace and
`claude plugin install apex-claude@apex-claude` instead if you prefer the plugin lifecycle and do
not mind the namespace prefix.

The Apex spine, which holds the principles, the determinism boundary, the lifecycle, and the
artifact registries, lives in this repo's `CLAUDE.md` and is not installed automatically. To adopt
Apex's conventions in another project, copy the spine sections from `CLAUDE.md` into that project's
`.claude/CLAUDE.md`, or into `~/.claude/CLAUDE.md` for user-wide use. In this repo the spine
auto-loads, so Apex builds itself under its own rules.
