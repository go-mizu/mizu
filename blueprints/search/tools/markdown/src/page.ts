export function renderPage(): string {
  return '<!DOCTYPE html>\n' +
'<html lang="en">\n' +
'<head>\n' +
'  <meta charset="UTF-8">\n' +
'  <meta name="viewport" content="width=device-width, initial-scale=1.0">\n' +
'  <title>URL \u2192 Markdown</title>\n' +
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
'code{font-family:var(--mono);font-size:11.5px;background:#f5f5f5;padding:1px 5px}\n' +
'.w{max-width:var(--w);margin:0 auto;padding:0 32px}\n' +
'\n' +
'/* header */\n' +
'header{padding:14px 32px}\n' +
'.hdr{display:flex;align-items:center;justify-content:space-between}\n' +
'.logo{font-size:13.5px;font-weight:500;letter-spacing:-.02em;display:flex;align-items:center;gap:8px;color:var(--fg)}\n' +
'.logo-sq{width:22px;height:22px;background:var(--fg);display:flex;align-items:center;justify-content:center;flex-shrink:0}\n' +
'nav a{font-size:13px;color:var(--fg3);margin-left:20px;transition:color .15s}\n' +
'nav a:hover{color:var(--fg)}\n' +
'\n' +
'/* hero */\n' +
'.hero{padding:72px 0 64px}\n' +
'.hero h1{font-size:clamp(40px,6vw,68px);font-weight:700;letter-spacing:-.04em;line-height:1.05;margin-bottom:36px;white-space:pre-line}\n' +
'.steps-list{list-style:none;margin-bottom:52px;display:flex;flex-direction:column;gap:14px}\n' +
'.step-item{display:flex;align-items:flex-start;gap:14px;font-size:15px;color:var(--fg2);line-height:1.5}\n' +
'.step-badge{width:22px;height:22px;background:#0a0a0a;color:#fff;font-size:12px;font-weight:700;font-family:var(--mono);display:flex;align-items:center;justify-content:center;flex-shrink:0;margin-top:2px}\n' +
'.step-item code{font-size:12px;background:#f5f5f5;padding:1px 6px}\n' +
'\n' +
'/* hero CTA grid */\n' +
'.cta-grid{display:grid;grid-template-columns:1fr 1fr;gap:48px;max-width:900px}\n' +
'@media(max-width:680px){.cta-grid{grid-template-columns:1fr}}\n' +
'.cta-col-lbl{font-family:var(--mono);font-size:11px;letter-spacing:.1em;text-transform:uppercase;color:var(--fg3);margin-bottom:10px}\n' +
'.url-row{display:flex}\n' +
'.url-in{flex:1;font-family:var(--mono);font-size:13px;padding:11px 14px;border:1px solid var(--border);border-right:none;background:#fff;color:var(--fg);outline:none;transition:border-color .15s;min-width:0}\n' +
'.url-in::placeholder{color:var(--fg3)}\n' +
'.url-in:focus{border-color:var(--fg)}\n' +
'.cvt-btn{font-family:var(--sans);font-size:13px;font-weight:500;padding:11px 20px;background:var(--fg);color:#fff;border:1px solid var(--fg);cursor:pointer;white-space:nowrap;transition:background .15s}\n' +
'.cvt-btn:hover{background:#333;border-color:#333}\n' +
'\n' +
'.agent-btn{width:100%;font-family:var(--sans);font-size:14px;font-weight:500;padding:14px 20px;background:#0a0a0a;color:#fff;border:1px solid #0a0a0a;cursor:pointer;display:flex;align-items:center;justify-content:center;gap:10px;transition:background .15s;margin-bottom:10px}\n' +
'.agent-btn:hover{background:#222;border-color:#222}\n' +
'.agent-confirm{font-size:13px;color:#15803d;display:none;margin-top:6px}\n' +
'\n' +
'hr.sep{border:none;border-top:1px solid var(--border2);margin:0}\n' +
'\n' +
'/* sections */\n' +
'.sec{padding:72px 0}\n' +
'.sec-lbl{font-family:var(--mono);font-size:11px;letter-spacing:.1em;text-transform:uppercase;color:var(--fg3);margin-bottom:20px}\n' +
'.sec h2{font-size:32px;font-weight:600;letter-spacing:-.03em;margin-bottom:12px}\n' +
'.sec-sub{font-size:17px;color:#555;margin-bottom:44px;max-width:540px;line-height:1.6}\n' +
'\n' +
'/* 3-col border trick grid */\n' +
'.bgrid{display:grid;grid-template-columns:repeat(3,1fr);gap:1px;background:var(--border2);border:1px solid var(--border2);margin-bottom:44px}\n' +
'@media(max-width:640px){.bgrid{grid-template-columns:1fr}}\n' +
'.bcard{background:#fff;padding:28px}\n' +
'.bcard-n{font-family:var(--mono);font-size:11px;color:var(--fg3);margin-bottom:10px}\n' +
'.bcard-t{font-size:15px;font-weight:600;margin-bottom:6px}\n' +
'.bcard-d{font-size:13px;color:var(--fg2);line-height:1.6}\n' +
'\n' +
'/* code tabs */\n' +
'.cbar{display:flex;background:#fafafa;border-bottom:1px solid var(--border)}\n' +
'.ctab{font-family:var(--mono);font-size:12px;padding:9px 14px;background:none;border:none;border-bottom:2px solid transparent;margin-bottom:-1px;cursor:pointer;color:var(--fg3);transition:color .15s,border-color .15s}\n' +
'.ctab.on{color:var(--fg);border-bottom-color:var(--fg)}\n' +
'.cpanel{display:none;background:var(--code-bg);padding:22px;overflow-x:auto;position:relative}\n' +
'.cpanel.on{display:block}\n' +
'.cpanel pre{font-family:var(--mono);font-size:13px;line-height:1.7;color:var(--code-fg);white-space:pre;margin:0}\n' +
'.c1{color:#6b7280}.c2{color:#93c5fd}.c3{color:#86efac}.c4{color:#fcd34d}\n' +
'.copy-btn{position:absolute;top:12px;right:12px;background:#222;border:1px solid #333;color:#aaa;font-size:11px;font-family:var(--mono);padding:4px 10px;cursor:pointer;transition:color .15s}\n' +
'.copy-btn:hover{color:#fff}\n' +
'\n' +
'/* pipeline tier badges */\n' +
'.b-native{background:#f3f0ff;color:#5b21b6}\n' +
'.b-ai{background:#eff6ff;color:#1d4ed8}\n' +
'.b-browser{background:#fffbeb;color:#92400e}\n' +
'.ptag{display:inline-block;font-family:var(--mono);font-size:11px;padding:2px 7px;margin-top:14px}\n' +
'\n' +
'/* api reference */\n' +
'.ep{border:1px solid var(--border2);margin-bottom:16px}\n' +
'.eph{padding:12px 16px;display:flex;align-items:center;gap:10px;background:#fafafa;border-bottom:1px solid var(--border2)}\n' +
'.mtag{font-family:var(--mono);font-size:11px;font-weight:600;padding:3px 8px;flex-shrink:0}\n' +
'.get{background:#f0fdf4;color:#15803d}\n' +
'.post{background:#eff6ff;color:#1d4ed8}\n' +
'.epath{font-family:var(--mono);font-size:13px}\n' +
'.edesc{font-size:12px;color:var(--fg3);margin-left:auto}\n' +
'.epcode{background:#111;padding:22px;overflow-x:auto;position:relative}\n' +
'.epcode pre{font-family:var(--mono);font-size:12.5px;line-height:1.8;color:#e4e4e7;white-space:pre;margin:0}\n' +
'.rk{color:#93c5fd}.rv{color:#e4e4e7}.rc{color:#6b7280}\n' +
'  <\/style>\n' +
'<\/head>\n' +
'<body>\n' +
'\n' +
'<header>\n' +
'  <div class="hdr">\n' +
'    <a href="/" class="logo">\n' +
'      <span class="logo-sq">\n' +
'        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="white" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">\n' +
'          <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>\n' +
'          <polyline points="14 2 14 8 20 8"/>\n' +
'        <\/svg>\n' +
'      <\/span>\n' +
'      URL \u2192 Markdown\n' +
'    <\/a>\n' +
'    <nav>\n' +
'      <a href="#agents">For agents<\/a>\n' +
'      <a href="#api">API<\/a>\n' +
'      <a href="/llms.txt">llms.txt<\/a>\n' +
'      <a href="https://github.com/go-mizu/mizu">GitHub<\/a>\n' +
'    <\/nav>\n' +
'  <\/div>\n' +
'<\/header>\n' +
'\n' +
'<main>\n' +
'\n' +
'<div class="w">\n' +
'  <section class="hero">\n' +
'    <h1>Any URL,\nclean Markdown<\/h1>\n' +
'\n' +
'    <ul class="steps-list">\n' +
'      <li class="step-item">\n' +
'        <span class="step-badge">1<\/span>\n' +
'        <span>Prepend your URL: <code>markdown.go-mizu.workers.dev\/{url}<\/code><\/span>\n' +
'      <\/li>\n' +
'      <li class="step-item">\n' +
'        <span class="step-badge">2<\/span>\n' +
'        <span>Get <code>text\/markdown<\/code> back \u2014 no account, no API key<\/span>\n' +
'      <\/li>\n' +
'    <\/ul>\n' +
'\n' +
'    <div class="cta-grid">\n' +
'      <div>\n' +
'        <div class="cta-col-lbl">CONVERT A URL<\/div>\n' +
'        <form id="form" onsubmit="handleSubmit(event)">\n' +
'          <div class="url-row">\n' +
'            <input id="url-in" type="url" class="url-in" placeholder="https://example.com" autocomplete="off" spellcheck="false">\n' +
'            <button type="submit" class="cvt-btn">Convert<\/button>\n' +
'          <\/div>\n' +
'        <\/form>\n' +
'      <\/div>\n' +
'      <div>\n' +
'        <div class="cta-col-lbl">FOR YOUR AGENT<\/div>\n' +
'        <button class="agent-btn" onclick="copyAgentInstructions()">\n' +
'          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="0"\/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"\/><\/svg>\n' +
'          Copy setup instructions for my agent\n' +
'        <\/button>\n' +
'        <div id="agent-confirm" class="agent-confirm">Copied to clipboard \u2014 paste into your agent chat.<\/div>\n' +
'      <\/div>\n' +
'    <\/div>\n' +
'  <\/section>\n' +
'<\/div>\n' +
'\n' +
'<hr class="sep">\n' +
'\n' +
'<div class="w">\n' +
'  <section class="sec" id="agents">\n' +
'    <p class="sec-lbl">How it works for agents<\/p>\n' +
'    <h2>Get Markdown in one request<\/h2>\n' +
'    <p class="sec-sub">No SDK, no setup. Any agent that can make HTTP requests works immediately.<\/p>\n' +
'\n' +
'    <div class="bgrid">\n' +
'      <div class="bcard">\n' +
'        <div class="bcard-n">01 \/ Request<\/div>\n' +
'        <div class="bcard-t">Prepend the URL<\/div>\n' +
'        <div class="bcard-d">Append any URL to <code>markdown.go-mizu.workers.dev\/<\/code>. Query strings are preserved.<\/div>\n' +
'      <\/div>\n' +
'      <div class="bcard">\n' +
'        <div class="bcard-n">02 \/ Receive<\/div>\n' +
'        <div class="bcard-t">Plain text\/markdown<\/div>\n' +
'        <div class="bcard-d">Response is clean <code>text\/markdown<\/code>. Metadata in headers: method, duration, title, token count.<\/div>\n' +
'      <\/div>\n' +
'      <div class="bcard">\n' +
'        <div class="bcard-n">03 \/ Scale<\/div>\n' +
'        <div class="bcard-t">Edge-cached at 1 hour<\/div>\n' +
'        <div class="bcard-d">CDN caches for 1 hour with stale-while-revalidate so latency stays low as you scale.<\/div>\n' +
'      <\/div>\n' +
'    <\/div>\n' +
'\n' +
'    <div>\n' +
'      <div class="cbar">\n' +
'        <button class="ctab on" id="ctab-sh" onclick="switchCode(\'sh\')">Shell<\/button>\n' +
'        <button class="ctab" id="ctab-js" onclick="switchCode(\'js\')">JavaScript<\/button>\n' +
'        <button class="ctab" id="ctab-py" onclick="switchCode(\'py\')">Python<\/button>\n' +
'      <\/div>\n' +
'      <div id="cpanel-sh" class="cpanel on">\n' +
'        <button class="copy-btn" onclick="copyBlock(\'cpanel-sh\', this)">copy<\/button>\n' +
'        <pre><span class="c1"># GET \u2014 returns text\/markdown<\/span>\n' +
'curl https:\/\/markdown.go-mizu.workers.dev\/https:\/\/example.com\n' +
'\n' +
'<span class="c1"># POST \u2014 structured JSON with metadata<\/span>\n' +
'curl -s -X POST https:\/\/markdown.go-mizu.workers.dev\/convert \\\n' +
'  -H \'Content-Type: application\/json\' \\\n' +
'  -d \'{"url":"https:\/\/example.com"}\' | jq .method<\/pre>\n' +
'      <\/div>\n' +
'      <div id="cpanel-js" class="cpanel">\n' +
'        <button class="copy-btn" onclick="copyBlock(\'cpanel-js\', this)">copy<\/button>\n' +
'        <pre><span class="c1">\/\/ One line \u2014 returns markdown text<\/span>\n' +
'<span class="c2">const<\/span> md = <span class="c2">await<\/span> <span class="c4">fetch<\/span>(\n' +
'  <span class="c3">\'https:\/\/markdown.go-mizu.workers.dev\/\'<\/span> + url\n' +
').<span class="c4">then<\/span>(r =&gt; r.<span class="c4">text<\/span>());\n' +
'\n' +
'<span class="c1">\/\/ JSON API with method + timing<\/span>\n' +
'<span class="c2">const<\/span> res = <span class="c2">await<\/span> <span class="c4">fetch<\/span>(<span class="c3">\'https:\/\/markdown.go-mizu.workers.dev\/convert\'<\/span>, {\n' +
'  method: <span class="c3">\'POST\'<\/span>,\n' +
'  headers: { <span class="c3">\'Content-Type\'<\/span>: <span class="c3">\'application\/json\'<\/span> },\n' +
'  body: JSON.stringify({ url })\n' +
'}).<span class="c4">then<\/span>(r =&gt; r.<span class="c4">json<\/span>());<\/pre>\n' +
'      <\/div>\n' +
'      <div id="cpanel-py" class="cpanel">\n' +
'        <button class="copy-btn" onclick="copyBlock(\'cpanel-py\', this)">copy<\/button>\n' +
'        <pre><span class="c2">import<\/span> httpx\n' +
'\n' +
'md = httpx.<span class="c4">get<\/span>(\n' +
'    <span class="c3">\'https:\/\/markdown.go-mizu.workers.dev\/\'<\/span> + url\n' +
').text\n' +
'\n' +
'res = httpx.<span class="c4">post<\/span>(\n' +
'    <span class="c3">\'https:\/\/markdown.go-mizu.workers.dev\/convert\'<\/span>,\n' +
'    json={<span class="c3">\'url\'<\/span>: url}\n' +
').<span class="c4">json<\/span>()<\/pre>\n' +
'      <\/div>\n' +
'    <\/div>\n' +
'  <\/section>\n' +
'<\/div>\n' +
'\n' +
'<hr class="sep">\n' +
'\n' +
'<div class="w">\n' +
'  <section class="sec" id="pipeline">\n' +
'    <p class="sec-lbl">Conversion pipeline<\/p>\n' +
'    <h2>Three tiers, one result<\/h2>\n' +
'    <p class="sec-sub">Every URL goes through the best available tier, falling back automatically until Markdown is produced.<\/p>\n' +
'    <div class="bgrid">\n' +
'      <div class="bcard">\n' +
'        <div class="bcard-n">Tier 1<\/div>\n' +
'        <div class="bcard-t">Native Markdown<\/div>\n' +
'        <div class="bcard-d"><code>Accept: text\/markdown<\/code> negotiation \u2014 sites that serve Markdown natively return it directly.<\/div>\n' +
'        <span class="ptag b-native">primary<\/span>\n' +
'      <\/div>\n' +
'      <div class="bcard">\n' +
'        <div class="bcard-n">Tier 2<\/div>\n' +
'        <div class="bcard-t">Workers AI<\/div>\n' +
'        <div class="bcard-d">HTML converted via Cloudflare Workers AI <code>toMarkdown()<\/code> \u2014 fast, structure-aware extraction.<\/div>\n' +
'        <span class="ptag b-ai">ai<\/span>\n' +
'      <\/div>\n' +
'      <div class="bcard">\n' +
'        <div class="bcard-n">Tier 3<\/div>\n' +
'        <div class="bcard-t">Browser Render<\/div>\n' +
'        <div class="bcard-d">JS-heavy SPAs rendered in a headless browser first, capturing dynamic content before AI conversion.<\/div>\n' +
'        <span class="ptag b-browser">browser<\/span>\n' +
'      <\/div>\n' +
'    <\/div>\n' +
'  <\/section>\n' +
'<\/div>\n' +
'\n' +
'<hr class="sep">\n' +
'\n' +
'<div class="w">\n' +
'  <section class="sec" id="api">\n' +
'    <p class="sec-lbl">API reference<\/p>\n' +
'    <h2>Endpoints<\/h2>\n' +
'\n' +
'    <div class="ep">\n' +
'      <div class="eph">\n' +
'        <span class="mtag get">GET<\/span>\n' +
'        <span class="epath">\/{url}<\/span>\n' +
'        <span class="edesc">Returns text\/markdown<\/span>\n' +
'      <\/div>\n' +
'      <div class="epcode">\n' +
'        <button class="copy-btn" onclick="copyBlock(\'ep-get\', this)">copy<\/button>\n' +
'        <pre id="ep-get"><span class="rc"># Append any absolute URL (http:\/\/ or https:\/\/)<\/span>\n' +
'<span class="rc">curl https:\/\/markdown.go-mizu.workers.dev\/https:\/\/example.com<\/span>\n' +
'\n' +
'<span class="rc"># Response headers<\/span>\n' +
'<span class="rk">Content-Type<\/span><span class="rv">: text\/markdown; charset=utf-8<\/span>\n' +
'<span class="rk">X-Conversion-Method<\/span><span class="rv">: primary | ai | browser<\/span>\n' +
'<span class="rk">X-Duration-Ms<\/span><span class="rv">: 342<\/span>\n' +
'<span class="rk">X-Title<\/span><span class="rv">: Example Domain<\/span>\n' +
'<span class="rk">X-Markdown-Tokens<\/span><span class="rv">: 1248<\/span>\n' +
'<span class="rk">Cache-Control<\/span><span class="rv">: public, max-age=300, s-maxage=3600, stale-while-revalidate=86400<\/span><\/pre>\n' +
'      <\/div>\n' +
'    <\/div>\n' +
'\n' +
'    <div class="ep">\n' +
'      <div class="eph">\n' +
'        <span class="mtag post">POST<\/span>\n' +
'        <span class="epath">\/convert<\/span>\n' +
'        <span class="edesc">Returns JSON<\/span>\n' +
'      <\/div>\n' +
'      <div class="epcode">\n' +
'        <button class="copy-btn" onclick="copyBlock(\'ep-post\', this)">copy<\/button>\n' +
'        <pre id="ep-post"><span class="rc"># Request<\/span>\n' +
'curl -X POST https:\/\/markdown.go-mizu.workers.dev\/convert \\\n' +
'  -H \'Content-Type: application\/json\' \\\n' +
'  -d \'{"url":"https:\/\/example.com"}\'\n' +
'\n' +
'<span class="rc"># Response JSON<\/span>\n' +
'{\n' +
'  <span class="rk">"markdown"<\/span>: <span class="rv">"# Example Domain\\n\\n..."<\/span>,\n' +
'  <span class="rk">"method"<\/span>: <span class="rv">"ai"<\/span>,\n' +
'  <span class="rk">"durationMs"<\/span>: <span class="rv">342<\/span>,\n' +
'  <span class="rk">"title"<\/span>: <span class="rv">"Example Domain"<\/span>,\n' +
'  <span class="rk">"tokens"<\/span>: <span class="rv">1248<\/span>\n' +
'}<\/pre>\n' +
'      <\/div>\n' +
'    <\/div>\n' +
'  <\/section>\n' +
'<\/div>\n' +
'\n' +
'<\/main>\n' +
'\n' +
'<script>\n' +
'var AGENT_INSTRUCTIONS = "I\'d like you to use https://markdown.go-mizu.workers.dev, the URL-to-Markdown API.\\n\\nTo fetch any URL as Markdown:\\n  GET https://markdown.go-mizu.workers.dev/{url}\\n\\nFor structured JSON with metadata:\\n  POST https://markdown.go-mizu.workers.dev/convert\\n  Body: {\\"url\\": \\"https://example.com\\"}\\n\\nReturns text/markdown. No API key needed. Free.";\n' +
'\n' +
'function handleSubmit(e) {\n' +
'  e.preventDefault();\n' +
'  var url = document.getElementById(\'url-in\').value.trim();\n' +
'  if (url) window.location.href = \'/preview?url=\' + encodeURIComponent(url);\n' +
'}\n' +
'\n' +
'function setEg(url) {\n' +
'  document.getElementById(\'url-in\').value = url;\n' +
'  handleSubmit({ preventDefault: function() {} });\n' +
'}\n' +
'\n' +
'function copyAgentInstructions() {\n' +
'  var confirm = document.getElementById(\'agent-confirm\');\n' +
'  function showConfirm() {\n' +
'    confirm.style.display = \'block\';\n' +
'    setTimeout(function() { confirm.style.display = \'none\'; }, 3000);\n' +
'  }\n' +
'  if (navigator.clipboard && navigator.clipboard.writeText) {\n' +
'    navigator.clipboard.writeText(AGENT_INSTRUCTIONS).then(showConfirm).catch(function() {\n' +
'      fallbackCopy(AGENT_INSTRUCTIONS);\n' +
'      showConfirm();\n' +
'    });\n' +
'  } else {\n' +
'    fallbackCopy(AGENT_INSTRUCTIONS);\n' +
'    showConfirm();\n' +
'  }\n' +
'}\n' +
'\n' +
'function fallbackCopy(text) {\n' +
'  var ta = document.createElement(\'textarea\');\n' +
'  ta.value = text;\n' +
'  ta.style.position = \'fixed\';\n' +
'  ta.style.opacity = \'0\';\n' +
'  document.body.appendChild(ta);\n' +
'  ta.focus();\n' +
'  ta.select();\n' +
'  document.execCommand(\'copy\');\n' +
'  document.body.removeChild(ta);\n' +
'}\n' +
'\n' +
'function switchCode(lang) {\n' +
'  [\'sh\', \'js\', \'py\'].forEach(function(k) {\n' +
'    document.getElementById(\'cpanel-\' + k).className = \'cpanel\' + (k === lang ? \' on\' : \'\');\n' +
'    document.getElementById(\'ctab-\' + k).className = \'ctab\' + (k === lang ? \' on\' : \'\');\n' +
'  });\n' +
'}\n' +
'\n' +
'function copyBlock(panelId, btn) {\n' +
'  var el = document.getElementById(panelId);\n' +
'  if (!el) return;\n' +
'  var text = el.innerText || el.textContent || \'\';\n' +
'  var origText = btn.textContent;\n' +
'  function done() {\n' +
'    btn.textContent = \'copied!\';\n' +
'    setTimeout(function() { btn.textContent = origText; }, 2000);\n' +
'  }\n' +
'  if (navigator.clipboard && navigator.clipboard.writeText) {\n' +
'    navigator.clipboard.writeText(text).then(done).catch(function() { fallbackCopy(text); done(); });\n' +
'  } else {\n' +
'    fallbackCopy(text);\n' +
'    done();\n' +
'  }\n' +
'}\n' +
'<\/script>\n' +
'<\/body>\n' +
'<\/html>';
}
