export function renderDocs(contentHtml: string): string {
  return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Docs — markdown.go-mizu</title>
  <script>(function(){if(localStorage.getItem('theme')==='dark')document.documentElement.classList.add('dark');})();<\/script>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Geist:wght@300;400;500;600;700&family=Geist+Mono:wght@400;500&display=swap" rel="stylesheet">
  <link rel="stylesheet" href="/styles.css">
  <style>
  .doc-wrap{max-width:900px;margin:0 auto;padding:48px 32px 80px}
  @media(max-width:640px){.doc-wrap{padding:32px 20px 60px}}
  .doc-wrap h1{font-size:28px;font-weight:700;letter-spacing:-.03em;margin-bottom:12px;color:#18181b}
  .doc-wrap>p:first-of-type{font-size:16px;color:#71717a;margin-bottom:40px;line-height:1.7}
  .doc-wrap h2{font-size:20px;font-weight:600;letter-spacing:-.02em;margin:0;scroll-margin-top:24px;color:#18181b;padding:48px 0 14px}
  .doc-wrap h2:first-of-type{padding-top:0}
  .doc-wrap h3{font-size:16px;font-weight:600;margin:28px 0 10px;color:#18181b}
  .doc-wrap p{font-size:16px;color:#71717a;line-height:1.75;margin-bottom:14px}
  .doc-wrap ul,.doc-wrap ol{padding-left:1.5em;margin-bottom:16px}
  .doc-wrap li{font-size:16px;color:#71717a;margin:4px 0;line-height:1.7}
  .doc-wrap strong{color:#18181b;font-weight:600}
  .doc-wrap a{color:#18181b;text-decoration:underline;text-underline-offset:2px;text-decoration-color:#d4d4d8}
  .doc-wrap a:hover{text-decoration-color:#18181b}
  .doc-wrap code{font-family:'Geist Mono',ui-monospace,monospace;font-size:.75rem;background:#fafafa;padding:2px 6px;color:#18181b}
  .doc-wrap pre{background:#09090b;padding:20px 22px;overflow-x:auto;margin:16px 0;position:relative}
  .doc-wrap pre code{font-family:'Geist Mono',ui-monospace,monospace;font-size:13px;line-height:1.7;color:#e4e4e7;background:none;padding:0}
  .doc-wrap table{width:100%;border-collapse:collapse;margin:16px 0;font-size:14px}
  .doc-wrap th{text-align:left;padding:8px 14px;font-weight:600;color:#18181b}
  .doc-wrap td{padding:8px 14px;color:#71717a}
  .doc-wrap pre .copy-btn{position:absolute;top:10px;right:10px}
  .copy-btn{font-family:'Geist Mono',ui-monospace,monospace;font-size:11px;background:#27272a;color:#a1a1aa;border:none;padding:4px 8px;cursor:pointer}
  .copy-btn:hover{color:#f4f4f5}
  .dark .doc-wrap h1,.dark .doc-wrap h2,.dark .doc-wrap h3,.dark .doc-wrap strong{color:#f4f4f5}
  .dark .doc-wrap p,.dark .doc-wrap li{color:#a1a1aa}
  .dark .doc-wrap a{color:#f4f4f5;text-decoration-color:#3f3f46}
  .dark .doc-wrap a:hover{text-decoration-color:#f4f4f5}
  .dark .doc-wrap code{background:#18181b;color:#f4f4f5}
  .dark .doc-wrap th{color:#f4f4f5}
  .dark .doc-wrap td{color:#a1a1aa}
  </style>
</head>
<body class="font-sans bg-white dark:bg-zinc-950 text-zinc-900 dark:text-zinc-100 transition-colors duration-200">

<header class="sticky top-0 z-50 bg-white dark:bg-zinc-950 px-8 py-3.5 flex items-center justify-between transition-colors duration-200">
  <a href="/" class="flex items-center text-zinc-900 dark:text-zinc-100 no-underline flex-shrink-0">
    <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
      <rect x="3" y="3" width="18" height="18" rx="2"/>
      <path d="M8 16V8.5a.5.5 0 0 1 .9-.3l2.7 3.6a.5.5 0 0 0 .8 0l2.7-3.6a.5.5 0 0 1 .9.3V16"/>
    </svg>
  </a>
  <nav class="flex items-center gap-1 ml-auto">
    <a href="/" class="text-sm text-zinc-500 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-100 px-3 py-1.5 transition-colors no-underline">Home</a>
    <span class="text-sm text-zinc-900 dark:text-zinc-100 font-medium px-3 py-1.5">Docs</span>
    <a href="https://github.com/go-mizu/mizu" class="text-sm text-zinc-500 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-100 px-3 py-1.5 transition-colors no-underline">GitHub</a>
    <a href="/llms.txt" class="text-sm text-zinc-500 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-100 px-3 py-1.5 transition-colors no-underline">llms.txt</a>
    <button onclick="toggleTheme()" title="Toggle dark mode" class="text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-100 p-1.5 transition-colors ml-1 cursor-pointer bg-transparent border-none flex items-center">
      <svg class="dark:hidden" width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 3a6 6 0 0 0 9 9 9 9 0 1 1-9-9z"/></svg>
      <svg class="hidden dark:block" width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="4"/><path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M6.34 17.66l-1.41 1.41M19.07 4.93l-1.41 1.41"/></svg>
    </button>
  </nav>
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

function toggleTheme() {
  var isDark = document.documentElement.classList.toggle('dark');
  localStorage.setItem('theme', isDark ? 'dark' : 'light');
}
<\/script>
</body>
</html>`;
}
