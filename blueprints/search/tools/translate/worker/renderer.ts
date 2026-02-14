import puppeteer, { type Page } from '@cloudflare/puppeteer'
import type { Env } from './types'

/**
 * Detect whether a fetched response needs browser rendering.
 *
 * Returns true if the response looks like a Cloudflare challenge page,
 * a JS-only SPA shell, or any page that requires JavaScript to show content.
 */
export function needsBrowserRender(status: number, body: string): boolean {
  // Cloudflare managed challenge: 403 with challenge token
  if ((status === 403 || status === 503) && (body.includes('challenge-platform') || body.includes('_cf_chl'))) {
    return true
  }

  // "Just a moment..." title — Cloudflare challenge page
  if (body.includes('<title>Just a moment...</title>')) {
    return true
  }

  // Noscript message asking to enable JavaScript
  if (body.includes('<noscript>') && /enable\s+javascript/i.test(body)) {
    return true
  }

  // SPA shell with almost no text content
  if (body.length < 2048) {
    if (body.includes('id="root"') || body.includes('id="__next"') || body.includes('id="app"')) {
      return true
    }
  }

  return false
}

/**
 * Wait for a Cloudflare challenge to resolve by polling the page title.
 * CF sets <title>Just a moment...</title> during the challenge.
 */
async function waitForChallenge(page: Page, timeout: number): Promise<void> {
  const deadline = Date.now() + timeout
  while (Date.now() < deadline) {
    const title = await page.title()
    if (title !== 'Just a moment...') return
    await new Promise((r) => setTimeout(r, 500))
  }
}

/**
 * Render a URL using headless Chromium via Cloudflare Browser Rendering.
 * Handles Cloudflare challenges, SPAs, and JS-rendered content.
 * Retries up to 2 times on 429 (rate limit) with exponential backoff.
 */
export async function renderWithBrowser(env: Env, url: string): Promise<string> {
  const maxRetries = 2
  const delays = [2000, 4000]

  for (let attempt = 0; attempt <= maxRetries; attempt++) {
    try {
      console.log(`[renderer] attempt=${attempt} url=${url}`)
      return await launchAndRender(env, url)
    } catch (err) {
      const is429 = err instanceof Error && err.message.includes('429')
      console.log(`[renderer] FAIL attempt=${attempt} is429=${is429} err=${err instanceof Error ? err.message : err}`)
      if (!is429 || attempt >= maxRetries) throw err
      await new Promise((r) => setTimeout(r, delays[attempt]))
    }
  }
  // Unreachable, but TypeScript needs it
  throw new Error('Browser rendering failed after retries')
}

async function launchAndRender(env: Env, url: string): Promise<string> {
  const t0 = Date.now()
  console.log(`[renderer] LAUNCH url=${url}`)
  const browser = await puppeteer.launch(env.BROWSER)
  console.log(`[renderer] BROWSER_READY ms=${Date.now() - t0}`)
  try {
    const page = await browser.newPage()

    await page.setViewport({ width: 1280, height: 800 })

    // Navigate — use domcontentloaded to avoid waiting for ads/trackers
    await page.goto(url, { waitUntil: 'domcontentloaded', timeout: 25000 })
    console.log(`[renderer] NAVIGATED ms=${Date.now() - t0}`)

    // Wait for CF challenge to resolve (up to 15s)
    await waitForChallenge(page, 15000)
    console.log(`[renderer] CHALLENGE_DONE title="${await page.title()}" ms=${Date.now() - t0}`)

    // Wait for dynamic content to settle
    try {
      await page.waitForNetworkIdle({ idleTime: 500, timeout: 10000 })
    } catch {
      // Network never fully idles — that's fine, proceed with what we have
    }
    console.log(`[renderer] NETWORK_IDLE ms=${Date.now() - t0}`)

    // Small extra delay for late JS renders
    await new Promise((r) => setTimeout(r, 1000))

    const html = await page.content()
    console.log(`[renderer] CONTENT size=${html.length} ms=${Date.now() - t0}`)
    return html
  } finally {
    await browser.close()
  }
}
