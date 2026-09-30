# Apex Claude — spine

This is the Apex spine: principles, the determinism boundary, the lifecycle, and the artifact
registries. It auto-loads in THIS repo (Apex dogfoods itself). Claude Code plugins cannot ship
an auto-loading CLAUDE.md, so to adopt Apex elsewhere, copy the spine sections below into your
project's `.claude/CLAUDE.md` (or `~/.claude/CLAUDE.md` for user-wide use). See the README.

## Principles

- Think before coding. State assumptions. Ask when uncertain. Stop when confused.
- Simplicity first. Minimum code. One abstraction per actual reuse.
- Surgical changes. Touch only what the task requires. Match existing style.
- Goal-driven. Define success criteria up front. Loop until verified.
- Verify before asserting. A factual claim about the codebase needs the tool call that proves it
  in the same turn. Hedging is not proof.
- Read before you write. Check exports, callers, shared utilities before changing them.
- Fail loud. "Done" means verified; "tests pass" means all tests ran. Surface gaps.

## Determinism boundary (core law)

The binary owns determinism; the model owns judgment.

- **Binary (`apex`)** — scan, lint, gate, detect, score. Deterministic, exit-code-driven, no LLM.
  Routing, status checks, staleness, and transforms that code can answer, code answers.
- **Model** — classification, drafting, summarization, extraction, design, the "is this right?" call.
- Exception: when the deterministic path itself is unreliable (a hook may be uninstalled, a binary
  absent), an LLM safeguard layer is acceptable defense-in-depth — name the exception when used.

## Orchestrator is the trusted authority

- The orchestrator (you, the main loop) owns final verification, fix-finding, and every gate decision.
- Subagents report evidence and findings UP. Only the orchestrator claims "done".
- Reviewers/doc-checks emit `CONFIDENCE: 0–100` + flag-only findings (severity 🟥🟧🟨 + 🟦 uncertain).
  The orchestrator aggregates confidence into repo health (`apex health`) and decides proceed/fix/block.

## Lifecycle

1. **Plan** — `/ax-plan` (design doc + checkpoint spec; trivial → inline spec). Gates: `/ax-pressure-test`.
2. **Implement** — `/ax-implement` runs the implement→review loop, commits per green iteration.
   `/ax-diagnose` for failure-driven work.
3. **Ship** — `/ax-ship [--push]` (review + commit), `/ax-pr` (push + PR), `/ax-merge [--squash]` (land on base).
4. **Sync docs** — `ax-documentation` skill (`/ax-documentation` for authoring). Ship verbs run maintenance mode automatically.
5. **Improve** — `/ax-improve` retrospective.

**Housekeeping:** `/ax-follow-up [review | remind <when> <what>]`, `/ax-refresh-signals` (first run
bootstraps the repo), `/ax-git-cleanup`.

**Session continuity:** `/ax-handoff [graceful|urgent]` captures a handoff at a stopping point;
`/ax-resume` consumes one (routes on `apex handoff status` — fresh/stale/absent). Binary scans the
deterministic state; the model composes the narrative.

**Autonomous shortcut:** `/ax-autopilot <task | issue#> [merge-verb]` runs plan → implement loop →
ship hands-off, with one human decision: how to merge.

## Two voices

- **TUI replies** → Apex output style (terse, article-drop, structural forms; tiers lite/full/ultra).
- **Enduring narrative docs** (README, docs/guides, CHANGELOG) → `ax-explainer` (expansive prose).
- **Everything else** (specs, designs, CLAUDE.md, signals) → terse technical prose.

## Specs: body is current truth, change log is history

`docs/spec/<topic>.md` is a contract read by fresh-context subagents as ground truth. The body must
always describe the *current* decision — never superseded content. Every spec ends with a
`## Change log`; superseding behavior rewrites the body and logs a `Superseded:` line.

## Registries

**Agents** (dispatch via Agent tool): `ax-investigator` (haiku, ro locator), `ax-builder` (sonnet,
TDD slice — orchestrator-only), `ax-reviewer` (sonnet, ro, CONFIDENCE), `ax-strategist` (opus, ro
reasoning), `ax-signals-inferrer` (sonnet, scoped rwx), `ax-git-scout` (sonnet, ro git hygiene),
`ax-haiku` (haiku, ro runner), `ax-debug` (opus, opt-in diagnose/fix), `ax-plan` (Fable 5 | Opus
per dispatch; summons any subagent except ax-builder), `ax-writer` (opus, prose drafter).

**Skills** (auto-fire or explicit): `ax-tdd`, `ax-commit`, `ax-verify`, `ax-review`,
`ax-documentation`, `ax-explainer`, `ax-handoff`.

**Backbone** (`apex <cmd>`): `signals scan|show|stale`, `health show|set`, `followups`, `reminder`,
`hooks session-start`, `doctor`, `validate`, `docs scan|stale`, `handoff scan|status|archive`,
`update [check] [--to vX.Y.Z]`.
Run `apex <cmd>` for usage.

<apex-signals>

## Project signals (auto-loaded)

@.claude/project/signals.md

</apex-signals>
