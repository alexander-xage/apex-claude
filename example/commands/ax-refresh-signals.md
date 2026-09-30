---
description: Refresh project signals on demand; on first run, also bootstraps the repo for Apex. Dispatches the ax-signals-inferrer agent to scan, infer domains, write signals.md, and wire the @-ref into CLAUDE.md.
---

<flow>
1. **Staleness check.** `apex signals stale`. Exit 0 (fresh) → report and stop unless the user forces (the fingerprint only covers manifests + top-level dirs, so force after large internal changes). Exit 1 (stale) → refresh. No signals file → first run: do step 2 first.
2. **Bootstrap (first run only).** Audit and propose only what's missing — never overwrite:
   - `.gitignore` entries for regenerable state: `bin/`, `.claude/project/deterministic-signals.md`, `.claude/project/health.md`, `.claude/project/doc-surfaces.md`, `.claude/project/handoff.md`, `.claude/project/handoffs/`, `.claude/.scratchpad/`, `.worktrees/`.
   - `docs/design/` and `docs/spec/` directories.
   Apply on approval.
3. **Dispatch ax-signals-inferrer.** It runs `apex signals scan`, infers domains, authors `.claude/project/signals.md`, and wires `@.claude/project/signals.md` into CLAUDE.md if absent.
4. **Verify.** Spot-check the inferred claims against the tree, confirm `apex signals stale` exits 0, and `apex doctor`. Stage `signals.md` (and per-domain files) if the user is committing. No commit here.
</flow>

<notes>
The deterministic scan is authoritative — the agent never hand-edits `deterministic-signals.md`, only regenerates it. Ship verbs dispatch this silently when signals go stale.
</notes>
