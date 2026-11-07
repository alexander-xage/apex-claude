---
checkout: /Users/alexanderstroev/Documents/GitHub/apex-claude
branch: fresh-slate
head: a948cae1c0ec7fed529fe13e769d28b599c4f817
---
## Startup
invoke the Agent skill (reads both LIVE.md files); apex list --agent main

## State
No item in progress. 004 closed: graphify removed as a dependency (verb, skill, repo helpers, install check, docs),
committed in `e8dd369`, role card in `a948cae`; installed at `a948cae`. Not pushed: `fresh-slate` is ahead of `fork`. Adopted repos (ClaudeCodeBrowser, LLMTraining, saas-health-check, saic-tools,
TamperMonkey) had graphify stripped from role cards, handoffs and LIVE.md, uncommitted there (operator may ask to
commit); graph caches deleted in
every repo. `tmp/` is untracked.

## Next
Nothing queued. Release left: open the `feat!` PR for `fresh-slate` when the operator asks, merge, tag.

## Open threads
- Per-ticket mode: `apex-init` and README layout tables show mono only; extend if an adopted repo turns it on.
- PR for `fresh-slate`: open only when the operator says so; breaking merge, see `.claude/LIVE.md`.
- `apex-init` conformant signals: deliberately few; calibrate against the live LLMTraining migration, now done.
