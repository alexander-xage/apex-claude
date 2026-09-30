---
name: ax-commit
description: >
  Commit message generator. Conventional Commits format, subject <=50 chars, body only when
  the "why" isn't obvious from the diff. Use when the user says "write a commit", "commit
  message", "generate commit", or invokes /ax-commit. Auto-triggers when staging changes.
allowed-tools: Bash(git status*), Bash(git diff*), Bash(git log*)
---

<trigger>
"write a commit", "commit message", "generate commit", staging changes for commit.
</trigger>

## Live context
- Staged diff: !`git diff --staged --stat`
- Recent style: !`git log --oneline -5`

## Rules
1. Format: `type(scope): subject` — types: feat, fix, refactor, docs, test, chore, perf.
2. Subject: imperative mood, <=50 chars, no trailing period.
3. Body: only when the *why* isn't obvious from the code. Wrap at 72 cols. Explain intent and tradeoffs, not what the diff already shows.
4. One logical change per commit. If the diff spans unrelated concerns, say so and suggest a split.

## Example
Bad: `fix: fixed the thing and also updated some tests and refactored`
Good:
```
fix(auth): reject tokens past expiry

Clock-skew grace was applied twice, so tokens up to 2x the window
were accepted. Apply skew once at validation.
```
