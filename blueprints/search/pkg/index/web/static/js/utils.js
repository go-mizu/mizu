// ===================================================================
// Utilities
// ===================================================================
function esc(str) {
  const d = document.createElement('div');
  d.textContent = str;
  return d.innerHTML;
}

function escRegex(str) {
  return str.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

function highlight(snippet, query) {
  if (!snippet || !query) return esc(snippet || '');
  const terms = query.split(/\s+/).filter(t => t.length > 1);
  let html = esc(snippet);
  for (const term of terms) {
    html = html.replace(new RegExp(`(${escRegex(term)})`, 'gi'), '<mark>$1</mark>');
  }
  return html;
}

function fmtBytes(bytes) {
  if (!bytes || bytes <= 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  return (bytes / Math.pow(1024, i)).toFixed(i > 0 ? 1 : 0) + ' ' + units[i];
}

function docTitle(docid) {
  let name = docid.replace(/\.md$/i, '').replace(/[-_]+/g, ' ');
  try { name = decodeURIComponent(name); } catch(e) {}
  return name.length > 80 ? name.slice(0, 80) + '\u2026' : name;
}

function $(id) { return document.getElementById(id); }

function fmtDuration(startOrSecs, endStr) {
  let s;
  if (typeof startOrSecs === 'number') {
    s = Math.floor(startOrSecs);
  } else {
    const start = new Date(startOrSecs);
    const end = endStr ? new Date(endStr) : new Date();
    const ms = end - start;
    if (ms < 1000) return ms + 'ms';
    s = Math.floor(ms / 1000);
  }
  if (s < 60) return s + 's';
  const m = Math.floor(s / 60);
  const rs = s % 60;
  if (m < 60) return m + 'm ' + rs + 's';
  const h = Math.floor(m / 60);
  const rm = m % 60;
  return h + 'h ' + rm + 'm';
}

function fmtNum(n) {
  if (n == null || n === 0) return '0';
  return n.toLocaleString();
}

function statusClass(status) {
  switch (status) {
    case 'running': return 'status-running';
    case 'completed': return 'status-completed';
    case 'failed': return 'status-failed';
    case 'cancelled': return 'status-cancelled';
    case 'queued': return 'status-queued';
    default: return 'status-default';
  }
}

function statusChip(label, ok) {
  const cls = ok ? 'ui-chip-ok' : 'ui-chip-off';
  return `<span class="ui-chip ${cls}">${label}</span>`;
}

function renderBars(rows) {
  const max = rows.reduce((acc, r) => Math.max(acc, r.value || 0), 0) || 1;
  return `
    <div class="space-y-2">
      ${rows.map(r => `
        <div class="flex items-center gap-3">
          <span class="w-24 shrink-0 text-[11px] font-mono ui-subtle">${esc(r.label)}</span>
          <div class="flex-1 progress-track">
            <div class="progress-fill" style="width:${Math.max(2, ((r.value || 0) / max) * 100)}%"></div>
          </div>
          <span class="w-24 shrink-0 text-right text-[11px] font-mono ui-subtle">${esc(r.text)}</span>
        </div>`).join('')}
    </div>`;
}

function parseISOTime(s) {
  if (!s) return null;
  const t = new Date(s);
  if (isNaN(t.getTime())) return null;
  return t;
}

function fmtRelativeTime(iso) {
  const t = parseISOTime(iso);
  if (!t) return '\u2014';
  const sec = Math.floor((Date.now() - t.getTime()) / 1000);
  if (sec < 5) return 'just now';
  if (sec < 60) return `${sec}s ago`;
  const min = Math.floor(sec / 60);
  if (min < 60) return `${min}m ago`;
  const hr = Math.floor(min / 60);
  if (hr < 24) return `${hr}h ago`;
  const day = Math.floor(hr / 24);
  return `${day}d ago`;
}

function metaContext() {
  const o = state.overview || {};
  const m = o.meta || {};      // OverviewResponse.Meta nested object
  const s = state.metaStatus || {};
  const generatedAt = m.generated_at || s.finished_at || '';
  const stale = !!m.stale;
  const refreshing = !!m.refreshing || !!s.refreshing;
  const backend = m.backend || s.backend || 'scan-fallback';
  const ttlMS = s.refresh_ttl_ms || 0;
  return { generatedAt, stale, refreshing, backend, ttlMS, lastError: s.last_error || '' };
}

function renderMetaSummaryLine() {
  const m = metaContext();
  const bits = [];
  bits.push(`meta:${m.backend}`);
  bits.push(`updated:${fmtRelativeTime(m.generatedAt)}`);
  if (m.ttlMS > 0) bits.push(`ttl:${Math.round(m.ttlMS / 1000)}s`);
  bits.push(`stale:${m.stale ? 'yes' : 'no'}`);
  if (m.refreshing) bits.push('refreshing');
  if (m.lastError) bits.push(`error:${m.lastError}`);
  return bits.join(' · ');
}

function updateHeaderMetaChip() {
  if (!isDashboard) return;
  const el = $('header-status');
  if (!el) return;
  el.classList.remove('hidden');
  el.style.display = 'inline-flex';
  const dot = $('header-status-dot');
  const text = $('header-status-text');
  const m = metaContext();
  if (dot) {
    if (m.refreshing) dot.style.background = '#f59e0b';
    else if (m.lastError) dot.style.background = 'var(--danger)';
    else dot.style.background = 'var(--success)';
  }
  if (text) {
    text.textContent = m.refreshing ? 'syncing' : fmtRelativeTime(m.generatedAt);
  }
}

async function refreshDashboardMeta(force = false) {
  if (!isDashboard) return;
  const crawl = state.overview && state.overview.crawl_id ? state.overview.crawl_id : '';
  try {
    await apiMetaRefresh(crawl, force);
  } catch (_) {
    // Ignore; status line shows last known error state from /api/meta/status.
  }
  await refreshDashboardContext();
}

async function refreshDashboardContext() {
  if (!isDashboard) return;
  const crawl = state.overview && state.overview.crawl_id ? state.overview.crawl_id : '';
  const [metaStatus, overview] = await Promise.all([
    apiMetaStatus(crawl).catch(() => null),
    apiOverview().catch(() => null),
  ]);
  if (metaStatus) state.metaStatus = metaStatus;
  if (overview) {
    state.overview = overview;
    state.overviewLoadedAt = Date.now();
  }
  // Backend triggers refresh when stale — no need to trigger from frontend.
  updateHeaderMetaChip();
}

let metaWatchdogTimer = null;
function startMetaWatchdog() {
  if (!isDashboard) return;
  if (metaWatchdogTimer) clearInterval(metaWatchdogTimer);
  const tick = () => refreshDashboardContext().catch(() => {});
  tick();
  metaWatchdogTimer = setInterval(tick, 20000);
}
