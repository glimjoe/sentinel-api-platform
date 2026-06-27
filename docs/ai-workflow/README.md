# AI workflow

> How this repo uses AI Coding tools (Claude Code, Copilot, etc.) as part of the development loop. This document is itself part of the deliverable — interviewers will read it.

## Why this exists

JD responsibility #4 (use AI Coding tools for decomposition, code gen, tests, review, refactor) and #6 (continuously accumulate AI-assisted R&D methodology) both demand a documented workflow, not ad-hoc usage. This file is the "how we do it" anchor.

## The loop

```
  ┌────────────────────────────────────────────────────┐
  │  1. PLAN      — write ADR or update existing        │
  │  2. PROMPT    — fill template from prompts/ dir     │
  │  3. GENERATE  — Claude Code / Copilot writes code   │
  │  4. REVIEW    — user reads, edits, rejects parts    │
  │  5. TEST      — write test, run, see it pass        │
  │  6. COMMIT    — trailer: AI-Assisted: <tool>        │
  │  7. REFLECT   — was the prompt good? update it       │
  └────────────────────────────────────────────────────┘
```

Steps 1, 4, 6, 7 are the user's. Steps 2, 3, 5 are AI-assisted.

## Prompt templates

Long-lived prompts are versioned in `prompts/` so they can be reviewed, improved, and shared:

| File | Use when… |
|---|---|
| `prompts/code-gen.md` | generating a new function / module |
| `prompts/test-gen.md` | generating unit / integration tests |
| `prompts/code-review.md` | asking AI to review a diff |
| `prompts/refactor.md` | restructuring without behavior change |
| `prompts/attribution.md` | failure-attribution LLM feature (Phase 4) |
| `prompts/completion.md` | test-case completion LLM feature (Phase 4) |
| `prompts/priority.md` | priority suggestion LLM feature (Phase 4) |

Each template follows a 5-section structure: **Goal · Context · Constraints · Output · Verification**.

## Commit convention

Every commit that includes AI-generated code must end with:

```
AI-Assisted: <tool>            # claude-code | copilot | cursor | qoder | manual
AI-Model: <model-id>           # e.g. claude-sonnet-4-6, gpt-4o
```

The `AI-Assisted` tag is non-negotiable. The model id is best-effort but encouraged — it lets the user show in interviews which tools they reach for.

## Session journaling

Significant AI-collaboration sessions get a record in `sessions/<date>-<slug>.md`. Example: `sessions/2026-06-27-plan.md` documents the conversation that produced the project plan.

Each session file includes:
- The original task in the user's words.
- Key prompts used.
- Decisions made.
- Files created / modified.
- Open follow-ups.

## What AI is *not* used for (deliberately)

- **Routing, retry, or any deterministic logic** (per user-global rule 5).
- **Generating the project's architecture** — the plan in `/.claude/plans/` is the source of truth; AI is the implementer, not the architect.
- **Commit messages** — the user writes the `subject:` and `body:`; AI may suggest wording only if asked.
- **Final review before merge** — the user always reads the AI-generated diff end-to-end.

## Metrics (post-launch)

When the project is past Phase 1, track in `metrics.md`:
- Lines committed by AI vs by hand (rough; the `AI-Assisted:` tag enables this)
- Acceptance rate (commits that pass first review / commits that need a follow-up fix)
- Prompt-template churn (how often a template changes)
