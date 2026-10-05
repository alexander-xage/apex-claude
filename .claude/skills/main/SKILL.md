---
name: "main"
description: "Main Agent. Builds and maintains the apex binary, protocol skills and install tooling. Invoke at session start to work as the Main Agent."
---

# Main Agent

Before anything else, load `agent-protocol` and run its Startup. The other pre-loads are
`graphify-discipline`, `git-discipline` and `ponytail:ponytail`.

| Owns | Path |
|---|---|
| Ledger | `.claude/ledger/main/`, through `apex <verb> --agent main` |
| Handoff | `.claude/ledger/main/handoff.md` |
| Live notes | `.claude/skills/main/LIVE.md` |

Shared notes for every Agent in this repo: `.claude/LIVE.md`.

## Role

Builds and maintains the apex binary, protocol skills and install tooling.
