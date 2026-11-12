---
checkout: /Users/alexanderstroev/Documents/GitHub/apex-claude
branch: master
head: 00e7aec3fb755566218c9d0e1ad5654c717c0c76
---
## Startup
invoke the Agent skill (reads its private folder and .claude/LIVE.md); apex list --agent main

## State
No item in progress. 006-010 closed: MIGRATION.md, design/README, protocol skills, apex-init and the main Agent's own
move to `.claude/apex/main/` (00e7aec), installed with make install. master is 5 commits ahead of fork/master, not
pushed. This handoff is uncommitted. `tmp/` is untracked.

## Next
Nothing queued. Candidates: run /apex-init in each adopted repo to move its Agents to private folders; collect
operator rules into Rules.md as they are decided.

## Open threads
- Per-ticket mode: `apex-init` and README layout tables show mono only; extend if an adopted repo turns it on.
- PowerShell installer edits are untested here (no pwsh); verify on Windows.
