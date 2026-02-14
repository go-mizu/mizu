const TRANSLATE_BASE = 'https://translate.googleapis.com/translate_a/single'

/**
 * Translate a chunk of text via Google Translate API.
 * Returns original text on failure or if text is trivial (empty, whitespace-only, numbers/punctuation).
 */
export async function translateChunk(text: string, sl: string, tl: string): Promise<string> {
  const trimmed = text.trim()
  if (!trimmed) return text
  if (/^[\s\d\p{P}\p{S}]+$/u.test(trimmed)) return text

  const params = new URLSearchParams()
  params.set('client', 'gtx')
  params.set('sl', sl)
  params.set('tl', tl)
  params.set('dj', '1')
  params.append('dt', 't')

  try {
    let resp: Response
    if (text.length <= 2000) {
      params.set('q', text)
      resp = await fetch(`${TRANSLATE_BASE}?${params.toString()}`, {
        headers: { 'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36' },
      })
    } else {
      const body = new URLSearchParams()
      body.set('q', text)
      resp = await fetch(`${TRANSLATE_BASE}?${params.toString()}`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/x-www-form-urlencoded',
          'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36',
        },
        body: body.toString(),
      })
    }

    if (!resp.ok) return text

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const data: any = await resp.json()
    if (!data.sentences) return text

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const translated = data.sentences
      .filter((s: { trans?: string }) => s.trans != null)
      .map((s: { trans: string }) => s.trans)
      .join('')

    return translated || text
  } catch {
    return text
  }
}

/** Selectors for elements whose text content should NOT be translated. */
const SKIP_SELECTORS = [
  'script', 'style', 'code', 'pre', 'kbd', 'samp', 'var',
  'svg', 'noscript', 'canvas', 'audio', 'video', 'iframe',
] as const

/** Selectors for elements whose text content should be translated. */
const TRANSLATABLE_SELECTORS = [
  'p', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6',
  'li', 'td', 'th', 'dt', 'dd', 'figcaption', 'blockquote',
  'title', 'label', 'button', 'caption', 'summary', 'legend', 'option',
] as const

/** URL schemes that should not be rewritten through the proxy. */
const SKIP_HREF_PREFIXES = ['#', 'javascript:', 'mailto:', 'tel:', 'data:']

/**
 * Rewrite a URL to go through the translation proxy.
 * Converts relative URLs to absolute using originUrl, then wraps as /page/TL/ABSOLUTE_URL.
 */
function rewriteUrl(href: string, originUrl: URL, proxyBase: string, tl: string): string {
  if (SKIP_HREF_PREFIXES.some((p) => href.startsWith(p))) {
    return href
  }

  let absolute: string
  try {
    // Try as absolute URL first
    new URL(href)
    absolute = href
  } catch {
    // Relative URL - resolve against origin
    try {
      absolute = new URL(href, originUrl.origin).toString()
    } catch {
      return href
    }
  }

  return `${proxyBase}/page/${tl}/${absolute}`
}

/**
 * Build the translation banner HTML injected at the top of the page.
 */
function buildBanner(originUrl: URL, tl: string, sl: string): string {
  const slLabel = sl === 'auto' ? 'auto-detected language' : sl
  return `<div style="position:fixed;top:0;left:0;right:0;z-index:2147483647;background:#4285f4;color:#fff;font:14px/1 sans-serif;padding:8px 16px;display:flex;align-items:center;justify-content:center;gap:12px;box-shadow:0 2px 6px rgba(0,0,0,.3)"><span>Translated from <b>${slLabel}</b> to <b>${tl}</b></span><a href="${originUrl.toString()}" style="color:#fff;text-decoration:underline;font-size:13px">View original</a></div><div style="height:38px"></div>`
}

/**
 * Create an HTMLRewriter chain that translates a web page from sl to tl.
 *
 * @param originUrl  The original page URL
 * @param proxyBase  The proxy origin (e.g. https://translate.example.com)
 * @param tl         Target language code
 * @param sl         Source language code (usually 'auto')
 */
export function makePageRewriter(originUrl: URL, proxyBase: string, tl: string, sl: string): HTMLRewriter {
  // Shared text buffer for accumulating chunked text nodes.
  // Safe because HTMLRewriter delivers chunks of a single text node consecutively
  // before moving to the next element/text node.
  const textBuffer: string[] = []

  // Track whether we're inside a skip element (script, style, code, etc.)
  let skipDepth = 0

  let rewriter = new HTMLRewriter()
    // --- <html> handler: set lang attribute ---
    .on('html', {
      element(el) {
        el.setAttribute('lang', tl)
      },
    })

    // --- <head> handler: inject <base> and translation banner ---
    .on('head', {
      element(el) {
        el.prepend(`<base href="${originUrl.origin}/">`, { html: true })
        el.prepend(buildBanner(originUrl, tl, sl), { html: true })
      },
    })

    // --- <a href> handler: rewrite links through proxy ---
    .on('a[href]', {
      element(el) {
        const href = el.getAttribute('href')
        if (href) {
          el.setAttribute('href', rewriteUrl(href, originUrl, proxyBase, tl))
        }
      },
    })

    // --- <form action> handler: rewrite form targets through proxy ---
    .on('form[action]', {
      element(el) {
        const action = el.getAttribute('action')
        if (action) {
          el.setAttribute('action', rewriteUrl(action, originUrl, proxyBase, tl))
        }
      },
    })

  // --- Skip elements: mark so text handler avoids translating their content ---
  for (const tag of SKIP_SELECTORS) {
    rewriter = rewriter.on(tag, {
      element(el) {
        skipDepth++
        el.onEndTag(() => {
          skipDepth--
        })
      },
    })
  }

  // --- Translatable text handler ---
  for (const tag of TRANSLATABLE_SELECTORS) {
    rewriter = rewriter.on(tag, {
      element(el) {
        // Skip elements with translate="no" attribute or notranslate class
        if (el.getAttribute('translate') === 'no') {
          skipDepth++
          el.onEndTag(() => { skipDepth-- })
          return
        }
        const cls = el.getAttribute('class') || ''
        if (cls.split(/\s+/).includes('notranslate')) {
          skipDepth++
          el.onEndTag(() => { skipDepth-- })
        }
      },

      async text(text) {
        // Don't translate content inside skip elements
        if (skipDepth > 0) return

        textBuffer.push(text.text)

        if (!text.lastInTextNode) {
          // Not the last chunk - remove this chunk, we'll output full text at the end
          text.remove()
          return
        }

        // Last chunk - join all accumulated chunks and translate
        const fullText = textBuffer.splice(0).join('')

        // Skip trivial text (whitespace-only or numbers/punctuation-only)
        if (!fullText.trim() || /^[\s\d\p{P}\p{S}]+$/u.test(fullText.trim())) {
          return
        }

        try {
          const translated = await translateChunk(fullText, sl, tl)
          text.replace(translated, { html: true })
        } catch {
          // Keep original text on error
        }
      },
    })
  }

  return rewriter
}
