# Prompt templates

Fill in each section. The more specific the Context and Constraints, the better the output.

## code-gen.md (template)

```markdown
## Goal
<one sentence>

## Context
- Files involved: <paths>
- Existing patterns: <link to similar code or ADR>
- Current state: <what already exists>

## Constraints
- Naming: <convention>
- Library version pins: <list>
- Must NOT touch: <files or boundaries>
- Style: <idioms to follow>

## Output
Expected signature: <Go func or Vue component shape>
Test names: <list of *_test.go functions to add>

## Verification
- `make test-unit` should pass
- `curl http://localhost:8081/<path>` returns <expected>
- Coverage of <package> should be ≥ 80%
```

## Other templates

To be filled in as the project grows:
- `test-gen.md` — generating unit / integration tests from a function signature
- `code-review.md` — asking AI to review a diff for correctness + style
- `refactor.md` — restructuring without behavior change (always pairs with the existing test suite as the safety net)
- `attribution.md`, `completion.md`, `priority.md` — AI-feature prompts (Phase 4)
