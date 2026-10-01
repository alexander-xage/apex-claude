# apex

A Claude Code skill set for user-defined Agents. Each Agent is a skill you write; it owns one ledger and one handoff,
works the ledger one item at a time, and keeps its notes in `LIVE.md` instead of memory. The `apex` CLI owns the
deterministic parts: ledger state and the per-checkout code graph.

The contract is [docs/design/agents.md](docs/design/agents.md).

## Install

Requires Go 1.26+, `graphify` 0.9.72+ on `PATH` (`uv tool install graphifyy`), and the ponytail plugin
(`/plugin marketplace add DietrichGebert/ponytail`, then `/plugin install ponytail@ponytail`).

```sh
git clone git@github.com:FNGApex/apex-claude.git && cd apex-claude
make install      # apex -> ~/.local/bin; skills and the Apex output style -> ${CLAUDE_CONFIG_DIR:-~/.claude}
make uninstall    # removes all three
```

`make install` stops if graphify or ponytail is missing, or if a skill folder or `output-styles/apex.md` of the same
name exists that it did not install; it never overwrites or removes a file of yours. `PREFIX` must be on the `PATH` of the shell Claude Code
runs Bash in; every skill calls a bare `apex`. The skills install as user skills, not as a plugin, so they keep bare
names (`/apex-init`, `/sign-off`); plugin skills are always prefixed with the plugin name. Restart Claude Code after
installing. The output style is optional: pick it with `/output-style Apex`.

Then adopt the protocol in a repo with one session that runs `/apex-init`.

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
then the verb exits 1. `/apex-init` creates the role card.

| Verb | Does |
|---|---|
| `add <task\|followup> <title>` | files a new open item, prints its path |
| `list [--status s,s\|all] [--kind k]` | lists items; default status `in-progress,open` |
| `start <id>` | open to in-progress |
| `close <id> <reason>` | to closed |
| `reopen <id>` | in-progress or closed to open |
| `retitle <id> <title>` | replaces an item's title, any status |
| `graph [--name n] <graphify args>` | runs graphify against this checkout's graph, or its named graph `n` |

Exit codes: 0 ok, 1 not found, 2 environment, 3 illegal transition, 4 corrupt ledger, 64 usage. `graph` exits
with graphify's own code, or 128+N when graphify dies by signal N.

## Skills

| Skill | Use |
|---|---|
| `/agent-protocol` | session start, the item loop, ledger and handoff rules; every Agent pre-loads it |
| `/graphify-discipline` | how Agents query and update the code graph through `apex graph` |
| `/git-discipline` | commit rules and per-repo conventions from `.claude/git.md` |
| `/sign-off` | ends a session: log, `LIVE.md`, handoff, cleanup, unpushed commits |
| `/apex-init` | adopts the protocol in a repo: initialize, migrate, or report conformant |
| `/pr-review <pr> [mode]` | reviews one pull request through a one-off workflow and serves the draft review before posting; modes `general`, `ponytail`, `security` (stub), `mega` |
