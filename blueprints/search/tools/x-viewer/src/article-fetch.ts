import { launch } from '@cloudflare/playwright'
import type { Env } from './types'

/**
 * Fetch an X Article body using Cloudflare Browser Rendering (Playwright).
 * Launches headless Chrome, injects auth cookies, navigates to the article,
 * waits for the React app to render, then extracts structured content from the DOM.
 */
export async function fetchArticleBody(env: Env, articleURL: string): Promise<string> {
  let browser
  try {
    browser = await launch(env.BROWSER)
    const page = await browser.newPage()

    // Set auth cookies before navigation
    if (env.X_AUTH_TOKEN && env.X_CT0) {
      await page.context().addCookies([
        { name: 'auth_token', value: env.X_AUTH_TOKEN, domain: '.x.com', path: '/' },
        { name: 'ct0', value: env.X_CT0, domain: '.x.com', path: '/' },
      ])
    }

    await page.goto(articleURL, { waitUntil: 'networkidle', timeout: 30_000 })

    // Wait for the article content to render (up to 15s)
    try {
      await page.waitForSelector('[data-testid="twitterArticleRichTextView"]', { timeout: 15_000 })
    } catch {
      // Element didn't appear
    }

    // Extract article content from the DOM
    const extracted = await page.evaluate(() => {
      const titleEl = document.querySelector('[data-testid="twitter-article-title"]')
      const bodyEl = document.querySelector('[data-testid="twitterArticleReadView"]')
        || document.querySelector('[data-testid="twitterArticleRichTextView"]')
      if (!bodyEl || bodyEl.innerText.length < 100) return ''

      const lines: string[] = []

      function processNode(node: Element) {
        const ds = (node as HTMLElement).dataset || {}

        // Code blocks
        if (ds.testid === 'markdown-code-block') {
          const langEl = node.querySelector('span')
          const lang = langEl ? langEl.textContent?.trim() || '' : ''
          let code = (node as HTMLElement).innerText.trim()
          if (lang && code.startsWith(lang)) code = code.slice(lang.length).trim()
          if (code.startsWith('Copy')) code = code.slice(4).trim()
          if (code) {
            lines.push('')
            lines.push('```' + lang)
            lines.push(code)
            lines.push('```')
            lines.push('')
          }
          return
        }

        // Images
        if (ds.testid === 'tweetPhoto') {
          const img = node.querySelector('img[src]') as HTMLImageElement | null
          if (img) {
            const src = img.src.replace(/name=small/, 'name=large')
            lines.push('')
            lines.push('![Image](' + src + ')')
            lines.push('')
          }
          return
        }

        // Headers
        if (/^H[123]$/.test(node.tagName)) {
          const level = node.tagName[1]
          const prefix = '#'.repeat(parseInt(level)) + ' '
          lines.push('')
          lines.push(prefix + (node as HTMLElement).innerText.trim())
          lines.push('')
          return
        }

        // Text blocks
        if (node.classList?.contains('longform-unstyled') && ds.block === 'true') {
          const text = processInlineContent(node)
          if (text.trim()) {
            lines.push('')
            lines.push(text.trim())
          }
          return
        }

        // List items
        if (node.classList?.contains('longform-ordered-list-item') || node.classList?.contains('longform-unordered-list-item')) {
          const text = processInlineContent(node)
          const prefix = node.classList.contains('longform-ordered-list-item') ? '1. ' : '- '
          if (text.trim()) lines.push(prefix + text.trim())
          return
        }

        // Blockquotes
        if (node.classList?.contains('longform-blockquote')) {
          const text = (node as HTMLElement).innerText.trim()
          if (text) {
            lines.push('')
            lines.push('> ' + text.replace(/\n/g, '\n> '))
            lines.push('')
          }
          return
        }

        for (const child of node.children || []) {
          processNode(child as Element)
        }
      }

      function processInlineContent(node: Element): string {
        let result = ''
        for (const child of node.querySelectorAll('[data-text="true"]')) {
          result += child.textContent
        }
        if (!result) result = (node as HTMLElement).innerText
        const links = node.querySelectorAll('a[href]')
        for (const a of links) {
          const text = (a as HTMLElement).innerText
          const href = (a as HTMLAnchorElement).href
          if (text && href && !href.startsWith('javascript:')) {
            result = result.replace(text, '[' + text + '](' + href + ')')
          }
        }
        return result
      }

      processNode(bodyEl)

      const title = titleEl ? (titleEl as HTMLElement).innerText.trim() : ''
      const body = lines.join('\n').replace(/\n{3,}/g, '\n\n').trim()
      return JSON.stringify({ title, body })
    })

    if (!extracted) return ''

    try {
      const result = JSON.parse(extracted) as { title: string; body: string }
      return result.body || ''
    } catch {
      return ''
    }
  } catch (e) {
    console.error('[article-fetch]', e instanceof Error ? e.message : String(e))
    return ''
  } finally {
    if (browser) {
      try { await browser.close() } catch { /* ignore */ }
    }
  }
}
