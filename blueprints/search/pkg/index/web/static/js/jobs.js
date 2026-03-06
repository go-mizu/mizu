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
  state.jobs = jobsData.jobs || [];
  return state.jobs;
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
  const rateStr = j.rate > 0 ? ` &middot; ${j.rate.toFixed(0)} docs/s` : '';
  const cancelBtn = (j.status === 'running' || j.status === 'queued')
    ? `<button onclick="cancelJob('${esc(j.id)}')" class="text-xs ui-link ml-2">cancel</button>`
    : '';
  return `
    <div class="mb-2 py-2 border-b" id="job-${esc(j.id)}">
      <div class="flex items-center justify-between mb-1">
        <span class="text-xs font-mono ui-subtle">${esc(j.id)} &middot; <span class="${statusClass(j.status)}">${esc(j.status)}</span>${rateStr}</span>
        <span class="text-xs font-mono ui-subtle">${pct}%${cancelBtn}</span>
      </div>
      <div class="progress-track mb-1">
        <div class="progress-fill" style="width:${pct}%"></div>
      </div>
      <div class="text-xs ui-subtle truncate">${esc(j.message || '')}</div>
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
            <th class="text-left px-4 py-2 text-xs font-mono">Status</th>
            <th class="text-left px-4 py-2 text-xs font-mono">Message</th>
            <th class="text-right px-4 py-2 text-xs font-mono">Elapsed</th>
          </tr>
        </thead>
        <tbody>
          ${jobs.map(j => `
            <tr>
              <td class="px-4 py-2 font-mono text-xs">${esc(j.id)}</td>
              <td class="px-4 py-2 text-xs">${esc(j.type)}</td>
              <td class="px-4 py-2 text-xs ${statusClass(j.status)}">${esc(j.status)}</td>
              <td class="px-4 py-2 text-xs ui-subtle truncate max-w-xs">${esc(j.message || j.error || '')}</td>
              <td class="px-4 py-2 font-mono text-xs text-right ui-subtle">${fmtDuration(j.started_at, j.ended_at)}</td>
            </tr>`).join('')}
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
            <th class="text-left px-4 py-2 text-xs font-mono">Status</th>
            <th class="text-left px-4 py-2 text-xs font-mono">Message</th>
            <th class="text-right px-4 py-2 text-xs font-mono">Elapsed</th>
          </tr>
        </thead>
        <tbody>
          ${recent.map(j => `
            <tr>
              <td class="px-4 py-2 font-mono text-xs">${esc(j.id)}</td>
              <td class="px-4 py-2 text-xs">${esc(j.type)}</td>
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

  if (type === 'markdown') {
    cfg.fast = form.elements.fast ? form.elements.fast.checked : false;
  } else if (type === 'pack') {
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
    // Add to state
    if (!state.jobs) state.jobs = [];
    state.jobs.unshift(job);

    // Subscribe to updates
    wsClient.subscribe(job.id, (msg) => onJobUpdate(msg));

    // Re-render job views if visible.
    if (state.currentPage === 'jobs') {
      const summary = $('jobs-summary');
      if (summary) summary.textContent = renderJobsSummaryLine(state.jobs || []);
      renderJobsContent();
    }
  } catch (e) {
    alert('Failed to start job: ' + e.message);
  }
}

async function cancelJob(id) {
  try {
    await apiCancelJob(id);
    // Update local state
    const job = state.jobs && state.jobs.find(j => j.id === id);
    if (job) {
      job.status = 'cancelled';
      job.ended_at = new Date().toISOString();
    }
    if (state.currentPage === 'jobs') {
      const summary = $('jobs-summary');
      if (summary) summary.textContent = renderJobsSummaryLine(state.jobs || []);
      renderJobsContent();
    }
  } catch (e) {
    alert('Failed to cancel job: ' + e.message);
  }
}

function onJobUpdate(msg) {
  if (!state.jobs) return;
  const job = state.jobs.find(j => j.id === msg.job_id);
  if (!job) return;

  if (msg.type === 'job_progress') {
    job.progress = msg.progress;
    job.message = msg.message;
    job.rate = msg.rate || 0;

    // Update in-place for active job rows on jobs page.
    if (state.currentPage === 'jobs') {
      updateJobInPlace(job);
    }
  } else if (msg.type === 'job_update') {
    job.status = msg.status;
    if (msg.error) job.error = msg.error;
    if (msg.status === 'completed' || msg.status === 'failed' || msg.status === 'cancelled') {
      job.ended_at = new Date().toISOString();
      if (msg.status === 'completed') job.progress = 1.0;
    }

    if (state.currentPage === 'jobs') {
      const summary = $('jobs-summary');
      if (summary) summary.textContent = renderJobsSummaryLine(state.jobs || []);
      renderJobsContent();
    }
  }
}

function updateJobInPlace(job) {
  const el = $('job-' + job.id);
  if (!el) return;
  const pct = Math.round((job.progress || 0) * 100);
  const rateStr = job.rate > 0 ? ` &middot; ${job.rate.toFixed(0)} docs/s` : '';
  const cancelBtn = (job.status === 'running' || job.status === 'queued')
    ? `<button onclick="cancelJob('${esc(job.id)}')" class="text-xs ui-link ml-2">cancel</button>`
    : '';

  el.innerHTML = `
    <div class="flex items-center justify-between mb-1">
      <span class="text-xs font-mono ui-subtle">${esc(job.id)} &middot; <span class="${statusClass(job.status)}">${esc(job.status)}</span>${rateStr}</span>
      <span class="text-xs font-mono ui-subtle">${pct}%${cancelBtn}</span>
    </div>
    <div class="progress-track mb-1">
      <div class="progress-fill" style="width:${pct}%"></div>
    </div>
    <div class="text-xs ui-subtle truncate">${esc(job.message || '')}</div>`;
}

// ===================================================================
// Tab 3: Jobs
// ===================================================================
function jobCounts(jobs) {
  const counts = { total: 0, queued: 0, running: 0, completed: 0, failed: 0, cancelled: 0 };
  for (const j of (jobs || [])) {
    counts.total += 1;
    if (counts[j.status] !== undefined) counts[j.status] += 1;
  }
  return counts;
}

function renderJobsSummaryLine(jobs) {
  const c = jobCounts(jobs);
  return `jobs:${c.total} · running:${c.running} · queued:${c.queued} · completed:${c.completed} · failed:${c.failed} · cancelled:${c.cancelled}`;
}

function renderJobsContent() {
  const el = $('jobs-content');
  if (!el) return;
  const jobs = state.jobs || [];
  const active = jobs.filter(j => j.status === 'running' || j.status === 'queued');
  const c = jobCounts(jobs);

  const cards = [
    { label: 'Total', value: c.total },
    { label: 'Running', value: c.running },
    { label: 'Queued', value: c.queued },
    { label: 'Failed', value: c.failed },
  ];

  el.innerHTML = `
    <div class="grid grid-cols-2 md:grid-cols-4 gap-4 mb-5">
      ${cards.map(card => `
        <div class="surface p-4">
          <div class="text-xs font-mono ui-subtle mb-1">${esc(String(card.label))}</div>
          <div class="text-lg font-medium">${esc(String(card.value))}</div>
        </div>`).join('')}
    </div>
    <div class="surface p-4 mb-6">
      <div class="flex items-center justify-between mb-3">
        <h2 class="text-sm font-medium">Active Jobs</h2>
        <span class="meta-line">${active.length} active</span>
      </div>
      ${active.length > 0 ? active.map(j => renderJobItem(j)).join('') : '<div class="ui-empty">No running or queued jobs.</div>'}
    </div>
    <div>
      <h2 class="text-sm font-medium mb-3">All Jobs</h2>
      ${renderJobHistory(jobs)}
    </div>`;
}

async function renderJobs() {
  state.currentPage = 'jobs';
  const main = $('main');
  main.innerHTML = `
    <div class="page-shell anim-fade-in">
      <div class="page-header">
        <h1 class="page-title">Jobs</h1>
        <button onclick="renderJobs()" class="ui-btn px-3 py-2 text-xs font-mono">Reload</button>
      </div>
      <div id="jobs-summary" class="meta-line mb-4">loading...</div>
      <div id="jobs-content"><div class="ui-empty">loading...</div></div>
    </div>`;

  try {
    await Promise.all([
      ensureEnginesLoaded(),
      refreshDashboardContext().catch(() => {}),
      reloadJobs(),
    ]);
    ensureJobStreamSubscribed();
    const summary = $('jobs-summary');
    if (summary) summary.textContent = renderJobsSummaryLine(state.jobs || []);
    renderJobsContent();

    if (jobsPollingTimer) clearInterval(jobsPollingTimer);
    jobsPollingTimer = setInterval(async () => {
      if (state.currentPage !== 'jobs') return;
      // Only poll when WebSocket is disconnected — WS handles real-time updates
      if (wsClient.connected) return;
      await reloadJobs();
      const line = $('jobs-summary');
      if (line) line.textContent = renderJobsSummaryLine(state.jobs || []);
      renderJobsContent();
    }, 5000);
  } catch (e) {
    $('jobs-content').innerHTML = `<div class="text-xs text-red-400">${esc(e.message)}</div>`;
  }
}
