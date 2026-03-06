// ===================================================================
// Document Viewer
// ===================================================================
async function renderDoc(shard, docid) {
  state.currentPage = 'doc';
  const main = $('main');
  main.innerHTML = `
    <div class="py-8 max-w-3xl mx-auto anim-fade-in">
      <a href="#" onclick="history.back();return false" class="text-xs ui-link transition-colors">&larr; Back</a>
      <div class="meta-line mt-3 mb-8">${esc(shard)} / ${esc(decodeURIComponent(docid))}</div>
      <div class="space-y-3">
        <div class="h-5 w-2/3 ui-skeleton"></div>
        <div class="h-4 w-full ui-skeleton"></div>
        <div class="h-4 w-full ui-skeleton"></div>
        <div class="h-4 w-4/5 ui-skeleton"></div>
      </div>
    </div>`;

  try {
    const data = await apiDoc(shard, docid);
    state.doc = data;
    const hasUrl = !!(data.url);
    const hasTitle = !!(data.title);
    main.innerHTML = `
      <div class="py-8 max-w-3xl mx-auto anim-fade-in">
        <a href="#" onclick="history.back();return false" class="text-xs ui-link transition-colors">&larr; Back</a>

        ${hasTitle || hasUrl ? `
        <div class="mt-4 mb-6 pl-4 border-l-2 border-[var(--accent)]">
          ${hasTitle ? `<h2 class="text-base font-semibold mb-1">${esc(data.title)}</h2>` : ''}
          ${hasUrl ? `<a href="${esc(data.url)}" target="_blank" rel="noopener noreferrer"
              class="text-xs font-mono ui-link break-all">${esc(data.url)}</a>` : ''}
          <div class="meta-line text-[11px] mt-2 flex flex-wrap gap-3">
            ${data.crawl_date ? `<span>Crawled ${fmtDate(data.crawl_date)}</span>` : ''}
            ${data.size_bytes ? `<span>${fmtBytes(data.size_bytes)}</span>` : ''}
            ${data.word_count ? `<span>${data.word_count.toLocaleString()} words</span>` : ''}
            ${data.warc_record_id ? `<span class="font-mono text-[10px] opacity-60">${esc(data.warc_record_id)}</span>` : ''}
          </div>
        </div>` : `<div class="meta-line mt-3 mb-6">${esc(data.shard || shard)} / ${esc(data.doc_id || docid)}</div>`}

        <div class="flex items-center gap-6 mb-6 border-b pb-3">
          <button id="btn-rendered" onclick="showRendered()" class="text-xs pb-1 tab-active transition-colors">Rendered</button>
          <button id="btn-source" onclick="showSource()" class="text-xs pb-1 tab-inactive transition-colors">Source</button>
          ${!(hasTitle || hasUrl) ? `<span class="meta-line ml-auto">
            ${data.word_count ? data.word_count.toLocaleString() + ' words' : ''}${data.word_count && data.size_bytes ? ' &middot; ' : ''}${data.size_bytes ? fmtBytes(data.size_bytes) : ''}
          </span>` : ''}
        </div>

        <div id="doc-rendered" class="prose dark:prose-invert max-w-none prose-zinc prose-sm prose-headings:font-semibold prose-headings:tracking-tight prose-a:text-blue-600 dark:prose-a:text-blue-400 prose-code:text-sm prose-code:font-mono prose-img:max-w-full">
          ${data.html || '<p class="ui-subtle">No content</p>'}
        </div>
        <div id="doc-source" class="hidden">
          <pre class="p-4 surface-soft text-sm font-mono whitespace-pre-wrap break-words overflow-auto max-h-[80vh]">${esc(data.markdown || '')}</pre>
        </div>
      </div>`;
  } catch(e) {
    main.innerHTML = `
      <div class="py-16 text-center anim-fade-in">
        <p class="text-red-400 text-sm mb-1">Failed to load document</p>
        <p class="ui-subtle text-xs mb-4">${esc(e.message)}</p>
        <a href="#" onclick="history.back();return false" class="text-xs ui-link transition-colors">&larr; Back</a>
      </div>`;
  }
}

function showRendered() {
  $('doc-rendered').classList.remove('hidden');
  $('doc-source').classList.add('hidden');
  $('btn-rendered').className = 'text-xs pb-1 tab-active transition-colors';
  $('btn-source').className = 'text-xs pb-1 tab-inactive transition-colors';
}

function showSource() {
  $('doc-rendered').classList.add('hidden');
  $('doc-source').classList.remove('hidden');
  $('btn-source').className = 'text-xs pb-1 tab-active transition-colors';
  $('btn-rendered').className = 'text-xs pb-1 tab-inactive transition-colors';
}

function formatCrawlDate(d) {
  if (!d) return '\u2014';
  // d could be ISO string or time.Time JSON
  try {
    const dt = new Date(d);
    if (isNaN(dt.getTime())) return esc(String(d));
    return dt.toLocaleDateString('en-US', { year: 'numeric', month: 'short', day: 'numeric' });
  } catch {
    return esc(String(d));
  }
}
