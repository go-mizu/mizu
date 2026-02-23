export const css = /* css */`
/* ================================================
   OpenIndex Design System — Vercel-inspired, mono font
   ================================================ */

:root {
  --bg: #ffffff;
  --bg-alt: #fafafa;
  --bg-code: #0a0a0a;
  --fg: #171717;
  --fg-2: #666666;
  --fg-3: #8f8f8f;
  --fg-4: #c9c9c9;
  --border: rgba(0,0,0,0.08);
  --border-h: rgba(0,0,0,0.14);
  --accent: #0070f3;
  --accent-h: #005cc5;
  --green: #16a34a;
  --red: #dc2626;
  --amber: #ca8a04;
  --r: 8px;
  --mono: 'JetBrains Mono', 'SF Mono', 'Cascadia Code', monospace;
  --t: 200ms ease;
  --max-w: 1100px;
  --content-w: 960px;
  --px: 2rem;
}

*, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }

html {
  font-family: var(--mono);
  -webkit-font-smoothing: antialiased;
  color: var(--fg);
  background: var(--bg);
  font-size: 15px;
  line-height: 1.6;
}

body { min-height: 100vh; display: flex; flex-direction: column; }

a { color: var(--fg); text-decoration: underline; text-decoration-color: var(--fg-4); text-underline-offset: 3px; transition: text-decoration-color var(--t); }
a:hover { text-decoration-color: var(--fg); }

h1, h2, h3, h4 { color: var(--fg); line-height: 1.25; }
h1 { font-size: 1.75rem; font-weight: 600; letter-spacing: -0.03em; margin-bottom: 0.625rem; }
h2 { font-size: 1.375rem; font-weight: 600; letter-spacing: -0.02em; margin: 2.5rem 0 0.875rem; }
h3 { font-size: 1.125rem; font-weight: 600; letter-spacing: -0.01em; margin: 2rem 0 0.625rem; }
h4 { font-size: 1rem; font-weight: 600; margin: 1.5rem 0 0.5rem; }
p { margin-bottom: 1rem; color: var(--fg-2); }
ul, ol { margin: 0.5rem 0 1.25rem 1.25rem; color: var(--fg-2); }
li { margin-bottom: 0.375rem; }
strong { color: var(--fg); font-weight: 600; }
hr { border: none; border-top: 1px solid var(--border); margin: 2.5rem 0; }

code { font-family: var(--mono); font-size: 0.875rem; background: var(--bg-alt); padding: 0.15em 0.4em; border-radius: 4px; }

pre {
  font-family: var(--mono); font-size: 0.875rem;
  background: var(--bg-code); color: #e0e0e0;
  padding: 1.25rem 1.5rem; border-radius: 12px;
  overflow-x: auto; line-height: 1.55; margin: 1.25rem 0;
  -webkit-overflow-scrolling: touch;
}
pre code { background: none; padding: 0; color: inherit; font-size: inherit; }

table { width: 100%; border-collapse: collapse; margin: 1.25rem 0; font-size: 0.875rem; }
th, td { padding: 0.75rem 1rem; text-align: left; border-bottom: 1px solid var(--border); }
th { font-weight: 500; color: var(--fg-3); font-size: 0.8125rem; text-transform: uppercase; letter-spacing: 0.05em; }

blockquote { border-left: 2px solid var(--border-h); padding: 0.625rem 1.25rem; margin: 1.25rem 0; }
blockquote p { color: var(--fg-2); margin: 0; font-style: italic; }

::-webkit-scrollbar { width: 6px; height: 6px; }
::-webkit-scrollbar-track { background: transparent; }
::-webkit-scrollbar-thumb { background: var(--fg-4); border-radius: 3px; }

/* ============= HEADER ============= */
.header {
  position: sticky; top: 0; z-index: 100;
  background: rgba(255,255,255,0.8);
  backdrop-filter: saturate(180%) blur(20px);
  -webkit-backdrop-filter: saturate(180%) blur(20px);
  border-bottom: 1px solid var(--border);
}

.header-inner {
  max-width: var(--max-w);
  margin: 0 auto;
  padding: 0 var(--px);
  display: flex;
  align-items: center;
  height: 64px;
  gap: 2rem;
}

.logo {
  display: flex; align-items: center; gap: 0.5rem;
  font-size: 1.0625rem; font-weight: 700;
  color: var(--fg); text-decoration: none;
  white-space: nowrap; letter-spacing: -0.02em;
}
.logo:hover { color: var(--fg); text-decoration: none; }
.logo svg { flex-shrink: 0; }

.nav { display: flex; align-items: center; gap: 0; flex: 1; }
.nav-group { position: relative; }

.nav-btn {
  all: unset; cursor: pointer;
  font-family: var(--mono);
  color: var(--fg-2); font-size: 0.875rem;
  padding: 0.375rem 0.625rem;
  border-radius: 6px;
  display: flex; align-items: center; gap: 3px;
  transition: color var(--t);
  white-space: nowrap;
}
.nav-btn:hover { color: var(--fg); }
.nav-btn svg { opacity: 0.4; }

.nav-drop {
  position: absolute; top: calc(100% + 8px); left: -8px;
  background: var(--bg);
  border-radius: 12px; padding: 8px; min-width: 200px;
  box-shadow: 0 4px 32px rgba(0,0,0,0.08), 0 0 0 1px rgba(0,0,0,0.06);
  opacity: 0; visibility: hidden;
  transform: translateY(-4px);
  transition: all var(--t);
}
.nav-group:hover .nav-drop { opacity: 1; visibility: visible; transform: translateY(0); }

.nav-drop a {
  display: block; padding: 0.5rem 0.75rem;
  color: var(--fg-2); font-size: 0.875rem;
  border-radius: 8px; text-decoration: none;
  transition: all var(--t);
}
.nav-drop a:hover { background: var(--bg-alt); color: var(--fg); }

.header-right { margin-left: auto; display: flex; align-items: center; gap: 0.5rem; flex-shrink: 0; }

.gh-link {
  display: flex; align-items: center; gap: 0.375rem;
  color: var(--fg-2); font-size: 0.8125rem; text-decoration: none;
  padding: 0.4rem 0.875rem;
  background: var(--bg-alt);
  border-radius: 999px;
  transition: all var(--t);
}
.gh-link:hover { color: var(--fg); background: #efefef; }

.mobile-toggle {
  display: none; all: unset; cursor: pointer;
  color: var(--fg-2); padding: 0.375rem;
  flex-shrink: 0;
}

/* Mobile nav */
@media (max-width: 900px) {
  .nav { display: none; }
  .header-right { display: none; }
  .mobile-toggle { display: flex; align-items: center; margin-left: auto; }

  .header.open .nav {
    display: flex;
    flex-direction: column;
    position: absolute;
    top: 64px; left: 0; right: 0;
    background: var(--bg);
    padding: 0.5rem var(--px) 1.5rem;
    border-bottom: 1px solid var(--border);
    box-shadow: 0 8px 32px rgba(0,0,0,0.06);
    max-height: calc(100vh - 64px);
    overflow-y: auto;
    gap: 0;
  }
  .header.open .nav-group { width: 100%; }
  .header.open .nav-btn {
    font-weight: 600; color: var(--fg);
    pointer-events: none;
    padding: 0.75rem 0 0.25rem;
    font-size: 0.75rem;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--fg-3);
  }
  .header.open .nav-btn svg { display: none; }
  .header.open .nav-drop {
    position: static;
    opacity: 1; visibility: visible;
    transform: none;
    box-shadow: none;
    padding: 0 0 0.5rem;
    min-width: auto;
    background: transparent;
    border-radius: 0;
  }
  .header.open .nav-drop a {
    padding: 0.4rem 0;
    border-radius: 0;
  }
}

/* ============= HERO ============= */
.hero { padding: 6rem var(--px) 5rem; text-align: center; max-width: var(--max-w); margin: 0 auto; }
.hero h1 { font-size: 2.75rem; letter-spacing: -0.04em; font-weight: 700; line-height: 1.1; margin-bottom: 1.25rem; }
.hero-sub { font-size: 1.0625rem; color: var(--fg-2); max-width: 640px; margin: 0 auto 2.5rem; line-height: 1.65; }
.hero-actions { display: flex; gap: 0.75rem; justify-content: center; flex-wrap: wrap; }

/* ============= BUTTONS ============= */
.btn {
  display: inline-flex; align-items: center; gap: 0.375rem;
  font-family: var(--mono); font-size: 0.875rem; font-weight: 500;
  padding: 0.625rem 1.25rem; border-radius: 999px;
  text-decoration: none; transition: all var(--t); cursor: pointer;
  border: none; white-space: nowrap;
}
.btn-p { background: var(--fg); color: var(--bg); box-shadow: 0 2px 8px rgba(0,0,0,0.12); }
.btn-p:hover { background: #333; color: var(--bg); box-shadow: 0 4px 16px rgba(0,0,0,0.16); }
.btn-s { background: var(--bg); color: var(--fg); border: 1px solid var(--border-h); }
.btn-s:hover { border-color: var(--fg-3); }

/* ============= STATS ============= */
.stats { display: grid; grid-template-columns: repeat(4, 1fr); background: var(--bg-alt); border-radius: 12px; margin: 3.5rem auto 0; max-width: 640px; }
.stat { text-align: center; padding: 1.5rem 0.75rem; }
.stat-v { font-size: 1.75rem; font-weight: 700; color: var(--fg); letter-spacing: -0.03em; }
.stat-l { font-size: 0.75rem; color: var(--fg-3); margin-top: 0.25rem; text-transform: uppercase; letter-spacing: 0.04em; }

/* ============= FEATURE GRID (Home page Vercel-style) ============= */
.feature-grid-wrap { padding: 0 var(--px); }
.feature-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 1px;
  background: var(--border);
  border-radius: 12px;
  overflow: hidden;
  border: 1px solid var(--border);
  margin: 0 auto;
  max-width: var(--max-w);
}
.feature-cell {
  background: var(--bg);
  padding: 2rem;
  transition: background var(--t);
  text-decoration: none; color: inherit;
  display: block;
}
.feature-cell:hover { background: var(--bg-alt); }
.feature-cell h3 { font-size: 1.25rem; font-weight: 700; letter-spacing: -0.02em; margin: 0 0 0.5rem; line-height: 1.3; }
.feature-cell p { font-size: 0.875rem; color: var(--fg-2); margin: 0 0 1rem; line-height: 1.6; }
.feature-cell .arrow { display: inline-flex; align-items: center; justify-content: center; width: 36px; height: 36px; border-radius: 999px; border: 1px solid var(--border-h); color: var(--fg-2); transition: all var(--t); }
.feature-cell:hover .arrow { border-color: var(--fg-3); color: var(--fg); }
.feature-cell .cell-icon { color: var(--fg-3); margin-bottom: 0.75rem; }
.feature-cell-hero { padding: 2.5rem 2rem; }
.feature-cell-hero h2 { font-size: 2rem; font-weight: 700; letter-spacing: -0.03em; margin: 0 0 0.5rem; line-height: 1.15; }
.feature-cell-hero p { font-size: 1rem; color: var(--fg-2); margin: 0; }

/* ============= SHOWCASE (Two-column feature sections) ============= */
.showcase {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 3rem;
  align-items: center;
  max-width: var(--max-w);
  margin: 0 auto;
  padding: 5rem var(--px);
}
.showcase-rev { direction: rtl; }
.showcase-rev > * { direction: ltr; }
.showcase-tag {
  font-size: 0.75rem;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--fg-3);
  margin-bottom: 1rem;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}
.showcase h2 {
  font-size: 1.625rem;
  line-height: 1.35;
  margin: 0 0 1.5rem;
  letter-spacing: -0.02em;
}
.showcase .muted { color: var(--fg-3); font-weight: 400; }
.showcase-visual {
  background: var(--bg-code);
  border-radius: 12px;
  padding: 1.5rem 1.75rem;
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
  font-size: 0.8125rem;
  line-height: 1.6;
  color: #a0a0a0;
}
.showcase-visual .hl { color: #e0e0e0; }
.showcase-visual .green { color: #4ade80; }
.showcase-visual .blue { color: #60a5fa; }
.showcase-visual .amber { color: #fbbf24; }
.showcase-visual .dim { color: #555; }

/* ============= CONTENT ============= */
.wrap { max-width: var(--content-w); margin: 0 auto; padding: 0 var(--px); }
.content { max-width: var(--content-w); margin: 0 auto; padding: 3rem var(--px) 5rem; }

/* ============= PAGE HEADER ============= */
.page-header { padding: 3.5rem var(--px) 2.5rem; max-width: var(--content-w); margin: 0 auto; border-bottom: 1px solid var(--border); }
.page-header h1 { font-size: 1.5rem; margin-bottom: 0.375rem; }
.page-header p { font-size: 1rem; color: var(--fg-3); margin: 0; }
.breadcrumb { font-size: 0.75rem; color: var(--fg-4); margin-bottom: 0.5rem; text-transform: uppercase; letter-spacing: 0.04em; }
.breadcrumb a { color: var(--fg-3); text-decoration: none; }
.breadcrumb a:hover { color: var(--fg); }

/* ============= CARDS (inner pages) ============= */
.cards { display: grid; grid-template-columns: repeat(auto-fill, minmax(min(280px, 100%), 1fr)); gap: 1rem; margin: 1.5rem 0; }
.card {
  border: 1px solid var(--border); border-radius: 12px;
  padding: 1.5rem; transition: all var(--t);
}
.card:hover { border-color: var(--border-h); }
.card-ic { display: flex; align-items: center; gap: 0.5rem; margin-bottom: 0.75rem; color: var(--fg-3); }
.card-ic svg { flex-shrink: 0; }
.card-ic span { font-weight: 600; color: var(--fg); font-size: 1rem; }
.card h3 { margin: 0 0 0.375rem; font-size: 1rem; }
.card p { color: var(--fg-2); font-size: 0.875rem; margin: 0; line-height: 1.6; }
.card-lk { display: inline-flex; align-items: center; gap: 0.25rem; margin-top: 0.875rem; font-size: 0.875rem; color: var(--fg-3); text-decoration: none; }
.card-lk:hover { color: var(--fg); }

/* ============= SECTIONS ============= */
.section { padding: 5rem var(--px); max-width: var(--max-w); margin: 0 auto; }
.section-alt { background: var(--bg-alt); }
.section-alt .section { max-width: var(--max-w); margin: 0 auto; }
.section-narrow { max-width: var(--content-w); }
.sh { text-align: center; margin-bottom: 2.5rem; }
.sh h2 { margin: 0 0 0.5rem; font-size: 1.5rem; letter-spacing: -0.02em; }
.sh p { color: var(--fg-2); font-size: 1rem; max-width: 560px; margin: 0 auto; }

/* ============= BLOG ============= */
.blog-list { margin: 1.5rem 0; display: flex; flex-direction: column; gap: 0; }
.blog-item {
  display: block; text-decoration: none; color: inherit;
  padding: 1.25rem 0; border-bottom: 1px solid var(--border);
  transition: all var(--t);
}
.blog-item:first-child { border-top: 1px solid var(--border); }
.blog-item:hover { background: var(--bg-alt); padding-left: 1rem; padding-right: 1rem; margin-left: -1rem; margin-right: -1rem; border-radius: 8px; border-color: transparent; }
.blog-item:hover + .blog-item { border-top-color: transparent; }
.blog-item-header { display: flex; align-items: center; gap: 0.75rem; margin-bottom: 0.375rem; }
.blog-item-date { font-size: 0.75rem; color: var(--fg-4); white-space: nowrap; }
.blog-item-title { font-size: 1.0625rem; font-weight: 600; margin: 0 0 0.25rem; letter-spacing: -0.01em; color: var(--fg); }
.blog-item-summary { font-size: 0.875rem; color: var(--fg-3); margin: 0; line-height: 1.55; }

/* ============= TAGS ============= */
.tag { display: inline-block; font-size: 0.6875rem; font-weight: 500; text-transform: uppercase; letter-spacing: 0.05em; color: var(--fg-3); background: var(--bg-alt); padding: 0.2rem 0.5rem; border-radius: 999px; }

/* ============= POST META ============= */
.post-meta { display: flex; align-items: center; gap: 0.75rem; flex-wrap: wrap; margin-top: 0.5rem; }
.post-meta time { font-size: 0.8125rem; color: var(--fg-3); }
.post-tags { display: flex; gap: 0.375rem; flex-wrap: wrap; }

/* ============= POST NAV ============= */
.post-nav { display: grid; grid-template-columns: 1fr 1fr; gap: 1rem; margin-top: 4rem; padding-top: 2rem; border-top: 1px solid var(--border); }
.post-nav-link { display: block; padding: 1.25rem; border: 1px solid var(--border); border-radius: 12px; text-decoration: none; transition: all var(--t); }
.post-nav-link:hover { border-color: var(--border-h); }
.post-nav-prev { text-align: left; }
.post-nav-next { text-align: right; }
.post-nav-label { display: block; font-size: 0.75rem; color: var(--fg-3); text-transform: uppercase; letter-spacing: 0.04em; margin-bottom: 0.25rem; }
.post-nav-title { display: block; font-size: 0.875rem; color: var(--fg); font-weight: 500; }

/* ============= COLLAB ============= */
.collab-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(min(160px, 100%), 1fr)); gap: 0.75rem; margin: 1rem 0; }
.collab-card { border: 1px solid var(--border); border-radius: 12px; padding: 1rem 0.75rem; text-align: center; }
.collab-card-name { font-size: 0.8125rem; color: var(--fg-2); font-weight: 500; }

/* ============= TEAM ============= */
.team-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(min(240px, 100%), 1fr)); gap: 0.75rem; margin: 1rem 0 2rem; }
.team-card { text-align: center; padding: 1.5rem 1rem; }
.team-avatar { width: 52px; height: 52px; border-radius: 50%; background: var(--bg-alt); margin: 0 auto 0.625rem; display: flex; align-items: center; justify-content: center; font-size: 0.8125rem; font-weight: 600; color: var(--fg-3); }
.team-card h4 { margin: 0 0 0.125rem; font-size: 0.875rem; }
.team-card p { font-size: 0.8125rem; color: var(--fg-3); margin: 0; }

/* ============= STATUS ============= */
.status-item { display: flex; align-items: center; justify-content: space-between; padding: 0.75rem 1rem; border: 1px solid var(--border); border-radius: 12px; margin-bottom: 0.5rem; gap: 0.5rem; }
.status-name { font-size: 0.875rem; font-weight: 500; min-width: 0; }
.status-badge { font-size: 0.75rem; font-weight: 500; padding: 0.15rem 0.625rem; border-radius: 999px; white-space: nowrap; flex-shrink: 0; }
.status-operational { background: #ecfdf5; color: var(--green); }

/* ============= ACCORDION ============= */
details { border: 1px solid var(--border); border-radius: 12px; margin-bottom: 0.5rem; }
summary { padding: 0.875rem 1rem; font-weight: 500; font-size: 0.875rem; cursor: pointer; user-select: none; list-style: none; display: flex; align-items: center; justify-content: space-between; border-radius: 12px; gap: 0.5rem; }
summary:hover { background: var(--bg-alt); }
summary::-webkit-details-marker { display: none; }
summary::after { content: '+'; font-size: 1.125rem; color: var(--fg-4); font-weight: 300; flex-shrink: 0; }
details[open] summary::after { content: '\\2212'; }
details[open] summary { border-radius: 12px 12px 0 0; }
.details-body { padding: 1rem 1rem 1.25rem; }
.details-body p:last-child { margin-bottom: 0; }

/* ============= FOOTER ============= */
.footer { border-top: 1px solid var(--border); padding: 3.5rem var(--px) 2rem; margin-top: auto; }
.footer-inner { max-width: var(--max-w); margin: 0 auto; }
.footer-grid { display: grid; grid-template-columns: 1.5fr repeat(5, 1fr); gap: 2rem; margin-bottom: 2.5rem; }
.footer-brand strong { font-size: 1rem; }
.footer-brand p { font-size: 0.8125rem; color: var(--fg-3); line-height: 1.5; margin-top: 0.375rem; }
.footer-col h4 { font-size: 0.6875rem; color: var(--fg-4); text-transform: uppercase; letter-spacing: 0.06em; margin-bottom: 0.625rem; font-weight: 500; }
.footer-col a { display: block; color: var(--fg-3); font-size: 0.8125rem; padding: 0.2rem 0; text-decoration: none; }
.footer-col a:hover { color: var(--fg); }
.footer-bottom { border-top: 1px solid var(--border); padding-top: 1.25rem; display: flex; justify-content: space-between; align-items: center; font-size: 0.75rem; color: var(--fg-4); flex-wrap: wrap; gap: 0.5rem; }
.footer-bottom a { color: var(--fg-4); text-decoration: none; }
.footer-bottom a:hover { color: var(--fg); }
.footer-social { display: flex; gap: 1rem; }

/* ============= TIMELINE ============= */
.timeline { position: relative; padding-left: 1.5rem; }
.timeline::before { content: ''; position: absolute; left: 0; top: 0.25rem; bottom: 0.25rem; width: 1px; background: var(--border); }
.timeline-item { position: relative; padding-bottom: 1.5rem; }
.timeline-item::before { content: ''; position: absolute; left: -1.5rem; top: 0.35rem; width: 7px; height: 7px; border-radius: 50%; background: var(--fg-3); transform: translateX(-3px); }
.timeline-item.done::before { background: var(--green); }
.timeline-item.next::before { background: var(--bg); border: 1.5px solid var(--fg-4); }
.timeline-date { font-size: 0.75rem; color: var(--fg-4); text-transform: uppercase; letter-spacing: 0.04em; }
.timeline-item h3 { margin: 0.125rem 0 0.25rem; font-size: 0.9375rem; }
.timeline-item p { margin: 0; font-size: 0.875rem; color: var(--fg-3); }

/* ============= NOTES ============= */
.note { border-radius: 12px; padding: 0.875rem 1.125rem; margin: 1.25rem 0; font-size: 0.875rem; color: var(--fg-2); background: var(--bg-alt); }
.note-warn { background: #fffbeb; color: var(--amber); }

/* ============= API ENDPOINTS ============= */
.endpoint { border: 1px solid var(--border); border-radius: 12px; margin-bottom: 1rem; overflow: hidden; }
.endpoint-header { display: flex; align-items: center; gap: 0.5rem; padding: 0.75rem 1rem; background: var(--bg-alt); border-bottom: 1px solid var(--border); flex-wrap: wrap; }
.endpoint-method { font-size: 0.6875rem; font-weight: 600; padding: 0.15rem 0.5rem; border-radius: 4px; text-transform: uppercase; letter-spacing: 0.03em; }
.method-get { background: #ecfdf5; color: var(--green); }
.method-post { background: #eff6ff; color: #1d4ed8; }
.endpoint-path { font-size: 0.875rem; color: var(--fg); word-break: break-all; }
.endpoint-body { padding: 1rem; }

/* ============= FORMS ============= */
.form-group { margin-bottom: 1rem; }
.form-group label { display: block; font-weight: 500; font-size: 0.875rem; margin-bottom: 0.375rem; }
.form-input {
  width: 100%; padding: 0.5rem 0.875rem;
  border: 1px solid var(--border); border-radius: 8px;
  font: inherit; font-size: 0.875rem; background: var(--bg);
  transition: border-color var(--t);
}
.form-input:focus { outline: none; border-color: var(--fg-3); }
textarea.form-input { min-height: 100px; resize: vertical; }

/* ============= UTILITY ============= */
.prompt { color: var(--fg-3); }
.prompt::before { content: '$ '; color: var(--fg-4); }
.text-center { text-align: center; }
.text-muted { color: var(--fg-3); }

/* ==============================================
   RESPONSIVE — Tablet (max-width: 768px)
   ============================================== */
@media (max-width: 768px) {
  .feature-grid { grid-template-columns: repeat(2, 1fr); }
  .feature-cell-hero { grid-column: 1 / -1; }

  .showcase { grid-template-columns: 1fr; padding: 3.5rem var(--px); gap: 2rem; }
  .showcase-rev { direction: ltr; }
  .showcase h2 { font-size: 1.375rem; }

  .footer-grid { grid-template-columns: repeat(3, 1fr); gap: 1.5rem; }
  .footer-brand { grid-column: 1 / -1; }

  .section { padding: 3.5rem var(--px); }

  table { display: block; overflow-x: auto; -webkit-overflow-scrolling: touch; }
}

/* ==============================================
   RESPONSIVE — Mobile (max-width: 640px)
   ============================================== */
@media (max-width: 640px) {
  :root { --px: 1.25rem; }

  html { font-size: 14px; }

  h1 { font-size: 1.5rem; }
  h2 { font-size: 1.25rem; margin: 2rem 0 0.75rem; }
  h3 { font-size: 1.0625rem; }

  .hero { padding: 3.5rem var(--px) 3rem; }
  .hero h1 { font-size: 2rem; }
  .hero-sub { font-size: 0.9375rem; margin-bottom: 2rem; }

  .stats { grid-template-columns: repeat(2, 1fr); margin-top: 2.5rem; }
  .stat { padding: 1.125rem 0.5rem; }
  .stat-v { font-size: 1.375rem; }

  .feature-grid { grid-template-columns: 1fr; }
  .feature-cell { padding: 1.5rem; }
  .feature-cell-hero { padding: 1.75rem 1.5rem; }
  .feature-cell-hero h2 { font-size: 1.5rem; }
  .feature-cell h3 { font-size: 1.125rem; }

  .showcase { padding: 2.5rem var(--px); gap: 1.5rem; }
  .showcase h2 { font-size: 1.25rem; }
  .showcase-visual { padding: 1rem 1.25rem; font-size: 0.75rem; }

  .page-header { padding: 2.5rem var(--px) 1.75rem; }
  .page-header h1 { font-size: 1.25rem; }

  .content { padding: 2rem var(--px) 3.5rem; }

  .section { padding: 2.5rem var(--px); }
  .sh h2 { font-size: 1.25rem; }

  .footer { padding: 2.5rem var(--px) 1.5rem; }
  .footer-grid { grid-template-columns: 1fr 1fr; gap: 1.25rem; }
  .footer-brand { grid-column: 1 / -1; }

  pre { padding: 1rem; font-size: 0.8125rem; border-radius: 8px; }
  th, td { padding: 0.5rem 0.625rem; font-size: 0.8125rem; }

  .btn { font-size: 0.8125rem; padding: 0.5rem 1rem; }

  .card { padding: 1.25rem; }

  .note { padding: 0.75rem 1rem; font-size: 0.8125rem; }

  .endpoint-header { padding: 0.625rem 0.75rem; }
  .endpoint-body { padding: 0.75rem; }
}
`
