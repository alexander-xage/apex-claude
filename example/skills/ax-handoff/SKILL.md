---
name: ax-handoff
description: >
  Session continuity — capture a handoff at a stopping point, or resume from one. Auto-fires on
  scoped, object-bearing phrases: capture on "pause here and handoff", "stop session for later",
  "hand off the session", "write a handoff"; resume on "resume session", "resume the work",
  "resume our session", "pick up where we left off". Explicit: /ax-handoff and /ax-resume. Bare
  "resume" does NOT fire (too common). The apex binary owns the deterministic scan + staleness
  verdict; the model composes the narrative.
allowed-tools: Bash(apex handoff*), Bash(git status*), Bash(git log*), Bash(apex followups*), Bash(apex health*)
---

<trigger>
Capture: "pause here and handoff", "stop session for later", "hand off the session",
"write a handoff".
Resume: "resume session", "resume the work", "resume our session", "pick up where we left off".
Each phrase carries an explicit object — bare "resume" / "stop" / "pause" do NOT fire.
</trigger>

## Determinism boundary
- Binary (`apex handoff scan|status|archive`) owns: the deterministic fact table (`scan`, which is
  read-only and writes nothing), the staleness exit code (0 fresh / 1 absent / 2 stale), and the
  archive-on-consume move. It has no writer.
- Model owns: the handoff document itself — frontmatter copied from the scan output plus the intent
  narrative (why we stopped, what's next) — and the reconciliation judgment.

## Routing
- **Capture intent** → run the `/ax-handoff` flow (mode graceful|urgent; default graceful).
- **Resume intent** → run the `/ax-resume` flow (status → reconcile → confirm → archive).

## Guardrail (auto-fire only)
When this skill fires from a PHRASE (not an explicit `/ax-handoff` or `/ax-resume`), confirm before
any destructive step:
- capture that would overwrite an existing un-consumed handoff (archive it with
  `apex handoff archive` first — say so and get a nod before doing it);
- resume that would archive the active doc.
Explicit commands skip this confirm — typing the verb already states intent.

## Notes
- One active handoff at a time: `.claude/project/handoff.md`. Consumed docs archive to
  `.claude/project/handoffs/<id>.md`.
- Do not edit handoff frontmatter by hand — `head` is the staleness anchor.
