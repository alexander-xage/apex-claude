---
description: Failure-driven investigation. Two modes — ci (a failed CI run seeds the brief) and bug (a freeform symptom paragraph seeds it). Dispatches the opt-in ax-debug agent for hypothesis-driven root-cause + repro test, loops fix→review until green.
argument-hint: ci <run-url> | bug <symptom>
---

<flow>
1. **Seed the brief.** Mode `ci`: pull the failed run's logs (gh / project CI CLI) as the symptom. Mode `bug`: take the symptom paragraph ($ARGUMENTS).
2. **Dispatch ax-debug** (opus, opt-in). Brief = symptom + repro context + mode (`diagnose-only` or `diagnose+fix`). It builds a symptom→hypothesis→cheapest-test chain to root cause and lands a failing repro test.
3. **Verify root cause yourself.** Reproduce the failing test before accepting the diagnosis.
4. **Fix loop.** If diagnose+fix: review the fix via `ax-reviewer` (CONFIDENCE gate), confirm repro now passes + broader suite green. You own the "fixed" claim.
5. **Commit** the fix + repro test (`ax-commit`). Record health.
</flow>

<notes>
ax-debug is the opt-in path for stuck/broken states — not routine implementation. For spec-driven work use `/ax-implement`.
</notes>
