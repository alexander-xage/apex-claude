---
name: "{{agent}}"
description: "{{Agent}} Agent. {{one-line role}} Invoke at session start to work as the {{Agent}} Agent."
---

# {{Agent}} Agent

Before anything else, load `agent-protocol` and run its Startup. The other pre-loads are
`graphify-discipline`, `git-discipline` and `ponytail:ponytail`.

| Owns | Path |
|---|---|
| Ledger | `.claude/ledger/{{agent}}/`, through `apex <verb> --agent {{agent}}` |
| Handoff | `.claude/ledger/{{agent}}/handoff.md` |
| Live notes | `.claude/skills/{{agent}}/LIVE.md` |

Shared notes for every Agent in this repo: `.claude/LIVE.md`.

## Role

{{role}}
