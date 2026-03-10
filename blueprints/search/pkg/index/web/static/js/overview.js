// ===================================================================
// Tab 1: Overview
// ===================================================================
function updateLiveIndicator() {
  const el = $('overview-live');
  if (!el) return;
  const connected = wsClient && wsClient.connected;
  if (connected) {
    el.className = 'text-[10px] font-mono live-blink';
    el.style.color = '#22c55e';
  } else {
    el.className = 'text-[10px] font-mono ui-subtle';
    el.style.color = '';
  }
}

async function renderOverview() {
  state.currentPage = 'overview';
  const main = $('main');
  const shell = `
    <div class="page-shell anim-fade-in">
      <div class="page-header">
        <h1 class="page-title">Overview</h1>
        <span id="overview-live" class="text-[10px] font-mono ui-subtle">● live</span>
      </div>
      <div id="overview-content">
        <div class="ui-empty">loading...</div>
      </div>
    </div>`;

  main.innerHTML = shell;
  updateLiveIndicator();
  // Poll live indicator every second (clears itself when element is gone).
  if (state._liveTimer) clearInterval(state._liveTimer);
  state._liveTimer = setInterval(updateLiveIndicator, 1000);
  // Always render immediately — cached data from localStorage means no "loading..." flash.
  // Falls back to zeros/empty if first ever visit.
  renderOverviewContent(state.central.overview || {}, state.central.jobs || []);
  // Subscribe to WS job updates so the page auto-refreshes when jobs complete.
  ensureJobStreamSubscribed();
  // Refresh in background; update when fresh data arrives.
  refreshCentralState().then(() => {
    if (state.currentPage === 'overview') renderOverviewContent(state.central.overview, state.central.jobs);
  }).catch(() => {});
}

function renderOverviewContent(d, jobs) {
  const el = $('overview-content');
  if (!el) return;

  const sys = d.system || {};
  const sto = d.storage || {};
  const mf = d.manifest || {};
  const dl = d.downloaded || {};
  const md = d.markdown || {};
  const ix = d.indexed || {};
  const activeJobs = (jobs || []).filter(j => j.status === 'running' || j.status === 'queued');
  const exactURLReady = (mf.real_total_urls || 0) > 0;
  const expectedWARCSize = mf.est_total_size_bytes || 0;
  const expectedWARCSizeReady = expectedWARCSize > 0;
  const parquetIndexSizeReady = (mf.real_total_size_bytes || 0) > 0;
  const metaReadyFiles = mf.meta_ready_files || 0;
  const metaTotalFiles = mf.meta_total_files || 0;
  const expectedReadyPct = metaTotalFiles > 0 ? Math.round((metaReadyFiles / metaTotalFiles) * 100) : 0;
  const currentDownloadPct = expectedWARCSizeReady && expectedWARCSize > 0
    ? Math.round(((dl.size_bytes || 0) / expectedWARCSize) * 100)
    : 0;

  function pct(a, b) { return b > 0 ? Math.round((a / b) * 100) : 0; }
  function stageColor(done, total) {
    if (done === 0) return 'var(--border)';
    if (done >= total && total > 0) return 'var(--success)';
    return '#f59e0b';
  }

  // ── Row 1: Storage + System ──
  const storageHTML = `
    <div class="grid md:grid-cols-2 gap-4 mb-4">
      <div class="surface p-4">
        <div class="text-[11px] font-mono ui-subtle mb-3">Storage</div>
        <div class="grid grid-cols-2 gap-3 mb-3">
          <div>
            <div class="text-[10px] font-mono ui-subtle">Current Downloaded</div>
            <div class="text-sm font-mono font-medium">${fmtBytes(dl.size_bytes || 0)}</div>
          </div>
          <div>
            <div class="text-[10px] font-mono ui-subtle">Full WARC Expected</div>
            <div class="text-sm font-mono font-medium">${expectedWARCSizeReady ? fmtBytes(expectedWARCSize) : '<span class="ui-subtle">syncing...</span>'}</div>
          </div>
        </div>
        <div class="grid grid-cols-2 gap-3 mb-3">
          <div>
            <div class="text-[10px] font-mono ui-subtle">CC URLs Expected</div>
            <div class="text-xs font-mono">${exactURLReady ? fmtNum(mf.real_total_urls) : '<span class="ui-subtle">syncing...</span>'}</div>
          </div>
          <div>
            <div class="text-[10px] font-mono ui-subtle">Parquet Index Size</div>
            <div class="text-xs font-mono">${parquetIndexSizeReady ? fmtBytes(mf.real_total_size_bytes) : '<span class="ui-subtle">syncing...</span>'}</div>
          </div>
        </div>
        ${expectedWARCSizeReady ? `
          <div class="border-t pt-3 mb-3">
            <div class="flex items-center justify-between mb-1">
              <span class="text-[10px] font-mono ui-subtle">WARC Download Progress</span>
              <span class="text-[10px] font-mono ui-subtle">${currentDownloadPct}% (${fmtBytes(dl.size_bytes || 0)} / ${fmtBytes(expectedWARCSize)})</span>
            </div>
            <div class="progress-track" style="height:6px">
              <div class="ov-c2" style="height:100%;width:${currentDownloadPct}%;opacity:0.7"></div>
            </div>
            <div class="text-[10px] font-mono ui-subtle mt-1">${fmtBytes(Math.max(0, expectedWARCSize - (dl.size_bytes || 0)))} remaining</div>
          </div>` : `
          <div class="border-t pt-3 mb-3">
            <div class="flex items-center justify-between mb-1">
              <span class="text-[10px] font-mono ui-subtle">Preparing CC metadata</span>
              <span class="text-[10px] font-mono ui-subtle">${metaReadyFiles}/${metaTotalFiles || 0} parquet files</span>
            </div>
            <div class="progress-track" style="height:6px">
              <div class="ov-c1" style="height:100%;width:${expectedReadyPct}%;opacity:0.7"></div>
            </div>
          </div>`}
        ${sto.disk_total ? `
          <div class="border-t pt-3">
            <div class="flex items-center justify-between mb-1">
              <span class="text-[10px] font-mono ui-subtle">Disk</span>
              <span class="text-[10px] font-mono ui-subtle">${fmtBytes(sto.disk_used || 0)} / ${fmtBytes(sto.disk_total)}</span>
            </div>
            <div class="progress-track" style="height:6px">
              <div class="ov-c5" style="height:100%;width:${pct(sto.disk_used || 0, sto.disk_total)}%;opacity:0.6"></div>
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
    { label: 'Indexed', done: ix.count || 0, total: dl.count || 0, cls: 'ov-c4' },
  ];
  const pipelineHTML = `
    <div class="surface p-4 mb-4" style="animation-delay:50ms">
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
    <div class="grid md:grid-cols-2 gap-4 mb-4" style="animation-delay:100ms">
      <div class="surface p-4" style="border-left:3px solid ${stageColor(mf.total_warcs, mf.total_warcs)}">
        <div class="text-[11px] font-mono ui-subtle mb-2">Stage 1: Manifest (Source)</div>
        <div class="grid grid-cols-3 gap-3">
          <div>
            <div class="text-[10px] font-mono ui-subtle">Total WARCs</div>
            <div class="text-sm font-mono font-medium">${fmtNum(mf.total_warcs || 0)}</div>
          </div>
          <div>
            <div class="text-[10px] font-mono ui-subtle">Full Expected URLs</div>
            <div class="text-sm font-mono font-medium">${exactURLReady ? fmtNum(mf.real_total_urls) : '<span class="ui-subtle">syncing...</span>'}</div>
          </div>
          <div>
            <div class="text-[10px] font-mono ui-subtle">Full Expected WARC Size</div>
            <div class="text-sm font-mono font-medium">${expectedWARCSizeReady ? fmtBytes(expectedWARCSize) : '<span class="ui-subtle">syncing...</span>'}</div>
          </div>
        </div>
        <div class="text-[10px] font-mono ui-subtle mt-2">
          URLs are exact from CC parquet metadata; WARC total size is expected from current average WARC bytes.
          ${!exactURLReady ? ` Progress: ${metaReadyFiles}/${metaTotalFiles || 0}.` : ''}
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

  // ── Row 4: Markdown + Index ──
  const row4HTML = `
    <div class="grid md:grid-cols-2 gap-4 mb-4" style="animation-delay:150ms">
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
      <div class="surface p-4" style="border-left:3px solid ${stageColor(ix.count, dl.count)}">
        <div class="flex items-center justify-between mb-2">
          <div class="text-[11px] font-mono ui-subtle">Stage 4: Index</div>
          <div class="text-[10px] font-mono ${ix.count > 0 ? '' : 'ui-subtle'}">${ix.count || 0} / ${dl.count || 0}</div>
        </div>
        <div class="progress-track mb-3" style="height:4px">
          <div class="ov-c4" style="height:100%;width:${pct(ix.count, dl.count)}%;transition:width 0.4s ease"></div>
        </div>
        <div class="space-y-1">
          <div class="flex justify-between text-[10px] font-mono"><span class="ui-subtle">Dahlia</span><span>${fmtBytes(ix.dahlia_bytes || 0)} (${ix.dahlia_shards || 0} sh)</span></div>
          <div class="flex justify-between text-[10px] font-mono"><span class="ui-subtle">Tantivy</span><span>${fmtBytes(ix.tantivy_bytes || 0)} (${ix.tantivy_shards || 0} sh)</span></div>
        </div>
      </div>
    </div>`;

  // ── Active Jobs ──
  const jobsHTML = activeJobs.length > 0 ? `
    <div class="surface p-4 mb-4" style="animation-delay:200ms">
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

  // ── Assemble: Pipeline → Stages → Jobs → Storage/System ──
  el.innerHTML = `
    ${pipelineHTML}
    ${row3HTML}
    ${row4HTML}
    ${jobsHTML}
    ${storageHTML}`;
}
