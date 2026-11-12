# Migration: per-Agent folders

Moves each Agent's notes out of its tracked role card folder into a private folder in the main checkout, and splits
them into three kinds: live notes, rules and reminder skills.

## Before and after

```
before                                    after
<repo>/.claude/                           <repo>/.claude/
  skills/<agent>/SKILL.md                   skills/<agent>/SKILL.md      role card, tracked, unchanged role
  skills/<agent>/LIVE.md                    apex/<agent>/.gitignore      one line: *
                                            apex/<agent>/LIVE.md         live notes
                                            apex/<agent>/RULES.md        operator rules
                                            apex/<agent>/skills/*.md     reminder skills
  LIVE.md                                   LIVE.md                      shared notes, tracked, unchanged
  ledger/<agent>/                           ledger/<agent>/              tracked, unchanged
  git.md                                    git.md                       unchanged
```

`.claude/apex/<agent>/` ignores itself: its `.gitignore` holds `*`, which also ignores the `.gitignore`. Nothing in it
is ever staged, and the repo's root `.gitignore` is not touched. It lives only in the main checkout; an Agent in a
worktree reads it through `<repo>`, as it already does for the ledger.

## What goes where

| File | Holds | Written by | Form |
|---|---|---|---|
| `LIVE.md` | facts about the current state that outlive one item: where things are, what is installed, what is pending outside the ledger | the Agent | short entries, rewritten when stale |
| `RULES.md` | what the operator has decided the Agent may do | the Agent, only from the operator's words | approval-first, see below |
| `skills/<topic>.md` | reminders on how to do the work: traps, good-to-knows, do's and don'ts | the Agent | one topic per file |

A reminder skill is not an Agent: it never changes the role, the owned paths or the protocol. It reminds the Agent what
to do and what to watch for in one topic. These files are plain markdown that the Agent reads with a file read. They
are not Claude Code skills and are never discovered or invoked as `/<name>`.

### RULES.md

Rules are decided by the operator. Each one:

- grants something: it starts with `Allowed:` and names what the Agent may do;
- is specific: the action, the target and the condition, so it can be checked against one action;
- carries no date, timestamp or history;
- contains no prohibition and no trap. A trap or a "don't" goes in a reminder skill. Whatever no rule allows still
  needs the operator's approval.

```markdown
# Rules

- Allowed: commit after a ledger item closes, without asking.
- Allowed: push `fresh-slate` and `master` to `fork` when the operator says "push".
- Allowed: delete a branch on `fork` after it is merged to `master`.
```

Rewrite a rule in place when the operator changes it; delete it when the operator withdraws it.

### Reminder skills

```markdown
# Install

- Do: run `make install` after changing anything under `skills/`; nothing is live until then.
- Don't: search git-ignored files in adopted repos.
- Trap: a legacy `~/.claude/bin/apex` shadows the new binary on `PATH`.
- Good to know: `pwsh` is not installed on this machine; PowerShell edits are read-checked only.
```

## Steps, per Agent

Run in the main checkout. `<a>` is the Agent's directory name under `.claude/skills/`.

1. Create the folder:

   ```bash
   mkdir -p .claude/apex/<a>/skills && printf '*\n' > .claude/apex/<a>/.gitignore
   ```

2. Sort every entry of `.claude/skills/<a>/LIVE.md`, one at a time:

   | Entry | Goes to |
   |---|---|
   | an operator decision that grants something | `RULES.md`, rewritten as `Allowed: ...` |
   | an operator decision phrased as a ban ("never push to origin") | `RULES.md` as the grant it implies ("Allowed: push to `fork`"), plus a `Don't` in a reminder skill if the ban carries a real risk |
   | a trap, a convention, a do or don't, a how-to | `skills/<topic>.md` |
   | a fact about the current state | `LIVE.md` |
   | stale or already covered by the ledger | nothing |

   Drop dates and "since"/"as of" wording from rules as they move. Show the operator the new `RULES.md` before step 3:
   only the operator decides rules.

3. Remove the old file from the tree and from the index:

   ```bash
   git rm -q .claude/skills/<a>/LIVE.md
   ```

4. In the role card, point the live notes row at the new folder:

   | Owns | Path |
   |---|---|
   | Live notes, rules, reminder skills | `.claude/apex/<a>/` |

   The role card is the operator's file: make this one edit with the operator's approval and change nothing else.

5. Check:

   ```bash
   git check-ignore -q .claude/apex/<a>/LIVE.md && echo ignored
   git status --porcelain -- .claude/apex          # prints nothing
   test ! -e .claude/skills/<a>/LIVE.md && echo moved
   grep -n '^- ' .claude/apex/<a>/RULES.md | grep -v '^[0-9]*:- Allowed: '   # prints nothing
   ```

6. Commit the removal and the role card edit per `git-discipline`. The new folder never appears in the commit.

`.claude/LIVE.md`, the ledger, the handoff and `.claude/git.md` do not move.

## Protocol changes that ship with this

The repo's skills and design need the following changes before a migrated Agent starts. They are separate ledger
items, not migration steps.

- `docs/design/agents.md`: the layout, the three file kinds and the Rules form.
- `agent-protocol`: at Startup, read `.claude/apex/<agent>/LIVE.md`, `RULES.md` and every `skills/*.md`. Write traps
  to reminder skills, and write rules only from the operator's words. In the LIVE.md section, the Agent's notes move
  to the private folder.
- `sign-off` and `git-discipline`: drop the Agent's `LIVE.md` from staged `.claude/` paths. The private folder is never
  staged.
- `apex-init`: create `.claude/apex/<agent>/` with its `.gitignore` on init. Treat a tracked
  `.claude/skills/<agent>/LIVE.md` as a foreign layout and route it to migrate.
- `README.md`: the layout table.
