---
name: ax-review
description: >
  Compressed code-review comments. Cuts noise while preserving actionable signal. One line per
  finding: location + problem (flag-only — names the problem, not the fix). CONFIDENCE 0–100
  replaces a pass/fail verdict; findings feed repo health. Four severity tiers 🟥🟧🟨🟦. Use
  when the user says "review this", "code review", "review the diff", or invokes /ax-review.
allowed-tools: Bash(git status*), Bash(git diff*), Bash(git log*), Read, Grep
---

<trigger>
"review this PR", "code review", "review the diff/branch", /ax-review, reviewing a change.
</trigger>

## Live context
- Diff: !`git diff --stat`
- Branch: !`git log --oneline -5`

## Rules
1. One line per finding: `path:line: <emoji> severity: problem.`
2. Severity tiers: 🟥 blocker · 🟧 risk · 🟨 nit · 🟦 uncertain (needs orchestrator judgment).
3. **Flag-only** — name the problem and where. Do NOT prescribe the fix; the orchestrator owns fix-finding.
4. No praise, no restating the diff, no scope creep. Signal only.
5. Check: correctness, security (secrets, injection, unsafe input), scope creep, missing tests for new behavior.

## Output
```
path:line: 🟥 blocker: <problem>.
path:line: 🟨 nit: <problem>.

Totals: <n blockers, n risks, n nits, n uncertain>
CONFIDENCE: <0–100>
```
CONFIDENCE = how sure the change is correct + complete + in-scope. Blockers drag it down; clean + verified pushes it up; genuine uncertainty lowers it. The orchestrator aggregates CONFIDENCE into repo health (`apex health`) and decides proceed/fix/block.
