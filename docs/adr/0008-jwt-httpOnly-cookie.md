# ADR-0008: JWT → httpOnly cookie + CSRF (Phase 5 migration plan)

**Status:** Accepted (2026-06-27)
**Phase:** 5 (formalizes the migration promised in ADR-0004)

## Context

- ADR-0004 §31 committed: "Storage on the client: `localStorage` (Phase 1); migrate to `httpOnly` cookies in Phase 5".
- The current frontend store `frontend/src/stores/auth.ts` keeps `accessToken` and `refreshToken` in `localStorage`. This is XSS-readable.
- The Phase 1 code review (commit 828cd91, 2026-06-27) tightened session handling but did not address token storage; the architectural review (2026-06-27) called out the missing migration budget in Phase 5's 2-week estimate.
- The user is the only customer of this app (demo + portfolio) — there is no production user base to migrate without breaking.

Without this ADR, the Phase 5 schedule risks underestimating the cookie + CSRF + cross-domain + browser-test rework.

## Decision

In **Phase 5**, switch token storage to **httpOnly, Secure, SameSite=Lax cookies** with a CSRF token in a separate readable cookie:

- **Access token**: `Set-Cookie: sent_access=...; HttpOnly; Secure; SameSite=Lax; Path=/; Max-Age=900`
- **Refresh token**: `Set-Cookie: sent_refresh=...; HttpOnly; Secure; SameSite=Strict; Path=/api/v1/auth; Max-Age=604800`
- **CSRF token**: `Set-Cookie: sent_csrf=...; Secure; SameSite=Lax; Path=/; Max-Age=86400` (readable by JS for inclusion in `X-CSRF-Token` header on state-changing requests)
- **Backend middleware order** (per §6.2):
  1. `cors.go` — explicit allow-list, `Access-Control-Allow-Credentials: true` for the frontend origin
  2. `auth.go` — read access cookie, fall back to refresh cookie + rotation if expired
  3. `csrf.go` (new) — verify `X-CSRF-Token` against the `sent_csrf` cookie on all non-`GET/HEAD/OPTIONS` requests
- **Logout**: server-side endpoint `/api/v1/auth/logout` clears both cookies and revokes the refresh row in `refresh_tokens` (already supported by §8 + Phase 1 code).
- **Migration trigger**: this ADR is only enacted in Phase 5. Until then, ADR-0004's localStorage path stands.

## Consequences

**Positive**
- XSS can no longer exfiltrate the access token; impact of a frontend XSS drops to CSRF surface only.
- `SameSite=Lax` on access + CSRF gives reasonable CSRF defense for typical flows.
- Server-side logout becomes instant (cookie clearing + refresh revoke).

**Negative**
- Cross-origin deployments (frontend on a different domain than backend) require CORS allow-list + `credentials: 'include'` on every axios call. The plan §1 currently uses `localhost:5180` + `:8081` (same host); must be re-checked if the deploy ever moves.
- Playwright tests need `context.addCookies(...)` per test; this is a non-trivial rewrite of the existing 5 journeys. Budget 1 day.
- Two new failure modes to test: missing CSRF (rejected), expired refresh (cookie cleared by server, frontend redirects to login).

## Alternatives considered

- **Keep localStorage** — simplest, but the migration was already promised. Rejected.
- **Pure httpOnly + double-submit cookie (no separate CSRF cookie)** — common pattern but harder to debug; we already have a JWT for the access token and adding a third cookie doesn't hurt. Rejected (could be revisited).
- **Move to opaque session IDs + Redis store** — simpler cookies, no JWT at all; would invalidate ADR-0004. Rejected (out of scope for Phase 5's 2-week budget).

## Notes

- This ADR **does not change Phase 1–4 work**. It is a Phase 5 deliverable, scheduled under the 5b sub-phase (per plan §10 split: 5b = Playwright + CI + README + ai-workflow; cookie migration joins 5b).
- The frontend axios client (`frontend/src/api/client.ts`) currently has a 401-refresh interceptor using `localStorage`; that interceptor will be rewritten in Phase 5b to use `withCredentials: true` + cookie-based refresh, and a separate 403 interceptor for CSRF failures.
- If the schedule slips and Phase 5b cannot accommodate the migration, defer to a post-v1 patch release; document the deferral in `docs/ai-workflow/CHANGELOG.md`.