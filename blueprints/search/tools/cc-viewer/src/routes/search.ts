import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { renderLayout, renderError } from '../html'

const app = new Hono<HonoEnv>()

app.get('/', (c) => {
  const q = (c.req.query('q') || '').trim()
  if (!q) {
    return c.redirect('/')
  }

  // Auto-detect: URL vs domain vs general
  if (q.startsWith('http://') || q.startsWith('https://')) {
    return c.redirect(`/url/${q}`)
  }

  // If it looks like a URL path (has /)
  if (q.includes('/') && q.includes('.')) {
    const url = q.startsWith('http') ? q : `https://${q}`
    return c.redirect(`/url/${url}`)
  }

  // If it looks like a domain (has a dot, no spaces)
  if (q.includes('.') && !q.includes(' ')) {
    return c.redirect(`/domain/${q}`)
  }

  // Otherwise show help
  return c.html(renderLayout('Search - CC Viewer', `<div class="page-header">
<div class="breadcrumb"><a href="/">Home</a> <span>Search</span></div>
<h1 class="page-title">Search</h1>
</div>
<div class="empty-state">
<h2>How to search</h2>
<p>Enter a <strong>URL</strong> (e.g., <code>https://example.com/page</code>) to find all captures.</p>
<p>Enter a <strong>domain</strong> (e.g., <code>example.com</code>) to browse all pages.</p>
<p><a href="/" class="btn-primary">Try Again</a></p>
</div>`, { query: q }))
})

export default app
