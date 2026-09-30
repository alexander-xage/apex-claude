---
description: Capture a session handoff so a fresh-context session can resume the work. Binary reports the deterministic state; you write the document. Two modes — graceful (clean stop) | urgent (limits/time).
argument-hint: [graceful | urgent]
---

<flow>
The binary reports facts; you write the document. `apex handoff scan` is READ-ONLY — it never
touches `.claude/project/handoff.md`, so it is always safe to run.

1. **Mode.** `graceful` (default) when stopping at a clean boundary; `urgent` when cut short by
   context/time.

2. **Scan.** Run `apex handoff scan`. It prints a fact table: branch, head, dirty, staged,
   last-commit, followups, reminders, health, signals, brief.

3. **Write the doc.** Create `.claude/project/handoff.md` yourself — frontmatter and body.
   Frontmatter, in this key order, from the scan output:

   ```
   ---
   mode: graceful | urgent
   created: <RFC3339 UTC, now>
   branch: <scan "branch">
   head: <scan "head">
   health: <scan "health", or -1 when unset>
   status: open
   ---
   ```

   `head` is the staleness anchor `/ax-resume` routes on — copy it from the scan output verbatim,
   never from memory. If an un-consumed `handoff.md` already exists, run `apex handoff archive`
   first so nothing is silently overwritten.

   Then the body, from THIS session — the part the binary can't know:
   - graceful → **Shipped** (what landed) / **Outcome** (decisions, locked choices) /
     **Next** (the very next action) / **Open threads** (loose ends).
   - urgent → **Cursor** (exact stopping point) / **Uncommitted** (in-flight edits) /
     **Resume here** (first action on return) / **Blockers**.

   Fold the scan's remaining facts into the body where they carry meaning — a dirty tree belongs in
   **Uncommitted**, open followups and due reminders in **Open threads**, stale signals or an active
   `BRIEF.md` in **Next**. Be concrete: name files, commands, decisions. Convert relative dates to
   absolute.

4. **Report.** Print the doc path and a one-line summary of what's captured.
</flow>

<notes>
- This is the capture verb. To consume a handoff, use `/ax-resume`.
- The doc stays `status: open` until `/ax-resume` archives it on accept.
- The binary has no writer: `apex handoff` is `scan` (report) | `status` (route) | `archive` (move).
</notes>
