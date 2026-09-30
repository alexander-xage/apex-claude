---
name: ax-documentation
description: >
  Diff-driven documentation classifier. Reads the project's indexed doc surfaces, matches a
  diff (staged/branch/range) against them, emits proposed edits. Two modes: maintenance (commit
  flow — stale/incomplete only, never proposes new pages) and authoring (/ax-documentation —
  full discovery + gap detection + content generation). Emits CONFIDENCE per proposal. Scope =
  README + docs/ + docstrings/comments/API docs. Raw narrative drafting belongs to ax-explainer.
allowed-tools: Bash(git diff*), Bash(git status*), Read, Grep, Glob, Edit, Write
---

<trigger>
"doc this change", "doc impact", "what needs documenting", /ax-documentation; invoked by ship
verbs (maintenance); explicit `/ax-documentation` runs authoring.
</trigger>

## Scope
README + `docs/` + docstrings/comments/API docs. Narrative prose drafting (README intro, guide
body) is delegated to the **ax-explainer** module via the ax-writer agent — this skill owns
diff-to-surface impact and targeted edits to stale/incomplete docs.

## Modes
| Mode | When | Behavior |
|---|---|---|
| maintenance | commit flow / ship verbs | stale + incomplete surfaces only; NEVER proposes new pages |
| authoring | `/ax-documentation` | full discovery, gap detection, content generation, new pages OK |

## Workflow
1. Find the `## Documentation surfaces` table in CLAUDE instructions. If absent (maintenance): print `no documentation surfaces indexed.` and stop. Authoring: discover surfaces and offer to index them.
2. Read the diff (`git diff` scoped to the request). Match changed paths/symbols/domain-terms against each surface row's `Covers`.
3. For each matched surface, emit a proposal: `path — <why stale> → <one-sentence targeted edit>` + `CONFIDENCE: <0–100>`.
4. On approval, make the targeted edit and stage it. Prose-heavy bodies → hand to ax-explainer; this skill keeps the surgical edits.

## Rule
Maintenance mode never invents new pages — only fixes what the indexed surfaces already claim to cover. New surfaces are an authoring-mode decision the user makes.
