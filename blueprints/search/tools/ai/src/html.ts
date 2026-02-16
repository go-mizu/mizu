import { cssURL } from './asset'
import { renderMarkdown } from './markdown'
import { MODELS } from './config'
import type { Thread, ThreadSummary, SearchResult, Citation, MediaItem, ThinkingStep } from './types'

function esc(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;').replace(/'/g, '&#39;')
}

function relTime(iso: string): string {
  const d = Date.now() - new Date(iso).getTime()
  if (d < 60000) return 'just now'
  if (d < 3600000) return `${Math.floor(d / 60000)}m ago`
  if (d < 86400000) return `${Math.floor(d / 3600000)}h ago`
  if (d < 604800000) return `${Math.floor(d / 86400000)}d ago`
  return new Date(iso).toLocaleDateString()
}

function dateGroup(iso: string): string {
  const d = Date.now() - new Date(iso).getTime()
  if (d < 86400000) return 'Today'
  if (d < 172800000) return 'Yesterday'
  if (d < 604800000) return 'Previous 7 Days'
  if (d < 2592000000) return 'Previous 30 Days'
  return 'Older'
}

const ic = {
  search: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.35-4.35"/></svg>',
  spark: '<svg viewBox="0 0 24 24" fill="currentColor"><path d="M12 2L9.19 8.63 2 9.24l5.46 4.73L5.82 21 12 17.27 18.18 21l-1.64-7.03L22 9.24l-7.19-.61z"/></svg>',
  globe: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="10"/><path d="M2 12h20M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>',
  chat: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>',
  clock: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>',
  trash: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>',
  empty: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/></svg>',
  send: '<svg viewBox="0 0 24 24" fill="currentColor"><path d="M3.478 2.404a.75.75 0 0 0-.926.941l2.432 7.905H13.5a.75.75 0 0 1 0 1.5H4.984l-2.432 7.905a.75.75 0 0 0 .926.94 60.519 60.519 0 0 0 18.445-8.986.75.75 0 0 0 0-1.218A60.517 60.517 0 0 0 3.478 2.404Z"/></svg>',
  plus: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>',
  menu: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="3" y1="12" x2="21" y2="12"/><line x1="3" y1="6" x2="21" y2="6"/><line x1="3" y1="18" x2="21" y2="18"/></svg>',
  close: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>',
  edit: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>',
  copy: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>',
  link: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/></svg>',
  image: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="3" y="3" width="18" height="18" rx="2" ry="2"/><circle cx="8.5" cy="8.5" r="1.5"/><polyline points="21 15 16 10 5 21"/></svg>',
  video: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polygon points="23 7 16 12 23 17 23 7"/><rect x="1" y="5" width="15" height="14" rx="2" ry="2"/></svg>',
  answer: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="10"/><path d="M12 16v-4M12 8h.01"/></svg>',
  play: '<svg viewBox="0 0 24 24" fill="currentColor"><polygon points="5 3 19 12 5 21 5 3"/></svg>',
  chevDown: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="6 9 12 15 18 9"/></svg>',
  brain: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M12 2a7 7 0 0 0-7 7c0 3.5 2.5 6.5 6 7v6h2v-6c3.5-.5 6-3.5 6-7a7 7 0 0 0-7-7z"/><path d="M9 10a3 3 0 0 1 6 0"/></svg>',
}

function renderSidebar(threads: ThreadSummary[], currentThreadId?: string): string {
  // Group threads by date
  const groups = new Map<string, ThreadSummary[]>()
  for (const t of threads) {
    const group = dateGroup(t.updatedAt)
    if (!groups.has(group)) groups.set(group, [])
    groups.get(group)!.push(t)
  }

  const groupsHtml = Array.from(groups.entries()).map(([label, items]) => `
    <div class="sb-group">
      <div class="sb-group-label">${esc(label)}</div>
      ${items.map(t => `
        <a href="/thread/${esc(t.id)}" class="sb-thread${t.id === currentThreadId ? ' active' : ''}" data-id="${esc(t.id)}">
          <span class="sb-thread-title">${esc(t.title)}</span>
          <button class="sb-thread-del" data-del-id="${esc(t.id)}" title="Delete">${ic.trash}</button>
        </a>
      `).join('')}
    </div>
  `).join('')

  return `
    <aside class="sidebar" id="sidebar">
      <div class="sb-header">
        <a href="/" class="sb-logo">${ic.spark} AI Search</a>
        <button class="sb-close" onclick="toggleSidebar()" title="Close">${ic.close}</button>
      </div>
      <a href="/" class="sb-new">${ic.plus} New Thread</a>
      <div class="sb-threads" id="sidebarThreads">
        ${groupsHtml || '<div class="sb-empty">No threads yet</div>'}
      </div>
    </aside>`
}

export function renderLayout(title: string, content: string, opts: {
  isHome?: boolean
  query?: string
  threads?: ThreadSummary[]
  currentThreadId?: string
} = {}): string {
  const threads = opts.threads || []
  const sidebar = renderSidebar(threads, opts.currentThreadId)

  return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>${esc(title)}</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap" rel="stylesheet">
<link rel="stylesheet" href="${cssURL}">
<link rel="icon" href="data:image/svg+xml,<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 100 100'><text y='.9em' font-size='90'>✦</text></svg>">
</head>
<body${opts.isHome ? ' class="home-page"' : ''}>
${sidebar}
<div class="overlay" id="overlay" onclick="toggleSidebar()"></div>
<main class="main" id="main">
  <button class="menu-btn" onclick="toggleSidebar()" title="Menu">${ic.menu}</button>
  ${content}
</main>
${renderClientScript()}
</body>
</html>`
}

export function renderHomePage(threads: ThreadSummary[] = []): string {
  return `
    <div class="home">
      <div class="home-center">
        <div class="home-h">${ic.spark}</div>
        <h1 class="home-title">What do you want to know?</h1>
        <div class="sb-box">
          <div class="sb-row" id="homeSearchBox">
            <input type="text" id="homeInput" class="sb-input" placeholder="Ask anything..." autofocus autocomplete="off">
            <button type="button" class="sb-btn" id="homeSubmit">${ic.send}</button>
          </div>
          <div class="mt">
            ${MODELS.map(m => `
              <label class="mc${m.id === 'auto' ? ' on' : ''}" title="${esc(m.desc)}">
                <input type="radio" name="m" value="${esc(m.id)}" ${m.id === 'auto' ? 'checked' : ''}>
                ${esc(m.name)}
              </label>
            `).join('')}
          </div>
        </div>
      </div>
    </div>`
}

function renderSourceCards(citations: Citation[]): string {
  if (!citations.length) return ''
  return `
    <div class="src-row">
      ${citations.map((c, i) => `
        <a href="${esc(c.url)}" target="_blank" rel="noopener" class="src-card" data-idx="${i + 1}" data-snippet="${esc(c.snippet || '')}" data-thumb="${esc(c.thumbnail || '')}" data-url="${esc(c.url)}" style="animation-delay:${i * 50}ms">
          ${c.thumbnail ? `
            <div class="src-card-img">
              <div class="src-card-num">${i + 1}</div>
              <img src="${esc(c.thumbnail)}" alt="" loading="lazy">
            </div>
            <div class="src-card-body">
              <div class="src-card-head">
                <img src="${esc(c.favicon)}" alt="" loading="lazy" class="src-card-ico">
              </div>
              <div class="src-card-title">${esc(c.title)}</div>
              <div class="src-card-domain">${esc(c.domain)}</div>
            </div>
          ` : `
            <div class="src-card-body">
              <div class="src-card-head">
                <div class="src-card-num">${i + 1}</div>
                <img src="${esc(c.favicon)}" alt="" loading="lazy" class="src-card-ico">
              </div>
              <div class="src-card-title">${esc(c.title)}</div>
              <div class="src-card-domain">${esc(c.domain)}</div>
            </div>
          `}
        </a>
      `).join('')}
    </div>`
}

function renderThinkingSteps(steps: ThinkingStep[]): string {
  if (!steps.length) return ''
  return `
    <div class="thinking-section">
      <button class="thinking-toggle" onclick="this.classList.toggle('open');this.nextElementSibling.classList.toggle('open')">
        ${ic.brain}
        <span class="thinking-label">Thinking <span class="thinking-count">${steps.length} step${steps.length > 1 ? 's' : ''}</span></span>
        ${ic.chevDown}
      </button>
      <div class="thinking-steps">
        ${steps.map(s => `
          <div class="thinking-step">
            <span class="thinking-step-type" data-type="${esc(s.stepType)}">${esc(formatStepType(s.stepType))}</span>
            <span class="thinking-step-content">${esc(s.content.length > 400 ? s.content.slice(0, 400) + '...' : s.content)}</span>
            ${s.timestamp ? `<span class="thinking-step-time">${(s.timestamp / 1000).toFixed(1)}s</span>` : ''}
          </div>
        `).join('')}
      </div>
    </div>`
}

function formatStepType(stepType: string): string {
  const labels: Record<string, string> = {
    'INITIAL_QUERY': 'Query',
    'INITIAL': 'Query',
    'SEARCH_WEB': 'Search',
    'SEARCH_RESULTS': 'Sources',
    'READING': 'Reading',
    'READ_RESULTS': 'Reading',
    'THINKING': 'Thinking',
    'REASONING': 'Reasoning',
    'ANALYZE': 'Analysis',
    'REWRITE_QUERY': 'Refine',
  }
  return labels[stepType] || stepType.toLowerCase().replace(/_/g, ' ')
}

function renderAssistantMessage(
  content: string,
  citations: Citation[],
  images: MediaItem[],
  videos: MediaItem[],
  mode: string,
  msgIdx: number,
  relatedQueries?: string[],
  threadId?: string,
  isLast?: boolean,
  thinkingSteps?: ThinkingStep[],
): string {
  const n = citations.length
  const hasLinks = citations.length > 0
  const hasImages = images.length > 0
  const hasVideos = videos.length > 0
  const hasTabs = hasLinks || hasImages || hasVideos
  const sources = renderSourceCards(citations)
  const thinking = thinkingSteps?.length ? renderThinkingSteps(thinkingSteps) : ''

  const answerInner = `
    ${thinking}
    ${sources}
    <div class="ans-header">
      <span class="ans-icon">${ic.spark}</span>
      <span class="ans-label">Answer</span>
      <span class="badge">${esc(mode)}</span>
    </div>
    <div class="ans-body">${renderMarkdown(content, n)}</div>`

  const related = (isLast && relatedQueries && relatedQueries.length > 0 && threadId) ? `
    <div class="related">
      ${relatedQueries.map(q => `
        <button class="related-btn" onclick="askFollowUp(this.textContent)">${esc(q)}</button>
      `).join('')}
    </div>` : ''

  if (!hasTabs) {
    return `
      <div class="msg msg-ai">
        <div class="msg-ai-inner">${answerInner}</div>
        ${related}
      </div>`
  }

  const linksPanel = hasLinks ? `
    <div class="tab-content" id="links-${msgIdx}">
      <div class="links-grid">
        ${citations.map((c, i) => `
          <a href="${esc(c.url)}" target="_blank" rel="noopener" class="link-card" data-og-url="${esc(c.url)}">
            ${c.thumbnail ? `<div class="link-card-img"><img src="${esc(c.thumbnail)}" alt="" loading="lazy"></div>` : `<div class="link-card-img" data-needs-og="true"></div>`}
            <div class="link-card-body">
              <div class="link-card-head">
                <img src="${esc(c.favicon)}" alt="" loading="lazy">
                <span class="link-card-num">${i + 1}</span>
              </div>
              <div class="link-card-title">${esc(c.title)}</div>
              <div class="link-card-domain">${esc(c.domain)}</div>
              ${c.snippet ? `<div class="link-card-snippet">${esc(c.snippet)}</div>` : ''}
            </div>
          </a>
        `).join('')}
      </div>
    </div>` : ''

  const imagesPanel = hasImages ? `
    <div class="tab-content" id="images-${msgIdx}">
      <div class="images-grid">
        ${images.map(img => `
          <a href="${esc(img.sourceUrl || img.url)}" target="_blank" rel="noopener" class="img-card">
            <img src="${esc(img.url)}" alt="${esc(img.title || '')}" loading="lazy">
            ${img.title ? `<div class="img-card-title">${esc(img.title)}</div>` : ''}
          </a>
        `).join('')}
      </div>
    </div>` : ''

  const videosPanel = hasVideos ? `
    <div class="tab-content" id="videos-${msgIdx}">
      <div class="videos-grid">
        ${videos.map(vid => `
          <a href="${esc(vid.url)}" target="_blank" rel="noopener" class="vid-card">
            <div class="vid-thumb">
              ${vid.thumbnail ? `<img src="${esc(vid.thumbnail)}" alt="" loading="lazy">` : '<div class="vid-thumb-ph"></div>'}
              <div class="vid-play">${ic.play}</div>
              ${vid.duration ? `<span class="vid-dur">${esc(vid.duration)}</span>` : ''}
            </div>
            ${vid.title ? `<div class="vid-title">${esc(vid.title)}</div>` : ''}
          </a>
        `).join('')}
      </div>
    </div>` : ''

  return `
    <div class="msg msg-ai">
      <div class="tabs" data-msg="${msgIdx}">
        <div class="tab-bar">
          <button class="tab active" data-tab="answer-${msgIdx}">${ic.answer} Answer</button>
          ${hasLinks ? `<button class="tab" data-tab="links-${msgIdx}">${ic.link} Links</button>` : ''}
          ${hasImages ? `<button class="tab" data-tab="images-${msgIdx}">${ic.image} Images</button>` : ''}
          ${hasVideos ? `<button class="tab" data-tab="videos-${msgIdx}">${ic.video} Videos</button>` : ''}
        </div>
        <div class="tab-content active" id="answer-${msgIdx}">
          <div class="msg-ai-inner">${answerInner}</div>
        </div>
        ${linksPanel}
        ${imagesPanel}
        ${videosPanel}
      </div>
      ${related}
    </div>`
}

export function renderSearchResults(result: SearchResult, threadId: string): string {
  const userMsg = `
    <div class="msg msg-user">
      <div class="msg-user-actions">
        <button class="msg-action" onclick="navigator.clipboard.writeText(this.closest('.msg').querySelector('.msg-user-text').textContent)" title="Copy">${ic.copy}</button>
      </div>
      <div class="msg-user-text">${esc(result.query)}</div>
    </div>`

  const aiMsg = renderAssistantMessage(
    result.answer,
    result.citations,
    result.images || [],
    result.videos || [],
    result.mode,
    0,
    result.relatedQueries,
    threadId,
    true,
    result.thinkingSteps || [],
  )

  return `
    <div class="thread-view" id="threadView" data-thread-id="${esc(threadId)}" data-mode="${esc(result.mode)}">
      <div class="messages" id="messages">
        ${userMsg}
        ${aiMsg}
      </div>
      ${renderFollowUp(threadId, result.mode)}
    </div>`
}

export function renderThreadPage(thread: Thread): string {
  const msgs = thread.messages.map((msg, i) => {
    if (msg.role === 'user') {
      return `
        <div class="msg msg-user">
          <div class="msg-user-actions">
            <button class="msg-action" onclick="navigator.clipboard.writeText(this.closest('.msg').querySelector('.msg-user-text').textContent)" title="Copy">${ic.copy}</button>
          </div>
          <div class="msg-user-text">${esc(msg.content)}</div>
        </div>`
    }

    const cites = msg.citations || []
    const images = msg.images || []
    const videos = msg.videos || []
    const isLast = i === thread.messages.length - 1
    const msgIdx = Math.floor(i / 2)

    return renderAssistantMessage(
      msg.content,
      cites,
      images,
      videos,
      msg.model || thread.mode,
      msgIdx,
      msg.relatedQueries,
      thread.id,
      isLast,
      msg.thinkingSteps || [],
    )
  }).join('')

  return `
    <div class="thread-view" id="threadView" data-thread-id="${esc(thread.id)}" data-mode="${esc(thread.mode)}">
      <div class="messages" id="messages">
        ${msgs}
      </div>
      ${renderFollowUp(thread.id, thread.mode)}
    </div>`
}

function renderFollowUp(threadId: string, mode: string): string {
  return `
    <div class="followup" id="followup">
      <div class="followup-box">
        <input type="text" id="followupInput" class="followup-input" placeholder="Ask a follow-up..." autocomplete="off">
        <div class="followup-actions">
          <div class="followup-mode">
            <button class="followup-mode-btn" id="modeBtn" onclick="toggleModeMenu()">
              <span id="modeBtnText">${esc(mode === 'auto' ? 'Auto' : mode.charAt(0).toUpperCase() + mode.slice(1))}</span>
              ${ic.chevDown}
            </button>
            <div class="mode-menu" id="modeMenu">
              ${MODELS.map(m => `
                <button class="mode-option${m.id === mode ? ' active' : ''}" data-mode="${esc(m.id)}" onclick="selectMode('${esc(m.id)}','${esc(m.name)}')">${esc(m.name)}</button>
              `).join('')}
            </div>
          </div>
          <button type="button" class="followup-send" id="followupSend">${ic.send}</button>
        </div>
      </div>
    </div>`
}

export function renderHistoryPage(threads: ThreadSummary[]): string {
  if (threads.length === 0) {
    return `
      <div class="history">
        <h1>History</h1>
        <div class="history-empty">
          ${ic.empty}
          <p>No search history yet</p>
          <a href="/" class="btn-primary">Start searching</a>
        </div>
      </div>`
  }

  return `
    <div class="history">
      <h1>History</h1>
      <div class="history-list">
        ${threads.map(t => `
          <a href="/thread/${esc(t.id)}" class="history-item">
            <div class="history-item-body">
              <div class="history-item-title">${esc(t.title)}</div>
              <div class="history-item-meta">
                <span class="badge">${esc(t.mode)}</span>
                <span>${relTime(t.updatedAt)}</span>
                <span>${t.messageCount} messages</span>
              </div>
            </div>
            <button class="history-item-del" data-del-id="${esc(t.id)}" title="Delete">${ic.trash}</button>
          </a>
        `).join('')}
      </div>
    </div>`
}

export function renderError(title: string, message: string): string {
  return `
    <div class="error-page">
      <h1>${esc(title)}</h1>
      <p>${esc(message)}</p>
      <a href="/" class="btn-primary">Back to home</a>
    </div>`
}

function renderClientScript(): string {
  // SVG icons for client-side use (pre-escaped for JS strings)
  const jsCopy = ic.copy.replace(/'/g, "\\'").replace(/\n/g, '')
  const jsSpark = ic.spark.replace(/'/g, "\\'").replace(/\n/g, '')
  const jsTrash = ic.trash.replace(/'/g, "\\'").replace(/\n/g, '')
  const jsSend = ic.send.replace(/'/g, "\\'").replace(/\n/g, '')
  const jsChev = ic.chevDown.replace(/'/g, "\\'").replace(/\n/g, '')
  const jsAnswer = ic.answer.replace(/'/g, "\\'").replace(/\n/g, '')
  const jsLink = ic.link.replace(/'/g, "\\'").replace(/\n/g, '')
  const jsImage = ic.image.replace(/'/g, "\\'").replace(/\n/g, '')
  const jsVideo = ic.video.replace(/'/g, "\\'").replace(/\n/g, '')
  const jsPlay = ic.play.replace(/'/g, "\\'").replace(/\n/g, '')
  const jsBrain = ic.brain.replace(/'/g, "\\'").replace(/\n/g, '')
  const jsModels = JSON.stringify(MODELS)

  return `<script>
// --- SVG icons ---
var _ic={copy:'${jsCopy}',spark:'${jsSpark}',trash:'${jsTrash}',send:'${jsSend}',chev:'${jsChev}',answer:'${jsAnswer}',link:'${jsLink}',image:'${jsImage}',video:'${jsVideo}',play:'${jsPlay}',brain:'${jsBrain}'};
var _models=${jsModels};

// --- Sidebar ---
function toggleSidebar(){
  document.getElementById('sidebar').classList.toggle('open');
  document.getElementById('overlay').classList.toggle('open');
}

// --- Thread deletion ---
async function delThread(id,btn){
  if(!confirm('Delete this thread?'))return;
  var r=await fetch('/api/thread/'+id,{method:'DELETE'});
  if(r.ok){
    var el=btn.closest('.sb-thread')||btn.closest('.history-item')||btn.closest('li');
    if(el)el.remove();
  }
}

// --- Tabs (event delegation) ---
document.addEventListener('click',function(e){
  var tab=e.target.closest('.tab');
  if(!tab)return;
  var bar=tab.closest('.tab-bar');
  var tabs=tab.closest('.tabs');
  if(!bar||!tabs)return;
  bar.querySelectorAll('.tab').forEach(function(t){t.classList.remove('active');});
  tab.classList.add('active');
  tabs.querySelectorAll(':scope > .tab-content').forEach(function(c){c.classList.remove('active');});
  var target=document.getElementById(tab.dataset.tab);
  if(target)target.classList.add('active');
});

// --- Mode selector ---
var currentMode='${MODELS[0].id}';
function toggleModeMenu(){
  var m=document.getElementById('modeMenu');
  if(m)m.classList.toggle('open');
}
function selectMode(mode,name){
  currentMode=mode;
  var btn=document.getElementById('modeBtnText');
  if(btn)btn.textContent=name;
  var m=document.getElementById('modeMenu');
  if(m)m.classList.remove('open');
  document.querySelectorAll('.mode-option').forEach(function(o){o.classList.toggle('active',o.dataset.mode===mode);});
}
document.addEventListener('click',function(e){
  if(!e.target.closest('.followup-mode')){var m=document.getElementById('modeMenu');if(m)m.classList.remove('open');}
});

// --- Model toggle (home) ---
document.querySelectorAll('.mc input').forEach(function(r){
  r.addEventListener('change',function(){
    currentMode=r.value;
    document.querySelectorAll('.mc').forEach(function(c){c.classList.remove('on');});
    r.parentElement.classList.add('on');
  });
});

// --- Citation hover preview (enhanced with snippet + thumbnail) ---
var _citeTimer=null;
document.addEventListener('mouseover',function(e){
  var cr=e.target.closest('.cr');
  if(!cr)return;
  if(_citeTimer){clearTimeout(_citeTimer);_citeTimer=null;}
  var num=parseInt(cr.textContent);
  var msg=cr.closest('.msg-ai');
  if(!msg)return;
  var card=msg.querySelector('.src-card[data-idx="'+num+'"]');
  if(!card)return;
  var hc=document.getElementById('citePreview');
  if(!hc){hc=document.createElement('div');hc.id='citePreview';hc.className='cite-preview';document.body.appendChild(hc);}
  var t=card.querySelector('.src-card-title');
  var d=card.querySelector('.src-card-domain');
  var ico=card.querySelector('.src-card-ico');
  var snippet=card.getAttribute('data-snippet')||'';
  var thumb=card.getAttribute('data-thumb')||'';
  hc.textContent='';
  // Thumbnail image at top
  if(thumb){
    var ti=document.createElement('div');ti.className='cp-thumb';
    var timg=document.createElement('img');timg.src=thumb;timg.alt='';timg.loading='lazy';
    timg.onerror=function(){ti.style.display='none';};
    ti.appendChild(timg);hc.appendChild(ti);
  }
  // Body wrapper
  var bodyWrap=document.createElement('div');bodyWrap.className='cp-body';
  // Domain + favicon row
  var dd=document.createElement('div');dd.className='cp-domain';
  if(ico){var im=document.createElement('img');im.src=ico.src;im.alt='';dd.appendChild(im);}
  dd.appendChild(document.createTextNode(d?d.textContent:''));
  bodyWrap.appendChild(dd);
  // Title
  var tt=document.createElement('div');tt.className='cp-title';tt.textContent=t?t.textContent:'';bodyWrap.appendChild(tt);
  // Snippet
  if(snippet){
    var ss=document.createElement('div');ss.className='cp-snippet';ss.textContent=snippet.length>180?snippet.slice(0,180)+'...':snippet;bodyWrap.appendChild(ss);
  }
  // URL
  var uu=document.createElement('div');uu.className='cp-url';uu.textContent=card.href||'';bodyWrap.appendChild(uu);
  hc.appendChild(bodyWrap);
  hc.style.display='block';
  // Position centered on citation, clamped to viewport
  var rect=cr.getBoundingClientRect();
  var pw=Math.min(360,window.innerWidth-24);
  var left=rect.left+rect.width/2-pw/2;
  left=Math.max(12,Math.min(left,window.innerWidth-pw-12));
  hc.style.width=pw+'px';
  hc.style.left=left+'px';
  hc.style.top=(rect.top-hc.offsetHeight-8+window.scrollY)+'px';
  if(rect.top-hc.offsetHeight-8<window.scrollY)hc.style.top=(rect.bottom+8+window.scrollY)+'px';
});
document.addEventListener('mouseout',function(e){
  if(!e.target.closest('.cr'))return;
  _citeTimer=setTimeout(function(){
    var hc=document.getElementById('citePreview');
    if(hc)hc.style.display='none';
  },150);
});
// Keep preview open when hovering over it
document.addEventListener('mouseover',function(e){
  if(e.target.closest('#citePreview')&&_citeTimer){clearTimeout(_citeTimer);_citeTimer=null;}
});
document.addEventListener('mouseout',function(e){
  if(e.target.closest('#citePreview')){
    _citeTimer=setTimeout(function(){
      var hc=document.getElementById('citePreview');
      if(hc)hc.style.display='none';
    },150);
  }
});
function escHtml(s){return s.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;').replace(/'/g,'&#39;');}

// --- Markdown renderer (client-side for streaming, line-based with table support) ---
function clientRenderMd(md,citeCount){
  if(!md)return '';
  var lines=md.split('\\n');
  var out=[];
  var inCode=false,codeBuf=[],codeLang='',inList='',inBQ=false;
  function closeList(){if(inList){out.push(inList==='ul'?'</ul>':'</ol>');inList='';}}
  function closeBQ(){if(inBQ){out.push('</blockquote>');inBQ=false;}}
  function inl(s){
    s=s.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;');
    s=s.replace(/\`([^\`]+)\`/g,'<code>$1</code>');
    if(citeCount>0){s=s.replace(/\\[(\\d+)\\]/g,function(_,n){var num=parseInt(n);if(num>=1&&num<=citeCount)return '<a class="cr" title="Source '+num+'">'+num+'</a>';return '['+n+']';});}
    s=s.replace(/\\*\\*(.+?)\\*\\*/g,'<strong>$1</strong>');
    s=s.replace(/\\*(.+?)\\*/g,'<em>$1</em>');
    s=s.replace(/_(.+?)_/g,'<em>$1</em>');
    s=s.replace(/~~(.+?)~~/g,'<del>$1</del>');
    s=s.replace(/\\[([^\\]]+)\\]\\(([^)]+)\\)/g,'<a href="$2" target="_blank" rel="noopener">$1</a>');
    return s;
  }
  for(var i=0;i<lines.length;i++){
    var line=lines[i];
    // Code blocks
    if(line.indexOf('\`\`\`')===0){
      if(inCode){var esc3=codeBuf.join('\\n').replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;');out.push('<div class="cb"><div class="cb-h"><span>'+escHtml(codeLang||'code')+'</span></div><pre><code>'+esc3+'</code></pre></div>');inCode=false;codeBuf=[];codeLang='';}
      else{closeList();closeBQ();inCode=true;codeLang=line.slice(3).trim();}
      continue;
    }
    if(inCode){codeBuf.push(line);continue;}
    // Blank line
    if(!line.trim()){closeList();closeBQ();continue;}
    // Blockquote
    if(line.indexOf('> ')===0){closeList();if(!inBQ){out.push('<blockquote>');inBQ=true;}out.push('<p>'+inl(line.slice(2))+'</p>');continue;}
    closeBQ();
    // Headers
    var hm=line.match(/^(#{1,6})\\s+(.+)/);
    if(hm){closeList();out.push('<h'+hm[1].length+'>'+inl(hm[2])+'</h'+hm[1].length+'>');continue;}
    // HR
    if(/^[-*_]{3,}\\s*$/.test(line)){closeList();out.push('<hr>');continue;}
    // Unordered list
    var ulm=line.match(/^\\s*[-*+]\\s+(.+)/);
    if(ulm){if(inList!=='ul'){closeList();out.push('<ul>');inList='ul';}out.push('<li>'+inl(ulm[1])+'</li>');continue;}
    // Ordered list
    var olm=line.match(/^\\s*\\d+[.)]\\s+(.+)/);
    if(olm){if(inList!=='ol'){closeList();out.push('<ol>');inList='ol';}out.push('<li>'+inl(olm[1])+'</li>');continue;}
    // Table
    if(line.indexOf('|')>=0&&line.trim().indexOf('|')===0){
      closeList();
      var rows=[];var j=i;
      while(j<lines.length&&lines[j].trim().indexOf('|')===0){
        var cells=lines[j].trim().replace(/^\\||\\|$/g,'').split('|').map(function(c){return c.trim();});
        if(!/^[-:\\s|]+$/.test(lines[j]))rows.push(cells);
        j++;
      }
      if(rows.length>0){
        out.push('<div class="table-wrap"><table>');
        out.push('<thead><tr>'+rows[0].map(function(c){return '<th>'+inl(c)+'</th>';}).join('')+'</tr></thead>');
        if(rows.length>1){out.push('<tbody>');for(var k=1;k<rows.length;k++){out.push('<tr>'+rows[k].map(function(c){return '<td>'+inl(c)+'</td>';}).join('')+'</tr>');}out.push('</tbody>');}
        out.push('</table></div>');
      }
      i=j-1;continue;
    }
    // Paragraph
    closeList();
    out.push('<p>'+inl(line)+'</p>');
  }
  closeList();closeBQ();
  if(inCode&&codeBuf.length){var esc4=codeBuf.join('\\n').replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;');out.push('<div class="cb"><pre><code>'+esc4+'</code></pre></div>');}
  return out.join('');
}

// --- Build full tabbed message (used after streaming completes) ---
function buildTabbedMessage(answer,citations,images,videos,mode,msgIdx,citeCount,steps){
  var hasLinks=citations.length>0;
  var hasImages=images.length>0;
  var hasVideos=videos.length>0;
  var hasTabs=hasLinks||hasImages||hasVideos;

  // Thinking steps
  var _stepLabels={INITIAL_QUERY:'Query',INITIAL:'Query',SEARCH_WEB:'Search',SEARCH_RESULTS:'Sources',READING:'Reading',READ_RESULTS:'Reading',THINKING:'Thinking',REASONING:'Reasoning',ANALYZE:'Analysis',REWRITE_QUERY:'Refine'};
  function fmtStep(t){return _stepLabels[t]||t.toLowerCase().replace(/_/g,' ');}
  var thinkHtml='';
  if(steps&&steps.length>0){
    thinkHtml='<div class="thinking-section"><button class="thinking-toggle" onclick="this.classList.toggle(\\x27open\\x27);this.nextElementSibling.classList.toggle(\\x27open\\x27)">'+_ic.brain+'<span class="thinking-label">Thinking <span class="thinking-count">'+steps.length+' step'+(steps.length>1?'s':'')+'</span></span>'+_ic.chev+'</button><div class="thinking-steps">';
    steps.forEach(function(s){
      thinkHtml+='<div class="thinking-step"><span class="thinking-step-type" data-type="'+escHtml(s.stepType)+'">'+escHtml(fmtStep(s.stepType))+'</span><span class="thinking-step-content">'+escHtml(s.content.length>400?s.content.slice(0,400)+'...':s.content)+'</span>'+(s.timestamp?'<span class="thinking-step-time">'+(s.timestamp/1000).toFixed(1)+'s</span>':'')+'</div>';
    });
    thinkHtml+='</div></div>';
  }

  // Source cards row
  var srcHtml=renderSourceCardsClient(citations);

  // Answer inner
  var ansInner=thinkHtml+srcHtml+'<div class="ans-header"><span class="ans-icon">'+_ic.spark+'</span><span class="ans-label">Answer</span><span class="badge">'+escHtml(mode)+'</span></div><div class="ans-body">'+clientRenderMd(answer,citeCount)+'</div>';

  if(!hasTabs){
    return '<div class="msg-ai-inner">'+ansInner+'</div>';
  }

  // Build tabs
  var tabBar='<div class="tab-bar"><button class="tab active" data-tab="answer-'+msgIdx+'">'+_ic.answer+' Answer</button>';
  if(hasLinks)tabBar+='<button class="tab" data-tab="links-'+msgIdx+'">'+_ic.link+' Links <span class="tab-count">'+citations.length+'</span></button>';
  if(hasImages)tabBar+='<button class="tab" data-tab="images-'+msgIdx+'">'+_ic.image+' Images <span class="tab-count">'+images.length+'</span></button>';
  if(hasVideos)tabBar+='<button class="tab" data-tab="videos-'+msgIdx+'">'+_ic.video+' Videos <span class="tab-count">'+videos.length+'</span></button>';
  tabBar+='</div>';

  // Answer panel
  var ansPanel='<div class="tab-content active" id="answer-'+msgIdx+'"><div class="msg-ai-inner">'+ansInner+'</div></div>';

  // Links panel
  var linksPanel='';
  if(hasLinks){
    linksPanel='<div class="tab-content" id="links-'+msgIdx+'"><div class="links-grid">';
    citations.forEach(function(c,i){
      var thumbHtml=c.thumbnail?'<div class="link-card-img"><img src="'+escHtml(c.thumbnail)+'" alt="" loading="lazy"></div>':'<div class="link-card-img" data-needs-og="true"></div>';
      linksPanel+='<a href="'+escHtml(c.url)+'" target="_blank" rel="noopener" class="link-card" data-og-url="'+escHtml(c.url)+'">'+thumbHtml+'<div class="link-card-body"><div class="link-card-head"><img src="'+escHtml(c.favicon)+'" alt="" loading="lazy"><span class="link-card-num">'+(i+1)+'</span></div><div class="link-card-title">'+escHtml(c.title)+'</div><div class="link-card-domain">'+escHtml(c.domain)+'</div>'+(c.snippet?'<div class="link-card-snippet">'+escHtml(c.snippet)+'</div>':'')+'</div></a>';
    });
    linksPanel+='</div></div>';
  }

  // Images panel
  var imagesPanel='';
  if(hasImages){
    imagesPanel='<div class="tab-content" id="images-'+msgIdx+'"><div class="images-grid">';
    images.forEach(function(img){
      imagesPanel+='<a href="'+escHtml(img.sourceUrl||img.url)+'" target="_blank" rel="noopener" class="img-card"><img src="'+escHtml(img.url)+'" alt="'+escHtml(img.title||'')+'" loading="lazy">'+(img.title?'<div class="img-card-title">'+escHtml(img.title)+'</div>':'')+'</a>';
    });
    imagesPanel+='</div></div>';
  }

  // Videos panel
  var videosPanel='';
  if(hasVideos){
    videosPanel='<div class="tab-content" id="videos-'+msgIdx+'"><div class="videos-grid">';
    videos.forEach(function(vid){
      videosPanel+='<a href="'+escHtml(vid.url)+'" target="_blank" rel="noopener" class="vid-card"><div class="vid-thumb">'+(vid.thumbnail?'<img src="'+escHtml(vid.thumbnail)+'" alt="" loading="lazy">':'<div class="vid-thumb-ph"></div>')+'<div class="vid-play">'+_ic.play+'</div>'+(vid.duration?'<span class="vid-dur">'+escHtml(vid.duration)+'</span>':'')+'</div>'+(vid.title?'<div class="vid-title">'+escHtml(vid.title)+'</div>':'')+'</a>';
    });
    videosPanel+='</div></div>';
  }

  return '<div class="tabs" data-msg="'+msgIdx+'">'+tabBar+ansPanel+linksPanel+imagesPanel+videosPanel+'</div>';
}

// --- Session pre-warm (reduces TTFB by ~400ms on first search) ---
fetch('/api/warm').catch(function(){});

// --- Streaming search ---
var streamingAbort=null;
var _streamMsgIdx=100; // Start high to avoid collision with server-rendered indexes
async function doStreamSearch(query,mode,threadId){
  if(streamingAbort)streamingAbort.abort();
  streamingAbort=new AbortController();
  var msgIdx=_streamMsgIdx++;
  var searchT0=Date.now();

  var messagesEl=document.getElementById('messages');
  if(!messagesEl)return;

  // Add user message (using DOM instead of innerHTML to avoid escaping issues)
  var userDiv=document.createElement('div');
  userDiv.className='msg msg-user';
  var actDiv=document.createElement('div');actDiv.className='msg-user-actions';
  var cpBtn=document.createElement('button');cpBtn.className='msg-action';cpBtn.title='Copy';cpBtn.innerHTML=_ic.copy;
  cpBtn.onclick=function(){navigator.clipboard.writeText(this.closest('.msg').querySelector('.msg-user-text').textContent);};
  actDiv.appendChild(cpBtn);
  userDiv.appendChild(actDiv);
  var utDiv=document.createElement('div');utDiv.className='msg-user-text';utDiv.textContent=query;
  userDiv.appendChild(utDiv);
  messagesEl.appendChild(userDiv);

  // Add streaming AI message with progress indicator + live timer
  var aiHtml='<div class="msg msg-ai" id="streaming-msg">'
    +'<div class="search-progress" id="streaming-progress">'
    +'<div class="progress-shimmer"></div>'
    +'<div class="progress-inner">'
    +'<div class="progress-dots"><span></span><span></span><span></span></div>'
    +'<span class="progress-text">Searching the web...</span>'
    +'<span class="progress-timer" id="streaming-timer">0.0s</span>'
    +'</div>'
    +'</div>'
    +'<div class="msg-ai-inner" id="streaming-inner" style="display:none">'
    +'<div class="ans-header"><span class="ans-icon">'+_ic.spark+'</span><span class="ans-label">Answer</span><span class="badge">'+escHtml(mode)+'</span></div>'
    +'<div class="ans-body streaming" id="streaming-body"><span class="cursor"></span></div>'
    +'</div>'
    +'</div>';
  messagesEl.insertAdjacentHTML('beforeend',aiHtml);
  scrollToBottom();

  // Live timer that counts up every 100ms
  var timerEl=document.getElementById('streaming-timer');
  var timerInterval=setInterval(function(){
    if(timerEl){var elapsed=((Date.now()-searchT0)/1000).toFixed(1);timerEl.textContent=elapsed+'s';}
  },100);

  var fuI=document.getElementById('followupInput');
  var fuS=document.getElementById('followupSend');
  if(fuI)fuI.disabled=true;
  if(fuS)fuS.disabled=true;

  var citations=[],webResults=[],images=[],videos=[],thinkingSteps=[];
  var fullAnswer='',finalResult=null,serverTiming=null;

  var params=new URLSearchParams({q:query,mode:mode});
  if(threadId)params.set('threadId',threadId);

  try{
    var resp=await fetch('/api/stream?'+params.toString(),{signal:streamingAbort.signal});
    if(!resp.ok){
      var errText=await resp.text();
      throw new Error('HTTP '+resp.status+': '+(errText||'').slice(0,200));
    }
    var reader=resp.body.getReader();
    var decoder=new TextDecoder();
    var buffer='';

    while(true){
      var chunk=await reader.read();
      if(chunk.done)break;
      buffer+=decoder.decode(chunk.value,{stream:true});
      var parts=buffer.split('\\n\\n');
      buffer=parts.pop()||'';

      for(var pi=0;pi<parts.length;pi++){
        var part=parts[pi];
        if(!part.trim())continue;
        var eventMatch=part.match(/^event: (\\w+)/);
        var dataMatch=part.match(/^data: (.+)$/m);
        if(!eventMatch||!dataMatch)continue;
        var evt=eventMatch[1];
        var data;
        try{data=JSON.parse(dataMatch[1]);}catch(ex){continue;}

        if(evt==='progress'){
          // Update progress text
          var pt=document.getElementById('streaming-progress');
          if(pt){
            var ptx=pt.querySelector('.progress-text');
            if(ptx)ptx.textContent=data.message||'Searching...';
          }
        }
        else if(evt==='thinking'){
          var step=data.step;
          if(step){
            thinkingSteps.push(step);
            // Create or update thinking section in the streaming message
            var thinkEl=document.getElementById('streaming-thinking');
            if(!thinkEl){
              thinkEl=document.createElement('div');thinkEl.id='streaming-thinking';thinkEl.className='thinking-section';
              thinkEl.innerHTML='<button class="thinking-toggle open" onclick="this.classList.toggle(\\x27open\\x27);this.nextElementSibling.classList.toggle(\\x27open\\x27)">'+_ic.brain+'<span class="thinking-label">Thinking <span class="thinking-count" id="thinking-count">1 step</span></span>'+_ic.chev+'</button><div class="thinking-steps open thinking-streaming" id="thinking-steps-list"></div>';
              // Insert at top of streaming-inner, or before progress if inner not visible yet
              var inner=document.getElementById('streaming-inner');
              if(inner&&inner.style.display!=='none'){
                inner.insertBefore(thinkEl,inner.firstChild);
              }else{
                var prog=document.getElementById('streaming-progress');
                if(prog)prog.parentNode.insertBefore(thinkEl,prog);
              }
            }
            // Update count
            var countEl=document.getElementById('thinking-count');
            if(countEl)countEl.textContent=thinkingSteps.length+' step'+(thinkingSteps.length>1?'s':'');
            // Add step
            var list=document.getElementById('thinking-steps-list');
            if(list){
              var sd=document.createElement('div');sd.className='thinking-step';
              var _sl={INITIAL_QUERY:'Query',INITIAL:'Query',SEARCH_WEB:'Search',SEARCH_RESULTS:'Sources',READING:'Reading',READ_RESULTS:'Reading',THINKING:'Thinking',REASONING:'Reasoning',ANALYZE:'Analysis',REWRITE_QUERY:'Refine'};
              var stLabel=_sl[step.stepType]||step.stepType.toLowerCase().replace(/_/g,' ');
              sd.innerHTML='<span class="thinking-step-type" data-type="'+escHtml(step.stepType)+'">'+escHtml(stLabel)+'</span><span class="thinking-step-content">'+escHtml(step.content.length>400?step.content.slice(0,400)+'...':step.content)+'</span>'+(step.timestamp?'<span class="thinking-step-time">'+(step.timestamp/1000).toFixed(1)+'s</span>':'');
              list.appendChild(sd);
            }
            scrollToBottom();
          }
        }
        else if(evt==='sources'){
          citations=data.citations||[];
          webResults=data.webResults||[];
          // Fade out progress, show answer area with sources
          var prog=document.getElementById('streaming-progress');
          if(prog){prog.classList.add('fade-out');setTimeout(function(){prog.style.display='none';},300);}
          var inner=document.getElementById('streaming-inner');
          if(inner){
            setTimeout(function(){
              inner.style.display='';
              inner.classList.add('fade-in');
              inner.insertAdjacentHTML('afterbegin',renderSourceCardsClient(citations));
            },200);
          }
          scrollToBottom();
        }
        else if(evt==='chunk'){
          fullAnswer=data.full||'';
          // If progress still showing (no sources), hide it
          var prog2=document.getElementById('streaming-progress');
          if(prog2&&prog2.style.display!=='none'){
            prog2.style.display='none';
            var inner2=document.getElementById('streaming-inner');
            if(inner2)inner2.style.display='';
          }
          var body=document.getElementById('streaming-body');
          if(body){
            body.innerHTML=clientRenderMd(fullAnswer,citations.length)+'<span class="cursor"></span>';
            scrollToBottom();
          }
        }
        else if(evt==='media'){
          images=data.images||[];
          videos=data.videos||[];
        }
        else if(evt==='related'){
          // Will be rendered after finalize
        }
        else if(evt==='done'){
          finalResult=data.result;
          serverTiming=data.timing||null;
        }
        else if(evt==='error'){
          var prog3=document.getElementById('streaming-progress');
          if(prog3)prog3.style.display='none';
          var inner3=document.getElementById('streaming-inner');
          if(inner3)inner3.style.display='';
          var eb=document.getElementById('streaming-body');
          if(eb){eb.innerHTML='<div class="stream-error">'+escHtml(data.message||'Search failed')+'</div>';eb.classList.remove('streaming');}
        }
      }
    }

    // Stop timer
    clearInterval(timerInterval);timerInterval=null;

    // Finalize: rebuild the full message with tabs
    // Prefer finalResult.answer (server-extracted clean text) over fullAnswer (may be raw JSON)
    var sm=document.getElementById('streaming-msg');
    var displayAnswer=(finalResult&&finalResult.answer)?finalResult.answer:fullAnswer;
    if(sm&&displayAnswer){
      var tabbedHtml=buildTabbedMessage(displayAnswer,citations,images,videos,mode,msgIdx,citations.length,thinkingSteps);
      sm.innerHTML=tabbedHtml;

      // Add timing badge
      var clientTotal=((Date.now()-searchT0)/1000).toFixed(1);
      var timingHtml='<div class="timing-bar">';
      timingHtml+='<span class="timing-total">'+'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>'+'Completed in '+clientTotal+'s</span>';
      if(serverTiming){
        timingHtml+='<span class="timing-detail" title="Session: '+serverTiming.sessionMs+'ms | Fetch: '+serverTiming.fetchMs+'ms | First byte: '+serverTiming.firstByteMs+'ms | First answer: '+serverTiming.firstAnswerMs+'ms | Total: '+serverTiming.totalMs+'ms">';
        timingHtml+='TTFB '+((serverTiming.firstAnswerMs||0)/1000).toFixed(1)+'s';
        timingHtml+='</span>';
      }
      timingHtml+='</div>';
      sm.insertAdjacentHTML('afterbegin',timingHtml);

      // Lazy OG image enrichment for source cards and link cards
      enrichWithOGImages();

      // Add related queries
      if(finalResult&&finalResult.relatedQueries&&finalResult.relatedQueries.length>0){
        var relHtml='<div class="related">'+finalResult.relatedQueries.map(function(q){return '<button class="related-btn" onclick="askFollowUp(this.textContent)">'+escHtml(q)+'</button>';}).join('')+'</div>';
        sm.insertAdjacentHTML('beforeend',relHtml);
      }
      scrollToBottom();
    } else {
      // No answer — just remove streaming class
      var fb=document.getElementById('streaming-body');
      if(fb)fb.classList.remove('streaming');
    }
    // Stop thinking animation
    var tsList=document.querySelector('.thinking-streaming');
    if(tsList)tsList.classList.remove('thinking-streaming');

    // Save thread
    if(finalResult){
      // Ensure thinking steps are included in saved result
      if(thinkingSteps.length>0&&!finalResult.thinkingSteps){finalResult.thinkingSteps=thinkingSteps;}
      var saveResp=await fetch('/api/thread/save',{
        method:'POST',
        headers:{'Content-Type':'application/json'},
        body:JSON.stringify({query:query,mode:mode,threadId:threadId||undefined,result:finalResult})
      });
      var saveData=await saveResp.json();
      if(saveData.thread){
        var tv=document.getElementById('threadView');
        if(tv){
          tv.dataset.threadId=saveData.thread.id;
          if(!threadId)history.pushState(null,'','/thread/'+saveData.thread.id);
        }
        refreshSidebar(saveData.thread.id);
      }
    }
  }catch(e){
    if(e.name!=='AbortError'){
      var prog4=document.getElementById('streaming-progress');
      if(prog4)prog4.style.display='none';
      var inner4=document.getElementById('streaming-inner');
      if(inner4)inner4.style.display='';
      var errb=document.getElementById('streaming-body');
      if(errb){errb.innerHTML='<div class="stream-error">'+escHtml(e.message||'Search failed')+'</div>';errb.classList.remove('streaming');}
    }
  }finally{
    if(timerInterval)clearInterval(timerInterval);
    if(fuI){fuI.disabled=false;fuI.focus();}
    if(fuS)fuS.disabled=false;
    streamingAbort=null;
  }
}

function renderSourceCardsClient(cites){
  if(!cites.length)return '';
  return '<div class="src-row">'+cites.map(function(c,i){
    if(c.thumbnail){
      return '<a href="'+escHtml(c.url)+'" target="_blank" rel="noopener" class="src-card" data-idx="'+(i+1)+'" data-snippet="'+escHtml(c.snippet||'')+'" data-thumb="'+escHtml(c.thumbnail||'')+'" data-url="'+escHtml(c.url)+'" style="animation-delay:'+(i*50)+'ms"><div class="src-card-img"><div class="src-card-num">'+(i+1)+'</div><img src="'+escHtml(c.thumbnail)+'" alt="" loading="lazy"></div><div class="src-card-body"><div class="src-card-head"><img src="'+escHtml(c.favicon)+'" alt="" loading="lazy" class="src-card-ico"></div><div class="src-card-title">'+escHtml(c.title)+'</div><div class="src-card-domain">'+escHtml(c.domain)+'</div></div></a>';
    }
    return '<a href="'+escHtml(c.url)+'" target="_blank" rel="noopener" class="src-card" data-idx="'+(i+1)+'" data-snippet="'+escHtml(c.snippet||'')+'" data-thumb="'+escHtml(c.thumbnail||'')+'" data-url="'+escHtml(c.url)+'" style="animation-delay:'+(i*50)+'ms"><div class="src-card-body"><div class="src-card-head"><div class="src-card-num">'+(i+1)+'</div><img src="'+escHtml(c.favicon)+'" alt="" loading="lazy" class="src-card-ico"></div><div class="src-card-title">'+escHtml(c.title)+'</div><div class="src-card-domain">'+escHtml(c.domain)+'</div></div></a>';
  }).join('')+'</div>';
}

function scrollToBottom(){
  var main=document.getElementById('main');
  if(main)main.scrollTop=main.scrollHeight;
}

// Lazy OG image enrichment: fetch og:image for source/link cards missing thumbnails
function enrichWithOGImages(){
  // Enrich source cards
  document.querySelectorAll('.src-card[data-url]').forEach(function(card){
    if(card.querySelector('.src-card-img img'))return; // already has image
    var url=card.getAttribute('data-url');
    if(!url)return;
    fetch('/api/og?url='+encodeURIComponent(url)).then(function(r){return r.json();}).then(function(og){
      if(!og.image)return;
      card.setAttribute('data-thumb',og.image);
      // Insert image section at top of card
      var imgDiv=document.createElement('div');imgDiv.className='src-card-img';
      var numEl=card.querySelector('.src-card-num');
      if(numEl){var numClone=numEl.cloneNode(true);imgDiv.appendChild(numClone);numEl.style.display='none';}
      var img=document.createElement('img');img.src=og.image;img.alt='';img.loading='lazy';
      img.onerror=function(){imgDiv.remove();if(numEl)numEl.style.display='';};
      imgDiv.appendChild(img);
      card.insertBefore(imgDiv,card.firstChild);
    }).catch(function(){});
  });
  // Enrich link cards missing images
  document.querySelectorAll('.link-card[data-og-url]').forEach(function(card){
    var imgDiv=card.querySelector('.link-card-img[data-needs-og]');
    if(!imgDiv)return;
    var url=card.getAttribute('data-og-url');
    if(!url)return;
    fetch('/api/og?url='+encodeURIComponent(url)).then(function(r){return r.json();}).then(function(og){
      if(!og.image){imgDiv.remove();return;}
      var img=document.createElement('img');img.src=og.image;img.alt='';img.loading='lazy';
      img.onerror=function(){imgDiv.remove();};
      imgDiv.removeAttribute('data-needs-og');
      imgDiv.appendChild(img);
    }).catch(function(){imgDiv.remove();});
  });
}

async function refreshSidebar(currentId){
  try{
    var r=await fetch('/api/threads');
    var d=await r.json();
    var threads=d.threads||[];
    var container=document.getElementById('sidebarThreads');
    if(!container)return;
    if(!threads.length){container.innerHTML='<div class="sb-empty">No threads yet</div>';return;}
    var groups=new Map();
    var now=Date.now();
    threads.forEach(function(t){
      var diff=now-new Date(t.updatedAt).getTime();
      var g='Older';
      if(diff<86400000)g='Today';else if(diff<172800000)g='Yesterday';else if(diff<604800000)g='Previous 7 Days';else if(diff<2592000000)g='Previous 30 Days';
      if(!groups.has(g))groups.set(g,[]);
      groups.get(g).push(t);
    });
    var html='';
    groups.forEach(function(items,label){
      html+='<div class="sb-group"><div class="sb-group-label">'+escHtml(label)+'</div>';
      items.forEach(function(t){
        var active=t.id===currentId?' active':'';
        html+='<a href="/thread/'+escHtml(t.id)+'" class="sb-thread'+active+'" data-id="'+escHtml(t.id)+'">';
        html+='<span class="sb-thread-title">'+escHtml(t.title)+'</span>';
        html+='<button class="sb-thread-del" data-del-id="'+escHtml(t.id)+'" title="Delete">'+_ic.trash+'</button></a>';
      });
      html+='</div>';
    });
    container.innerHTML=html;
  }catch(ex){}
}

// --- Delete via event delegation (sidebar + history) ---
document.addEventListener('click',function(e){
  var delBtn=e.target.closest('[data-del-id]');
  if(!delBtn)return;
  e.preventDefault();e.stopPropagation();
  delThread(delBtn.dataset.delId,delBtn);
});

// --- Home page search (streaming, no page reload) ---
var homeInput=document.getElementById('homeInput');
var homeSubmit=document.getElementById('homeSubmit');
if(homeInput&&homeSubmit){
  function doHomeSearch(){
    var q=homeInput.value.trim();
    if(!q)return;
    document.body.classList.remove('home-page');
    var main=document.getElementById('main');
    var home=main.querySelector('.home');
    if(!home)return;
    // Build follow-up HTML
    var modeLabel=currentMode==='auto'?'Auto':currentMode.charAt(0).toUpperCase()+currentMode.slice(1);
    var modeOpts=_models.map(function(m){
      return '<button class="mode-option'+(m.id===currentMode?' active':'')+'" data-mode="'+escHtml(m.id)+'" onclick="selectMode(\\x27'+escHtml(m.id)+'\\x27,\\x27'+escHtml(m.name)+'\\x27)">'+escHtml(m.name)+'</button>';
    }).join('');
    home.outerHTML='<div class="thread-view" id="threadView" data-mode="'+escHtml(currentMode)+'">'
      +'<div class="messages" id="messages"></div>'
      +'<div class="followup" id="followup"><div class="followup-box">'
      +'<input type="text" id="followupInput" class="followup-input" placeholder="Ask a follow-up..." autocomplete="off">'
      +'<div class="followup-actions">'
      +'<div class="followup-mode"><button class="followup-mode-btn" id="modeBtn" onclick="toggleModeMenu()"><span id="modeBtnText">'+escHtml(modeLabel)+'</span>'+_ic.chev+'</button>'
      +'<div class="mode-menu" id="modeMenu">'+modeOpts+'</div></div>'
      +'<button type="button" class="followup-send" id="followupSend">'+_ic.send+'</button>'
      +'</div></div></div></div>';
    // Rebind follow-up events
    bindFollowUp();
    doStreamSearch(q,currentMode,'');
  }
  homeSubmit.addEventListener('click',doHomeSearch);
  homeInput.addEventListener('keydown',function(e){if(e.key==='Enter')doHomeSearch();});
}

// --- Follow-up ---
function askFollowUp(query){
  var input=document.getElementById('followupInput');
  if(input){input.value=query;submitFollowUp();}
}
function submitFollowUp(){
  var input=document.getElementById('followupInput');
  if(!input)return;
  var query=input.value.trim();
  if(!query)return;
  input.value='';
  var tv=document.getElementById('threadView');
  var threadId=tv?tv.dataset.threadId:'';
  var mode=currentMode||(tv?tv.dataset.mode:'')||'auto';
  doStreamSearch(query,mode,threadId);
}
function bindFollowUp(){
  var fi=document.getElementById('followupInput');
  var fs=document.getElementById('followupSend');
  if(fi)fi.addEventListener('keydown',function(e){if(e.key==='Enter'&&!e.shiftKey){e.preventDefault();submitFollowUp();}});
  if(fs)fs.addEventListener('click',submitFollowUp);
}
bindFollowUp();

// Enrich OG images on page load (for server-rendered results)
if(document.querySelector('.src-card[data-url]'))enrichWithOGImages();

// --- Init mode from thread view ---
(function(){
  var tv=document.getElementById('threadView');
  if(tv&&tv.dataset.mode){
    currentMode=tv.dataset.mode;
    var name=currentMode==='auto'?'Auto':currentMode.charAt(0).toUpperCase()+currentMode.slice(1);
    var btn=document.getElementById('modeBtnText');
    if(btn)btn.textContent=name;
  }
})();
</script>`
}
