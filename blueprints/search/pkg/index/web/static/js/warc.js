// ===================================================================
// Tab 5: WARC Console  (3-phase pipeline: Download → Markdown → Index)
// ===================================================================

const WARC_PHASES = [
  { key: '', label: 'All' },
  { key: 'download', label: 'Download' },
  { key: 'markdown', label: 'Markdown' },
  { key: 'index', label: 'Index' },
  { key: 'complete', label: 'Complete' },
];

function warcPhaseCount(summary, key) {
  if (!summary) return 0;
  const t = summary.total || 0;
  const dl = summary.downloaded || 0;
  const md = summary.markdown_ready || 0;
  const ix = summary.indexed || 0;
  switch (key) {
    case '': return t;
    case 'download': return t - dl;
    case 'markdown': return dl - md;
    case 'index': return md - ix;
    case 'complete': return ix;
    default: return 0;
  }
}

// What does this WARC need next? (3-phase)
function warcNextStep(w) {
  if (!w.has_warc) return 'download';
  if (!w.has_markdown) return 'markdown';
  if (!w.has_fts) return 'index';
  return '';
}

function warcDoneCount(w) {
  return (w.has_warc ? 1 : 0) + (w.has_markdown ? 1 : 0) + (w.has_fts ? 1 : 0);
}

// ── Main entry ──
async function renderWARC(offset = state.warcOffset || 0, query = state.warcQuery || '') {
  state.currentPage = 'warc';
  state.warcOffset = offset;
  state.warcQuery = query;
  const phase = state.warcPhase || '';
  const pageSize = state.warcPageSize || 100;
  const main = $('main');
  main.innerHTML = `
    <div class="page-shell anim-fade-in">
      <div class="page-header mb-4">
        <h1 class="page-title">WARC Console</h1>
        <button onclick="renderWARC(0)" class="ui-btn px-3 py-2 text-xs font-mono">Reload</button>
      </div>
      <div id="warc-summary"></div>
      <div id="warc-tabs" class="mb-3"></div>
      <div class="flex flex-col sm:flex-row items-stretch sm:items-center gap-2 mb-3">
        <input id="warc-q" type="text" value="${esc(query)}" placeholder="Filter by index or filename\u2026"
          class="ui-input flex-1 text-sm px-3 py-2"
          onkeydown="if(event.key==='Enter'){state.warcOffset=0;renderWARC(0,this.value)}">
        <select id="warc-page-size" class="ui-select text-xs px-2 py-1.5 w-auto" onchange="state.warcPageSize=+this.value;state.warcOffset=0;renderWARC(0,state.warcQuery)">
          ${[50,100,200,500].map(n => `<option value="${n}"${n === pageSize ? ' selected' : ''}>${n} / page</option>`).join('')}
        </select>
      </div>
      <div id="warc-content"><div class="ui-empty">loading\u2026</div></div>
    </div>`;

  try {
    // Fetch WARC list and ensure central state is available
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
    renderWARCSummary(data.summary);
    renderWARCTabs(data.summary, data.total);
    renderWARCTable(data);
  } catch (e) {
    $('warc-content').innerHTML = `<div class="text-xs text-red-400">${esc(e.message)}</div>`;
  }
}

// ── Summary: global stats + 3-phase pipeline ──
function renderWARCSummary(summary) {
  const el = $('warc-summary');
  if (!el || !summary) return;
  const t = summary.total || 0;
  const dl = summary.downloaded || 0;
  const md = summary.markdown_ready || 0;
  const ix = summary.indexed || 0;
  const totalBytes = (summary.warc_bytes || 0) + (summary.markdown_bytes || 0) + (summary.pack_bytes || 0) + (summary.fts_bytes || 0);

  // Get manifest total from central state for accurate total
  const ov = state.central.overview || {};
  const mf = ov.manifest || {};
  const manifestTotal = mf.total_warcs || t;
  const hasManifest = manifestTotal > 0 && manifestTotal !== t;

  if (t <= 0 && manifestTotal <= 0) { el.innerHTML = ''; return; }

  function pct(n) { return t > 0 ? Math.round((n / t) * 100) : 0; }

  const stages = [
    { label: 'Downloaded', count: dl, cls: 'ov-c1', bytes: summary.warc_bytes || 0, byteLabel: '.warc.gz' },
    { label: 'Markdown', count: md, cls: 'ov-c2', bytes: summary.markdown_bytes || 0, byteLabel: '.md.warc.gz' },
    { label: 'Indexed', count: ix, cls: 'ov-c4', bytes: summary.fts_bytes || 0, byteLabel: 'index' },
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

  // Full crawl estimation
  const dlStage = ov.downloaded || {};
  const mdStage = ov.markdown || {};
  const ixStage = ov.indexed || {};
  const avgWARC = dlStage.avg_warc_bytes || 0;
  const avgDocs = mdStage.avg_docs_per_warc || 0;
  const estHTML = (manifestTotal > 0 && dl > 0 && avgWARC > 0) ? `
    <details class="surface mb-4">
      <summary class="p-3 text-[11px] font-mono ui-subtle cursor-pointer select-none">Full Crawl Estimate (${manifestTotal.toLocaleString()} WARCs)</summary>
      <div class="px-3 pb-3 grid grid-cols-2 sm:grid-cols-4 gap-3">
        <div>
          <div class="text-[10px] font-mono ui-subtle">Est. Download</div>
          <div class="text-xs font-mono font-medium">${fmtBytes(manifestTotal * avgWARC)}</div>
          <div class="text-[9px] font-mono ui-subtle">${fmtBytes(avgWARC)}/WARC avg</div>
        </div>
        ${avgDocs > 0 ? `<div>
          <div class="text-[10px] font-mono ui-subtle">Est. Documents</div>
          <div class="text-xs font-mono font-medium">${fmtNum(manifestTotal * avgDocs)}</div>
          <div class="text-[9px] font-mono ui-subtle">${fmtNum(avgDocs)}/WARC avg</div>
        </div>` : ''}
        ${ixStage.count > 0 ? `<div>
          <div class="text-[10px] font-mono ui-subtle">Est. Index Size</div>
          <div class="text-xs font-mono font-medium">${fmtBytes(Math.round(((ixStage.dahlia_bytes || 0) + (ixStage.tantivy_bytes || 0)) / ixStage.count * manifestTotal))}</div>
        </div>` : ''}
        <div>
          <div class="text-[10px] font-mono ui-subtle">Est. Total Disk</div>
          <div class="text-xs font-mono font-medium">${fmtBytes(ov.storage && ov.storage.projected_full_bytes || manifestTotal * avgWARC)}</div>
        </div>
      </div>
    </details>` : '';

  el.innerHTML = `
    ${activeJobsHTML}
    <div class="grid grid-cols-2 sm:grid-cols-${hasManifest ? '5' : '4'} gap-px border border-[var(--border)] mb-4" style="background:var(--border)">
      ${hasManifest ? `<div class="bg-[var(--panel)] px-3 py-2.5">
        <div class="text-[10px] font-mono ui-subtle uppercase tracking-wider">Manifest</div>
        <div class="text-base font-semibold font-mono">${manifestTotal.toLocaleString()}</div>
      </div>` : ''}
      <div class="bg-[var(--panel)] px-3 py-2.5">
        <div class="text-[10px] font-mono ui-subtle uppercase tracking-wider">${hasManifest ? 'Known' : 'Total'} WARCs</div>
        <div class="text-base font-semibold font-mono">${t.toLocaleString()}</div>
      </div>
      <div class="bg-[var(--panel)] px-3 py-2.5">
        <div class="text-[10px] font-mono ui-subtle uppercase tracking-wider">Fully Indexed</div>
        <div class="text-base font-semibold font-mono">${ix.toLocaleString()} <span class="text-xs ui-subtle font-normal">/ ${t.toLocaleString()}</span></div>
        <div class="progress-track mt-1" style="height:4px"><div class="ov-c4" style="height:100%;width:${pct(ix)}%"></div></div>
      </div>
      <div class="bg-[var(--panel)] px-3 py-2.5">
        <div class="text-[10px] font-mono ui-subtle uppercase tracking-wider">Remaining</div>
        <div class="text-base font-semibold font-mono">${(t - ix).toLocaleString()}</div>
      </div>
      <div class="bg-[var(--panel)] px-3 py-2.5">
        <div class="text-[10px] font-mono ui-subtle uppercase tracking-wider">Disk Usage</div>
        <div class="text-base font-semibold font-mono">${fmtBytes(totalBytes)}</div>
      </div>
    </div>

    <div class="surface p-4 mb-4">
      <div class="flex items-stretch gap-0">
        ${stages.map((s, i) => `
          ${i > 0 ? '<div class="ov-pipeline-arrow">\u2192</div>' : ''}
          <div class="ov-pipeline-step">
            <div class="flex items-baseline justify-between mb-1">
              <span class="text-[11px] font-mono ui-subtle">${esc(s.label)}</span>
              <span class="text-[11px] font-mono ${pct(s.count) === 100 ? 'status-completed' : pct(s.count) > 0 ? '' : 'ui-subtle'}">${s.count.toLocaleString()} / ${t.toLocaleString()}</span>
            </div>
            <div class="progress-track" style="height:6px">
              <div class="${s.cls}" style="height:100%;width:${pct(s.count)}%;transition:width 0.4s ease"></div>
            </div>
            <div class="flex items-center justify-between mt-1">
              <span class="text-[10px] font-mono ui-subtle">${pct(s.count)}%</span>
              ${s.bytes > 0 ? `<span class="text-[10px] font-mono ui-subtle">${fmtBytes(s.bytes)}</span>` : ''}
            </div>
          </div>`).join('')}
      </div>
    </div>
    ${estHTML}`;
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
  const limit = data.limit || state.warcPageSize || 100;
  const summary = data.summary || state.warcSummary || {};
  const globalTotal = summary.total || 0;

  const incomplete = warcs.filter(w => warcNextStep(w) !== '');

  const rows = warcs.map((w, i) => {
    const next = warcNextStep(w);
    const done = warcDoneCount(w);
    const docsStr = (w.warc_md_docs || 0) > 0 ? (w.warc_md_docs).toLocaleString() : '\u2014';
    const sizeStr = w.total_bytes > 0 ? fmtBytes(w.total_bytes) : '\u2014';

    // 3-segment stacked bar
    const stackedBar = `
      <div class="ov-stacked" style="width:42px;height:6px">
        <div class="ov-stacked-seg ${w.has_warc ? 'ov-c1' : ''}" style="width:33.3%;${!w.has_warc ? 'background:transparent' : ''}"></div>
        <div class="ov-stacked-seg ${w.has_markdown ? 'ov-c2' : ''}" style="width:33.4%;${!w.has_markdown ? 'background:transparent' : ''}"></div>
        <div class="ov-stacked-seg ${w.has_fts ? 'ov-c4' : ''}" style="width:33.3%;${!w.has_fts ? 'background:transparent' : ''}"></div>
      </div>`;

    const isRunning = warcRunning.has(w.index);
    const nextBtnCls = next ? (isRunning ? 'ui-btn' : 'ui-btn-primary') : 'ui-btn';
    const nextAction = warcActionCall(w.index, next);
    const btnId = warcBtnId(w.index);
    const btnLabel = next === 'download' ? 'download' : next === 'markdown' ? 'markdown' : next === 'index' ? 'index' : '';

    return `
    <tr class="anim-fade-up" style="animation-delay:${Math.min(i, 20)*12}ms">
      <td class="px-3 py-2 font-mono text-xs"><a href="#/warc/${encodeURIComponent(w.index)}" class="ui-link hover:text-[var(--accent)]">${esc(w.index)}</a></td>
      <td class="px-3 py-2 text-xs">
        <div class="flex items-center gap-2">
          ${stackedBar}
          <span class="text-[10px] font-mono ui-subtle">${done}/3</span>
        </div>
      </td>
      <td class="px-3 py-2 text-right text-xs font-mono ui-subtle hidden sm:table-cell">${docsStr}</td>
      <td class="px-3 py-2 text-right text-xs font-mono ui-subtle whitespace-nowrap hidden md:table-cell">${sizeStr}</td>
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
    ${isDashboard && incomplete.length > 0 ? `
      <div class="flex items-center gap-2 mb-3">
        <span class="text-[11px] font-mono ui-subtle">${incomplete.length} incomplete on this page</span>
        <button id="warc-batch-btn" onclick="warcBatchNext()" class="ui-btn px-3 py-1 text-[11px] font-mono">Run next step (${incomplete.length})</button>
      </div>` : ''}
    <div class="surface overflow-x-auto">
      <table class="w-full text-sm ui-table">
        <thead>
          <tr>
            <th class="text-left px-3 py-2 text-[11px] font-mono">Index</th>
            <th class="text-left px-3 py-2 text-[11px] font-mono">Pipeline</th>
            <th class="text-right px-3 py-2 text-[11px] font-mono hidden sm:table-cell">Docs</th>
            <th class="text-right px-3 py-2 text-[11px] font-mono hidden md:table-cell">Size</th>
            <th class="text-right px-3 py-2 text-[11px] font-mono">Action</th>
          </tr>
        </thead>
        <tbody>
          ${rows || `<tr><td colspan="5" class="px-3 py-4 text-xs font-mono ui-subtle">No WARC records</td></tr>`}
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

// ── Action helpers (3-phase) ──
function warcActionCall(index, step) {
  switch (step) {
    case 'download': return `warcAction('${esc(index)}','download')`;
    case 'markdown': return `warcAction('${esc(index)}','markdown')`;
    case 'index': return `warcAction('${esc(index)}','index',{engine:currentSearchEngine(),source:'files'})`;
    default: return '';
  }
}

function warcActionLabel(step) {
  switch (step) {
    case 'download': return 'Download WARC';
    case 'markdown': return 'Extract Markdown';
    case 'index': return `Build Index (${currentSearchEngine()})`;
    default: return '';
  }
}

// Batch
async function warcBatchNext() {
  const warcs = state.warcRows || [];
  const incomplete = warcs.filter(w => warcNextStep(w) !== '');
  if (incomplete.length === 0) return;
  if (!confirm(`Run the next pipeline step on ${incomplete.length} WARC${incomplete.length !== 1 ? 's' : ''}?`)) return;

  const batchBtn = $('warc-batch-btn');
  if (batchBtn) { batchBtn.disabled = true; batchBtn.textContent = 'running\u2026'; }

  for (const w of incomplete) {
    const next = warcNextStep(w);
    if (next === 'download') warcAction(w.index, 'download');
    else if (next === 'markdown') warcAction(w.index, 'markdown');
    else if (next === 'index') warcAction(w.index, 'index', { engine: currentSearchEngine(), source: 'files' });
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

  // Sizes
  const packTotal = Object.values(w.pack_bytes || {}).reduce((a,b)=>a+b,0);
  const ftsTotal = Object.values(w.fts_bytes || {}).reduce((a,b)=>a+b,0);
  const diskPhases = [
    { label: 'warc', value: w.warc_bytes || 0, cls: 'ov-c1' },
    { label: 'markdown', value: w.warc_md_bytes || 0, cls: 'ov-c2' },
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

  // 3-phase steps
  const stepsAll = [
    { key: 'download', label: 'Download', done: !!w.has_warc, cls: 'ov-c1' },
    { key: 'markdown', label: 'Markdown', done: !!w.has_markdown, cls: 'ov-c2' },
    { key: 'index', label: 'Index', done: !!w.has_fts, cls: 'ov-c4' },
  ];
  const nextStep = stepsAll.find(s => !s.done);
  const done = stepsAll.filter(s => s.done).length;

  const enginesOpts = (state.engines||[]).map(e=>`<option value="${esc(e)}">${esc(e)}</option>`).join('') ||
    `<option value="${DEFAULT_ENGINE}">${DEFAULT_ENGINE}</option>`;

  const relatedJobs = data.jobs || [];
  const activeJobs = relatedJobs.filter(j => j.status === 'running' || j.status === 'queued');

  const compressionRatio = (w.warc_bytes > 0 && (w.warc_md_bytes || 0) > 0) ? ((w.warc_md_bytes / w.warc_bytes) * 100).toFixed(1) + '%' : '\u2014';
  const docsCount = w.warc_md_docs || 0;
  const sys = data.system || {};

  // Step timeline (3 steps)
  const timelineHTML = `
    <div class="surface p-4 mb-4">
      <div class="text-[11px] font-mono ui-subtle mb-3">Pipeline ${done} / 3</div>
      <div class="flex items-center gap-0">
        ${stepsAll.map((s, i) => {
          const isNext = nextStep && nextStep.key === s.key;
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

  // Primary action
  let primaryActionHTML = '';
  if (nextStep) {
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
            <button onclick="${warcActionCall(w.index, nextStep.key).replace(/\)$/, ',{},true)')}" class="ui-btn ui-btn-primary px-4 py-2 text-xs font-mono">${warcActionLabel(nextStep.key)}</button>
            ${remaining > 1 ? `<button onclick="warcRunAll('${esc(w.index)}')" class="ui-btn px-3 py-2 text-xs font-mono" title="Run all remaining">Run All</button>` : ''}
          </div>
        </div>
      </div>`;
  } else {
    primaryActionHTML = `
      <div class="surface p-4 mb-4">
        <span class="text-sm font-mono status-completed">\u2713 All pipeline steps complete</span>
      </div>`;
  }

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

  $('warc-detail-content').innerHTML = `
    <div class="page-header mb-3">
      <h1 class="page-title">WARC ${esc(w.index || index)}</h1>
      <button onclick="renderWARCDetail('${esc(w.index || index)}')" class="ui-btn px-3 py-2 text-xs font-mono">Reload</button>
    </div>
    <div id="warc-action-msg" class="meta-line mb-3">${esc(warcActionMessage)}</div>

    ${activeJobsHTML}
    ${timelineHTML}
    ${primaryActionHTML}

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
            <span class="text-xs font-mono font-medium">${docsCount.toLocaleString()}</span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-[10px] font-mono ui-subtle">Total on Disk</span>
            <span class="text-xs font-mono font-medium">${fmtBytes(w.total_bytes || 0)}</span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-[10px] font-mono ui-subtle">WARC Size</span>
            <span class="text-xs font-mono">${fmtBytes(w.warc_bytes || 0)}</span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-[10px] font-mono ui-subtle">Markdown Size</span>
            <span class="text-xs font-mono">${fmtBytes(w.warc_md_bytes || 0)}</span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-[10px] font-mono ui-subtle">MD / WARC Ratio</span>
            <span class="text-xs font-mono">${compressionRatio}</span>
          </div>
          ${packTotal > 0 ? `<div class="flex items-center justify-between"><span class="text-[10px] font-mono ui-subtle">Pack Size</span><span class="text-xs font-mono">${fmtBytes(packTotal)}</span></div>` : ''}
          ${ftsTotal > 0 ? `<div class="flex items-center justify-between"><span class="text-[10px] font-mono ui-subtle">Index Size</span><span class="text-xs font-mono">${fmtBytes(ftsTotal)}</span></div>` : ''}
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

    <!-- Advanced actions -->
    <details class="surface mb-4">
      <summary class="p-4 text-[11px] font-mono ui-subtle cursor-pointer select-none">Advanced Actions</summary>
      <div class="px-4 pb-4">
        <div class="grid md:grid-cols-2 gap-4">
          <div class="space-y-2">
            <div class="text-[10px] font-mono ui-subtle mb-1 uppercase tracking-wider">Pipeline Steps</div>
            <button onclick="warcAction('${esc(w.index)}','download',{},true)" class="ui-btn w-full px-3 py-2 text-xs font-mono">Download WARC</button>
            <button onclick="warcAction('${esc(w.index)}','markdown',{fast:false},true)" class="ui-btn w-full px-3 py-2 text-xs font-mono">Extract Markdown</button>
            <button onclick="warcAction('${esc(w.index)}','markdown',{fast:true},true)" class="ui-btn w-full px-3 py-2 text-xs font-mono">Extract Markdown (fast)</button>
            <div class="flex items-center gap-2">
              <select id="warc-pack-format" class="ui-select flex-1 text-xs px-2 py-2">
                <option value="parquet">parquet</option><option value="bin">bin</option><option value="duckdb">duckdb</option><option value="markdown">markdown</option>
              </select>
              <button onclick="warcAction('${esc(w.index)}','pack',{format:$('warc-pack-format').value},true)" class="ui-btn px-3 py-2 text-xs font-mono shrink-0">Pack</button>
            </div>
          </div>
          <div class="space-y-2">
            <div class="text-[10px] font-mono ui-subtle mb-1 uppercase tracking-wider">Index</div>
            <div class="flex items-center gap-2">
              <select id="warc-index-engine" class="ui-select flex-1 text-xs px-2 py-2">${enginesOpts}</select>
              <select id="warc-index-source" class="ui-select text-xs px-2 py-2">
                <option value="files">files</option><option value="parquet">parquet</option><option value="bin">bin</option><option value="duckdb">duckdb</option><option value="markdown">markdown</option>
              </select>
              <button onclick="warcAction('${esc(w.index)}','index',{engine:$('warc-index-engine').value,source:$('warc-index-source').value},true)" class="ui-btn px-3 py-2 text-xs font-mono shrink-0">Index</button>
            </div>
            <button onclick="warcAction('${esc(w.index)}','reindex',{engine:$('warc-index-engine').value,source:$('warc-index-source').value},true)" class="ui-btn w-full px-3 py-2 text-xs font-mono">Re-index</button>
            <div class="pt-2 mt-2 border-t">
              <div class="text-[10px] font-mono ui-subtle mb-1 uppercase tracking-wider">Danger Zone</div>
              <div class="flex items-center gap-2">
                <select id="warc-delete-target" class="ui-select flex-1 text-xs px-2 py-2">
                  <option value="index">index</option><option value="pack">pack</option><option value="markdown">markdown</option><option value="warc">warc</option><option value="all">all</option>
                </select>
                <button onclick="if(confirm('Delete '+$('warc-delete-target').value+' for WARC ${esc(w.index)}?'))warcAction('${esc(w.index)}','delete',{target:$('warc-delete-target').value,format:($('warc-pack-format')||{}).value||'parquet',engine:($('warc-index-engine')||{}).value||currentSearchEngine()},true)" class="ui-btn ui-btn-danger px-3 py-2 text-xs font-mono shrink-0">Delete</button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </details>

    <!-- System -->
    ${(sys.disk_total || sys.mem_alloc) ? `
    <details class="surface mb-4">
      <summary class="p-4 text-[11px] font-mono ui-subtle cursor-pointer select-none">System</summary>
      <div class="px-4 pb-4 grid grid-cols-2 sm:grid-cols-4 gap-3">
        ${sys.disk_total ? `<div><div class="text-[10px] font-mono ui-subtle">Disk</div><div class="text-xs font-mono">${fmtBytes(sys.disk_used || 0)} / ${fmtBytes(sys.disk_total)}</div><div class="progress-track mt-1" style="height:4px"><div class="ov-c5" style="height:100%;width:${sys.disk_total > 0 ? Math.round(((sys.disk_used||0)/sys.disk_total)*100) : 0}%;opacity:0.6"></div></div></div>` : ''}
        ${sys.mem_alloc ? `<div><div class="text-[10px] font-mono ui-subtle">Heap</div><div class="text-xs font-mono">${fmtBytes(sys.mem_alloc)}</div></div>` : ''}
        ${sys.mem_stack_inuse ? `<div><div class="text-[10px] font-mono ui-subtle">Stack</div><div class="text-xs font-mono">${fmtBytes(sys.mem_stack_inuse)}</div></div>` : ''}
        ${sys.goroutines ? `<div><div class="text-[10px] font-mono ui-subtle">Goroutines</div><div class="text-xs font-mono">${sys.goroutines.toLocaleString()}</div></div>` : ''}
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

// Run all remaining steps sequentially (3-phase)
async function warcRunAll(index) {
  const data = state.warcDetail;
  if (!data) return;
  const w = data.warc || {};
  const steps = [];
  if (!w.has_warc) steps.push({ action: 'download', extra: {} });
  if (!w.has_markdown) steps.push({ action: 'markdown', extra: { fast: false } });
  if (!w.has_fts) steps.push({ action: 'index', extra: { engine: currentSearchEngine(), source: 'files' } });
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
