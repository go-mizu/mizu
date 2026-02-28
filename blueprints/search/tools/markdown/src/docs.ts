export function renderDocs(contentHtml: string): string {
  return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Docs — URL → Markdown</title>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Geist:wght@300;400;500;600;700&family=Geist+Mono:wght@400;500&display=swap" rel="stylesheet">
  <link rel="stylesheet" href="/styles.css">
  <style>
  /* docs layout — no borders */
  .layout{display:flex;min-height:calc(100vh - 52px)}
  .sidebar{width:220px;flex-shrink:0;padding:32px 24px;position:sticky;top:0;height:calc(100vh - 52px);overflow-y:auto}
  .sidebar-title{font-family:var(--mono);font-size:11px;letter-spacing:.1em;text-transform:uppercase;color:var(--fg3);margin-bottom:16px}
  .sidebar a{display:block;font-size:14px;color:var(--fg3);padding:4px 0;transition:color .15s}
  .sidebar a:hover{color:var(--fg)}
  .sidebar a.on{color:var(--fg);font-weight:500}
  /* content — centered, generous width */
  .content{flex:1;padding:48px 64px;max-width:800px}
  @media(max-width:720px){.sidebar{display:none}.content{padding:32px 24px;max-width:100%}}
  /* rendered markdown styles */
  .content h1{font-size:36px;font-weight:700;letter-spacing:-.03em;margin-bottom:16px}
  .content h2{font-size:22px;font-weight:600;letter-spacing:-.02em;margin:48px 0 14px;scroll-margin-top:24px}
  .content h2:first-of-type{margin-top:0}
  .content h3{font-size:17px;font-weight:600;margin:28px 0 10px}
  .content p{font-size:16px;color:var(--fg2);line-height:1.75;margin-bottom:16px}
  .content ul,.content ol{padding-left:1.5em;margin-bottom:16px}
  .content li{font-size:16px;color:var(--fg2);margin:4px 0;line-height:1.7}
  .content strong{color:var(--fg);font-weight:600}
  .content a{color:var(--fg);text-decoration:underline;text-underline-offset:2px}
  .content pre{background:var(--code-bg);padding:20px 22px;overflow-x:auto;margin:20px 0;position:relative}
  .content pre code{font-family:var(--mono);font-size:13.5px;line-height:1.7;color:var(--code-fg);background:none;padding:0}
  .content table{width:100%;border-collapse:collapse;margin:16px 0;font-size:14px}
  .content th{text-align:left;padding:8px 14px;border-bottom:1px solid var(--border);font-weight:600}
  .content td{padding:8px 14px;border-bottom:1px solid var(--border2);color:var(--fg2)}
  .content td:first-child{font-family:var(--mono);font-size:12.5px;color:var(--fg)}
  /* copy buttons on pre blocks */
  .content pre .copy-btn{position:absolute;top:10px;right:10px;background:#222;border:1px solid #333;color:#aaa;font-family:var(--mono);font-size:11px;padding:4px 10px;cursor:pointer;transition:all .15s}
  .content pre .copy-btn:hover{background:#333;color:#fff}
  </style>
</head>
<body>
<header>
  <div class="hdr">
    <a href="/" class="logo">
      <span class="logo-sq">
        <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="white" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <rect width="16" height="12" x="4" y="6" rx="1"/>
          <path d="M4 12h2l2-2 2 2 2-4 2 4 2-2h2"/>
        </svg>
      </span>
      URL → Markdown
    </a>
    <nav>
      <a href="/">Home</a>
      <a href="/llms.txt">llms.txt</a>
      <a href="https://github.com/go-mizu/mizu">GitHub</a>
    </nav>
  </div>
</header>

<div class="layout">
  <nav class="sidebar" id="sidebar">
    <div class="sidebar-title">Docs</div>
    <!-- JS builds nav from h2 headings -->
  </nav>
  <div class="content" id="content">
    ${contentHtml}
  </div>
</div>

<script>
(function() {
  var headings = document.querySelectorAll('.content h2');
  var sidebar = document.getElementById('sidebar');
  headings.forEach(function(h) {
    if (!h.id) {
      h.id = h.textContent.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '');
    }
    var a = document.createElement('a');
    a.href = '#' + h.id;
    a.textContent = h.textContent;
    sidebar.appendChild(a);
  });

  // Add copy buttons to pre blocks
  document.querySelectorAll('.content pre').forEach(function(pre) {
    var btn = document.createElement('button');
    btn.className = 'copy-btn';
    btn.textContent = 'copy';
    btn.onclick = function() {
      var text = (pre.querySelector('code') || pre).innerText || '';
      navigator.clipboard.writeText(text.trim()).catch(function() {
        var ta = document.createElement('textarea');
        ta.style.position = 'fixed'; ta.style.top = '-9999px';
        ta.value = text.trim();
        document.body.appendChild(ta); ta.select();
        document.execCommand('copy'); document.body.removeChild(ta);
      });
      btn.textContent = 'copied!';
      setTimeout(function() { btn.textContent = 'copy'; }, 2000);
    };
    pre.appendChild(btn);
  });

  // Scroll-spy
  var links = [];
  document.querySelectorAll('.sidebar a[href^="#"]').forEach(function(l) { links.push(l); });
  var sections = document.querySelectorAll('.content h2[id]');
  window.addEventListener('scroll', function() {
    var pos = window.scrollY + 80;
    var active = null;
    sections.forEach(function(s) { if (s.getBoundingClientRect().top + window.scrollY <= pos) active = s; });
    links.forEach(function(l) {
      l.className = (active && l.getAttribute('href') === '#' + active.id) ? 'on' : '';
    });
  }, { passive: true });
})();
</script>
</body>
</html>`;
}
