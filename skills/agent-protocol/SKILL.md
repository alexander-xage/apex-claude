---
name: agent-protocol
description: The shared protocol every Apex Agent follows. Covers session startup from the handoff, working the ledger one item at a time through the apex binary, the handoff format and freshness check, LIVE.md upkeep, and collaboration with peer Agents. Pre-loaded by an Agent's role card; use whenever an Agent starts a session, picks, works or closes a ledger item, or writes its handoff.
---

# Agent protocol

You are an Agent: a role card at `.claude/skills/<agent>/SKILL.md` that owns one ledger and one handoff. `<agent>` below
is your role card's directory name.

```
<repo>/.claude/
  skills/<agent>/SKILL.md      role card, user-owned; never edit it
  skills/<agent>/LIVE.md       your live notes; you own and rewrite them
  LIVE.md                      shared notes for every Agent in this repo; any Agent edits them
  ledger/<agent>/NNN.md        one item per file, never deleted
  ledger/<agent>/handoff.md    your handoff, overwritten each time
```

`<repo>` is always the main checkout, including when you work in a worktree. Find it with:

```bash
dirname "$(git rev-parse --path-format=absolute --git-common-dir)"
```

## apex

Every stateful call passes `--agent <agent>` explicitly. There is no default Agent. Subagents and Ultracode workers
never run stateful verbs.

| Call | Does |
|---|---|
| `apex add <task\|followup> <title> --agent <a>` | files a new open item, prints its path |
| `apex list [--status s,s\|all] [--kind task\|followup] --agent <a>` | lists items; default status `in-progress,open`. The summary line counts every status for the selected `--kind`; `--status` filters only the rows |
| `apex start <id> --agent <a>` | open to in-progress, prints the item path |
| `apex close <id> <reason> --agent <a>` | to closed, from open or in-progress, prints the item path |
| `apex reopen <id> --agent <a>` | in-progress or closed back to open, prints the item path |
| `apex retitle <id> <title> --agent <a>` | replaces the item's title in any status, prints the item path |
| `apex graph <graphify args>` | graphify on this checkout's graph; no `--agent` |

| Exit | Meaning | Response |
|---|---|---|
| 0 | ok | continue |
| 1 | not found | check the id and the `--agent` value |
| 2 | environment | read the message, stop and report |
| 3 | illegal transition | read the message; usually another item is already in progress |
| 4 | corrupt ledger | stop and report the file; never hand-edit frontmatter to fix it |
| 64 | usage | fix the call |

The table does not apply to `apex graph`; its exits and errors are in `graphify-discipline`.

The binary owns item frontmatter. You edit only the body sections. Fix a wrong title with `apex retitle`, and a
wrong close reason with `apex reopen` then `apex close`.

## Startup

1. Read `<repo>/.claude/skills/<agent>/LIVE.md` with a file read.
2. Read `<repo>/.claude/LIVE.md` with a file read, if it exists.
3. Load every skill your role card lists.
4. Read `<repo>/.claude/ledger/<agent>/handoff.md`. If it does not exist, skip to step 6.
5. Check freshness. The handoff is stale when any of these holds:

   | Check | Command |
   |---|---|
   | recorded `checkout` no longer exists | `test -d <checkout>` |
   | `branch` differs | `git -C <checkout> rev-parse --abbrev-ref HEAD` |
   | code outside `.claude/` differs from `head` | `git -C <checkout> diff --quiet <head> HEAD -- . ':(exclude).claude'` exits non-zero |

   An unknown `head` makes the diff exit 128 with `fatal: bad object`: that is stale, not a failed command.
   A stale handoff is still read. Staleness means State may be wrong: reconcile it against what changed
   (`git diff --stat <head> HEAD`, `git status`) before trusting it.
6. Run `apex graph update .` and `apex list --agent <agent>`, with or without a handoff.

## Pick the next item

`apex list --agent <agent>`. Resume the in-progress item if there is one; at most one exists. Otherwise take what the
user names or the handoff's Next. New work the user asks for gets filed first with `apex add` and then worked like any
item.

## Work an item

One item at a time. Close the current one, or return it to open with `apex reopen <id> --agent <agent>`, before
starting another.

1. `apex start <id> --agent <agent>` (skip when resuming an in-progress item).
2. Fill `## Goal`: a `From:` line naming who asked (`user`, a parent item id such as `003`, or `repo:agent/004` for a
   peer, where `repo` is the peer's home repo directory name), then done-criteria that can be checked by running or
   reading something.
3. Fill `## Approach` with one of these and the reason:

   | Mode | When |
   |---|---|
   | direct | the change is small, or the context it needs is already in this session |
   | subagent | a self-contained slice that benefits from a fresh context |
   | ultracode | the item splits into independent slices that run in parallel |

   Subagents and Ultracode workers get the item's Goal and read-only graph access. They report back; they own no
   state.
4. Append one line to `## Log` per step taken, as you go. The Log is the crash cursor: after an interruption, the last
   line tells you where to resume. Cite a peer's item in full: `<repo>:<agent>/NNN`.
5. Review the result against the Goal's done-criteria, run `/ponytail:ponytail-review` on the result, and act on its
   findings. If that skill is unavailable, review against the ponytail ladder yourself and say so in the Log.
6. Update your `LIVE.md` or `<repo>/.claude/LIVE.md` when you learned something durable: a repo fact, a convention, a
   trap. Skip it when nothing durable came up.
7. Write `## Outcome`.
8. `apex close <id> "done: <one line>" --agent <agent>`.
9. Commit the item's work per `git-discipline`: the code plus the item's `.claude/` changes. In a worktree, the
   code commits in the worktree and the `.claude/` changes commit in `<repo>`. An item with no code commits its
   `.claude/` changes alone, like any other.
10. Rewrite `handoff.md` after the commit, so `head` covers the item's code. It rides in the next item's commit.

`.claude/` files may be committed at any time, per `git-discipline`.

Every close reason starts with one of these prefixes:

| Prefix | When |
|---|---|
| `done: ` | the Goal is met |
| `dropped: ` | the item will not be done; say why |
| `split: <ids>` | the item was replaced by children |

### Drop an item

Close it straight from open, without starting it: `apex close <id> "dropped: <why>" --agent <agent>`, then commit
as in step 9.

### Split an item

When an item is too big for one pass:

1. `apex add` each child. Give each child a `From: <parent id>` Goal line.
2. `apex close <parent> "split: <ids>" --agent <agent>`, listing the ids the adds printed, for example
   `split: 008-010` or `split: 008, 011`.

### Worktrees and other repos

`apex` resolves your state from a worktree too. Work in another repo keeps state in your home repo; note that repo's
path and what changed there in the item's Log.

## Handoff

Overwrite `<repo>/.claude/ledger/<agent>/handoff.md` after every close and whenever you stop mid-item. Fill the
frontmatter by running the commands in the checkout you worked in; `head` is always the full SHA.

The frontmatter keys and the four headings are literal; replace each `<...>` with its value.

```
---
checkout: <absolute path of the checkout worked in>
branch: <output of git rev-parse --abbrev-ref HEAD>
head: <output of git rev-parse HEAD>
---
## Startup
invoke the Agent skill (reads both LIVE.md files); apex graph update .; apex list --agent <agent>

## State
<current item and where it stopped; uncommitted work>

## Next
<the item to take next, and why>

## Open threads
<questions and waits that outlive the current item>
```

The Startup line is literal apart from `<agent>`.

## LIVE.md

`LIVE.md` replaces memory. Keep it current and short: rewrite or delete stale entries instead of appending
contradictions. It holds facts that outlive one item. Per-item progress belongs in the item's Log.

| File | Holds | Who edits |
|---|---|---|
| `<repo>/.claude/LIVE.md` | facts every Agent in this repo needs: build steps, repo conventions, shared traps | any Agent |
| `<repo>/.claude/skills/<agent>/LIVE.md` | facts only your role needs | you |

A fact goes in one file, never both. Move it to `<repo>/.claude/LIVE.md` once a second Agent needs it.

## Collaboration

- Find peers with `ListAgents` and talk to them with `SendMessage`.
- To give a peer work, file it into the peer's ledger. Get the peer's home repo path from the peer with `SendMessage`
  or from a `LIVE.md`, then run `cd <peer home repo> && apex add <kind> <title> --agent <peer>` as one command; `apex`
  has no directory flag. Write `From: <your repo>:<you>/<your item id>` in its Goal, then tell the peer with
  `SendMessage` if `ListAgents` shows it running.
- Only the owner starts or closes an item. A peer only files: it never runs `start`, `close`, `reopen` or `retitle`
  with another Agent's name.
