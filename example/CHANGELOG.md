# Changelog

This file records what changed in each Apex Claude release, newest first. Versions match the git
tags on the repository, and `apex version` tells you which one you are running.

## v0.3.0 (Unreleased)

This is the first Apex release that can update itself. From here on, new releases arrive through
`apex update`. Getting onto v0.3.0 is the one exception: v0.2.0 has no update command, so you
re-run the installer once.

### Upgrading from v0.2.0

Re-run the installer you used originally. On Linux or macOS that is the `install-release.sh`
one-liner from the README, on Windows the `install.ps1` one-liner, and from a source checkout
`git pull && make install`. The installer copies the new artifacts, deletes the commands this
release dropped, and rewires the session hook. Restart Claude Code afterward. If you installed
through the `.claude-plugin/` marketplace route, keep using `claude plugin update`.

Three changes can break existing habits or scripts.

**The Bash guard is gone.** Apex no longer ships a `PreToolUse` hook: `apex hooks pre-bash` was
removed, and `apex hooks` accepts only `session-start`. Claude Code already decides which commands may run, and the extra
pattern-matching layer underneath it produced false blocks rather than real safety. You do not
need to edit `settings.json` by hand: every installer strips the old Apex `PreToolUse` entry so it
cannot point at a subcommand that no longer exists.

**`apex handoff scan` reports and nothing more.** It takes no mode argument and writes no file; it
prints one `key: value` row per fact (branch, HEAD, dirty and staged files, last commit, open
follow-ups, due reminders, health, signals staleness, active scratchpad brief). Writing
`.claude/project/handoff.md` is now the model's job inside `/ax-handoff`. A script that passed
`graceful` or `urgent` to `scan` gets a usage error.

**Nine slash commands were removed**, taking the set from 23 to 14. Most were thin wrappers around
something another command or Claude Code itself already does.

| Removed | Use instead |
|---------|-------------|
| `/ax-squash` | `/ax-merge --squash` |
| `/ax-push` | `/ax-ship --push` |
| `/ax-review-branch` | The review gate built into `/ax-merge` and `/ax-pr`, or Claude Code's `/code-review` for an ad-hoc look |
| `/ax-remind-me` | `/ax-follow-up remind <when> <what>`, or Claude Code's `/schedule` for a reminder that must fire at a set time |
| `/ax-setup` | The first run of `/ax-refresh-signals`, which proposes the `.gitignore` entries and `docs/` folders Apex expects |
| `/ax-documentation` (command) | Still works: the `ax-documentation` skill answers to `/ax-documentation` |
| `/ax-watch-ci`, `/ax-help`, `/ax-report-issue` | Claude Code covers these directly: `/loop` for watching, or simply asking |

### Self-updating installs

`apex update` downloads the newest release bundle for your platform, checks it against the
release's `SHA256SUMS` file, and replaces the binary and the commands, agents, skills, and output
style together, so the binary and the artifacts it expects never drift apart. `apex update check`
only reports (exit `1` when a newer release exists, `0` when up to date, `2` when the check
failed), and `apex update --to vX.Y.Z` installs a specific release. The command refuses to run in a
source checkout or a plugin directory; those upgrade with `git pull && make install` and
`claude plugin update` respectively.

You never have to remember to check. At the start of each Claude Code session, the hook reads a
cached record of the latest release and, if it is newer than what you run, adds a one-line note
suggesting `apex update`. The hook itself never touches the network. When the cache is missing or
more than 24 hours old, it starts a detached background process to refresh it and returns without
waiting, so session start stays fast. Updates are never applied on their own; Apex only tells you
one is available. Set `APEX_NO_UPDATE_CHECK=1` to turn off both the note and the background check.

Windows does not allow overwriting a running executable, so `apex update` renames the running
`apex.exe` to `apex.exe.old` and writes the new one in its place. If that write fails, the old
binary is renamed back. The next session start or `apex update` deletes the leftover `.old` file.

Both `apex update` and all three installers remove `ax-*` commands, agents, and skills that the new
release no longer ships. Without this, the nine removed commands would linger in your slash menu
forever. Files without the `ax-` prefix belong to you and are never touched.

### Safer installs

`install-release.sh` and `install.ps1` now fetch the release's `SHA256SUMS` before downloading the
bundle. If the bundle's hash does not match, or the file does not list the bundle at all, the
installer stops without installing anything. Releases published before checksums existed, such as
v0.2.0, have no `SHA256SUMS`; for those the installers print a warning and install unverified.
`apex update` is stricter and refuses such releases outright.

Windows PowerShell 5.1, the version built into Windows, had several problems that are now fixed:

- `install.ps1` failed to parse under 5.1 when run as a file, because it contained non-ASCII
  characters. The PowerShell scripts are now ASCII-only.
- Every 5.1 install wrote `settings.json` with a byte-order mark, which made `apex doctor` report
  the hooks as unwired. The installer now writes the file without one, and `apex doctor` and
  `apex update` accept an existing file that has one, so installs from older versions check out
  correctly too.
- Checksum verification hashes through .NET rather than `Get-FileHash`, which 5.1 could fail to
  find when launched from a PowerShell 7 session.

### Leaner command set

The 14 commands that remain cover the lifecycle and a few housekeeping verbs: `/ax-plan`,
`/ax-pressure-test`, `/ax-implement`, `/ax-diagnose`, `/ax-ship`, `/ax-pr`, `/ax-merge`,
`/ax-autopilot`, `/ax-handoff`, `/ax-resume`, `/ax-follow-up`, `/ax-refresh-signals`,
`/ax-git-cleanup`, and `/ax-improve`. Where each removed command went is listed under
[Upgrading from v0.2.0](#upgrading-from-v020).

### Session handoff

`/ax-handoff` now splits the work the same way the rest of Apex does. The binary gathers the
deterministic state and prints it; the model writes the whole handoff document from that report,
frontmatter included, adding the intent and next steps that only it knows. Before writing over an
existing, unconsumed handoff, the command runs `apex handoff archive` so nothing is silently lost.
Because `apex handoff scan` writes nothing, it is safe to run at any time, including while
`/ax-resume` is consuming a handoff.

### Fixes

- **Follow-up IDs are never reused.** Closing a follow-up used to free its ID, so the next new
  entry received the same number and `CLOSED.md` then pointed at the wrong item. New IDs now take
  the closed ledger into account.
- **`apex followups add` understands flags.** It accepts `--kind`, `--severity`, and `--origin`,
  and rejects any other argument that starts with `-`. Previously such arguments were stored as the
  entry's title or severity.
- **Session start reports a broken signals check.** If the staleness check itself fails, the
  session note says so instead of staying silent.
- **`apex doctor` is clearer about where things are.** In a dev checkout where `hooks/hooks.json`
  points at a binary that was never built, it fails with a message telling you to run
  `make build`. When the binary lives somewhere other than `~/.claude/bin` (for example
  `/usr/local/bin`), it falls back to checking `~/.claude`. Its `PATH` check now matches entries
  through symlinks and trailing slashes.
- **The test suite no longer spawns copies of itself.** One test reached the real background
  update check, which relaunched the test binary and reran the suite.

### For maintainers

The version now lives in one place, `internal/version/version.go`. `.claude-plugin/plugin.json`
and `.claude-plugin/marketplace.json` must carry the same version, and a test fails if they do not.

`scripts/publish.sh` and `scripts/publish.ps1` both read the version from that file and refuse an
explicit `--version` (or `-Version`) that does not match it, which catches a forgotten bump. Both
write a `SHA256SUMS` file covering every bundle and upload it alongside the bundles; the installers
and `apex update` depend on it.

CI (`.github/workflows/ci.yml`) now runs on Ubuntu, Windows, and macOS, so the Windows-only update
code and the darwin builds are actually exercised. It runs only for release-candidate tags matching
`v*-rc*` (for example `v0.3.0-rc.1`) or when started manually from the Actions tab; ordinary pushes
and pull requests do not trigger it. The release flow is: push a release-candidate tag, wait for CI
to pass on all three systems, then publish.
