---
checkout: /Users/alexanderstroev/Documents/GitHub/apex-claude
branch: fresh-slate
head: 63286fb04a89269a4caf76e01525f971a5867ba2
---
## Startup
invoke the Agent skill (reads both LIVE.md files); apex graph update .; apex list --agent main

## State
No item in progress; the ledger is empty. Carried from the root `HANDOFF.md` of 2026-10-01 by `apex-init` on
2026-10-05.

`fresh-slate` is 13 commits ahead of `master` and pushed to `fork`; no PR is open. `tmp/` (dry-run reference
material in `tmp/dryrun/`) is untracked.

| Piece | State |
|---|---|
| `apex` binary (`cmd/`, `internal/`) | done, tested, reviewed |
| Five protocol skills (`skills/`) | done, reviewed, exercised end to end in scratch repos |
| `pr-review` skill | workflow script passes a dry run against stub agents; never run on a real PR; not installed |
| `Apex` output style | installed; not yet selected or seen in the `/output-style` picker |
| Named graphs (`apex graph --name`) | done, tested against real graphify, installed |
| Cross-Agent communication rules | written; not yet exercised by live Agents |
| Install | run on this Mac at `956c5f0`; `pr-review` (`63286fb`) needs another `make install` |
| LLMTraining migration | dry run on a snapshot passed; the live repo was never touched |

## Next
Install `pr-review` (`make install`, restart Claude Code), then try `/pr-review <pr>` on a real PR: check the
`gh pr checkout --worktree` path, the inline-comment line mapping and the posted review format. It is the only piece
never run for real.

## Open threads
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
