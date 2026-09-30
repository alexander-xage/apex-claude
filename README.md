# apex

A Claude Code plugin for user-defined Agents. Each Agent is a skill you write; it owns one ledger and one handoff,
works the ledger one item at a time, and keeps its notes in `LIVE.md` instead of memory. The `apex` CLI owns the
deterministic parts: ledger state and the per-checkout code graph.

The contract is [docs/design/agents.md](docs/design/agents.md).

## Install

Requires Go 1.26+, `graphify` 0.9.72+ on `PATH`, and the ponytail marketplace.

```sh
git clone git@github.com:FNGApex/apex-claude.git && cd apex-claude
make install        # builds apex into ~/.local/bin (override with PREFIX=...)
```

`PREFIX` must be on the `PATH` of the shell Claude Code runs Bash in; every skill calls a bare `apex`.

From a Claude Code session inside the clone:

```text
/plugin marketplace add DietrichGebert/ponytail
/plugin marketplace add ./
/plugin install apex@apex
```

The plugin and the binary then come from one checkout, so the skills match the binary's verbs. Installing `apex` pulls
in `ponytail@ponytail`.

For local development, `make build` writes `bin/apex`, which Claude Code puts on the Bash `PATH` when the plugin loads
with `claude --plugin-dir .`.

Then adopt the protocol in a repo with one session that runs `/apex:apex-init`.

## The Agent model

```text
<repo>/.claude/
  skills/<agent>/SKILL.md      role card, user-owned; lists the skills to pre-load
  skills/<agent>/LIVE.md       live notes, agent-owned; replaces memory
  LIVE.md                      shared notes: facts every Agent in the repo needs
  ledger/<agent>/NNN.md        one item per file, never deleted
  ledger/<agent>/handoff.md    one file, overwritten
  graphify/<checkout>/         one graph per checkout
  git.md                       this repo's commit, push and PR conventions
```

The graph covers code only: `apex graph` keeps `.claude/` out of it without writing into the checkout.

## apex verbs

Every verb but `graph` takes `--agent <name>`, which must name an existing `.claude/skills/<name>/SKILL.md`; until
then the verb exits 1. `/apex:apex-init` creates the role card.

| Verb | Does |
|---|---|
| `add <task\|followup> <title>` | files a new open item, prints its path |
| `list [--status s,s\|all] [--kind k]` | lists items; default status `in-progress,open` |
| `start <id>` | open to in-progress |
| `close <id> <reason>` | to closed |
| `reopen <id>` | in-progress or closed to open |
| `retitle <id> <title>` | replaces an item's title, any status |
| `graph <graphify args>` | runs graphify against this checkout's graph |

Exit codes: 0 ok, 1 not found, 2 environment, 3 illegal transition, 4 corrupt ledger, 64 usage. `graph` exits
with graphify's own code, or 128+N when graphify dies by signal N.

## Skills

| Skill | Use |
|---|---|
| `/apex:agent-protocol` | session start, the item loop, ledger and handoff rules; every Agent pre-loads it |
| `/apex:graphify-discipline` | how Agents query and update the code graph through `apex graph` |
| `/apex:git-discipline` | commit rules and per-repo conventions from `.claude/git.md` |
| `/apex:sign-off` | ends a session: log, `LIVE.md`, handoff, cleanup, unpushed commits |
| `/apex:apex-init` | adopts the protocol in a repo: initialize, migrate, or report conformant |
