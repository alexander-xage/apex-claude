---
name: git-discipline
description: Commit, push and PR rules for an Apex Agent. Reads the repo's .claude/git.md first, then applies default permissions (commit on approval, push on approval, PR only on request), the silent check that decides whether a commit is worth making, explicit-path staging, and Conventional Commits message rules. Use before any git commit, push, or PR, and when writing a commit message or PR body.
---

# Git discipline

## 1. Read `.claude/git.md` first

It holds this repo's conventions: commit and push permissions, message format, branch naming, ticket prefixes,
remotes and PR template. Anything it states overrides the defaults below; anything it leaves out falls back to them. If
the file is missing, use the defaults.

## 2. Permissions

| Action | Default |
|---|---|
| Commit | on the operator's approval, or automatically when `.claude/git.md` says so |
| Push | on the operator's approval only; the `Push` row of `.claude/git.md` adds rules, never permission |
| PR | only when the operator asks for one |

- Never remind the operator to open a PR.
- Say nothing about pushing until Sign off.

## 3. When to commit

Commit less. A commit is a unit someone would review or revert on its own, never a save point. `Commit: automatically`
in `.claude/git.md` removes the question to the operator; it does not mean committing more often.

Consider a commit only at these moments. An edit, a passing test or a new Log line is not one of them.

| Moment | Why |
|---|---|
| a ledger item closes | the work has a Goal that was met |
| Sign off | the session's state must survive it |
| before a branch switch, rebase or worktree removal | uncommitted work could be lost |

At that moment, commit only when all of these hold:

1. **Whole.** The checks pass and no step still to come will change the same files.
2. **Stands alone.** Its subject needs no "wip", "part 2", "more" or "fix previous commit".
3. **More than bookkeeping.** `.claude/` changes alone (ledger, handoff, `LIVE.md`) wait for the next commit that
   carries code, or for Sign off.

When one fails, do not commit. The work stays in the tree and joins the next commit; the handoff's State names it as
uncommitted.

Run this check silently. Never tell the operator that you checked, what it found or that a commit is being held, and
never ask whether now is a good time to commit. The operator hears about a commit only when section 2 requires
approval, and only after the check passed. A direct order from the operator to commit skips the check.

## 4. Staging

An item's commit carries its code plus its `.claude/` changes: the ledger item, both `LIVE.md` files and the handoff
written after the previous commit. In a worktree, commit the code there and the `.claude/` changes in the main checkout.

An item filed into a peer's ledger is the filer's change: the filer commits it in the peer's repo, in its own commit.
That is the one `.claude/`-only commit that does not wait (section 3).

Stage by explicit path. Never `git add -A`, `git add .`, or `git commit -a`.

```bash
git status --porcelain -uall -- <paths>   # what can be staged, one line per file
git add -- <paths status listed>
git rm --cached -q -- <deleted path>      # a deletion, even under an ignored directory
git diff --cached --stat                  # confirm before committing
```

Leave out any path `git status` does not list, without comment.

## 5. Commit message

```
<type>(<scope>): <imperative summary>

<optional body: the non-obvious why>
```

| Rule | Detail |
|---|---|
| Types | `feat`, `fix`, `refactor`, `perf`, `docs`, `test`, `chore`, `build`, `ci`, `style`, `revert`; scope optional |
| Subject | imperative ("add", not "added"), 50 chars or fewer when possible, 72 hard cap, no trailing period; capitalization after the colon matches the repo's history |
| Body | only for a non-obvious why, a breaking change, a migration note, or an issue ref; 4 lines max, wrapped at 72, bullets with `-` |
| Always a body | breaking change (`!` after type plus `BREAKING CHANGE:`), security fix, data migration, revert |
| Refs | at the end: `Closes #42`, `Refs #17` |

Never write:

- what the diff already shows: file lists, renamed symbols, mechanical changes
- "this commit", "I", "we", "now"
- emoji
- AI attribution in any form, including `Co-Authored-By` trailers for a model and session links

## 6. PR title and body

Only when the operator asked for a PR. Use the PR template from `.claude/git.md` if it names one.

- **Title:** Conventional Commits prefix, imperative, 70 chars max, no trailing period. A squash merge turns the title
  into the commit subject.
- **Body:** 120 words max. Only what the diff cannot show: why the change was needed, why this approach over the
  obvious one, any decision a reviewer would flag as a mistake.
- **Cut:** file lists, test plans, dependency bumps, headings on a short body, local detail (branch, worktree,
  scratchpad paths, iteration counts), references a stranger cannot open, AI attribution.
