---
name: apex-init
description: Adopt the Apex Agent protocol in a repo. Inspects the repo and does exactly one of three things. It initializes Agents from a blank slate, migrates a foreign layout of role cards, ledgers, handoffs and memory onto the protocol, or reports the repo conformant and changes nothing. Use once per repo, on /apex:apex-init or when the operator asks to set up or migrate Apex Agents.
---

# apex-init

A single-use session. Inspect, route to exactly one outcome, finish. `<repo>` is the main checkout:

```bash
dirname "$(git rev-parse --path-format=absolute --git-common-dir)"
```

Templates are in this skill's `templates/` directory.

| Placeholder | Value | Example |
|---|---|---|
| `{{agent}}` | Agent name: directory, ledger and `--agent` value | `web-dev` |
| `{{Agent}}` | display name: `{{agent}}` with each `-`-separated word capitalized, hyphens kept | `Web-Dev` |
| `{{one-line role}}` | one sentence ending in a period | `Builds and deploys the web apps.` |
| `{{role}}` | the `## Role` body: the one-line role, or the old card's role content | |
| `{{...}}` in `git.md` | the row's value; the text inside is a hint | |

The `description` in `agent-SKILL.md` is a double-quoted string: escape any `\` in the role as `\\` and any `"` as `\"`.

## 1. Inspect

Collect these facts before routing. Read only; change nothing yet.

| Fact | How |
|---|---|
| Apex Agents | each `<repo>/.claude/skills/<a>/` holding both `SKILL.md` and `LIVE.md`, whose `SKILL.md` loads `apex:agent-protocol` |
| Foreign ledgers | directories of numbered item files kept outside `.claude/ledger/`, such as `DevLedger/` with `INDEX.md` and `NNN-slug.md` |
| Foreign handoffs | handoffs of an Agent defined in this repo kept outside `.claude/ledger/<a>/handoff.md`, such as `docs/handoff/*.md`. Notes files do not count |
| Foreign role cards | `.claude/skills/<x>/SKILL.md` that describe an agent role but are not Apex Agents, such as `.claude/skills/dev-agent/` |
| Other-home files | handoffs, ledgers or cards of an Agent whose home is another repo, such as Xage files in LLMTraining. Reported, never migrated, and not counted as foreign layout |
| Project memory | `find ~/.claude/projects -maxdepth 3 -path "*/<slug>*/memory/*.md"` (glob-safe in every shell), where `<slug>` is `<repo>` with every non-alphanumeric character replaced by `-`. The slug maps `/` and `-` alike, so a `<slug>-*` match counts only when a directory inside `<repo>` produces it and no sibling directory of `<repo>` does (`ls -d <repo>-*`); when both could, ask the operator |

## 2. Route

| State found | Outcome |
|---|---|
| `apex list --status all --agent <a>` exits non-zero for any Apex Agent | Stop. Report the Agent, exit code and message; change nothing |
| No Apex Agents, and no foreign role cards, ledgers, handoffs or project memory files | 3. Blank slate |
| Any foreign role card, ledger or handoff | 4. Migrate |
| Project memory files and no Apex Agents | 4. Migrate |
| Apex Agents present, no foreign layout, and only the `CLAUDE.md`, shared-files or auto-memory signal in 5 fails | 6. Shared setup, then 7 |
| Apex Agents present and every signal in 5 holds | 5. Conformant |

When the route is unclear, show the operator the facts from step 1 and ask.

Before the first write on any route, record the output of `git -C <repo> status --porcelain`: the files already
modified hold the operator's own edits, and section 7 needs the list.

## 3. Blank slate

1. Take the Agents from the operator's request when it names them; otherwise ask. Per Agent: a name matching
   `^[a-z0-9][a-z0-9-]*$` that no existing `.claude/skills/` directory uses, and a one-line role. Ask for more role
   text only if the operator offers it.
2. Per Agent:
   - write `<repo>/.claude/skills/<agent>/SKILL.md` from `templates/agent-SKILL.md`;
   - write `<repo>/.claude/skills/<agent>/LIVE.md` from `templates/LIVE.md`.
3. Do the shared setup in 6.
4. Check with `apex list --status all --agent <agent>` per Agent: exit 0, no items.
5. Commit as in 7.
6. Report the files written and how to start: `/<agent>`.

## 4. Migrate

Migration moves and rewrites existing state. Nothing is moved or deleted before the operator approves the plan, and old
files are removed only after the operator confirms the migrated result.

### 4.1 Read

Read every foreign ledger item, index, handoff and role card, and every project memory file. Collect every free-text
shape used to cite an old item, such as `DevLedger 033`, `(025)`, `R007`, `M011`, `Master-Tasks/M011-...`,
`XageLedger 018`.

### 4.2 Plan

Show the operator one plan and wait for approval.

| Plan section | Contents |
|---|---|
| Agents | old role card or ledger to new Agent name; default is the old name without an `-agent` suffix. With nothing to map from (memory files only), ask as in 3 step 1 |
| Shared ledgers | a ledger several Agents work, such as `Master-Tasks/` kept by research: folds into its keeper's ledger after the keeper's own items; its per-agent columns become peer references |
| Items | per old item: old path, new id, title, kind, new status, close reason |
| Ids | one map per Agent: its old items renumbered from `001` in old order, shared-ledger items last |
| Reference shapes | every shape from 4.1, each with its rewrite; the operator confirms the list |
| Status conflicts | every item whose file status differs from its `INDEX.md` row or from its old handoff; the item file wins, the operator confirms, and a handoff's differing claim goes into the new handoff's State as "check before resuming" |
| In progress | at most one item per Agent; the operator picks it when several old items are running |
| Handoffs | old path to `.claude/ledger/<agent>/handoff.md`, or dropped, and why |
| Memory | per rule: the Agent `LIVE.md` it goes to, `.claude/LIVE.md`, or dropped |
| References | in-repo files citing old items or paths (to rewrite), and references in other repos (listed only) |
| Left alone | other-home files and anything else that stays, and why |
| To delete after confirmation | every old file the migration replaces |

**Title** is the item's H1, else the `INDEX.md` task column. Drop the item's own old id from it, such as `M003` in
`M003 run5 setup`, and rewrite any other old id to its peer form. **Created** stays the migration date that `apex add`
writes. **Kind** is `task` unless the old item says it was found during another item's work, then `followup`.

| Old status says | New status | Reason |
|---|---|---|
| done | `closed` | `done: <short result>` |
| abandoned, handed to another Agent | `closed` | `dropped: <why>` |
| done, but waiting on someone | `closed`; the waiting part goes into the handoff's Open threads unless a later item or this migration resolved it (list those in the plan) | `done: <short result>` |
| in progress, running, launched | `in-progress` for the one picked, `open` for the rest | none |
| open, awaiting someone | `open` | none |

A reason is one line that does not repeat its prefix (`done: 5a passes all gates`, not `done: done`), with old
references rewritten by the id map. An item marked running or launched may be stale. Check it against `INDEX.md`, the
old handoff and `git log`, and offer to close the finished ones in the same prompt that asks for the in-progress pick.

**Reference forms.** An item's own-ledger reference is `NNN`. Every other reference, from a peer's item, code or
docs, is `<repo>:<agent>/NNN`, where `<repo>` is the home repo's directory name, such as `xage-guard:xage/018`. Old
shapes citing an Agent not migrated here, such as `XageLedger 018`, stay as they are. A bare number may cite another
Agent's ledger, such as a research item's `035 rebuild` meaning a dev item: resolve each occurrence from its context,
and list the uncertain ones in the plan. A `memory <name>` mention becomes a reference to the `LIVE.md` that took that
rule. Where old text already pairs a shape with its path, such as `M011 (Master-Tasks/M011-x.md)`, write the new
reference once, not twice.

**Handoffs.** Of several dated handoffs for one Agent, the newest is carried and the superseded ones are dropped, not
merged. README and index files are dropped. A notes file becomes `LIVE.md` facts or stays where it is; ask which.

**Memory.** One `LIVE.md` entry per rule, not per file. Keep facts; drop procedure (protocol and session steps) and
stale state (what was running or last committed). A fact several Agents need goes to `.claude/LIVE.md`, once. Index
files such as `MEMORY.md` are dropped.

**References.** Search the main checkout only: tracked and untracked text files, honoring ignore rules, skipping
worktrees, the graph and scratch directories. Never search ignored files: they hold model weights, run outputs and
virtualenvs, and reading them can exhaust memory on a machine that is training.

Pass each old shape and path as a fixed string, one `-e` each, so `(025)` and `.` match literally:

```bash
git -C <repo> grep -nIF --untracked -e '<shape>' -e '<path>' -- . \
  ':(exclude).claude/worktrees' ':(exclude).claude/graphify' ':(exclude,glob)**/node_modules/**' \
  ':(exclude,glob)**/.venv/**' ':(exclude,glob)**/venv/**' ':(exclude,glob)tmp/**' ':(exclude,glob)**/scratch*/**'
```

Read the protocol files under `.claude/` directly; the search may not reach them.

Search other repos too, read only, with the same command: the home repo of each other-home Agent found in step 1, and
any repo the operator names.

Anything the plan cannot map or merge, ask about.

### 4.3 Execute

Before the first write, the working tree must be clean, or the operator accepts carrying the uncommitted edits into the
migration. Then, in this order, because `apex` needs the Agent's `SKILL.md` to exist:

1. **Role cards.** Write `.claude/skills/<agent>/SKILL.md` from `templates/agent-SKILL.md`. Put the old card's role
   content under `## Role`: ownership, peers, working method, voice. Drop what the protocol now supplies: session-start
   steps, ledger and handoff procedure, pointers to memory or an index.
2. **LIVE.md.** Write each Agent's from `templates/LIVE.md` and `.claude/LIVE.md` from `templates/shared-LIVE.md` if
   missing, then fold in the planned memory entries.
3. **Items.** Per Agent, in id map order:
   1. `apex add <kind> "<title>" --agent <agent>`; it prints the new path. Stop and ask if its id differs from the map.
   2. Write the body below the frontmatter; never touch the frontmatter. Rewrite references by the id map.

      | Section | From the old item |
      |---|---|
      | `## Goal` | `From:` line (`user` for the operator, a reference form above for a peer, `<agent>` for a peer that asked without an item), then the old task and done-when text |
      | `## Approach` | the old approach if stated, else leave empty |
      | `## Log` | first line `migrated from <old path>; old status: <text>; created <old date>` (drop the date if the old item has none), then the old progress notes, then each other old section (Rules, Gates, Owners, Design) as a `###` subsection |
      | `## Outcome` | the old result, for closed items only |

      For an item not closed, the old result or next-step text goes in the Log.
   3. Set status: `apex close <id> "<reason>" --agent <agent>`, or `apex start <id> --agent <agent>` for the one
      in-progress item. Open items need no call.

   Fix a wrong title with `apex retitle <id> "<title>" --agent <agent>` and a wrong reason with `apex reopen` then
   `apex close`; never file the item again.
4. **Handoffs.** Write `.claude/ledger/<agent>/handoff.md` in the format from `apex:agent-protocol`. Its frontmatter
   is filled after the commit, in 7. Carry State, Next and Open threads from the carried handoff, plus the waiting
   parts from the status table; rewrite references by the id map. Procedure text
   that belongs to the role card is dropped.
5. **References.** Rewrite every in-repo reference the plan lists: comments, docs, both `LIVE.md` levels. Rewrite
   each file in one pass from the map; chained replacements turn old `014 -> 013` into `012`. A string literal that
   reaches program output is not rewritten; list it for the operator.
6. **Shared setup** in 6.

### 4.4 Confirm, then remove

1. Per Agent, `apex list --status all --agent <agent>`: the item count equals the planned count, and statuses match the
   plan.
2. Rerun the reference search from 4.2: nothing in the repo still cites an old shape or path, except what the plan
   leaves alone, the files on the delete list, and the Log sections of migrated items, which keep history as written.
3. Show the operator the results of 1 and 2, the delete list, and the references in other repos for them to rewrite.
   On confirmation, delete the old ledgers, indexes, handoffs and role card directories. Otherwise fix what the
   operator names and ask again.
4. Deleting the migrated memory files touches `~/.claude`: ask for an explicit yes, and delete only on it. Without
   one, the deletion stays a pending step.
5. Commit as in 7.

## 5. Conformant

A repo is conformant when every signal holds. Report which held and change nothing. Other-home files do not count.
Project memory files left while Apex Agents exist are a pending deletion from 4.4 step 4: name it in the report, not
as a failed signal.

| Signal | Check |
|---|---|
| Every Apex Agent has a ledger that reads cleanly | `apex list --status all --agent <a>` exits 0 |
| No foreign layout remains | step 1 found no foreign ledgers, handoffs or role cards |
| `CLAUDE.md` frames sessions | `<repo>/CLAUDE.md` contains the `## Apex Agents` heading |
| Shared files exist | `<repo>/.claude/git.md` and `<repo>/.claude/LIVE.md` exist |
| Auto-memory is off | `<repo>/.claude/settings.json` has `"autoMemoryEnabled": false` |

## 6. Shared setup

Blank slate and migrate both finish with these; the shared-setup route runs them alone.

1. **CLAUDE.md.** Take `templates/CLAUDE-section.md` as is. If `<repo>/CLAUDE.md` is missing, create it with
   `# <repo directory name>` and that section. If it already has an `## Apex Agents` heading, replace that section;
   otherwise append it. Never add it twice.
2. **Shared LIVE.md.** Create `<repo>/.claude/LIVE.md` from `templates/shared-LIVE.md` if missing.
3. **git.md.** Fill `templates/git.md` and write `<repo>/.claude/git.md`. Infer every row first, then ask the operator
   only about the rows still unknown. A row with no source, such as no remote, is written as `none` once the operator
   confirms. If `.claude/git.md` exists, keep it and ask only about missing rows.

   | Row | Source |
   |---|---|
   | Commit, Push | the old role cards and memory; ask only when neither states it |
   | Message format, ticket prefix | `git log -50 --format=%s` |
   | Branch naming | `git branch -a` |
   | Remotes | `git remote -v` |
   | PR template | `.github/pull_request_template.md` or `.github/PULL_REQUEST_TEMPLATE/` |

4. **settings.json.** Set `"autoMemoryEnabled": false` in `<repo>/.claude/settings.json`. Merge into the existing
   JSON and keep every other key; create the file with `{"autoMemoryEnabled": false}` if missing.

## 7. Commit

The last step, after the operator approves. Load `apex:git-discipline` and commit what this session created and
deleted in the repo, staged by explicit path. A file on the 4.3 list of files already modified holds the operator's own
edits: leave it out, or name it to the operator and stage it only on their yes.

After a migration, write each migrated handoff's `checkout`, `branch` and `head` from the main checkout, whether or
not the operator approved the commit, so the next session's freshness check has values to run on. If the commit
happened, commit `.claude/` again.
