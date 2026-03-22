/* ══════════════════════════════════════════════════════════════════════
   Storage — Dashboard v2 SPA
   Vanilla JS. No framework. Config from window.__DASH_CONFIG.
   Features: command palette, keyboard shortcuts, file preview with
   syntax highlighting, markdown, media players, context menus,
   real upload progress, hash-based routing.
   ══════════════════════════════════════════════════════════════════════ */
'use strict';

/* ── Config & DOM refs ────────────────────────────────────────────── */
var CFG = window.__DASH_CONFIG || {};
var $main = document.getElementById('main');

/* ── Helpers ──────────────────────────────────────────────────────── */
function h(s) { return String(s).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;') }
function $(id) { return document.getElementById(id) }
function qe(s) { return String(s).replace(/\\/g,'\\\\').replace(/'/g,"\\'") }
var Q = "'";

function fmtSize(b) {
  if (!b || b === 0) return '\u2014';
  if (b < 1024) return b + ' B';
  if (b < 1048576) return (b / 1024).toFixed(1) + ' KB';
  if (b < 1073741824) return (b / 1048576).toFixed(1) + ' MB';
  return (b / 1073741824).toFixed(1) + ' GB';
}

function fmtRel(ts) {
  if (!ts) return '\u2014';
  var d = Date.now() - ts;
  if (d < 60000) return 'now';
  if (d < 3600000) return Math.floor(d / 60000) + 'm ago';
  if (d < 86400000) return Math.floor(d / 3600000) + 'h ago';
  if (d < 2592000000) return Math.floor(d / 86400000) + 'd ago';
  return fmtDate(ts);
}

function fmtDate(ts) {
  if (!ts) return '\u2014';
  var d = new Date(ts);
  var now = new Date();
  var m = d.toLocaleDateString('en', { month: 'short', day: 'numeric' });
  return d.getFullYear() === now.getFullYear() ? m : m + ', ' + d.getFullYear();
}

async function api(path, opts) {
  var res = await fetch(path, opts);
  if (!res.ok) {
    var body = null;
    try { body = await res.json(); } catch(e) {}
    throw new Error((body && body.message) || 'Request failed: ' + res.status);
  }
  return res.json();
}

function encodePath(p) { return p.split('/').map(encodeURIComponent).join('/') }

function truncPath(p) {
  if (!p) return '\u2014';
  return p.length > 45 ? '\u2026' + p.slice(-42) : p;
}

/* ── Icons ────────────────────────────────────────────────────────── */
var IC = {
  folder: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z"/></svg>',
  file: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>',
  image: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="3" y="3" width="18" height="18"/><circle cx="8.5" cy="8.5" r="1.5"/><polyline points="21 15 16 10 5 21"/></svg>',
  video: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polygon points="23 7 16 12 23 17 23 7"/><rect x="1" y="5" width="15" height="14"/></svg>',
  audio: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M9 18V5l12-2v13"/><circle cx="6" cy="18" r="3"/><circle cx="18" cy="16" r="3"/></svg>',
  code: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/></svg>',
  markdown: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><polyline points="14 2 14 8 20 8"/><path d="M7 15l2-4 2 4"/></svg>',
  doc: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/></svg>',
  sheet: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="3" y="3" width="18" height="18"/><line x1="3" y1="9" x2="21" y2="9"/><line x1="3" y1="15" x2="21" y2="15"/><line x1="9" y1="3" x2="9" y2="21"/></svg>',
  archive: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polyline points="21 8 21 21 3 21 3 8"/><rect x="1" y="3" width="22" height="5"/></svg>',
  text: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="17" y1="10" x2="3" y2="10"/><line x1="21" y1="6" x2="3" y2="6"/><line x1="21" y1="14" x2="3" y2="14"/><line x1="17" y1="18" x2="3" y2="18"/></svg>',
  download: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>',
  upload: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/></svg>',
  share: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="18" cy="5" r="3"/><circle cx="6" cy="12" r="3"/><circle cx="18" cy="19" r="3"/><line x1="8.59" y1="13.51" x2="15.42" y2="17.49"/><line x1="15.41" y1="6.51" x2="8.59" y2="10.49"/></svg>',
  trash: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/></svg>',
  edit: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M17 3a2.83 2.83 0 114 4L7.5 20.5 2 22l1.5-5.5L17 3z"/></svg>',
  copy: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="9" y="9" width="13" height="13"/><path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1"/></svg>',
  plus: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>',
  search: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>',
  arrowL: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="19" y1="12" x2="5" y2="12"/><polyline points="12 19 5 12 12 5"/></svg>',
  arrowR: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="5" y1="12" x2="19" y2="12"/><polyline points="12 5 19 12 12 19"/></svg>',
  link: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M10 13a5 5 0 007.54.54l3-3a5 5 0 00-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 00-7.54-.54l-3 3a5 5 0 007.07 7.07l1.71-1.71"/></svg>',
  x: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>',
};

/* ── State ────────────────────────────────────────────────────────── */
var S = {
  section: 'overview',
  path: '',
  searchQ: '',
  items: [],
  previewItem: null,
  previewContent: null,
  previewLoading: false,
  mdView: 'preview',
  uploading: [],
  _mediaCleanup: null,
};

/* ══════════════════════════════════════════════════════════════════════
   TOAST
   ══════════════════════════════════════════════════════════════════════ */
var toastBox = document.createElement('div');
toastBox.className = 'toast-container';
document.body.appendChild(toastBox);

function toast(msg, type) {
  var el = document.createElement('div');
  el.className = 'toast' + (type === 'ok' ? ' toast--ok' : type === 'err' ? ' toast--err' : '');
  el.textContent = msg;
  toastBox.appendChild(el);
  setTimeout(function() { el.remove(); }, 3500);
}

/* ══════════════════════════════════════════════════════════════════════
   MODAL
   ══════════════════════════════════════════════════════════════════════ */
function showModal(title, text, confirmLabel, onConfirm) {
  var bg = document.createElement('div');
  bg.className = 'modal-bg';
  bg.innerHTML = '<div class="modal-box">' +
    '<div class="modal-title">' + h(title) + '</div>' +
    '<div class="modal-text">' + text + '</div>' +
    '<div class="modal-actions">' +
      '<button class="modal-cancel" onclick="this.closest(\'.modal-bg\').remove()">Cancel</button>' +
      '<button class="modal-confirm" id="modal-ok">' + h(confirmLabel) + '</button>' +
    '</div></div>';
  document.body.appendChild(bg);
  bg.querySelector('#modal-ok').onclick = async function() {
    var btn = bg.querySelector('#modal-ok');
    btn.disabled = true; btn.textContent = '\u2026';
    try { await onConfirm(); } finally { bg.remove(); }
  };
  bg.addEventListener('click', function(e) { if (e.target === bg) bg.remove(); });
}

function showPromptModal(title, text, placeholder, defaultVal, confirmLabel, onConfirm) {
  var bg = document.createElement('div');
  bg.className = 'modal-bg';
  bg.innerHTML = '<div class="modal-box">' +
    '<div class="modal-title">' + h(title) + '</div>' +
    '<div class="modal-text">' + text + '</div>' +
    '<input class="modal-input" id="modal-input" placeholder="' + h(placeholder) + '" value="' + h(defaultVal) + '">' +
    '<div class="modal-actions">' +
      '<button class="modal-cancel" onclick="this.closest(\'.modal-bg\').remove()">Cancel</button>' +
      '<button class="modal-confirm--primary" id="modal-ok">' + h(confirmLabel) + '</button>' +
    '</div></div>';
  document.body.appendChild(bg);
  var inp = bg.querySelector('#modal-input');
  inp.focus(); inp.select();
  bg.querySelector('#modal-ok').onclick = function() {
    var val = inp.value.trim();
    if (val) { bg.remove(); onConfirm(val); }
  };
  inp.addEventListener('keydown', function(e) { if (e.key === 'Enter') bg.querySelector('#modal-ok').click(); });
  bg.addEventListener('click', function(e) { if (e.target === bg) bg.remove(); });
}

function showCustomModal(title, bodyHtml) {
  var bg = document.createElement('div');
  bg.className = 'modal-bg';
  bg.innerHTML = '<div class="modal-box">' +
    '<div class="modal-title">' + h(title) + '</div>' +
    '<div class="modal-text">' + bodyHtml + '</div>' +
    '<div class="modal-actions">' +
      '<button class="modal-cancel" onclick="this.closest(\'.modal-bg\').remove()">Close</button>' +
    '</div></div>';
  document.body.appendChild(bg);
  bg.addEventListener('click', function(e) { if (e.target === bg) bg.remove(); });
  return bg;
}

function closeModals() {
  document.querySelectorAll('.modal-bg').forEach(function(el) { el.remove(); });
}

/* ══════════════════════════════════════════════════════════════════════
   FILE TYPE DETECTION + ICONS (from browse.js)
   ══════════════════════════════════════════════════════════════════════ */
function fileType(item) {
  if (item.type === 'directory' || item.is_folder) return 'folder';
  var ct = item.content_type || item.type || '';
  var n = (item.name || '').toLowerCase();
  if (ct.startsWith('image/') || /\.(png|jpe?g|gif|svg|webp|bmp|ico)$/.test(n)) return 'image';
  if (ct.startsWith('video/') || /\.(mp4|webm|mov|avi|mkv)$/.test(n)) return 'video';
  if (ct.startsWith('audio/') || /\.(mp3|wav|ogg|flac|aac|m4a)$/.test(n)) return 'audio';
  if (/\.pdf$/.test(n) || ct === 'application/pdf') return 'doc';
  if (/\.(docx?|odt|rtf)$/.test(n) || ct.includes('wordprocessing')) return 'doc';
  if (/\.(xlsx?|ods)$/.test(n) || ct.includes('spreadsheet')) return 'sheet';
  if (/\.(pptx?|odp)$/.test(n) || ct.includes('presentation')) return 'doc';
  if (/\.(zip|tar|gz|rar|7z|bz2)$/.test(n) || ct.includes('zip') || ct.includes('gzip')) return 'archive';
  if (/\.(js|ts|jsx|tsx|py|go|rs|rb|php|java|c|cpp|h|cs|swift|kt|sh|bash|sql|r|scala|lua|pl|ex|hs|zig)$/.test(n)) return 'code';
  if (/\.mdx?$/.test(n)) return 'markdown';
  if (/\.(json|ya?ml|toml|ini|env|xml|html?|css|scss|vue|svelte)$/.test(n)) return 'code';
  if (/\.(csv|tsv)$/.test(n) || ct === 'text/csv') return 'sheet';
  if (/\.(txt|log|rst)$/.test(n) || ct.startsWith('text/')) return 'text';
  if (ct.includes('json') || ct.includes('xml') || ct.includes('yaml')) return 'code';
  if (/Dockerfile|Makefile|go\.mod|Cargo\.toml|package\.json|tsconfig/.test(n)) return 'code';
  return 'file';
}

function fileIcon(item) {
  var t = fileType(item);
  var icon = IC[t] || IC.file;
  var cls = t === 'folder' ? 'fb-row-icon fb-row-icon--folder' : 'fb-row-icon';
  return '<div class="' + cls + '">' + icon + '</div>';
}

/* ── Syntax highlighting ──────────────────────────────────────────── */
function langFromName(n) {
  var ext = (n || '').split('.').pop().toLowerCase();
  var map = {js:'js',jsx:'js',ts:'js',tsx:'js',json:'json',py:'py',go:'go',rs:'rs',rb:'rb',java:'java',c:'c',cpp:'c',h:'c',cs:'cs',swift:'swift',kt:'kt',sh:'sh',bash:'sh',zsh:'sh',sql:'sql',html:'html',htm:'html',xml:'xml',svg:'xml',css:'css',scss:'css',yaml:'yaml',yml:'yaml',toml:'toml',md:'md',dockerfile:'sh',makefile:'sh'};
  return map[ext] || (/Dockerfile/i.test(n) ? 'sh' : /Makefile/i.test(n) ? 'sh' : null);
}

function highlightCode(code, lang) {
  var s = h(code);
  if (!lang) return s.split('\n').map(function(l) { return '<span class="line">' + l + '</span>'; }).join('');
  var rules = [];
  if (lang === 'js' || lang === 'ts' || lang === 'json') rules = [
    [/\b(const|let|var|function|return|if|else|for|while|do|switch|case|break|continue|new|this|class|extends|import|export|from|default|async|await|try|catch|throw|typeof|instanceof|in|of|yield|void|delete|true|false|null|undefined)\b/g, 'tok-kw'],
    [/(\/\/.*$|\/\*[\s\S]*?\*\/)/gm, 'tok-cm'],
    [/("(?:\\[\s\S]|[^"\\])*"|'(?:\\[\s\S]|[^'\\])*'|`(?:\\[\s\S]|[^`\\])*`)/g, 'tok-str'],
    [/\b(\d+\.?\d*(?:e[+-]?\d+)?)\b/gi, 'tok-num'],
    [/\b([A-Z][a-zA-Z0-9]*)\b/g, 'tok-type'],
    [/\b(\w+)(?=\s*\()/g, 'tok-fn'],
  ];
  else if (lang === 'py') rules = [
    [/\b(def|class|return|if|elif|else|for|while|break|continue|import|from|as|try|except|raise|with|yield|lambda|pass|True|False|None|and|or|not|in|is|async|await)\b/g, 'tok-kw'],
    [/(#.*$)/gm, 'tok-cm'],
    [/("""[\s\S]*?"""|'''[\s\S]*?'''|"(?:\\[\s\S]|[^"\\])*"|'(?:\\[\s\S]|[^'\\])*')/g, 'tok-str'],
    [/\b(\d+\.?\d*(?:e[+-]?\d+)?)\b/gi, 'tok-num'],
    [/\b(\w+)(?=\s*\()/g, 'tok-fn'],
  ];
  else if (lang === 'go') rules = [
    [/\b(package|import|func|return|if|else|for|range|switch|case|default|break|continue|go|defer|chan|select|type|struct|interface|map|var|const|true|false|nil|string|int|int8|int16|int32|int64|uint|float32|float64|bool|byte|rune|error|make|new|len|cap|append|copy|delete|panic|recover)\b/g, 'tok-kw'],
    [/(\/\/.*$|\/\*[\s\S]*?\*\/)/gm, 'tok-cm'],
    [/("(?:\\[\s\S]|[^"\\])*"|`[^`]*`)/g, 'tok-str'],
    [/\b(\d+\.?\d*(?:e[+-]?\d+)?)\b/gi, 'tok-num'],
    [/\b([A-Z][a-zA-Z0-9]*)\b/g, 'tok-type'],
    [/\b(\w+)(?=\s*\()/g, 'tok-fn'],
  ];
  else if (lang === 'html' || lang === 'xml') rules = [
    [/(<!--[\s\S]*?-->)/g, 'tok-cm'],
    [/(<\/?[a-zA-Z][a-zA-Z0-9-]*)/g, 'tok-tag'],
    [/\b([a-zA-Z-]+)(=)/g, 'tok-attr'],
    [/("(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*')/g, 'tok-str'],
  ];
  else if (lang === 'css' || lang === 'scss') rules = [
    [/(\/\*[\s\S]*?\*\/)/g, 'tok-cm'],
    [/([.#][a-zA-Z_][a-zA-Z0-9_-]*)/g, 'tok-fn'],
    [/\b([\d.]+(?:px|em|rem|%|vh|vw|deg|s|ms)?)\b/g, 'tok-num'],
    [/("(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*')/g, 'tok-str'],
    [/@[a-zA-Z-]+/g, 'tok-kw'],
  ];
  else if (lang === 'sh') rules = [
    [/(#.*$)/gm, 'tok-cm'],
    [/("(?:\\[\s\S]|[^"\\])*"|'[^']*')/g, 'tok-str'],
    [/\b(if|then|else|elif|fi|for|while|do|done|case|esac|in|function|return|exit|echo|export|local)\b/g, 'tok-kw'],
    [/\$[a-zA-Z_][a-zA-Z0-9_]*/g, 'tok-type'],
  ];
  else if (lang === 'sql') rules = [
    [/(--.*$)/gm, 'tok-cm'],
    [/\b(SELECT|FROM|WHERE|INSERT|INTO|VALUES|UPDATE|SET|DELETE|CREATE|DROP|ALTER|TABLE|JOIN|LEFT|RIGHT|ON|AND|OR|NOT|IN|IS|NULL|AS|ORDER|BY|GROUP|HAVING|LIMIT|OFFSET|UNION|COUNT|SUM|AVG|MIN|MAX|BETWEEN|LIKE|PRIMARY|KEY|FOREIGN|REFERENCES|DEFAULT|INTEGER|TEXT|REAL|BLOB|BOOLEAN)\b/gi, 'tok-kw'],
    [/('(?:[^'\\]|\\.)*')/g, 'tok-str'],
    [/\b(\d+\.?\d*)\b/g, 'tok-num'],
  ];
  else if (lang === 'rs') rules = [
    [/\b(fn|let|mut|const|if|else|for|while|loop|match|return|break|continue|struct|enum|impl|trait|type|pub|use|mod|crate|self|as|in|ref|move|async|await|true|false|Some|None|Ok|Err|Self|Box|Vec|String|Option|Result)\b/g, 'tok-kw'],
    [/(\/\/.*$|\/\*[\s\S]*?\*\/)/gm, 'tok-cm'],
    [/("(?:\\[\s\S]|[^"\\])*")/g, 'tok-str'],
    [/\b(\d+\.?\d*(?:e[+-]?\d+)?)\b/gi, 'tok-num'],
    [/\b([A-Z][a-zA-Z0-9]*)\b/g, 'tok-type'],
  ];
  else rules = [
    [/(\/\/.*$|#.*$|\/\*[\s\S]*?\*\/)/gm, 'tok-cm'],
    [/("(?:\\[\s\S]|[^"\\])*"|'(?:\\[\s\S]|[^'\\])*'|`[^`]*`)/g, 'tok-str'],
    [/\b(\d+\.?\d*)\b/g, 'tok-num'],
    [/\b(\w+)(?=\s*\()/g, 'tok-fn'],
  ];
  var tokens = [];
  rules.forEach(function(r) {
    s.replace(r[0], function(m) { var i = arguments[arguments.length - 2]; tokens.push({start: i, end: i + m.length, cls: r[1], text: m}); return m; });
  });
  tokens.sort(function(a, b) { return a.start - b.start || b.end - a.end; });
  var out = '', pos = 0, used = [];
  tokens.forEach(function(t) {
    if (t.start < pos) return;
    if (used.some(function(u) { return t.start < u; })) return;
    out += s.slice(pos, t.start) + '<span class="' + t.cls + '">' + t.text + '</span>';
    pos = t.end; used.push(t.end);
  });
  out += s.slice(pos);
  return out.split('\n').map(function(l) { return '<span class="line">' + l + '</span>'; }).join('');
}

/* ── Markdown renderer ────────────────────────────────────────────── */
function renderMarkdown(md) {
  var maths = [], codeBlocks = [], inlineCodes = [];
  function saveMath(m) { maths.push(m); return '%%%MATH_' + (maths.length - 1) + '%%%'; }
  function saveCode(_, lang, code) { codeBlocks.push({lang: lang, code: code}); return '%%%CODE_' + (codeBlocks.length - 1) + '%%%'; }
  function saveInline(_, code) { inlineCodes.push(code); return '%%%IC_' + (inlineCodes.length - 1) + '%%%'; }
  // 1. Fenced code blocks
  md = md.replace(/```(\w*)\n([\s\S]*?)```/g, saveCode);
  // 2. Inline code (before math so `$x$` in code is protected)
  md = md.replace(/`([^`]+)`/g, saveInline);
  // 3. Math: display ($$, \[) then inline ($, \()
  md = md.replace(/\$\$([\s\S]*?)\$\$/g, saveMath);
  md = md.replace(/\\\[([\s\S]*?)\\\]/g, saveMath);
  md = md.replace(/\$([^\$\n]+?)\$/g, saveMath);
  md = md.replace(/\\\((.+?)\\\)/g, saveMath);

  var html = md
    .replace(/^###\s+(.+)$/gm, '<h3>$1</h3>')
    .replace(/^##\s+(.+)$/gm, '<h2>$1</h2>')
    .replace(/^#\s+(.+)$/gm, '<h1>$1</h1>')
    .replace(/^\*\*\*$|^---$/gm, '')
    .replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
    .replace(/\*(.+?)\*/g, '<em>$1</em>')
    .replace(/!\[([^\]]*)\]\(([^)]+)\)/g, '<img src="$2" alt="$1">')
    .replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2">$1</a>')
    .replace(/^\s*>\s+(.+)$/gm, '<blockquote>$1</blockquote>')
    .replace(/^\s*[-*]\s+(.+)$/gm, '<li>$1</li>')
    .replace(/(<li>[\s\S]*?<\/li>)/g, '<ul>$1</ul>')
    .replace(/<\/ul>\s*<ul>/g, '')
    .replace(/\n\|(.+)\|\s*\n\|[-\s|:]+\|\s*\n((?:\|.+\|\s*\n)*)/g, function(_, hdr, body) {
      var ths = hdr.split('|').map(function(c) { return '<th>' + c.trim() + '</th>'; }).join('');
      var rows = body.trim().split('\n').map(function(r) {
        return '<tr>' + r.split('|').filter(Boolean).map(function(c) { return '<td>' + c.trim() + '</td>'; }).join('') + '</tr>';
      }).join('');
      return '<table><thead><tr>' + ths + '</tr></thead><tbody>' + rows + '</tbody></table>';
    })
    .replace(/\n{2,}/g, '</p><p>');
  html = html.replace(/%%%IC_(\d+)%%%/g, function(_, i) { return '<code>' + h(inlineCodes[parseInt(i)]) + '</code>'; });
  var langAlias = {python: 'py', javascript: 'js', typescript: 'ts', rust: 'rs', golang: 'go', shell: 'sh'};
  html = html.replace(/%%%CODE_(\d+)%%%/g, function(_, i) {
    var cb = codeBlocks[parseInt(i)];
    var lk = cb.lang ? (langAlias[cb.lang.toLowerCase()] || cb.lang.toLowerCase()) : null;
    var lang = lk ? langFromName('x.' + lk) : null;
    var code = lang ? highlightCode(cb.code.trim(), lang) : h(cb.code.trim());
    return '<div class="md-code-block"' + (cb.lang ? ' data-lang="' + h(cb.lang) + '"' : '') + '><code>' + code + '</code></div>';
  });
  // Restore math as HTML-escaped text (KaTeX auto-render will process in DOM)
  html = html.replace(/%%%MATH_(\d+)%%%/g, function(_, i) { return h(maths[parseInt(i)]); });
  return html;
}

/* ── KaTeX post-render ────────────────────────────────────────────── */
function postRenderMath() {
  var el = document.querySelector('.preview-md');
  if (!el || !window.renderMathInElement) return;
  try {
    window.renderMathInElement(el, {
      delimiters: [
        {left: '$$', right: '$$', display: true},
        {left: '\\[', right: '\\]', display: true},
        {left: '$', right: '$', display: false},
        {left: '\\(', right: '\\)', display: false},
      ],
      throwOnError: false,
    });
  } catch (e) {}
}

/* ── Lazy script loader ────────────────────────────────────────────── */
var _loadingScripts = {};
function loadScript(url, globalName) {
  if (window[globalName]) return Promise.resolve();
  if (_loadingScripts[url]) return _loadingScripts[url];
  _loadingScripts[url] = new Promise(function(resolve, reject) {
    var s = document.createElement('script');
    s.src = url;
    s.onload = function() { delete _loadingScripts[url]; resolve(); };
    s.onerror = function() { delete _loadingScripts[url]; reject(new Error('Failed to load ' + globalName)); };
    document.head.appendChild(s);
  });
  return _loadingScripts[url];
}

var MAMMOTH_URL = 'https://cdn.jsdelivr.net/npm/mammoth@1.8.0/mammoth.browser.min.js';
var SHEETJS_URL = 'https://cdn.sheetjs.com/xlsx-0.20.3/package/dist/xlsx.full.min.js';

/* ── DOCX preview (Mammoth.js) ───────────────────────────────────── */
function loadDocxPreview(item) {
  var container = document.getElementById('pv-docx');
  if (!container) return;
  Promise.all([
    loadScript(MAMMOTH_URL, 'mammoth'),
    resolveFileUrl(item._fullPath).then(function(url) { return fetch(url); }).then(function(r) {
      if (!r.ok) throw new Error('fetch failed');
      return r.arrayBuffer();
    })
  ]).then(function(results) {
    var buf = results[1];
    return window.mammoth.convertToHtml({ arrayBuffer: buf });
  }).then(function(result) {
    if (!document.getElementById('pv-docx')) return;
    container.innerHTML = result.value;
  }).catch(function() {
    if (!document.getElementById('pv-docx')) return;
    container.innerHTML = '<div class="preview-error"><p>Could not render document</p>' +
      '<button class="d-filter" style="margin-top:12px" onclick="fbDownload(\'' + qe(item._fullPath) + '\')">' + IC.download + ' Download instead</button></div>';
  });
}

/* ── XLSX preview (SheetJS) ──────────────────────────────────────── */
var xlsxState = { wb: null, activeSheet: 0 };

function loadXlsxPreview(item) {
  var container = document.getElementById('pv-xlsx');
  if (!container) return;
  xlsxState = { wb: null, activeSheet: 0 };
  Promise.all([
    loadScript(SHEETJS_URL, 'XLSX'),
    resolveFileUrl(item._fullPath).then(function(url) { return fetch(url); }).then(function(r) {
      if (!r.ok) throw new Error('fetch failed');
      return r.arrayBuffer();
    })
  ]).then(function(results) {
    var buf = results[1];
    xlsxState.wb = window.XLSX.read(new Uint8Array(buf), { type: 'array' });
    if (!document.getElementById('pv-xlsx')) return;
    renderXlsxSheet();
  }).catch(function() {
    if (!document.getElementById('pv-xlsx')) return;
    container.innerHTML = '<div class="preview-error"><p>Could not render spreadsheet</p>' +
      '<button class="d-filter" style="margin-top:12px" onclick="fbDownload(\'' + qe(item._fullPath) + '\')">' + IC.download + ' Download instead</button></div>';
  });
}

function renderXlsxSheet() {
  var container = document.getElementById('pv-xlsx');
  if (!container || !xlsxState.wb) return;
  var wb = xlsxState.wb;
  var names = wb.SheetNames;
  var idx = xlsxState.activeSheet;
  var ws = wb.Sheets[names[idx]];
  var tableHtml = window.XLSX.utils.sheet_to_html(ws, { editable: false });

  var tabs = '';
  if (names.length > 1) {
    tabs = '<div class="xlsx-tabs">';
    names.forEach(function(name, i) {
      tabs += '<button class="xlsx-tab' + (i === idx ? ' xlsx-tab--active' : '') + '" onclick="switchXlsxSheet(' + i + ')">' + h(name) + '</button>';
    });
    tabs += '</div>';
  }
  container.innerHTML = tabs + '<div class="preview-xlsx-table">' + tableHtml + '</div>';
}

window.switchXlsxSheet = function(i) {
  xlsxState.activeSheet = i;
  renderXlsxSheet();
};

function csvToTable(csv) {
  var rows = csv.trim().split('\n').map(function(r) {
    var cols = [], cur = '', inQ = false;
    for (var i = 0; i < r.length; i++) {
      if (r[i] === '"') inQ = !inQ;
      else if (r[i] === ',' && !inQ) { cols.push(cur.trim()); cur = ''; }
      else cur += r[i];
    }
    cols.push(cur.trim()); return cols;
  });
  if (!rows.length) return '<p>Empty file</p>';
  var hdr = rows[0], body = rows.slice(1);
  var out = '<div class="preview-table"><table><thead><tr>' + hdr.map(function(c) { return '<th>' + h(c) + '</th>'; }).join('') + '</tr></thead><tbody>';
  body.forEach(function(r) { out += '<tr>' + r.map(function(c) { return '<td>' + h(c) + '</td>'; }).join('') + '</tr>'; });
  return out + '</tbody></table></div>';
}

/* ── Badges ───────────────────────────────────────────────────────── */
function actionBadge(action) {
  var cls = 'default';
  if (action === 'write') cls = 'write';
  else if (action === 'move') cls = 'move';
  else if (action === 'delete') cls = 'delete';
  return '<span class="badge badge--' + cls + '">' + h(action) + '</span>';
}

function auditBadge(action) {
  var cls = 'default';
  if (action === 'write' || action === 'register') cls = 'write';
  else if (action === 'read') cls = 'read';
  else if (action === 'rm') cls = 'delete';
  else if (action === 'mv') cls = 'move';
  else if (action === 'login') cls = 'login';
  else if (action === 'share') cls = 'share';
  else if (action && action.startsWith('key')) cls = 'key';
  return '<span class="badge badge--' + cls + '">' + h(action) + '</span>';
}

/* ══════════════════════════════════════════════════════════════════════
   CONTEXT MENU
   ══════════════════════════════════════════════════════════════════════ */
var ctxEl = document.createElement('div');
ctxEl.className = 'ctx-menu';
ctxEl.id = 'ctx-menu';
document.body.appendChild(ctxEl);

function showCtx(x, y, item) {
  var w = window.innerWidth, ht = window.innerHeight;
  if (w < 640) { ctxEl.style.left = '0'; ctxEl.style.top = 'auto'; ctxEl.style.bottom = '0'; }
  else { ctxEl.style.left = Math.min(x, w - 200) + 'px'; ctxEl.style.top = Math.min(y, ht - 250) + 'px'; ctxEl.style.bottom = 'auto'; }
  var html = '';
  if (!item) {
    html += '<div class="ctx-item" onclick="hideCtx();fbNewFolder()">' + IC.plus + ' New folder</div>';
    html += '<div class="ctx-item" onclick="hideCtx();fbUploadClick()">' + IC.upload + ' Upload</div>';
  } else if (item.type === 'directory') {
    html += '<div class="ctx-item" onclick="hideCtx();fbNav(' + Q + '' + qe(item._fullPath || '') + '' + Q + ')">' + IC.folder + ' Open</div>';
    html += '<div class="ctx-sep"></div>';
    html += '<div class="ctx-item" onclick="hideCtx();fbRename(' + Q + '' + qe(item._fullPath || '') + '' + Q + ',' + Q + '' + qe(item.name) + '' + Q + ',true)">' + IC.edit + ' Rename</div>';
    html += '<div class="ctx-sep"></div>';
    html += '<div class="ctx-item ctx-item--danger" onclick="hideCtx();fbDelete(' + Q + '' + qe(item._fullPath || '') + '' + Q + ',' + Q + '' + qe(item.name) + '' + Q + ')">' + IC.trash + ' Delete</div>';
  } else {
    html += '<div class="ctx-item" onclick="hideCtx();openPreview(' + Q + '' + qe(item._fullPath || '') + '' + Q + ')">' + IC.file + ' Preview</div>';
    html += '<div class="ctx-item" onclick="hideCtx();fbDownload(' + Q + '' + qe(item._fullPath || '') + '' + Q + ')">' + IC.download + ' Download</div>';
    html += '<div class="ctx-sep"></div>';
    html += '<div class="ctx-item" onclick="hideCtx();fbShare(' + Q + '' + qe(item._fullPath || '') + '' + Q + ')">' + IC.share + ' Share</div>';
    html += '<div class="ctx-item" onclick="hideCtx();fbRename(' + Q + '' + qe(item._fullPath || '') + '' + Q + ',' + Q + '' + qe(item.name) + '' + Q + ',false)">' + IC.edit + ' Rename</div>';
    html += '<div class="ctx-sep"></div>';
    html += '<div class="ctx-item ctx-item--danger" onclick="hideCtx();fbDelete(' + Q + '' + qe(item._fullPath || '') + '' + Q + ',' + Q + '' + qe(item.name) + '' + Q + ')">' + IC.trash + ' Delete</div>';
  }
  ctxEl.innerHTML = html;
  ctxEl.classList.add('open');
}
function hideCtx() { ctxEl.classList.remove('open'); }
document.addEventListener('click', hideCtx);

/* ══════════════════════════════════════════════════════════════════════
   COMMAND PALETTE
   ══════════════════════════════════════════════════════════════════════ */
var cmdEl = document.createElement('div');
cmdEl.className = 'cmd-palette';
cmdEl.id = 'cmd-palette';
document.body.appendChild(cmdEl);

function openCmdPalette() {
  cmdEl.innerHTML = '<div class="cmd-box"><div class="cmd-input">' + IC.search + '<input type="text" id="cmd-search" placeholder="Search files..." autocomplete="off"></div><div class="cmd-results" id="cmd-results"></div></div>';
  cmdEl.classList.add('open');
  var inp = $('cmd-search'); inp.focus();
  var timer;
  inp.oninput = function() { clearTimeout(timer); timer = setTimeout(function() { updateCmdResults(inp.value.trim()); }, 150); };
  inp.onkeydown = cmdKeydown;
}
function closeCmdPalette() { cmdEl.classList.remove('open'); cmdEl.innerHTML = ''; }

function updateCmdResults(q) {
  if (!q) { $('cmd-results').innerHTML = ''; return; }
  api('/files/search?q=' + encodeURIComponent(q) + '&limit=10').then(function(d) {
    var r = $('cmd-results'); if (!r) return;
    var items = d.results || [];
    if (!items.length) { r.innerHTML = '<div class="cmd-empty">No results</div>'; return; }
    r.innerHTML = '<div class="cmd-group">Files</div>' + items.map(function(f, i) {
      var isDir = f.path.endsWith('/');
      var icon = isDir ? IC.folder : IC.file;
      return '<div class="cmd-result' + (i === 0 ? ' active' : '') + '" data-path="' + h(f.path) + '">' + icon + '<span>' + h(f.name || f.path) + '</span><span class="cmd-result-path">' + h(f.path) + '</span></div>';
    }).join('');
  }).catch(function() {});
}

function selectCmdResult(path) {
  closeCmdPalette();
  if (path.endsWith('/')) {
    S.path = path; S.searchQ = '';
    go('files');
  } else {
    var parent = path.replace(/[^/]+$/, '');
    S.path = parent; S.searchQ = '';
    go('files');
    setTimeout(function() { openPreview(path); }, 100);
  }
}

function cmdKeydown(e) {
  var res = $('cmd-results'); if (!res) return;
  var items = res.querySelectorAll('.cmd-result');
  var idx = -1; items.forEach(function(el, i) { if (el.classList.contains('active')) idx = i; });
  if (e.key === 'ArrowDown') { e.preventDefault(); if (idx < items.length - 1) { if (idx >= 0) items[idx].classList.remove('active'); items[idx + 1].classList.add('active'); } }
  else if (e.key === 'ArrowUp') { e.preventDefault(); if (idx > 0) { items[idx].classList.remove('active'); items[idx - 1].classList.add('active'); } }
  else if (e.key === 'Enter') { e.preventDefault(); if (idx >= 0 && items[idx]) selectCmdResult(items[idx].dataset.path); }
  else if (e.key === 'Escape') { closeCmdPalette(); }
}

document.addEventListener('mousedown', function(e) {
  if (!cmdEl.classList.contains('open')) return;
  var result = e.target.closest('.cmd-result');
  if (result) { e.preventDefault(); selectCmdResult(result.dataset.path); return; }
  if (!e.target.closest('.cmd-box')) closeCmdPalette();
});

/* ══════════════════════════════════════════════════════════════════════
   SECTION ROUTER (hash-based)
   ══════════════════════════════════════════════════════════════════════ */
var sections = {};

function go(section) {
  S.section = section;
  // Close preview if switching sections
  if (section !== 'files') { S.previewItem = null; S.previewContent = null; }
  // Update hash without triggering hashchange
  var hash = '#' + section;
  if (section === 'files' && S.path) hash = '#files/' + S.path;
  if (location.hash !== hash) history.pushState(null, '', hash);
  // Update sidebar
  document.querySelectorAll('.dash-tab').forEach(function(btn) {
    btn.classList.toggle('active', btn.getAttribute('data-section') === section);
  });
  // Render section
  var render = sections[section];
  if (render) render();
}
window.go = go;

function parseHash() {
  var hash = location.hash.slice(1) || 'overview';
  if (hash.startsWith('files/')) {
    var fp = hash.slice(6);
    // If it doesn't end with / and has an extension, it's a file preview
    if (fp && !fp.endsWith('/') && fp.includes('.')) {
      S.path = fp.replace(/[^/]+$/, '');
      S._initPreview = fp;
    } else {
      S.path = fp;
    }
    return 'files';
  }
  if (hash.startsWith('files')) {
    S.path = '';
    return 'files';
  }
  return hash;
}

window.addEventListener('hashchange', function() {
  var section = parseHash();
  go(section);
});

/* ══════════════════════════════════════════════════════════════════════
   OVERVIEW
   ══════════════════════════════════════════════════════════════════════ */
sections.overview = async function() {
  $main.innerHTML = '<div class="dash-header"><div class="dash-title">Overview</div><div class="dash-subtitle">Your storage at a glance</div></div>' +
    '<div class="stat-grid" id="stat-grid"><div class="stat-card"><div class="stat-label">Files</div><div class="stat-value"><span class="spinner"></span></div></div>' +
    '<div class="stat-card"><div class="stat-label">Storage</div><div class="stat-value"><span class="spinner"></span></div></div>' +
    '<div class="stat-card"><div class="stat-label">API Keys</div><div class="stat-value"><span class="spinner"></span></div></div></div>' +
    '<div class="dash-header"><div class="dash-title" style="font-size:15px">Recent Activity</div></div>' +
    '<div id="overview-activity"><div class="dash-loading"><span class="spinner"></span></div></div>';

  var [stats, keys, events, shares] = await Promise.all([
    api('/files/stats').catch(function() { return { files: 0, bytes: 0 }; }),
    api('/auth/keys').catch(function() { return { keys: [] }; }),
    api('/files/log?limit=10').catch(function() { return { events: [] }; }),
    api('/dashboard/shares').catch(function() { return { shares: [] }; }),
  ]);
  sharesData = shares.shares || [];

  var grid = $('stat-grid');
  if (grid) {
    grid.innerHTML =
      '<div class="stat-card" style="cursor:pointer" onclick="go(\'files\')"><div class="stat-label">Files</div><div class="stat-value">' + stats.files + '</div><div class="stat-detail">' + fmtSize(stats.bytes) + ' used</div></div>' +
      '<div class="stat-card" style="cursor:pointer" onclick="go(\'shares\')"><div class="stat-label">Shares</div><div class="stat-value">' + (sharesData ? sharesData.length : 0) + '</div><div class="stat-detail">active links</div></div>' +
      '<div class="stat-card" style="cursor:pointer" onclick="go(\'keys\')"><div class="stat-label">API Keys</div><div class="stat-value">' + keys.keys.length + '</div><div class="stat-detail">active</div></div>';
  }

  var actEl = $('overview-activity');
  if (!actEl) return;
  if (!events.events || events.events.length === 0) {
    actEl.innerHTML = '<div class="fb-empty">No recent activity</div>';
    return;
  }
  var html = '<div class="ev-list">';
  events.events.forEach(function(ev) {
    var detail = fmtSize(ev.size);
    if (ev.msg) detail += ' \u00b7 ' + ev.msg;
    html += '<div class="ev-row" style="cursor:pointer" onclick="go(\'events\')">' +
      '<span class="ev-tx">#' + ev.tx + '</span>' +
      '<span class="ev-badge">' + actionBadge(ev.action) + '</span>' +
      '<span class="ev-path" title="' + h(ev.path) + '">' + h(ev.path) + '</span>' +
      '<span class="ev-detail"><span class="ev-msg" title="' + h(detail) + '">' + h(detail) + '</span></span>' +
      '<span class="ev-time">' + fmtRel(ev.ts) + '</span></div>';
  });
  html += '</div>';
  actEl.innerHTML = html;
};

/* ══════════════════════════════════════════════════════════════════════
   FILES — Complete file browser
   ══════════════════════════════════════════════════════════════════════ */
sections.files = function() {
  if (S.previewItem) { renderPreview(); return; }
  if (S._initPreview) {
    var fp = S._initPreview;
    delete S._initPreview;
    renderFiles().then(function() { openPreview(fp); });
    return;
  }
  renderFiles();
};

async function renderFiles() {
  $main.innerHTML =
    '<div class="dash-header"><div class="dash-title">Files</div><div class="dash-subtitle">Browse, upload, and manage your files</div></div>' +
    '<div class="fb-crumbs" id="fb-crumbs"></div>' +
    '<div class="fb-toolbar">' +
      '<div class="fb-toolbar-left">' +
        '<input class="d-search" id="fb-search" placeholder="Search files..." value="' + h(S.searchQ) + '">' +
      '</div>' +
      '<div class="fb-toolbar-right">' +
        '<button class="d-filter" onclick="fbNewFolder()" title="New folder">' + IC.plus + ' Folder</button>' +
        '<button class="d-filter" onclick="fbUploadClick()" title="Upload file" style="font-weight:600;border-color:var(--text);color:var(--text)">' + IC.upload + ' Upload</button>' +
      '</div>' +
    '</div>' +
    '<div id="fb-list"></div>' +
    '<input type="file" id="fb-file-input" multiple style="display:none" onchange="fbHandleFiles(this.files)">';

  renderCrumbs();
  bindFileSearch();
  await loadFiles();
}

function renderCrumbs() {
  var el = $('fb-crumbs');
  if (!el) return;
  var parts = S.path ? S.path.split('/').filter(Boolean) : [];
  var html = '<button class="fb-crumb" onclick="fbNav(\'\')">~</button>';
  var acc = '';
  parts.forEach(function(p) {
    acc += p + '/';
    html += '<span class="fb-sep">/</span><button class="fb-crumb" onclick="fbNav(\'' + h(acc) + '\')">' + h(p) + '</button>';
  });
  el.innerHTML = html;
}

function bindFileSearch() {
  var inp = $('fb-search'); if (!inp) return;
  var timer;
  inp.addEventListener('input', function() {
    clearTimeout(timer);
    var q = inp.value.trim();
    timer = setTimeout(function() {
      S.searchQ = q;
      loadFiles();
    }, 250);
  });
  inp.addEventListener('keydown', function(e) {
    if (e.key === 'Escape') { inp.value = ''; inp.blur(); S.searchQ = ''; loadFiles(); }
    if (e.key === 'Enter') { e.preventDefault(); S.searchQ = inp.value.trim(); loadFiles(); }
  });
}

var filesTruncated = false;
var filesOffset = 0;
var filesSortCol = 'name';
var filesSortAsc = true;

async function loadFiles(append) {
  var listEl = $('fb-list');
  if (!listEl) return;
  if (!append) { S.items = []; filesOffset = 0; filesTruncated = false; }
  try {
    var data;
    if (S.searchQ) {
      data = await api('/files/search?q=' + encodeURIComponent(S.searchQ) + '&limit=100');
      S.items = (data.results || []).map(function(r) {
        var isDir = r.path.endsWith('/');
        return { name: r.name || r.path, type: isDir ? 'directory' : 'file', size: 0, updated_at: 0, _fullPath: r.path };
      });
      filesTruncated = false;
    } else {
      data = await api('/files?prefix=' + encodeURIComponent(S.path) + '&limit=200&offset=' + filesOffset);
      var newItems = (data.entries || []).map(function(e) {
        var isDir = e.type === 'directory';
        var name = e.name.replace(/\/$/, '');
        var fp = S.path + (isDir ? name + '/' : e.name);
        var ts = e.updated_at;
        if (ts && typeof ts === 'string') ts = new Date(ts).getTime();
        return { name: name, type: e.type, size: e.size || 0, updated_at: ts || 0, _fullPath: fp };
      });
      if (append) S.items = S.items.concat(newItems);
      else S.items = newItems;
      filesOffset = S.items.length;
      filesTruncated = data.truncated || false;
    }
    sortFiles();
    renderFileList(listEl);
  } catch (e) {
    listEl.innerHTML = '<div class="fb-empty">Failed to load files: ' + h(e.message) + '</div>';
  }
}

function sortFiles() {
  S.items.sort(function(a, b) {
    var aDir = a.type === 'directory' ? 0 : 1;
    var bDir = b.type === 'directory' ? 0 : 1;
    if (aDir !== bDir) return aDir - bDir;
    var cmp = 0;
    if (filesSortCol === 'name') cmp = a.name.localeCompare(b.name);
    else if (filesSortCol === 'size') cmp = (a.size || 0) - (b.size || 0);
    else if (filesSortCol === 'time') cmp = (a.updated_at || 0) - (b.updated_at || 0);
    return filesSortAsc ? cmp : -cmp;
  });
}

window.fbSort = function(col) {
  if (filesSortCol === col) filesSortAsc = !filesSortAsc;
  else { filesSortCol = col; filesSortAsc = col === 'name'; }
  sortFiles();
  var listEl = $('fb-list');
  if (listEl) renderFileList(listEl);
};

function sortArrow(col) {
  if (filesSortCol !== col) return '';
  return filesSortAsc ? ' \u2191' : ' \u2193';
}

function renderFileList(el) {
  if (!S.items || S.items.length === 0) {
    el.innerHTML = '<div class="fb-empty">' +
      '<div class="fb-empty-icon">' + IC.file + '</div>' +
      (S.searchQ ? 'No files match your search' : 'This folder is empty') + '</div>';
    return;
  }
  var html = '<div class="fb-list">';
  html += '<div class="fb-header fb-row"><div class="fb-row-icon"></div>' +
    '<div class="fb-row-name" style="cursor:pointer" onclick="fbSort(\'name\')">Name' + sortArrow('name') + '</div>' +
    '<div class="fb-row-size" style="cursor:pointer" onclick="fbSort(\'size\')">Size' + sortArrow('size') + '</div>' +
    '<div class="fb-row-time" style="cursor:pointer" onclick="fbSort(\'time\')">Modified' + sortArrow('time') + '</div>' +
    '<div class="fb-row-actions"></div></div>';
  S.items.forEach(function(f, idx) {
    var isDir = f.type === 'directory';
    var icon = fileIcon(f);
    var nameClass = isDir ? 'fb-row-name is-folder' : 'fb-row-name';
    var size = isDir ? '\u2014' : fmtSize(f.size);
    var time = f.updated_at ? fmtRel(f.updated_at) : '';

    html += '<div class="fb-row" data-idx="' + idx + '" onclick="fbOpen(' + idx + ')" oncontextmenu="fbCtx(event,' + idx + ')">' +
      icon +
      '<div class="' + nameClass + '">' + h(f.name) + '</div>' +
      '<div class="fb-row-size">' + size + '</div>' +
      '<div class="fb-row-time">' + time + '</div>' +
      '<div class="fb-row-actions">';

    if (!isDir) {
      html += '<button class="fb-act" title="Download" onclick="event.stopPropagation();fbDownload(' + Q + '' + qe(f._fullPath) + '' + Q + ')"><span>' + IC.download + '</span></button>';
      html += '<button class="fb-act" title="Share" onclick="event.stopPropagation();fbShare(' + Q + '' + qe(f._fullPath) + '' + Q + ')"><span>' + IC.share + '</span></button>';
    }
    html += '<button class="fb-act" title="Rename" onclick="event.stopPropagation();fbRename(' + Q + '' + qe(f._fullPath) + '' + Q + ',' + Q + '' + qe(f.name) + '' + Q + ',' + isDir + ')"><span>' + IC.edit + '</span></button>';
    html += '<button class="fb-act fb-act--danger" title="Delete" onclick="event.stopPropagation();fbDelete(' + Q + '' + qe(f._fullPath) + '' + Q + ',' + Q + '' + qe(f.name) + '' + Q + ')"><span>' + IC.trash + '</span></button>';
    html += '</div></div>';
  });
  html += '</div>';
  if (filesTruncated) {
    html += '<button class="load-more" onclick="loadFiles(true)">Load more files</button>';
  }
  el.innerHTML = html;
}

/* File browser actions */
window.fbNav = function(path) {
  S.path = path;
  S.searchQ = '';
  S.previewItem = null;
  S.previewContent = null;
  history.replaceState(null, '', '#files/' + path);
  renderFiles();
};

window.fbOpen = function(idx) {
  var item = S.items[idx];
  if (!item) return;
  if (item.type === 'directory') {
    fbNav(item._fullPath);
  } else {
    openPreview(item._fullPath, item);
  }
};

window.fbCtx = function(e, idx) {
  e.preventDefault();
  e.stopPropagation();
  var item = S.items[idx];
  showCtx(e.clientX, e.clientY, item);
};

window.fbDownload = function(path) {
  window.open('/files/' + encodePath(path), '_blank');
};

window.fbShare = function(path) {
  showShareTTLModal(path);
};

window.fbRename = function(path, name, isDir) {
  showPromptModal('Rename', 'Enter new name for <code>' + h(name) + '</code>', 'New name', name, 'Rename', async function(newName) {
    if (newName === name) return;
    var parentParts = path.split('/');
    if (isDir) { parentParts.pop(); parentParts.pop(); } else { parentParts.pop(); }
    var parent = parentParts.length ? parentParts.join('/') + '/' : '';
    var newPath = parent + newName + (isDir ? '/' : '');
    try {
      await api('/files/move', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ from: path, to: newPath }) });
      toast('Renamed to ' + newName, 'ok');
      loadFiles();
    } catch (e) {
      toast('Rename failed: ' + e.message, 'err');
    }
  });
};

window.fbDelete = function(path, name) {
  showModal('Delete', 'Are you sure you want to delete <code>' + h(name) + '</code>? This cannot be undone.', 'Delete', async function() {
    try {
      await api('/files/' + encodePath(path), { method: 'DELETE' });
      toast('Deleted ' + name, 'ok');
      loadFiles();
    } catch (e) {
      toast('Delete failed: ' + e.message, 'err');
    }
  });
};

window.fbNewFolder = function() {
  showPromptModal('New Folder', 'Create a new folder in the current directory.', 'Folder name', '', 'Create', async function(name) {
    var path = S.path + name + '/';
    try {
      await api('/files/mkdir', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ path: path }) });
      toast('Created folder ' + name, 'ok');
      loadFiles();
    } catch (e) {
      toast('Failed: ' + e.message, 'err');
    }
  });
};

window.fbUploadClick = function() {
  var bg = document.createElement('div');
  bg.className = 'modal-bg';
  bg.innerHTML = '<div class="modal-box upload-modal-box">' +
    '<div class="modal-title">Upload</div>' +
    '<div class="upload-modal-body">' +
      '<div class="upload-url-row">' +
        '<div class="upload-url-field" style="flex:2"><label>URL</label>' +
          '<input id="upload-url" placeholder="https://example.com/file.pdf"></div>' +
        '<div class="upload-url-field" style="flex:1"><label>Save as</label>' +
          '<input id="upload-url-name" placeholder="auto-detect"></div>' +
        '<button class="key-create-btn" id="upload-url-btn" style="height:35px;display:flex;align-items:center;gap:6px;white-space:nowrap">Fetch</button>' +
      '</div>' +
      '<div class="upload-sep">or</div>' +
      '<div class="upload-drop" id="upload-drop-area" onclick="$(\'fb-file-input\').click()">' +
        '<div class="upload-drop-icon">' + IC.upload + '</div>' +
        '<div class="upload-drop-text">Drop files here or <strong>browse</strong></div>' +
      '</div>' +
    '</div>' +
    '<div class="modal-actions" style="margin-top:16px">' +
      '<button class="modal-cancel" onclick="this.closest(\'.modal-bg\').remove()">Close</button>' +
    '</div></div>';
  document.body.appendChild(bg);
  bg.addEventListener('click', function(e) { if (e.target === bg) bg.remove(); });

  // URL auto-detect filename
  var urlInput = bg.querySelector('#upload-url');
  urlInput.addEventListener('input', function() {
    try {
      var u = new URL(urlInput.value.trim());
      var name = u.pathname.split('/').filter(Boolean).pop() || '';
      bg.querySelector('#upload-url-name').placeholder = name || 'auto-detect';
    } catch(e) {}
  });

  // URL upload
  bg.querySelector('#upload-url-btn').onclick = async function() {
    var url = urlInput.value.trim();
    if (!url) { toast('Enter a URL', 'err'); return; }
    var name = bg.querySelector('#upload-url-name').value.trim();
    var path = S.path + (name || '');
    try {
      toast('Downloading...', 'info');
      var data = await api('/files/upload-url', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ url: url, path: path || undefined }),
      });
      toast('Uploaded ' + (data.path || 'file'), 'ok');
      bg.remove();
      loadFiles();
    } catch(e) {
      toast('Failed: ' + e.message, 'err');
    }
  };

  // Drop area within modal
  var dropArea = bg.querySelector('#upload-drop-area');
  ['dragenter','dragover'].forEach(function(ev) {
    dropArea.addEventListener(ev, function(e) { e.preventDefault(); e.stopPropagation(); dropArea.classList.add('active'); });
  });
  ['dragleave','drop'].forEach(function(ev) {
    dropArea.addEventListener(ev, function(e) { e.preventDefault(); e.stopPropagation(); dropArea.classList.remove('active'); });
  });
  dropArea.addEventListener('drop', function(e) {
    if (e.dataTransfer && e.dataTransfer.files.length) {
      bg.remove();
      fbHandleFiles(e.dataTransfer.files);
    }
  });
};

window.fbHandleFiles = function(fileList) {
  uploadFiles(Array.from(fileList));
  var fi = $('fb-file-input');
  if (fi) fi.value = '';
  // Close the upload modal if it's open
  var modal = document.querySelector('.modal-bg');
  if (modal && modal.querySelector('#upload-drop-area')) modal.remove();
};

/* ── Upload system (real XHR progress, parallel, retry) ────────── */
var uploadPanel = document.createElement('div');
uploadPanel.className = 'upload-panel';
uploadPanel.id = 'upload-panel';
uploadPanel.innerHTML = '<div class="upload-panel-head"><span class="upload-panel-title" id="upload-title">Uploads</span><button class="upload-panel-close" onclick="$(\'upload-panel\').classList.remove(\'open\')">' + IC.x + '</button></div><div class="upload-list" id="upload-list"></div>';
document.body.appendChild(uploadPanel);

function uploadFiles(files) {
  uploadPanel.classList.add('open');
  files.forEach(function(file) {
    S.uploading.push({
      name: file.name, size: file.size, progress: 0, loaded: 0,
      status: 'pending', id: Math.random().toString(36).slice(2),
      file: file, retries: 0,
    });
  });
  renderUploadList();
  processUploadQueue();
}

function processUploadQueue() {
  var active = S.uploading.filter(function(u) { return u.status === 'uploading'; }).length;
  while (active < 3) {
    var next = S.uploading.find(function(u) { return u.status === 'pending'; });
    if (!next) break;
    next.status = 'uploading'; active++;
    doUpload(next);
  }
  renderUploadList();
}

function doUpload(u) {
  var file = u.file;
  var path = S.path + (file._relativePath || file.name);
  api('/files/uploads', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ path: path, content_type: file.type || 'application/octet-stream' }),
  }).then(function(d) {
    if (!d.url) { uploadDone(u, false); return; }
    var xhr = new XMLHttpRequest();
    xhr.open('PUT', d.url);
    if (d.content_type) xhr.setRequestHeader('Content-Type', d.content_type);
    xhr.upload.onprogress = function(e) {
      if (e.lengthComputable) {
        u.loaded = e.loaded;
        u.progress = Math.round(e.loaded / e.total * 100);
        renderUploadList();
      }
    };
    xhr.onload = function() {
      if (xhr.status >= 200 && xhr.status < 300) {
        api('/files/uploads/complete', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ path: path }),
        }).then(function() { uploadDone(u, true); }).catch(function() { uploadDone(u, true); });
      } else { uploadDone(u, false); }
    };
    xhr.onerror = function() {
      if (u.retries < 3) {
        u.retries++; u.status = 'retrying'; renderUploadList();
        setTimeout(function() { u.status = 'uploading'; doUpload(u); }, 1000 * u.retries);
      } else { uploadDone(u, false); }
    };
    xhr.send(file);
  }).catch(function() { uploadDone(u, false); });
}

function uploadDone(u, ok) {
  u.status = ok ? 'done' : 'error';
  u.progress = ok ? 100 : u.progress;
  renderUploadList();
  var allDone = S.uploading.every(function(x) { return x.status === 'done' || x.status === 'error'; });
  if (allDone) {
    var okCount = S.uploading.filter(function(x) { return x.status === 'done'; }).length;
    if (okCount) toast(okCount + ' uploaded', 'ok');
    if (S.section === 'files') loadFiles();
  } else {
    processUploadQueue();
  }
}

window._retryUpload = function(id) {
  var u = S.uploading.find(function(x) { return x.id === id; });
  if (!u || u.status === 'uploading') return;
  u.status = 'pending'; u.retries = 0;
  processUploadQueue();
};

function renderUploadList() {
  var ul = $('upload-list'); if (!ul) return;
  var title = $('upload-title');
  if (title) {
    var done = S.uploading.filter(function(u) { return u.status === 'done'; }).length;
    var total = S.uploading.length;
    var active = S.uploading.filter(function(u) { return u.status === 'uploading' || u.status === 'retrying'; }).length;
    if (active) title.textContent = 'Uploading ' + done + '/' + total + '...';
    else if (done === total) title.textContent = total + ' uploaded';
    else title.textContent = done + '/' + total + ' uploaded';
  }
  ul.innerHTML = S.uploading.map(function(u) {
    var icon, extra = '';
    if (u.status === 'done') icon = '<span class="upload-ok">\u2713</span>';
    else if (u.status === 'error') { icon = '<span class="upload-err">\u2717</span>'; extra = '<button class="upload-retry" onclick="_retryUpload(\'' + u.id + '\')">retry</button>'; }
    else if (u.status === 'uploading') icon = '<span class="upload-pct">' + u.progress + '%</span>';
    else icon = '<span class="upload-pct">\u2022</span>';
    var bar = '';
    if (u.status === 'uploading' || u.status === 'retrying' || u.status === 'done') {
      bar = '<div class="upload-item-bar"><div class="upload-item-fill' + (u.status === 'done' ? ' fill--done' : u.status === 'retrying' ? ' fill--retry' : '') + '" style="width:' + u.progress + '%"></div></div>';
    } else if (u.status === 'error') {
      bar = '<div class="upload-item-bar"><div class="upload-item-fill fill--err" style="width:' + u.progress + '%"></div></div>';
    }
    var sizeStr = u.size ? (u.loaded > 0 ? fmtSize(u.loaded) : '0 B') + ' / ' + fmtSize(u.size) : '';
    return '<div class="upload-item"><div class="upload-item-top"><span class="upload-item-name">' + h(u.name) + '</span>' + icon + extra + '</div>' + bar + '<div class="upload-item-meta">' + sizeStr + '</div></div>';
  }).join('');
}

/* ── Drop zone ────────────────────────────────────────────────────── */
var dropZone = document.createElement('div');
dropZone.className = 'drop-zone';
dropZone.id = 'drop-zone';
dropZone.innerHTML = '<div class="drop-zone-icon">' + IC.upload + '</div><div class="drop-zone-text">Drop files to upload</div><div class="drop-zone-sub">Upload to ' + h(S.path || '/') + '</div>';
document.body.appendChild(dropZone);

var dragCount = 0;
function collectDropFiles(dt) {
  return new Promise(function(resolve) {
    if (!dt.items || !dt.items.length) { resolve(dt.files ? Array.from(dt.files) : []); return; }
    var files = [], pending = 0, done = false;
    function finish() { if (!done && pending === 0) { done = true; resolve(files); } }
    function readEntry(entry, pathPrefix) {
      if (entry.isFile) {
        pending++;
        entry.file(function(f) {
          var fullPath = pathPrefix ? pathPrefix + '/' + f.name : f.name;
          Object.defineProperty(f, '_relativePath', { value: fullPath });
          files.push(f); pending--; finish();
        }, function() { pending--; finish(); });
      } else if (entry.isDirectory) {
        pending++;
        var reader = entry.createReader();
        reader.readEntries(function(entries) {
          pending--;
          entries.forEach(function(e) { readEntry(e, pathPrefix ? pathPrefix + '/' + entry.name : entry.name); });
          finish();
        }, function() { pending--; finish(); });
      }
    }
    for (var i = 0; i < dt.items.length; i++) {
      var item = dt.items[i];
      if (item.webkitGetAsEntry) { var entry = item.webkitGetAsEntry(); if (entry) { readEntry(entry, ''); continue; } }
      var f = item.getAsFile(); if (f) files.push(f);
    }
    setTimeout(function() { if (!done) { done = true; resolve(files); } }, 3000);
    finish();
  });
}

document.addEventListener('dragenter', function(e) { e.preventDefault(); dragCount++; dropZone.classList.add('open'); dropZone.querySelector('.drop-zone-sub').textContent = 'Upload to ' + (S.path || '/'); });
document.addEventListener('dragleave', function(e) { e.preventDefault(); dragCount--; if (dragCount <= 0) { dragCount = 0; dropZone.classList.remove('open'); } });
document.addEventListener('dragover', function(e) { e.preventDefault(); e.dataTransfer.dropEffect = 'copy'; });
document.addEventListener('drop', function(e) {
  e.preventDefault(); dragCount = 0; dropZone.classList.remove('open');
  if (!e.dataTransfer) return;
  collectDropFiles(e.dataTransfer).then(function(files) { if (files.length) uploadFiles(files); });
});

/* (inline drop zone removed — using upload modal instead) */

/* ══════════════════════════════════════════════════════════════════════
   FILE PREVIEW
   ══════════════════════════════════════════════════════════════════════ */
function openPreview(path, itemOverride) {
  var item = itemOverride || S.items.find(function(f) { return f._fullPath === path; });
  if (!item) {
    // Item not in current list — create a minimal one
    var name = path.replace(/\/$/, '').split('/').pop();
    item = { name: name, type: 'file', size: 0, updated_at: 0, _fullPath: path };
  }
  if (item.type === 'directory') { fbNav(item._fullPath); return; }
  S.previewItem = item;
  S.previewContent = null;
  S.previewLoading = false;
  S.mdView = 'preview';
  history.replaceState(null, '', '#files/' + item._fullPath);

  var ft = fileType(item);
  var n = (item.name || '').toLowerCase();
  // Binary spreadsheet formats — don't fetch as text
  var isBinarySheet = ft === 'sheet' && !/\.(csv|tsv)$/i.test(n);
  if (!isBinarySheet && (ft === 'code' || ft === 'text' || ft === 'sheet' || ft === 'markdown')) {
    renderPreview();
    fetchTextContent(item._fullPath).then(function(t) {
      S.previewContent = t; renderPreview();
    }).catch(function() { renderPreview(); });
    return;
  }
  renderPreview();
}
window.openPreview = openPreview;

function closePreview() {
  if (S._mediaCleanup) { S._mediaCleanup(); S._mediaCleanup = null; }
  S.previewItem = null;
  S.previewContent = null;
  history.replaceState(null, '', '#files/' + S.path);
  renderFiles();
}
window.closePreview = closePreview;

function fetchTextContent(path) {
  // First try: get presigned URL and fetch from R2
  return resolveFileUrl(path).then(function(url) {
    return fetch(url).then(function(r) {
      if (!r.ok) throw new Error('not ok');
      return r.text();
    });
  }).catch(function() {
    // Fallback: fetch through server endpoint (follows redirect)
    return fetch('/files/' + encodePath(path)).then(function(r) { return r.text(); });
  });
}

function resolveFileUrl(path) {
  return fetch('/files/' + encodePath(path), {
    headers: { 'Accept': 'application/json' },
    redirect: 'manual',
  }).then(function(r) {
    if (r.type === 'opaqueredirect' || r.status === 302 || r.status === 301) {
      // Server redirected — use the Location header or fall back
      var loc = r.headers.get('Location');
      return loc || '/files/' + encodePath(path);
    }
    return r.json();
  }).then(function(d) {
    if (typeof d === 'string') return d; // already resolved from redirect
    if (d && d.url) return d.url;
    return '/files/' + encodePath(path);
  }).catch(function() {
    return '/files/' + encodePath(path);
  });
}

function renderPreview() {
  if (S._mediaCleanup) { S._mediaCleanup(); S._mediaCleanup = null; }
  var item = S.previewItem;
  if (!item) { renderFiles(); return; }
  var ft = fileType(item);
  var name = h(item.name);
  var meta = ft + ' \u00b7 ' + fmtSize(item.size) + (item.updated_at ? ' \u00b7 ' + fmtRel(item.updated_at) : '');

  // Build body
  var body = '';
  if (S.previewContent !== null) {
    var c = S.previewContent;
    if (ft === 'markdown') {
      if (S.mdView === 'code') { var lang = langFromName(item.name); body = '<pre class="preview-code">' + highlightCode(c, lang) + '</pre>'; }
      else { body = '<div class="preview-md">' + renderMarkdown(c) + '</div>'; }
    } else if (ft === 'code') { var lang = langFromName(item.name); body = '<pre class="preview-code">' + highlightCode(c, lang) + '</pre>'; }
    else if (ft === 'sheet') { body = csvToTable(c); }
    else { body = '<pre class="preview-text">' + h(c) + '</pre>'; }
  } else if (ft === 'doc' && (item.name || '').toLowerCase().endsWith('.pdf')) {
    body = '<iframe class="preview-pdf" id="pv-pdf" src="" style="width:100%;height:min(80vh,600px);min-height:300px;border:1px solid var(--border)"></iframe>';
    setTimeout(function() {
      var iframe = $('pv-pdf');
      if (iframe) resolveFileUrl(item._fullPath).then(function(url) { iframe.src = url; });
    }, 0);
  } else if (ft === 'doc' && /\.docx$/i.test(item.name || '')) {
    body = '<div class="preview-docx" id="pv-docx"><div class="dash-loading"><span class="spinner"></span></div></div>';
    setTimeout(function() { loadDocxPreview(item); }, 0);
  } else if (ft === 'sheet' && S.previewContent === null) {
    // Binary spreadsheet (xlsx/xls/ods) — render with SheetJS
    body = '<div class="preview-xlsx" id="pv-xlsx"><div class="dash-loading"><span class="spinner"></span></div></div>';
    setTimeout(function() { loadXlsxPreview(item); }, 0);
  } else if (ft === 'image') {
    body = '<img class="preview-img" id="pv-img" src="" alt="' + name + '">';
    setTimeout(function() {
      var img = $('pv-img');
      if (img) resolveFileUrl(item._fullPath).then(function(url) { img.src = url; });
    }, 0);
  } else if (ft === 'audio') {
    var bars = '';
    for (var i = 0; i < 50; i++) bars += '<div class="mp-wave-bar" style="height:' + (10 + Math.random() * 28) + 'px"></div>';
    body = '<div class="mp mp--audio" id="mp-audio"><div class="mp-art">' + IC.audio + '</div><div class="mp-title">' + name + '</div><div class="mp-wave" id="mp-wave">' + bars + '</div>' +
      '<div class="mp-controls"><button class="mp-play-btn" id="mp-play"><svg viewBox="0 0 24 24" fill="currentColor"><polygon points="5 3 19 12 5 21 5 3"/></svg></button>' +
      '<span class="mp-time" id="mp-time">0:00</span><div class="mp-progress" id="mp-progress"><div class="mp-progress-fill" id="mp-fill" style="width:0"><div class="mp-progress-thumb" id="mp-thumb"></div></div></div><span class="mp-time" id="mp-dur">0:00</span>' +
      '<div class="mp-vol-wrap"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polygon points="11 5 6 9 2 9 2 15 6 15 11 19 11 5"/></svg><div class="mp-vol" id="mp-vol"><div class="mp-vol-fill" style="width:100%"></div></div></div></div></div>';
    setTimeout(function() { setupAudio(item); }, 0);
  } else if (ft === 'video') {
    body = '<div class="mp mp--video" id="mp-video"><div class="mp-viewport" id="mp-viewport"><div class="mp-play-overlay" id="mp-play-big"><svg viewBox="0 0 24 24" fill="currentColor"><polygon points="5 3 19 12 5 21 5 3"/></svg></div></div>' +
      '<div class="mp-controls"><button class="mp-play-btn" id="mp-play"><svg viewBox="0 0 24 24" fill="currentColor"><polygon points="5 3 19 12 5 21 5 3"/></svg></button>' +
      '<span class="mp-time" id="mp-time">0:00</span><div class="mp-progress" id="mp-progress"><div class="mp-progress-fill" id="mp-fill" style="width:0"><div class="mp-progress-thumb" id="mp-thumb"></div></div></div><span class="mp-time" id="mp-dur">0:00</span>' +
      '<button class="mp-fs-btn" id="mp-fs"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" width="14" height="14"><polyline points="15 3 21 3 21 9"/><polyline points="9 21 3 21 3 15"/><line x1="21" y1="3" x2="14" y2="10"/><line x1="3" y1="21" x2="10" y2="14"/></svg></button></div></div>';
    setTimeout(function() { setupVideo(item); }, 0);
  } else {
    body = '<div class="preview-generic"><div class="preview-generic-icon">' + IC.file + '</div><div class="preview-generic-name">' + name + '</div>' +
      '<div class="preview-generic-meta">' + h(item.type || 'unknown') + ' \u00b7 ' + fmtSize(item.size) + '</div>' +
      '<button class="d-filter" style="margin-top:16px" onclick="fbDownload(\'' + qe(item._fullPath) + '\')">' + IC.download + ' Download</button></div>';
  }

  // Sibling navigation
  var siblings = S.items.filter(function(f) { return f.type !== 'directory'; });
  var idx = -1; siblings.forEach(function(f, i) { if (f._fullPath === item._fullPath) idx = i; });
  var prev = idx > 0 ? siblings[idx - 1] : null;
  var next = idx < siblings.length - 1 ? siblings[idx + 1] : null;
  var sibCount = siblings.length > 1 ? ' <span class="pv-count">' + (idx + 1) + ' / ' + siblings.length + '</span>' : '';

  // Md toggle
  var mdToggle = '';
  if (ft === 'markdown') {
    var isP = S.mdView === 'preview';
    mdToggle = '<div class="pv-md-toggle">' +
      '<button class="pv-md-btn' + (isP ? ' pv-md-btn--active' : '') + '" onclick="setMdView(\'preview\')" title="Preview">Preview</button>' +
      '<button class="pv-md-btn' + (!isP ? ' pv-md-btn--active' : '') + '" onclick="setMdView(\'code\')" title="Source">Source</button></div>';
  }

  // Crumbs
  var crumbs = '<button class="pv-back" onclick="closePreview()" title="Back">' + IC.arrowL + '</button>';
  crumbs += '<div class="pv-crumbs"><button class="pv-crumb" onclick="closePreview()">files</button>';
  if (S.path) {
    var segs = S.path.replace(/\/$/, '').split('/');
    var cur = '';
    segs.forEach(function(seg) {
      cur += seg + '/';
      crumbs += '<span class="pv-sep">/</span><button class="pv-crumb" onclick="fbNav(\'' + h(cur) + '\')">' + h(seg) + '</button>';
    });
  }
  crumbs += '<span class="pv-sep">/</span><span class="pv-crumb pv-crumb--current">' + name + '</span></div>';

  var nav = '<div class="pv-nav">' +
    '<button class="pv-nav-btn' + (prev ? '' : ' pv-nav-btn--disabled') + '"' + (prev ? ' onclick="previewNav(-1)"' : '') + '>' + IC.arrowL + '</button>' +
    sibCount +
    '<button class="pv-nav-btn' + (next ? '' : ' pv-nav-btn--disabled') + '"' + (next ? ' onclick="previewNav(1)"' : '') + '>' + IC.arrowR + '</button></div>';

  var copyBtn = '';
  if (S.previewContent !== null) {
    copyBtn = '<button class="fb-act" onclick="copyPreview()" title="Copy content"><span>' + IC.copy + '</span></button>';
  }

  var acts = '<div class="pv-actions">' +
    copyBtn +
    '<button class="fb-act" onclick="fbDownload(\'' + qe(item._fullPath) + '\')" title="Download"><span>' + IC.download + '</span></button>' +
    '<button class="fb-act" onclick="fbShare(\'' + qe(item._fullPath) + '\')" title="Share"><span>' + IC.share + '</span></button>' +
    '</div>';

  $main.innerHTML = '<div class="pv">' +
    '<div class="pv-bar">' +
      '<div class="pv-bar-left">' + crumbs + '</div>' +
      '<div class="pv-bar-right">' + '<span class="pv-info">' + meta + '</span>' + mdToggle + nav + acts + '</div>' +
    '</div>' +
    '<div class="pv-body">' + body + '</div></div>';

  // Render math in markdown preview
  if (ft === 'markdown' && S.mdView === 'preview') {
    setTimeout(postRenderMath, 0);
  }
}

window.setMdView = function(v) { S.mdView = v; renderPreview(); };

window.copyPreview = function() {
  if (S.previewContent !== null) {
    navigator.clipboard.writeText(S.previewContent).then(function() { toast('Copied', 'ok'); });
  }
};

window.previewNav = function(dir) {
  var siblings = S.items.filter(function(f) { return f.type !== 'directory'; });
  var idx = -1; siblings.forEach(function(f, i) { if (S.previewItem && f._fullPath === S.previewItem._fullPath) idx = i; });
  var next = siblings[idx + dir];
  if (next) openPreview(next._fullPath, next);
};

/* ── Media players ────────────────────────────────────────────────── */
function setupAudio(item) {
  var el = $('mp-audio'); if (!el) return;
  var audio = new Audio();
  resolveFileUrl(item._fullPath).then(function(src) { audio.src = src; });
  var play = $('mp-play'), time = $('mp-time'), dur = $('mp-dur'), prog = $('mp-progress'), fill = $('mp-fill'), wave = $('mp-wave');
  var bars = wave ? wave.children : [];
  function fmtT(s) { if (isNaN(s)) return '0:00'; var m = Math.floor(s / 60), ss = Math.floor(s % 60); return m + ':' + (ss < 10 ? '0' : '') + ss; }
  function updateWave() {
    if (!bars.length) return;
    var pct = audio.duration ? audio.currentTime / audio.duration : 0;
    for (var i = 0; i < bars.length; i++) {
      bars[i].style.height = (20 + Math.sin(i * 0.3 + audio.currentTime * 2) * 18) + 'px';
      bars[i].className = i / bars.length <= pct ? 'mp-wave-bar active' : 'mp-wave-bar';
    }
    if (!audio.paused) requestAnimationFrame(updateWave);
  }
  if (play) play.onclick = function() { audio.paused ? audio.play() : audio.pause(); };
  audio.onplay = function() { if (play) play.innerHTML = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="6" y1="4" x2="6" y2="20"/><line x1="18" y1="4" x2="18" y2="20"/></svg>'; updateWave(); };
  audio.onpause = function() { if (play) play.innerHTML = '<svg viewBox="0 0 24 24" fill="currentColor"><polygon points="5 3 19 12 5 21 5 3"/></svg>'; };
  audio.onloadedmetadata = function() { if (dur) dur.textContent = fmtT(audio.duration); };
  audio.ontimeupdate = function() { if (time) time.textContent = fmtT(audio.currentTime); if (fill && audio.duration) fill.style.width = (audio.currentTime / audio.duration * 100) + '%'; };
  if (prog) prog.onclick = function(e) { var r = prog.getBoundingClientRect(); audio.currentTime = ((e.clientX - r.left) / r.width) * audio.duration; };
  var vol = $('mp-vol');
  if (vol) vol.onclick = function(e) { var r = vol.getBoundingClientRect(); audio.volume = Math.max(0, Math.min(1, (e.clientX - r.left) / r.width)); vol.querySelector('.mp-vol-fill').style.width = (audio.volume * 100) + '%'; };
  S._mediaCleanup = function() { audio.pause(); audio.src = ''; };
}

function setupVideo(item) {
  var viewport = $('mp-viewport'); if (!viewport) return;
  var video = document.createElement('video'); video.preload = 'metadata';
  resolveFileUrl(item._fullPath).then(function(src) { video.src = src; });
  var overlay = viewport.querySelector('.mp-play-overlay');
  if (overlay) viewport.insertBefore(video, overlay);
  else viewport.appendChild(video);
  var play = $('mp-play'), playBig = $('mp-play-big'), time = $('mp-time'), dur = $('mp-dur'), prog = $('mp-progress'), fill = $('mp-fill'), fs = $('mp-fs');
  function fmtT(s) { if (isNaN(s)) return '0:00'; var m = Math.floor(s / 60), ss = Math.floor(s % 60); return m + ':' + (ss < 10 ? '0' : '') + ss; }
  if (play) play.onclick = function() { video.paused ? video.play() : video.pause(); };
  if (playBig) playBig.onclick = function() { video.play(); playBig.style.display = 'none'; };
  video.onplay = function() { if (play) play.innerHTML = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="6" y1="4" x2="6" y2="20"/><line x1="18" y1="4" x2="18" y2="20"/></svg>'; if (playBig) playBig.style.display = 'none'; };
  video.onpause = function() { if (play) play.innerHTML = '<svg viewBox="0 0 24 24" fill="currentColor"><polygon points="5 3 19 12 5 21 5 3"/></svg>'; if (playBig) playBig.style.display = 'flex'; };
  video.onloadedmetadata = function() { if (dur) dur.textContent = fmtT(video.duration); };
  video.ontimeupdate = function() { if (time) time.textContent = fmtT(video.currentTime); if (fill && video.duration) fill.style.width = (video.currentTime / video.duration * 100) + '%'; };
  if (prog) prog.onclick = function(e) { var r = prog.getBoundingClientRect(); video.currentTime = ((e.clientX - r.left) / r.width) * video.duration; };
  if (fs) fs.onclick = function() { if (viewport.requestFullscreen) viewport.requestFullscreen(); else if (viewport.webkitRequestFullscreen) viewport.webkitRequestFullscreen(); };
  S._mediaCleanup = function() { video.pause(); video.src = ''; };
}

/* ══════════════════════════════════════════════════════════════════════
   EVENTS
   ══════════════════════════════════════════════════════════════════════ */
var eventsData = [];
var eventsFilter = '';
var eventsHasMore = false;

sections.events = function() {
  eventsData = [];
  eventsFilter = '';
  eventsHasMore = false;
  renderEvents();
};

async function renderEvents() {
  $main.innerHTML =
    '<div class="dash-header"><div class="dash-title">Events</div><div class="dash-subtitle">File mutation log \u2014 writes, moves, deletes</div></div>' +
    '<div class="d-toolbar">' +
      '<button class="d-filter active" onclick="evFilter(\'\')" data-ev-filter="">All</button>' +
      '<button class="d-filter" onclick="evFilter(\'write\')" data-ev-filter="write">Write</button>' +
      '<button class="d-filter" onclick="evFilter(\'move\')" data-ev-filter="move">Move</button>' +
      '<button class="d-filter" onclick="evFilter(\'delete\')" data-ev-filter="delete">Delete</button>' +
    '</div>' +
    '<div id="ev-table"><div class="dash-loading"><span class="spinner"></span></div></div>';

  await loadEvents(true);
}

async function loadEvents(reset) {
  if (reset) { eventsData = []; eventsHasMore = false; }
  var limit = 100;
  try {
    var url = '/files/log?limit=' + limit;
    if (!reset && eventsData.length) {
      var lastTx = eventsData[eventsData.length - 1].tx;
      url += '&before_tx=' + lastTx;
    }
    var data = await api(url);
    var newEvents = data.events || [];
    if (reset) eventsData = newEvents;
    else eventsData = eventsData.concat(newEvents);
    eventsHasMore = newEvents.length >= limit;
    renderEventsTable();
  } catch (e) {
    var el = $('ev-table');
    if (el) el.innerHTML = '<div class="fb-empty">Failed to load events: ' + h(e.message) + '</div>';
  }
}

function renderEventsTable() {
  var el = $('ev-table');
  if (!el) return;
  var filtered = eventsFilter ? eventsData.filter(function(ev) { return ev.action === eventsFilter; }) : eventsData;

  if (filtered.length === 0) {
    el.innerHTML = '<div class="fb-empty">No events' + (eventsFilter ? ' for "' + h(eventsFilter) + '"' : '') + '</div>';
    return;
  }

  var html = '<div class="ev-list">';
  filtered.forEach(function(ev) {
    var msg = ev.msg || '';
    var detail = fmtSize(ev.size);
    if (msg) detail += ' \u00b7 ' + msg;
    html += '<div class="ev-row">' +
      '<span class="ev-tx">#' + ev.tx + '</span>' +
      '<span class="ev-badge">' + actionBadge(ev.action) + '</span>' +
      '<span class="ev-path" title="' + h(ev.path) + '" style="cursor:pointer" onclick="evClickPath(\'' + qe(ev.path) + '\')">' + h(ev.path) + '</span>' +
      '<span class="ev-detail"><span class="ev-msg" title="' + h(detail) + '">' + h(detail) + '</span></span>' +
      '<span class="ev-time">' + fmtRel(ev.ts) + '</span></div>';
  });
  html += '</div>';
  if (eventsHasMore && !eventsFilter) {
    html += '<button class="load-more" onclick="loadMoreEvents()">Load more</button>';
  }
  el.innerHTML = html;
}

window.evFilter = function(f) {
  eventsFilter = f;
  document.querySelectorAll('[data-ev-filter]').forEach(function(btn) {
    btn.classList.toggle('active', btn.getAttribute('data-ev-filter') === f);
  });
  renderEventsTable();
};

window.loadMoreEvents = function() { loadEvents(false); };

window.evClickPath = function(path) {
  if (!path) return;
  var parent = path.replace(/[^/]+$/, '');
  S.path = parent;
  S.searchQ = '';
  go('files');
  setTimeout(function() { openPreview(path); }, 200);
};

/* ══════════════════════════════════════════════════════════════════════
   AUDIT LOG
   ══════════════════════════════════════════════════════════════════════ */
var auditEntries = [];
var auditOffset = 0;
var auditFilter = '';
var auditTotal = 0;

sections.audit = function() {
  auditEntries = [];
  auditOffset = 0;
  auditFilter = '';
  auditTotal = 0;
  renderAudit();
};

async function renderAudit() {
  $main.innerHTML =
    '<div class="dash-header"><div class="dash-title">Audit Log</div><div class="dash-subtitle">All API actions \u2014 90 day retention</div></div>' +
    '<div class="d-toolbar">' +
      '<button class="d-filter active" onclick="auditFilterFn(\'\')" data-au-filter="">All</button>' +
      '<button class="d-filter" onclick="auditFilterFn(\'read\')" data-au-filter="read">Read</button>' +
      '<button class="d-filter" onclick="auditFilterFn(\'write\')" data-au-filter="write">Write</button>' +
      '<button class="d-filter" onclick="auditFilterFn(\'rm\')" data-au-filter="rm">Delete</button>' +
      '<button class="d-filter" onclick="auditFilterFn(\'share\')" data-au-filter="share">Share</button>' +
      '<button class="d-filter" onclick="auditFilterFn(\'login\')" data-au-filter="login">Login</button>' +
    '</div>' +
    '<div id="au-table"><div class="dash-loading"><span class="spinner"></span></div></div>';

  await loadAudit(true);
}

async function loadAudit(reset) {
  if (reset) { auditEntries = []; auditOffset = 0; }
  try {
    var url = '/dashboard/audit?limit=50&offset=' + auditOffset;
    if (auditFilter) url += '&action=' + encodeURIComponent(auditFilter);
    var data = await api(url);
    auditEntries = auditEntries.concat(data.entries);
    auditTotal = data.total;
    auditOffset = auditEntries.length;
    renderAuditTable();
  } catch (e) {
    var el = $('au-table');
    if (el) el.innerHTML = '<div class="fb-empty">Failed to load audit log: ' + h(e.message) + '</div>';
  }
}

function renderAuditTable() {
  var el = $('au-table');
  if (!el) return;
  if (auditEntries.length === 0) {
    el.innerHTML = '<div class="fb-empty">No audit entries' + (auditFilter ? ' for "' + h(auditFilter) + '"' : '') + '</div>';
    return;
  }
  var html = '<table class="d-table"><thead><tr><th>Action</th><th>Path</th><th>IP</th><th>Time</th></tr></thead><tbody>';
  auditEntries.forEach(function(e) {
    html += '<tr>' +
      '<td>' + auditBadge(e.action) + '</td>' +
      '<td title="' + h(e.path || '') + '">' + (e.path ? h(truncPath(e.path)) : '<span style="color:var(--text-3)">\u2014</span>') + '</td>' +
      '<td>' + (e.ip ? h(e.ip) : '<span style="color:var(--text-3)">\u2014</span>') + '</td>' +
      '<td>' + fmtRel(e.ts) + '</td></tr>';
  });
  html += '</tbody></table>';
  if (auditEntries.length < auditTotal) {
    html += '<button class="load-more" onclick="loadMoreAudit()">Load more (' + (auditTotal - auditEntries.length) + ' remaining)</button>';
  }
  el.innerHTML = html;
}

window.loadMoreAudit = function() { loadAudit(false); };

window.auditFilterFn = function(f) {
  auditFilter = f;
  document.querySelectorAll('[data-au-filter]').forEach(function(btn) {
    btn.classList.toggle('active', btn.getAttribute('data-au-filter') === f);
  });
  loadAudit(true);
};

/* ══════════════════════════════════════════════════════════════════════
   API KEYS
   ══════════════════════════════════════════════════════════════════════ */
var keysData = [];
var revealedToken = null;

sections.keys = function() {
  keysData = [];
  revealedToken = null;
  renderKeys();
};

async function renderKeys() {
  $main.innerHTML =
    '<div class="dash-header"><div class="dash-title">API Keys</div><div class="dash-subtitle">Create and manage API tokens for programmatic access</div></div>' +
    '<div id="key-reveal"></div>' +
    '<div class="key-form">' +
      '<div class="key-field"><label>Name</label><input id="key-name" placeholder="my-bot"></div>' +
      '<div class="key-field"><label>Path Prefix</label><input id="key-prefix" placeholder="docs/ (optional)"></div>' +
      '<div class="key-field"><label>Expires</label><select id="key-expiry">' +
        '<option value="">Never</option>' +
        '<option value="3600">1 hour</option>' +
        '<option value="86400">1 day</option>' +
        '<option value="604800">7 days</option>' +
        '<option value="2592000">30 days</option>' +
        '<option value="7776000">90 days</option>' +
      '</select></div>' +
      '<button class="key-create-btn" onclick="createKey()">Create Key</button>' +
    '</div>' +
    '<div id="key-table"><div class="dash-loading"><span class="spinner"></span></div></div>';

  if (revealedToken) renderTokenReveal();
  await loadKeys();
}

async function loadKeys() {
  try {
    var data = await api('/auth/keys');
    keysData = data.keys;
    renderKeysTable();
  } catch (e) {
    $('key-table').innerHTML = '<div class="fb-empty">Failed to load keys: ' + h(e.message) + '</div>';
  }
}

function renderKeysTable() {
  var el = $('key-table');
  if (!el) return;
  if (keysData.length === 0) {
    el.innerHTML = '<div class="fb-empty">No API keys. Create one above.</div>';
    return;
  }
  var html = '<table class="d-table"><thead><tr><th>Name</th><th>Prefix</th><th>Created</th><th>Expires</th><th>Last Used</th><th></th></tr></thead><tbody>';
  keysData.forEach(function(k) {
    var lastUsed = k.last_accessed ? fmtRel(k.last_accessed) : '<span style="color:var(--text-3)">never</span>';
    html += '<tr>' +
      '<td>' + (k.name ? h(k.name) : '<span style="color:var(--text-3)">unnamed</span>') + '</td>' +
      '<td>' + (k.prefix ? '<code>' + h(k.prefix) + '</code>' : '<span style="color:var(--text-3)">all</span>') + '</td>' +
      '<td>' + fmtRel(k.created_at) + '</td>' +
      '<td>' + (k.expires_at ? fmtDate(k.expires_at) : '<span style="color:var(--text-3)">never</span>') + '</td>' +
      '<td>' + lastUsed + '</td>' +
      '<td><button class="fb-act fb-act--danger" title="Revoke" onclick="deleteKey(\'' + h(k.id) + '\',\'' + h(k.name || k.id) + '\')"><span>' + IC.trash + '</span></button></td></tr>';
  });
  html += '</tbody></table>';
  el.innerHTML = html;
}

function renderTokenReveal() {
  var el = $('key-reveal');
  if (!el || !revealedToken) return;
  el.innerHTML = '<div class="key-reveal">' +
    '<div class="key-reveal-title">API Key Created</div>' +
    '<div class="key-reveal-token"><code>' + h(revealedToken) + '</code>' +
      '<button class="fb-act" title="Copy" onclick="copyToken()"><span>' + IC.copy + '</span></button></div>' +
    '<div class="key-reveal-warn">Store this token securely. It will not be shown again.</div></div>';
}

window.createKey = async function() {
  var name = $('key-name').value.trim();
  var prefix = $('key-prefix').value.trim();
  var expiry = $('key-expiry').value;
  var body = {};
  if (name) body.name = name;
  if (prefix) body.prefix = prefix;
  if (expiry) body.expires_in = parseInt(expiry, 10);

  var btn = document.querySelector('.key-create-btn');
  if (btn) { btn.disabled = true; btn.textContent = 'Creating\u2026'; }
  try {
    var data = await api('/auth/keys', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    revealedToken = data.token;
    toast('API key created', 'ok');
    $('key-name').value = '';
    $('key-prefix').value = '';
    $('key-expiry').value = '';
    renderTokenReveal();
    loadKeys();
  } catch (e) {
    toast('Failed: ' + e.message, 'err');
  } finally {
    if (btn) { btn.disabled = false; btn.textContent = 'Create Key'; }
  }
};

window.deleteKey = function(id, name) {
  showModal('Revoke API Key', 'Revoke key <code>' + h(name) + '</code>? Any services using this key will lose access immediately.', 'Revoke', async function() {
    try {
      await api('/auth/keys/' + id, { method: 'DELETE' });
      toast('Key revoked', 'ok');
      loadKeys();
    } catch (e) {
      toast('Failed: ' + e.message, 'err');
    }
  });
};

window.copyToken = function() {
  if (revealedToken) {
    navigator.clipboard.writeText(revealedToken).then(function() { toast('Token copied', 'ok'); });
  }
};

/* ══════════════════════════════════════════════════════════════════════
   SHARES
   ══════════════════════════════════════════════════════════════════════ */
var sharesData = [];

sections.shares = function() {
  sharesData = [];
  renderShares();
};

async function renderShares() {
  $main.innerHTML =
    '<div class="dash-header"><div class="dash-title">Shares</div><div class="dash-subtitle">Manage shared file links</div></div>' +
    '<div class="d-toolbar">' +
      '<button class="d-filter" onclick="createShareDialog()" style="font-weight:600;border-color:var(--text);color:var(--text)">' + IC.link + ' Share</button>' +
    '</div>' +
    '<div id="shares-list"></div>';

  try {
    var data = await api('/dashboard/shares');
    sharesData = data.shares || [];
    renderSharesList();
  } catch (e) {
    $('shares-list').innerHTML = '<div class="fb-empty">Failed to load shares: ' + h(e.message) + '</div>';
  }
}

function renderSharesList() {
  var el = $('shares-list');
  if (!el) return;
  if (sharesData.length === 0) {
    el.innerHTML = '<div class="fb-empty">No active shares. Share a file from the Files section or create one above.</div>';
    return;
  }
  var origin = location.origin;
  var html = '';
  sharesData.forEach(function(s) {
    var url = origin + '/s/' + s.token;
    var expiresIn = s.expires_at - Date.now();
    var expiresText = expiresIn > 86400000 ? Math.floor(expiresIn / 86400000) + 'd left' :
      expiresIn > 3600000 ? Math.floor(expiresIn / 3600000) + 'h left' :
      Math.floor(expiresIn / 60000) + 'm left';
    html += '<div class="share-card">' +
      '<span class="share-path" title="' + h(s.path) + '">' + h(s.path) + '</span>' +
      '<span class="share-meta">' +
        '<span class="share-views">' + (s.views || 0) + ' views</span>' +
        '<span>' + h(expiresText) + '</span>' +
        '<a class="share-link" href="' + h(url) + '" target="_blank" title="' + h(url) + '">' + h(url.replace(origin, '')) + '</a>' +
        '<button class="fb-act" title="Copy link" onclick="event.stopPropagation();navigator.clipboard.writeText(\'' + qe(url) + '\').then(function(){toast(\'Copied\',\'ok\')})"><span>' + IC.copy + '</span></button>' +
        '<button class="fb-act fb-act--danger" title="Revoke" onclick="event.stopPropagation();revokeShare(\'' + qe(s.token) + '\',\'' + qe(s.path) + '\')"><span>' + IC.trash + '</span></button>' +
      '</span></div>';
  });
  el.innerHTML = html;
}

window.createShareDialog = function() {
  var bg = document.createElement('div');
  bg.className = 'modal-bg';
  bg.innerHTML = '<div class="modal-box">' +
    '<div class="modal-title">Create Share</div>' +
    '<div class="modal-text">Enter the file path to share.</div>' +
    '<input class="modal-input" id="share-path-input" placeholder="e.g. docs/report.pdf" autocomplete="off">' +
    '<div id="share-suggestions" style="max-height:160px;overflow-y:auto;margin:-12px 0 16px"></div>' +
    '<div class="modal-actions">' +
      '<button class="modal-cancel" onclick="this.closest(\'.modal-bg\').remove()">Cancel</button>' +
      '<button class="modal-confirm--primary" id="share-path-ok">Next</button>' +
    '</div></div>';
  document.body.appendChild(bg);
  bg.addEventListener('click', function(e) { if (e.target === bg) bg.remove(); });
  var inp = bg.querySelector('#share-path-input');
  inp.focus();
  var timer;
  inp.addEventListener('input', function() {
    clearTimeout(timer);
    var q = inp.value.trim();
    timer = setTimeout(function() {
      if (!q) { bg.querySelector('#share-suggestions').innerHTML = ''; return; }
      api('/files/search?q=' + encodeURIComponent(q) + '&limit=6').then(function(d) {
        var sg = bg.querySelector('#share-suggestions');
        if (!sg) return;
        sg.innerHTML = (d.results || []).filter(function(r) { return !r.path.endsWith('/'); }).map(function(r) {
          return '<div style="padding:6px 12px;font-family:JetBrains Mono,monospace;font-size:11px;color:var(--text-2);cursor:pointer;border-bottom:1px solid var(--border)" onclick="this.closest(\'.modal-bg\').querySelector(\'#share-path-input\').value=\'' + qe(r.path) + '\';this.parentNode.innerHTML=\'\'">' + h(r.path) + '</div>';
        }).join('');
      }).catch(function() {});
    }, 200);
  });
  inp.addEventListener('keydown', function(e) { if (e.key === 'Enter') bg.querySelector('#share-path-ok').click(); });
  bg.querySelector('#share-path-ok').onclick = function() {
    var path = inp.value.trim();
    if (!path) return;
    bg.remove();
    showShareTTLModal(path);
  };
};

function showShareTTLModal(path) {
  var bg = document.createElement('div');
  bg.className = 'modal-bg';
  bg.innerHTML = '<div class="modal-box">' +
    '<div class="modal-title">Share: ' + h(path) + '</div>' +
    '<div class="modal-text">Select link expiration:</div>' +
    '<div class="key-field" style="margin-bottom:16px"><select id="share-ttl">' +
      '<option value="3600">1 hour</option>' +
      '<option value="86400" selected>1 day</option>' +
      '<option value="604800">7 days</option>' +
      '<option value="2592000">30 days</option>' +
    '</select></div>' +
    '<div class="modal-actions">' +
      '<button class="modal-cancel" onclick="this.closest(\'.modal-bg\').remove()">Cancel</button>' +
      '<button class="key-create-btn" id="share-confirm">Create Link</button>' +
    '</div></div>';
  document.body.appendChild(bg);
  bg.addEventListener('click', function(e) { if (e.target === bg) bg.remove(); });
  bg.querySelector('#share-confirm').onclick = async function() {
    var ttl = parseInt(bg.querySelector('#share-ttl').value, 10);
    bg.remove();
    try {
      var data = await api('/files/share', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ path: path, ttl: ttl }),
      });
      var url = data.url;
      navigator.clipboard.writeText(url).then(function() { toast('Link copied', 'ok'); });
      toast('Share created', 'ok');
      if (S.section === 'shares') renderShares();
    } catch (e) {
      toast('Share failed: ' + e.message, 'err');
    }
  };
}

window.revokeShare = function(token, path) {
  showModal('Revoke Share', 'Revoke share link for <code>' + h(path) + '</code>? The link will stop working immediately.', 'Revoke', async function() {
    try {
      await api('/dashboard/shares/' + token, { method: 'DELETE' });
      toast('Share revoked', 'ok');
      renderShares();
    } catch (e) {
      toast('Failed: ' + e.message, 'err');
    }
  });
};

/* ══════════════════════════════════════════════════════════════════════
   SETTINGS
   ══════════════════════════════════════════════════════════════════════ */
sections.settings = async function() {
  $main.innerHTML =
    '<div class="dash-header"><div class="dash-title">Settings</div><div class="dash-subtitle">Account information</div></div>' +
    '<div id="settings-content"><div class="dash-loading"><span class="spinner"></span></div></div>';

  try {
    var acct = await api('/dashboard/account');
    var el = $('settings-content');
    if (!el) return;

    var html = '<div class="settings-section">' +
      '<div class="settings-label">Account</div>' +
      '<div class="settings-row"><span class="settings-key">Actor</span><span class="settings-val">' + h(acct.actor) + '</span></div>' +
      '<div class="settings-row"><span class="settings-key">Email</span><span class="settings-val">' + (acct.email ? h(acct.email) : '<span style="color:var(--text-3)">not set</span>') + '</span></div>' +
      '<div class="settings-row"><span class="settings-key">Created</span><span class="settings-val">' + (acct.created_at ? fmtDate(acct.created_at) : '\u2014') + '</span></div>' +
      '<div class="settings-row"><span class="settings-key">Sessions</span><span class="settings-val">' + acct.active_sessions + ' active</span></div>' +
    '</div>';

    html += '<div class="settings-section">' +
      '<div class="settings-label">Actions</div>' +
      '<button class="settings-danger" onclick="signOut()">Sign Out</button></div>';

    el.innerHTML = html;
  } catch (e) {
    $('settings-content').innerHTML = '<div class="fb-empty">Failed to load account: ' + h(e.message) + '</div>';
  }
};

window.signOut = function() {
  showModal('Sign Out', 'Are you sure you want to sign out?', 'Sign Out', function() {
    window.location.href = '/auth/logout';
  });
};

/* ══════════════════════════════════════════════════════════════════════
   KEYBOARD SHORTCUTS
   ══════════════════════════════════════════════════════════════════════ */
var gPending = false;

function showShortcuts() {
  var items = [
    ['/', 'Search'],
    ['Cmd+K', 'Command palette'],
    ['Esc', 'Close / Go back'],
    ['\u2190 \u2192', 'Prev / Next file'],
    ['Backspace', 'Parent folder'],
    ['?', 'Shortcuts'],
    ['g o', 'Go to Overview'],
    ['g f', 'Go to Files'],
    ['g e', 'Go to Events'],
    ['g a', 'Go to Audit'],
    ['g h', 'Go to Shares'],
    ['g k', 'Go to Keys'],
    ['g s', 'Go to Settings'],
  ];
  var bodyHtml = '<div class="shortcuts">' + items.map(function(s) {
    return '<div class="shortcut"><span>' + s[1] + '</span><kbd>' + s[0] + '</kbd></div>';
  }).join('') + '</div>';
  showCustomModal('Keyboard Shortcuts', bodyHtml);
}

document.addEventListener('keydown', function(e) {
  var tag = e.target.tagName;
  if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return;

  // Command palette
  if (e.key === 'k' && (e.metaKey || e.ctrlKey)) { e.preventDefault(); openCmdPalette(); return; }

  // Preview navigation
  if (S.previewItem) {
    if (e.key === 'Escape') { closePreview(); return; }
    if (e.key === 'ArrowLeft') { previewNav(-1); return; }
    if (e.key === 'ArrowRight') { previewNav(1); return; }
  }

  // g + key combos
  if (gPending) {
    gPending = false;
    var map = { o: 'overview', f: 'files', e: 'events', a: 'audit', h: 'shares', k: 'keys', s: 'settings' };
    if (map[e.key]) { go(map[e.key]); return; }
  }
  if (e.key === 'g') { gPending = true; setTimeout(function() { gPending = false; }, 500); return; }

  if (e.key === '/') { e.preventDefault(); var si = $('fb-search'); if (si) si.focus(); else openCmdPalette(); return; }
  if (e.key === '?') { showShortcuts(); return; }
  if (e.key === 'Escape') { closeModals(); closeCmdPalette(); hideCtx(); return; }
  if (e.key === 'Backspace' && S.section === 'files' && S.path && !S.previewItem) {
    e.preventDefault();
    var parent = S.path.replace(/[^/]+\/$/, '');
    fbNav(parent);
    return;
  }
});

/* ── Touch: long-press for context menu ───────────────────────────── */
var touchStart = null, touchTimer = null;
document.addEventListener('touchstart', function(e) {
  var row = e.target.closest('.fb-row[data-idx]');
  if (!row) return;
  var idx = parseInt(row.dataset.idx, 10);
  touchStart = { x: e.touches[0].clientX, y: e.touches[0].clientY, idx: idx };
  touchTimer = setTimeout(function() {
    if (!touchStart) return;
    var item = S.items[touchStart.idx];
    if (item) showCtx(touchStart.x, touchStart.y, item);
    touchStart = null;
  }, 500);
}, { passive: true });
document.addEventListener('touchmove', function(e) {
  if (!touchStart) return;
  var dx = e.touches[0].clientX - touchStart.x, dy = e.touches[0].clientY - touchStart.y;
  if (Math.abs(dx) > 10 || Math.abs(dy) > 10) { clearTimeout(touchTimer); touchTimer = null; }
}, { passive: true });
document.addEventListener('touchend', function() { clearTimeout(touchTimer); touchStart = null; }, { passive: true });

/* ══════════════════════════════════════════════════════════════════════
   INIT
   ══════════════════════════════════════════════════════════════════════ */
var initSection = parseHash();
go(initSection);

/* ══════════════════════════════════════════════════════════════════════
   ONBOARDING MODAL
   ══════════════════════════════════════════════════════════════════════ */
function showOnboardingModal() {
  var emailPrefix = (CFG.email || '').split('@')[0].replace(/[^a-z0-9-]/gi, '').toLowerCase().slice(0, 20) || '';
  var html = '<div class="modal-bg" id="onboard-modal" style="z-index:600">' +
    '<div class="modal-box onboard-modal-box">' +
    '<div style="font-size:18px;font-weight:700;margin-bottom:4px">Welcome to Storage</div>' +
    '<div style="font-size:13px;color:var(--text-2);margin-bottom:24px;line-height:1.6">Choose a username and display name to get started.</div>' +
    '<div class="onboard-field">' +
      '<label class="onboard-label">USERNAME</label>' +
      '<div class="onboard-input-wrap">' +
        '<span class="onboard-prefix">u/</span>' +
        '<input type="text" id="onboard-username" class="onboard-input" placeholder="your-username" maxlength="20" autocomplete="off" spellcheck="false">' +
      '</div>' +
      '<div class="onboard-hint" id="onboard-username-hint">3\u201320 characters. Lowercase, numbers, hyphens. Permanent.</div>' +
    '</div>' +
    '<div class="onboard-field">' +
      '<label class="onboard-label">DISPLAY NAME</label>' +
      '<input type="text" id="onboard-displayname" class="modal-input" style="margin-bottom:4px" placeholder="Your Name" maxlength="50" value="' + h(emailPrefix) + '">' +
      '<div class="onboard-hint">Shown in the UI. Can be changed later.</div>' +
    '</div>' +
    '<div class="onboard-error" id="onboard-error"></div>' +
    '<div class="modal-actions" style="margin-top:20px">' +
      '<button class="modal-confirm--primary" id="onboard-btn" onclick="submitOnboarding()">Continue</button>' +
    '</div>' +
    '</div></div>';
  document.body.insertAdjacentHTML('beforeend', html);

  var inp = document.getElementById('onboard-username');
  if (inp) {
    inp.addEventListener('input', function() {
      var v = this.value.toLowerCase().replace(/[^a-z0-9-]/g, '');
      this.value = v;
      validateOnboardUsername(v);
    });
    inp.focus();
  }
  var dn = document.getElementById('onboard-displayname');
  if (dn) dn.addEventListener('keydown', function(e) { if (e.key === 'Enter') submitOnboarding(); });
  if (inp) inp.addEventListener('keydown', function(e) { if (e.key === 'Enter') document.getElementById('onboard-displayname').focus(); });
}

function validateOnboardUsername(v) {
  var hint = document.getElementById('onboard-username-hint');
  var inp = document.getElementById('onboard-username');
  if (!v) { hint.textContent = '3\u201320 characters. Lowercase, numbers, hyphens. Permanent.'; hint.style.color = ''; inp.style.borderColor = ''; return false; }
  if (v.length < 3) { hint.textContent = 'At least 3 characters'; hint.style.color = 'var(--red)'; inp.style.borderColor = 'var(--red)'; return false; }
  if (!/^[a-z0-9][a-z0-9-]*[a-z0-9]$/.test(v) && v.length > 2) { hint.textContent = 'Cannot start or end with a hyphen'; hint.style.color = 'var(--red)'; inp.style.borderColor = 'var(--red)'; return false; }
  if (/--/.test(v)) { hint.textContent = 'No consecutive hyphens'; hint.style.color = 'var(--red)'; inp.style.borderColor = 'var(--red)'; return false; }
  hint.textContent = 'Looks good'; hint.style.color = 'var(--green,#22c55e)'; inp.style.borderColor = 'var(--green,#22c55e)';
  return true;
}

function submitOnboarding() {
  var username = (document.getElementById('onboard-username').value || '').trim().toLowerCase();
  var displayName = (document.getElementById('onboard-displayname').value || '').trim();
  var errEl = document.getElementById('onboard-error');
  var btn = document.getElementById('onboard-btn');

  if (!validateOnboardUsername(username)) { errEl.textContent = 'Please enter a valid username'; return; }
  if (!displayName) { errEl.textContent = 'Please enter a display name'; return; }

  errEl.textContent = '';
  btn.disabled = true;
  btn.textContent = 'Setting up...';

  fetch('/dashboard/profile', {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username: username, display_name: displayName })
  })
  .then(function(res) { return res.json().then(function(d) { return { ok: res.ok, data: d }; }); })
  .then(function(r) {
    if (!r.ok) {
      errEl.textContent = r.data.message || 'Something went wrong';
      btn.disabled = false;
      btn.textContent = 'Continue';
      return;
    }
    // Success — update config and UI
    CFG.displayName = displayName;
    CFG.needsOnboarding = false;
    var navUser = document.querySelector('.nav-user');
    if (navUser) navUser.textContent = displayName;
    var modal = document.getElementById('onboard-modal');
    if (modal) modal.remove();
    toast('Welcome, ' + displayName + '!', 'ok');
  })
  .catch(function() {
    errEl.textContent = 'Network error — please try again';
    btn.disabled = false;
    btn.textContent = 'Continue';
  });
}

if (CFG.needsOnboarding) {
  showOnboardingModal();
}
