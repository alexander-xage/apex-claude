# Shared live notes

Facts every Agent in this repo needs, kept once. Every Agent reads this after its own `LIVE.md`; any Agent may edit it.

- Design contract: `docs/design/agents.md`. It decides; code and skills follow it.
- `example/` is the old Apex (v0.3.0 plugin), reference only. Never edit or extend it.
- Checks: `go test -race ./...` for the binary; `make test` also dry-runs the `pr-review` workflow against stub agents.
- Install: `make install` builds `~/.local/bin/apex` and copies the skills (marked `.apex-installed`) and the `Apex`
  output style into `~/.claude/`; restart Claude Code after. `make uninstall` removes only what it marked.
- A skill changed under `skills/` is not live until `make install` runs again.
- Merging `fresh-slate` to `master` is breaking (removes the v0.3.0 plugin): mark that PR or merge `feat!`.
- Never search git-ignored files in adopted repos (LLMTraining `runs/` is 5.4 GB; a walk there tripped memguard).
