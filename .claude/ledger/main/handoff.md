---
checkout: /Users/alexanderstroev/Documents/GitHub/apex-claude
branch: master
head: fdf8205012a604b55f8b7c7a022d4d02bdaa7084
---
## Startup
invoke the Agent skill (reads both LIVE.md files); apex list --agent main

## State
No item in progress. 006 closed: MIGRATION.md (per-Agent private `.claude/apex/<agent>/` with LIVE.md, Rules.md,
reminder skills) committed in fdf8205, not pushed. master tracks origin (read-only): push to fork. This handoff is
uncommitted. Adopted repos still hold uncommitted graphify removals. `tmp/` is untracked.

## Next
The protocol changes listed in MIGRATION.md (design doc, agent-protocol, sign-off, git-discipline, apex-init, README),
then migrate this repo's main Agent. Not filed yet: file when the operator says go.

## Open threads
- Per-ticket mode: `apex-init` and README layout tables show mono only; extend if an adopted repo turns it on.
- PowerShell installer edits are untested here (no pwsh); verify on Windows.
