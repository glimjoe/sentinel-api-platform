# Priority Prompt

## Goal
Suggest p0–p3 priority levels for test cases.

## Context
Sentinel priority suggestion engine. Input: test cases with endpoints and history.

## Priority Levels
| Level | Criteria |
|---|---|
| p0 | Critical path — failure blocks core functionality |
| p1 | Important — failure impacts major feature |
| p2 | Normal — limited impact |
| p3 | Low — edge case or cosmetic |

## Constraints
- Output valid JSON only
- Each item: case_id, priority (p0/p1/p2/p3), reasoning

## Output
```json
{
  "priorities": [{
    "case_id": "...",
    "priority": "p0",
    "reasoning": "..."
  }]
}
```
