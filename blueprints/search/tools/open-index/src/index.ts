import { Hono } from 'hono'
import { layout } from './layout'

import { homePage } from './content/home'
import { overviewPage } from './content/overview'
import { architecturePage } from './content/architecture'
import { crawlerPage } from './content/crawler'
import { indexerPage } from './content/indexer'
import { knowledgeGraphPage } from './content/knowledge-graph'
import { vectorSearchPage } from './content/vector-search'
import { ontologyPage } from './content/ontology'
import { latestBuildPage } from './content/latest-build'
import { getStartedPage } from './content/get-started'
import { dataFormatsPage } from './content/data-formats'
import { apiReferencePage } from './content/api-reference'
import { queryLanguagePage } from './content/query-language'
import { errataPage } from './content/errata'
import { docsPage } from './content/docs'
import { researchPage } from './content/research'
import { faqPage } from './content/faq'
import { statusPage } from './content/status'
import { collaboratorsPage } from './content/collaborators'
import { contributingPage } from './content/contributing'
import { missionPage } from './content/mission'
import { impactPage } from './content/impact'
import { teamPage } from './content/team'
import { roadmapPage } from './content/roadmap'
import { privacyPage } from './content/privacy'
import { termsPage } from './content/terms'
import { contactPage } from './content/contact'
import { posts, renderBlogListing, renderPostNav, renderPostMeta } from './blog-registry'

const app = new Hono()

function respond(body: string, status = 200) {
  return new Response(body, {
    status,
    headers: {
      'Content-Type': 'text/html;charset=utf-8',
      'Cache-Control': 'public, max-age=3600',
    },
  })
}

function page(title: string, subtitle: string, bc: string, content: string): string {
  return layout(title, `
  <div class="page-header">
    <div class="breadcrumb">${bc}</div>
    <h1>${title}</h1>
    <p>${subtitle}</p>
  </div>
  <div class="content">
    ${content}
  </div>`)
}

// Home
app.get('/', () => respond(layout('Open Web Intelligence Platform', homePage)))

// All pages
const pages = [
  // Platform
  { path: '/overview', title: 'Overview', sub: 'A complete open web intelligence platform — from crawl to knowledge.', bc: '<a href="/">Home</a> / Platform', content: overviewPage },
  { path: '/architecture', title: 'Architecture', sub: 'System architecture and technology stack.', bc: '<a href="/">Home</a> / Platform', content: architecturePage },
  { path: '/crawler', title: 'Open Crawler', sub: 'A transparent, distributed web crawler for the open web.', bc: '<a href="/">Home</a> / Platform', content: crawlerPage },
  { path: '/indexer', title: 'Multi-layer Indexer', sub: 'CDX, Columnar, Full-text, and Vector — four index types, one platform.', bc: '<a href="/">Home</a> / Platform', content: indexerPage },
  { path: '/knowledge-graph', title: 'Knowledge Graph', sub: 'Entities, relationships, and semantic connections from the open web.', bc: '<a href="/">Home</a> / Platform', content: knowledgeGraphPage },
  { path: '/vector-search', title: 'Vector Search', sub: 'Semantic search powered by dense embeddings.', bc: '<a href="/">Home</a> / Platform', content: vectorSearchPage },
  { path: '/ontology', title: 'Open Ontology', sub: 'A community-maintained schema for web entities.', bc: '<a href="/">Home</a> / Platform', content: ontologyPage },
  { path: '/latest-build', title: 'Latest Build', sub: 'Current index build details and download links.', bc: '<a href="/">Home</a> / Platform', content: latestBuildPage },
  // Data & APIs
  { path: '/get-started', title: 'Get Started', sub: 'Start querying the open web in minutes.', bc: '<a href="/">Home</a> / Data & APIs', content: getStartedPage },
  { path: '/data-formats', title: 'Data Formats', sub: 'WARC, WAT, WET, Parquet, Vector, and Knowledge Graph formats.', bc: '<a href="/">Home</a> / Data & APIs', content: dataFormatsPage },
  { path: '/api', title: 'API Reference', sub: 'REST API for search, lookup, vector search, and graph queries.', bc: '<a href="/">Home</a> / Data & APIs', content: apiReferencePage },
  { path: '/query-language', title: 'Query Language', sub: 'OpenIndex Query Language (OQL) — SQL-like queries for the web.', bc: '<a href="/">Home</a> / Data & APIs', content: queryLanguagePage },
  { path: '/errata', title: 'Errata', sub: 'Known issues and corrections.', bc: '<a href="/">Home</a> / Data & APIs', content: errataPage },
  // Resources
  { path: '/docs', title: 'Documentation', sub: 'Guides, tutorials, and technical reference.', bc: '<a href="/">Home</a> / Resources', content: docsPage },
  { path: '/research', title: 'Research', sub: 'Academic research powered by OpenIndex data.', bc: '<a href="/">Home</a> / Resources', content: researchPage },
  { path: '/faq', title: 'FAQ', sub: 'Frequently asked questions.', bc: '<a href="/">Home</a> / Resources', content: faqPage },
  { path: '/status', title: 'System Status', sub: 'Real-time health and uptime.', bc: '<a href="/">Home</a> / Resources', content: statusPage },
  // Community
  { path: '/collaborators', title: 'Collaborators', sub: 'Organizations partnering with OpenIndex.', bc: '<a href="/">Home</a> / Community', content: collaboratorsPage },
  { path: '/contributing', title: 'Contributing', sub: 'How to contribute to the open-source project.', bc: '<a href="/">Home</a> / Community', content: contributingPage },
  // About
  { path: '/mission', title: 'Mission', sub: 'Democratizing web intelligence for everyone.', bc: '<a href="/">Home</a> / About', content: missionPage },
  { path: '/impact', title: 'Impact', sub: 'How OpenIndex is advancing research and society.', bc: '<a href="/">Home</a> / About', content: impactPage },
  { path: '/team', title: 'Team', sub: 'The people building OpenIndex.', bc: '<a href="/">Home</a> / About', content: teamPage },
  { path: '/roadmap', title: 'Roadmap', sub: 'Where we are and where we are headed.', bc: '<a href="/">Home</a> / About', content: roadmapPage },
  { path: '/privacy', title: 'Privacy Policy', sub: 'How we handle your data.', bc: '<a href="/">Home</a> / About', content: privacyPage },
  { path: '/terms', title: 'Terms of Use', sub: 'Terms governing use of OpenIndex data and services.', bc: '<a href="/">Home</a> / About', content: termsPage },
  { path: '/contact', title: 'Contact', sub: 'Get in touch with the OpenIndex team.', bc: '<a href="/">Home</a> / About', content: contactPage },
]

for (const p of pages) {
  app.get(p.path, () => respond(page(p.title, p.sub, p.bc, p.content)))
}

// Blog listing (auto-generated from registry)
app.get('/blog', () => respond(page('Blog', 'Updates, research highlights, and engineering deep-dives.', '<a href="/">Home</a> / Resources', renderBlogListing())))

// Blog posts (auto-discovered from registry)
for (const p of posts) {
  app.get(`/blog/${p.slug}`, () => {
    const meta = renderPostMeta(p)
    const nav = renderPostNav(p.slug)
    const body = layout(p.title, `
    <div class="page-header">
      <div class="breadcrumb"><a href="/">Home</a> / <a href="/blog">Blog</a></div>
      <h1>${p.title}</h1>
      ${meta}
    </div>
    <div class="content">
      ${p.content}
      ${nav}
    </div>`)
    return respond(body)
  })
}

// 404
app.notFound(() => {
  const body = layout('Not Found', `
  <div style="text-align:center;padding:8rem 2rem">
    <h1 style="font-size:4rem;letter-spacing:-0.04em;margin-bottom:0.5rem">404</h1>
    <p style="font-size:1rem;color:var(--fg-2);margin-bottom:2rem">This page could not be found.</p>
    <a href="/" class="btn btn-p">Back to Home</a>
  </div>`)
  return respond(body, 404)
})

export default app
