# Sentinel — Design Document

## C4 Context

```
User (Browser) → [Sentinel SPA :5180] → [Sentinel API :8081] → [MySQL :3307]
                                                       → [Redis :6380]
                                                       → [Target API]
                                                       → [AI Provider]
```

## MVP Data Flow

```mermaid
sequenceDiagram
    participant U as User
    participant F as Frontend (Vue)
    participant A as API Handler (Gin)
    participant S as Service
    participant R as Repository
    participant M as MySQL
    participant RD as Redis

    Note over U,RD: Phase 1 — Auth
    U->>F: Register / Login
    F->>A: POST /auth/register
    A->>S: Register(email, password)
    S->>R: Create(user)
    R->>M: INSERT users
    M-->>R: OK
    A-->>F: {user, access_token, refresh_token}

    Note over U,RD: Phase 2 — Contract Import
    U->>F: Import OpenAPI spec
    F->>A: POST /projects/:pid/apis/import-openapi
    A->>S: ImportOpenAPI(spec)
    S->>contract: Load + ExtractAPIs
    S->>R: Create(api) x N
    R->>M: INSERT apis
    A-->>F: {imported: N}

    Note over U,RD: Phase 2 — Mock Server
    F->>A: GET /mock/:slug/:path
    A->>mock.Engine: Serve
    mock.Engine->>R: FindBySlug + FindRules
    mock.Engine->>mock.Matcher: PickBest(rules, request)
    mock.Engine->>mock.Extractor: Apply(request, response)
    mock.Engine->>RD: HGET/HINCR (varbag)
    mock.Engine-->>A: matched response
    A-->>F: JSON + X-Mock-Session

    Note over U,RD: Phase 3 — Test Run
    U->>F: Create Run + Start
    F->>A: POST /projects/:pid/runs/:runId/start
    A->>S: Start(runID)
    S->>runner.Run: Run(ctx, run, cases, baseURL)
    loop Each TestCase
        runner->>Target: HTTP request
        Target-->>runner: HTTP response
        runner->>runner.Assert: status + body + assertions
        runner->>R: persister.Create(result)
        runner->>RD: pub.Publish(progress event)
    end
    RD-->>F: SSE stream
    runner-->>S: complete
    S->>R: updater.Update(run status)

    Note over U,RD: Phase 4 — AI Attribution
    F->>A: POST /projects/:pid/ai/attribution
    A->>S: Attribute(resultJSON)
    S->>ai.Engine: engine.call → guard.Allow → cache.Get → provider.Complete
    ai.Engine->>RD: cache.Set
    ai.Engine->>R: guard.Record(ai_usage)
    S-->>A: AttributionResult
    A-->>F: {analysis, root_cause, confidence}
```

## Error Code Table

| HTTP | Code | Constant | Sentinel |
|---|---|---|---|
| 400 | 40000 | `BAD_REQUEST` | `ErrBadRequest` |
| 401 | 40100 | `AUTH_INVALID_TOKEN` | `ErrInvalidToken` |
| 401 | 40101 | `AUTH_TOKEN_EXPIRED` | `ErrTokenExpired` |
| 401 | 40102 | `AUTH_INVALID_CREDENTIALS` | `ErrInvalidCredentials` |
| 403 | 40300 | `FORBIDDEN` | `ErrForbidden` |
| 404 | 40400 | `NOT_FOUND` | `ErrNotFound` |
| 409 | 40900 | `CONFLICT` | `ErrConflict` |
| 422 | 42200 | `MOCK_NO_MATCH` | (mock internal) |
| 429 | 42900 | `AI_*_BUDGET_EXCEEDED` | `ErrDailyBudgetExceeded` |
| 503 | 50300 | `AI_DISABLED` | `ErrAIDisabled` |
| 500 | 50000 | (internal) | (fallback) |
