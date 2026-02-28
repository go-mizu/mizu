export function renderDocs(contentHtml: string): string {
  return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Docs — markdown.go-mizu</title>
  <script>
  (function(){var t=localStorage.getItem('theme');if(t)document.documentElement.setAttribute('data-theme',t);})();
  <\/script>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Geist:wght@300;400;500;600;700&family=Geist+Mono:wght@400;500&display=swap" rel="stylesheet">
  <link rel="stylesheet" href="/styles.css">
  <style>
  .doc-wrap{max-width:720px;margin:0 auto;padding:48px 32px 80px}
  @media(max-width:640px){.doc-wrap{padding:32px 20px 60px}}
  .doc-wrap h1{font-size:32px;font-weight:700;letter-spacing:-.03em;margin-bottom:12px;color:var(--fg)}
  .doc-wrap>p:first-of-type{font-size:16px;color:var(--fg2);margin-bottom:40px;line-height:1.7}
  .doc-wrap h2{font-size:20px;font-weight:600;letter-spacing:-.02em;margin:0;scroll-margin-top:24px;color:var(--fg);padding:48px 0 14px;border-top:1px solid var(--border)}
  .doc-wrap h2:first-of-type{padding-top:0;border-top:none}
  .doc-wrap h3{font-size:16px;font-weight:600;margin:28px 0 10px;color:var(--fg)}
  .doc-wrap p{font-size:15px;color:var(--fg2);line-height:1.75;margin-bottom:14px}
  .doc-wrap ul,.doc-wrap ol{padding-left:1.5em;margin-bottom:16px}
  .doc-wrap li{font-size:15px;color:var(--fg2);margin:4px 0;line-height:1.7}
  .doc-wrap strong{color:var(--fg);font-weight:600}
  .doc-wrap a{color:var(--fg);text-decoration:underline;text-underline-offset:2px;text-decoration-color:var(--border2)}
  .doc-wrap a:hover{text-decoration-color:var(--fg)}
  .doc-wrap code{font-family:var(--mono);font-size:12px;background:var(--bg2);padding:2px 6px;color:var(--fg)}
  .doc-wrap pre{background:var(--code-bg);padding:20px 22px;overflow-x:auto;margin:16px 0;position:relative}
  .doc-wrap pre code{font-family:var(--mono);font-size:13px;line-height:1.7;color:var(--code-fg);background:none;padding:0}
  .doc-wrap table{width:100%;border-collapse:collapse;margin:16px 0;font-size:14px}
  .doc-wrap th{text-align:left;padding:8px 14px;border-bottom:1px solid var(--border);font-weight:600;color:var(--fg)}
  .doc-wrap td{padding:8px 14px;border-bottom:1px solid var(--border2);color:var(--fg2)}
  .doc-wrap pre .copy-btn{position:absolute;top:10px;right:10px;background:#222;border:1px solid #333;color:#aaa;font-family:var(--mono);font-size:11px;padding:4px 10px;cursor:pointer;transition:all .15s}
  .doc-wrap pre .copy-btn:hover{color:#fff}
  </style>
</head>
<body>
<header>
  <div class="hdr">
    <a href="/" class="logo">
      <span class="logo-sq">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="var(--bg)" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <rect x="3" y="3" width="18" height="18" rx="2"/>
          <path d="M8 16V8.5a.5.5 0 0 1 .9-.3l2.7 3.6a.5.5 0 0 0 .8 0l2.7-3.6a.5.5 0 0 1 .9.3V16"/>
        </svg>
      </span>
      markdown.go-mizu
    </a>
    <nav>
      <a href="/">Home</a>
      <a href="https://github.com/go-mizu/mizu">GitHub</a>
      <button class="theme-toggle" id="theme-toggle" onclick="toggleTheme()" title="Toggle dark mode">
        <svg id="icon-moon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 3a6 6 0 0 0 9 9 9 9 0 1 1-9-9z"/></svg>
        <svg id="icon-sun" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="display:none"><circle cx="12" cy="12" r="4"/><path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M6.34 17.66l-1.41 1.41M19.07 4.93l-1.41 1.41"/></svg>
      </button>
    </nav>
  </div>
</header>

<div class="doc-wrap">
  ${contentHtml}
</div>

<script>
(function() {
  document.querySelectorAll('.doc-wrap pre').forEach(function(pre) {
    var btn = document.createElement('button');
    btn.className = 'copy-btn';
    btn.textContent = 'copy';
    btn.onclick = function() {
      var code = pre.querySelector('code') || pre;
      var text = code.innerText || '';
      var self = btn;
      function done() { self.textContent = 'copied!'; setTimeout(function() { self.textContent = 'copy'; }, 2000); }
      if (navigator.clipboard) {
        navigator.clipboard.writeText(text.trim()).then(done).catch(function() {
          var ta = document.createElement('textarea');
          ta.style.position='fixed';ta.style.top='-9999px';ta.value=text.trim();
          document.body.appendChild(ta);ta.select();document.execCommand('copy');document.body.removeChild(ta);
          done();
        });
      } else {
        var ta = document.createElement('textarea');
        ta.style.position='fixed';ta.style.top='-9999px';ta.value=text.trim();
        document.body.appendChild(ta);ta.select();document.execCommand('copy');document.body.removeChild(ta);
        done();
      }
    };
    pre.appendChild(btn);
  });
})();

function updateToggleIcon() {
  var dark = document.documentElement.getAttribute('data-theme') === 'dark';
  document.getElementById('icon-moon').style.display = dark ? 'none' : '';
  document.getElementById('icon-sun').style.display = dark ? '' : 'none';
}

function toggleTheme() {
  var cur = document.documentElement.getAttribute('data-theme');
  var next = cur === 'dark' ? 'light' : 'dark';
  document.documentElement.setAttribute('data-theme', next);
  localStorage.setItem('theme', next);
  updateToggleIcon();
}

updateToggleIcon();
<\/script>
</body>
</html>`;
}
