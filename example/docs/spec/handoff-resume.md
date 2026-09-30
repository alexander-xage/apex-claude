# Spec — handoff / resume

Binary-backed session continuity. `apex handoff` owns the deterministic fact table + staleness
exit codes; `/ax-handoff`, `/ax-resume`, and the `ax-handoff` skill write the document. Design rationale and
pressure-test conclusions: `docs/design/handoff-resume.md`.

## Contract

- The binary has **no writer**. `apex handoff` is `scan` (report) | `status` (route) | `archive`
  (move). The model authors `.claude/project/handoff.md`; the binary only reads and moves it.
- Single active doc: `.claude/project/handoff.md`. Frontmatter via `internal/fm`, ordered keys:
  `mode` (graceful|urgent), `created` (RFC3339), `branch`, `head` (short sha), `health`,
  `status` (open|consumed). The model copies `branch`/`head`/`health` verbatim from scan output;
  `head` remains the staleness anchor even though the model transcribes it.
- On consume, archive to `.claude/project/handoffs/<id>.md`, `id` = `%03d` next over that dir
  (mirrors followups CLOSED archive).
- Before writing over an un-consumed active doc, the model runs `apex handoff archive` first —
  nothing is silently lost. (Previously enforced by a binary `Write`; now a command-flow step.)
- `apex handoff scan` is ONE **read-only** reporter serving both capture (input the model composes
  from) and recovery (rescan with no doc). It writes nothing, so it is safe to run mid-consume.
  Reports, one `key: value` row each: git branch / head / dirty / staged, last commit, open
  followups, due reminders, health score, signals staleness, active scratchpad `BRIEF.md`.
  Every field of `State` must appear in the report — a scanned-but-unreported field is a defect.
- `apex handoff status` exit codes: `0` present+fresh, `2` present+stale (live HEAD != recorded
  `head`), `1` absent. Branch drift = human-readable note, not a code.
- Two modes — graceful sections: Shipped / Outcome / Next / Open threads; urgent sections:
  Cursor / Uncommitted / Resume here / Blockers.
- Auto-fire phrases object-bearing only; bare "resume" excluded. Skill auto-fire confirms before
  destructive ops; explicit commands skip confirm.

## Checkpoints

| # | Checkpoint | File(s) | Verify |
|---|---|---|---|
| 1 | `internal/handoff` package: `State` struct + `Scan(root) (State, error)` capturing git branch/HEAD/dirty/staged + last commit via isolated `git*` helpers | `internal/handoff/handoff.go` | `go test ./internal/handoff` — Scan populates git fields in a temp repo |
| 2 | Scan extends to non-git sources: open followups, due reminders, health score, signals staleness, active `BRIEF.md` (library calls, not re-shell) | `internal/handoff/handoff.go` | unit test: seeded state dir → Scan returns counts/score |
| 3 | `Report(State) string` renders every `State` field as a `key: value` fact table for stdout. No frontmatter, no sections, no mode — the model composes those | `internal/handoff/handoff.go` | test: every scanned field appears in the report; zero-value State renders `(none)`/`unset` |
| 4 | `Path(root)`, `Status(root) int` (exit 0/1/2 per staleness), `Archive(root) (id, error)` (move active → `handoffs/<id>.md`, set `status: consumed`, preserve the model's body) | `internal/handoff/handoff.go` | tests: Status returns 1 absent / 0 fresh / 2 after HEAD change; Archive moves file, assigns `%03d`, keeps the narrative |
| 5 | `cmd/apex/cmd_handoff.go`: `register("handoff",…)` → `switch args[0]` over `scan\|status\|archive`, exit-code returns, usage on unknown sub. `scan` takes no mode argument and writes nothing | `cmd/apex/cmd_handoff.go` | `go build ./...`; `apex handoff status` exits 1 in a repo with no handoff; `apex handoff scan` leaves no `handoff.md` |
| 6 | `commands/ax-handoff.md` (argument-hint `[graceful|urgent]`): scan → model writes frontmatter + mode sections → active doc | `commands/ax-handoff.md` | `apex validate` passes; `/ax-handoff` listed |
| 7 | `commands/ax-resume.md`: `handoff status` → route 0/1/2 → reconcile against `handoff scan` (read-only, safe mid-consume) → confirm → `handoff archive` on accept | `commands/ax-resume.md` | `apex validate` passes |
| 8 | `skills/ax-handoff/SKILL.md`: scoped auto-fire phrases (object-bearing; bare "resume" excluded), destructive-confirm guardrail on auto-fire | `skills/ax-handoff/SKILL.md` | `apex validate` passes; skill discoverable |
| 9 | Roster wiring: add handoff/resume to CLAUDE.md command roster + `.claude/project/signals.md` plugin signals | `CLAUDE.md`, `.claude/project/signals.md` | grep shows entries; `apex validate` clean |
| 10 | Full gate: `make fmt && make vet && make test && make build && ./bin/apex doctor && ./bin/apex validate` all green | — | all commands exit 0 |

Doctor/validate scan command + skill dirs by glob — no code edit needed there for checkpoints 6–8.

## Change log

- 2026-06-19 — Initial spec. Strategy B (binary-backed). Staleness = sha-inequality → exit 2
  (ancestry math rejected). Auto-fire phrases object-bearing, bare "resume" excluded.
- 2026-06-19 — Approval-gate decisions locked: `handoff write` archives an existing un-consumed
  active doc before writing (no silent overwrite); `head` = short sha; adopt the recommended
  object-bearing phrase additions; destructive-confirm guardrail on skill auto-fire only.
- 2026-09-12 — Superseded: `Render` and `Write` removed. The binary no longer writes the handoff
  document — `apex handoff scan` is a read-only fact reporter (`Report(State) string`) and the model
  composes `handoff.md` end-to-end, frontmatter included. Motivation: `Render` emitted only 5 of the
  13 scanned fields, silently discarding dirty/staged/last-commit/followups/reminders/signals/brief,
  and the split writer meant the doc's contract lived in two places. Consequence accepted: `head` is
  now model-transcribed, so the staleness anchor depends on the model copying scan output verbatim;
  `scan` gains the property of being safe to run while a doc is being consumed. Archive-before-
  overwrite moves from binary enforcement to a command-flow step.
- 2026-06-19 — Checkpoints 1–4 implemented. `Render`/`Write` take an injected `now time.Time` for a
  deterministic `created` stamp (refinement during impl; body updated to current truth).
