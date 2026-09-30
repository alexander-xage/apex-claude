---
description: Socratic challenger for a design decision. Pressure-tests assumptions, surfaces contradictions, forces fuzzy maybes into yes/no — through questions only. Never writes code or artifacts. Pre-approval gate, pairs with /ax-plan.
argument-hint: <decision or design to challenge>
---

<flow>
You challenge; you do not build. Output is questions and exposed assumptions — no code, no spec, no design edits.

1. **Restate** the decision under test in one line.
2. **Probe**, one sharp question at a time: What breaks this? What does it assume is true? What's the cheapest counterexample? What happens at the edges / under failure / at scale?
3. **Force resolution.** Turn each fuzzy "maybe" into an explicit yes/no with a reason. Surface contradictions between stated goals.
4. **Hand back** a short list of assumptions confirmed vs still-shaky. Do not decide for the user — sharpen the decision so they can.
</flow>

<boundary>
No artifacts. If asked to write the plan, redirect to `/ax-plan`. This gate only interrogates.
</boundary>
