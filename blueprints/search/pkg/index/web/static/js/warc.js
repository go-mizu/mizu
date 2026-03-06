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
      <div class="flex flex-col sm:flex-row items-stretch sm:items-center gap-2 mb-4">
        <input id="warc-q" type="text" value="${esc(query)}" placeholder="Filter by index or filename\u2026"
          class="ui-input flex-1 text-sm px-3 py-2"
          onkeydown="if(event.key==='Enter')renderWARC(0,this.value)">
        <div id="warc-filter-chips" class="flex items-center gap-1.5"></div>
      </div>
      <div id="warc-content"><div class="ui-empty">loading\u2026</div></div>
    </div>`;

  try {
    const data = await apiWARCList({ offset, limit: state.warcLimit || 100, q: query || '' });
    state.warcRows = data.warcs || [];
    state.warcSummary = data.summary || null;
    state.warcSystem = data.system || null;
    state.warcOffset = data.offset || 0;
    state.warcLimit = data.limit || state.warcLimit || 100;
    if (!state.warcFilter) state.warcFilter = 'all';
    renderWARCFilterChips();
    renderWARCContent(data);
  } catch (e) {
    $('warc-content').innerHTML = `<div class="text-xs text-red-400">${esc(e.message)}</div>`;
  }
}

function renderWARCFilterChips() {
  const el = $('warc-filter-chips');
  if (!el) return;
  const filter = state.warcFilter || 'all';
  const filters = [
    { key: 'all', label: 'All' },
    { key: 'incomplete', label: 'Incomplete' },
    { key: 'complete', label: 'Complete' },
  ];
  el.innerHTML = filters.map(f => {
    const active = f.key === filter;
    return `<button onclick="state.warcFilter='${f.key}';renderWARCContent({warcs:state.warcRows,summary:state.warcSummary,offset:state.warcOffset,limit:state.warcLimit,total:(state.warcRows||[]).length})"
      class="ui-btn px-2 py-1 text-[10px] font-mono ${active ? 'ui-btn-primary' : ''}">${f.label}</button>`;
  }).join('');
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

  // ── Pipeline waterfall ──
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

  // ── Apply client-side filter ──
  const filter = state.warcFilter || 'all';
  let visibleRows = data.warcs || [];
  if (filter === 'incomplete') {
    visibleRows = visibleRows.filter(w => !w.has_warc || !w.has_markdown || !w.has_pack || !w.has_fts);
  } else if (filter === 'complete') {
    visibleRows = visibleRows.filter(w => w.has_warc && w.has_markdown && w.has_pack && w.has_fts);
  }

  // ── Table rows ──
  const rows = visibleRows.map((w, i) => {
    const next = !w.has_warc ? 'download' : !w.has_markdown ? 'markdown' : !w.has_pack ? 'pack' : !w.has_fts ? 'index' : '';
    const done = (w.has_warc ? 1 : 0) + (w.has_markdown ? 1 : 0) + (w.has_pack ? 1 : 0) + (w.has_fts ? 1 : 0);
    const docsStr = (w.warc_md_docs || 0) > 0 ? (w.warc_md_docs).toLocaleString() : '\u2014';
    const sizeStr = w.total_bytes > 0 ? fmtBytes(w.total_bytes) : '\u2014';

    // Stacked progress bar segments
    const stackedBar = `
      <div class="ov-stacked" style="width:48px;height:6px">
        <div class="ov-stacked-seg ${w.has_warc ? 'ov-c1' : ''}" style="width:25%;${!w.has_warc ? 'background:transparent' : ''}"></div>
        <div class="ov-stacked-seg ${w.has_markdown ? 'ov-c2' : ''}" style="width:25%;${!w.has_markdown ? 'background:transparent' : ''}"></div>
        <div class="ov-stacked-seg ${w.has_pack ? 'ov-c3' : ''}" style="width:25%;${!w.has_pack ? 'background:transparent' : ''}"></div>
        <div class="ov-stacked-seg ${w.has_fts ? 'ov-c4' : ''}" style="width:25%;${!w.has_fts ? 'background:transparent' : ''}"></div>
      </div>`;

    const nextBtnLabel = next || 'done';
    const isRunning = warcRunning.has(w.index);
    const nextBtnCls = next ? (isRunning ? 'ui-btn' : 'ui-btn-primary') : 'ui-btn';
    const nextAction = next === 'download' ? `warcAction('${esc(w.index)}','download')`
      : next === 'markdown' ? `warcAction('${esc(w.index)}','markdown')`
      : next === 'pack' ? `warcAction('${esc(w.index)}','pack',{format:'parquet'})`
      : next === 'index' ? `warcAction('${esc(w.index)}','index',{engine:currentSearchEngine(),source:'files'})`
      : '';
    const btnId = warcBtnId(w.index);

    return `
    <tr class="anim-fade-up" style="animation-delay:${Math.min(i, 20)*12}ms">
      <td class="px-3 py-2 font-mono text-xs"><a href="#/warc/${encodeURIComponent(w.index)}" class="ui-link hover:text-[var(--accent)]">${esc(w.index)}</a></td>
      <td class="px-3 py-2 text-xs">
        <div class="flex items-center gap-2">
          ${stackedBar}
          <span class="text-[10px] font-mono ui-subtle">${done}/4</span>
        </div>
      </td>
      <td class="px-3 py-2 text-right text-xs font-mono ui-subtle">${docsStr}</td>
      <td class="px-3 py-2 text-right text-xs font-mono ui-subtle whitespace-nowrap hidden sm:table-cell">${sizeStr}</td>
      <td class="px-3 py-2 text-right text-xs font-mono whitespace-nowrap">
        ${next ? `<button id="${btnId}" onclick="${nextAction}" ${isRunning ? 'disabled' : ''} class="${nextBtnCls} px-2.5 py-1 text-[11px]">${isRunning ? 'running\u2026' : esc(nextBtnLabel)}</button>` : `<span class="text-[11px] status-completed">\u2713 done</span>`}
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
  const incomplete = visibleRows.filter(w => !w.has_warc || !w.has_markdown || !w.has_pack || !w.has_fts);

  // System stats from API response
  const sys = state.warcSystem || {};
  const sysHTML = (sys.disk_total || sys.mem_alloc) ? `
    <details class="mt-4">
      <summary class="text-[11px] font-mono ui-subtle cursor-pointer select-none">System</summary>
      <div class="grid grid-cols-2 sm:grid-cols-4 gap-3 mt-3">
        ${sys.disk_total ? `
          <div>
            <div class="text-[10px] font-mono ui-subtle">Disk</div>
            <div class="text-xs font-mono">${fmtBytes(sys.disk_used || 0)} / ${fmtBytes(sys.disk_total)}</div>
            <div class="progress-track mt-1" style="height:4px">
              <div class="ov-c5" style="height:100%;width:${sys.disk_total > 0 ? Math.round(((sys.disk_used||0)/sys.disk_total)*100) : 0}%;opacity:0.6"></div>
            </div>
            <div class="text-[10px] font-mono ui-subtle mt-0.5">${fmtBytes(sys.disk_free || 0)} free</div>
          </div>` : ''}
        ${sys.mem_alloc ? `
          <div>
            <div class="text-[10px] font-mono ui-subtle">Heap Alloc</div>
            <div class="text-xs font-mono">${fmtBytes(sys.mem_alloc)}</div>
          </div>` : ''}
        ${sys.mem_heap_sys ? `
          <div>
            <div class="text-[10px] font-mono ui-subtle">Heap Sys</div>
            <div class="text-xs font-mono">${fmtBytes(sys.mem_heap_sys)}</div>
          </div>` : ''}
        ${sys.goroutines ? `
          <div>
            <div class="text-[10px] font-mono ui-subtle">Goroutines</div>
            <div class="text-xs font-mono">${sys.goroutines.toLocaleString()}</div>
          </div>` : ''}
      </div>
    </details>` : '';

  el.innerHTML = `
    ${statCards}
    ${pipelineHTML}
    ${isDashboard && incomplete.length > 0 ? `
      <div class="flex items-center gap-2 mb-3">
        <span class="text-[11px] font-mono ui-subtle">${incomplete.length} incomplete${filter !== 'all' ? ' (filtered)' : ''}</span>
        <button id="warc-batch-btn" onclick="warcBatchNext()" class="ui-btn px-3 py-1 text-[11px] font-mono">Run next step (${incomplete.length})</button>
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
    </div>
    ${sysHTML}`;
}

// Batch: run next missing step on all visible incomplete WARCs
async function warcBatchNext() {
  const filter = state.warcFilter || 'all';
  let warcs = state.warcRows || [];
  if (filter === 'incomplete') {
    warcs = warcs.filter(w => !w.has_warc || !w.has_markdown || !w.has_pack || !w.has_fts);
  } else if (filter === 'complete') {
    warcs = warcs.filter(w => w.has_warc && w.has_markdown && w.has_pack && w.has_fts);
  }
  const incomplete = warcs.filter(w => !w.has_warc || !w.has_markdown || !w.has_pack || !w.has_fts);
  if (incomplete.length === 0) return;
  if (!confirm(`Run the next pipeline step on ${incomplete.length} incomplete WARC${incomplete.length !== 1 ? 's' : ''}?`)) return;

  const batchBtn = $('warc-batch-btn');
  if (batchBtn) { batchBtn.disabled = true; batchBtn.textContent = 'running\u2026'; }

  for (const w of incomplete) {
    if (!w.has_warc) {
      warcAction(w.index, 'download');
    } else if (!w.has_markdown) {
      warcAction(w.index, 'markdown');
    } else if (!w.has_pack) {
      warcAction(w.index, 'pack', { format: 'parquet' });
    } else if (!w.has_fts) {
      warcAction(w.index, 'index', { engine: currentSearchEngine(), source: 'files' });
    }
  }
}

// ── Detail page ──
async function renderWARCDetail(index) {
  state.currentPage = 'warc';
  state.warcDetail = null;
  warcActionMessage = '';
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
    renderWARCDetailContent(data, index);
  } catch (e) {
    $('warc-detail-content').innerHTML = `<div class="text-xs text-red-400">${esc(e.message)}</div>`;
  }
}

function renderWARCDetailContent(data, index) {
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

  const stepsAll = [
    { key: 'download', label: 'Download', done: !!w.has_warc, action: 'download', cls: 'ov-c1' },
    { key: 'markdown', label: 'Markdown', done: !!w.has_markdown, action: 'markdown', cls: 'ov-c2' },
    { key: 'pack', label: 'Pack', done: !!w.has_pack, action: 'pack', cls: 'ov-c3' },
    { key: 'index', label: 'Index', done: !!w.has_fts, action: 'index', cls: 'ov-c4' },
  ];
  const nextStep = stepsAll.find(s => !s.done);
  const done = stepsAll.filter(s => s.done).length;

  const enginesOpts = (state.engines||[]).map(e=>`<option value="${esc(e)}">${esc(e)}</option>`).join('') ||
    `<option value="${DEFAULT_ENGINE}">${DEFAULT_ENGINE}</option>`;

  // Active jobs
  const relatedJobs = data.jobs || [];
  const activeJobs = relatedJobs.filter(j => j.status === 'running' || j.status === 'queued');

  // Compression ratio
  const compressionRatio = (w.warc_bytes > 0 && (w.warc_md_bytes || 0) > 0) ? ((w.warc_md_bytes / w.warc_bytes) * 100).toFixed(1) + '%' : '\u2014';
  const docsCount = w.warc_md_docs || 0;

  // System stats
  const sys = data.system || {};

  // Step timeline - visual stepper
  const timelineHTML = `
    <div class="surface p-4 mb-4">
      <div class="text-[11px] font-mono ui-subtle mb-3">Pipeline Progress</div>
      <div class="flex items-center gap-0">
        ${stepsAll.map((s, i) => {
          const isNext = nextStep && nextStep.key === s.key;
          const dotBg = s.done ? clsColorMap[s.cls] || 'var(--accent)' : isNext ? 'var(--border)' : 'transparent';
          const dotBorder = s.done ? 'transparent' : 'var(--border)';
          const textCls = s.done ? 'status-completed' : isNext ? '' : 'ui-subtle';
          const checkmark = s.done ? '\u2713' : isNext ? '\u25CB' : '';
          return `
            ${i > 0 ? `<div class="flex-1 h-px" style="background:${stepsAll[i-1].done ? clsColorMap[stepsAll[i-1].cls] : 'var(--border)'}"></div>` : ''}
            <div class="flex flex-col items-center gap-1 shrink-0" style="min-width:56px">
              <div class="w-6 h-6 flex items-center justify-center text-[10px] font-mono font-medium"
                style="border:2px solid ${dotBorder};background:${dotBg};color:${s.done ? '#fff' : 'var(--text)'};border-radius:50%!important">
                ${checkmark}
              </div>
              <span class="text-[10px] font-mono ${textCls}">${s.label}</span>
            </div>`;
        }).join('')}
      </div>
    </div>`;

  // Primary action buttons
  let primaryActionHTML = '';
  if (nextStep) {
    const actionMap = {
      download: { label: 'Download WARC', extra: '{}' },
      markdown: { label: 'Extract Markdown', extra: '{fast:false}' },
      pack: { label: 'Pack (parquet)', extra: "{format:'parquet'}" },
      index: { label: `Build Index (${esc(currentSearchEngine())})`, extra: `{engine:currentSearchEngine(),source:'files'}` },
    };
    const a = actionMap[nextStep.key];
    const remaining = stepsAll.filter(s => !s.done).length;
    primaryActionHTML = `
      <div class="surface p-4 mb-4">
        <div class="flex flex-col sm:flex-row items-start sm:items-center gap-3">
          <div class="flex-1">
            <div class="text-[11px] font-mono ui-subtle mb-1">Next Step</div>
            <div class="text-sm font-mono font-medium">${esc(nextStep.label)}</div>
            ${remaining > 1 ? `<div class="text-[10px] font-mono ui-subtle mt-0.5">${remaining} steps remaining</div>` : ''}
          </div>
          <div class="flex items-center gap-2">
            <button onclick="warcAction('${esc(w.index)}','${nextStep.action}',${a.extra},true)" class="ui-btn ui-btn-primary px-4 py-2 text-xs font-mono">${a.label}</button>
            ${remaining > 1 ? `<button onclick="warcRunAll('${esc(w.index)}')" class="ui-btn px-3 py-2 text-xs font-mono" title="Run all remaining steps sequentially">Run All</button>` : ''}
          </div>
        </div>
      </div>`;
  } else {
    primaryActionHTML = `
      <div class="surface p-4 mb-4">
        <div class="flex items-center gap-2">
          <span class="text-sm font-mono status-completed">\u2713 All pipeline steps complete</span>
        </div>
      </div>`;
  }

  // Active jobs section
  const activeJobsHTML = activeJobs.length > 0 ? `
    <div class="surface p-4 mb-4 border-l-2" style="border-left-color:#2563eb">
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
    </div>` : '';

  $('warc-detail-content').innerHTML = `
    <div class="page-header mb-3">
      <h1 class="page-title">WARC ${esc(w.index || index)}</h1>
      <button onclick="renderWARCDetail('${esc(w.index || index)}')" class="ui-btn px-3 py-2 text-xs font-mono">Reload</button>
    </div>
    <div id="warc-action-msg" class="meta-line mb-3">${esc(warcActionMessage)}</div>

    ${activeJobsHTML}
    ${timelineHTML}
    ${primaryActionHTML}

    <!-- Info + Disk side by side -->
    <div class="grid md:grid-cols-2 gap-4 mb-4">
      <div class="surface p-4">
        <div class="text-[11px] font-mono ui-subtle mb-3">Info</div>
        <div class="space-y-2">
          <div class="flex items-center justify-between">
            <span class="text-[10px] font-mono ui-subtle">Index</span>
            <span class="text-xs font-mono font-medium">${esc(w.index || index)}</span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-[10px] font-mono ui-subtle">Documents</span>
            <span class="text-xs font-mono font-medium">${docsCount.toLocaleString()}</span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-[10px] font-mono ui-subtle">Total on Disk</span>
            <span class="text-xs font-mono font-medium">${fmtBytes(w.total_bytes || 0)}</span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-[10px] font-mono ui-subtle">MD / WARC Ratio</span>
            <span class="text-xs font-mono font-medium">${compressionRatio}</span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-[10px] font-mono ui-subtle">Progress</span>
            <span class="text-xs font-mono font-medium">${done}/4</span>
          </div>
          ${w.updated_at ? `
          <div class="flex items-center justify-between">
            <span class="text-[10px] font-mono ui-subtle">Updated</span>
            <span class="text-xs font-mono">${fmtRelativeTime(w.updated_at)}</span>
          </div>` : ''}
        </div>
        ${w.filename ? `
        <div class="mt-3 pt-3 border-t">
          <div class="text-[10px] font-mono ui-subtle mb-1">File</div>
          <div class="text-[11px] font-mono break-all">${esc(w.filename)}</div>
          ${w.remote_path ? `<div class="text-[10px] font-mono ui-subtle break-all mt-1">${esc(w.remote_path)}</div>` : ''}
        </div>` : ''}
      </div>

      <div class="surface p-4">
        <div class="text-[11px] font-mono ui-subtle mb-3">Disk Breakdown</div>
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

    <!-- Advanced actions -->
    <details class="surface mb-4">
      <summary class="p-4 text-[11px] font-mono ui-subtle cursor-pointer select-none">Advanced Actions</summary>
      <div class="px-4 pb-4">
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
      </div>
    </details>

    <!-- System info -->
    ${(sys.disk_total || sys.mem_alloc) ? `
    <details class="surface mb-4">
      <summary class="p-4 text-[11px] font-mono ui-subtle cursor-pointer select-none">System</summary>
      <div class="px-4 pb-4 grid grid-cols-2 sm:grid-cols-4 gap-3">
        ${sys.disk_total ? `
          <div>
            <div class="text-[10px] font-mono ui-subtle">Disk</div>
            <div class="text-xs font-mono">${fmtBytes(sys.disk_used || 0)} / ${fmtBytes(sys.disk_total)}</div>
            <div class="progress-track mt-1" style="height:4px">
              <div class="ov-c5" style="height:100%;width:${sys.disk_total > 0 ? Math.round(((sys.disk_used||0)/sys.disk_total)*100) : 0}%;opacity:0.6"></div>
            </div>
          </div>` : ''}
        ${sys.mem_alloc ? `
          <div>
            <div class="text-[10px] font-mono ui-subtle">Heap</div>
            <div class="text-xs font-mono">${fmtBytes(sys.mem_alloc)}</div>
          </div>` : ''}
        ${sys.mem_stack_inuse ? `
          <div>
            <div class="text-[10px] font-mono ui-subtle">Stack</div>
            <div class="text-xs font-mono">${fmtBytes(sys.mem_stack_inuse)}</div>
          </div>` : ''}
        ${sys.goroutines ? `
          <div>
            <div class="text-[10px] font-mono ui-subtle">Goroutines</div>
            <div class="text-xs font-mono">${sys.goroutines.toLocaleString()}</div>
          </div>` : ''}
      </div>
    </details>` : ''}

    <!-- Related Jobs -->
    <div class="surface p-4">
      <div class="flex items-center justify-between mb-2">
        <div class="text-[11px] font-mono ui-subtle">Related Jobs (${relatedJobs.length})</div>
        ${relatedJobs.length > 5 ? `<span class="text-[10px] font-mono ui-subtle">last ${Math.min(relatedJobs.length, 20)}</span>` : ''}
      </div>
      ${renderJobHistory(relatedJobs.slice(0, 20))}
    </div>`;
}

// Run all remaining pipeline steps sequentially
async function warcRunAll(index) {
  const data = state.warcDetail;
  if (!data) return;
  const w = data.warc || {};
  const steps = [];
  if (!w.has_warc) steps.push({ action: 'download', extra: {} });
  if (!w.has_markdown) steps.push({ action: 'markdown', extra: { fast: false } });
  if (!w.has_pack) steps.push({ action: 'pack', extra: { format: 'parquet' } });
  if (!w.has_fts) steps.push({ action: 'index', extra: { engine: currentSearchEngine(), source: 'files' } });
  if (steps.length === 0) return;
  if (!confirm(`Run ${steps.length} remaining pipeline step${steps.length !== 1 ? 's' : ''} for WARC ${index}?`)) return;

  for (const step of steps) {
    await warcAction(index, step.action, step.extra, false);
  }
  setTimeout(() => renderWARCDetail(index), 500);
}

let warcActionMessage = '';
const warcRunning = new Set(); // track indices with in-flight actions

function warcBtnId(index) { return 'warc-btn-' + index.replace(/\W/g, '_'); }

function setWarcBtnRunning(index, running) {
  if (running) warcRunning.add(index); else warcRunning.delete(index);
  const btn = $(warcBtnId(index));
  if (!btn) return;
  if (running) {
    btn.disabled = true;
    btn.dataset.origLabel = btn.textContent;
    btn.textContent = 'running\u2026';
    btn.classList.remove('ui-btn-primary');
    btn.classList.add('ui-btn');
  } else {
    btn.disabled = false;
    if (btn.dataset.origLabel) btn.textContent = btn.dataset.origLabel;
  }
}

async function warcAction(index, action, extra = {}, refreshDetail = false) {
  const msg = $('warc-action-msg');
  if (msg) msg.textContent = `running ${action} on ${index}\u2026`;
  warcActionMessage = '';
  setWarcBtnRunning(index, true);
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
    setWarcBtnRunning(index, false);
  }
}
