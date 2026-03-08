// ===================================================================
// Tab: Parquet Index  (CC columnar index management + query console)
// ===================================================================

const PARQUET_SUBSETS = [
  { key: '', label: 'All' },
  { key: 'warc', label: 'warc' },
  { key: 'non200responses', label: 'non200' },
  { key: 'robotstxt', label: 'robots' },
  { key: 'crawldiagnostics', label: 'diag' },
];

const PARQUET_PRESETS = [
  { label: 'Sample 100 rows', sql: 'SELECT * FROM ccindex LIMIT 100' },
  { label: 'Record count', sql: 'SELECT COUNT(*) AS total_records FROM ccindex' },
  { label: 'Top 20 TLDs', sql: `SELECT url_host_tld, COUNT(*) AS cnt\nFROM ccindex\nWHERE url_host_tld IS NOT NULL\nGROUP BY url_host_tld\nORDER BY cnt DESC\nLIMIT 20` },
  { label: 'Top 20 domains', sql: `SELECT url_host_registered_domain, COUNT(*) AS cnt\nFROM ccindex\nWHERE url_host_registered_domain IS NOT NULL\nGROUP BY url_host_registered_domain\nORDER BY cnt DESC\nLIMIT 20` },
  { label: 'Status codes', sql: `SELECT fetch_status, COUNT(*) AS cnt\nFROM ccindex\nGROUP BY fetch_status\nORDER BY cnt DESC` },
  { label: 'MIME types', sql: `SELECT content_mime_detected, COUNT(*) AS cnt\nFROM ccindex\nWHERE content_mime_detected IS NOT NULL\nGROUP BY content_mime_detected\nORDER BY cnt DESC\nLIMIT 20` },
  { label: 'Languages', sql: `SELECT content_languages, COUNT(*) AS cnt\nFROM ccindex\nWHERE content_languages IS NOT NULL\nGROUP BY content_languages\nORDER BY cnt DESC\nLIMIT 20` },
  { label: 'Hosts per TLD (top 10)', sql: `SELECT url_host_tld,\n  COUNT(DISTINCT url_host_name) AS hosts,\n  COUNT(*) AS records\nFROM ccindex\nWHERE url_host_tld IS NOT NULL\nGROUP BY url_host_tld\nORDER BY hosts DESC\nLIMIT 10` },
];

// ── API helpers ──
async function apiParquetManifest(opts = {}) {
  const params = new URLSearchParams();
  if (opts.subset) params.set('subset', opts.subset);
  if (opts.q) params.set('q', opts.q);
  if (opts.offset !== undefined) params.set('offset', String(opts.offset));
  if (opts.limit !== undefined) params.set('limit', String(opts.limit));
  const suffix = params.toString() ? `?${params.toString()}` : '';
  return apiFetch('/api/parquet/manifest' + suffix);
}

async function apiParquetSchema() {
  return apiFetch('/api/parquet/schema');
}

async function apiParquetQuery(sql, limit = 1000) {
  return apiPost('/api/parquet/query', { sql, limit });
}

async function apiParquetDownload(body) {
  return apiPost('/api/parquet/download', body);
}

async function apiParquetStats() {
  return apiFetch('/api/parquet/stats');
}

async function apiParquetSubsetStats(subset) {
  return apiFetch(`/api/parquet/subset/${encodeURIComponent(subset)}/stats`);
}

// ── Main entry ──
async function renderParquet() {
  state.currentPage = 'parquet';
  const main = $('main');
  main.innerHTML = `
    <div class="page-shell anim-fade-in">
      <div class="page-header mb-4">
        <h1 class="page-title">Parquet Index</h1>
      </div>
      <div id="parquet-stats" class="mb-4"></div>
      <div id="parquet-tabs" class="mb-3"></div>
      <div class="flex items-center gap-2 mb-3">
        <input id="parquet-q" type="text" value="${esc(state.parquetQuery || '')}"
          placeholder="Filter by filename\u2026"
          class="ui-input flex-1 text-sm px-3 py-2"
          onkeydown="if(event.key==='Enter'){state.parquetOffset=0;state.parquetQuery=this.value;renderParquetFiles()}">
        <button onclick="parquetDownloadSubset()" class="ui-btn px-3 py-2 text-xs font-mono">Download subset</button>
      </div>
      <div id="parquet-files" class="mb-6"></div>
      <div id="parquet-schema-section" class="mb-6"></div>
      <div id="parquet-query-section"></div>
    </div>`;

  // Load everything in parallel.
  const [manifestData, statsData] = await Promise.allSettled([
    apiParquetManifest({
      subset: state.parquetSubset || '',
      q: state.parquetQuery || '',
      offset: state.parquetOffset || 0,
      limit: 200,
    }),
    apiParquetStats(),
  ]);

  if (statsData.status === 'fulfilled') {
    renderParquetStats(statsData.value);
  }
  if (manifestData.status === 'fulfilled') {
    const data = manifestData.value;
    state.parquetManifest = data;
    renderParquetSubsetTabs(data.summary);
    renderParquetFilesTable(data);
  } else {
    $('parquet-files').innerHTML = `<div class="ui-empty">${esc(manifestData.reason?.message || 'Failed to load manifest')}</div>`;
  }

  renderParquetSchemaSection();
  renderParquetQuerySection();
}

// ── Stats summary ──
function renderParquetStats(stats) {
  const el = $('parquet-stats');
  if (!el) return;
  el.innerHTML = `
    <div class="surface p-4">
      <div class="grid grid-cols-2 sm:grid-cols-5 gap-4">
        <div>
          <div class="text-[10px] font-mono ui-subtle">Crawl</div>
          <div class="text-sm font-mono font-medium">${esc(stats.crawl_id || '')}</div>
        </div>
        <div>
          <div class="text-[10px] font-mono ui-subtle">Local Files</div>
          <div class="text-sm font-mono font-medium">${fmtNum(stats.local_files)}</div>
        </div>
        <div>
          <div class="text-[10px] font-mono ui-subtle">Total Rows</div>
          <div class="text-sm font-mono font-medium">${fmtNum(stats.total_rows)}</div>
        </div>
        <div>
          <div class="text-[10px] font-mono ui-subtle">Disk Usage</div>
          <div class="text-sm font-mono font-medium">${fmtBytes(stats.disk_bytes)}</div>
        </div>
        <div>
          <div class="text-[10px] font-mono ui-subtle">Columns</div>
          <div class="text-sm font-mono font-medium">${fmtNum(stats.schema_columns)}</div>
        </div>
      </div>
    </div>`;
}

// ── Subset tabs ──
function renderParquetSubsetTabs(summary) {
  const el = $('parquet-tabs');
  if (!el || !summary) return;
  const current = state.parquetSubset || '';
  const bySubset = summary.by_subset || {};

  el.innerHTML = `
    <div class="flex items-center gap-0 border-b border-[var(--border)] overflow-x-auto" style="scrollbar-width:none">
      ${PARQUET_SUBSETS.map(s => {
        const count = s.key === '' ? summary.total : (bySubset[s.key]?.total || 0);
        const dlCount = s.key === '' ? summary.downloaded : (bySubset[s.key]?.downloaded || 0);
        if (s.key !== '' && count === 0) return '';
        const active = s.key === current;
        const cls = active ? 'tab-active' : 'tab-inactive';
        return `<button onclick="state.parquetSubset='${s.key}';state.parquetOffset=0;renderParquetFiles()"
          class="px-3 py-2 text-[11px] font-mono whitespace-nowrap ${cls} transition-colors shrink-0">
          ${esc(s.label)} <span class="ui-subtle">${count}</span>${dlCount > 0 ? ` <span class="status-completed">(${dlCount})</span>` : ''}
        </button>`;
      }).join('')}
    </div>`;
}

// ── Reload file list ──
async function renderParquetFiles() {
  const el = $('parquet-files');
  if (!el) return;
  el.innerHTML = '<div class="text-xs font-mono ui-subtle py-4">Loading\u2026</div>';
  try {
    const data = await apiParquetManifest({
      subset: state.parquetSubset || '',
      q: state.parquetQuery || '',
      offset: state.parquetOffset || 0,
      limit: 200,
    });
    state.parquetManifest = data;
    renderParquetSubsetTabs(data.summary);
    renderParquetFilesTable(data);
  } catch (e) {
    el.innerHTML = `<div class="ui-empty">${esc(e.message)}</div>`;
  }
}

// ── File table ──
function renderParquetFilesTable(data) {
  const el = $('parquet-files');
  if (!el) return;
  const files = data.files || [];
  const total = data.total || 0;
  const offset = data.offset || 0;
  const limit = data.limit || 200;

  const rows = files.map(f => `
    <tr class="file-row">
      <td class="px-3 py-2 text-xs font-mono"><a href="#/parquet/${f.manifest_index}" class="ui-link hover:text-[var(--accent)]">${f.manifest_index}</a></td>
      <td class="px-3 py-2 text-xs font-mono truncate max-w-xs" title="${esc(f.remote_path)}"><a href="#/parquet/${f.manifest_index}" class="ui-link hover:text-[var(--accent)]">${esc(f.filename)}</a></td>
      <td class="px-3 py-2 text-xs font-mono hidden sm:table-cell"><a href="#/parquet/subset/${esc(f.subset)}" class="ui-link hover:text-[var(--accent)]">${esc(f.subset)}</a></td>
      <td class="px-3 py-2 text-xs font-mono text-right hidden md:table-cell">${f.downloaded ? fmtBytes(f.local_size) : '\u2014'}</td>
      <td class="px-3 py-2 text-xs font-mono text-right">
        ${f.downloaded
          ? `<a href="#/parquet/${f.manifest_index}" class="status-completed ui-link">\u2713 local</a>`
          : `<button onclick="parquetDownloadOne(${f.manifest_index})" class="ui-btn px-2 py-1 text-[10px]">download</button>`}
      </td>
    </tr>`).join('');

  // Pagination
  const currentPage = Math.floor(offset / limit) + 1;
  const totalPages = Math.max(1, Math.ceil(total / limit));
  const canPrev = offset > 0;
  const canNext = offset + limit < total;
  const showFrom = total > 0 ? offset + 1 : 0;
  const showTo = Math.min(offset + limit, total);

  const pageButtons = buildPageNumbers(currentPage, totalPages).map(p => {
    if (p === '...') return `<span class="text-[10px] font-mono ui-subtle px-1">\u2026</span>`;
    const active = p === currentPage;
    return `<button onclick="state.parquetOffset=${(p - 1) * limit};renderParquetFiles()" class="ui-btn px-2 py-1 text-[10px] font-mono ${active ? 'ui-btn-primary' : ''}" ${active ? 'disabled' : ''}>${p}</button>`;
  }).join('');

  el.innerHTML = `
    <div class="surface overflow-x-auto">
      <table class="w-full text-sm ui-table">
        <thead>
          <tr>
            <th class="text-left px-3 py-2 text-[11px] font-mono w-16">#</th>
            <th class="text-left px-3 py-2 text-[11px] font-mono">Filename</th>
            <th class="text-left px-3 py-2 text-[11px] font-mono hidden sm:table-cell">Subset</th>
            <th class="text-right px-3 py-2 text-[11px] font-mono hidden md:table-cell">Size</th>
            <th class="text-right px-3 py-2 text-[11px] font-mono w-24">Status</th>
          </tr>
        </thead>
        <tbody>
          ${rows || '<tr><td colspan="5" class="px-3 py-4 text-xs font-mono ui-subtle">No parquet files</td></tr>'}
        </tbody>
      </table>
    </div>
    <div class="flex flex-col sm:flex-row items-center justify-between mt-3 gap-2">
      <div class="text-xs font-mono ui-subtle">${showFrom > 0 ? `${showFrom}\u2013${showTo} of ${total}` : 'no files'}</div>
      <div class="flex items-center gap-1.5 flex-wrap justify-center">
        <button ${canPrev ? '' : 'disabled'} onclick="state.parquetOffset=0;renderParquetFiles()" class="ui-btn px-2 py-1 text-[10px] font-mono">\u00ab</button>
        <button ${canPrev ? '' : 'disabled'} onclick="state.parquetOffset=${Math.max(0, offset - limit)};renderParquetFiles()" class="ui-btn px-2 py-1 text-[10px] font-mono">\u2190</button>
        ${pageButtons}
        <button ${canNext ? '' : 'disabled'} onclick="state.parquetOffset=${offset + limit};renderParquetFiles()" class="ui-btn px-2 py-1 text-[10px] font-mono">\u2192</button>
        <button ${canNext ? '' : 'disabled'} onclick="state.parquetOffset=${(totalPages - 1) * limit};renderParquetFiles()" class="ui-btn px-2 py-1 text-[10px] font-mono">\u00bb</button>
      </div>
    </div>`;
}

// ── Download helpers ──
async function parquetDownloadOne(manifestIndex) {
  try {
    const res = await apiParquetDownload({ indices: [manifestIndex] });
    if (res.started) {
      setTimeout(renderParquetFiles, 2000);
    }
  } catch (e) {
    alert('Download failed: ' + e.message);
  }
}

async function parquetDownloadSubset() {
  const subset = state.parquetSubset || 'warc';
  const sample = prompt(`Download parquet files.\n\nSubset: ${subset || 'all'}\nHow many files? (0 = all, or enter a number)`, '1');
  if (sample === null) return;
  const n = parseInt(sample, 10) || 0;
  try {
    const res = await apiParquetDownload({
      subset: subset || '',
      sample: n > 0 ? n : 0,
    });
    if (res.started) {
      alert(res.message);
      setTimeout(renderParquetFiles, 3000);
    } else {
      alert(res.message);
    }
  } catch (e) {
    alert('Download failed: ' + e.message);
  }
}

// ── Schema section ──
async function renderParquetSchemaSection() {
  const el = $('parquet-schema-section');
  if (!el) return;
  el.innerHTML = `
    <details>
      <summary class="text-[11px] font-mono ui-subtle cursor-pointer select-none mb-2">Schema</summary>
      <div id="parquet-schema-content" class="text-xs font-mono ui-subtle">Loading\u2026</div>
    </details>`;

  try {
    const data = await apiParquetSchema();
    const cols = data.columns || [];
    const content = $('parquet-schema-content');
    if (!content) return;
    if (cols.length === 0) {
      content.innerHTML = '<div class="ui-empty">No schema available (download a parquet file first)</div>';
      return;
    }
    content.innerHTML = `
      <div class="surface overflow-x-auto">
        <div class="text-[10px] font-mono ui-subtle px-3 py-2">Source: ${esc(data.source)} \u00b7 ${cols.length} columns</div>
        <table class="w-full text-sm ui-table">
          <thead>
            <tr>
              <th class="text-left px-3 py-1 text-[10px] font-mono w-12">#</th>
              <th class="text-left px-3 py-1 text-[10px] font-mono">Column</th>
              <th class="text-left px-3 py-1 text-[10px] font-mono">Type</th>
            </tr>
          </thead>
          <tbody>
            ${cols.map(c => `
              <tr>
                <td class="px-3 py-1 text-[10px] font-mono ui-subtle">${c.order}</td>
                <td class="px-3 py-1 text-xs font-mono">${esc(c.name)}</td>
                <td class="px-3 py-1 text-xs font-mono ui-subtle">${esc(c.type)}</td>
              </tr>`).join('')}
          </tbody>
        </table>
      </div>`;
  } catch (e) {
    const content = $('parquet-schema-content');
    if (content) content.innerHTML = `<div class="ui-empty">${esc(e.message)}</div>`;
  }
}

// ── Query console ──
function renderParquetQuerySection() {
  const el = $('parquet-query-section');
  if (!el) return;
  const defaultSQL = state.parquetSQL || PARQUET_PRESETS[0].sql;

  el.innerHTML = `
    <div class="surface p-4">
      <div class="flex items-center justify-between mb-3">
        <div class="text-[11px] font-mono font-medium">Query Console</div>
        <select id="parquet-preset" onchange="parquetApplyPreset(this.value)"
          class="ui-select text-xs px-2 py-1">
          <option value="">Presets\u2026</option>
          ${PARQUET_PRESETS.map((p, i) => `<option value="${i}">${esc(p.label)}</option>`).join('')}
        </select>
      </div>
      <textarea id="parquet-sql"
        class="ui-input w-full font-mono text-xs px-3 py-2 mb-3"
        rows="6" spellcheck="false"
        placeholder="SELECT * FROM ccindex LIMIT 100">${esc(defaultSQL)}</textarea>
      <div class="flex items-center gap-2 mb-3">
        <button onclick="parquetRunQuery()" class="ui-btn ui-btn-primary px-4 py-2 text-xs font-mono">Execute</button>
        <span id="parquet-query-status" class="text-[10px] font-mono ui-subtle"></span>
      </div>
      <div id="parquet-query-results"></div>
    </div>`;
}

function parquetApplyPreset(idx) {
  if (idx === '') return;
  const preset = PARQUET_PRESETS[parseInt(idx, 10)];
  if (!preset) return;
  const ta = $('parquet-sql');
  if (ta) ta.value = preset.sql;
  $('parquet-preset').value = '';
}

// ── Detail page API helpers ──
async function apiParquetFileDetail(idx) {
  return apiFetch(`/api/parquet/file/${idx}`);
}

async function apiParquetFileData(idx, opts = {}) {
  const params = new URLSearchParams();
  if (opts.page) params.set('page', String(opts.page));
  if (opts.page_size) params.set('page_size', String(opts.page_size));
  if (opts.sort) params.set('sort', opts.sort);
  if (opts.filter) params.set('filter', opts.filter);
  const suffix = params.toString() ? `?${params.toString()}` : '';
  return apiFetch(`/api/parquet/file/${idx}/data${suffix}`);
}

// ── Detail page ──
async function renderParquetDetail(idx) {
  state.currentPage = 'parquet';
  state.parquetDetailIdx = idx;
  state.parquetDetailPage = state.parquetDetailPage || 1;
  state.parquetDetailFilter = state.parquetDetailFilter || '';
  state.parquetDetailSort = state.parquetDetailSort || '';

  const main = $('main');
  main.innerHTML = `
    <div class="page-shell anim-fade-in">
      <a href="#/parquet" class="text-xs font-mono ui-link">\u2190 Parquet Index</a>
      <div id="pq-detail-content" class="mt-4">
        <div class="text-xs font-mono ui-subtle py-4">Loading\u2026</div>
      </div>
    </div>`;

  try {
    const detail = await apiParquetFileDetail(idx);
    state.parquetDetail = detail;
    renderParquetDetailContent(detail);
  } catch (e) {
    $('pq-detail-content').innerHTML = `<div class="ui-empty">${esc(e.message)}</div>`;
  }
}

function renderParquetDetailContent(detail) {
  const el = $('pq-detail-content');
  if (!el) return;

  const cols = detail.columns || [];
  const schemaHTML = cols.length > 0 ? `
    <details class="mt-4">
      <summary class="text-[11px] font-mono ui-subtle cursor-pointer select-none mb-2">Schema (${cols.length} columns)</summary>
      <div class="surface overflow-x-auto">
        <table class="w-full text-sm ui-table">
          <thead><tr>
            <th class="text-left px-3 py-1 text-[10px] font-mono w-12">#</th>
            <th class="text-left px-3 py-1 text-[10px] font-mono">Column</th>
            <th class="text-left px-3 py-1 text-[10px] font-mono">Type</th>
          </tr></thead>
          <tbody>
            ${cols.map(c => `<tr>
              <td class="px-3 py-1 text-[10px] font-mono ui-subtle">${c.order}</td>
              <td class="px-3 py-1 text-xs font-mono">${esc(c.name)}</td>
              <td class="px-3 py-1 text-xs font-mono ui-subtle">${esc(c.type)}</td>
            </tr>`).join('')}
          </tbody>
        </table>
      </div>
    </details>` : '';

  el.innerHTML = `
    <div class="page-header mb-3">
      <h1 class="page-title">${esc(detail.filename)}</h1>
    </div>

    <div class="surface p-4 mb-4">
      <div class="grid grid-cols-2 sm:grid-cols-4 gap-4">
        <div>
          <div class="text-[10px] font-mono ui-subtle">Subset</div>
          <div class="text-sm font-mono font-medium">${esc(detail.subset)}</div>
        </div>
        <div>
          <div class="text-[10px] font-mono ui-subtle">Manifest #</div>
          <div class="text-sm font-mono font-medium">${detail.manifest_index}</div>
        </div>
        <div>
          <div class="text-[10px] font-mono ui-subtle">Status</div>
          <div class="text-sm font-mono font-medium">${detail.downloaded ? '<span class="status-completed">\u2713 Downloaded</span>' : '<span class="ui-subtle">Remote</span>'}</div>
        </div>
        <div>
          <div class="text-[10px] font-mono ui-subtle">${detail.downloaded ? 'Size' : ''}</div>
          <div class="text-sm font-mono font-medium">${detail.downloaded ? fmtBytes(detail.local_size) : ''}</div>
        </div>
      </div>
      ${detail.downloaded && detail.row_count > 0 ? `
      <div class="grid grid-cols-2 sm:grid-cols-4 gap-4 mt-3 pt-3 border-t">
        <div>
          <div class="text-[10px] font-mono ui-subtle">Rows</div>
          <div class="text-sm font-mono font-medium">${fmtNum(detail.row_count)}</div>
        </div>
        <div>
          <div class="text-[10px] font-mono ui-subtle">Columns</div>
          <div class="text-sm font-mono font-medium">${fmtNum(cols.length)}</div>
        </div>
      </div>` : ''}
      <div class="mt-3 pt-3 border-t">
        <div class="text-[10px] font-mono ui-subtle mb-1">Remote Path</div>
        <div class="text-[11px] font-mono break-all">${esc(detail.remote_path)}</div>
      </div>
      ${!detail.downloaded ? `
      <div class="mt-3 pt-3 border-t">
        <button onclick="parquetDownloadAndRefreshDetail(${detail.manifest_index})" class="ui-btn ui-btn-primary px-4 py-2 text-xs font-mono">Download this file</button>
      </div>` : ''}
    </div>

    ${schemaHTML}

    ${detail.downloaded ? `
    <div class="mt-4">
      <div class="flex items-center justify-between mb-3">
        <div class="text-[11px] font-mono font-medium">Data</div>
        <div class="flex items-center gap-2">
          <input id="pq-data-filter" type="text" value="${esc(state.parquetDetailFilter || '')}"
            placeholder="WHERE clause (e.g. fetch_status = 200)"
            class="ui-input text-xs px-2 py-1 w-64"
            onkeydown="if(event.key==='Enter'){state.parquetDetailFilter=this.value;state.parquetDetailPage=1;loadParquetData()}">
          <select id="pq-data-sort" onchange="state.parquetDetailSort=this.value;loadParquetData()"
            class="ui-select text-xs px-2 py-1">
            <option value="">Default order</option>
            ${cols.map(c => `<option value="${esc(c.name)}" ${state.parquetDetailSort === c.name ? 'selected' : ''}>${esc(c.name)}</option>`).join('')}
            ${cols.map(c => `<option value="${esc(c.name)} DESC" ${state.parquetDetailSort === c.name + ' DESC' ? 'selected' : ''}>${esc(c.name)} DESC</option>`).join('')}
          </select>
        </div>
      </div>
      <div id="pq-data-content">
        <div class="text-xs font-mono ui-subtle py-4">Loading data\u2026</div>
      </div>
    </div>` : ''}`;

  if (detail.downloaded) {
    loadParquetData();
  }
}

async function loadParquetData() {
  const el = $('pq-data-content');
  if (!el) return;

  const idx = state.parquetDetailIdx;
  const page = state.parquetDetailPage || 1;
  const pageSize = 100;

  el.innerHTML = '<div class="text-xs font-mono ui-subtle py-4">Loading\u2026</div>';

  try {
    const data = await apiParquetFileData(idx, {
      page,
      page_size: pageSize,
      sort: state.parquetDetailSort || '',
      filter: state.parquetDetailFilter || '',
    });
    renderParquetDataTable(data, pageSize);
  } catch (e) {
    el.innerHTML = `<div class="ui-empty">${esc(e.message)}</div>`;
  }
}

function renderParquetDataTable(data, pageSize) {
  const el = $('pq-data-content');
  if (!el) return;

  const cols = data.columns || [];
  const rows = data.rows || [];
  const total = data.total || 0;
  const page = data.page || 1;
  const elapsed = data.elapsed_ms || 0;

  const totalPages = Math.max(1, Math.ceil(total / pageSize));
  const canPrev = page > 1;
  const canNext = page < totalPages;
  const showFrom = total > 0 ? (page - 1) * pageSize + 1 : 0;
  const showTo = Math.min(page * pageSize, total);

  // Column header — clickable to sort.
  const headerCells = cols.map(c => {
    const isActive = (state.parquetDetailSort || '').replace(/ DESC$/, '') === c;
    const isDesc = (state.parquetDetailSort || '').endsWith(' DESC');
    const nextSort = isActive && !isDesc ? c + ' DESC' : (isActive && isDesc ? '' : c);
    const arrow = isActive ? (isDesc ? ' \u25BC' : ' \u25B2') : '';
    return `<th class="text-left px-3 py-2 text-[10px] font-mono whitespace-nowrap cursor-pointer select-none hover:text-[var(--accent)]"
      onclick="state.parquetDetailSort='${esc(nextSort)}';state.parquetDetailPage=1;loadParquetData()">${esc(c)}${arrow}</th>`;
  }).join('');

  const bodyRows = rows.map(row => `
    <tr class="file-row">${row.map(cell => {
      const val = cell === null ? '<span class="ui-subtle">NULL</span>' : esc(String(cell));
      const truncated = cell !== null && String(cell).length > 80;
      return `<td class="px-3 py-1.5 text-xs font-mono whitespace-nowrap ${truncated ? 'max-w-xs truncate' : ''}" ${truncated ? `title="${esc(String(cell))}"` : ''}>${val}</td>`;
    }).join('')}</tr>`).join('');

  const pageButtons = buildPageNumbers(page, totalPages).map(p => {
    if (p === '...') return `<span class="text-[10px] font-mono ui-subtle px-1">\u2026</span>`;
    const active = p === page;
    return `<button onclick="state.parquetDetailPage=${p};loadParquetData()" class="ui-btn px-2 py-1 text-[10px] font-mono ${active ? 'ui-btn-primary' : ''}" ${active ? 'disabled' : ''}>${p}</button>`;
  }).join('');

  el.innerHTML = `
    <div class="surface overflow-x-auto" style="max-height:70vh;overflow-y:auto">
      <table class="w-full text-sm ui-table">
        <thead style="position:sticky;top:0;background:var(--panel);z-index:1">
          <tr>${headerCells}</tr>
        </thead>
        <tbody>
          ${bodyRows || '<tr><td colspan="' + cols.length + '" class="px-3 py-4 text-xs font-mono ui-subtle">No data</td></tr>'}
        </tbody>
      </table>
    </div>
    <div class="flex flex-col sm:flex-row items-center justify-between mt-3 gap-2">
      <div class="text-xs font-mono ui-subtle">
        ${showFrom > 0 ? `${fmtNum(showFrom)}\u2013${fmtNum(showTo)} of ${fmtNum(total)}` : 'no rows'}
        \u00b7 ${elapsed}ms
      </div>
      <div class="flex items-center gap-1.5 flex-wrap justify-center">
        <button ${canPrev ? '' : 'disabled'} onclick="state.parquetDetailPage=1;loadParquetData()" class="ui-btn px-2 py-1 text-[10px] font-mono">\u00ab</button>
        <button ${canPrev ? '' : 'disabled'} onclick="state.parquetDetailPage=${page - 1};loadParquetData()" class="ui-btn px-2 py-1 text-[10px] font-mono">\u2190</button>
        ${pageButtons}
        <button ${canNext ? '' : 'disabled'} onclick="state.parquetDetailPage=${page + 1};loadParquetData()" class="ui-btn px-2 py-1 text-[10px] font-mono">\u2192</button>
        <button ${canNext ? '' : 'disabled'} onclick="state.parquetDetailPage=${totalPages};loadParquetData()" class="ui-btn px-2 py-1 text-[10px] font-mono">\u00bb</button>
        <span class="text-[10px] font-mono ui-subtle mx-1">page</span>
        <input type="number" min="1" max="${totalPages}" value="${page}"
          class="ui-input w-14 text-xs px-2 py-1 text-center"
          onkeydown="if(event.key==='Enter'){state.parquetDetailPage=Math.max(1,Math.min(${totalPages},+this.value));loadParquetData()}"
          onchange="state.parquetDetailPage=Math.max(1,Math.min(${totalPages},+this.value));loadParquetData()">
        <span class="text-[10px] font-mono ui-subtle">/ ${fmtNum(totalPages)}</span>
      </div>
    </div>`;
}

async function parquetDownloadAndRefreshDetail(manifestIndex) {
  try {
    const res = await apiParquetDownload({ indices: [manifestIndex] });
    if (res.started) {
      $('pq-detail-content').innerHTML = '<div class="text-xs font-mono ui-subtle py-4">Downloading\u2026 refresh in a few seconds.</div>';
      setTimeout(() => renderParquetDetail(manifestIndex), 5000);
    }
  } catch (e) {
    alert('Download failed: ' + e.message);
  }
}

// ── Subset stats page ──
async function renderParquetSubsetStats(subset) {
  state.currentPage = 'parquet';
  const main = $('main');
  const label = (PARQUET_SUBSETS.find(s => s.key === subset) || {}).label || subset;
  main.innerHTML = `
    <div class="page-shell anim-fade-in">
      <a href="#/parquet" class="text-xs font-mono ui-link">\u2190 Parquet Index</a>
      <div class="page-header mt-4 mb-4">
        <h1 class="page-title">Subset: ${esc(label)}</h1>
      </div>
      <div id="pq-subset-summary" class="mb-4"></div>
      <div id="pq-subset-charts">
        <div class="text-xs font-mono ui-subtle py-4">Loading stats\u2026</div>
      </div>
      <div class="mt-4">
        <a href="#/parquet" onclick="state.parquetSubset='${esc(subset)}';return true" class="ui-btn px-4 py-2 text-xs font-mono">View files</a>
        <button onclick="navigateTo('/search');setTimeout(()=>{const ta=$('parquet-sql');if(ta)ta.value='SELECT * FROM ccindex WHERE subset=\\'${esc(subset)}\\' LIMIT 100'},100)" class="ui-btn px-4 py-2 text-xs font-mono ml-2">Query console</button>
      </div>
    </div>`;

  try {
    const data = await apiParquetSubsetStats(subset);
    renderSubsetSummary(data);
    renderSubsetCharts(data);
  } catch (e) {
    $('pq-subset-charts').innerHTML = `<div class="ui-empty">${esc(e.message)}</div>`;
  }
}

function renderSubsetSummary(data) {
  const el = $('pq-subset-summary');
  if (!el) return;
  el.innerHTML = `
    <div class="surface p-4">
      <div class="grid grid-cols-2 sm:grid-cols-4 gap-4">
        <div>
          <div class="text-[10px] font-mono ui-subtle">Subset</div>
          <div class="text-sm font-mono font-medium">${esc(data.subset)}</div>
        </div>
        <div>
          <div class="text-[10px] font-mono ui-subtle">Total Rows</div>
          <div class="text-sm font-mono font-medium">${fmtNum(data.total_rows)}</div>
        </div>
        <div>
          <div class="text-[10px] font-mono ui-subtle">Files</div>
          <div class="text-sm font-mono font-medium">${fmtNum(data.file_count)}</div>
        </div>
        <div>
          <div class="text-[10px] font-mono ui-subtle">Query Time</div>
          <div class="text-sm font-mono font-medium">${fmtNum(data.elapsed_ms)}ms</div>
        </div>
      </div>
    </div>`;
}

function renderSubsetCharts(data) {
  const el = $('pq-subset-charts');
  if (!el) return;
  const charts = data.charts || {};
  const keys = Object.keys(charts);
  if (keys.length === 0) {
    el.innerHTML = '<div class="ui-empty">No chart data available. Download some parquet files first.</div>';
    return;
  }

  const chartNames = {
    tld: 'Top TLDs', domain: 'Top Domains', mime: 'MIME Types',
    language: 'Languages', charset: 'Charsets', protocol: 'Protocol',
    status: 'Status Codes', redirect: 'Redirect Targets',
  };

  el.innerHTML = `
    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      ${keys.map(key => {
        const entries = charts[key] || [];
        if (entries.length === 0) return '';
        const rows = entries.map(e => ({ label: String(e.label), value: e.value, text: fmtNum(e.value) }));
        return `
          <div class="surface p-4">
            <div class="text-[11px] font-mono font-medium mb-3">${esc(chartNames[key] || key)}</div>
            ${renderBars(rows)}
          </div>`;
      }).join('')}
    </div>`;
}

async function parquetRunQuery() {
  const ta = $('parquet-sql');
  const status = $('parquet-query-status');
  const results = $('parquet-query-results');
  if (!ta || !results) return;

  const sql = ta.value.trim();
  if (!sql) return;
  state.parquetSQL = sql;

  status.textContent = 'Running\u2026';
  results.innerHTML = '';

  try {
    const data = await apiParquetQuery(sql);
    const elapsed = data.elapsed_ms || 0;
    const totalRows = data.total_rows || 0;
    status.textContent = `${totalRows.toLocaleString()} rows \u00b7 ${elapsed}ms${data.truncated ? ' (truncated)' : ''}`;

    const cols = data.columns || [];
    const rows = data.rows || [];

    if (rows.length === 0) {
      results.innerHTML = '<div class="ui-empty mt-2">No results</div>';
      return;
    }

    results.innerHTML = `
      <div class="overflow-x-auto mt-2">
        <table class="w-full text-sm ui-table">
          <thead>
            <tr>${cols.map(c => `<th class="text-left px-3 py-2 text-[10px] font-mono whitespace-nowrap">${esc(c)}</th>`).join('')}</tr>
          </thead>
          <tbody>
            ${rows.map(row => `
              <tr>${row.map(cell => {
                const val = cell === null ? '<span class="ui-subtle">NULL</span>' : esc(String(cell));
                return `<td class="px-3 py-1.5 text-xs font-mono whitespace-nowrap max-w-xs truncate">${val}</td>`;
              }).join('')}</tr>`).join('')}
          </tbody>
        </table>
      </div>`;
  } catch (e) {
    status.textContent = 'Error';
    results.innerHTML = `<div class="ui-empty mt-2 text-red-400">${esc(e.message)}</div>`;
  }
}
