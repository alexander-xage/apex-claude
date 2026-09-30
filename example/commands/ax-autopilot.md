---
description: Autonomous end-to-end delivery — plan, the implement→review loop, then ship. Hands-off except one decision: how to merge. Always uses the subagent loop, addresses every reviewer finding in-iteration (nothing deferred), keeps the spec currency-clean.
argument-hint: <task | issue#> [merge-verb]
---

<flow>
1. **Merge decision.** If a merge verb wasn't given ($ARGUMENTS), ask once: how to land it (`/ax-merge` | `/ax-merge --squash` | `/ax-pr`). This is the only human gate.
2. **Plan.** Run `/ax-plan` flow → design + spec. No approval pause in autopilot; proceed once the spec is written and currency-clean.
3. **Implement.** Run the `/ax-implement` loop. Autopilot is hands-off: pin the builder to Sonnet and never auto-escalate to Opus or higher — skip the model question (no human gate beyond merge). Address EVERY reviewer finding in-iteration — defer nothing. If stuck, dispatch `ax-strategist` (read-only) for root-cause reasoning before retrying.
4. **Ship.** Run the chosen merge verb. Re-run tests on the merged/squashed tip.
5. **Report.** What was built, what was verified (commands + results), where it landed.
</flow>

<guardrails>
- You remain the trusted authority: own final verification and the "done" claim end to end.
- Keep the spec body = current truth so fresh subagents can't be diverted.
- Stop and surface to the user on irreversible-action ambiguity or repeated unexplained failure.
</guardrails>
