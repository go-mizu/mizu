import { css } from './styles'
import { icons } from './icons'

// Logo: Lucide globe icon at 22px
const logoIcon = `<svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m12.83 2.18a2 2 0 0 0-1.66 0L2.6 6.08a1 1 0 0 0 0 1.83l8.58 3.91a2 2 0 0 0 1.66 0l8.58-3.9a1 1 0 0 0 0-1.83Z"/><path d="m22 17.65-9.17 4.16a2 2 0 0 1-1.66 0L2 17.65"/><path d="m22 12.65-9.17 4.16a2 2 0 0 1-1.66 0L2 12.65"/></svg>`

const nav = [
  {
    label: 'Platform',
    items: [
      { label: 'Overview', href: '/overview' },
      { label: 'Architecture', href: '/architecture' },
      { label: 'Crawler', href: '/crawler' },
      { label: 'Indexer', href: '/indexer' },
      { label: 'Knowledge Graph', href: '/knowledge-graph' },
      { label: 'Vector Search', href: '/vector-search' },
      { label: 'Ontology', href: '/ontology' },
      { label: 'Latest Build', href: '/latest-build' },
    ],
  },
  {
    label: 'Data',
    items: [
      { label: 'Get Started', href: '/get-started' },
      { label: 'Data Formats', href: '/data-formats' },
      { label: 'API Reference', href: '/api' },
      { label: 'Query Language', href: '/query-language' },
      { label: 'Errata', href: '/errata' },
    ],
  },
  {
    label: 'Resources',
    items: [
      { label: 'Blog', href: '/blog' },
      { label: 'Docs', href: '/docs' },
      { label: 'Research', href: '/research' },
      { label: 'FAQ', href: '/faq' },
      { label: 'Status', href: '/status' },
    ],
  },
  {
    label: 'Community',
    items: [
      { label: 'Collaborators', href: '/collaborators' },
      { label: 'Contributing', href: '/contributing' },
    ],
  },
  {
    label: 'About',
    items: [
      { label: 'Mission', href: '/mission' },
      { label: 'Impact', href: '/impact' },
      { label: 'Team', href: '/team' },
      { label: 'Roadmap', href: '/roadmap' },
      { label: 'Privacy', href: '/privacy' },
      { label: 'Terms', href: '/terms' },
      { label: 'Contact', href: '/contact' },
    ],
  },
]

function renderNav(): string {
  return nav
    .map(
      (g) => `<div class="nav-group">
      <button class="nav-btn">${g.label} ${icons.chevronDown}</button>
      <div class="nav-drop">${g.items.map((i) => `<a href="${i.href}">${i.label}</a>`).join('')}</div>
    </div>`
    )
    .join('')
}

function renderFooter(): string {
  return `<footer class="footer">
    <div class="footer-inner">
      <div class="footer-grid">
        <div class="footer-brand">
          <strong>OpenIndex</strong>
          <p>Open-source web intelligence.<br>Crawl, index, understand.</p>
        </div>
        ${nav.map((g) => `<div class="footer-col"><h4>${g.label}</h4>${g.items.map((i) => `<a href="${i.href}">${i.label}</a>`).join('')}</div>`).join('')}
      </div>
      <div class="footer-bottom">
        <span>&copy; 2026 OpenIndex &middot; Apache 2.0</span>
        <div class="footer-social">
          <a href="https://github.com/nicholasgasior/gopher-crawl">GitHub</a>
          <a href="https://discord.gg/openindex">Discord</a>
        </div>
      </div>
    </div>
  </footer>`
}

export function layout(title: string, body: string): string {
  return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>${title} — OpenIndex</title>
  <meta name="description" content="OpenIndex: Open-source web intelligence platform. Crawler, indexer, knowledge graph, vector search.">
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;600;700&display=swap" rel="stylesheet">
  <style>${css}</style>
</head>
<body>
  <header class="header">
    <div class="header-inner">
      <a href="/" class="logo">
        ${logoIcon}
        OpenIndex
      </a>
      <nav class="nav">${renderNav()}</nav>
      <div class="header-right">
        <a href="https://github.com/nicholasgasior/gopher-crawl" class="gh-link">${icons.github} GitHub</a>
      </div>
      <button class="mobile-toggle" aria-label="Menu">${icons.menu}</button>
    </div>
  </header>
  ${body}
  ${renderFooter()}
  <script>document.querySelector('.mobile-toggle').addEventListener('click',function(){document.querySelector('.header').classList.toggle('open')})</script>
</body>
</html>`
}
