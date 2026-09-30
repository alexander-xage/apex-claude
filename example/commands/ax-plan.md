---
description: Plan non-trivial work. Dispatches the ax-plan agent to research (online + codebase) and write a design doc + checkpoint-table spec in its own context. Trivial work gets an inline spec. Human approves before implementation. Subsumes evidence-gathering.
argument-hint: <task description or issue#>
---

<flow>
1. **Model ask.** Before dispatch, ask which model to run ax-plan on: **Fable 5** or **Opus 4.8** (pass as the Agent `model` override). Default Opus while Fable disabled.
2. **Triviality gauge.** Trivial (one obvious change, no design surface) → write an inline spec to `docs/spec/<topic>.md` directly, skip the agent.
3. **Dispatch ax-plan** (non-trivial). Brief = the task ($ARGUMENTS) + success criteria. The agent researches, decides, and writes `docs/design/<topic>.md` + `docs/spec/<topic>.md` in ITS context (keeps this cache clean). It may summon any subagent except ax-builder.
4. **Surface artifacts.** Show the design + spec paths, the decision, the evidence trail, and open questions.
5. **Approval gate.** Wait for the human to approve before any building. On approval, hand off to `/ax-implement`.
</flow>

<notes>
- ax-plan subsumes `/ax-gather-evidence` — hunches are chased to primary sources inside the plan.
- Spec body = current truth only; history lives in its `## Change log`.
- Pre-approval challenge available via `/ax-pressure-test`.
</notes>
