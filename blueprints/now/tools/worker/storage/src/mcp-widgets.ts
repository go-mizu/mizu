// src/mcp-widgets.ts — ChatGPT UI widgets for Storage MCP tools
//
// Each widget is a self-contained HTML document rendered in ChatGPT's
// sandboxed iframe. Widgets read data from window.openai.toolOutput
// (structuredContent) and window.openai.toolResponseMetadata (_meta).

export const WIDGET_DOMAIN = "https://storage.liteio.dev";

export const WIDGET_RESOURCE_META = {
  ui: {
    prefersBorder: true,
    domain: WIDGET_DOMAIN,
    csp: {
      connectDomains: [] as string[],
      resourceDomains: [] as string[],
      frameDomains: [] as string[],
    },
  },
};

export const WIDGET_RESOURCES = [
  { uri: "ui://storage/files", name: "File Browser", description: "Interactive file and folder browser with navigation", mimeType: "text/html;profile=mcp-app" as const },
  { uri: "ui://storage/viewer", name: "File Viewer", description: "File content viewer with metadata", mimeType: "text/html;profile=mcp-app" as const },
  { uri: "ui://storage/result", name: "Operation Result", description: "File operation confirmation", mimeType: "text/html;profile=mcp-app" as const },
  { uri: "ui://storage/share", name: "Share Link", description: "Share link display with copy button", mimeType: "text/html;profile=mcp-app" as const },
  { uri: "ui://storage/stats", name: "Storage Usage", description: "Storage usage dashboard", mimeType: "text/html;profile=mcp-app" as const },
];

/** Map a resource URI to its widget HTML. Returns null for unknown URIs. */
export function getWidgetHtml(uri: string): string | null {
  switch (uri) {
    case "ui://storage/files": return filesWidget();
    case "ui://storage/viewer": return viewerWidget();
    case "ui://storage/result": return resultWidget();
    case "ui://storage/share": return shareWidget();
    case "ui://storage/stats": return statsWidget();
    default: return null;
  }
}

// ── Tool → Widget mapping ──────────────────────────────────────────────

export const TOOL_WIDGET_MAP: Record<string, { uri: string; invoking: string; invoked: string; widgetDescription: string }> = {
  storage_list:   { uri: "ui://storage/files",  invoking: "Listing files\u2026",        invoked: "Files loaded",     widgetDescription: "Interactive file browser with folder navigation" },
  storage_read:   { uri: "ui://storage/viewer", invoking: "Reading file\u2026",         invoked: "File loaded",      widgetDescription: "File content viewer" },
  storage_write:  { uri: "ui://storage/result", invoking: "Saving file\u2026",          invoked: "File saved",       widgetDescription: "File save confirmation" },
  storage_delete: { uri: "ui://storage/result", invoking: "Deleting\u2026",             invoked: "Deleted",          widgetDescription: "Delete confirmation" },
  storage_search: { uri: "ui://storage/files",  invoking: "Searching\u2026",            invoked: "Search complete",  widgetDescription: "Search results browser" },
  storage_move:   { uri: "ui://storage/result", invoking: "Moving file\u2026",          invoked: "File moved",       widgetDescription: "File move confirmation" },
  storage_share:  { uri: "ui://storage/share",  invoking: "Creating share link\u2026",  invoked: "Link created",     widgetDescription: "Share link with copy button" },
  storage_stats:  { uri: "ui://storage/stats",  invoking: "Checking usage\u2026",       invoked: "Usage loaded",     widgetDescription: "Storage usage dashboard" },
};

// ── Base widget wrapper ────────────────────────────────────────────────

function widget(css: string, js: string): string {
  return '<!DOCTYPE html>\n<html>\n<head>\n<meta charset="utf-8">\n<meta name="viewport" content="width=device-width,initial-scale=1">\n<style>\n'
    + '*,*::before,*::after{box-sizing:border-box;margin:0;padding:0}\n'
    + ':root{\n'
    + '  --bg:#fff;--surface:#f9fafb;--surface-2:#f3f4f6;--text:#111827;--text-2:#6b7280;--text-3:#9ca3af;\n'
    + '  --border:#e5e7eb;--border-2:#d1d5db;\n'
    + '  --accent:#2563eb;--accent-soft:#eff6ff;--accent-text:#1d4ed8;\n'
    + '  --success:#059669;--success-soft:#ecfdf5;--success-text:#065f46;\n'
    + '  --danger:#dc2626;--danger-soft:#fef2f2;--danger-text:#991b1b;\n'
    + '  --radius:8px;--radius-sm:6px;\n'
    + '  font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;\n'
    + '  font-size:14px;line-height:1.5;color:var(--text);background:var(--bg);\n'
    + '}\n'
    + '[data-theme="dark"]{\n'
    + '  --bg:#0d1117;--surface:#161b22;--surface-2:#21262d;--text:#e6edf3;--text-2:#8b949e;--text-3:#484f58;\n'
    + '  --border:#30363d;--border-2:#484f58;\n'
    + '  --accent:#58a6ff;--accent-soft:#0d1117;--accent-text:#79c0ff;\n'
    + '  --success:#3fb950;--success-soft:#0d1117;--success-text:#56d364;\n'
    + '  --danger:#f85149;--danger-soft:#0d1117;--danger-text:#ff7b72;\n'
    + '}\n'
    + 'body{padding:0;overflow-x:hidden}\n'
    + css
    + '\n</style>\n</head>\n<body>\n<div id="root"></div>\n<script>\n(function(){\n'
    + 'var oi=window.openai||{};\n'
    + 'var theme=oi.theme||"light";\n'
    + 'var data=oi.toolOutput||{};\n'
    + 'var meta=oi.toolResponseMetadata||{};\n'
    + 'document.documentElement.setAttribute("data-theme",theme);\n'
    + 'var root=document.getElementById("root");\n'
    // Shared utilities
    + 'function esc(s){return String(s||"").replace(/&/g,"&amp;").replace(/</g,"&lt;").replace(/>/g,"&gt;").replace(/"/g,"&quot;")}\n'
    + 'function humanSize(b){if(!b)return"0 B";var u=["B","KB","MB","GB"];var i=Math.min(Math.floor(Math.log(b)/Math.log(1024)),u.length-1);return(b/Math.pow(1024,i)).toFixed(i===0?0:1)+" "+u[i]}\n'
    + 'function timeAgo(ts){if(!ts)return"";var d=Date.now()-ts;if(d<60000)return"just now";if(d<3600000)return Math.floor(d/60000)+"m ago";if(d<86400000)return Math.floor(d/3600000)+"h ago";return Math.floor(d/86400000)+"d ago"}\n'
    // SVG icons
    + 'var IC={\n'
    + '  folder:\'<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M22 19a2 2 0 01-2 2H4a2 2 0 01-2-2V5a2 2 0 012-2h5l2 3h9a2 2 0 012 2z"/></svg>\',\n'
    + '  file:\'<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>\',\n'
    + '  code:\'<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/></svg>\',\n'
    + '  image:\'<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="2" ry="2"/><circle cx="8.5" cy="8.5" r="1.5"/><polyline points="21 15 16 10 5 21"/></svg>\',\n'
    + '  chevron:\'<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="9 18 15 12 9 6"/></svg>\',\n'
    + '  check:\'<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 11.08V12a10 10 0 11-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>\',\n'
    + '  copy:\'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1"/></svg>\',\n'
    + '  link:\'<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M10 13a5 5 0 007.54.54l3-3a5 5 0 00-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 00-7.54-.54l-3 3a5 5 0 007.07 7.07l1.71-1.71"/></svg>\',\n'
    + '  chart:\'<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="20" x2="18" y2="10"/><line x1="12" y1="20" x2="12" y2="4"/><line x1="6" y1="20" x2="6" y2="14"/></svg>\',\n'
    + '  arrow:\'<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="5" y1="12" x2="19" y2="12"/><polyline points="12 5 19 12 12 19"/></svg>\',\n'
    + '  trash:\'<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/></svg>\',\n'
    + '  download:\'<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>\',\n'
    + '  move:\'<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="8" y1="13" x2="16" y2="13"/><polyline points="13 10 16 13 13 16"/></svg>\'\n'
    + '};\n'
    + 'function fileIcon(type){\n'
    + '  if(!type)return IC.file;\n'
    + '  if(type==="directory")return IC.folder;\n'
    + '  if(type.indexOf("image/")===0)return IC.image;\n'
    + '  var textTypes=["text/","json","xml","javascript","typescript","css","html","markdown","yaml","toml"];\n'
    + '  for(var i=0;i<textTypes.length;i++){if(type.indexOf(textTypes[i])!==-1)return IC.code;}\n'
    + '  return IC.file;\n'
    + '}\n'
    + js
    + '\n})();\n</script>\n</body>\n</html>';
}

// ── 1. File Browser Widget (storage_list + storage_search) ─────────────

export function filesWidget(): string {
  return widget(
    // CSS
    '.fb{padding:12px 16px}'
    + '\n.fb-header{margin-bottom:12px}'
    + '\n.fb-title{font-size:13px;font-weight:600;color:var(--text)}'
    + '\n.fb-sub{font-size:12px;color:var(--text-3);margin-top:2px}'
    + '\n.fb-list{border:1px solid var(--border);border-radius:var(--radius);overflow:hidden}'
    + '\n.fb-item{display:flex;align-items:center;gap:10px;padding:10px 12px;border-bottom:1px solid var(--border);cursor:pointer;transition:background .15s}'
    + '\n.fb-item:last-child{border-bottom:none}'
    + '\n.fb-item:hover{background:var(--surface)}'
    + '\n.fb-item:active{background:var(--surface-2)}'
    + '\n.fb-icon{flex-shrink:0;color:var(--text-3);display:flex;align-items:center}'
    + '\n.fb-icon.dir{color:var(--accent-text)}'
    + '\n.fb-info{flex:1;min-width:0}'
    + '\n.fb-name{font-size:13px;font-weight:500;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}'
    + '\n.fb-meta{font-size:11px;color:var(--text-3);margin-top:1px}'
    + '\n.fb-chev{flex-shrink:0;color:var(--text-3);display:flex;align-items:center}'
    + '\n.fb-empty{padding:32px 16px;text-align:center;color:var(--text-3);font-size:13px}'
    + '\n.fb-footer{padding:8px 0 0;font-size:11px;color:var(--text-3);text-align:center}',
    // JS
    'var entries=data.entries||data.items||[];\n'
    + 'var prefix=data.prefix||"/";\n'
    + 'var query=data.query||"";\n'
    + 'var isSearch=!!query;\n'
    + 'var count=data.count||entries.length;\n'
    + '\n'
    + '// Sort: folders first, then alphabetical\n'
    + 'entries.sort(function(a,b){\n'
    + '  var ad=a.type==="directory"?0:1,bd=b.type==="directory"?0:1;\n'
    + '  if(ad!==bd)return ad-bd;\n'
    + '  return(a.name||"").localeCompare(b.name||"");\n'
    + '});\n'
    + '\n'
    + 'var MAX=50;\n'
    + 'var visible=entries.length>MAX?entries.slice(0,MAX):entries;\n'
    + '\n'
    + 'var html=\'<div class="fb">\';\n'
    + 'html+=\'<div class="fb-header">\';\n'
    + 'if(isSearch){\n'
    + '  html+=\'<div class="fb-title">\'+count+" result"+(count!==1?"s":"")+" for \\u201c"+esc(query)+"\\u201d</div>";\n'
    + '}else{\n'
    + '  html+=\'<div class="fb-title">\'+esc(prefix==="/"?"My Files":prefix)+"</div>";\n'
    + '  html+=\'<div class="fb-sub">\'+entries.length+" item"+(entries.length!==1?"s":"")+"</div>";\n'
    + '}\n'
    + 'html+="</div>";\n'
    + '\n'
    + 'if(visible.length===0){\n'
    + '  html+=\'<div class="fb-empty">\'+(isSearch?"No files found":"This folder is empty")+"</div>";\n'
    + '}else{\n'
    + '  html+=\'<div class="fb-list">\';\n'
    + '  for(var i=0;i<visible.length;i++){\n'
    + '    var e=visible[i];\n'
    + '    var isDir=e.type==="directory";\n'
    + '    var displayName=isSearch?(e.path||e.name):e.name;\n'
    + '    var clickPath=isSearch?(e.path||e.name):((prefix==="/"?"":prefix)+e.name);\n'
    + '    html+=\'<div class="fb-item" data-path="\'+esc(clickPath)+\'" data-type="\'+esc(e.type)+\'" data-dir="\'+isDir+\'">\';\n'
    + '    html+=\'<div class="fb-icon\'+(isDir?" dir":"")+"\\">"+fileIcon(e.type)+"</div>";\n'
    + '    html+=\'<div class="fb-info">\';\n'
    + '    html+=\'<div class="fb-name">\'+esc(displayName)+"</div>";\n'
    + '    if(!isDir&&e.size!=null){\n'
    + '      html+=\'<div class="fb-meta">\'+humanSize(e.size);\n'
    + '      if(e.updated_at)html+=" \\u00b7 "+timeAgo(e.updated_at);\n'
    + '      html+="</div>";\n'
    + '    }\n'
    + '    html+="</div>";\n'
    + '    if(isDir)html+=\'<div class="fb-chev">\'+IC.chevron+"</div>";\n'
    + '    html+="</div>";\n'
    + '  }\n'
    + '  html+="</div>";\n'
    + '  if(entries.length>MAX){\n'
    + '    html+=\'<div class="fb-footer">\'+MAX+" of "+entries.length+" items shown</div>";\n'
    + '  }\n'
    + '}\n'
    + 'html+="</div>";\n'
    + 'root.innerHTML=html;\n'
    + '\n'
    + '// Click handlers\n'
    + 'var items=root.querySelectorAll(".fb-item");\n'
    + 'for(var j=0;j<items.length;j++){\n'
    + '  items[j].addEventListener("click",function(){\n'
    + '    var path=this.getAttribute("data-path");\n'
    + '    var isD=this.getAttribute("data-dir")==="true";\n'
    + '    if(isD&&oi.callTool){\n'
    + '      oi.callTool("storage_list",{prefix:path});\n'
    + '    }else if(!isD&&oi.callTool){\n'
    + '      oi.callTool("storage_read",{path:path});\n'
    + '    }\n'
    + '  });\n'
    + '}\n'
    + '\n'
    + '// Report height\n'
    + 'if(oi.notifyIntrinsicHeight){\n'
    + '  setTimeout(function(){oi.notifyIntrinsicHeight(document.body.scrollHeight)},0);\n'
    + '}'
  );
}

// ── 2. File Viewer Widget (storage_read) ───────────────────────────────

export function viewerWidget(): string {
  return widget(
    // CSS
    '.fv{padding:12px 16px}'
    + '\n.fv-header{display:flex;align-items:center;gap:8px;margin-bottom:8px}'
    + '\n.fv-icon{flex-shrink:0;color:var(--text-3);display:flex;align-items:center}'
    + '\n.fv-path{font-size:13px;font-weight:600;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}'
    + '\n.fv-meta{font-size:11px;color:var(--text-3);margin-bottom:12px;display:flex;gap:8px;flex-wrap:wrap}'
    + '\n.fv-meta span{display:inline-flex;align-items:center;gap:4px}'
    + '\n.fv-code{position:relative;border:1px solid var(--border);border-radius:var(--radius);overflow:hidden}'
    + '\n.fv-code pre{margin:0;padding:12px 16px;font-family:"SF Mono",Menlo,Consolas,"Liberation Mono",monospace;font-size:12px;line-height:1.6;overflow-x:auto;white-space:pre;background:var(--surface);color:var(--text);max-height:400px;overflow-y:auto}'
    + '\n.fv-copy{position:absolute;top:8px;right:8px;background:var(--surface-2);border:1px solid var(--border);border-radius:var(--radius-sm);padding:4px 8px;cursor:pointer;display:flex;align-items:center;gap:4px;font-size:11px;color:var(--text-2);transition:all .15s;z-index:1}'
    + '\n.fv-copy:hover{background:var(--border);color:var(--text)}'
    + '\n.fv-copy.copied{color:var(--success);border-color:var(--success)}'
    // Markdown rendered output
    + '\n.fv-md{border:1px solid var(--border);border-radius:var(--radius);padding:16px 20px;background:var(--bg);position:relative}'
    + '\n.fv-md h1{font-size:20px;font-weight:700;margin:0 0 12px;padding-bottom:8px;border-bottom:1px solid var(--border)}'
    + '\n.fv-md h2{font-size:17px;font-weight:600;margin:20px 0 8px;padding-bottom:6px;border-bottom:1px solid var(--border)}'
    + '\n.fv-md h3{font-size:15px;font-weight:600;margin:16px 0 6px}'
    + '\n.fv-md h4,.fv-md h5,.fv-md h6{font-size:14px;font-weight:600;margin:12px 0 4px}'
    + '\n.fv-md p{margin:0 0 10px;line-height:1.7}'
    + '\n.fv-md ul,.fv-md ol{margin:0 0 10px;padding-left:24px}'
    + '\n.fv-md li{margin-bottom:4px;line-height:1.6}'
    + '\n.fv-md blockquote{border-left:3px solid var(--border-2);padding:4px 12px;margin:0 0 10px;color:var(--text-2)}'
    + '\n.fv-md code{font-family:"SF Mono",Menlo,Consolas,monospace;font-size:0.85em;background:var(--surface-2);padding:1px 5px;border-radius:3px}'
    + '\n.fv-md pre{margin:0 0 10px;border-radius:var(--radius-sm);overflow-x:auto}'
    + '\n.fv-md pre code{display:block;padding:12px 16px;background:var(--surface);border:1px solid var(--border);border-radius:var(--radius-sm);font-size:12px;line-height:1.6}'
    + '\n.fv-md hr{border:none;border-top:1px solid var(--border);margin:16px 0}'
    + '\n.fv-md a{color:var(--accent-text);text-decoration:underline;text-underline-offset:2px}'
    + '\n.fv-md a:hover{opacity:0.8}'
    + '\n.fv-md strong{font-weight:600}'
    + '\n.fv-md table{border-collapse:collapse;margin:0 0 10px;width:100%}'
    + '\n.fv-md th,.fv-md td{border:1px solid var(--border);padding:6px 10px;text-align:left;font-size:13px}'
    + '\n.fv-md th{background:var(--surface);font-weight:600}'
    // Syntax highlighting tokens
    + '\n.fv-code .tk-kw{color:#c678dd}[data-theme="dark"] .fv-code .tk-kw{color:#c678dd}'
    + '\n.fv-code .tk-str{color:#98c379}[data-theme="dark"] .fv-code .tk-str{color:#98c379}'
    + '\n.fv-code .tk-cm{color:#5c6370;font-style:italic}[data-theme="dark"] .fv-code .tk-cm{color:#5c6370}'
    + '\n.fv-code .tk-num{color:#d19a66}[data-theme="dark"] .fv-code .tk-num{color:#d19a66}'
    + '\n.fv-code .tk-fn{color:#61afef}[data-theme="dark"] .fv-code .tk-fn{color:#61afef}'
    + '\n:root .fv-code .tk-kw{color:#7c3aed}.fv-code .tk-str{color:#16a34a}.fv-code .tk-cm{color:#9ca3af}.fv-code .tk-num{color:#ea580c}.fv-code .tk-fn{color:#2563eb}'
    // Binary file
    + '\n.fv-binary{border:1px solid var(--border);border-radius:var(--radius);padding:24px;text-align:center}'
    + '\n.fv-binary-icon{color:var(--text-3);margin-bottom:8px}'
    + '\n.fv-binary-text{font-size:13px;color:var(--text-2);margin-bottom:12px}'
    + '\n.fv-dl{display:inline-flex;align-items:center;gap:6px;padding:8px 16px;background:var(--accent);color:#fff;border:none;border-radius:var(--radius-sm);font-size:13px;font-weight:500;cursor:pointer;text-decoration:none;transition:opacity .15s}'
    + '\n.fv-dl:hover{opacity:0.9}',
    // JS
    'var path=data.path||"";\n'
    + 'var size=data.size||0;\n'
    + 'var ct=data.content_type||"";\n'
    + 'var isText=data.is_text!==false;\n'
    + 'var content=meta.fileContent||"";\n'
    + 'var dlUrl=data.download_url||meta.downloadUrl||"";\n'
    + 'var name=path.split("/").pop()||path;\n'
    + 'var ext=(name.match(/\\.([^.]+)$/)||[])[1]||"";\n'
    + 'var isMd=ct.indexOf("markdown")!==-1||ext==="md"||ext==="mdx";\n'
    + '\n'
    // Lightweight markdown-to-HTML renderer
    + 'function md2html(src){\n'
    + '  var out="",lines=src.split("\\n"),inCode=false,inList="",codeBuf="";\n'
    + '  function closeList(){if(inList){out+="</"+inList+">";inList="";}}\n'
    + '  function inl(s){\n'
    + '    return esc(s)\n'
    + '      .replace(/\\*\\*(.+?)\\*\\*/g,"<strong>$1</strong>")\n'
    + '      .replace(/\\*(.+?)\\*/g,"<em>$1</em>")\n'
    + '      .replace(/`(.+?)`/g,"<code>$1</code>")\n'
    + '      .replace(/\\[(.+?)\\]\\((.+?)\\)/g,\'<a href="$2" target="_blank" rel="noopener">$1</a>\');\n'
    + '  }\n'
    + '  for(var i=0;i<lines.length;i++){\n'
    + '    var L=lines[i];\n'
    + '    if(L.match(/^```/)){if(inCode){out+="<pre><code>"+esc(codeBuf)+"</code></pre>";codeBuf="";inCode=false;}else{closeList();inCode=true;}continue;}\n'
    + '    if(inCode){codeBuf+=L+"\\n";continue;}\n'
    + '    var hm=L.match(/^(#{1,6})\\s+(.*)/);if(hm){closeList();var lv=hm[1].length;out+="<h"+lv+">"+inl(hm[2])+"</h"+lv+">";continue;}\n'
    + '    if(L.match(/^---+\\s*$/)){closeList();out+="<hr>";continue;}\n'
    + '    if(L.match(/^>\\s?/)){closeList();out+="<blockquote>"+inl(L.replace(/^>\\s?/,""))+"</blockquote>";continue;}\n'
    + '    var um=L.match(/^\\s*[-*+]\\s+(.*)/);if(um){if(inList!=="ul"){closeList();out+="<ul>";inList="ul";}out+="<li>"+inl(um[1])+"</li>";continue;}\n'
    + '    var om=L.match(/^\\s*\\d+\\.\\s+(.*)/);if(om){if(inList!=="ol"){closeList();out+="<ol>";inList="ol";}out+="<li>"+inl(om[1])+"</li>";continue;}\n'
    + '    if(L.trim()===""){closeList();continue;}\n'
    + '    closeList();out+="<p>"+inl(L)+"</p>";\n'
    + '  }\n'
    + '  closeList();if(inCode)out+="<pre><code>"+esc(codeBuf)+"</code></pre>";\n'
    + '  return out;\n'
    + '}\n'
    + '\n'
    // Basic syntax highlighting (keywords, strings, comments, numbers)
    + 'function highlight(code,lang){\n'
    + '  var s=esc(code);\n'
    + '  // Comments (line and block)\n'
    + '  s=s.replace(/(^|\\n)(\\s*\\/\\/.*)(?=\\n|$)/g,\'$1<span class="tk-cm">$2</span>\');\n'
    + '  s=s.replace(/(\\/\\*[\\s\\S]*?\\*\\/)/g,\'<span class="tk-cm">$1</span>\');\n'
    + '  s=s.replace(/(^|\\n)(\\s*#[^!].*)(?=\\n|$)/g,function(m,pre,cm){if(lang==="py"||lang==="python"||lang==="sh"||lang==="bash"||lang==="yaml"||lang==="toml")return pre+\'<span class="tk-cm">\'+cm+"</span>";return m;});\n'
    + '  // Strings\n'
    + '  s=s.replace(/("|\'|`)(?:(?!\\1).)*?\\1/g,function(m){if(m.indexOf("tk-")!==-1)return m;return\'<span class="tk-str">\'+m+"</span>";});\n'
    + '  // Keywords\n'
    + '  var kw="\\\\b(function|const|let|var|if|else|return|import|export|from|class|new|this|async|await|for|while|do|switch|case|break|continue|try|catch|throw|finally|typeof|instanceof|in|of|true|false|null|undefined|def|self|elif|pass|with|as|yield|lambda|raise|except|None|True|False|fn|pub|mod|use|impl|struct|enum|match|mut|trait|type|interface|extends|implements|package|func|go|defer|select|chan)\\\\b";\n'
    + '  s=s.replace(new RegExp(kw,"g"),function(m){return\'<span class="tk-kw">\'+m+"</span>";});\n'
    + '  // Numbers\n'
    + '  s=s.replace(/\\b(\\d+\\.?\\d*)\\b/g,function(m){return\'<span class="tk-num">\'+m+"</span>";});\n'
    + '  return s;\n'
    + '}\n'
    + '\n'
    // Detect language from extension
    + 'function langFromExt(e){\n'
    + '  var map={js:"js",ts:"ts",jsx:"js",tsx:"ts",py:"py",rb:"py",go:"go",rs:"rs",java:"java",c:"c",cpp:"c",h:"c",css:"css",html:"html",xml:"html",json:"json",yaml:"yaml",yml:"yaml",toml:"toml",sh:"sh",bash:"sh",sql:"sql",md:"md"};\n'
    + '  return map[e]||"";\n'
    + '}\n'
    + '\n'
    + 'var html=\'<div class="fv">\';\n'
    + 'html+=\'<div class="fv-header">\';\n'
    + 'html+=\'<div class="fv-icon">\'+fileIcon(ct)+"</div>";\n'
    + 'html+=\'<div class="fv-path">\'+esc(path)+"</div>";\n'
    + 'html+="</div>";\n'
    + 'html+=\'<div class="fv-meta">\';\n'
    + 'html+="<span>"+humanSize(size)+"</span>";\n'
    + 'if(ct)html+="<span>"+esc(ct)+"</span>";\n'
    + 'html+="</div>";\n'
    + '\n'
    + 'if(isText&&content){\n'
    + '  if(isMd){\n'
    + '    // Render markdown as formatted HTML\n'
    + '    html+=\'<div class="fv-md" style="position:relative">\';\n'
    + '    html+=md2html(content);\n'
    + '    html+=\'<button class="fv-copy" id="copyBtn" style="position:absolute;top:8px;right:8px">\'+IC.copy+" Copy</button>";\n'
    + '    html+="</div>";\n'
    + '  }else{\n'
    + '    // Render source code with syntax highlighting\n'
    + '    var lang=langFromExt(ext);\n'
    + '    html+=\'<div class="fv-code">\';\n'
    + '    html+="<pre>"+(lang?highlight(content,lang):esc(content))+"</pre>";\n'
    + '    html+=\'<button class="fv-copy" id="copyBtn">\'+IC.copy+" Copy</button>";\n'
    + '    html+="</div>";\n'
    + '  }\n'
    + '}else if(dlUrl){\n'
    + '  html+=\'<div class="fv-binary">\';\n'
    + '  html+=\'<div class="fv-binary-icon">\'+fileIcon(ct)+"</div>";\n'
    + '  html+=\'<div class="fv-binary-text">Binary file \\u00b7 \'+humanSize(size)+"</div>";\n'
    + '  html+=\'<a class="fv-dl" href="\'+esc(dlUrl)+\'" target="_blank" rel="noopener">\'+IC.download+" Download</a>";\n'
    + '  html+="</div>";\n'
    + '}else{\n'
    + '  html+=\'<div class="fv-binary">\';\n'
    + '  html+=\'<div class="fv-binary-icon">\'+fileIcon(ct)+"</div>";\n'
    + '  html+=\'<div class="fv-binary-text">Binary file \\u00b7 \'+humanSize(size)+"</div>";\n'
    + '  html+="</div>";\n'
    + '}\n'
    + '\n'
    + 'html+="</div>";\n'
    + 'root.innerHTML=html;\n'
    + '\n'
    + '// Copy button handler\n'
    + 'var copyBtn=document.getElementById("copyBtn");\n'
    + 'if(copyBtn){\n'
    + '  copyBtn.addEventListener("click",function(){\n'
    + '    navigator.clipboard.writeText(content).then(function(){\n'
    + '      copyBtn.classList.add("copied");\n'
    + '      copyBtn.innerHTML=IC.check+" Copied";\n'
    + '      setTimeout(function(){copyBtn.classList.remove("copied");copyBtn.innerHTML=IC.copy+" Copy"},2000);\n'
    + '    });\n'
    + '  });\n'
    + '}\n'
    + '\n'
    + 'if(oi.notifyIntrinsicHeight){\n'
    + '  setTimeout(function(){oi.notifyIntrinsicHeight(document.body.scrollHeight)},0);\n'
    + '}'
  );
}

// ── 3. Result Widget (storage_write, storage_delete, storage_move) ─────

export function resultWidget(): string {
  return widget(
    // CSS
    '.rs{padding:16px}'
    + '\n.rs-card{border:1px solid var(--border);border-radius:var(--radius);padding:16px;display:flex;align-items:flex-start;gap:12px}'
    + '\n.rs-card.success{border-color:var(--success);background:var(--success-soft)}'
    + '\n.rs-card.danger{border-color:var(--danger);background:var(--danger-soft)}'
    + '\n.rs-icon{flex-shrink:0;display:flex;align-items:center}'
    + '\n.rs-icon.success{color:var(--success)}'
    + '\n.rs-icon.danger{color:var(--danger)}'
    + '\n.rs-body{flex:1;min-width:0}'
    + '\n.rs-title{font-size:14px;font-weight:600;margin-bottom:4px}'
    + '\n.rs-detail{font-size:12px;color:var(--text-2);line-height:1.6}'
    + '\n.rs-detail code{font-family:"SF Mono",Menlo,Consolas,monospace;font-size:11px;background:var(--surface-2);padding:1px 5px;border-radius:3px}'
    + '\n.rs-paths{margin-top:8px;display:flex;align-items:center;gap:8px;flex-wrap:wrap}'
    + '\n.rs-path{font-family:"SF Mono",Menlo,Consolas,monospace;font-size:12px;background:var(--surface-2);padding:4px 8px;border-radius:var(--radius-sm);max-width:100%;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}'
    + '\n.rs-arrow{color:var(--text-3);flex-shrink:0;display:flex}',
    // JS
    '// Detect operation type from data shape\n'
    + 'var op="unknown";\n'
    + 'var title="";\n'
    + 'var icon="";\n'
    + 'var cls="success";\n'
    + '\n'
    + 'if(data.deleted){\n'
    + '  op="delete";title="Deleted";icon=IC.trash;cls="danger";\n'
    + '}else if(data.old_path&&data.new_path){\n'
    + '  op="move";title="File moved";icon=IC.move;\n'
    + '}else if(data.path&&data.size!=null){\n'
    + '  op="write";title="File saved";icon=IC.check;\n'
    + '}else{\n'
    + '  title="Done";icon=IC.check;\n'
    + '}\n'
    + '\n'
    + 'var html=\'<div class="rs">\';\n'
    + 'html+=\'<div class="rs-card \'+cls+\'">\';\n'
    + 'html+=\'<div class="rs-icon \'+cls+\'">\'+(op==="delete"?IC.trash:IC.check)+"</div>";\n'
    + 'html+=\'<div class="rs-body">\';\n'
    + 'html+=\'<div class="rs-title">\'+esc(title)+"</div>";\n'
    + '\n'
    + 'if(op==="write"){\n'
    + '  html+=\'<div class="rs-detail">\';\n'
    + '  html+="<code>"+esc(data.path)+"</code> \\u00b7 "+humanSize(data.size);\n'
    + '  if(data.content_type)html+=" \\u00b7 "+esc(data.content_type);\n'
    + '  html+="</div>";\n'
    + '}else if(op==="delete"){\n'
    + '  var deleted=data.deleted||[];\n'
    + '  html+=\'<div class="rs-detail">\'+deleted.length+" item"+(deleted.length!==1?"s":"")+" deleted</div>";\n'
    + '  if(deleted.length<=5){\n'
    + '    html+=\'<div class="rs-paths">\';\n'
    + '    for(var i=0;i<deleted.length;i++){\n'
    + '      html+=\'<span class="rs-path">\'+esc(deleted[i])+"</span>";\n'
    + '    }\n'
    + '    html+="</div>";\n'
    + '  }\n'
    + '}else if(op==="move"){\n'
    + '  html+=\'<div class="rs-paths">\';\n'
    + '  html+=\'<span class="rs-path">\'+esc(data.old_path)+"</span>";\n'
    + '  html+=\'<span class="rs-arrow">\'+IC.arrow+"</span>";\n'
    + '  html+=\'<span class="rs-path">\'+esc(data.new_path)+"</span>";\n'
    + '  html+="</div>";\n'
    + '}\n'
    + '\n'
    + 'html+="</div></div></div>";\n'
    + 'root.innerHTML=html;\n'
    + '\n'
    + 'if(oi.notifyIntrinsicHeight){\n'
    + '  setTimeout(function(){oi.notifyIntrinsicHeight(document.body.scrollHeight)},0);\n'
    + '}'
  );
}

// ── 4. Share Link Widget (storage_share) ───────────────────────────────

export function shareWidget(): string {
  return widget(
    // CSS
    '.sh{padding:16px}'
    + '\n.sh-card{border:1px solid var(--border);border-radius:var(--radius);overflow:hidden}'
    + '\n.sh-header{display:flex;align-items:center;gap:8px;padding:12px 16px;border-bottom:1px solid var(--border);background:var(--success-soft)}'
    + '\n.sh-header-icon{color:var(--success);display:flex;align-items:center}'
    + '\n.sh-header-text{font-size:13px;font-weight:600;color:var(--success-text)}'
    + '\n.sh-body{padding:16px}'
    + '\n.sh-file{font-size:12px;color:var(--text-2);margin-bottom:12px;display:flex;align-items:center;gap:6px}'
    + '\n.sh-file-icon{color:var(--text-3);display:flex}'
    + '\n.sh-url-row{display:flex;align-items:stretch;border:1px solid var(--border);border-radius:var(--radius-sm);overflow:hidden}'
    + '\n.sh-url{flex:1;padding:10px 12px;font-family:"SF Mono",Menlo,Consolas,monospace;font-size:12px;background:var(--surface);color:var(--text);white-space:nowrap;overflow:hidden;text-overflow:ellipsis;display:flex;align-items:center}'
    + '\n.sh-copy-btn{flex-shrink:0;display:flex;align-items:center;gap:6px;padding:10px 14px;background:var(--accent);color:#fff;border:none;cursor:pointer;font-size:12px;font-weight:500;font-family:inherit;transition:opacity .15s}'
    + '\n.sh-copy-btn:hover{opacity:0.9}'
    + '\n.sh-copy-btn.copied{background:var(--success)}'
    + '\n.sh-footer{padding:12px 16px;border-top:1px solid var(--border);font-size:11px;color:var(--text-3);display:flex;align-items:center;gap:6px}',
    // JS
    'var url=data.url||"";\n'
    + 'var path=data.path||"";\n'
    + 'var expiresAt=data.expires_at||"";\n'
    + 'var name=path.split("/").pop()||path;\n'
    + '\n'
    + '// Format expiry\n'
    + 'var expiryText="";\n'
    + 'if(expiresAt){\n'
    + '  var diff=new Date(expiresAt).getTime()-Date.now();\n'
    + '  if(diff>0){\n'
    + '    if(diff<3600000)expiryText="Expires in "+Math.ceil(diff/60000)+" minutes";\n'
    + '    else if(diff<86400000)expiryText="Expires in "+Math.ceil(diff/3600000)+" hours";\n'
    + '    else expiryText="Expires in "+Math.ceil(diff/86400000)+" days";\n'
    + '  }else{\n'
    + '    expiryText="Expired";\n'
    + '  }\n'
    + '}\n'
    + '\n'
    + 'var html=\'<div class="sh"><div class="sh-card">\';\n'
    + 'html+=\'<div class="sh-header">\';\n'
    + 'html+=\'<div class="sh-header-icon">\'+IC.link+"</div>";\n'
    + 'html+=\'<div class="sh-header-text">Share link created</div>\';\n'
    + 'html+="</div>";\n'
    + 'html+=\'<div class="sh-body">\';\n'
    + 'html+=\'<div class="sh-file"><span class="sh-file-icon">\'+fileIcon("")+"</span>"+esc(name)+"</div>";\n'
    + 'html+=\'<div class="sh-url-row">\';\n'
    + 'html+=\'<div class="sh-url">\'+esc(url)+"</div>";\n'
    + 'html+=\'<button class="sh-copy-btn" id="copyUrl">\'+IC.copy+" Copy</button>";\n'
    + 'html+="</div>";\n'
    + 'html+="</div>";\n'
    + 'if(expiryText){\n'
    + '  html+=\'<div class="sh-footer">\\u23f1 \'+esc(expiryText)+"</div>";\n'
    + '}\n'
    + 'html+="</div></div>";\n'
    + 'root.innerHTML=html;\n'
    + '\n'
    + '// Copy handler\n'
    + 'var copyBtn=document.getElementById("copyUrl");\n'
    + 'if(copyBtn&&url){\n'
    + '  copyBtn.addEventListener("click",function(){\n'
    + '    navigator.clipboard.writeText(url).then(function(){\n'
    + '      copyBtn.classList.add("copied");\n'
    + '      copyBtn.innerHTML=IC.check+" Copied!";\n'
    + '      setTimeout(function(){copyBtn.classList.remove("copied");copyBtn.innerHTML=IC.copy+" Copy"},2000);\n'
    + '    });\n'
    + '  });\n'
    + '}\n'
    + '\n'
    + '// Open in browser\n'
    + 'var urlDiv=root.querySelector(".sh-url");\n'
    + 'if(urlDiv&&url){\n'
    + '  urlDiv.style.cursor="pointer";\n'
    + '  urlDiv.addEventListener("click",function(){\n'
    + '    if(oi.openExternal)oi.openExternal({href:url});\n'
    + '  });\n'
    + '}\n'
    + '\n'
    + 'if(oi.notifyIntrinsicHeight){\n'
    + '  setTimeout(function(){oi.notifyIntrinsicHeight(document.body.scrollHeight)},0);\n'
    + '}'
  );
}

// ── 5. Stats Widget (storage_stats) ────────────────────────────────────

export function statsWidget(): string {
  return widget(
    // CSS
    '.st{padding:16px}'
    + '\n.st-grid{display:grid;grid-template-columns:1fr 1fr;gap:12px}'
    + '\n.st-card{border:1px solid var(--border);border-radius:var(--radius);padding:16px}'
    + '\n.st-label{font-size:11px;font-weight:500;color:var(--text-3);text-transform:uppercase;letter-spacing:0.5px;margin-bottom:4px}'
    + '\n.st-value{font-size:24px;font-weight:700;letter-spacing:-0.5px}'
    + '\n.st-sub{font-size:12px;color:var(--text-2);margin-top:2px}'
    + '\n.st-icon{color:var(--text-3);margin-bottom:8px;display:flex}',
    // JS
    'var fileCount=data.file_count||0;\n'
    + 'var totalSize=data.total_size||0;\n'
    + 'var sizeHuman=data.total_size_human||humanSize(totalSize);\n'
    + '\n'
    + 'var html=\'<div class="st"><div class="st-grid">\';\n'
    + '\n'
    + '// File count card\n'
    + 'html+=\'<div class="st-card">\';\n'
    + 'html+=\'<div class="st-icon">\'+IC.file+"</div>";\n'
    + 'html+=\'<div class="st-label">Files</div>\';\n'
    + 'html+=\'<div class="st-value">\'+fileCount+"</div>";\n'
    + 'html+=\'<div class="st-sub">total files stored</div>\';\n'
    + 'html+="</div>";\n'
    + '\n'
    + '// Size card\n'
    + 'html+=\'<div class="st-card">\';\n'
    + 'html+=\'<div class="st-icon">\'+IC.chart+"</div>";\n'
    + 'html+=\'<div class="st-label">Storage</div>\';\n'
    + 'html+=\'<div class="st-value">\'+esc(sizeHuman)+"</div>";\n'
    + 'if(totalSize>0)html+=\'<div class="st-sub">\'+totalSize.toLocaleString()+" bytes</div>";\n'
    + 'html+="</div>";\n'
    + '\n'
    + 'html+="</div></div>";\n'
    + 'root.innerHTML=html;\n'
    + '\n'
    + 'if(oi.notifyIntrinsicHeight){\n'
    + '  setTimeout(function(){oi.notifyIntrinsicHeight(document.body.scrollHeight)},0);\n'
    + '}'
  );
}
