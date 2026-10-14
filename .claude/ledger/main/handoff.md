---
checkout: /Users/alexanderstroev/Documents/GitHub/apex-claude
branch: fresh-slate
head: 700c3f5cab75862894f51ab72fceef0fef37a90a
---
## Startup
invoke the Agent skill (reads both LIVE.md files); apex graph update .; apex list --agent main

## State
No item in progress. 002 closed: per-ticket ledger mode (`per-ticket` marker in `ledger/<agent>/`), committed in
`700c3f5`. Ledger 002 and this handoff are uncommitted and ride in a later commit.

`fresh-slate` has unpushed commits on top of `fork`; no PR is open. `tmp/` (dry-run reference
material in `tmp/dryrun/`) is untracked.

| Piece | State |
|---|---|
| `apex` binary (`cmd/`, `internal/`) | done, tested, reviewed |
| Five protocol skills (`skills/`) | done, reviewed, exercised end to end in scratch repos |
| `pr-review` skill | used on a real PR by the operator; works; installed |
| `Apex` output style | installed; not yet selected or seen in the `/output-style` picker |
| Named graphs (`apex graph --name`) | done, tested against real graphify, installed |
| Cross-Agent communication rules | written; not yet exercised by live Agents |
| Per-ticket ledger mode | done, tested, installed; never used by a live Agent |
| Install | run on this Mac at `700c3f5`; live after a Claude Code restart |
| LLMTraining migration | dry run on a snapshot passed; the live repo was never touched |

## Next
Nothing queued. Candidates from Open threads: the LLMTraining migration (needs no active training run), or the
per-ticket rows for `apex-init` and README if an adopted repo turns the mode on.

## Open threads
- Per-ticket mode: `apex-init` and README layout tables show mono only; extend if an adopted repo turns it on.
- Security lens: `skills/pr-review/prompts/lens-security.md` is a stub, waiting on model access.
- Named graph in LLMTraining: the Agent there asked for a second graph; tell it to run
  `apex graph --name <n> update <path>` and list it in `.claude/LIVE.md`.
- LLMTraining migration: only when no training run is active. Confirm with its Dev Agent, have a clean `git status`,
  run `/apex-init` there (routes to migrate). Expected from the snapshot run: dev 55 items, research 21, web-dev 6;
  Xage stays in `xage-guard`; it asks separately before deleting the 24 project-memory files; 11 XageLedger files in
  `xage-guard` cite old numbers for the Xage Agent to fix.
- PR for `fresh-slate`: open only when the operator says so; breaking merge, see `.claude/LIVE.md`.
- `apex-init` conformant signals: deliberately few; calibrate after the live LLMTraining migration.
- graphify internal file: exclusion relies on graphify 0.9.x's `.graphify_build.json`; recheck after a graphify
  upgrade (marked `ponytail:` in `cmd/apex/graph.go`).
