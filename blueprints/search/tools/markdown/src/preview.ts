export function renderPreview(): string {
  return '<!DOCTYPE html>\n' +
'<html lang="en">\n' +
'<head>\n' +
'  <meta charset="UTF-8">\n' +
'  <meta name="viewport" content="width=device-width, initial-scale=1.0">\n' +
'  <title>Preview \u2014 markdown.go-mizu</title>\n' +
'  <link rel="preconnect" href="https://fonts.googleapis.com">\n' +
'  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>\n' +
'  <link href="https://fonts.googleapis.com/css2?family=Geist:wght@300;400;500;600;700&family=Geist+Mono:wght@400;500&display=swap" rel="stylesheet">\n' +
'  <script src="https://cdn.jsdelivr.net/npm/marked@15/marked.min.js"><\/script>\n' +
'  <script src="https://cdn.jsdelivr.net/npm/dompurify@3/dist/purify.min.js"><\/script>\n' +
'  <style>\n' +
'*,*::before,*::after{box-sizing:border-box;margin:0;padding:0}\n' +
':root{\n' +
'  --sans:\'Geist\',-apple-system,sans-serif;\n' +
'  --mono:\'Geist Mono\',ui-monospace,monospace;\n' +
'  --bg:#fff;--fg:#0a0a0a;--fg2:#555;--fg3:#999;\n' +
'  --border:#e8e8e8;--border2:#e0e0e0;\n' +
'  --code-bg:#111;--code-fg:#e4e4e7;\n' +
'  --w:1120px\n' +
'}\n' +
'html{font-size:15px}\n' +
'body{font-family:var(--sans);background:var(--bg);color:var(--fg);line-height:1.6;-webkit-font-smoothing:antialiased}\n' +
'a{color:inherit;text-decoration:none}\n' +
'\n' +
'/* header */\n' +
'header{padding:14px 32px}\n' +
'.hdr{display:flex;align-items:center;justify-content:space-between}\n' +
'.logo{font-size:13.5px;font-weight:500;letter-spacing:-.02em;display:flex;align-items:center;gap:8px;color:var(--fg)}\n' +
'.logo-sq{width:22px;height:22px;background:var(--fg);display:flex;align-items:center;justify-content:center;flex-shrink:0}\n' +
'nav a{font-size:13px;color:var(--fg3);margin-left:20px;transition:color .15s}\n' +
'nav a:hover{color:var(--fg)}\n' +
'\n' +
'/* url bar */\n' +
'.url-bar{width:100%;border-bottom:1px solid var(--border2);padding:20px 32px}\n' +
'.url-bar-inner{max-width:var(--w);margin:0 auto;display:flex;gap:0}\n' +
'.url-in{flex:1;font-family:var(--mono);font-size:13px;padding:11px 14px;border:1px solid var(--border);border-right:none;background:#fff;color:var(--fg);outline:none;transition:border-color .15s;min-width:0}\n' +
'.url-in::placeholder{color:var(--fg3)}\n' +
'.url-in:focus{border-color:var(--fg)}\n' +
'.cvt-btn{font-family:var(--sans);font-size:13px;font-weight:500;padding:11px 20px;background:var(--fg);color:#fff;border:1px solid var(--fg);cursor:pointer;white-space:nowrap;transition:background .15s}\n' +
'.cvt-btn:hover:not(:disabled){background:#333;border-color:#333}\n' +
'.cvt-btn:disabled{opacity:.6;cursor:default}\n' +
'\n' +
'/* states */\n' +
'.loading-state{display:none;padding:80px 32px;text-align:center}\n' +
'.spinner{display:inline-block;width:22px;height:22px;border:2px solid var(--border2);border-top-color:var(--fg);animation:spin .7s linear infinite;margin-right:10px;vertical-align:middle}\n' +
'@keyframes spin{to{transform:rotate(360deg)}}\n' +
'.loading-txt{font-size:14px;color:var(--fg3);vertical-align:middle}\n' +
'\n' +
'.error-state{display:none;padding:32px}\n' +
'.error-box{max-width:var(--w);margin:0 auto;background:#fff5f5;border:1px solid #fecaca;padding:16px 20px;font-size:14px;color:#b91c1c}\n' +
'\n' +
'.result-state{display:none}\n' +
'\n' +
'/* meta bar */\n' +
'.meta-bar{width:100%;padding:14px 32px;border-bottom:1px solid var(--border2)}\n' +
'.meta-inner{max-width:var(--w);margin:0 auto;display:flex;align-items:center;gap:10px}\n' +
'.r-title{flex:1;font-size:14px;font-weight:500;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}\n' +
'.badge{font-family:var(--mono);font-size:11px;padding:2px 7px;white-space:nowrap;flex-shrink:0}\n' +
'.b-native{background:#f3f0ff;color:#5b21b6}\n' +
'.b-ai{background:#eff6ff;color:#1d4ed8}\n' +
'.b-browser{background:#fffbeb;color:#92400e}\n' +
'.b-dim{background:#f5f5f5;color:#555}\n' +
'.r-raw{font-size:13px;color:var(--fg3);white-space:nowrap;flex-shrink:0}\n' +
'.r-raw:hover{color:var(--fg)}\n' +
'\n' +
'/* tab bar */\n' +
'.tab-bar{width:100%;padding:0 32px;border-bottom:1px solid var(--border2)}\n' +
'.tab-bar-inner{max-width:var(--w);margin:0 auto;display:flex;align-items:center;justify-content:space-between}\n' +
'.tabs{display:flex}\n' +
'.tab{font-size:13px;padding:12px 16px;background:none;border:none;border-bottom:2px solid transparent;margin-bottom:-1px;cursor:pointer;color:var(--fg3);transition:color .15s,border-color .15s;font-family:var(--sans)}\n' +
'.tab.on{color:var(--fg);border-bottom-color:var(--fg)}\n' +
'.tab-actions{display:flex;gap:8px}\n' +
'.act-btn{font-family:var(--sans);font-size:13px;padding:7px 14px;background:#fff;color:var(--fg2);border:1px solid transparent;cursor:pointer;transition:border-color .15s,color .15s}\n' +
'.act-btn:hover{border:1px solid var(--border2);color:var(--fg)}\n' +
'\n' +
'/* panels */\n' +
'.panel{display:none}\n' +
'.panel.on{display:block}\n' +
'.md-panel{padding:32px}\n' +
'.pv-panel{padding:32px}\n' +
'.md-wrap{max-width:var(--w);margin:0 auto}\n' +
'.pv-wrap{max-width:860px;margin:0 auto}\n' +
'#md-out{font-family:var(--mono);font-size:13.5px;line-height:1.75;white-space:pre-wrap;color:var(--fg)}\n' +
'\n' +
'/* prose styles for preview panel */\n' +
'#prev-out h1{font-size:1.6em;font-weight:600;margin-bottom:16px;letter-spacing:-.02em}\n' +
'#prev-out h2{font-size:1.3em;font-weight:600;margin:24px 0 12px}\n' +
'#prev-out h3{font-size:1.1em;font-weight:600;margin:20px 0 10px}\n' +
'#prev-out h4,#prev-out h5,#prev-out h6{font-weight:600;margin:16px 0 8px}\n' +
'#prev-out p{color:#333;line-height:1.75;margin-bottom:14px}\n' +
'#prev-out ul,#prev-out ol{padding-left:24px;margin-bottom:14px}\n' +
'#prev-out li{line-height:1.7;color:#333}\n' +
'#prev-out code{font-family:var(--mono);font-size:12px;background:#f5f5f5;padding:1px 5px}\n' +
'#prev-out pre{background:#111;padding:20px;overflow-x:auto;margin-bottom:16px}\n' +
'#prev-out pre code{background:none;padding:0;color:#e4e4e7;font-size:13px}\n' +
'#prev-out blockquote{border-left:3px solid var(--border2);padding-left:16px;color:var(--fg3);margin-bottom:14px}\n' +
'#prev-out a{color:#1a73e8;text-decoration:underline;text-decoration-color:#c7d7f5}\n' +
'#prev-out a:hover{text-decoration-color:#1a73e8}\n' +
'#prev-out table{width:100%;border-collapse:collapse;margin-bottom:16px;font-size:14px}\n' +
'#prev-out th,#prev-out td{border:1px solid var(--border2);padding:8px 12px;text-align:left}\n' +
'#prev-out th{background:#fafafa;font-weight:600}\n' +
'#prev-out hr{border:none;border-top:1px solid var(--border2);margin:24px 0}\n' +
'#prev-out img{max-width:100%;height:auto}\n' +
'  <\/style>\n' +
'<\/head>\n' +
'<body>\n' +
'\n' +
'<header>\n' +
'  <div class="hdr">\n' +
'    <a href="/" class="logo">\n' +
'      <span class="logo-sq">\n' +
'        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="white" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">\n' +
'          <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"\/>\n' +
'          <polyline points="14 2 14 8 20 8"\/>\n' +
'        <\/svg>\n' +
'      <\/span>\n' +
'      markdown.go-mizu\n' +
'    <\/a>\n' +
'    <nav>\n' +
'      <a href="/docs">Docs<\/a>\n' +
'      <a href="https://github.com/go-mizu/mizu">GitHub<\/a>\n' +
'    <\/nav>\n' +
'  <\/div>\n' +
'<\/header>\n' +
'\n' +
'<div class="url-bar">\n' +
'  <form class="url-bar-inner" id="form" onsubmit="handleSubmit(event)">\n' +
'    <input id="url-in" type="url" class="url-in" placeholder="https://example.com" autocomplete="off" spellcheck="false">\n' +
'    <button type="submit" id="sub-btn" class="cvt-btn"><span id="btn-sp" style="display:none" class="spinner"><\/span><span id="btn-t">Convert<\/span><\/button>\n' +
'  <\/form>\n' +
'<\/div>\n' +
'\n' +
'<div class="loading-state" id="loading-state">\n' +
'  <span class="spinner"><\/span>\n' +
'  <span class="loading-txt">Converting\u2026<\/span>\n' +
'<\/div>\n' +
'\n' +
'<div class="error-state" id="error-state">\n' +
'  <div class="error-box" id="err-msg"><\/div>\n' +
'<\/div>\n' +
'\n' +
'<div class="result-state" id="result-state">\n' +
'\n' +
'  <div class="meta-bar">\n' +
'    <div class="meta-inner">\n' +
'      <span class="r-title" id="r-title"><\/span>\n' +
'      <span class="badge b-ai" id="r-method"><\/span>\n' +
'      <span class="badge b-dim" id="r-dur"><\/span>\n' +
'      <span class="badge b-dim" id="r-tok" style="display:none"><\/span>\n' +
'      <a class="r-raw" id="r-raw" href="#" target="_blank" rel="noopener">View raw \u2192<\/a>\n' +
'    <\/div>\n' +
'  <\/div>\n' +
'\n' +
'  <div class="tab-bar">\n' +
'    <div class="tab-bar-inner">\n' +
'      <div class="tabs">\n' +
'        <button class="tab on" id="tab-md" onclick="switchTab(\'md\')">Markdown<\/button>\n' +
'        <button class="tab" id="tab-pv" onclick="switchTab(\'pv\')">Preview<\/button>\n' +
'      <\/div>\n' +
'      <div class="tab-actions">\n' +
'        <button class="act-btn" onclick="copyMd()"><span id="copy-lbl">Copy<\/span><\/button>\n' +
'        <button class="act-btn" onclick="saveMd()">Save .md<\/button>\n' +
'      <\/div>\n' +
'    <\/div>\n' +
'  <\/div>\n' +
'\n' +
'  <div id="panel-md" class="panel md-panel on">\n' +
'    <div class="md-wrap">\n' +
'      <pre id="md-out"><\/pre>\n' +
'    <\/div>\n' +
'  <\/div>\n' +
'\n' +
'  <div id="panel-pv" class="panel pv-panel">\n' +
'    <div class="pv-wrap">\n' +
'      <div id="prev-out"><\/div>\n' +
'    <\/div>\n' +
'  <\/div>\n' +
'\n' +
'<\/div>\n' +
'\n' +
'<script>\n' +
'var md = \'\';\n' +
'var currentUrl = \'\';\n' +
'\n' +
'var METHOD_MAP = {\n' +
'  primary: [\'\u2726 Native\', \'badge b-native\'],\n' +
'  ai:      [\'\u26a1 Workers AI\', \'badge b-ai\'],\n' +
'  browser: [\'\uD83D\uDDA5 Browser\', \'badge b-browser\'],\n' +
'};\n' +
'\n' +
'function handleSubmit(e) {\n' +
'  e.preventDefault();\n' +
'  var url = document.getElementById(\'url-in\').value.trim();\n' +
'  if (!url) return;\n' +
'  history.replaceState(null, \'\', \'/preview?url=\' + encodeURIComponent(url));\n' +
'  convertUrl(url);\n' +
'}\n' +
'\n' +
'function setLoading(v) {\n' +
'  document.getElementById(\'sub-btn\').disabled = v;\n' +
'  document.getElementById(\'btn-t\').textContent = v ? \'Converting\u2026\' : \'Convert\';\n' +
'  document.getElementById(\'btn-sp\').style.display = v ? \'\' : \'none\';\n' +
'  document.getElementById(\'loading-state\').style.display = v ? \'block\' : \'none\';\n' +
'}\n' +
'\n' +
'async function convertUrl(url) {\n' +
'  currentUrl = url;\n' +
'  document.getElementById(\'url-in\').value = url;\n' +
'  document.getElementById(\'error-state\').style.display = \'none\';\n' +
'  document.getElementById(\'result-state\').style.display = \'none\';\n' +
'  setLoading(true);\n' +
'  try {\n' +
'    var r = await fetch(\'/convert\', {\n' +
'      method: \'POST\',\n' +
'      headers: {\'Content-Type\': \'application/json\'},\n' +
'      body: JSON.stringify({url: url})\n' +
'    });\n' +
'    if (!r.ok) {\n' +
'      var j = await r.json().catch(function() { return {error: \'Conversion failed\'}; });\n' +
'      throw new Error(j.error || \'HTTP \' + r.status);\n' +
'    }\n' +
'    showResult(await r.json());\n' +
'  } catch(e) {\n' +
'    document.getElementById(\'err-msg\').textContent = e.message || \'Conversion failed\';\n' +
'    document.getElementById(\'error-state\').style.display = \'block\';\n' +
'  } finally {\n' +
'    setLoading(false);\n' +
'  }\n' +
'}\n' +
'\n' +
'function showResult(data) {\n' +
'  md = data.markdown || \'\';\n' +
'  document.getElementById(\'r-title\').textContent = data.title || currentUrl;\n' +
'  var cfg = METHOD_MAP[data.method] || METHOD_MAP.ai;\n' +
'  var mb = document.getElementById(\'r-method\');\n' +
'  mb.textContent = cfg[0]; mb.className = cfg[1];\n' +
'  document.getElementById(\'r-dur\').textContent = data.durationMs + \'ms\';\n' +
'  var tb = document.getElementById(\'r-tok\');\n' +
'  if (data.tokens) { tb.textContent = \'\u007e\' + data.tokens.toLocaleString() + \' tokens\'; tb.style.display = \'\'; }\n' +
'  else { tb.style.display = \'none\'; }\n' +
'  document.getElementById(\'r-raw\').href = \'/\' + currentUrl;\n' +
'  document.getElementById(\'md-out\').textContent = md;\n' +
'  var parsed = marked.parse(md);\n' +
'  if (typeof parsed !== \'string\') return;\n' +
'  document.getElementById(\'prev-out\').innerHTML = DOMPurify.sanitize(parsed);\n' +
'  document.getElementById(\'result-state\').style.display = \'block\';\n' +
'  switchTab(\'md\');\n' +
'}\n' +
'\n' +
'function switchTab(t) {\n' +
'  [\'md\',\'pv\'].forEach(function(k) {\n' +
'    document.getElementById(\'panel-\' + k).className = \'panel\' + (k === t ? \' on\' : \'\') + (k === \'md\' ? \' md-panel\' : \' pv-panel\');\n' +
'    document.getElementById(\'tab-\' + k).className = \'tab\' + (k === t ? \' on\' : \'\');\n' +
'  });\n' +
'}\n' +
'\n' +
'async function copyMd() {\n' +
'  try { await navigator.clipboard.writeText(md); }\n' +
'  catch(e) {\n' +
'    var ta = document.createElement(\'textarea\');\n' +
'    ta.value = md; document.body.appendChild(ta); ta.select();\n' +
'    document.execCommand(\'copy\'); document.body.removeChild(ta);\n' +
'  }\n' +
'  var el = document.getElementById(\'copy-lbl\');\n' +
'  el.textContent = \'Copied!\';\n' +
'  setTimeout(function() { el.textContent = \'Copy\'; }, 2000);\n' +
'}\n' +
'\n' +
'function saveMd() {\n' +
'  var blob = new Blob([md], {type: \'text/markdown\'});\n' +
'  var a = document.createElement(\'a\');\n' +
'  a.href = URL.createObjectURL(blob);\n' +
'  var title = document.getElementById(\'r-title\').textContent || \'document\';\n' +
'  a.download = title.replace(/[^\\w\\s-]/g,\'\').trim().replace(/\\s+/g,\'-\').toLowerCase() + \'.md\';\n' +
'  document.body.appendChild(a);\n' +
'  a.click();\n' +
'  document.body.removeChild(a);\n' +
'  setTimeout(function() { URL.revokeObjectURL(a.href); }, 100);\n' +
'}\n' +
'\n' +
'window.addEventListener(\'load\', function() {\n' +
'  var params = new URLSearchParams(window.location.search);\n' +
'  var url = params.get(\'url\');\n' +
'  if (url) {\n' +
'    convertUrl(url);\n' +
'  } else {\n' +
'    document.getElementById(\'url-in\').focus();\n' +
'  }\n' +
'});\n' +
'<\/script>\n' +
'<\/body>\n' +
'<\/html>';
}
