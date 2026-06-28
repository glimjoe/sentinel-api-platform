/**
 * Phase 5b — Demo GIF generator.
 *
 * Usage:
 *   1. Start the stack:  bash scripts/start_all.sh
 *   2. Seed demo data:   make seed
 *   3. Install tsx:      npm install -g tsx
 *   4. Run this script:  npx tsx scripts/demo-screenshots.ts
 *   5. Convert to GIF:   convert -delay 150 -loop 0 demo-*.png demo.gif
 *
 * Requires: npx playwright install chromium (once), ImageMagick.
 */

import { chromium } from 'playwright'

const BASE = 'http://localhost:5180'
const CREDS = { email: 'admin@sentinel.local', password: 'admin123' }

async function main() {
  const browser = await chromium.launch({ headless: true })
  const context = await browser.newContext({ viewport: { width: 1280, height: 800 } })
  const page = await context.newPage()

  // 1. Login page
  await page.goto(`${BASE}/login`)
  await page.screenshot({ path: 'demo-01-login.png', fullPage: true })
  await page.fill('input[type="email"]', CREDS.email)
  await page.fill('input[type="password"]', CREDS.password)
  await page.click('button[type="submit"]')
  await page.waitForURL('**/dashboard', { timeout: 10000 })

  // 2. Dashboard with ECharts
  await page.waitForTimeout(1500)
  await page.screenshot({ path: 'demo-02-dashboard.png', fullPage: true })

  // 3. Project list
  await page.goto(`${BASE}/projects`)
  await page.waitForTimeout(1000)
  await page.screenshot({ path: 'demo-03-projects.png', fullPage: true })

  // 4. Go back to dashboard (for loop GIF)
  await page.goto(`${BASE}/dashboard`)
  await page.waitForTimeout(1000)
  await page.screenshot({ path: 'demo-04-dashboard-again.png', fullPage: true })

  console.log('Screenshots saved as demo-*.png')
  console.log('To create GIF: convert -delay 150 -loop 0 demo-*.png demo.gif')

  await browser.close()
}

main().catch((err) => {
  console.error('Demo screenshot failed:', err)
  process.exit(1)
})
