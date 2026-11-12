---
checkout: /Users/alexanderstroev/Documents/GitHub/apex-claude
branch: master
head: 5cdd2905052e0139c5208bdaa4f9f8f68377e40a
---
## Startup
invoke the Agent skill (reads both LIVE.md files); apex list --agent main

## State
No item in progress. fresh-slate merged into master as 5cdd290 (feat!: graphify removal is the breaking part) and
pushed to fork; fresh-slate deleted locally and on fork. master tracks origin (read-only): push to fork. LIVE.md and
this handoff are uncommitted. Adopted repos still hold uncommitted graphify removals. `tmp/` is untracked.

## Next
Nothing queued. Release left: tag the merge when the operator asks; getting master into origin (FNGApex) needs a PR
from fork, only on request.

## Open threads
- Per-ticket mode: `apex-init` and README layout tables show mono only; extend if an adopted repo turns it on.
- PowerShell installer edits are untested here (no pwsh); verify on Windows.
