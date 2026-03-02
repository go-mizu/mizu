import { marked } from 'marked'
import { postFiles } from './posts-manifest'

export interface BlogPost {
  slug: string
  title: string
  date: string
  summary: string
  tags: string[]
  content: string
}

function parseFrontmatter(raw: string): { meta: Record<string, string | string[]>; body: string } {
  const match = raw.match(/^---\r?\n([\s\S]*?)\r?\n---\r?\n([\s\S]*)$/)
  if (!match) return { meta: {}, body: raw }
  const meta: Record<string, string | string[]> = {}
  for (const line of match[1].split('\n')) {
    const idx = line.indexOf(':')
    if (idx === -1) continue
    const key = line.slice(0, idx).trim()
    let val = line.slice(idx + 1).trim()
    if (val.startsWith('[') && val.endsWith(']')) {
      meta[key] = val
        .slice(1, -1)
        .split(',')
        .map((s) => s.trim().replace(/^["']|["']$/g, ''))
    } else {
      meta[key] = val.replace(/^["']|["']$/g, '')
    }
  }
  return { meta, body: match[2] }
}

function parsePost(raw: string): BlogPost {
  const { meta, body } = parseFrontmatter(raw)
  return {
    slug: meta.slug as string,
    title: (meta.title as string) || '',
    date: String(meta.date),
    summary: (meta.summary as string) || '',
    tags: Array.isArray(meta.tags) ? meta.tags : [],
    content: marked.parse(body, { async: false }) as string,
  }
}

// Sorted newest first by date
export const posts: BlogPost[] = postFiles.map(parsePost).sort((a, b) => b.date.localeCompare(a.date))

function formatDate(d: string): string {
  return new Date(d + 'T00:00:00').toLocaleDateString('en-US', { year: 'numeric', month: 'long', day: 'numeric' })
}

export function getNav(slug: string): { prev?: BlogPost; next?: BlogPost } {
  const idx = posts.findIndex((p) => p.slug === slug)
  if (idx === -1) return {}
  return {
    prev: idx < posts.length - 1 ? posts[idx + 1] : undefined,
    next: idx > 0 ? posts[idx - 1] : undefined,
  }
}

export function renderBlogListing(): string {
  const items = posts
    .map(
      (p) => `
    <a href="/blog/${p.slug}" class="blog-item">
      <div class="blog-item-header">
        <time class="blog-item-date">${formatDate(p.date)}</time>
        <div class="post-tags">${p.tags.map((t) => `<span class="tag">${t}</span>`).join(' ')}</div>
      </div>
      <h3 class="blog-item-title">${p.title}</h3>
      ${p.summary ? `<p class="blog-item-summary">${p.summary}</p>` : ''}
    </a>`
    )
    .join('')

  return `<h2>Blog</h2>
<p>Project updates and technical deep-dives from the OpenIndex project.</p>
<div class="blog-list">${items}</div>`
}

export function renderPostNav(slug: string): string {
  const { prev, next } = getNav(slug)
  if (!prev && !next) return ''

  const prevHtml = prev
    ? `<a href="/blog/${prev.slug}" class="post-nav-link post-nav-prev">
        <span class="post-nav-label">&larr; Previous</span>
        <span class="post-nav-title">${prev.title}</span>
      </a>`
    : '<div></div>'

  const nextHtml = next
    ? `<a href="/blog/${next.slug}" class="post-nav-link post-nav-next">
        <span class="post-nav-label">Next &rarr;</span>
        <span class="post-nav-title">${next.title}</span>
      </a>`
    : '<div></div>'

  return `<nav class="post-nav">${prevHtml}${nextHtml}</nav>`
}

export function renderPostMeta(post: BlogPost): string {
  const tags = post.tags.map((t) => `<span class="tag">${t}</span>`).join(' ')
  return `<div class="post-meta"><time>${formatDate(post.date)}</time><div class="post-tags">${tags}</div></div>`
}
