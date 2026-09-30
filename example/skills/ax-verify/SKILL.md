---
name: ax-verify
description: >
  Evidence-before-claim gate for CODE claims. Auto-triggers when about to claim "done",
  "fixed", "passing", "complete", "ready to merge", "looks good", "should work", "green", or
  any synonym about code behavior. Iron rule: no completion claim without a fresh verification
  command run THIS turn. Orchestrator-wired — the builder emits an evidence block; the
  orchestrator runs the check and owns the "done" claim. Explicit: /ax-verify.
allowed-tools: Bash
---

<trigger>
Fires before any code-completion claim: "done", "fixed", "passing", "complete", "green",
"ready to merge", "looks good", "should work/pass". Code claims only — NOT research,
analysis, or design assertions (those are out of scope; this skill is about behavior).
</trigger>

## Iron rule
No "done/fixed/passing" for code without a verification command run in THIS turn whose output you show. Stale output from an earlier turn does not count. A claim without fresh evidence is a guess.

## Roles (orchestrator-wired)
- **Builder/surgeon** emit an *evidence block*: the command they ran + its result. They do not claim "done".
- **Orchestrator** re-runs the decisive check, reads the output, and is the only one that claims "done". Trust nothing it didn't reproduce.

## What counts as evidence
- The actual test/build/lint command and its real output (pass/fail summary, exit code).
- For a bug fix: the repro now passing AND the broader suite still green.
- Not evidence: "the change looks correct", "tests should pass", a description of what was done.

## On failure
Report the failure plainly with the output. "Completed" means nothing was skipped; "tests pass" means all tests ran. Surface the gap — never round a partial up to done.
