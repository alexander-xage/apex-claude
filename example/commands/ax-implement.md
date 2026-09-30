---
description: Orchestrate the implement→review subagent loop until the task is complete. Reads the approved spec, writes a thin brief to scratchpad, dispatches fresh-context ax-builder, gates each iteration on ax-reviewer CONFIDENCE, commits per green iteration, then syncs docs.
argument-hint: <spec path or topic>
---

<flow>
You are the orchestrator and the trusted authority. You own final verification, fix-finding, and the gate. Subagents report evidence up; only you claim "done".

1. **Read spec.** Load `docs/spec/<topic>.md` as ground truth. If none, route to `/ax-plan`.
2. **Brief.** Write a thin brief to `.claude/.scratchpad/<date>-<topic>/BRIEF.md` (spec pointer + this iteration's scope + any reviewer feedback). Keep an append-only `STATE.md`.
3. **Implement.** Pick the builder model, then dispatch `ax-builder` (or `ax-surgeon`-style for 1–2 files) with the iteration brief. It writes failing test first, then code; reports an evidence block.
   - **Model protocol.** Default is Sonnet. Before the FIRST builder dispatch of this work, ask the user once which model to use (Sonnet | other), recommending Sonnet. Never auto-select Opus or a higher tier — Opus+ is opt-in only, chosen explicitly by the user, never inferred from difficulty. The choice is sticky: once a model has been used for the builder, keep using it for the rest of this work without re-asking. If Sonnet was used before, stay on Sonnet.
4. **Verify yourself.** Re-run the decisive test (ax-verify discipline). Trust nothing you didn't reproduce.
5. **Review.** Dispatch `ax-reviewer` against the diff. Read findings + CONFIDENCE.
6. **Gate.** Blockers or low CONFIDENCE → YOU find the fix, fold it into the next iteration brief, loop to step 3. Clean + high confidence → continue.
7. **Commit** the green iteration (`ax-commit` message). Record health via `apex health set`.
8. **Loop** until the spec's checkpoints are all met. Then sync docs (the `ax-documentation` skill, maintenance mode) and clean the scratchpad.
</flow>

<notes>
- Address reviewer findings in-iteration where possible; defer only with the user's nod (file via `apex followups add`).
- The reviewer flags; the orchestrator fixes. Never let a subagent claim completion.
</notes>
