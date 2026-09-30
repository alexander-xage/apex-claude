---
description: Review the current diff, then commit it. Dispatches ax-reviewer; gates the commit on its CONFIDENCE; uses the ax-commit skill for the message. --push also pushes after the commit.
argument-hint: [--push]
---

<ship-flow>
Run the change through review before it lands.

1. **Pre-flight.** Confirm there are changes: `git status --short`. If clean, stop and say so.

2. **Review.** Dispatch the `ax-reviewer` agent against the working diff. Pass the task
   context as the brief. Wait for its findings + `CONFIDENCE:` line.

3. **Gate (orchestrator decides — you own this call).**
   - Any 🟥 blocker, OR `CONFIDENCE` below ~70 → surface findings, stop, do NOT commit. Offer
     to fix (dispatch `ax-builder`) or let the user decide. You find the fix; the reviewer only flags.
   - No blockers and high confidence → continue.

4. **Stage + message.** Stage the intended files by path (`git add <path>`). Invoke the
   `ax-commit` skill to generate the message from the staged diff.

5. **Commit.** Commit with that message. Report the resulting commit hash.
6. **Health.** Record the review outcome via `apex health set` from the aggregated CONFIDENCE.
7. **Push (`--push` only).** `git push` (`-u` if the branch has no upstream). On the base branch,
   confirm first unless the user has already said to push base. Report the remote ref.
</ship-flow>

<safety>
- Never commit secrets — if review flags any, abort regardless of confidence.
- Never `git add -A` blindly; stage only files relevant to this change.
- Pushing publishes to the remote; it may be cached or indexed.
</safety>
