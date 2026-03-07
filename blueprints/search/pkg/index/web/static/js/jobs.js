// ===================================================================
// Job utilities (shared by Overview + Jobs)
// ===================================================================
let pipelinePollingTimer = null;
let jobsPollingTimer = null;

function ensureJobStreamSubscribed() {
  if (state.jobsStreamSubscribed) return;
  wsClient.subscribe('*', (msg) => onJobUpdate(msg));
  state.jobsStreamSubscribed = true;
}

async function reloadJobs() {
  const jobsData = await apiJobs().catch(() => ({ jobs: [] }));
  state.central.jobs = jobsData.jobs || [];
  return state.central.jobs;
}

// renderPipeline removed — pipeline is now integrated into overview page

function countActive(jobs) {
  return jobs.filter(j => j.status === 'running' || j.status === 'queued').length;
}

function renderJobList(jobs) {
  const active = jobs.filter(j => j.status === 'running' || j.status === 'queued');
  if (active.length === 0) return '';
  return active.map(j => renderJobItem(j)).join('');
}

function renderJobItem(j) {
  const pct = Math.round((j.progress || 0) * 100);
  const rateStr = j.rate > 0 ? ` &middot; ${j.rate.toFixed(0)}/s` : '';
  const elapsed = j.started_at ? fmtDuration(j.started_at, j.ended_at) : '';
  const startedStr = j.started_at ? fmtRelativeTime(j.started_at) : '';
  const cancelBtn = (j.status === 'running' || j.status === 'queued')
    ? `<button onclick="cancelJob('${esc(j.id)}')" class="text-xs ui-link ml-2">cancel</button>`
    : '';
  const cfg = j.config || {};
  const cfgParts = [];
  if (cfg.files && cfg.files !== '0') cfgParts.push(`files:${esc(cfg.files)}`);
  if (cfg.engine) cfgParts.push(`engine:${esc(cfg.engine)}`);
  if (cfg.format) cfgParts.push(`fmt:${esc(cfg.format)}`);
  if (cfg.fast) cfgParts.push('fast');
  const cfgStr = cfgParts.length > 0 ? ` &middot; ${cfgParts.join(' ')}` : '';
  return `
    <div class="mb-2 py-2 border-b" id="job-${esc(j.id)}">
      <div class="flex items-center justify-between mb-1">
        <span class="text-xs font-mono ui-subtle">${esc(j.id)} &middot; ${jobTypeBadge(j.type)} &middot; <span class="${statusClass(j.status)}">${esc(j.status)}</span>${rateStr}${cfgStr}</span>
        <span class="text-xs font-mono ui-subtle">${pct}%${elapsed ? ' &middot; ' + elapsed : ''}${cancelBtn}</span>
      </div>
      <div class="progress-track mb-1">
        <div class="progress-fill" style="width:${pct}%"></div>
      </div>
      <div class="flex items-center justify-between">
        <div class="text-xs ui-subtle truncate">${esc(j.message || '')}</div>
        ${startedStr ? `<div class="text-[10px] font-mono ui-subtle shrink-0 ml-2">${startedStr}</div>` : ''}
      </div>
    </div>`;
}

function renderJobHistory(jobs) {
  if (!jobs || jobs.length === 0) {
    return '<div class="ui-empty">No jobs yet.</div>';
  }

  return `
    <div class="surface overflow-x-auto">
      <table class="w-full text-sm ui-table">
        <thead>
          <tr>
            <th class="text-left px-4 py-2 text-xs font-mono">ID</th>
            <th class="text-left px-4 py-2 text-xs font-mono">Type</th>
            <th class="text-left px-4 py-2 text-xs font-mono hidden sm:table-cell">Started</th>
            <th class="text-left px-4 py-2 text-xs font-mono">Status</th>
            <th class="text-left px-4 py-2 text-xs font-mono">Message / Config</th>
            <th class="text-right px-4 py-2 text-xs font-mono">Duration</th>
          </tr>
        </thead>
        <tbody>
          ${jobs.map(j => {
            const cfg = j.config || {};
            const cfgParts = [];
            if (cfg.files && cfg.files !== '0') cfgParts.push(`files:${esc(cfg.files)}`);
            if (cfg.engine) cfgParts.push(`engine:${esc(cfg.engine)}`);
            if (cfg.format) cfgParts.push(`fmt:${esc(cfg.format)}`);
            const detail = [j.message || j.error || '', cfgParts.join(' ')].filter(Boolean).join(' · ');
            return `
            <tr class="cursor-pointer hover:bg-white/5" onclick="navigateTo('/jobs/${esc(j.id)}')">
              <td class="px-4 py-2 font-mono text-xs">${esc(j.id)}</td>
              <td class="px-4 py-2 text-xs">${jobTypeBadge(j.type)}</td>
              <td class="px-4 py-2 font-mono text-xs ui-subtle hidden sm:table-cell">${j.started_at ? fmtRelativeTime(j.started_at) : '—'}</td>
              <td class="px-4 py-2 text-xs ${statusClass(j.status)}">${esc(j.status)}</td>
              <td class="px-4 py-2 text-xs ui-subtle truncate max-w-xs">${detail}</td>
              <td class="px-4 py-2 font-mono text-xs text-right ui-subtle">${fmtDuration(j.started_at, j.ended_at)}</td>
            </tr>`;
          }).join('')}
        </tbody>
      </table>
    </div>`;
}

function renderRecentJobs(jobs) {
  if (!jobs || jobs.length === 0) {
    return '<div class="ui-empty">No jobs yet.</div>';
  }
  const recent = jobs.slice(0, 5);
  return `
    <div class="surface overflow-x-auto">
      <table class="w-full text-sm ui-table">
        <thead>
          <tr>
            <th class="text-left px-4 py-2 text-xs font-mono">ID</th>
            <th class="text-left px-4 py-2 text-xs font-mono">Type</th>
            <th class="text-left px-4 py-2 text-xs font-mono hidden sm:table-cell">Started</th>
            <th class="text-left px-4 py-2 text-xs font-mono">Status</th>
            <th class="text-left px-4 py-2 text-xs font-mono">Message</th>
            <th class="text-right px-4 py-2 text-xs font-mono">Duration</th>
          </tr>
        </thead>
        <tbody>
          ${recent.map(j => `
            <tr class="cursor-pointer hover:bg-white/5" onclick="navigateTo('/jobs/${esc(j.id)}')">
              <td class="px-4 py-2 font-mono text-xs">${esc(j.id)}</td>
              <td class="px-4 py-2 text-xs">${jobTypeBadge(j.type)}</td>
              <td class="px-4 py-2 font-mono text-xs ui-subtle hidden sm:table-cell">${j.started_at ? fmtRelativeTime(j.started_at) : '—'}</td>
              <td class="px-4 py-2 text-xs ${statusClass(j.status)}">${esc(j.status)}</td>
              <td class="px-4 py-2 text-xs ui-subtle truncate max-w-xs">${esc(j.message || j.error || '')}</td>
              <td class="px-4 py-2 font-mono text-xs text-right ui-subtle">${fmtDuration(j.started_at, j.ended_at)}</td>
            </tr>`).join('')}
        </tbody>
      </table>
    </div>`;
}

function submitJob(event, type) {
  event.preventDefault();
  const form = event.target;
  const files = form.elements.files ? form.elements.files.value : '0';

  const cfg = { type, files };

  if (type === 'pack') {
    cfg.format = form.elements.format ? form.elements.format.value : 'parquet';
  } else if (type === 'index') {
    cfg.engine = form.elements.engine ? form.elements.engine.value : DEFAULT_ENGINE;
    cfg.source = form.elements.source ? form.elements.source.value : 'files';
  }

  startJob(cfg);
  return false;
}

async function startJob(cfg) {
  try {
    const job = await apiCreateJob(cfg);
    // Add to central state
    if (!state.central.jobs) state.central.jobs = [];
    state.central.jobs.unshift(job);

    // Subscribe to updates
    wsClient.subscribe(job.id, (msg) => onJobUpdate(msg));

    // Update header badge
    updateHeaderStatus();

    // Re-render job views if visible.
    if (state.currentPage === 'jobs') {
      refreshJobsUI();
    }
  } catch (e) {
    alert('Failed to start job: ' + e.message);
  }
}

async function cancelJob(id) {
  try {
    await apiCancelJob(id);
    // Update central state
    const job = state.central.jobs && state.central.jobs.find(j => j.id === id);
    if (job) {
      job.status = 'cancelled';
      job.ended_at = new Date().toISOString();
    }
    updateHeaderStatus();
    if (state.currentPage === 'jobs') {
      refreshJobsUI();
    }
    if (state.currentPage === 'job-detail') {
      const el = $('job-detail-content');
      if (el) renderJobDetailContent(job || { id, status: 'cancelled' });
    }
  } catch (e) {
    alert('Failed to cancel job: ' + e.message);
  }
}

async function clearJobHistory() {
  if (!confirm('Clear all completed/failed/cancelled jobs?')) return;
  try {
    await apiClearJobs();
    await reloadJobs();
    if (state.currentPage === 'jobs') {
      refreshJobsUI();
    }
  } catch (e) {
    alert('Failed to clear: ' + e.message);
  }
}

function onJobUpdate(msg) {
  if (!state.central.jobs) return;
  const job = state.central.jobs.find(j => j.id === msg.job_id);
  if (!job) return;

  if (msg.type === 'job_progress') {
    job.progress = msg.progress;
    job.message = msg.message;
    job.rate = msg.rate || 0;

    if (state.currentPage === 'jobs') {
      updateJobInPlace(job);
    }
    if (state.currentPage === 'job-detail') {
      const el = $('job-detail-content');
      if (el) renderJobDetailContent(job);
    }
    // Live progress on Overview: re-render the active-jobs pane (pure in-memory, fast).
    if (state.currentPage === 'overview' && $('overview-content')) {
      renderOverviewContent(state.central.overview, state.central.jobs);
    }
    // Live progress on WARC detail: update the matching job entry and re-render.
    if (state.currentPage === 'warc' && state.warcDetail && $('warc-detail-content')) {
      const wdJob = (state.warcDetail.jobs || []).find(j => j.id === msg.job_id);
      if (wdJob) {
        wdJob.progress = msg.progress;
        wdJob.message = msg.message;
        renderWARCDetailContent(state.warcDetail, (state.warcDetail.warc || {}).index);
      }
    }
  } else if (msg.type === 'job_update') {
    job.status = msg.status;
    if (msg.error) job.error = msg.error;
    const isDone = msg.status === 'completed' || msg.status === 'failed' || msg.status === 'cancelled';
    if (isDone) {
      job.ended_at = new Date().toISOString();
      if (msg.status === 'completed') job.progress = 1.0;
    }

    // Update header badge on status change
    updateHeaderStatus();

    if (state.currentPage === 'jobs') {
      refreshJobsUI();
    }
    if (state.currentPage === 'job-detail') {
      const el = $('job-detail-content');
      if (el) renderJobDetailContent(job);
    }
    // Keep overview active-jobs pane in sync on any status change.
    if (state.currentPage === 'overview' && $('overview-content')) {
      renderOverviewContent(state.central.overview, state.central.jobs);
    }
    // When a pipeline job completes, refresh stats on visible pages.
    if (isDone && msg.status === 'completed') {
      setTimeout(() => refreshAfterJobComplete(job), 800);
    }
  }
}

function updateJobInPlace(job) {
  const el = $('job-' + job.id);
  if (!el) return;
  const tmp = document.createElement('div');
  tmp.innerHTML = renderJobItem(job);
  const newEl = tmp.firstElementChild;
  if (newEl) el.replaceWith(newEl);
}

// ===================================================================
// Job type helpers
// ===================================================================
const JOB_TYPE_ICONS = {
  download: { icon: '&#8595;', color: 'text-blue-400', label: 'Download' },
  markdown: { icon: '&#9998;', color: 'text-green-400', label: 'Markdown' },
  pack:     { icon: '&#9635;', color: 'text-purple-400', label: 'Pack' },
  index:    { icon: '&#9733;', color: 'text-amber-400', label: 'Index' },
};

function jobTypeBadge(type) {
  const t = JOB_TYPE_ICONS[type] || { icon: '?', color: 'ui-subtle', label: type };
  return `<span class="${t.color} font-mono">${t.icon}</span> ${esc(t.label)}`;
}

function jobTypeLabel(type) {
  return (JOB_TYPE_ICONS[type] || { label: type }).label;
}

// ===================================================================
// Jobs page
// ===================================================================
function jobCounts(jobs) {
  const counts = { total: 0, queued: 0, running: 0, completed: 0, failed: 0, cancelled: 0 };
  for (const j of (jobs || [])) {
    counts.total += 1;
    if (counts[j.status] !== undefined) counts[j.status] += 1;
  }
  return counts;
}

function refreshJobsUI() {
  renderJobsContent();
}

function renderJobsContent() {
  const el = $('jobs-content');
  if (!el) return;
  const jobs = state.central.jobs || [];
  const active = jobs.filter(j => j.status === 'running' || j.status === 'queued');
  const history = jobs.filter(j => j.status !== 'running' && j.status !== 'queued');
  const c = jobCounts(jobs);

  const cards = [
    { label: 'Total', value: c.total, cls: '', border: 'var(--border)' },
    { label: 'Running', value: c.running, cls: c.running > 0 ? 'text-blue-400' : '', border: '#3b82f6' },
    { label: 'Queued', value: c.queued, cls: c.queued > 0 ? 'text-amber-400' : '', border: '#f59e0b' },
    { label: 'Completed', value: c.completed, cls: c.completed > 0 ? 'text-green-400' : '', border: '#10b981' },
    { label: 'Failed', value: c.failed, cls: c.failed > 0 ? 'text-red-400' : '', border: '#ef4444' },
  ];

  // Group history by type
  const grouped = {};
  for (const j of history) {
    if (!grouped[j.type]) grouped[j.type] = [];
    grouped[j.type].push(j);
  }
  const typeOrder = ['download', 'markdown', 'pack', 'index'];
  const sortedTypes = typeOrder.filter(t => grouped[t]).concat(
    Object.keys(grouped).filter(t => !typeOrder.includes(t))
  );

  el.innerHTML = `
    <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-5 gap-px border border-[var(--border)] mb-5" style="background:var(--border)">
      ${cards.map(card => `
        <div class="bg-[var(--panel)] px-3 py-2.5" style="border-left:3px solid ${card.value > 0 ? card.border : 'transparent'}">
          <div class="text-[10px] font-mono ui-subtle uppercase tracking-wider">${esc(String(card.label))}</div>
          <div class="text-lg font-semibold font-mono ${card.cls}">${esc(String(card.value))}</div>
        </div>`).join('')}
    </div>

    ${active.length > 0 ? `
    <div class="surface p-4 mb-6">
      <div class="flex items-center justify-between mb-3">
        <h2 class="text-sm font-medium">Active Jobs</h2>
        <span class="meta-line">${active.length} active</span>
      </div>
      ${active.map(j => renderJobItem(j)).join('')}
    </div>` : ''}

    <div class="mb-4">
      <div class="flex items-center justify-between mb-3">
        <h2 class="text-sm font-medium">Job History</h2>
        ${history.length > 0 ? `<button onclick="clearJobHistory()" class="ui-btn px-3 py-1.5 text-xs font-mono">Clear History</button>` : ''}
      </div>
      ${sortedTypes.length === 0 ? '<div class="ui-empty">No completed jobs.</div>' : ''}
      ${sortedTypes.map(type => renderJobTypeGroup(type, grouped[type])).join('')}
    </div>`;
}

function renderJobTypeGroup(type, jobs) {
  const t = JOB_TYPE_ICONS[type] || { icon: '?', color: 'ui-subtle', label: type };
  const counts = jobCounts(jobs);
  return `
    <details class="surface mb-3" open>
      <summary class="p-3 cursor-pointer flex items-center justify-between">
        <span class="text-sm font-medium">${t.icon} ${esc(t.label)} <span class="ui-subtle font-normal">(${jobs.length})</span></span>
        <span class="text-xs font-mono ui-subtle">
          ${counts.completed > 0 ? `<span class="text-green-400">${counts.completed} ok</span>` : ''}
          ${counts.failed > 0 ? ` <span class="text-red-400">${counts.failed} fail</span>` : ''}
          ${counts.cancelled > 0 ? ` <span class="ui-subtle">${counts.cancelled} cancel</span>` : ''}
        </span>
      </summary>
      <div class="px-3 pb-3 overflow-x-auto">
        <table class="w-full text-sm ui-table">
          <thead>
            <tr>
              <th class="text-left px-3 py-1.5 text-xs font-mono">ID</th>
              <th class="text-left px-3 py-1.5 text-xs font-mono hidden sm:table-cell">Started</th>
              <th class="text-left px-3 py-1.5 text-xs font-mono">Status</th>
              <th class="text-left px-3 py-1.5 text-xs font-mono">Message / Config</th>
              <th class="text-right px-3 py-1.5 text-xs font-mono">Duration</th>
            </tr>
          </thead>
          <tbody>
            ${jobs.map(j => {
              const cfg = j.config || {};
              const cfgParts = [];
              if (cfg.files && cfg.files !== '0') cfgParts.push(`files:${esc(cfg.files)}`);
              if (cfg.engine) cfgParts.push(`engine:${esc(cfg.engine)}`);
              if (cfg.format) cfgParts.push(`fmt:${esc(cfg.format)}`);
                const detail = [j.message || j.error || '', cfgParts.join(' ')].filter(Boolean).join(' &middot; ');
              return `
              <tr class="cursor-pointer hover:bg-white/5" onclick="navigateTo('/jobs/${esc(j.id)}')">
                <td class="px-3 py-1.5 font-mono text-xs">${esc(j.id)}</td>
                <td class="px-3 py-1.5 font-mono text-xs ui-subtle hidden sm:table-cell">${j.started_at ? fmtRelativeTime(j.started_at) : '—'}</td>
                <td class="px-3 py-1.5 text-xs ${statusClass(j.status)}">${esc(j.status)}</td>
                <td class="px-3 py-1.5 text-xs ui-subtle truncate max-w-xs">${detail}</td>
                <td class="px-3 py-1.5 font-mono text-xs text-right ui-subtle">${fmtDuration(j.started_at, j.ended_at)}</td>
              </tr>`;
            }).join('')}
          </tbody>
        </table>
      </div>
    </details>`;
}

async function renderJobs() {
  state.currentPage = 'jobs';
  const main = $('main');
  const hasCache = !!(state.central.jobs);
  main.innerHTML = `
    <div class="page-shell${hasCache ? '' : ' anim-fade-in'}">
      <div class="page-header mb-4">
        <h1 class="page-title">Jobs</h1>
        <span id="jobs-live" class="text-[10px] font-mono ui-subtle">● live</span>
      </div>
      <div id="jobs-content"></div>
    </div>`;

  // Render cached state immediately — no loading flash.
  if (hasCache) {
    refreshJobsUI();
  }

  try {
    await Promise.all([
      ensureEnginesLoaded(),
      refreshCentralState(true).catch(() => {}),
      reloadJobs(),
    ]);
    ensureJobStreamSubscribed();
    refreshJobsUI();

    if (jobsPollingTimer) clearInterval(jobsPollingTimer);
    jobsPollingTimer = setInterval(async () => {
      if (state.currentPage !== 'jobs') return;
      // Only poll when WebSocket is disconnected — WS handles real-time updates
      if (wsClient.connected) return;
      await reloadJobs();
      refreshJobsUI();
    }, 5000);
  } catch (e) {
    $('jobs-content').innerHTML = `<div class="text-xs text-red-400">${esc(e.message)}</div>`;
  }
}

// ===================================================================
// Single job detail page
// ===================================================================
async function renderJobDetail(jobId) {
  state.currentPage = 'job-detail';
  const main = $('main');
  // Try central state first — avoids loading flash for known jobs.
  const cachedJob = (state.central.jobs || []).find(j => j.id === jobId);
  main.innerHTML = `
    <div class="page-shell${cachedJob ? '' : ' anim-fade-in'}">
      <div class="page-header">
        <div class="flex items-center gap-3">
          <a href="#/jobs" class="ui-link text-sm">&larr; Jobs</a>
          <h1 class="page-title">Job ${esc(jobId)}</h1>
        </div>
      </div>
      <div id="job-detail-content"></div>
    </div>`;

  if (cachedJob) {
    renderJobDetailContent(cachedJob);
  }

  try {
    let job = cachedJob;
    if (!job) {
      job = await apiGetJob(jobId);
    }
    ensureJobStreamSubscribed();
    wsClient.subscribe(jobId, (msg) => onJobUpdate(msg));
    renderJobDetailContent(job);
  } catch (e) {
    $('job-detail-content').innerHTML = `<div class="text-xs text-red-400">${esc(e.message)}</div>`;
  }
}

function renderJobDetailContent(job) {
  const el = $('job-detail-content');
  if (!el) return;

  const pct = Math.round((job.progress || 0) * 100);
  const isActive = job.status === 'running' || job.status === 'queued';
  const t = JOB_TYPE_ICONS[job.type] || { icon: '?', color: 'ui-subtle', label: job.type };
  const cfg = job.config || {};

  const configRows = [];
  if (cfg.crawl) configRows.push(['Crawl', cfg.crawl]);
  if (cfg.files) configRows.push(['Files', cfg.files]);
  if (cfg.engine) configRows.push(['Engine', cfg.engine]);
  if (cfg.source) configRows.push(['Source', cfg.source]);
  if (cfg.format) configRows.push(['Format', cfg.format]);

  el.innerHTML = `
    <div class="grid grid-cols-1 md:grid-cols-2 gap-4 mb-5">
      <div class="surface p-4">
        <div class="text-xs font-mono ui-subtle mb-2">Status</div>
        <div class="flex items-center gap-3">
          <span class="text-2xl ${t.color}">${t.icon}</span>
          <div>
            <div class="text-sm font-medium">${esc(t.label)}</div>
            <span class="text-xs ${statusClass(job.status)} font-medium">${esc(job.status)}</span>
            ${job.rate > 0 ? `<span class="text-xs ui-subtle ml-2">${job.rate.toFixed(1)} docs/s</span>` : ''}
          </div>
        </div>
      </div>
      <div class="surface p-4">
        <div class="text-xs font-mono ui-subtle mb-2">Progress</div>
        <div class="text-2xl font-medium mb-2">${pct}%</div>
        <div class="progress-track">
          <div class="progress-fill" style="width:${pct}%"></div>
        </div>
      </div>
    </div>

    ${job.message ? `
    <div class="surface p-4 mb-4">
      <div class="text-xs font-mono ui-subtle mb-1">Message</div>
      <div class="text-sm font-mono">${esc(job.message)}</div>
    </div>` : ''}

    ${job.error ? `
    <div class="surface p-4 mb-4 border border-red-500/30">
      <div class="text-xs font-mono text-red-400 mb-1">Error</div>
      <div class="text-sm font-mono text-red-300">${esc(job.error)}</div>
    </div>` : ''}

    <div class="grid grid-cols-1 md:grid-cols-2 gap-4 mb-5">
      <div class="surface p-4">
        <div class="text-xs font-mono ui-subtle mb-2">Timing</div>
        <table class="text-xs w-full">
          <tr><td class="py-1 ui-subtle">Started</td><td class="py-1 text-right font-mono">${job.started_at ? fmtRelativeTime(job.started_at) : '-'}</td></tr>
          <tr><td class="py-1 ui-subtle">Ended</td><td class="py-1 text-right font-mono">${job.ended_at ? fmtRelativeTime(job.ended_at) : (isActive ? 'running...' : '-')}</td></tr>
          <tr><td class="py-1 ui-subtle">Duration</td><td class="py-1 text-right font-mono">${fmtDuration(job.started_at, job.ended_at)}</td></tr>
        </table>
      </div>
      ${configRows.length > 0 ? `
      <div class="surface p-4">
        <div class="text-xs font-mono ui-subtle mb-2">Configuration</div>
        <table class="text-xs w-full">
          ${configRows.map(([k, v]) => `<tr><td class="py-1 ui-subtle">${esc(k)}</td><td class="py-1 text-right font-mono">${esc(v)}</td></tr>`).join('')}
        </table>
      </div>` : ''}
    </div>

    ${isActive ? `
    <div class="flex gap-2">
      <button onclick="cancelJob('${esc(job.id)}')" class="ui-btn px-4 py-2 text-xs font-mono border-red-500/30 text-red-400">Cancel Job</button>
    </div>` : ''}

    <div class="mt-4 text-xs font-mono ui-subtle">ID: ${esc(job.id)}</div>`;
}

// Refresh stats on whichever page is currently visible after a job completes.
async function refreshAfterJobComplete(completedJob) {
  try {
    // For pipeline jobs, force a metadata recalc before fetching fresh state.
    const pipelineTypes = ['download', 'markdown', 'pack', 'index'];
    if (completedJob && pipelineTypes.includes(completedJob.type)) {
      const crawl = (state.central.overview && state.central.overview.crawl_id) || '';
      apiMetaRefresh(crawl, true).catch(() => {});
    }
    await refreshCentralState(true);
    if (state.currentPage === 'warc' && state.warcDetail) {
      // On WARC detail: reload the detail to get updated metadata.
      const detailIndex = (state.warcDetail.warc || {}).index;
      if (detailIndex) {
        const data = await apiWARCDetail(detailIndex);
        state.warcDetail = data;
        if ($('warc-detail-content')) renderWARCDetailContent(data, detailIndex);
      }
    } else if (state.currentPage === 'warc') {
      // On WARC list: re-fetch to get updated summary counts (no animation replay).
      const data = await apiWARCList({
        offset: state.warcOffset || 0,
        limit: state.warcLimit || 200,
        q: state.warcQuery || '',
        phase: state.warcPhase || '',
      });
      state.warcSummary = data.summary || state.warcSummary;
      state.warcTotal = data.total || state.warcTotal;
      state.warcRows = data.warcs || state.warcRows;
      if ($('warc-summary')) renderWARCSummary(state.warcSummary);
      if ($('warc-tabs')) renderWARCTabs(state.warcSummary, state.warcTotal);
      if ($('warc-content')) renderWARCTable(data);
    } else if (state.currentPage === 'browse') {
      try {
        const data = await apiBrowse();
        state.browseShards = data.shards || [];
        renderShardList(state.browseShard);
        updateBrowseRefreshedAt();
        if (state.browseShard) loadShardDocs(state.browseShard, state.browsePage || 1);
      } catch (_) {}
    } else if (state.currentPage === 'overview') {
      if ($('overview-content')) renderOverviewContent(state.central.overview, state.central.jobs);
    }
  } catch (_) {}
}
