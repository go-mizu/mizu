// ===================================================================
// Tab 5: WARC Console
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
        <button onclick="renderWARC(0, state.warcQuery||'')" class="ui-btn px-3 py-2 text-xs font-mono">Reload</button>
      </div>
      <div class="surface p-3 mb-4">
        <input id="warc-q" type="text" value="${esc(query)}" placeholder="Filter by index or filename\u2026"
          class="ui-input w-full text-sm px-3 py-2"
          onkeydown="if(event.key==='Enter')renderWARC(0,this.value)">
      </div>
      <div id="warc-content"><div class="ui-empty">loading\u2026</div></div>
    </div>`;

  try {
    const data = await apiWARCList({ offset, limit: state.warcLimit || 100, q: query || '' });
    state.warcRows = data.warcs || [];
    state.warcSummary = data.summary || null;
    state.warcOffset = data.offset || 0;
    state.warcLimit = data.limit || state.warcLimit || 100;
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

  // ── Stat cards ──
  const totalBytes = (summary.warc_bytes || 0) + (summary.markdown_bytes || 0) + (summary.pack_bytes || 0) + (summary.fts_bytes || 0);
  const remaining = total - ix;
  const statCards = total > 0 ? `
    <div class="grid grid-cols-2 sm:grid-cols-4 gap-px border border-[var(--border)] mb-4" style="background:var(--border)">
      <div class="bg-[var(--panel)] px-3 py-2.5">
        <div class="text-[10px] font-mono ui-subtle uppercase tracking-wider">Total WARCs</div>
        <div class="text-base font-semibold font-mono">${total.toLocaleString()}</div>
      </div>
      <div class="bg-[var(--panel)] px-3 py-2.5">
        <div class="text-[10px] font-mono ui-subtle uppercase tracking-wider">Fully Indexed</div>
        <div class="text-base font-semibold font-mono">${ix.toLocaleString()} <span class="text-xs ui-subtle font-normal">${total > 0 ? Math.round((ix/total)*100) + '%' : ''}</span></div>
      </div>
      <div class="bg-[var(--panel)] px-3 py-2.5">
        <div class="text-[10px] font-mono ui-subtle uppercase tracking-wider">Remaining</div>
        <div class="text-base font-semibold font-mono">${remaining.toLocaleString()}</div>
      </div>
      <div class="bg-[var(--panel)] px-3 py-2.5">
        <div class="text-[10px] font-mono ui-subtle uppercase tracking-wider">Disk Usage</div>
        <div class="text-base font-semibold font-mono">${fmtBytes(totalBytes)}</div>
      </div>
    </div>` : '';

  // ── Pipeline waterfall (like overview) ──
  const stages = [
    { label: 'Downloaded', count: dl, cls: 'ov-c1' },
    { label: 'Markdown', count: md, cls: 'ov-c2' },
    { label: 'Packed', count: pk, cls: 'ov-c3' },
    { label: 'Indexed', count: ix, cls: 'ov-c4' },
  ];
  const pipelineHTML = total > 0 ? `
    <div class="surface p-4 mb-4">
      <div class="flex items-stretch gap-0">
        ${stages.map((s, i) => {
          const p = total > 0 ? Math.round((s.count / total) * 100) : 0;
          return `
            ${i > 0 ? '<div class="ov-pipeline-arrow">\u2192</div>' : ''}
            <div class="ov-pipeline-step">
              <div class="flex items-baseline justify-between mb-1">
                <span class="text-[11px] font-mono ui-subtle">${esc(s.label)}</span>
                <span class="text-[11px] font-mono ${p === 100 ? 'status-completed' : p > 0 ? '' : 'ui-subtle'}">${s.count.toLocaleString()}</span>
              </div>
              <div class="progress-track" style="height:6px">
                <div class="${s.cls}" style="height:100%;width:${p}%;transition:width 0.4s ease"></div>
              </div>
            </div>`;
        }).join('')}
      </div>
      ${(summary.warc_bytes || summary.markdown_bytes || summary.pack_bytes || summary.fts_bytes) ? `
        <div class="flex items-center gap-4 mt-3 pt-3 border-t text-[10px] font-mono ui-subtle">
          ${summary.warc_bytes ? `<span class="flex items-center gap-1"><span class="ov-legend-dot ov-c1"></span>.warc.gz ${fmtBytes(summary.warc_bytes)}</span>` : ''}
          ${summary.markdown_bytes ? `<span class="flex items-center gap-1"><span class="ov-legend-dot ov-c2"></span>.md.warc.gz ${fmtBytes(summary.markdown_bytes)}</span>` : ''}
          ${summary.pack_bytes ? `<span class="flex items-center gap-1"><span class="ov-legend-dot ov-c3"></span>pack ${fmtBytes(summary.pack_bytes)}</span>` : ''}
          ${summary.fts_bytes ? `<span class="flex items-center gap-1"><span class="ov-legend-dot ov-c4"></span>index ${fmtBytes(summary.fts_bytes)}</span>` : ''}
        </div>` : ''}
    </div>` : '';

  // ── Table rows — simplified ──
  const rows = (data.warcs || []).map((w, i) => {
    const next = !w.has_warc ? 'download' : !w.has_markdown ? 'markdown' : !w.has_pack ? 'pack' : !w.has_fts ? 'index' : '';
    const done = (w.has_warc ? 1 : 0) + (w.has_markdown ? 1 : 0) + (w.has_pack ? 1 : 0) + (w.has_fts ? 1 : 0);
    const docsStr = (w.warc_md_docs || 0) > 0 ? (w.warc_md_docs).toLocaleString() : '\u2014';
    const sizeStr = w.total_bytes > 0 ? fmtBytes(w.total_bytes) : '\u2014';

    const nextBtnLabel = next || 'done';
    const nextBtnCls = next ? 'ui-btn-primary' : 'ui-btn';
    const nextAction = next === 'download' ? `warcAction('${esc(w.index)}','download')`
      : next === 'markdown' ? `warcAction('${esc(w.index)}','markdown')`
      : next === 'pack' ? `warcAction('${esc(w.index)}','pack',{format:'parquet'})`
      : next === 'index' ? `warcAction('${esc(w.index)}','index',{engine:currentSearchEngine(),source:'files'})`
      : '';

    return `
    <tr class="anim-fade-up" style="animation-delay:${Math.min(i, 20)*12}ms">
      <td class="px-3 py-2 font-mono text-xs"><a href="#/warc/${encodeURIComponent(w.index)}" class="ui-link hover:text-[var(--accent)]">${esc(w.index)}</a></td>
      <td class="px-3 py-2 text-xs">
        <div class="flex items-center gap-2">
          <div class="flex gap-0.5">
            ${statusChip('dl', !!w.has_warc)}
            ${statusChip('md', !!w.has_markdown)}
            ${statusChip('pk', !!w.has_pack)}
            ${statusChip('ix', !!w.has_fts)}
          </div>
          <div class="progress-track hidden sm:block" style="width:40px;height:4px">
            <div class="${done === 4 ? 'ov-c4' : 'progress-fill'}" style="height:100%;width:${done * 25}%"></div>
          </div>
        </div>
      </td>
      <td class="px-3 py-2 text-right text-xs font-mono ui-subtle">${docsStr}</td>
      <td class="px-3 py-2 text-right text-xs font-mono ui-subtle whitespace-nowrap hidden sm:table-cell">${sizeStr}</td>
      <td class="px-3 py-2 text-right text-xs font-mono whitespace-nowrap">
        ${next ? `<button onclick="${nextAction}" class="${nextBtnCls} px-2.5 py-1 text-[11px]">${esc(nextBtnLabel)}</button>` : `<span class="text-[11px] status-completed">done</span>`}
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
    ${statCards}
    ${pipelineHTML}
    ${isDashboard && incomplete.length > 0 ? `
      <div class="flex items-center gap-2 mb-3">
        <span class="text-[11px] font-mono ui-subtle">${incomplete.length} incomplete on this page</span>
        <button onclick="warcBatchNext()" class="ui-btn px-3 py-1 text-[11px] font-mono">Run next step (${incomplete.length})</button>
      </div>` : ''}
    <div class="surface overflow-x-auto">
      <table class="w-full text-sm ui-table">
        <thead>
          <tr>
            <th class="text-left px-3 py-2 text-[11px] font-mono">Index</th>
            <th class="text-left px-3 py-2 text-[11px] font-mono">Pipeline</th>
            <th class="text-right px-3 py-2 text-[11px] font-mono">Docs</th>
            <th class="text-right px-3 py-2 text-[11px] font-mono hidden sm:table-cell">Size</th>
            <th class="text-right px-3 py-2 text-[11px] font-mono">Action</th>
          </tr>
        </thead>
        <tbody>
          ${rows || `<tr><td colspan="5" class="px-3 py-4 text-xs font-mono ui-subtle">No WARC records</td></tr>`}
        </tbody>
      </table>
    </div>
    <div class="flex items-center justify-between mt-3">
      <div class="text-xs font-mono ui-subtle">${showFrom > 0 ? `${showFrom}\u2013${showTo} of ${(data.total||0).toLocaleString()}` : 'no results'}</div>
      <div class="flex items-center gap-2">
        <button ${canPrev ? '' : 'disabled'} onclick="renderWARC(${prevOffset}, state.warcQuery || '')" class="ui-btn px-3 py-1 text-xs font-mono">\u2190 prev</button>
        <span class="text-[11px] font-mono ui-subtle">${total > 0 ? `page ${Math.floor((data.offset||0)/(data.limit||100))+1}` : ''}</span>
        <button ${canNext ? '' : 'disabled'} onclick="renderWARC(${nextOffset}, state.warcQuery || '')" class="ui-btn px-3 py-1 text-xs font-mono">next \u2192</button>
      </div>
    </div>`;
}

// Batch: run next missing step on all visible incomplete WARCs
async function warcBatchNext() {
  const warcs = state.warcRows || [];
  const incomplete = warcs.filter(w => !w.has_warc || !w.has_markdown || !w.has_pack || !w.has_fts);
  if (incomplete.length === 0) return;
  if (!confirm(`Run the next pipeline step on ${incomplete.length} incomplete WARC${incomplete.length !== 1 ? 's' : ''}?`)) return;

  let started = 0;
  for (const w of incomplete) {
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
}

async function renderWARCDetail(index) {
  state.currentPage = 'warc';
  state.warcDetail = null;
  warcActionMessage = ''; // clear stale messages
  const main = $('main');
  main.innerHTML = `
    <div class="page-shell anim-fade-in">
      <a href="#/warc" class="text-xs font-mono ui-link">\u2190 WARC Console</a>
      <div id="warc-detail-content" class="mt-4"><div class="ui-empty">loading\u2026</div></div>
    </div>`;
  await ensureEnginesLoaded();
  try {
    const data = await apiWARCDetail(index);
    state.warcDetail = data;
    const w = data.warc || {};

    // Phase size breakdown
    const packTotal = Object.values(w.pack_bytes || {}).reduce((a,b)=>a+b,0);
    const ftsTotal = Object.values(w.fts_bytes || {}).reduce((a,b)=>a+b,0);
    const phases = [
      { label: 'warc', value: w.warc_bytes || 0, cls: 'ov-c1' },
      { label: 'markdown', value: w.warc_md_bytes || 0, cls: 'ov-c2' },
      { label: 'pack', value: packTotal, cls: 'ov-c3' },
      { label: 'index', value: ftsTotal, cls: 'ov-c4' },
    ];
    const diskTotal = phases.reduce((a, p) => a + p.value, 0) || 1;

    // Donut
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
            <div class="text-[10px] font-mono ui-subtle mb-1">Pack formats</div>
            ${renderBars(packEntries.map(([fmt, b]) => ({ label: fmt, value: b, text: fmtBytes(b) })))}
          </div>` : ''}
        ${ftsEntries.length > 0 ? `
          <div>
            <div class="text-[10px] font-mono ui-subtle mb-1">FTS engines</div>
            ${renderBars(ftsEntries.map(([eng, b]) => ({ label: eng, value: b, text: fmtBytes(b) })))}
          </div>` : ''}
      </div>` : '';

    const nextStep = !w.has_warc ? 'download' : !w.has_markdown ? 'markdown' : !w.has_pack ? 'pack' : !w.has_fts ? 'index' : '';
    const done = (w.has_warc ? 1 : 0) + (w.has_markdown ? 1 : 0) + (w.has_pack ? 1 : 0) + (w.has_fts ? 1 : 0);

    const enginesOpts = (state.engines||[]).map(e=>`<option value="${esc(e)}">${esc(e)}</option>`).join('') ||
      `<option value="${DEFAULT_ENGINE}">${DEFAULT_ENGINE}</option>`;

    // Active jobs
    const relatedJobs = data.jobs || [];
    const activeJobs = relatedJobs.filter(j => j.status === 'running' || j.status === 'queued');

    // Compression ratio
    const compressionRatio = (w.warc_bytes > 0 && (w.warc_md_bytes || 0) > 0) ? ((w.warc_md_bytes / w.warc_bytes) * 100).toFixed(1) + '%' : '\u2014';
    const docsCount = w.warc_md_docs || 0;

    $('warc-detail-content').innerHTML = `
      <div class="page-header mb-3">
        <h1 class="page-title">WARC ${esc(w.index || index)}</h1>
        <button onclick="renderWARCDetail('${esc(w.index || index)}')" class="ui-btn px-3 py-2 text-xs font-mono">Reload</button>
      </div>
      <div id="warc-action-msg" class="meta-line mb-3">${esc(warcActionMessage)}</div>

      <!-- Info cards -->
      <div class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-px border border-[var(--border)] mb-4" style="background:var(--border)">
        <div class="bg-[var(--panel)] px-3 py-2.5 col-span-2 sm:col-span-1">
          <div class="text-[10px] font-mono ui-subtle uppercase tracking-wider">Index</div>
          <div class="text-sm font-mono font-semibold">${esc(w.index || index)}</div>
        </div>
        <div class="bg-[var(--panel)] px-3 py-2.5">
          <div class="text-[10px] font-mono ui-subtle uppercase tracking-wider">Documents</div>
          <div class="text-sm font-mono font-semibold">${docsCount.toLocaleString()}</div>
        </div>
        <div class="bg-[var(--panel)] px-3 py-2.5">
          <div class="text-[10px] font-mono ui-subtle uppercase tracking-wider">Total on Disk</div>
          <div class="text-sm font-mono font-semibold">${fmtBytes(w.total_bytes || 0)}</div>
        </div>
        <div class="bg-[var(--panel)] px-3 py-2.5">
          <div class="text-[10px] font-mono ui-subtle uppercase tracking-wider">MD / WARC</div>
          <div class="text-sm font-mono font-semibold">${compressionRatio}</div>
        </div>
        <div class="bg-[var(--panel)] px-3 py-2.5">
          <div class="text-[10px] font-mono ui-subtle uppercase tracking-wider">Progress</div>
          <div class="text-sm font-mono font-semibold">${done}/4</div>
          <div class="progress-track mt-1" style="height:4px">
            <div class="${done === 4 ? 'ov-c4' : 'progress-fill'}" style="height:100%;width:${done * 25}%"></div>
          </div>
        </div>
      </div>

      ${w.filename ? `
      <div class="text-[11px] font-mono ui-subtle mb-4 break-all">${esc(w.filename)}${w.remote_path ? ` \u00b7 ${esc(w.remote_path)}` : ''}</div>` : ''}

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

      <!-- Phase status + disk breakdown -->
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
            ${nextStep ? `next: <span class="font-medium" style="color:var(--text)">${nextStep}</span>` : '<span class="status-completed">all phases complete</span>'}
            ${w.updated_at ? ` \u00b7 updated ${fmtRelativeTime(w.updated_at)}` : ''}
          </div>
        </div>
        <div class="surface p-3">
          <div class="text-[11px] font-mono ui-subtle mb-2">Disk Breakdown</div>
          <div class="flex items-start gap-4">
            <div class="ov-donut shrink-0" style="width:100px;height:100px;background:${donutGrad}">
              <div class="ov-donut-hole">
                <span class="text-[10px] font-mono font-medium">${fmtBytes(w.total_bytes || 0)}</span>
              </div>
            </div>
            <div class="flex-1 space-y-1.5">
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

      <!-- Actions: primary flow + advanced -->
      <div class="surface p-3 mb-4">
        <div class="text-[11px] font-mono ui-subtle mb-3">Actions</div>

        <!-- Primary next-step action -->
        ${nextStep ? `
        <div class="mb-4">
          ${nextStep === 'download' ? `<button onclick="warcAction('${esc(w.index)}','download',{},true)" class="ui-btn-primary ui-btn px-4 py-2 text-xs font-mono w-full sm:w-auto">Download WARC</button>` : ''}
          ${nextStep === 'markdown' ? `<button onclick="warcAction('${esc(w.index)}','markdown',{fast:false},true)" class="ui-btn-primary ui-btn px-4 py-2 text-xs font-mono w-full sm:w-auto">Extract Markdown</button>` : ''}
          ${nextStep === 'pack' ? `<button onclick="warcAction('${esc(w.index)}','pack',{format:'parquet'},true)" class="ui-btn-primary ui-btn px-4 py-2 text-xs font-mono w-full sm:w-auto">Pack (parquet)</button>` : ''}
          ${nextStep === 'index' ? `<button onclick="warcAction('${esc(w.index)}','index',{engine:currentSearchEngine(),source:'files'},true)" class="ui-btn-primary ui-btn px-4 py-2 text-xs font-mono w-full sm:w-auto">Build Index (${esc(currentSearchEngine())})</button>` : ''}
        </div>` : ''}

        <!-- Advanced actions (collapsed) -->
        <details class="text-xs">
          <summary class="text-[11px] font-mono ui-subtle cursor-pointer mb-3 select-none">Advanced actions</summary>
          <div class="grid md:grid-cols-2 gap-4">
            <div class="space-y-2">
              <div class="text-[10px] font-mono ui-subtle mb-1 uppercase tracking-wider">Fetch &amp; Extract</div>
              <button onclick="warcAction('${esc(w.index)}','download',{},true)" class="ui-btn w-full px-3 py-2 text-xs font-mono">Download WARC</button>
              <button onclick="warcAction('${esc(w.index)}','markdown',{fast:false},true)" class="ui-btn w-full px-3 py-2 text-xs font-mono">Extract Markdown</button>
              <button onclick="warcAction('${esc(w.index)}','markdown',{fast:true},true)" class="ui-btn w-full px-3 py-2 text-xs font-mono">Extract Markdown --fast</button>
            </div>
            <div class="space-y-2">
              <div class="text-[10px] font-mono ui-subtle mb-1 uppercase tracking-wider">Pack &amp; Index</div>
              <div class="flex items-center gap-2">
                <select id="warc-pack-format" class="ui-select flex-1 text-xs px-2 py-2">
                  <option value="parquet">parquet</option><option value="bin">bin</option><option value="duckdb">duckdb</option><option value="markdown">markdown</option>
                </select>
                <button onclick="warcAction('${esc(w.index)}','pack',{format:$('warc-pack-format').value},true)" class="ui-btn px-3 py-2 text-xs font-mono shrink-0">Pack</button>
              </div>
              <div class="flex items-center gap-2">
                <select id="warc-index-engine" class="ui-select flex-1 text-xs px-2 py-2">${enginesOpts}</select>
                <select id="warc-index-source" class="ui-select text-xs px-2 py-2">
                  <option value="files">files</option><option value="parquet">parquet</option><option value="bin">bin</option><option value="duckdb">duckdb</option><option value="markdown">markdown</option>
                </select>
                <button onclick="warcAction('${esc(w.index)}','index',{engine:$('warc-index-engine').value,source:$('warc-index-source').value},true)" class="ui-btn px-3 py-2 text-xs font-mono shrink-0">Index</button>
              </div>
              <button onclick="warcAction('${esc(w.index)}','reindex',{engine:$('warc-index-engine').value,source:$('warc-index-source').value},true)" class="ui-btn w-full px-3 py-2 text-xs font-mono">Re-index</button>
              <div class="pt-2 mt-2 border-t">
                <div class="flex items-center gap-2">
                  <select id="warc-delete-target" class="ui-select flex-1 text-xs px-2 py-2">
                    <option value="index">index</option><option value="pack">pack</option><option value="markdown">markdown</option><option value="warc">warc</option><option value="all">all</option>
                  </select>
                  <button onclick="if(confirm('Delete '+$('warc-delete-target').value+' for WARC ${esc(w.index)}?'))warcAction('${esc(w.index)}','delete',{target:$('warc-delete-target').value,format:$('warc-pack-format').value,engine:$('warc-index-engine').value},true)" class="ui-btn ui-btn-danger px-3 py-2 text-xs font-mono shrink-0">Delete</button>
                </div>
              </div>
            </div>
          </div>
        </details>
      </div>

      <!-- Related Jobs -->
      <div class="surface p-3">
        <div class="flex items-center justify-between mb-2">
          <div class="text-[11px] font-mono ui-subtle">Related Jobs (${relatedJobs.length})</div>
          ${relatedJobs.length > 5 ? `<span class="text-[10px] font-mono ui-subtle">last ${Math.min(relatedJobs.length, 20)}</span>` : ''}
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
  if (msg) msg.textContent = `running ${action} on ${index}\u2026`;
  warcActionMessage = '';
  try {
    const res = await apiWARCAction(index, { action, ...extra });
    if (res && res.job && res.job.id) {
      if (!state.jobs) state.jobs = [];
      state.jobs.unshift(res.job);
      wsClient.subscribe(res.job.id, (m) => onJobUpdate(m));
      warcActionMessage = `job ${res.job.id} started: ${action}`;
    } else {
      warcActionMessage = `${action} completed`;
    }
    if (msg) msg.textContent = warcActionMessage;
    if (refreshDetail) {
      setTimeout(() => renderWARCDetail(index), 400);
    } else if (state.currentPage === 'warc') {
      setTimeout(() => renderWARC(state.warcOffset || 0, state.warcQuery || ''), 400);
    }
  } catch (e) {
    warcActionMessage = `failed: ${e.message}`;
    if (msg) msg.textContent = warcActionMessage;
  }
}
