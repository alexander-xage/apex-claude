---
checkout: /Users/alexanderstroev/Documents/GitHub/apex-claude
branch: fresh-slate
head: 3d518751bc10cb011d860c7f0f5e8b5dcb701183
---
## Startup
invoke the Agent skill (reads both LIVE.md files); apex graph update .; apex list --agent main

## State
No item in progress. 003 closed: the `pr-review` security lens is written and runs only on `mythos`, committed in
`3d51875`. Committed and pushed to `fork` at `72d8dfa`; this handoff edit is uncommitted.

`fresh-slate` matches `fork`; no PR is open. `tmp/` (dry-run reference
material in `tmp/dryrun/`) is untracked.

| Piece | State |
|---|---|
| `apex` binary (`cmd/`, `internal/`) | done, tested, reviewed |
| Five protocol skills (`skills/`) | done, reviewed, exercised end to end in scratch repos |
| `pr-review` skill | used on a real PR by the operator; works; installed. Security lens on `mythos`: run live by the operator, works |
| `Apex` output style | installed; not yet selected or seen in the `/output-style` picker |
| Named graphs (`apex graph --name`) | done, tested against real graphify, installed |
| Cross-Agent communication rules | written; exercised by live Agents, works |
| Per-ticket ledger mode | done, tested, installed; in active use |
| Install | run on this Mac at `3d51875`; live after a Claude Code restart |
| LLMTraining migration | done live by the operator |

## Next
Nothing queued. Release left: open the `feat!` PR for `fresh-slate` when the operator asks, merge, tag.

## Open threads
- Per-ticket mode: `apex-init` and README layout tables show mono only; extend if an adopted repo turns it on.
- Named graph in LLMTraining: the Agent there asked for a second graph; tell it to run
  `apex graph --name <n> update <path>` and list it in `.claude/LIVE.md`.

- PR for `fresh-slate`: open only when the operator says so; breaking merge, see `.claude/LIVE.md`.
- `apex-init` conformant signals: deliberately few; calibrate against the live LLMTraining migration, now done.
- graphify internal file: exclusion relies on graphify 0.9.x's `.graphify_build.json`; recheck after a graphify
  upgrade (marked `ponytail:` in `cmd/apex/graph.go`).
