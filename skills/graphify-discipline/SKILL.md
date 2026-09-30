---
name: graphify-discipline
description: How an Apex Agent and its subagents use the graphify code graph. Covers running graphify only through `apex graph`, when to update the graph, who writes and who reads, budgeted queries, the forbidden commands, and recovery from graphify errors. Use whenever an Agent orients in code, dispatches a subagent that reads code, or hits a graphify error.
---

# Graphify discipline

The code graph is the Agent's first look at code: who calls what, what a change touches, how two symbols connect.
Read the graph before grepping or opening files.

## Always through apex

Run every graphify command as `apex graph <args>`. Never run bare `graphify`, and never use the `/graphify` skill.

`apex graph` sets `GRAPHIFY_OUT` to `<main>/.claude/graphify/<checkout>/` and passes the arguments to graphify
unchanged. `<checkout>` is `main` in the main checkout and the worktree's directory name in a worktree, so each branch
has its own graph.

Never pass `--graph` or `--out`; `apex graph` already points graphify at the right place.

Everything under `.claude/graphify/` (graphs, dated snapshot directories, `.graphify_root`, `cache/`) belongs to
graphify. Never edit it; the one exception is deleting `graph.json` as the Errors table says.

## When to update

| Moment | Command |
|---|---|
| Session startup (handoff Startup step) | `apex graph update .` |
| Before the first read after code changed | `apex graph update .` |

`update` re-extracts code only, without an LLM. Run it every time; do not check whether the graph is stale first.

Ignore the lines `update` prints about running `/graphify --update` for docs, `Tip: set GEMINI_API_KEY ...`, and
`N file(s) not classified ... skipped`; none applies here.

## Who writes, who reads

| Role | May run |
|---|---|
| The session's Agent | `update` and every read |
| Subagents, Ultracode workers | reads only |

When dispatching a subagent or Ultracode worker, update the graph first and tell it to read through `apex graph`
and never run `update`.

## Reading

| Question | Command |
|---|---|
| What in the code relates to X | `apex graph query "<question>" --budget N` |
| One symbol and its neighbors | `apex graph explain "<symbol>"` |
| How A reaches B | `apex graph path "<A>" "<B>"` |
| What a change to X affects | `apex graph affected "<X>" --depth N` |

Every `query` passes `--budget`. Start around 1000 to 2000 tokens.

A `query` answer that starts with `[!] TRUNCATED` is missing nodes. Its hint may suggest a bigger `--budget` or
`context_filter`; narrow the question instead, with `--context` for the filter:

- ask about one symbol or one file, not a feature;
- add `--context call` (repeatable) to follow one edge kind;
- switch to `explain` for a single symbol, or `path` for a single connection.

`path` with an ambiguous endpoint prints `warning: source match was ambiguous` or `warning: target match was
ambiguous` and answers for the node it picked, which may be the wrong one; `No directed path found` from a wrong pick
is not a real negative. Rerun with `<path>::<symbol>` or the node id for that endpoint. The warning stays after the
rerun; check that the printed endpoints are the ones you meant.

`affected` has no budget. Keep it small with `--depth 1` and `--relation <R>` (repeatable, for example `calls`).

## Never

| Command | Why |
|---|---|
| bare `graphify`, `/graphify` | writes `graphify-out/` at the repo root |
| `hook install` | the Agent updates at defined moments, not on every commit |
| `claude install` | edits `CLAUDE.md` and adds a hook |
| `watch` | a background writer outside the session's control |
| `save-result`, `reflect` | graph memory overlaps `LIVE.md` |
| `extract` | LLM semantic extraction; only inside a ledger item created for it, and always with `--exclude .claude` (its `--exclude` replaces the stored excludes for that run) |

## Errors

`apex graph` exits with graphify's own code, or 128+N when graphify is killed by signal N. Exit 2 from `apex graph`
itself (a message starting `apex graph:`) is an environment problem: graphify missing from PATH, a worktree directory
named `main`, or two live worktrees sharing a directory name. Report it to the operator. Never fall back to bare
`graphify`.

A worktree's graph lasts as long as the worktree directory: every `apex graph` call deletes the graphs of worktrees
that no longer exist. A missing graph after a worktree is removed is expected.

| Symptom | Action |
|---|---|
| `update` exits 1 with `Refusing to overwrite ... Pass --force` | report it to the operator; do not force |
| `Cannot read .../graph.json ... Delete the file and run a full rebuild` from `update`, or a `json.decoder.JSONDecodeError` traceback from a read | delete `<main>/.claude/graphify/<checkout>/graph.json`, then `apex graph update .`; `--force` does not bypass this |
| a read fails with `graph file not found` | `apex graph update .`, then read again |
| `No node matching` | take the exact label from a `query` answer |
| `Ambiguous` | retry with `<path>::<symbol>` or the node id it lists |
| `affected` prints `No unique node match` | take the node id from `explain` and pass it to `affected` |
