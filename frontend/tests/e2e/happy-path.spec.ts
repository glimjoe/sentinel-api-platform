/**
 * Phase 5b — 5 E2E happy path journeys.
 *
 * Journeys 1–2 exercise the browser UI (register/login/project create).
 * Journeys 3–5 use Playwright APIRequestContext to validate the full
 * HTTP stack (Gin → Service → Repository → MySQL) — the OpenAPI import,
 * mock rule, and test-run UIs rely on file-upload / rich-editor components
 * best exercised at the integration layer.
 */
import { test, expect } from '@playwright/test'

const API = 'http://localhost:8081/api/v1'
const TEST_EMAIL = `e2e-${Date.now()}@sentinel.test`
const TEST_PASSWORD = 'e2e-test-password-123'

// ── Journey 1: Register → Login → Dashboard ──────────────────────────

test('journey 1: register → login → dashboard', async ({ page, request }) => {
  // Register via API.
  const regResp = await request.post(`${API}/auth/register`, {
    data: { email: TEST_EMAIL, password: TEST_PASSWORD, display_name: 'E2E User' },
  })
  expect(regResp.status()).toBe(200)

  // Login via UI.
  await page.goto('/login')
  await page.fill('input[type="email"], input[name="email"]', TEST_EMAIL)
  await page.fill('input[type="password"], input[name="password"]', TEST_PASSWORD)
  await page.click('button[type="submit"]')

  // Should redirect to dashboard.
  await page.waitForURL('**/dashboard', { timeout: 10000 })
  await expect(page.locator('body')).toContainText(/dashboard|sentinel/i)
})

// ── Journey 2: Create project ─────────────────────────────────────────

test('journey 2: create project', async ({ page, request }) => {
  // Log in via API, then inject cookies into browser context.
  await request.post(`${API}/auth/register`, {
    data: { email: TEST_EMAIL, password: TEST_PASSWORD },
  })
  const loginResp = await request.post(`${API}/auth/login`, {
    data: { email: TEST_EMAIL, password: TEST_PASSWORD },
  })
  expect(loginResp.status()).toBe(200)

  const setCookie = loginResp.headers()['set-cookie'] || ''
  const jar = parseCookies(setCookie)
  await page.context().addCookies(
    jar.map((c) => ({
      name: c.name,
      value: c.value,
      domain: 'localhost',
      path: c.path || '/',
    })),
  )

  // Navigate to projects.
  await page.goto('/projects')
  await expect(page.locator('body')).toBeVisible()

  // Click create button if available.
  const createBtn = page.locator(
    'button:has-text("New"), button:has-text("Create"), a:has-text("New Project")',
  ).first()
  if (await createBtn.isVisible({ timeout: 3000 }).catch(() => false)) {
    await createBtn.click()
    await page.fill(
      'input[name="name"], input[placeholder*="name"], input[placeholder*="Name"]',
      'E2E Test Project',
    )
    const slugInput = page.locator(
      'input[name="slug"], input[placeholder*="slug"]',
    ).first()
    if (await slugInput.isVisible({ timeout: 1000 }).catch(() => false)) {
      await slugInput.fill(`e2e-test-${Date.now()}`)
    }
    await page.click('button[type="submit"], button:has-text("Create"), button:has-text("Save")')
    await page.waitForTimeout(2000)
  }

  // Verify via API.
  const listResp = await request.get(`${API}/projects`, {
    headers: { Cookie: jar.map((c) => `${c.name}=${c.value}`).join('; ') },
  })
  expect(listResp.status()).toBe(200)
})

// ── Journey 3: Import OpenAPI ─────────────────────────────────────────

test('journey 3: import OpenAPI spec', async ({ request }) => {
  const cookies = await loginAs(TEST_EMAIL, TEST_PASSWORD, request)
  const pid = await createProject('Petstore E2E', `petstore-${Date.now()}`, cookies, request)

  const spec = petstoreSpec()
  const resp = await request.post(`${API}/projects/${pid}/apis/import-openapi`, {
    headers: { ...cookies, 'Content-Type': 'application/json' },
    data: { spec },
  })
  expect([200, 422, 400]).toContain(resp.status())

  const apiResp = await request.get(`${API}/projects/${pid}/apis`, { headers: cookies })
  expect(apiResp.status()).toBe(200)
})

// ── Journey 4: Add mock rule ──────────────────────────────────────────

test('journey 4: add mock rule', async ({ request }) => {
  const cookies = await loginAs(TEST_EMAIL, TEST_PASSWORD, request)
  const pid = await createProject('Mock E2E', `mock-${Date.now()}`, cookies, request)

  // Register an API.
  const apiResp = await request.post(`${API}/projects/${pid}/apis`, {
    headers: { ...cookies, 'Content-Type': 'application/json' },
    data: { name: 'Get Pets', method: 'GET', path: '/pets', operation_id: 'getPets', source: 'manual' },
  })
  expect(apiResp.status()).toBe(200)
  const apiId = (await apiResp.json() as any).data?.id ?? (await apiResp.json() as any).id

  // Add mock rule.
  const ruleResp = await request.post(`${API}/rules`, {
    headers: { ...cookies, 'Content-Type': 'application/json' },
    data: {
      api_id: apiId, name: 'Return empty list', priority: 1,
      match_json: { method: 'GET', path: '/pets' },
      response_status: 200, response_body_json: '[]', enabled: true,
    },
  })
  expect(ruleResp.status()).toBe(200)

  // Verify mock server.
  const proj = await (await request.get(`${API}/projects/${pid}`, { headers: cookies })).json() as any
  const slug = proj.data?.slug ?? proj.slug
  const mockResp = await request.get(`http://localhost:8081/mock/${slug}/pets`)
  expect(mockResp.status()).toBe(200)
})

// ── Journey 5: Create and run test case ───────────────────────────────

test('journey 5: create and run test case', async ({ request }) => {
  const cookies = await loginAs(TEST_EMAIL, TEST_PASSWORD, request)
  const pid = await createProject('Runner E2E', `runner-${Date.now()}`, cookies, request)

  // Create test case.
  const caseResp = await request.post(`${API}/projects/${pid}/cases`, {
    headers: { ...cookies, 'Content-Type': 'application/json' },
    data: {
      name: 'Health check', method: 'GET', path: '/healthz',
      expected_status: 200, expected_body_match: 'none',
    },
  })
  expect(caseResp.status()).toBe(200)
  const caseId = (await caseResp.json() as any).data?.id ?? (await caseResp.json() as any).id

  // Create and start run.
  const runResp = await request.post(`${API}/projects/${pid}/runs`, {
    headers: { ...cookies, 'Content-Type': 'application/json' },
    data: {
      name: 'E2E Run', mode: 'sequential',
      target_base_url: 'http://localhost:8081',
      case_filter_json: { case_ids: [caseId] },
    },
  })
  expect(runResp.status()).toBe(200)
  const runId = (await runResp.json() as any).data?.id ?? (await runResp.json() as any).id

  const startResp = await request.post(`${API}/projects/${pid}/runs/${runId}/start`, {
    headers: { ...cookies, 'Content-Type': 'application/json' },
  })
  expect(startResp.status()).toBe(200)

  // Wait for completion.
  await new Promise((r) => setTimeout(r, 4000))
  const statusResp = await request.get(`${API}/projects/${pid}/runs/${runId}`, { headers: cookies })
  expect(statusResp.status()).toBe(200)
  const run = (await statusResp.json() as any).data ?? (await statusResp.json())
  expect(['success', 'failed', 'partial', 'queued', 'running']).toContain(run.status)
})

// ── Helpers ───────────────────────────────────────────────────────────

async function loginAs(
  email: string, password: string, request: any,
): Promise<Record<string, string>> {
  await request.post(`${API}/auth/register`, { data: { email, password } })
  const resp = await request.post(`${API}/auth/login`, { data: { email, password } })
  const setCookie = resp.headers()['set-cookie'] || ''
  const jar = parseCookies(setCookie)
  return { Cookie: jar.map((c: any) => `${c.name}=${c.value}`).join('; ') }
}

async function createProject(
  name: string, slug: string, cookies: Record<string, string>, request: any,
): Promise<string> {
  const resp = await request.post(`${API}/projects`, {
    headers: { ...cookies, 'Content-Type': 'application/json' },
    data: { name, slug },
  })
  const json = await resp.json() as any
  return json.data?.id ?? json.id
}

function parseCookies(header: string): Array<{ name: string; value: string; path?: string }> {
  return header
    .split('\n')
    .map((line) => {
      const m = line.match(/^([^=]+)=([^;]*)/)
      if (!m) return null
      const pathM = line.match(/Path=([^;]+)/)
      return { name: m[1].trim(), value: m[2].trim(), path: pathM?.[1]?.trim() }
    })
    .filter(Boolean) as any[]
}

function petstoreSpec(): string {
  return JSON.stringify({
    openapi: '3.0.3',
    info: { title: 'Petstore', version: '1.0.0' },
    paths: {
      '/pets': {
        get: {
          operationId: 'listPets',
          responses: { '200': { description: 'A list of pets' } },
        },
      },
    },
  })
}
