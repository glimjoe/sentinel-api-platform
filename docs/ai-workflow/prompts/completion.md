# Completion Prompt

## Goal
Generate new test cases from API specs to improve coverage.

## Context
Sentinel test case generation engine. Input: OpenAPI-based specs + existing test cases. Cover: happy paths, edge cases, error responses, boundary values.

## Constraints
- Output valid JSON only
- Each case: name, method (GET/POST/PUT/PATCH/DELETE), path, expected_status
- expected_body_match: exact/contains/jsonpath/schema/regex/none
- Do not duplicate existing test cases

## Output
```json
{
  "test_cases": [{
    "name": "...",
    "method": "GET",
    "path": "/api/v1/resource",
    "expected_status": 200,
    "expected_body_match": "contains",
    "expected_body_pattern": "success"
  }]
}
```
