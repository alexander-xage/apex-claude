---
checkout: /Users/alexanderstroev/Documents/GitHub/apex-claude
branch: fresh-slate
head: e8dd369ea2144aca0b090bd4c03a8b0caa39199a
---
## Startup
invoke the Agent skill (reads both LIVE.md files); apex list --agent main

## State
No item in progress. 004 closed: graphify removed as a dependency (verb, skill, repo helpers, install check, docs),
committed in `e8dd369`; installed with `make install`. Not pushed: `fresh-slate` is 1 commit ahead of `fork`. Role card edit and
this handoff are uncommitted. Adopted repos (ClaudeCodeBrowser, LLMTraining, saas-health-check, saic-tools,
TamperMonkey) had graphify stripped from role cards, handoffs and LIVE.md, uncommitted there; graph caches deleted in
every repo. `tmp/` is untracked.

## Next
Nothing queued. Release left: open the `feat!` PR for `fresh-slate` when the operator asks, merge, tag.

## Open threads
- Per-ticket mode: `apex-init` and README layout tables show mono only; extend if an adopted repo turns it on.
- PR for `fresh-slate`: open only when the operator says so; breaking merge, see `.claude/LIVE.md`.
- `apex-init` conformant signals: deliberately few; calibrate against the live LLMTraining migration, now done.
