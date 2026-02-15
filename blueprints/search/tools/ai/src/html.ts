import { cssURL } from './asset'
import { renderMarkdown } from './markdown'
import { MODELS } from './config'
import type { Thread, ThreadSummary, SearchResult, Citation } from './types'

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

const ic = {
  search: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.35-4.35"/></svg>',
  arrow: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M5 12h14M12 5l7 7-7 7"/></svg>',
  spark: '<svg viewBox="0 0 24 24" fill="currentColor"><path d="M12 2L9.19 8.63 2 9.24l5.46 4.73L5.82 21 12 17.27 18.18 21l-1.64-7.03L22 9.24l-7.19-.61z"/></svg>',
  globe: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="10"/><path d="M2 12h20M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>',
  chat: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>',
  clock: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>',
  trash: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>',
  empty: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/></svg>',
  send: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="22" y1="2" x2="11" y2="13"/><polygon points="22 2 15 22 11 13 2 9 22 2"/></svg>',
}

export function renderLayout(title: string, content: string, opts: { isHome?: boolean; query?: string } = {}): string {
  const nav = opts.isHome ? '' : `
    <header class="hd">
      <div class="hd-in">
        <a href="/" class="hd-logo">${ic.spark} AI Search</a>
        <div class="hd-q">
          <form action="/search" method="get">
            <input type="text" name="q" placeholder="Ask anything..." value="${esc(opts.query || '')}" autocomplete="off">
          </form>
        </div>
        <nav class="hd-nav">
          <a href="/history">${ic.clock} History</a>
        </nav>
      </div>
    </header>`

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
<body>
${nav}
${content}
</body>
</html>`
}

export function renderHomePage(threads: ThreadSummary[] = []): string {
  const threadsHtml = threads.length > 0 ? `
    <div class="rec">
      <h3>Recent</h3>
      ${threads.slice(0, 8).map(t => `
        <a href="/thread/${esc(t.id)}" class="ti">
          <div class="ti-ic">${ic.chat}</div>
          <div class="ti-body">
            <div class="ti-t">${esc(t.title)}</div>
            <div class="ti-m">
              <span class="badge">${esc(t.mode)}</span>
              <span>${relTime(t.updatedAt)}</span>
            </div>
          </div>
        </a>
      `).join('')}
    </div>` : ''

  return `
    <div class="home">
      <div class="home-h">${ic.spark} AI Search</div>
      <div class="home-sub">Ask anything, get answers with sources</div>
      <div class="sb">
        <form action="/search" method="get" id="sf">
          <div class="sb-row">
            <input type="text" name="q" class="sb-input" placeholder="Ask anything..." autofocus autocomplete="off">
            <input type="hidden" name="mode" id="mi" value="auto">
            <button type="submit" class="sb-btn">${ic.send}</button>
          </div>
        </form>
      </div>
      <div class="mt">
        ${MODELS.map(m => `
          <label class="mc${m.id === 'auto' ? ' on' : ''}" title="${esc(m.desc)}">
            <input type="radio" name="m" value="${esc(m.id)}" ${m.id === 'auto' ? 'checked' : ''}>
            ${esc(m.name)}
          </label>
        `).join('')}
      </div>
      ${threadsHtml}
    </div>
    <script>
      document.querySelectorAll('.mc input').forEach(r=>{
        r.addEventListener('change',()=>{
          document.getElementById('mi').value=r.value;
          document.querySelectorAll('.mc').forEach(c=>c.classList.remove('on'));
          r.parentElement.classList.add('on');
        });
      });
    </script>`
}

export function renderSearchResults(result: SearchResult, threadId: string): string {
  const n = result.citations.length

  const sources = n > 0 ? `
    <div class="src-s">
      <div class="lbl">${ic.globe} Sources</div>
      <div class="src-r">
        ${result.citations.map(c => `
          <a href="${esc(c.url)}" target="_blank" rel="noopener" class="src-c">
            <img src="${esc(c.favicon)}" alt="" loading="lazy">
            <div>
              <div class="src-c-t">${esc(c.title)}</div>
              <div class="src-c-d">${esc(c.domain)}</div>
            </div>
          </a>
        `).join('')}
      </div>
    </div>` : ''

  const answer = `
    <div class="ans-s">
      <div class="lbl">${ic.spark} Answer <span class="badge">${esc(result.mode)}</span></div>
      <div class="ans">${renderMarkdown(result.answer, n)}</div>
    </div>`

  const citations = n > 0 ? `
    <div class="cite-s">
      <div class="lbl">Citations</div>
      ${result.citations.map((c, i) => `
        <div class="ci" id="cite-${i + 1}">
          <div class="ci-n">${i + 1}</div>
          <div class="ci-b">
            <a href="${esc(c.url)}" target="_blank" rel="noopener" class="ci-t">
              <img src="${esc(c.favicon)}" alt="" loading="lazy">
              ${esc(c.title)}
            </a>
            <div class="ci-u">${esc(c.url)}</div>
            ${c.snippet ? `<div class="ci-sn">${esc(c.snippet)}</div>` : ''}
          </div>
        </div>
      `).join('')}
    </div>` : ''

  const related = result.relatedQueries.length > 0 ? `
    <div class="rel-s">
      <div class="lbl">Related</div>
      <div class="rel-ch">
        ${result.relatedQueries.map(q => `
          <a href="/search?q=${encodeURIComponent(q)}&mode=${encodeURIComponent(result.mode)}" class="rc">${esc(q)}</a>
        `).join('')}
      </div>
    </div>` : ''

  const followup = `
    <div class="fu">
      <form class="fu-f" action="/thread/${esc(threadId)}/follow-up" method="get">
        <input type="text" name="q" class="fu-i" placeholder="Ask a follow-up..." autocomplete="off">
        <input type="hidden" name="mode" value="${esc(result.mode)}">
        <button type="submit" class="fu-b">${ic.send}</button>
      </form>
    </div>`

  return `
    <div class="res">
      ${sources}
      ${answer}
      ${citations}
      ${related}
      ${followup}
      <div class="ft">Powered by AI</div>
    </div>`
}

export function renderThreadPage(thread: Thread): string {
  const msgs = thread.messages.map((msg, i) => {
    if (msg.role === 'user') {
      return `<div class="msg"><div class="msg-u">${esc(msg.content)}</div></div>`
    }

    const cites = msg.citations || []
    const n = cites.length
    const isLast = i === thread.messages.length - 1

    const sources = n > 0 ? `
      <div class="src-s">
        <div class="lbl">${ic.globe} Sources</div>
        <div class="src-r">
          ${cites.map(c => `
            <a href="${esc(c.url)}" target="_blank" rel="noopener" class="src-c">
              <img src="${esc(c.favicon)}" alt="" loading="lazy">
              <div>
                <div class="src-c-t">${esc(c.title)}</div>
                <div class="src-c-d">${esc(c.domain)}</div>
              </div>
            </a>
          `).join('')}
        </div>
      </div>` : ''

    const answer = `
      <div class="ans-s">
        <div class="lbl">${ic.spark} Answer <span class="badge">${esc(msg.model || thread.mode)}</span></div>
        <div class="ans">${renderMarkdown(msg.content, n)}</div>
      </div>`

    const related = (isLast && msg.relatedQueries && msg.relatedQueries.length > 0) ? `
      <div class="rel-s">
        <div class="lbl">Related</div>
        <div class="rel-ch">
          ${msg.relatedQueries.map(q => `
            <a href="/thread/${esc(thread.id)}/follow-up?q=${encodeURIComponent(q)}&mode=${encodeURIComponent(thread.mode)}" class="rc">${esc(q)}</a>
          `).join('')}
      </div>
    </div>` : ''

    return `<div class="msg msg-a">${sources}${answer}${related}</div>`
  }).join('')

  return `
    <div class="th">
      <div class="th-hd">
        <div class="th-t">${esc(thread.title)}</div>
        <div class="th-m">
          <span class="badge">${esc(thread.mode)}</span>
          <span>${thread.messages.length} messages</span>
          <span>${relTime(thread.createdAt)}</span>
        </div>
      </div>
      ${msgs}
      <div class="fu">
        <form class="fu-f" action="/thread/${esc(thread.id)}/follow-up" method="get">
          <input type="text" name="q" class="fu-i" placeholder="Ask a follow-up..." autocomplete="off">
          <input type="hidden" name="mode" value="${esc(thread.mode)}">
          <button type="submit" class="fu-b">${ic.send}</button>
        </form>
      </div>
      <div class="ft">Powered by AI</div>
    </div>`
}

export function renderHistoryPage(threads: ThreadSummary[]): string {
  if (threads.length === 0) {
    return `
      <div class="hist">
        <h1>History</h1>
        <div class="hist-e">
          ${ic.empty}
          <p>No search history yet</p>
          <p><a href="/">Start searching</a></p>
        </div>
      </div>`
  }

  return `
    <div class="hist">
      <h1>History</h1>
      <ul>
        ${threads.map(t => `
          <li>
            <a href="/thread/${esc(t.id)}" class="ti">
              <div class="ti-ic">${ic.chat}</div>
              <div class="ti-body">
                <div class="ti-t">${esc(t.title)}</div>
                <div class="ti-m">
                  <span class="badge">${esc(t.mode)}</span>
                  <span>${relTime(t.updatedAt)}</span>
                  <span>${t.messageCount} messages</span>
                </div>
              </div>
              <button class="ti-del" onclick="event.preventDefault();event.stopPropagation();delTh('${esc(t.id)}',this)" title="Delete">${ic.trash}</button>
            </a>
          </li>
        `).join('')}
      </ul>
    </div>
    <script>
      async function delTh(id,btn){
        if(!confirm('Delete this thread?'))return;
        const r=await fetch('/api/thread/'+id,{method:'DELETE'});
        if(r.ok)btn.closest('li').remove();
      }
    </script>`
}

export function renderError(title: string, message: string): string {
  return `
    <div class="err">
      <h1>${esc(title)}</h1>
      <p>${esc(message)}</p>
      <a href="/" class="btn">Back to home</a>
    </div>`
}
