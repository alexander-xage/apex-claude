---
name: git-discipline
description: Commit, push and PR rules for an Apex Agent. Reads the repo's .claude/git.md first, then applies default permissions (commit on approval, push on approval, PR only on request), explicit-path staging, and Conventional Commits message rules. Use before any git commit, push, or PR, and when writing a commit message or PR body.
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

## 3. Staging

An item's commit carries its code plus its `.claude/` changes: the ledger item, both `LIVE.md` files and the handoff
written after the previous commit. In a worktree, commit the code there and the `.claude/` changes in the main checkout.

An item filed into a peer's ledger is the filer's change: the filer commits it in the peer's repo, in its own commit.
A commit of `.claude/` changes alone, with no code, is fine at any time.

Stage by explicit path. Never `git add -A`, `git add .`, or `git commit -a`.

```bash
git status --porcelain -uall -- <paths>   # what can be staged, one line per file
git add -- <paths status listed>
git rm --cached -q -- <deleted path>      # a deletion, even under an ignored directory
git diff --cached --stat                  # confirm before committing
```

Leave out any path `git status` does not list, without comment.

## 4. Commit message

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

## 5. PR title and body

Only when the operator asked for a PR. Use the PR template from `.claude/git.md` if it names one.

- **Title:** Conventional Commits prefix, imperative, 70 chars max, no trailing period. A squash merge turns the title
  into the commit subject.
- **Body:** 120 words max. Only what the diff cannot show: why the change was needed, why this approach over the
  obvious one, any decision a reviewer would flag as a mistake.
- **Cut:** file lists, test plans, dependency bumps, headings on a short body, local detail (branch, worktree,
  scratchpad paths, iteration counts), references a stranger cannot open, AI attribution.
