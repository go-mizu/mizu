// ===================================================================
// Tab 5: WARC
// ===================================================================
async function renderWARC(offset = state.warcOffset || 0, query = state.warcQuery || '') {
  state.currentPage = 'warc';
  state.warcOffset = offset;
  state.warcQuery = query;
  const main = $('main');
  main.innerHTML = `
    <div class="page-shell anim-fade-in">
      <div class="page-header mb-4">
        <h1 class="page-title">WARC Console</h1>
        <div class="flex items-center gap-2">
          <button onclick="refreshDashboardMeta(true)" class="ui-btn px-3 py-2 text-xs font-mono">Refresh Metadata</button>
          <button onclick="renderWARC(0, state.warcQuery||'')" class="ui-btn px-3 py-2 text-xs font-mono">Reload</button>
        </div>
      </div>
      <div id="warc-meta" class="meta-line mb-5">loading metadata...</div>
      <div class="surface p-3 mb-4">
        <input id="warc-q" type="text" value="${esc(query)}" placeholder="Filter by index, filename, or remote path"
          class="ui-input w-full text-sm px-3 py-2"
          onkeydown="if(event.key==='Enter')renderWARC(0,this.value)">
      </div>
      <div id="warc-content"><div class="ui-empty">loading...</div></div>
    </div>`;

  try {
    const data = await apiWARCList({ offset, limit: state.warcLimit || 100, q: query || '' });
    state.warcRows = data.warcs || [];
    state.warcSummary = data.summary || null;
    state.warcOffset = data.offset || 0;
    state.warcLimit = data.limit || state.warcLimit || 100;

    const meta = $('warc-meta');
    if (meta) {
      const parts = [];
      if (data.meta_backend) parts.push(`meta:${data.meta_backend}`);
      if (data.meta_generated_at) parts.push(`updated:${fmtRelativeTime(data.meta_generated_at)}`);
      if (data.meta_stale) parts.push('stale');
      if (data.meta_refreshing) parts.push('refreshing\u2026');
      if (data.meta_last_error) parts.push(`error:${data.meta_last_error}`);
      meta.textContent = parts.join(' · ');
    }
    renderWARCContent(data);
  } catch (e) {
    $('warc-content').innerHTML = `<div class="text-xs text-red-400">${esc(e.message)}</div>`;
  }
}

function renderWARCContent(data) {
  const el = $('warc-content');
  if (!el) return;
  const summary = data.summary || {};
  const total = summary.total || 0;
  const dl = summary.downloaded || 0;
  const md = summary.markdown_ready || 0;
  const pk = summary.packed || 0;
  const ix = summary.indexed || 0;

  // ── Pipeline Funnel — stacked bar + step counts ──
  const funnelSteps = [
    { label: 'Downloaded', count: dl, cls: 'ov-c1' },
    { label: 'Markdown', count: md, cls: 'ov-c2' },
    { label: 'Packed', count: pk, cls: 'ov-c3' },
    { label: 'Indexed', count: ix, cls: 'ov-c4' },
  ];
  const hasSizes = summary.warc_bytes || summary.markdown_bytes || summary.pack_bytes || summary.fts_bytes;
  const totalBytes = (summary.warc_bytes || 0) + (summary.markdown_bytes || 0) + (summary.pack_bytes || 0) + (summary.fts_bytes || 0);
  const funnelHTML = total > 0 ? `
    <div class="surface p-4 mb-4">
      <div class="flex items-center justify-between mb-3">
        <div class="text-[11px] font-mono ui-subtle">${total.toLocaleString()} WARC files${hasSizes ? ` \u00b7 ${fmtBytes(totalBytes)} total` : ''}</div>
        <div class="text-[11px] font-mono ui-subtle">${ix === total ? 'all indexed' : `${(total - ix).toLocaleString()} remaining`}</div>
      </div>
      <div class="ov-stacked mb-3">
        ${funnelSteps.map(s => {
          const w = total > 0 ? Math.max(s.count > 0 ? 1 : 0, (s.count / total) * 100) : 0;
          return `<div class="ov-stacked-seg ${s.cls}" style="width:${w}%" title="${s.label}: ${s.count}/${total}"></div>`;
        }).join('')}
      </div>
      <div class="grid grid-cols-4 gap-3">
        ${funnelSteps.map(s => {
          const pct = total > 0 ? Math.round((s.count / total) * 100) : 0;
          return `
            <div class="flex items-center gap-2">
              <div class="ov-legend-dot ${s.cls}"></div>
              <div>
                <div class="text-[11px] font-mono">${s.count.toLocaleString()} <span class="ui-subtle">${esc(s.label.toLowerCase())}</span></div>
                <div class="text-[10px] font-mono ui-subtle">${pct}%</div>
              </div>
            </div>`;
        }).join('')}
      </div>
      ${hasSizes ? `
        <div class="flex items-center gap-4 mt-3 pt-3 border-t text-[11px] font-mono ui-subtle">
          ${summary.warc_bytes ? `<span>.warc.gz ${fmtBytes(summary.warc_bytes)}</span>` : ''}
          ${summary.markdown_bytes ? `<span>.md.warc.gz ${fmtBytes(summary.markdown_bytes)}</span>` : ''}
          ${summary.pack_bytes ? `<span>pk ${fmtBytes(summary.pack_bytes)}</span>` : ''}
          ${summary.fts_bytes ? `<span>ix ${fmtBytes(summary.fts_bytes)}</span>` : ''}
        </div>` : ''}
    </div>` : '';

  // ── Table rows with smart next-step highlighting + mini progress ──
  const rows = (data.warcs || []).map((w, i) => {
    const next = !w.has_warc ? 'dl' : !w.has_markdown ? 'md' : !w.has_pack ? 'pk' : !w.has_fts ? 'ix' : '';
    const done = (w.has_warc ? 1 : 0) + (w.has_markdown ? 1 : 0) + (w.has_pack ? 1 : 0) + (w.has_fts ? 1 : 0);
    const docsStr = (w.warc_md_docs || 0) > 0 ? (w.warc_md_docs).toLocaleString() : '\u2014';
    const warcSz = w.warc_bytes > 0 ? fmtBytes(w.warc_bytes) : '\u2014';
    const mdSz = w.warc_md_bytes > 0 ? fmtBytes(w.warc_md_bytes) : '\u2014';
    return `
    <tr class="anim-fade-up" style="animation-delay:${Math.min(i, 20)*15}ms">
      <td class="px-3 py-2 font-mono text-xs"><a href="#/warc/${encodeURIComponent(w.index)}" class="hover:underline">${esc(w.index)}</a></td>
      <td class="px-3 py-2 text-xs ui-subtle truncate max-w-[200px]" title="${esc(w.filename || '')}">${esc(w.filename || '\u2014')}</td>
      <td class="px-3 py-2 text-xs">
        <div class="flex items-center gap-2">
          <div class="flex gap-1">
            ${statusChip('dl', !!w.has_warc)}
            ${statusChip('md', !!w.has_markdown)}
            ${statusChip('pk', !!w.has_pack)}
            ${statusChip('ix', !!w.has_fts)}
          </div>
          <div class="progress-track" style="width:40px;height:4px">
            <div class="${done === 4 ? 'ov-c4' : 'progress-fill'}" style="height:100%;width:${done * 25}%"></div>
          </div>
        </div>
      </td>
      <td class="px-3 py-2 text-right text-xs font-mono ui-subtle">${docsStr}</td>
      <td class="px-3 py-2 text-right text-xs font-mono ui-subtle whitespace-nowrap">${warcSz}</td>
      <td class="px-3 py-2 text-right text-xs font-mono ui-subtle whitespace-nowrap">${mdSz}</td>
      <td class="px-3 py-2 text-right text-xs font-mono whitespace-nowrap">
        <button onclick="warcAction('${esc(w.index)}','download')" class="ui-btn px-2 py-1 text-[11px] ${next==='dl'?'ui-btn-primary':''}" title="Download WARC">dl</button>
        <button onclick="warcAction('${esc(w.index)}','markdown')" class="ui-btn px-2 py-1 text-[11px] ml-1 ${next==='md'?'ui-btn-primary':''}" title="Extract markdown">md</button>
        <button onclick="warcAction('${esc(w.index)}','pack',{format:'parquet'})" class="ui-btn px-2 py-1 text-[11px] ml-1 ${next==='pk'?'ui-btn-primary':''}" title="Pack (parquet)">pk</button>
        <button onclick="warcAction('${esc(w.index)}','index',{engine:currentSearchEngine(),source:'files'})" class="ui-btn px-2 py-1 text-[11px] ml-1 ${next==='ix'?'ui-btn-primary':''}" title="Build FTS index">ix</button>
      </td>
    </tr>`;
  }).join('');

  const nextOffset = (data.offset || 0) + (data.limit || 0);
  const prevOffset = Math.max(0, (data.offset || 0) - (data.limit || 0));
  const canPrev = (data.offset || 0) > 0;
  const canNext = nextOffset < (data.total || 0);
  const showFrom = total > 0 ? (data.offset || 0) + 1 : 0;
  const showTo = Math.min((data.offset || 0) + (data.limit || 0), data.total || 0);

  // Incomplete count for batch actions
  const incomplete = (data.warcs || []).filter(w => !w.has_warc || !w.has_markdown || !w.has_pack || !w.has_fts);

  el.innerHTML = `
    ${funnelHTML}
    ${isDashboard && incomplete.length > 0 ? `
      <div class="flex items-center gap-2 mb-3">
        <span class="text-[11px] font-mono ui-subtle">${incomplete.length} incomplete on this page</span>
        <button onclick="warcBatchNext()" class="ui-btn px-3 py-1 text-[11px] font-mono" title="Run the next missing step on all visible incomplete WARCs">Run next step (all)</button>
      </div>` : ''}
    <div class="surface overflow-x-auto">
      <table class="w-full text-sm ui-table">
        <thead>
          <tr>
            <th class="text-left px-3 py-2 text-[11px] font-mono">WARC</th>
            <th class="text-left px-3 py-2 text-[11px] font-mono">Filename</th>
            <th class="text-left px-3 py-2 text-[11px] font-mono">Phases</th>
            <th class="text-right px-3 py-2 text-[11px] font-mono">Docs</th>
            <th class="text-right px-3 py-2 text-[11px] font-mono">.warc.gz</th>
            <th class="text-right px-3 py-2 text-[11px] font-mono">.md.warc.gz</th>
            <th class="text-right px-3 py-2 text-[11px] font-mono">Actions</th>
          </tr>
        </thead>
        <tbody>
          ${rows || `<tr><td colspan="7" class="px-3 py-4 text-xs font-mono ui-subtle">No WARC records</td></tr>`}
        </tbody>
      </table>
    </div>
    <div class="flex items-center justify-between mt-3">
      <div class="text-xs font-mono ui-subtle">${showFrom > 0 ? `showing ${showFrom}\u2013${showTo} of ${(data.total||0).toLocaleString()}` : 'no results'}</div>
      <div class="flex items-center gap-2">
        <button ${canPrev ? '' : 'disabled'} onclick="renderWARC(${prevOffset}, state.warcQuery || '')" class="ui-btn px-3 py-1 text-xs font-mono ${canPrev ? '' : 'opacity-40 cursor-not-allowed'}">prev</button>
        <span class="text-[11px] font-mono ui-subtle">${total > 0 ? `page ${Math.floor((data.offset||0)/(data.limit||100))+1}` : ''}</span>
        <button ${canNext ? '' : 'disabled'} onclick="renderWARC(${nextOffset}, state.warcQuery || '')" class="ui-btn px-3 py-1 text-xs font-mono ${canNext ? '' : 'opacity-40 cursor-not-allowed'}">next</button>
      </div>
    </div>`;
}

// Batch: run next missing step on all visible incomplete WARCs
async function warcBatchNext() {
  const warcs = state.warcRows || [];
  let started = 0;
  for (const w of warcs) {
    if (!w.has_warc) {
      warcAction(w.index, 'download'); started++;
    } else if (!w.has_markdown) {
      warcAction(w.index, 'markdown'); started++;
    } else if (!w.has_pack) {
      warcAction(w.index, 'pack', { format: 'parquet' }); started++;
    } else if (!w.has_fts) {
      warcAction(w.index, 'index', { engine: currentSearchEngine(), source: 'files' }); started++;
    }
  }
  const msg = $('warc-meta');
  if (msg) msg.textContent = `started ${started} job${started !== 1 ? 's' : ''} (next step for each incomplete WARC)`;
}

async function renderWARCDetail(index) {
  state.currentPage = 'warc';
  state.warcDetail = null;
  const main = $('main');
  main.innerHTML = `
    <div class="page-shell anim-fade-in">
      <a href="#/warc" class="text-xs font-mono ui-link">&larr; back to WARC list</a>
      <div id="warc-detail-content" class="mt-4"><div class="ui-empty">loading...</div></div>
    </div>`;
  await ensureEnginesLoaded();
  try {
    const data = await apiWARCDetail(index);
    state.warcDetail = data;
    const w = data.warc || {};
    const actionMsg = `<div id="warc-action-msg" class="meta-line mb-3">${esc(warcActionMessage)}</div>`;

    // Phase size breakdown
    const packTotal = Object.values(w.pack_bytes || {}).reduce((a,b)=>a+b,0);
    const ftsTotal = Object.values(w.fts_bytes || {}).reduce((a,b)=>a+b,0);
    const phases = [
      { label: 'warc', value: w.warc_bytes || 0, cls: 'ov-c1' },
      { label: 'markdown', value: w.markdown_bytes || 0, cls: 'ov-c2' },
      { label: 'pack', value: packTotal, cls: 'ov-c3' },
      { label: 'index', value: ftsTotal, cls: 'ov-c4' },
    ];
    const diskTotal = phases.reduce((a, p) => a + p.value, 0) || 1;

    // Mini donut for phase proportions
    let donutAngle = 0;
    const donutStops = [];
    const clsColorMap = { 'ov-c1': 'var(--accent)', 'ov-c2': '#6366f1', 'ov-c3': '#f59e0b', 'ov-c4': '#10b981' };
    for (const p of phases) {
      if (p.value <= 0) continue;
      const deg = (p.value / diskTotal) * 360;
      donutStops.push(`${clsColorMap[p.cls] || 'var(--border)'} ${donutAngle}deg ${donutAngle + deg}deg`);
      donutAngle += deg;
    }
    const donutGrad = donutStops.length > 0 ? `conic-gradient(${donutStops.join(', ')})` : 'var(--border)';

    // Per-format/engine breakdown
    const packEntries = Object.entries(w.pack_bytes || {});
    const ftsEntries = Object.entries(w.fts_bytes || {});
    const breakdownHTML = (packEntries.length > 0 || ftsEntries.length > 0) ? `
      <div class="mt-3 pt-3 border-t space-y-3">
        ${packEntries.length > 0 ? `
          <div>
            <div class="text-[10px] font-mono ui-subtle mb-1">Pack formats (${packEntries.length})</div>
            ${renderBars(packEntries.map(([fmt, b]) => ({ label: fmt, value: b, text: fmtBytes(b) })))}
          </div>` : ''}
        ${ftsEntries.length > 0 ? `
          <div>
            <div class="text-[10px] font-mono ui-subtle mb-1">FTS engines (${ftsEntries.length})</div>
            ${renderBars(ftsEntries.map(([eng, b]) => ({ label: eng, value: b, text: fmtBytes(b) })))}
          </div>` : ''}
      </div>` : '';

    // Next action suggestion
    const nextStep = !w.has_warc ? 'download' : !w.has_markdown ? 'markdown' : !w.has_pack ? 'pack' : !w.has_fts ? 'index' : '';
    const done = (w.has_warc ? 1 : 0) + (w.has_markdown ? 1 : 0) + (w.has_pack ? 1 : 0) + (w.has_fts ? 1 : 0);

    const enginesOpts = (state.engines||[]).map(e=>`<option value="${esc(e)}">${esc(e)}</option>`).join('') ||
      `<option value="${DEFAULT_ENGINE}">${DEFAULT_ENGINE}</option>`;

    // Active jobs for this WARC
    const relatedJobs = data.jobs || [];
    const activeJobs = relatedJobs.filter(j => j.status === 'running' || j.status === 'queued');

    $('warc-detail-content').innerHTML = `
      <div class="page-header mb-3">
        <h1 class="page-title">WARC ${esc(w.index || index)}</h1>
        <div class="flex items-center gap-2">
          <button onclick="refreshDashboardMeta(true)" class="ui-btn px-3 py-2 text-xs font-mono">Refresh</button>
          <button onclick="renderWARCDetail('${esc(w.index || index)}')" class="ui-btn px-3 py-2 text-xs font-mono">Reload</button>
        </div>
      </div>
      ${actionMsg}

      <!-- Info cards: 5 columns -->
      <div class="grid grid-cols-2 md:grid-cols-5 gap-3 mb-4">
        <div class="surface p-3 col-span-2">
          <div class="text-[11px] font-mono ui-subtle mb-1">Filename</div>
          <div class="text-xs break-all font-mono">${esc(w.filename || '\u2014')}</div>
          ${w.remote_path ? `<div class="text-[10px] break-all font-mono ui-subtle mt-1">${esc(w.remote_path)}</div>` : ''}
        </div>
        <div class="surface p-3">
          <div class="text-[11px] font-mono ui-subtle mb-1">Total on Disk</div>
          <div class="text-sm font-mono">${fmtBytes(w.total_bytes || 0)}</div>
        </div>
        <div class="surface p-3">
          <div class="text-[11px] font-mono ui-subtle mb-1">Documents</div>
          <div class="text-sm font-mono">${(w.markdown_docs || 0).toLocaleString()}</div>
        </div>
        <div class="surface p-3">
          <div class="text-[11px] font-mono ui-subtle mb-1">Progress</div>
          <div class="text-sm font-mono">${done}/4</div>
          <div class="progress-track mt-1" style="height:4px">
            <div class="${done === 4 ? 'ov-c4' : 'progress-fill'}" style="height:100%;width:${done * 25}%"></div>
          </div>
        </div>
      </div>

      ${activeJobs.length > 0 ? `
        <div class="surface p-3 mb-4 border-l-2" style="border-left-color:#2563eb">
          <div class="text-[11px] font-mono mb-2" style="color:#2563eb">Active Jobs (${activeJobs.length})</div>
          ${activeJobs.map(j => {
            const pct = Math.round((j.progress || 0) * 100);
            return `
              <div class="mb-2 last:mb-0">
                <div class="flex items-center justify-between mb-1">
                  <span class="text-[11px] font-mono ui-subtle">${esc(j.id)} \u00b7 ${esc(j.type)} \u00b7 <span class="status-running">${esc(j.status)}</span></span>
                  <span class="text-[11px] font-mono">${pct}%</span>
                </div>
                <div class="progress-track" style="height:4px">
                  <div class="progress-fill" style="width:${pct}%"></div>
                </div>
                ${j.message ? `<div class="text-[10px] font-mono ui-subtle mt-1 truncate">${esc(j.message)}</div>` : ''}
              </div>`;
          }).join('')}
        </div>` : ''}

      <!-- Phase status + sizes with donut -->
      <div class="grid md:grid-cols-2 gap-4 mb-4">
        <div class="surface p-3">
          <div class="text-[11px] font-mono ui-subtle mb-2">Phase Status</div>
          <div class="flex flex-wrap gap-2 mb-3">
            ${statusChip('download', !!w.has_warc)}
            ${statusChip('markdown', !!w.has_markdown)}
            ${statusChip('pack', !!w.has_pack)}
            ${statusChip('index', !!w.has_fts)}
          </div>
          <div class="text-[11px] font-mono ui-subtle">
            ${nextStep ? `next: <span class="font-medium" style="color:var(--text)">${nextStep}</span> &middot; ` : '<span class="status-completed">all phases complete</span> &middot; '}
            updated: ${w.updated_at ? fmtRelativeTime(w.updated_at) : '\u2014'}
          </div>
        </div>
        <div class="surface p-3">
          <div class="text-[11px] font-mono ui-subtle mb-2">Phase Size on Disk</div>
          <div class="flex items-start gap-4">
            <div class="ov-donut shrink-0" style="width:80px;height:80px;background:${donutGrad}">
              <div class="ov-donut-hole">
                <span class="text-[10px] font-mono font-medium">${fmtBytes(w.total_bytes || 0)}</span>
              </div>
            </div>
            <div class="flex-1 space-y-1">
              ${phases.filter(p => p.value > 0).map(p => `
                <div class="flex items-center gap-2">
                  <div class="ov-legend-dot ${p.cls}"></div>
                  <span class="text-[11px] font-mono ui-subtle flex-1">${esc(p.label)}</span>
                  <span class="text-[11px] font-mono">${fmtBytes(p.value)}</span>
                  <span class="text-[10px] font-mono ui-subtle w-8 text-right">${Math.round((p.value / diskTotal) * 100)}%</span>
                </div>`).join('')}
              ${phases.every(p => p.value <= 0) ? '<div class="text-[11px] font-mono ui-subtle">no data on disk</div>' : ''}
            </div>
          </div>
          ${breakdownHTML}
        </div>
      </div>

      <!-- Actions -->
      <div class="surface p-3 mb-4">
        <div class="text-[11px] font-mono ui-subtle mb-3">Actions</div>
        <div class="grid md:grid-cols-2 gap-4">
          <!-- Left: download + markdown -->
          <div class="space-y-2">
            <div class="text-[10px] font-mono ui-subtle mb-1 uppercase tracking-wider">Fetch &amp; Extract</div>
            <button onclick="warcAction('${esc(w.index)}','download',{},true)" class="ui-btn w-full px-3 py-2 text-xs font-mono ${nextStep==='download'?'ui-btn-primary':''}">Download WARC</button>
            <button onclick="warcAction('${esc(w.index)}','markdown',{fast:false},true)" class="ui-btn w-full px-3 py-2 text-xs font-mono ${nextStep==='markdown'?'ui-btn-primary':''}">Extract Markdown</button>
            <button onclick="warcAction('${esc(w.index)}','markdown',{fast:true},true)" class="ui-btn w-full px-3 py-2 text-xs font-mono">Extract Markdown --fast</button>
          </div>
          <!-- Right: pack + index + delete -->
          <div class="space-y-2">
            <div class="text-[10px] font-mono ui-subtle mb-1 uppercase tracking-wider">Pack &amp; Index</div>
            <div class="flex items-center gap-2">
              <label class="text-[11px] font-mono ui-subtle w-14 shrink-0">format</label>
              <select id="warc-pack-format" class="ui-select flex-1 text-xs px-2 py-2">
                <option value="parquet">parquet</option><option value="bin">bin</option><option value="duckdb">duckdb</option><option value="markdown">markdown</option>
              </select>
              <button onclick="warcAction('${esc(w.index)}','pack',{format:$('warc-pack-format').value},true)" class="ui-btn px-3 py-2 text-xs font-mono shrink-0 ${nextStep==='pack'?'ui-btn-primary':''}">Pack</button>
            </div>
            <div class="flex items-center gap-2">
              <label class="text-[11px] font-mono ui-subtle w-14 shrink-0">engine</label>
              <select id="warc-index-engine" class="ui-select flex-1 text-xs px-2 py-2">${enginesOpts}</select>
              <select id="warc-index-source" class="ui-select text-xs px-2 py-2">
                <option value="files">files</option><option value="parquet">parquet</option><option value="bin">bin</option><option value="duckdb">duckdb</option><option value="markdown">markdown</option>
              </select>
              <button onclick="warcAction('${esc(w.index)}','index',{engine:$('warc-index-engine').value,source:$('warc-index-source').value},true)" class="ui-btn px-3 py-2 text-xs font-mono shrink-0 ${nextStep==='index'?'ui-btn-primary':''}">Index</button>
            </div>
            <div class="flex items-center gap-2">
              <label class="text-[11px] font-mono ui-subtle w-14 shrink-0">re-index</label>
              <button onclick="warcAction('${esc(w.index)}','reindex',{engine:$('warc-index-engine').value,source:$('warc-index-source').value},true)" class="ui-btn flex-1 px-3 py-2 text-xs font-mono">Re-index</button>
            </div>
            <div class="text-[10px] font-mono ui-subtle mt-2 mb-1 uppercase tracking-wider pt-2 border-t" style="color:var(--danger)">Danger Zone</div>
            <div class="flex items-center gap-2">
              <label class="text-[11px] font-mono ui-subtle w-14 shrink-0">target</label>
              <select id="warc-delete-target" class="ui-select flex-1 text-xs px-2 py-2">
                <option value="index">index</option><option value="pack">pack</option><option value="markdown">markdown</option><option value="warc">warc</option><option value="all">all</option>
              </select>
              <button onclick="if(confirm('Delete '+$('warc-delete-target').value+' for WARC ${esc(w.index)}?'))warcAction('${esc(w.index)}','delete',{target:$('warc-delete-target').value,format:$('warc-pack-format').value,engine:$('warc-index-engine').value},true)" class="ui-btn ui-btn-danger px-3 py-2 text-xs font-mono shrink-0">Delete</button>
            </div>
          </div>
        </div>
      </div>

      <!-- Related Jobs -->
      <div class="surface p-3">
        <div class="flex items-center justify-between mb-2">
          <div class="text-[11px] font-mono ui-subtle">Related Jobs (${relatedJobs.length})</div>
          ${relatedJobs.length > 5 ? `<span class="text-[10px] font-mono ui-subtle">showing last ${Math.min(relatedJobs.length, 20)}</span>` : ''}
        </div>
        ${renderJobHistory(relatedJobs.slice(0, 20))}
      </div>`;
  } catch (e) {
    $('warc-detail-content').innerHTML = `<div class="text-xs text-red-400">${esc(e.message)}</div>`;
  }
}

let warcActionMessage = '';

async function warcAction(index, action, extra = {}, refreshDetail = false) {
  const msg = $('warc-action-msg');
  if (msg) msg.textContent = `running ${action} on ${index}...`;
  warcActionMessage = '';
  try {
    const res = await apiWARCAction(index, { action, ...extra });
    if (res && res.job && res.job.id) {
      if (!state.jobs) state.jobs = [];
      state.jobs.unshift(res.job);
      wsClient.subscribe(res.job.id, (m) => onJobUpdate(m));
      warcActionMessage = `job ${res.job.id} started: ${action}`;
    } else {
      warcActionMessage = `action ${action} completed`;
    }
    if (msg) msg.textContent = warcActionMessage;
    if (refreshDetail) {
      // Delay re-render so user sees the feedback message
      setTimeout(() => renderWARCDetail(index), 400);
    } else if (state.currentPage === 'warc') {
      setTimeout(() => renderWARC(state.warcOffset || 0, state.warcQuery || ''), 400);
    }
  } catch (e) {
    warcActionMessage = `action failed: ${e.message}`;
    if (msg) msg.textContent = warcActionMessage;
  }
}
