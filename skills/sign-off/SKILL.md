---
name: sign-off
description: End an Agent session cleanly. Logs or closes the in-progress ledger item, updates LIVE.md, writes the handoff, cleans up what the session started, commits the session's .claude/ changes, and reports unpushed commits. Use on /apex:sign-off or when the operator says "sign off".
---

# Sign off

The operator's notice that the session ends. Run the six steps in order, as the Agent this session was started as.
Every `apex` call passes `--agent <agent>`. Ledger, handoff, `LIVE.md` and `<repo>` come from `apex:agent-protocol`.

## 1. In-progress item

Find it with `apex list --status in-progress --agent <agent>`. None: skip to step 2.

| Item state | Action |
|---|---|
| Goal met and reviewed | `apex:agent-protocol` Work an item steps 7-9 |
| Not done | append one `## Log` line saying where it stopped and what comes next; leave it in progress |

## 2. LIVE.md

Update `<repo>/.claude/skills/<agent>/LIVE.md` with what this session learned that a future session needs. A fact every
Agent in the repo needs goes to `<repo>/.claude/LIVE.md` instead, once. Remove notes the session proved wrong, in either
file.

## 3. Handoff

Overwrite `<repo>/.claude/ledger/<agent>/handoff.md` in the format `apex:agent-protocol` defines. Take `checkout`,
`branch` and `head` from the checkout worked in at this moment. State names the item from step 1 and any uncommitted
work outside `.claude/`.

## 4. Clean up what the session started

Only what this session created or launched. Anything that predates the session stays.

| Kind | What counts | How |
|---|---|---|
| Scratch files | temp files and directories the session created | `rm` the paths |
| Servers | dev servers, watchers, background shells the session started and that still run | stop the background task, or `kill <pid>` |
| Worktrees | worktrees the session created whose branch is merged into `<base>` (`git branch --merged <base>`) | `git worktree remove <path>`, then `git branch -d <branch>` |

`<base>` is the branch checked out in `<repo>`. Unmerged worktrees and uncommitted changes are reported, never
removed.

## 5. Commit `.claude/`

Commit the session's `.claude/` changes in `<repo>` per `apex:git-discipline`: on the operator's approval, or
automatically when `.claude/git.md` says so. The commit lands on whatever branch `<repo>` has checked out and stages
only the `.claude/` paths this session changed, never the operator's other staged or unstaged work.

## 6. Unpushed commits

For each checkout the session committed in:

```sh
git log --oneline HEAD --not --remotes   # unpushed commits
git rev-parse --abbrev-ref @{u}          # upstream name; fails when there is none
```

Report branch, upstream (or "no upstream") and the commits. Push only if the operator says so, following
`apex:git-discipline`.
