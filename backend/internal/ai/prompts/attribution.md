# Attribution Prompt

## Goal
Analyze a failed API test result and determine the root cause.

## Context
Sentinel contract-driven API testing platform. Input: test result with expected/actual status, body, assertion failures, duration.

## Constraints
- Output valid JSON only (no markdown, no preamble)
- Confidence must be 0.0–1.0
- Root cause: single sentence
- Analysis must cite specific assertion failures

## Output
```json
{
  "analysis": "<one-paragraph analysis>",
  "root_cause": "<concise root cause>",
  "confidence": <0.0–1.0>,
  "suggested_fix": "<optional fix suggestion>"
}
```
