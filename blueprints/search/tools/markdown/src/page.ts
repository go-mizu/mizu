export function renderPage(): string {
  return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>URL → Markdown · go-mizu</title>
  <script src="https://cdn.tailwindcss.com"></script>
  <script src="https://cdn.jsdelivr.net/npm/marked@15/marked.min.js"></script>
  <script src="https://cdn.jsdelivr.net/npm/dompurify@3/dist/purify.min.js"></script>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; }
    .prose h1 { font-size: 1.5rem; font-weight: 700; margin: 1rem 0 0.5rem; color: #111827; }
    .prose h2 { font-size: 1.25rem; font-weight: 600; margin: 0.875rem 0 0.375rem; color: #111827; }
    .prose h3 { font-size: 1.1rem; font-weight: 600; margin: 0.75rem 0 0.25rem; color: #1f2937; }
    .prose h4 { font-size: 1rem; font-weight: 600; margin: 0.625rem 0 0.25rem; color: #1f2937; }
    .prose p { margin: 0.5rem 0; line-height: 1.65; color: #374151; }
    .prose ul { list-style: disc; padding-left: 1.5rem; margin: 0.5rem 0; }
    .prose ol { list-style: decimal; padding-left: 1.5rem; margin: 0.5rem 0; }
    .prose li { margin: 0.2rem 0; color: #374151; }
    .prose a { color: #4f46e5; text-decoration: underline; }
    .prose a:hover { color: #3730a3; }
    .prose blockquote { border-left: 3px solid #e5e7eb; padding-left: 1rem; color: #6b7280; margin: 0.75rem 0; font-style: italic; }
    .prose code { background: #f3f4f6; padding: 0.125rem 0.375rem; border-radius: 0.25rem; font-family: ui-monospace, monospace; font-size: 0.85em; color: #1f2937; }
    .prose pre { background: #f3f4f6; padding: 1rem; border-radius: 0.5rem; overflow-x: auto; margin: 0.75rem 0; }
    .prose pre code { background: none; padding: 0; font-size: 0.8rem; }
    .prose hr { border: none; border-top: 1px solid #e5e7eb; margin: 1rem 0; }
    .prose table { border-collapse: collapse; width: 100%; margin: 0.75rem 0; }
    .prose th, .prose td { border: 1px solid #e5e7eb; padding: 0.375rem 0.75rem; font-size: 0.875rem; }
    .prose th { background: #f9fafb; font-weight: 600; color: #374151; }
    .prose img { max-width: 100%; border-radius: 0.375rem; }
    .tab-active { border-bottom: 2px solid #18181b; color: #18181b; font-weight: 500; }
    .tab-inactive { border-bottom: 2px solid transparent; color: #6b7280; }
    .tab-inactive:hover { color: #374151; }
    .spinner { animation: spin 0.8s linear infinite; }
    @keyframes spin { to { transform: rotate(360deg); } }
  </style>
</head>
<body class="bg-gray-50 min-h-screen text-gray-900 antialiased">

  <!-- Header -->
  <header class="bg-white border-b border-gray-200 sticky top-0 z-10">
    <div class="max-w-4xl mx-auto px-4 py-3 flex items-center justify-between">
      <div class="flex items-center gap-2.5">
        <svg class="w-5 h-5 text-zinc-700" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
          <polyline points="14 2 14 8 20 8"/>
          <line x1="16" y1="13" x2="8" y2="13"/>
          <line x1="16" y1="17" x2="8" y2="17"/>
          <polyline points="10 9 9 9 8 9"/>
        </svg>
        <span class="font-semibold text-gray-900 tracking-tight">URL → Markdown</span>
        <span class="hidden sm:inline text-xs text-gray-400 font-normal">Convert any webpage to clean Markdown</span>
      </div>
      <a href="https://github.com/go-mizu/mizu" class="text-xs text-gray-400 hover:text-gray-600 transition-colors">go-mizu</a>
    </div>
  </header>

  <!-- Main -->
  <main class="max-w-4xl mx-auto px-4 py-8 space-y-4">

    <!-- URL Input Card -->
    <div class="bg-white rounded-xl border border-gray-200 shadow-sm p-5">
      <form id="form" onsubmit="handleSubmit(event)">
        <div class="flex gap-2">
          <input
            id="url-input"
            type="url"
            placeholder="https://example.com"
            autocomplete="off"
            spellcheck="false"
            class="flex-1 border border-gray-300 rounded-lg px-3 py-2.5 text-sm text-gray-900 placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-zinc-900 focus:border-transparent transition"
          />
          <button
            type="submit"
            id="submit-btn"
            class="bg-zinc-900 text-white px-5 py-2.5 rounded-lg text-sm font-medium hover:bg-zinc-700 focus:outline-none focus:ring-2 focus:ring-zinc-900 focus:ring-offset-2 transition-colors flex items-center gap-2 whitespace-nowrap"
          >
            <span id="btn-text">Convert</span>
            <svg id="btn-spinner" class="hidden spinner w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round">
              <path d="M21 12a9 9 0 1 1-6.219-8.56"/>
            </svg>
          </button>
        </div>
        <div class="mt-2.5 flex items-center gap-1.5 flex-wrap">
          <span class="text-xs text-gray-400">Try:</span>
          <button type="button" onclick="setExample('https://example.com')" class="text-xs text-indigo-600 hover:text-indigo-800 hover:underline transition-colors">example.com</button>
          <span class="text-gray-300 text-xs">·</span>
          <button type="button" onclick="setExample('https://news.ycombinator.com')" class="text-xs text-indigo-600 hover:text-indigo-800 hover:underline transition-colors">news.ycombinator.com</button>
          <span class="text-gray-300 text-xs">·</span>
          <button type="button" onclick="setExample('https://blog.cloudflare.com')" class="text-xs text-indigo-600 hover:text-indigo-800 hover:underline transition-colors">blog.cloudflare.com</button>
        </div>
      </form>
    </div>

    <!-- Error Card -->
    <div id="error-card" class="hidden bg-red-50 border border-red-200 rounded-xl p-4">
      <div class="flex items-start gap-3">
        <svg class="w-4 h-4 text-red-500 mt-0.5 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
          <circle cx="12" cy="12" r="10"/>
          <line x1="12" y1="8" x2="12" y2="12"/>
          <line x1="12" y1="16" x2="12.01" y2="16"/>
        </svg>
        <p id="error-msg" class="text-sm text-red-700"></p>
      </div>
    </div>

    <!-- Result Card -->
    <div id="result-card" class="hidden bg-white rounded-xl border border-gray-200 shadow-sm overflow-hidden">

      <!-- Meta bar -->
      <div class="px-4 py-3 bg-gray-50 border-b border-gray-200 flex items-center gap-2 flex-wrap min-w-0">
        <svg class="w-3.5 h-3.5 text-gray-400 flex-shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
          <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/>
          <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/>
        </svg>
        <span id="result-title" class="text-sm font-medium text-gray-900 truncate flex-1 min-w-0"></span>
        <span id="method-badge" class="text-xs px-2.5 py-1 rounded-full font-medium flex-shrink-0 border"></span>
        <span id="duration-badge" class="text-xs px-2.5 py-1 rounded-full bg-gray-100 text-gray-600 font-mono flex-shrink-0"></span>
        <span id="tokens-badge" class="text-xs px-2.5 py-1 rounded-full bg-gray-100 text-gray-500 flex-shrink-0 hidden"></span>
      </div>

      <!-- Tab bar -->
      <div class="flex items-center border-b border-gray-200 px-4">
        <button id="tab-md" onclick="switchTab('md')"
          class="tab-active py-3 px-1 mr-4 text-sm transition-colors">
          Markdown
        </button>
        <button id="tab-preview" onclick="switchTab('preview')"
          class="tab-inactive py-3 px-1 mr-4 text-sm transition-colors">
          Preview
        </button>
        <div class="flex-1"></div>
        <button onclick="copyMarkdown()"
          class="text-xs text-gray-500 hover:text-gray-900 px-2 py-1.5 rounded hover:bg-gray-100 transition flex items-center gap-1">
          <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
            <rect x="9" y="9" width="13" height="13" rx="2"/>
            <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>
          </svg>
          <span id="copy-label">Copy</span>
        </button>
        <button onclick="saveMarkdown()"
          class="text-xs text-gray-500 hover:text-gray-900 px-2 py-1.5 rounded hover:bg-gray-100 transition ml-1 flex items-center gap-1">
          <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
            <polyline points="7 10 12 15 17 10"/>
            <line x1="12" y1="15" x2="12" y2="3"/>
          </svg>
          Save .md
        </button>
      </div>

      <!-- Markdown panel -->
      <div id="panel-md" class="overflow-auto" style="max-height:65vh">
        <pre id="md-content" class="p-5 text-xs font-mono text-gray-800 whitespace-pre-wrap leading-relaxed"></pre>
      </div>

      <!-- Preview panel -->
      <div id="panel-preview" class="hidden overflow-auto p-6" style="max-height:65vh">
        <div id="preview-content" class="prose text-sm max-w-none"></div>
      </div>

    </div>

    <!-- How it works -->
    <div class="bg-white rounded-xl border border-gray-200 shadow-sm p-5">
      <h2 class="text-xs font-semibold text-gray-500 uppercase tracking-wider mb-4">How it works</h2>
      <div class="grid grid-cols-1 sm:grid-cols-3 gap-5">
        <div class="flex gap-3">
          <div class="w-6 h-6 rounded-full bg-violet-100 text-violet-700 text-xs font-bold flex items-center justify-center flex-shrink-0">1</div>
          <div>
            <div class="text-sm font-medium text-gray-900">Native Markdown</div>
            <div class="text-xs text-gray-500 mt-1 leading-relaxed">Sites supporting <code class="bg-gray-100 px-1 py-0.5 rounded text-gray-700">Accept: text/markdown</code> return clean Markdown directly from the edge — zero parsing.</div>
          </div>
        </div>
        <div class="flex gap-3">
          <div class="w-6 h-6 rounded-full bg-blue-100 text-blue-700 text-xs font-bold flex items-center justify-center flex-shrink-0">2</div>
          <div>
            <div class="text-sm font-medium text-gray-900">Workers AI</div>
            <div class="text-xs text-gray-500 mt-1 leading-relaxed">HTML pages are converted via Cloudflare Workers AI <code class="bg-gray-100 px-1 py-0.5 rounded text-gray-700">toMarkdown()</code> — intelligent, structured output.</div>
          </div>
        </div>
        <div class="flex gap-3">
          <div class="w-6 h-6 rounded-full bg-amber-100 text-amber-700 text-xs font-bold flex items-center justify-center flex-shrink-0">3</div>
          <div>
            <div class="text-sm font-medium text-gray-900">Browser Render</div>
            <div class="text-xs text-gray-500 mt-1 leading-relaxed">JS-heavy pages are rendered in a headless browser for full content extraction before AI conversion.</div>
          </div>
        </div>
      </div>
      <div class="mt-5 pt-4 border-t border-gray-100 space-y-1.5">
        <div class="text-xs text-gray-400 font-mono">
          <span class="text-gray-500">GET</span>  /https://example.com → <span class="text-gray-600">text/markdown</span>  <span class="text-gray-300 ml-2"># X-Conversion-Method, X-Duration-Ms</span>
        </div>
        <div class="text-xs text-gray-400 font-mono">
          <span class="text-gray-500">POST</span> /convert <span class="text-gray-600">{"url":"..."}</span> → <span class="text-gray-600">{"markdown","method","durationMs","tokens"}</span>
        </div>
      </div>
    </div>

  </main>

<script>
let currentMarkdown = '';

function handleSubmit(e) {
  e.preventDefault();
  const url = document.getElementById('url-input').value.trim();
  if (!url) return;
  convertUrl(url);
}

function setExample(url) {
  document.getElementById('url-input').value = url;
  convertUrl(url);
}

function setLoading(loading) {
  const btn = document.getElementById('submit-btn');
  btn.disabled = loading;
  document.getElementById('btn-text').textContent = loading ? 'Converting\u2026' : 'Convert';
  document.getElementById('btn-spinner').classList.toggle('hidden', !loading);
}

function showError(msg) {
  document.getElementById('error-card').classList.remove('hidden');
  document.getElementById('error-msg').textContent = msg;
  document.getElementById('result-card').classList.add('hidden');
}

function hideError() {
  document.getElementById('error-card').classList.add('hidden');
}

async function convertUrl(url) {
  setLoading(true);
  hideError();

  try {
    const resp = await fetch('/convert', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ url }),
    });
    if (!resp.ok) {
      const err = await resp.json().catch(() => ({ error: 'Conversion failed' }));
      throw new Error(err.error || 'HTTP ' + resp.status);
    }
    showResult(await resp.json());
  } catch (err) {
    showError(err.message || 'Failed to convert. Please try again.');
  } finally {
    setLoading(false);
  }
}

const METHOD_CONFIG = {
  primary: { text: '\u2726 Native Markdown', cls: 'bg-violet-50 text-violet-700 border-violet-200' },
  ai:      { text: '\u26a1 Workers AI',      cls: 'bg-blue-50 text-blue-700 border-blue-200' },
  browser: { text: '\uD83D\uDDA5 Browser Render', cls: 'bg-amber-50 text-amber-700 border-amber-200' },
};

function showResult(data) {
  currentMarkdown = data.markdown || '';

  document.getElementById('result-title').textContent = data.title || data.sourceUrl || '';

  const cfg = METHOD_CONFIG[data.method] || METHOD_CONFIG.ai;
  const badge = document.getElementById('method-badge');
  badge.textContent = cfg.text;
  badge.className = 'text-xs px-2.5 py-1 rounded-full font-medium flex-shrink-0 border ' + cfg.cls;

  document.getElementById('duration-badge').textContent = data.durationMs + 'ms';

  const tokensBadge = document.getElementById('tokens-badge');
  if (data.tokens) {
    tokensBadge.textContent = '\u007e' + data.tokens.toLocaleString() + ' tokens';
    tokensBadge.classList.remove('hidden');
  } else {
    tokensBadge.classList.add('hidden');
  }

  document.getElementById('md-content').textContent = currentMarkdown;
  document.getElementById('preview-content').innerHTML = DOMPurify.sanitize(marked.parse(currentMarkdown));

  document.getElementById('result-card').classList.remove('hidden');
  switchTab('md');
  document.getElementById('result-card').scrollIntoView({ behavior: 'smooth', block: 'start' });
}

function switchTab(tab) {
  const panels = { md: 'panel-md', preview: 'panel-preview' };
  const tabs   = { md: 'tab-md',   preview: 'tab-preview'   };
  for (const [key, panelId] of Object.entries(panels)) {
    document.getElementById(panelId).classList.toggle('hidden', key !== tab);
  }
  for (const [key, tabId] of Object.entries(tabs)) {
    document.getElementById(tabId).className =
      (key === tab ? 'tab-active' : 'tab-inactive') + ' py-3 px-1 mr-4 text-sm transition-colors';
  }
}

async function copyMarkdown() {
  try {
    await navigator.clipboard.writeText(currentMarkdown);
    const label = document.getElementById('copy-label');
    label.textContent = 'Copied!';
    setTimeout(() => { label.textContent = 'Copy'; }, 2000);
  } catch {
    // fallback: select text
  }
}

function saveMarkdown() {
  const blob = new Blob([currentMarkdown], { type: 'text/markdown' });
  const a = document.createElement('a');
  a.href = URL.createObjectURL(blob);
  const title = document.getElementById('result-title').textContent || 'document';
  a.download = title.replace(/[^\w\s-]/g, '').trim().replace(/\s+/g, '-').toLowerCase() + '.md';
  a.click();
  URL.revokeObjectURL(a.href);
}

// Handle URL in path on load (e.g., /https://example.com)
window.addEventListener('load', () => {
  const path = window.location.pathname.slice(1);
  if (path.startsWith('http://') || path.startsWith('https://')) {
    document.getElementById('url-input').value = decodeURIComponent(path);
    convertUrl(decodeURIComponent(path));
  }
});
</script>
</body>
</html>`;
}
