---
description: Resume work from a saved handoff. Routes on `apex handoff status` exit code — fresh, stale, or absent — reconciles the recorded plan against current reality, then archives the doc on accept.
---

<flow>
The binary owns the present/fresh/stale/absent verdict and the fact table; you own the
reconciliation and the go-ahead. Every `apex handoff` verb except `archive` is read-only.

1. **Status.** Run `apex handoff status` and read the exit code (the routing signal):

   | Code | Meaning | Route |
   |---|---|---|
   | `0` | handoff present, fresh (`head` == live HEAD) | step 2 |
   | `2` | handoff present, STALE (HEAD moved past `head`) | step 2, with a drift warning |
   | `1` | no active handoff | step 4 |

2. **Read + reconcile (codes 0 / 2).** Read `.claude/project/handoff.md`. Reconcile its recorded
   state against reality with `apex handoff scan` — it is read-only and prints the same fact table
   the doc was captured from, so a field-by-field diff surfaces every drift at once. Widen with
   `git status --short`, `git log --oneline` as needed. For code `2`, lead with the drift: HEAD
   moved since capture, so call out what changed (new commits, branch switch) before trusting the
   "Next" step.

3. **Present + confirm.** Summarize: where the work stopped, the next action, open threads, and any
   reconciliation deltas. Wait for the user to confirm the resume point.

4. **Absent (code 1).** No handoff. Offer three routes — let the user pick:
   - **rescan-reconstruct** — `apex handoff scan` reports current git + followups + reminders +
     health + signals; reconstruct the narrative from that plus history, then present as in step 3.
     No doc is written on this route — you are resuming, not capturing.
   - **tell me where** — the user points at the resume context directly.
   - **start fresh** — no resume; proceed as a new session.

5. **Archive on accept.** Once the user confirms the resume point (codes 0/2), run
   `apex handoff archive` to move the consumed doc to `.claude/project/handoffs/<id>.md`
   (`status: consumed`). Then continue the work.
</flow>

<notes>
- Capture is `/ax-handoff`; this is the consume verb.
- Archive only after the user confirms — never discard an unconsumed handoff silently.
</notes>
