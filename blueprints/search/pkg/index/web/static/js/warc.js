// ===================================================================
// Tab 5: WARC Console  (4-phase pipeline: Download → Markdown → Index → Export)
// ===================================================================

const WARC_PHASES = [
  { key: '', label: 'All' },
  { key: 'downloaded', label: 'Downloaded' },
  { key: 'markdown', label: 'Markdown' },
  { key: 'indexed', label: 'Indexed' },
  { key: 'exported', label: 'Exported' },
];

function warcPhaseCount(summary, key) {
  if (!summary) return 0;
  const t = summary.total || 0;
  const dl = summary.downloaded || 0;
  const md = summary.markdown_ready || 0;
  const ix = summary.indexed || 0;
  const ex = summary.packed || 0;
  switch (key) {
    case '': return t;
    case 'downloaded': return dl;
    case 'markdown': return md;
    case 'indexed': return ix;
    case 'exported': return ex;
    default: return 0;
  }
}

function hasParquetExport(w) {
  return ((w.pack_bytes || {}).parquet || 0) > 0;
}

// What does this WARC need next? (4-phase)
function warcNextStep(w) {
  if (!w.has_warc) return 'download';
  if (!w.has_markdown) return 'markdown';
  if (!w.has_fts) return 'index';
  if (!hasParquetExport(w)) return 'export';
  return '';
}

function warcDoneCount(w) {
  return (w.has_warc ? 1 : 0) + (w.has_markdown ? 1 : 0) + (w.has_fts ? 1 : 0) + (hasParquetExport(w) ? 1 : 0);
}

// ── Main entry ──
async function renderWARC(offset = state.warcOffset || 0, query = state.warcQuery || '') {
  state.currentPage = 'warc';
  state.warcOffset = offset;
  state.warcQuery = query;
  const phase = state.warcPhase || '';
  const pageSize = 200;
  const main = $('main');
  const hasCache = !!(state.warcRows && state.warcSummary);
  main.innerHTML = `
    <div class="page-shell${hasCache ? '' : ' anim-fade-in'}">
      <div class="page-header mb-4">
        <h1 class="page-title">WARC Console</h1>
        <span id="warc-live" class="text-[10px] font-mono ui-subtle">● live</span>
      </div>
      <div id="warc-summary"></div>
      <div id="warc-tabs" class="mb-3"></div>
      <div class="mb-3">
        <input id="warc-q" type="text" value="${esc(query)}" placeholder="Filter by index or filename\u2026"
          class="ui-input w-full text-sm px-3 py-2"
          onkeydown="if(event.key==='Enter'){state.warcOffset=0;renderWARC(0,this.value)}">
      </div>
      <div id="warc-content"></div>
    </div>`;

  // Render cached data immediately — no flash.
  if (hasCache) {
    renderWARCSummary(state.warcSummary);
    renderWARCTabs(state.warcSummary, state.warcTotal);
    renderWARCTable({ warcs: state.warcRows, summary: state.warcSummary, total: state.warcTotal,
      offset: state.warcOffset, limit: state.warcLimit });
  }

  // Subscribe to WS job updates so the table auto-refreshes when jobs complete.
  ensureJobStreamSubscribed();

  try {
    const [data] = await Promise.all([
      apiWARCList({ offset, limit: pageSize, q: query || '', phase }),
      refreshCentralState(),
    ]);
    state.warcRows = data.warcs || [];
    state.warcSummary = data.summary || null;
    state.warcSystem = data.system || null;
    state.warcOffset = data.offset || 0;
    state.warcLimit = data.limit || pageSize;
    state.warcTotal = data.total || 0;
    if (state.currentPage === 'warc') {
      renderWARCSummary(data.summary);
      renderWARCTabs(data.summary, data.total);
      renderWARCTable(data);
    }
  } catch (e) {
    if ($('warc-content')) $('warc-content').innerHTML = `<div class="text-xs text-red-400">${esc(e.message)}</div>`;
  }
}

// ── Summary: global stats + 4-phase pipeline ──
function renderWARCSummary(summary) {
  const el = $('warc-summary');
  if (!el || !summary) return;
  const t = summary.total || 0;
  const dl = summary.downloaded || 0;
  const md = summary.markdown_ready || 0;
  const ix = summary.indexed || 0;
  const ex = summary.packed || 0;
  const totalBytes = (summary.warc_bytes || 0) + (summary.markdown_bytes || 0) + (summary.pack_bytes || 0) + (summary.fts_bytes || 0);

  // Get manifest total from central state for accurate total
  const ov = state.central.overview || {};
  const mf = ov.manifest || {};
  const manifestTotal = mf.total_warcs || 0;
  const total = manifestTotal > 0 ? manifestTotal : t;

  if (total <= 0) { el.innerHTML = ''; return; }

  function pctOf(n, base) { return base > 0 ? Math.round((n / base) * 100) : 0; }

  const stages = [
    { label: 'Downloaded', count: dl, cls: 'ov-c1', bytes: summary.warc_bytes || 0 },
    { label: 'Markdown', count: md, cls: 'ov-c2', bytes: summary.markdown_bytes || 0 },
    { label: 'Indexed', count: ix, cls: 'ov-c4', bytes: summary.fts_bytes || 0 },
    { label: 'Exported', count: ex, cls: 'ov-c3', bytes: summary.pack_bytes || 0 },
  ];

  // Active jobs from central state
  const activeJobs = (state.central.jobs || []).filter(j => j.status === 'running' || j.status === 'queued');
  const activeJobsHTML = activeJobs.length > 0 ? `
    <div class="surface p-4 mb-4 border-l-2" style="border-left-color:#3b82f6">
      <div class="flex items-center justify-between mb-2">
        <span class="text-[11px] font-mono" style="color:#3b82f6">Active Jobs (${activeJobs.length})</span>
        <a href="#/jobs" class="text-[11px] font-mono ui-link">view all &rarr;</a>
      </div>
      ${activeJobs.slice(0, 3).map(j => {
        const p = Math.round((j.progress || 0) * 100);
        const rateStr = j.rate > 0 ? ` &middot; ${j.rate.toFixed(0)}/s` : '';
        return `
          <div class="mb-2 last:mb-0">
            <div class="flex items-center justify-between mb-1">
              <span class="text-[10px] font-mono ui-subtle">${esc(j.id)} &middot; ${esc(j.type)} &middot; <span class="status-running">${esc(j.status)}</span>${rateStr}</span>
              <span class="text-[10px] font-mono">${p}%</span>
            </div>
            <div class="progress-track" style="height:4px"><div class="progress-fill" style="width:${p}%"></div></div>
          </div>`;
      }).join('')}
      ${activeJobs.length > 3 ? `<div class="text-[10px] font-mono ui-subtle mt-1">+${activeJobs.length - 3} more</div>` : ''}
    </div>` : '';

  el.innerHTML = `
    ${activeJobsHTML}
    <div class="surface p-4 mb-4">
      <div class="ov-pipeline-flow flex items-stretch gap-0">
        <div class="ov-pipeline-step">
          <div class="flex items-baseline justify-between mb-1">
            <span class="text-[11px] font-mono ui-subtle">Total</span>
            <span class="text-[11px] font-mono">${total.toLocaleString()}</span>
          </div>
          <div class="progress-track" style="height:6px"><div class="ov-c1" style="height:100%;width:100%"></div></div>
          <div class="text-[10px] font-mono ui-subtle mt-1">${fmtBytes(totalBytes)} on disk</div>
        </div>
        ${stages.map((s, i) => `
          <div class="ov-pipeline-arrow">\u2192</div>
          <div class="ov-pipeline-step">
            <div class="flex items-baseline justify-between mb-1">
              <span class="text-[11px] font-mono ui-subtle">${esc(s.label)}</span>
              <span class="text-[11px] font-mono ${pctOf(s.count, total) === 100 ? 'status-completed' : s.count > 0 ? '' : 'ui-subtle'}">${s.count.toLocaleString()}</span>
            </div>
            <div class="progress-track" style="height:6px">
              <div class="${s.cls}" style="height:100%;width:${pctOf(s.count, total)}%;transition:width 0.4s ease"></div>
            </div>
            <div class="flex items-center justify-between mt-1">
              <span class="text-[10px] font-mono ui-subtle">${pctOf(s.count, total)}%</span>
              ${s.bytes > 0 ? `<span class="text-[10px] font-mono ui-subtle">${fmtBytes(s.bytes)}</span>` : ''}
            </div>
          </div>`).join('')}
      </div>
    </div>`;
}

// ── Phase tabs ──
function renderWARCTabs(summary, filteredTotal) {
  const el = $('warc-tabs');
  if (!el) return;
  const current = state.warcPhase || '';
  el.innerHTML = `
    <div class="flex items-center gap-0 border-b border-[var(--border)] overflow-x-auto" style="scrollbar-width:none">
      ${WARC_PHASES.map(p => {
        const count = warcPhaseCount(summary, p.key);
        const active = p.key === current;
        const cls = active ? 'tab-active' : 'tab-inactive';
        return `<button onclick="state.warcPhase='${p.key}';state.warcOffset=0;renderWARC(0,state.warcQuery)"
          class="px-3 py-2 text-[11px] font-mono whitespace-nowrap ${cls} transition-colors shrink-0">${esc(p.label)} <span class="ui-subtle">${count.toLocaleString()}</span></button>`;
      }).join('')}
    </div>`;
}

// ── Table + pagination ──
function renderWARCTable(data) {
  const el = $('warc-content');
  if (!el) return;
  const warcs = data.warcs || [];
  const pageTotal = data.total || 0;
  const offset = data.offset || 0;
  const limit = data.limit || 200;
  const summary = data.summary || state.warcSummary || {};
  const globalTotal = summary.total || 0;

  const rows = warcs.map((w, i) => {
    const next = warcNextStep(w);
    const done = warcDoneCount(w);
    const docsCount = w.warc_md_docs || w.markdown_docs || 0;
    const mdBytes = w.warc_md_bytes || w.markdown_bytes || 0;
    const docsStr = docsCount > 0 ? docsCount.toLocaleString() : (w.has_markdown ? '\u2014' : '');
    const parquetBytes = (w.pack_bytes || {}).parquet || 0;
    const parquetSizeStr = parquetBytes > 0 ? fmtBytes(parquetBytes) : '\u2014';
    const warcSizeStr = w.warc_bytes > 0 ? fmtBytes(w.warc_bytes) : '\u2014';
    const mdSizeStr = mdBytes > 0 ? fmtBytes(mdBytes) : (w.has_markdown ? '\u2014' : '');

    // 4-segment stacked bar
    const stackedBar = `
      <div class="ov-stacked" style="width:56px;height:6px">
        <div class="ov-stacked-seg ${w.has_warc ? 'ov-c1' : ''}" style="width:25%;${!w.has_warc ? 'background:transparent' : ''}"></div>
        <div class="ov-stacked-seg ${w.has_markdown ? 'ov-c2' : ''}" style="width:25%;${!w.has_markdown ? 'background:transparent' : ''}"></div>
        <div class="ov-stacked-seg ${w.has_fts ? 'ov-c4' : ''}" style="width:25%;${!w.has_fts ? 'background:transparent' : ''}"></div>
        <div class="ov-stacked-seg ${hasParquetExport(w) ? 'ov-c3' : ''}" style="width:25%;${!hasParquetExport(w) ? 'background:transparent' : ''}"></div>
      </div>`;

    const isRunning = warcRunning.has(w.index);
    const nextBtnCls = next ? (isRunning ? 'ui-btn' : 'ui-btn-primary') : 'ui-btn';
    const nextAction = warcActionCall(w.index, next);
    const btnId = warcBtnId(w.index);
    const btnLabel = next === 'download' ? 'download' : next === 'markdown' ? 'markdown' : next === 'index' ? 'index' : next === 'export' ? 'export' : '';

    return `
    <tr>
      <td class="px-3 py-2 font-mono text-xs"><a href="#/warc/${encodeURIComponent(w.index)}" class="ui-link hover:text-[var(--accent)]">${esc(w.index)}</a></td>
      <td class="px-3 py-2 text-xs">
        <div class="flex items-center gap-2">
          ${stackedBar}
          <span class="text-[10px] font-mono ui-subtle">${done}/4</span>
        </div>
      </td>
      <td class="px-3 py-2 text-right text-xs font-mono ui-subtle hidden sm:table-cell">${docsStr}</td>
      <td class="px-3 py-2 text-right text-xs font-mono ui-subtle whitespace-nowrap hidden md:table-cell">${warcSizeStr}</td>
      <td class="px-3 py-2 text-right text-xs font-mono ui-subtle whitespace-nowrap hidden lg:table-cell">${mdSizeStr}</td>
      <td class="px-3 py-2 text-right text-xs font-mono ui-subtle whitespace-nowrap hidden lg:table-cell">${parquetSizeStr}</td>
      <td class="px-3 py-2 text-right text-xs font-mono whitespace-nowrap">
        ${next ? `<button id="${btnId}" onclick="${nextAction}" ${isRunning ? 'disabled' : ''} class="${nextBtnCls} px-2.5 py-1 text-[11px]">${isRunning ? 'running\u2026' : esc(btnLabel)}</button>` : `<span class="text-[11px] status-completed">\u2713 done</span>`}
      </td>
    </tr>`;
  }).join('');

  // Pagination
  const currentPage = Math.floor(offset / limit) + 1;
  const totalPages = Math.max(1, Math.ceil(pageTotal / limit));
  const canPrev = offset > 0;
  const canNext = offset + limit < pageTotal;
  const showFrom = pageTotal > 0 ? offset + 1 : 0;
  const showTo = Math.min(offset + limit, pageTotal);

  const pageButtons = buildPageNumbers(currentPage, totalPages).map(p => {
    if (p === '...') return `<span class="text-[10px] font-mono ui-subtle px-1">\u2026</span>`;
    const active = p === currentPage;
    return `<button onclick="renderWARC(${(p - 1) * limit},state.warcQuery)" class="ui-btn px-2 py-1 text-[10px] font-mono ${active ? 'ui-btn-primary' : ''}" ${active ? 'disabled' : ''}>${p}</button>`;
  }).join('');

  // System
  const sys = state.warcSystem || {};
  const sysHTML = (sys.disk_total || sys.mem_alloc) ? `
    <details class="mt-4">
      <summary class="text-[11px] font-mono ui-subtle cursor-pointer select-none">System</summary>
      <div class="grid grid-cols-2 sm:grid-cols-4 gap-3 mt-3">
        ${sys.disk_total ? `<div><div class="text-[10px] font-mono ui-subtle">Disk</div><div class="text-xs font-mono">${fmtBytes(sys.disk_used || 0)} / ${fmtBytes(sys.disk_total)}</div><div class="progress-track mt-1" style="height:4px"><div class="ov-c5" style="height:100%;width:${sys.disk_total > 0 ? Math.round(((sys.disk_used||0)/sys.disk_total)*100) : 0}%;opacity:0.6"></div></div></div>` : ''}
        ${sys.mem_alloc ? `<div><div class="text-[10px] font-mono ui-subtle">Heap</div><div class="text-xs font-mono">${fmtBytes(sys.mem_alloc)}</div></div>` : ''}
        ${sys.goroutines ? `<div><div class="text-[10px] font-mono ui-subtle">Goroutines</div><div class="text-xs font-mono">${sys.goroutines.toLocaleString()}</div></div>` : ''}
      </div>
    </details>` : '';

  el.innerHTML = `
    <div class="surface overflow-x-auto">
      <table class="w-full text-sm ui-table">
        <thead>
          <tr>
            <th class="text-left px-3 py-2 text-[11px] font-mono">Index</th>
            <th class="text-left px-3 py-2 text-[11px] font-mono">Pipeline</th>
            <th class="text-right px-3 py-2 text-[11px] font-mono hidden sm:table-cell">Docs</th>
            <th class="text-right px-3 py-2 text-[11px] font-mono hidden md:table-cell">warc.gz</th>
            <th class="text-right px-3 py-2 text-[11px] font-mono hidden lg:table-cell">md.warc.gz</th>
            <th class="text-right px-3 py-2 text-[11px] font-mono hidden lg:table-cell">parquet</th>
            <th class="text-right px-3 py-2 text-[11px] font-mono">Action</th>
          </tr>
        </thead>
        <tbody>
          ${rows || `<tr><td colspan="7" class="px-3 py-4 text-xs font-mono ui-subtle">No WARC records</td></tr>`}
        </tbody>
      </table>
    </div>
    <div class="flex flex-col sm:flex-row items-center justify-between mt-3 gap-2">
      <div class="text-xs font-mono ui-subtle">${showFrom > 0 ? `${showFrom.toLocaleString()}\u2013${showTo.toLocaleString()} of ${pageTotal.toLocaleString()}` : 'no results'}${globalTotal !== pageTotal ? ` (${globalTotal.toLocaleString()} total)` : ''}</div>
      <div class="flex items-center gap-1.5 flex-wrap justify-center">
        <button ${canPrev ? '' : 'disabled'} onclick="renderWARC(0,state.warcQuery)" class="ui-btn px-2 py-1 text-[10px] font-mono" title="First">\u00ab</button>
        <button ${canPrev ? '' : 'disabled'} onclick="renderWARC(${Math.max(0, offset - limit)},state.warcQuery)" class="ui-btn px-2 py-1 text-[10px] font-mono">\u2190</button>
        ${pageButtons}
        <button ${canNext ? '' : 'disabled'} onclick="renderWARC(${offset + limit},state.warcQuery)" class="ui-btn px-2 py-1 text-[10px] font-mono">\u2192</button>
        <button ${canNext ? '' : 'disabled'} onclick="renderWARC(${(totalPages - 1) * limit},state.warcQuery)" class="ui-btn px-2 py-1 text-[10px] font-mono" title="Last">\u00bb</button>
        <span class="text-[10px] font-mono ui-subtle mx-1">page</span>
        <input type="number" min="1" max="${totalPages}" value="${currentPage}"
          class="ui-input w-14 text-xs px-2 py-1 text-center"
          onkeydown="if(event.key==='Enter'){const p=Math.max(1,Math.min(${totalPages},+this.value));renderWARC((p-1)*${limit},state.warcQuery)}"
          onchange="const p=Math.max(1,Math.min(${totalPages},+this.value));renderWARC((p-1)*${limit},state.warcQuery)">
        <span class="text-[10px] font-mono ui-subtle">/ ${totalPages.toLocaleString()}</span>
      </div>
    </div>
    ${sysHTML}`;
}

function buildPageNumbers(current, total) {
  if (total <= 9) return Array.from({ length: total }, (_, i) => i + 1);
  const pages = [];
  pages.push(1);
  if (current > 4) pages.push('...');
  const start = Math.max(2, current - 2);
  const end = Math.min(total - 1, current + 2);
  for (let i = start; i <= end; i++) pages.push(i);
  if (current < total - 3) pages.push('...');
  pages.push(total);
  return pages;
}

// ── Action helpers (4-phase) ──
function warcActionCall(index, step) {
  switch (step) {
    case 'download': return `warcAction('${esc(index)}','download')`;
    case 'markdown': return `warcAction('${esc(index)}','markdown')`;
    case 'index': return `warcAction('${esc(index)}','index',{engine:currentSearchEngine(),source:'files'})`;
    case 'export': return `warcAction('${esc(index)}','pack',{format:'parquet'})`;
    default: return '';
  }
}

function warcActionLabel(step) {
  switch (step) {
    case 'download': return 'Download WARC';
    case 'markdown': return 'Extract Markdown';
    case 'index': return `Build Index (${currentSearchEngine()})`;
    case 'export': return 'Export Parquet';
    default: return '';
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
      <div id="warc-detail-content" class="mt-4"></div>
    </div>`;
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

  // Sizes
  const packTotal = Object.values(w.pack_bytes || {}).reduce((a,b)=>a+b,0);
  const ftsTotal = Object.values(w.fts_bytes || {}).reduce((a,b)=>a+b,0);
  const mdBytes = w.warc_md_bytes || w.markdown_bytes || 0;
  const diskPhases = [
    { label: 'warc', value: w.warc_bytes || 0, cls: 'ov-c1' },
    { label: 'markdown', value: mdBytes, cls: 'ov-c2' },
    { label: 'pack', value: packTotal, cls: 'ov-c3' },
    { label: 'index', value: ftsTotal, cls: 'ov-c4' },
  ];
  const diskTotal = diskPhases.reduce((a, p) => a + p.value, 0) || 1;

  // Donut
  let donutAngle = 0;
  const donutStops = [];
  const clsColorMap = { 'ov-c1': 'var(--accent)', 'ov-c2': '#6366f1', 'ov-c3': '#f59e0b', 'ov-c4': '#10b981' };
  for (const p of diskPhases) {
    if (p.value <= 0) continue;
    const deg = (p.value / diskTotal) * 360;
    donutStops.push(`${clsColorMap[p.cls] || 'var(--border)'} ${donutAngle}deg ${donutAngle + deg}deg`);
    donutAngle += deg;
  }
  const donutGrad = donutStops.length > 0 ? `conic-gradient(${donutStops.join(', ')})` : 'var(--border)';

  // Breakdown bars
  const packEntries = Object.entries(w.pack_bytes || {});
  const ftsEntries = Object.entries(w.fts_bytes || {});
  const breakdownHTML = (packEntries.length > 0 || ftsEntries.length > 0) ? `
    <div class="mt-3 pt-3 border-t space-y-3">
      ${packEntries.length > 0 ? `<div><div class="text-[10px] font-mono ui-subtle mb-1">Pack formats</div>${renderBars(packEntries.map(([fmt, b]) => ({ label: fmt, value: b, text: fmtBytes(b) })))}</div>` : ''}
      ${ftsEntries.length > 0 ? `<div><div class="text-[10px] font-mono ui-subtle mb-1">Index engines</div>${renderBars(ftsEntries.map(([eng, b]) => ({ label: eng, value: b, text: fmtBytes(b) })))}</div>` : ''}
    </div>` : '';

  // 4-phase steps
  const stepsAll = [
    { key: 'download', label: 'Download', done: !!w.has_warc, cls: 'ov-c1' },
    { key: 'markdown', label: 'Markdown', done: !!w.has_markdown, cls: 'ov-c2' },
    { key: 'index', label: 'Index', done: !!w.has_fts, cls: 'ov-c4' },
    { key: 'export', label: 'Export', done: hasParquetExport(w), cls: 'ov-c3' },
  ];
  const done = stepsAll.filter(s => s.done).length;

  const relatedJobs = data.jobs || [];
  const activeJobs = relatedJobs.filter(j => j.status === 'running' || j.status === 'queued');

  const compressionRatio = (w.warc_bytes > 0 && mdBytes > 0) ? ((mdBytes / w.warc_bytes) * 100).toFixed(1) + '%' : '\u2014';
  const docsCount = w.warc_md_docs || w.markdown_docs || 0;
  // If markdown exists but no doc count yet, trigger a scan and reload after.
  if (w.has_markdown && docsCount === 0 && isDashboard && !w._scanTriggered) {
    w._scanTriggered = true;
    apiMetaScanDocs().then(() => {
      setTimeout(() => {
        if (state.currentPage === 'warc' && state.warcDetail) {
          apiWARCDetail(index).then(d => { state.warcDetail = d; if ($('warc-detail-content')) renderWARCDetailContent(d, index); }).catch(() => {});
        }
      }, 1500);
    }).catch(() => {});
  }

  // Step timeline (4 steps)
  const remainingSteps = stepsAll.filter(s => !s.done).length;
  const timelineHTML = `
    <div class="surface p-4 mb-4">
      <div class="flex items-center justify-between mb-3">
        <div class="text-[11px] font-mono ui-subtle">Pipeline ${done} / 4</div>
        ${remainingSteps > 1 ? `<button onclick="warcRunAll('${esc(w.index || index)}')" class="ui-btn px-3 py-1.5 text-xs font-mono">Run All Remaining</button>` : ''}
      </div>
      <div class="flex items-center gap-0">
        ${stepsAll.map((s, i) => {
          const isNext = !s.done && stepsAll.slice(0, i).every(prev => prev.done);
          const dotBg = s.done ? clsColorMap[s.cls] || 'var(--accent)' : isNext ? 'var(--border)' : 'transparent';
          const dotBorder = s.done ? 'transparent' : 'var(--border)';
          const textCls = s.done ? 'status-completed' : isNext ? '' : 'ui-subtle';
          const icon = s.done ? '\u2713' : isNext ? '\u25CB' : '';
          return `
            ${i > 0 ? `<div class="flex-1 h-px" style="background:${stepsAll[i-1].done ? clsColorMap[stepsAll[i-1].cls] : 'var(--border)'}"></div>` : ''}
            <div class="flex flex-col items-center gap-1 shrink-0" style="min-width:64px">
              <div class="w-7 h-7 flex items-center justify-center text-[11px] font-mono font-medium"
                style="border:2px solid ${dotBorder};background:${dotBg};color:${s.done ? '#fff' : 'var(--text)'};border-radius:50%!important">
                ${icon}
              </div>
              <span class="text-[11px] font-mono ${textCls}">${s.label}</span>
            </div>`;
        }).join('')}
      </div>
    </div>`;

  // Active jobs
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
            <div class="progress-track" style="height:4px"><div class="progress-fill" style="width:${pct}%"></div></div>
            ${j.message ? `<div class="text-[10px] font-mono ui-subtle mt-1 truncate">${esc(j.message)}</div>` : ''}
          </div>`;
      }).join('')}
    </div>` : '';

  // Four pipeline step panels
  const stepsHTML = `
    <div class="mb-4">
      <div class="text-[10px] font-mono ui-subtle uppercase tracking-wider mb-2">Pipeline Steps</div>

      <!-- Step 1: Download -->
      <div class="surface p-4 mb-2" style="border-left:3px solid ${w.has_warc ? 'var(--success)' : 'var(--border)'}">
        <div class="flex items-center justify-between gap-3">
          <div class="flex-1 min-w-0">
            <div class="flex items-center gap-2 mb-0.5">
              <span class="text-[10px] font-mono ui-subtle">1.</span>
              <span class="text-xs font-mono font-medium ${w.has_warc ? 'status-completed' : ''}">Download WARC${w.has_warc ? ' \u2713' : ''}</span>
            </div>
            ${w.warc_bytes > 0 ? `<div class="text-[10px] font-mono ui-subtle">${fmtBytes(w.warc_bytes)}</div>` : '<div class="text-[10px] font-mono ui-subtle">Not downloaded yet</div>'}
          </div>
          <button onclick="warcAction('${esc(w.index)}','download',{},true)"
            class="ui-btn ${!w.has_warc ? 'ui-btn-primary' : ''} px-3 py-1.5 text-xs font-mono shrink-0">
            ${w.has_warc ? 'Re-download' : 'Download'}
          </button>
        </div>
      </div>

      <!-- Step 2: Markdown -->
      <div class="surface p-4 mb-2" style="border-left:3px solid ${w.has_markdown ? 'var(--success)' : w.has_warc ? '#6366f1' : 'var(--border)'}; ${!w.has_warc ? 'opacity:0.6' : ''}">
        <div class="flex items-center justify-between gap-3">
          <div class="flex-1 min-w-0">
            <div class="flex items-center gap-2 mb-0.5">
              <span class="text-[10px] font-mono ui-subtle">2.</span>
              <span class="text-xs font-mono font-medium ${w.has_markdown ? 'status-completed' : ''}">Extract Markdown${w.has_markdown ? ' \u2713' : ''}</span>
            </div>
            ${mdBytes > 0
              ? `<div class="text-[10px] font-mono ui-subtle">${fmtBytes(mdBytes)} extracted${compressionRatio !== '\u2014' ? ` (${compressionRatio} of WARC)` : ''}</div>`
              : `<div class="text-[10px] font-mono ui-subtle">${w.has_warc ? 'Ready to extract' : 'Download first'}</div>`}
          </div>
          <button onclick="warcAction('${esc(w.index)}','markdown',{},true)" ${!w.has_warc ? 'disabled' : ''}
            class="ui-btn ${w.has_warc && !w.has_markdown ? 'ui-btn-primary' : ''} px-3 py-1.5 text-xs font-mono shrink-0">
            ${w.has_markdown ? 'Re-extract' : 'Extract'}
          </button>
        </div>
      </div>

      <!-- Step 3: Index -->
      <div class="surface p-4 mb-2" style="border-left:3px solid ${w.has_fts ? 'var(--success)' : w.has_markdown ? '#10b981' : 'var(--border)'}; ${!w.has_markdown ? 'opacity:0.6' : ''}">
        <div class="flex items-center justify-between gap-3">
          <div class="flex-1 min-w-0">
            <div class="flex items-center gap-2 mb-0.5">
              <span class="text-[10px] font-mono ui-subtle">3.</span>
              <span class="text-xs font-mono font-medium ${w.has_fts ? 'status-completed' : ''}">Build Index${w.has_fts ? ' \u2713' : ''}</span>
            </div>
            ${ftsTotal > 0
              ? `<div class="text-[10px] font-mono ui-subtle">${fmtBytes(ftsTotal)} index &middot; ${currentSearchEngine()}</div>`
              : `<div class="text-[10px] font-mono ui-subtle">${w.has_markdown ? 'Ready to index' : 'Extract markdown first'}</div>`}
          </div>
          <button onclick="warcAction('${esc(w.index)}','index',{engine:currentSearchEngine(),source:'files'},true)" ${!w.has_markdown ? 'disabled' : ''}
            class="ui-btn ${w.has_markdown && !w.has_fts ? 'ui-btn-primary' : ''} px-3 py-1.5 text-xs font-mono shrink-0">
            ${w.has_fts ? 'Re-index' : 'Build Index'}
          </button>
        </div>
      </div>

      <!-- Step 4: Export -->
      <div class="surface p-4" style="border-left:3px solid ${hasParquetExport(w) ? 'var(--success)' : w.has_fts ? '#f59e0b' : 'var(--border)'}; ${!w.has_fts ? 'opacity:0.6' : ''}">
        <div class="flex items-center justify-between gap-3">
          <div class="flex-1 min-w-0">
            <div class="flex items-center gap-2 mb-0.5">
              <span class="text-[10px] font-mono ui-subtle">4.</span>
              <span class="text-xs font-mono font-medium ${hasParquetExport(w) ? 'status-completed' : ''}">Export Parquet${hasParquetExport(w) ? ' \u2713' : ''}</span>
            </div>
            ${((w.pack_bytes || {}).parquet || 0) > 0
              ? `<div class="text-[10px] font-mono ui-subtle">${fmtBytes((w.pack_bytes || {}).parquet || 0)} parquet exported</div>`
              : `<div class="text-[10px] font-mono ui-subtle">${w.has_fts ? 'Ready to export parquet' : 'Build index first'}</div>`}
          </div>
          <button onclick="warcAction('${esc(w.index)}','pack',{format:'parquet'},true)" ${!w.has_fts ? 'disabled' : ''}
            class="ui-btn ${w.has_fts && !hasParquetExport(w) ? 'ui-btn-primary' : ''} px-3 py-1.5 text-xs font-mono shrink-0">
            ${hasParquetExport(w) ? 'Re-export' : 'Export'}
          </button>
        </div>
      </div>
    </div>`;

  // Danger Zone
  const diskSummary = [
    w.warc_bytes > 0 ? `warc.gz ${fmtBytes(w.warc_bytes)}` : '',
    mdBytes > 0 ? `markdown ${fmtBytes(mdBytes)}` : '',
    ftsTotal > 0 ? `index ${fmtBytes(ftsTotal)}` : '',
    packTotal > 0 ? `pack ${fmtBytes(packTotal)}` : '',
  ].filter(Boolean).join(' · ');
  const dangerHTML = `
    <div class="surface p-4 mb-4" style="border:1px solid rgba(239,68,68,0.4)">
      <div class="flex items-center gap-2 mb-3">
        <span class="text-[11px] font-mono font-medium" style="color:#f87171">\u26a0 Danger Zone</span>
        <span class="text-[10px] font-mono ui-subtle">\u2014 permanent, cannot be undone</span>
      </div>
      <div class="text-[10px] font-mono ui-subtle mb-3">
        WARC <span class="font-medium" style="color:var(--text)">${esc(w.index)}</span>
        ${diskSummary ? `\u00b7 ${diskSummary}` : ''}
        \u00b7 total ${fmtBytes(w.total_bytes || 0)} on disk
      </div>
      <div class="grid grid-cols-1 sm:grid-cols-3 gap-2">
        ${[
          { target: 'index', label: 'Delete Index', size: ftsTotal, color: '#10b981' },
          { target: 'markdown', label: 'Delete Markdown', size: mdBytes, color: '#6366f1' },
          { target: 'warc', label: 'Delete WARC', size: w.warc_bytes || 0, color: 'var(--accent)' },
        ].map(d => `
          <div class="p-3" style="border:1px solid rgba(239,68,68,0.25);border-radius:var(--radius)">
            <div class="text-[10px] font-mono font-medium mb-0.5" style="color:#f87171">${esc(d.label)}</div>
            <div class="text-[10px] font-mono ui-subtle mb-2">${d.size > 0 ? fmtBytes(d.size) + ' freed' : 'nothing on disk'}</div>
            <button onclick="warcConfirmDelete('${esc(w.index)}','${d.target}')"
              ${d.size <= 0 ? 'disabled' : ''}
              class="ui-btn ui-btn-danger w-full px-2 py-1.5 text-[11px] font-mono">Delete ${esc(d.target)}\u2026</button>
          </div>`).join('')}
      </div>
      <div class="mt-2 p-3" style="border:1px solid rgba(239,68,68,0.4);border-radius:var(--radius);background:rgba(239,68,68,0.05)">
        <div class="text-[10px] font-mono font-medium mb-0.5" style="color:#f87171">\u26a0 Delete All Data</div>
        <div class="text-[10px] font-mono ui-subtle mb-2">${fmtBytes(w.total_bytes || 0)} total \u2014 WARC + markdown + index + pack</div>
        <button onclick="warcConfirmDelete('${esc(w.index)}','all')"
          class="ui-btn ui-btn-danger px-3 py-1.5 text-[11px] font-mono">Delete all\u2026</button>
      </div>
    </div>`;

  $('warc-detail-content').innerHTML = `
    <div class="page-header mb-3">
      <h1 class="page-title">WARC ${esc(w.index || index)}</h1>
      <span class="text-[10px] font-mono ui-subtle">● live</span>
    </div>
    <div id="warc-action-msg" class="meta-line mb-3">${esc(warcActionMessage)}</div>

    ${activeJobsHTML}
    ${timelineHTML}
    ${stepsHTML}

    <!-- Info + Disk -->
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
            <span class="text-xs font-mono font-medium">${docsCount > 0 ? docsCount.toLocaleString() : (w.has_markdown ? '\u2014 scanning\u2026' : '\u2014')}</span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-[10px] font-mono ui-subtle">warc.gz</span>
            <span class="text-xs font-mono">${w.warc_bytes > 0 ? fmtBytes(w.warc_bytes) : '\u2014'}</span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-[10px] font-mono ui-subtle">md${w.warc_md_bytes > 0 ? '.warc.gz' : ''}</span>
            <span class="text-xs font-mono">${mdBytes > 0 ? fmtBytes(mdBytes) : '\u2014'}${compressionRatio !== '\u2014' ? ` <span class="ui-subtle">(${compressionRatio} of WARC)</span>` : ''}</span>
          </div>
          ${packTotal > 0 ? `<div class="flex items-center justify-between"><span class="text-[10px] font-mono ui-subtle">Pack</span><span class="text-xs font-mono">${fmtBytes(packTotal)}</span></div>` : ''}
          ${ftsTotal > 0 ? `<div class="flex items-center justify-between"><span class="text-[10px] font-mono ui-subtle">Index</span><span class="text-xs font-mono">${fmtBytes(ftsTotal)}</span></div>` : ''}
          <div class="flex items-center justify-between">
            <span class="text-[10px] font-mono ui-subtle">Total on Disk</span>
            <span class="text-xs font-mono font-medium">${fmtBytes(w.total_bytes || 0)}</span>
          </div>
          ${w.updated_at ? `<div class="flex items-center justify-between"><span class="text-[10px] font-mono ui-subtle">Updated</span><span class="text-xs font-mono">${fmtRelativeTime(w.updated_at)}</span></div>` : ''}
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
            ${diskPhases.filter(p => p.value > 0).map(p => `
              <div class="flex items-center gap-2">
                <div class="ov-legend-dot ${p.cls}"></div>
                <span class="text-[11px] font-mono ui-subtle flex-1">${esc(p.label)}</span>
                <span class="text-[11px] font-mono">${fmtBytes(p.value)}</span>
                <span class="text-[10px] font-mono ui-subtle w-8 text-right">${Math.round((p.value / diskTotal) * 100)}%</span>
              </div>`).join('')}
            ${diskPhases.every(p => p.value <= 0) ? '<div class="text-[11px] font-mono ui-subtle">no data on disk</div>' : ''}
          </div>
        </div>
        ${breakdownHTML}
      </div>
    </div>

    ${dangerHTML}

    <!-- Related Jobs -->
    <div class="surface p-4">
      <div class="flex items-center justify-between mb-2">
        <div class="text-[11px] font-mono ui-subtle">Related Jobs (${relatedJobs.length})</div>
        ${relatedJobs.length > 5 ? `<span class="text-[10px] font-mono ui-subtle">last ${Math.min(relatedJobs.length, 20)}</span>` : ''}
      </div>
      ${renderJobHistory(relatedJobs.slice(0, 20))}
    </div>`;
}

// Run all remaining steps sequentially (4-phase)
async function warcRunAll(index) {
  const data = state.warcDetail;
  if (!data) return;
  const w = data.warc || {};
  const steps = [];
  if (!w.has_warc) steps.push({ action: 'download', extra: {} });
  if (!w.has_markdown) steps.push({ action: 'markdown', extra: {} });
  if (!w.has_fts) steps.push({ action: 'index', extra: { engine: currentSearchEngine(), source: 'files' } });
  if (!hasParquetExport(w)) steps.push({ action: 'pack', extra: { format: 'parquet' } });
  if (steps.length === 0) return;
  if (!confirm(`Run ${steps.length} remaining step${steps.length !== 1 ? 's' : ''} for WARC ${index}?`)) return;

  for (const step of steps) {
    await warcAction(index, step.action, step.extra, false);
  }
  setTimeout(() => renderWARCDetail(index), 500);
}

// ── Action state tracking ──
let warcActionMessage = '';
const warcRunning = new Set();

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

// GitHub-style typed-confirmation delete for WARC danger zone.
async function warcConfirmDelete(index, target) {
  const confirmWord = target === 'all' ? index : target;
  const what = target === 'all' ? `ALL data for WARC ${index}` : `${target} for WARC ${index}`;

  // Build a modal overlay with typed confirmation.
  const overlay = document.createElement('div');
  overlay.style.cssText = 'position:fixed;inset:0;background:rgba(0,0,0,0.7);z-index:9999;display:flex;align-items:center;justify-content:center;padding:1rem';
  overlay.innerHTML = `
    <div style="background:var(--panel);border:1px solid rgba(239,68,68,0.5);border-radius:var(--radius);padding:1.5rem;max-width:420px;width:100%">
      <div style="font-size:13px;font-weight:600;color:#f87171;margin-bottom:0.75rem">\u26a0 Confirm Deletion</div>
      <div style="font-size:12px;color:var(--text-muted);margin-bottom:1rem;line-height:1.5">
        This will permanently delete <strong style="color:var(--text)">${esc(what)}</strong>.<br>
        This action <strong>cannot be undone</strong>.
      </div>
      <div style="font-size:11px;color:var(--text-muted);margin-bottom:0.5rem;font-family:monospace">
        Type <strong style="color:var(--text)">${esc(confirmWord)}</strong> to confirm:
      </div>
      <input id="danger-confirm-input" type="text" autocomplete="off" spellcheck="false"
        placeholder="${esc(confirmWord)}"
        style="width:100%;box-sizing:border-box;padding:0.5rem 0.75rem;font-family:monospace;font-size:12px;background:var(--bg);border:1px solid var(--border);border-radius:var(--radius);color:var(--text);margin-bottom:1rem;outline:none">
      <div style="display:flex;gap:0.5rem;justify-content:flex-end">
        <button id="danger-cancel-btn" style="padding:0.4rem 1rem;font-size:12px;font-family:monospace;background:transparent;border:1px solid var(--border);border-radius:var(--radius);color:var(--text-muted);cursor:pointer">Cancel</button>
        <button id="danger-confirm-btn" disabled
          style="padding:0.4rem 1rem;font-size:12px;font-family:monospace;background:rgba(239,68,68,0.15);border:1px solid rgba(239,68,68,0.5);border-radius:var(--radius);color:#f87171;cursor:pointer;opacity:0.5">
          Delete ${esc(target)}
        </button>
      </div>
    </div>`;
  document.body.appendChild(overlay);

  const input = overlay.querySelector('#danger-confirm-input');
  const confirmBtn = overlay.querySelector('#danger-confirm-btn');
  const cancelBtn = overlay.querySelector('#danger-cancel-btn');

  function close() { document.body.removeChild(overlay); }

  input.addEventListener('input', () => {
    const match = input.value === confirmWord;
    confirmBtn.disabled = !match;
    confirmBtn.style.opacity = match ? '1' : '0.5';
    confirmBtn.style.cursor = match ? 'pointer' : 'not-allowed';
  });
  input.addEventListener('keydown', e => { if (e.key === 'Escape') close(); });
  cancelBtn.addEventListener('click', close);
  overlay.addEventListener('click', e => { if (e.target === overlay) close(); });
  confirmBtn.addEventListener('click', () => {
    if (confirmBtn.disabled) return;
    close();
    warcAction(index, 'delete', { target, engine: currentSearchEngine() }, true);
  });

  requestAnimationFrame(() => input.focus());
}

async function warcAction(index, action, extra = {}, refreshDetail = false) {
  const msg = $('warc-action-msg');
  if (msg) msg.textContent = `running ${action} on ${index}\u2026`;
  warcActionMessage = '';
  setWarcBtnRunning(index, true);
  try {
    const res = await apiWARCAction(index, { action, ...extra });
    if (res && res.job && res.job.id) {
      if (!state.central.jobs) state.central.jobs = [];
      state.central.jobs.unshift(res.job);
      updateHeaderStatus();
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
