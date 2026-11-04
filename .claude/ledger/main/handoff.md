---
checkout: /Users/alexanderstroev/Documents/GitHub/apex-claude
branch: fresh-slate
head: 3d518751bc10cb011d860c7f0f5e8b5dcb701183
---
## Startup
invoke the Agent skill (reads both LIVE.md files); apex graph update .; apex list --agent main

## State
No item in progress. 003 closed: the `pr-review` security lens is written and runs only on `mythos`, committed in
`3d51875`. This handoff is uncommitted and rides in a later commit.

`fresh-slate` has unpushed commits on top of `fork`; no PR is open. `tmp/` (dry-run reference
material in `tmp/dryrun/`) is untracked.

| Piece | State |
|---|---|
| `apex` binary (`cmd/`, `internal/`) | done, tested, reviewed |
| Five protocol skills (`skills/`) | done, reviewed, exercised end to end in scratch repos |
| `pr-review` skill | used on a real PR by the operator; works; installed. Security lens on `mythos`: dry-run only |
| `Apex` output style | installed; not yet selected or seen in the `/output-style` picker |
| Named graphs (`apex graph --name`) | done, tested against real graphify, installed |
| Cross-Agent communication rules | written; not yet exercised by live Agents |
| Per-ticket ledger mode | done, tested, installed; never used by a live Agent |
| Install | run on this Mac at `3d51875`; live after a Claude Code restart |
| LLMTraining migration | dry run on a snapshot passed; the live repo was never touched |

## Next
Run `/pr-review <pr> security` on the work account (Mythos access): confirm alias `mythos` resolves, and that a
missing model makes reviewers return nothing (the skip) rather than abort the workflow.

## Open threads
- Per-ticket mode: `apex-init` and README layout tables show mono only; extend if an adopted repo turns it on.
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
