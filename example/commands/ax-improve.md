---
description: Session retrospective. Mines this conversation (and session history) for friction, corrections, and misbehavior; cross-references installed Apex artifacts; walks findings one at a time so they can be turned into fixes or follow-ups.
---

<flow>
1. **Mine.** Scan the current session for: user corrections, repeated friction, points where Apex did the wrong thing or needed re-steering.
2. **Cross-reference.** Map each friction point to the artifact that should have prevented it (a skill trigger, an agent scope guard, a command step, the output style, a backbone check).
3. **Walk findings** one at a time: propose a concrete change (edit the artifact / file a follow-up via `apex followups add` / adjust config). User picks.
4. **Apply** approved changes. Summarize what changed and what was deferred.
</flow>

<notes>
Retrospective only — surfaces and fixes process drift. Does not implement feature work.
</notes>
