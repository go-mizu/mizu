// ===================================================================
// Tab 1: Overview
// ===================================================================
async function renderOverview() {
  state.currentPage = 'overview';
  const main = $('main');
  main.innerHTML = `
    <div class="page-shell anim-fade-in">
      <div class="page-header">
        <h1 class="page-title">Overview</h1>
        <button onclick="refreshOverviewMeta()" class="ui-btn px-3 py-2 text-xs font-mono">
          Refresh Metadata
        </button>
      </div>
      <div id="overview-refresh-msg" class="meta-line mb-4"></div>
      <div id="overview-content">
        <div class="ui-empty">loading...</div>
      </div>
    </div>`;

  try {
    const [jobsData] = await Promise.all([
      apiJobs().catch(() => ({ jobs: [] })),
    ]);
    state.overview = await apiOverview();
    state.overviewLoadedAt = Date.now();
    state.jobs = (jobsData && jobsData.jobs) || [];
    renderOverviewContent(state.overview, state.jobs);
  } catch (e) {
    $('overview-content').innerHTML = `<div class="text-xs text-red-400">${esc(e.message)}</div>`;
  }
}

function renderOverviewContent(d, jobs) {
  const el = $('overview-content');
  if (!el) return;

  const sys = d.system || {};
  const sto = d.storage || {};
  const mf = d.manifest || {};
  const dl = d.downloaded || {};
  const md = d.markdown || {};
  const pk = d.pack || {};
  const ix = d.indexed || {};
  const activeJobs = (jobs || []).filter(j => j.status === 'running' || j.status === 'queued');

  function pct(a, b) { return b > 0 ? Math.round((a / b) * 100) : 0; }
  function stageColor(done, total) {
    if (done === 0) return 'var(--border)';
    if (done >= total && total > 0) return 'var(--success)';
    return '#f59e0b';
  }

  // ── Row 1: Storage + System ──
  const storageHTML = `
    <div class="grid md:grid-cols-2 gap-4 mb-4 anim-fade-up">
      <div class="surface p-4">
        <div class="text-[11px] font-mono ui-subtle mb-3">Storage</div>
        <div class="grid grid-cols-2 gap-3 mb-3">
          <div>
            <div class="text-[10px] font-mono ui-subtle">Crawl Data</div>
            <div class="text-sm font-mono font-medium">${fmtBytes(sto.crawl_bytes || 0)}</div>
          </div>
          <div>
            <div class="text-[10px] font-mono ui-subtle">Projected Full</div>
            <div class="text-sm font-mono font-medium">${fmtBytes(sto.projected_full_bytes || 0)}</div>
          </div>
        </div>
        ${sto.disk_total ? `
          <div class="border-t pt-3">
            <div class="flex items-center justify-between mb-1">
              <span class="text-[10px] font-mono ui-subtle">Disk</span>
              <span class="text-[10px] font-mono ui-subtle">${fmtBytes(sto.disk_used || 0)} / ${fmtBytes(sto.disk_total)}</span>
            </div>
            <div class="progress-track" style="height:6px">
              <div class="ov-c5" style="height:100%;width:${pct(sto.disk_used || 0, sto.disk_total)};opacity:0.6"></div>
            </div>
            <div class="text-[10px] font-mono ui-subtle mt-1">${fmtBytes(sto.disk_free || 0)} free</div>
          </div>` : ''}
      </div>
      <div class="surface p-4">
        <div class="text-[11px] font-mono ui-subtle mb-3">System</div>
        <div class="grid grid-cols-2 gap-x-4 gap-y-2">
          <div>
            <div class="text-[10px] font-mono ui-subtle">Heap Alloc</div>
            <div class="text-xs font-mono">${fmtBytes(sys.heap_alloc || 0)}</div>
          </div>
          <div>
            <div class="text-[10px] font-mono ui-subtle">Heap Sys</div>
            <div class="text-xs font-mono">${fmtBytes(sys.heap_sys || 0)}</div>
          </div>
          <div>
            <div class="text-[10px] font-mono ui-subtle">Stack</div>
            <div class="text-xs font-mono">${fmtBytes(sys.stack_inuse || 0)}</div>
          </div>
          <div>
            <div class="text-[10px] font-mono ui-subtle">GC Cycles</div>
            <div class="text-xs font-mono">${(sys.num_gc || 0).toLocaleString()}</div>
          </div>
          <div>
            <div class="text-[10px] font-mono ui-subtle">Goroutines</div>
            <div class="text-xs font-mono">${(sys.goroutines || 0).toLocaleString()}</div>
          </div>
          <div>
            <div class="text-[10px] font-mono ui-subtle">GOMEMLIMIT</div>
            <div class="text-xs font-mono">${sys.gomemlimit > 0 ? fmtBytes(sys.gomemlimit) : '\u2014'}</div>
          </div>
          <div>
            <div class="text-[10px] font-mono ui-subtle">Go / PID</div>
            <div class="text-xs font-mono">${esc(sys.go_version || '')} / ${sys.pid || ''}</div>
          </div>
          <div>
            <div class="text-[10px] font-mono ui-subtle">Uptime</div>
            <div class="text-xs font-mono">${fmtDuration(sys.uptime_seconds || 0)}</div>
          </div>
        </div>
      </div>
    </div>`;

  // ── Row 2: Pipeline Summary Bar ──
  const stages = [
    { label: 'Manifest', done: mf.total_warcs || 0, total: mf.total_warcs || 0, cls: 'ov-c1' },
    { label: 'Downloaded', done: dl.count || 0, total: mf.total_warcs || 0, cls: 'ov-c2' },
    { label: 'Markdown', done: md.count || 0, total: dl.count || 0, cls: 'ov-c3' },
    { label: 'Pack', done: pk.count || 0, total: dl.count || 0, cls: 'ov-c4' },
    { label: 'FTS Index', done: ix.count || 0, total: dl.count || 0, cls: 'ov-c6' },
  ];
  const pipelineHTML = `
    <div class="surface p-4 mb-4 anim-fade-up" style="animation-delay:50ms">
      <div class="flex items-center justify-between mb-2">
        <div class="text-[11px] font-mono ui-subtle">Pipeline: ${esc(d.crawl_id || '')}</div>
        <div class="text-[10px] font-mono ui-subtle">${d.crawl_from ? formatCrawlDate(d.crawl_from) + ' \u2013 ' + formatCrawlDate(d.crawl_to) : ''}</div>
      </div>
      <div class="flex items-stretch gap-0">
        ${stages.map((s, i) => {
          const p = pct(s.done, s.total);
          return `
            ${i > 0 ? '<div class="ov-pipeline-arrow">\u2192</div>' : ''}
            <div class="ov-pipeline-step">
              <div class="flex items-baseline justify-between mb-1">
                <span class="text-[11px] font-mono ui-subtle">${esc(s.label)}</span>
                <span class="text-[11px] font-mono ${p === 100 ? 'status-completed' : p > 0 ? '' : 'ui-subtle'}">${s.done.toLocaleString()}</span>
              </div>
              <div class="progress-track" style="height:6px">
                <div class="${s.cls}" style="height:100%;width:${p}%;transition:width 0.4s ease"></div>
              </div>
            </div>`;
        }).join('')}
      </div>
    </div>`;

  // ── Row 3: Manifest + Downloaded ──
  const row3HTML = `
    <div class="grid md:grid-cols-2 gap-4 mb-4 anim-fade-up" style="animation-delay:100ms">
      <div class="surface p-4" style="border-left:3px solid ${stageColor(mf.total_warcs, mf.total_warcs)}">
        <div class="text-[11px] font-mono ui-subtle mb-2">Stage 1: Manifest (Source)</div>
        <div class="grid grid-cols-3 gap-3">
          <div>
            <div class="text-[10px] font-mono ui-subtle">Total WARCs</div>
            <div class="text-sm font-mono font-medium">${fmtNum(mf.total_warcs || 0)}</div>
          </div>
          <div>
            <div class="text-[10px] font-mono ui-subtle">Est. URLs</div>
            <div class="text-sm font-mono font-medium">${fmtNum(mf.est_total_urls || 0)}</div>
          </div>
          <div>
            <div class="text-[10px] font-mono ui-subtle">Est. Size</div>
            <div class="text-sm font-mono font-medium">${fmtBytes(mf.est_total_size_bytes || 0)}</div>
          </div>
        </div>
      </div>
      <div class="surface p-4" style="border-left:3px solid ${stageColor(dl.count, mf.total_warcs)}">
        <div class="flex items-center justify-between mb-2">
          <div class="text-[11px] font-mono ui-subtle">Stage 2: Downloaded</div>
          <div class="text-[10px] font-mono ${dl.count > 0 ? '' : 'ui-subtle'}">${fmtNum(dl.count || 0)} / ${fmtNum(mf.total_warcs || 0)}</div>
        </div>
        <div class="progress-track mb-3" style="height:4px">
          <div class="ov-c2" style="height:100%;width:${pct(dl.count, mf.total_warcs)}%;transition:width 0.4s ease"></div>
        </div>
        <div class="grid grid-cols-2 gap-3">
          <div>
            <div class="text-[10px] font-mono ui-subtle">Total Size</div>
            <div class="text-xs font-mono">${fmtBytes(dl.size_bytes || 0)}</div>
          </div>
          <div>
            <div class="text-[10px] font-mono ui-subtle">Avg / WARC</div>
            <div class="text-xs font-mono">${fmtBytes(dl.avg_warc_bytes || 0)}</div>
          </div>
        </div>
      </div>
    </div>`;

  // ── Row 4: Markdown, Pack, FTS Index ──
  const row4HTML = `
    <div class="grid md:grid-cols-3 gap-4 mb-4 anim-fade-up" style="animation-delay:150ms">
      <div class="surface p-4" style="border-left:3px solid ${stageColor(md.count, dl.count)}">
        <div class="flex items-center justify-between mb-2">
          <div class="text-[11px] font-mono ui-subtle">Stage 3: Markdown</div>
          <div class="text-[10px] font-mono ${md.count > 0 ? '' : 'ui-subtle'}">${md.count || 0} / ${dl.count || 0}</div>
        </div>
        <div class="progress-track mb-3" style="height:4px">
          <div class="ov-c3" style="height:100%;width:${pct(md.count, dl.count)}%;transition:width 0.4s ease"></div>
        </div>
        <div class="space-y-1">
          <div class="flex justify-between text-[10px] font-mono"><span class="ui-subtle">Docs</span><span>${fmtNum(md.total_docs || 0)}</span></div>
          <div class="flex justify-between text-[10px] font-mono"><span class="ui-subtle">Size</span><span>${fmtBytes(md.size_bytes || 0)}</span></div>
          <div class="flex justify-between text-[10px] font-mono"><span class="ui-subtle">Avg/WARC</span><span>${fmtNum(md.avg_docs_per_warc || 0)} docs</span></div>
          <div class="flex justify-between text-[10px] font-mono"><span class="ui-subtle">Avg doc</span><span>${fmtBytes(md.avg_doc_bytes || 0)}</span></div>
        </div>
      </div>
      <div class="surface p-4" style="border-left:3px solid ${stageColor(pk.count, dl.count)}">
        <div class="flex items-center justify-between mb-2">
          <div class="text-[11px] font-mono ui-subtle">Stage 4: Pack</div>
          <div class="text-[10px] font-mono ${pk.count > 0 ? '' : 'ui-subtle'}">${pk.count || 0} / ${dl.count || 0}</div>
        </div>
        <div class="progress-track mb-3" style="height:4px">
          <div class="ov-c4" style="height:100%;width:${pct(pk.count, dl.count)}%;transition:width 0.4s ease"></div>
        </div>
        <div class="space-y-1">
          <div class="flex justify-between text-[10px] font-mono"><span class="ui-subtle">Parquet</span><span>${fmtBytes(pk.parquet_bytes || 0)}</span></div>
          <div class="flex justify-between text-[10px] font-mono"><span class="ui-subtle">.md.warc.gz</span><span>${fmtBytes(pk.warc_md_bytes || 0)}</span></div>
        </div>
      </div>
      <div class="surface p-4" style="border-left:3px solid ${stageColor(ix.count, dl.count)}">
        <div class="flex items-center justify-between mb-2">
          <div class="text-[11px] font-mono ui-subtle">Stage 5: FTS Index</div>
          <div class="text-[10px] font-mono ${ix.count > 0 ? '' : 'ui-subtle'}">${ix.count || 0} / ${dl.count || 0}</div>
        </div>
        <div class="progress-track mb-3" style="height:4px">
          <div class="ov-c6" style="height:100%;width:${pct(ix.count, dl.count)}%;transition:width 0.4s ease"></div>
        </div>
        <div class="space-y-1">
          <div class="flex justify-between text-[10px] font-mono"><span class="ui-subtle">Dahlia</span><span>${fmtBytes(ix.dahlia_bytes || 0)} (${ix.dahlia_shards || 0} sh)</span></div>
          <div class="flex justify-between text-[10px] font-mono"><span class="ui-subtle">Tantivy</span><span>${fmtBytes(ix.tantivy_bytes || 0)} (${ix.tantivy_shards || 0} sh)</span></div>
        </div>
      </div>
    </div>`;

  // ── Active Jobs ──
  const jobsHTML = activeJobs.length > 0 ? `
    <div class="surface p-4 mb-4 anim-fade-up" style="animation-delay:200ms">
      <div class="flex items-center justify-between mb-3">
        <div class="text-[11px] font-mono ui-subtle">Active Jobs (${activeJobs.length})</div>
        <a href="#/jobs" class="text-[11px] font-mono ui-link">all jobs \u2192</a>
      </div>
      ${activeJobs.slice(0, 5).map(j => {
        const p = Math.round((j.progress || 0) * 100);
        const rateStr = j.rate > 0 ? ` \u00b7 ${j.rate.toFixed(0)}/s` : '';
        return `
          <div class="mb-2 last:mb-0">
            <div class="flex items-center justify-between mb-1">
              <span class="text-[11px] font-mono ui-subtle">${esc(j.id)} \u00b7 ${esc(j.type)} \u00b7 <span class="${statusClass(j.status)}">${esc(j.status)}</span>${rateStr}</span>
              <span class="text-[11px] font-mono">${p}%</span>
            </div>
            <div class="progress-track" style="height:4px">
              <div class="progress-fill" style="width:${p}%"></div>
            </div>
          </div>`;
      }).join('')}
    </div>` : '';

  // ── Assemble ──
  el.innerHTML = `
    ${storageHTML}
    ${pipelineHTML}
    ${row3HTML}
    ${row4HTML}
    ${jobsHTML}
    <div class="flex gap-3 mt-2 anim-fade-up" style="animation-delay:250ms">
      <a href="#/search" class="ui-btn px-4 py-2 text-sm">Search</a>
      <a href="#/warc" class="ui-btn px-4 py-2 text-sm">WARC Console</a>
      <a href="#/jobs" class="ui-btn px-4 py-2 text-sm">Jobs</a>
    </div>`;
}

async function refreshOverviewMeta() {
  const msg = $('overview-refresh-msg');
  if (msg) msg.textContent = 'requesting metadata refresh...';
  try {
    const crawl = state.overview && state.overview.crawl_id ? state.overview.crawl_id : '';
    const res = await apiMetaRefresh(crawl, true);
    const accepted = !!(res && res.accepted);
    if (msg) msg.textContent = accepted ? 'refresh started' : 'refresh already in progress';
    setTimeout(() => {
      refreshDashboardContext().catch(() => {});
      renderOverview();
    }, 350);
  } catch (e) {
    if (msg) msg.textContent = `refresh failed: ${e.message}`;
  }
}
