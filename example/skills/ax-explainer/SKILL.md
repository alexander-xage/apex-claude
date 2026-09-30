---
name: ax-explainer
description: >
  Voice module for ENDURING human-facing narrative — README.md, docs/guides/, CHANGELOG
  narrative. Inverts the terse Apex output style: clear, expansive, detailed prose for readers
  who read start to finish. Runs a draft→expand chain: draft on Opus 4.8, expand/critique on
  Fable 5 (fallback Opus while Fable disabled), dispatched via the ax-writer agent. Auto-fires
  when editing human-facing markdown. Boundary: specs/designs/CLAUDE.md/signals stay terse.
---

<trigger>
"draft the README", "write the docs", "improve this prose", "edit the guide"; auto-fires when
editing README.md, docs/guides/*, or CHANGELOG narrative. Does NOT fire for specs, design docs,
CLAUDE.md, or signals files — those keep terse technical prose.
</trigger>

## Two voices
- **Claude's TUI replies** → Apex output style (terse, article-drop, fragments).
- **Enduring narrative docs** → this module (expansive, complete sentences, reader-first).
Everything else (specs, designs, CLAUDE.md, signals) → terse technical prose, neither voice.

## Voice rules (narrative)
- Clear, direct, technical. Complete sentences and connected paragraphs.
- No marketing language, no AI-tell phrases ("delve", "seamless", "in today's world"), no em dashes, no throat-clearing.
- Explain the why and the how, with enough context for a newcomer. Expansive where the terse style would compress.
- Ground every claim in the repo — read the code/spec before describing behavior. Do not invent.

## Draft→expand chain (via ax-writer agent)
1. **DRAFT** — dispatch ax-writer (model: Opus 4.8) for a complete, correct first draft.
2. **EXPAND/CRITIQUE** — re-dispatch ax-writer (model: Fable 5; fallback Opus while Fable disabled) to tighten flow, fill gaps, fix voice.
3. Orchestrator reviews against the repo and owns the final claim that the doc is accurate.

## Boundary
This module owns *narrative voice and the drafting chain*. Diff-driven surface impact and
surgical doc edits belong to `ax-documentation`, which delegates prose bodies here.
