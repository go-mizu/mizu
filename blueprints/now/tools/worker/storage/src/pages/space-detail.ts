import { esc } from "./layout";

/**
 * /space/:id — Single Space detail page.
 * Two-column: main (Feed + Assets tabs) + sidebar (members, activity).
 * Feed: GitHub-style borderless activity items with action icons.
 * Assets: grouped sections with compact/grid toggle.
 */
export function spaceDetailPage(actor: string | null, spaceId: string): string {
  if (!actor) {
    return `<!DOCTYPE html><html><head><meta http-equiv="refresh" content="0;url=/spaces"></head></html>`;
  }
  const displayName = esc(actor.slice(2));
  const sid = esc(spaceId);

  return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Space — storage.now</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800&family=JetBrains+Mono:wght@400;500;600;700&display=swap" rel="stylesheet">
<link rel="stylesheet" href="/space.css">
</head>
<body>

<nav>
  <div class="nav-inner">
    <a href="/" class="logo"><span class="logo-dot"></span> storage.now</a>
    <button class="mobile-toggle" onclick="document.querySelector('.nav-links').classList.toggle('open')" aria-label="Menu">
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="3" y1="6" x2="21" y2="6"/><line x1="3" y1="12" x2="21" y2="12"/><line x1="3" y1="18" x2="21" y2="18"/></svg>
    </button>
    <div class="nav-links">
      <a href="/browse">Browse</a>
      <a href="/spaces">Spaces</a>
      <a href="/developers">Developers</a>
      <a href="/api">API</a>
      <a href="/pricing">Pricing</a>
      <a href="/ai">AI</a>
    </div>
    <div class="nav-right">
      <span class="nav-user">${displayName}</span>
      <a href="/auth/logout" class="nav-signout">sign out</a>
      <button class="theme-toggle" onclick="toggleTheme()" aria-label="Toggle theme">
        <svg class="icon-moon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 12.79A9 9 0 1111.21 3 7 7 0 0021 12.79z"/></svg>
        <svg class="icon-sun" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="5"/><line x1="12" y1="1" x2="12" y2="3"/><line x1="12" y1="21" x2="12" y2="23"/><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/><line x1="1" y1="12" x2="3" y2="12"/><line x1="21" y1="12" x2="23" y2="12"/><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/></svg>
      </button>
    </div>
  </div>
</nav>

<main id="space-root">
  <div class="space-loading">Loading space...</div>
</main>

<script>
/* ── Theme ────────────────────────────────────────────── */
function toggleTheme(){
  var isDark=document.documentElement.classList.toggle('dark');
  localStorage.setItem('theme',isDark?'dark':'light');
}
(function(){
  var saved=localStorage.getItem('theme');
  if(saved==='dark'||(!saved&&window.matchMedia('(prefers-color-scheme:dark)').matches))
    document.documentElement.classList.add('dark');
})();

/* ── Constants ────────────────────────────────────────── */
var SPACE_ID='${sid}';
var now=Date.now();
var spaceData=null;
var viewMode='compact';

/* ── Helpers ──────────────────────────────────────────── */
function esc(s){return(s||'').replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;')}

function relTime(ms){
  if(!ms)return'\u2014';
  var d=now-ms;
  if(d<60000)return'just now';
  if(d<3600000)return Math.floor(d/60000)+'m ago';
  if(d<86400000)return Math.floor(d/3600000)+'h ago';
  if(d<604800000)return Math.floor(d/86400000)+'d ago';
  return new Date(ms).toLocaleDateString('en-US',{month:'short',day:'numeric'});
}

function fmtSize(b){
  if(!b)return'\u2014';
  var u=['B','KB','MB','GB'];
  var i=Math.floor(Math.log(b)/Math.log(1024));
  return(b/Math.pow(1024,i)).toFixed(i>0?1:0)+' '+u[i];
}

function langFromCt(ct,name){
  if(!ct&&!name)return null;
  var ext=(name||'').split('.').pop().toLowerCase();
  var map={
    'application/typescript':'TS','text/typescript':'TS',
    'text/x-go':'Go','application/json':'JSON',
    'text/markdown':'MD','text/yaml':'YAML',
    'text/html':'HTML','text/css':'CSS',
    'application/javascript':'JS','text/javascript':'JS',
    'text/x-python':'PY','application/x-python':'PY',
    'text/x-rust':'RS','application/pdf':'PDF','text/plain':'TXT',
  };
  if(ct&&map[ct])return map[ct];
  var extMap={
    ts:'TS',tsx:'TS',js:'JS',jsx:'JS',go:'Go',py:'PY',rs:'RS',rb:'RB',
    json:'JSON',yaml:'YAML',yml:'YAML',toml:'TOML',
    md:'MD',html:'HTML',css:'CSS',sql:'SQL',sh:'SH',
    txt:'TXT',log:'TXT',pdf:'PDF',csv:'CSV',xml:'XML',
    doc:'DOC',docx:'DOCX',mp3:'MP3',wav:'WAV',ogg:'OGG',
  };
  return extMap[ext]||null;
}

function isImage(ct){return(ct||'').startsWith('image/')}

function initials(name){
  var clean=(name||'').replace(/^[ua]\//, '');
  var parts=clean.split(/[\\s@._-]+/).filter(Boolean);
  if(parts.length>=2)return(parts[0][0]+parts[1][0]).toUpperCase();
  return clean.slice(0,2).toUpperCase();
}

function stripActor(s){return(s||'').replace(/^[ua]\//, '')}

/* ── Markdown ─────────────────────────────────────────── */
function renderMd(src){
  if(!src)return'';
  src=src.replace(/\\n/g,'\n');
  var lines=src.split('\n');
  var html='',inCode=false,codeBuf='',inUl=false,inOl=false;
  function cl(){if(inUl){html+='</ul>';inUl=false}if(inOl){html+='</ol>';inOl=false}}
  for(var k=0;k<lines.length;k++){
    var line=lines[k];
    if(line.startsWith('\`\`\`')){
      if(inCode){html+='<pre><code>'+esc(codeBuf)+'</code></pre>';codeBuf='';inCode=false}
      else{cl();inCode=true}
      continue
    }
    if(inCode){codeBuf+=(codeBuf?'\n':'')+line;continue}
    if(!line.trim()){cl();continue}
    var hm=line.match(/^(#{1,4})\\s+(.+)/);
    if(hm){cl();var n=hm[1].length+1;html+='<h'+n+'>'+inl(hm[2])+'</h'+n+'>';continue}
    if(line.match(/^[-*]\\s/)){
      if(!inUl){cl();html+='<ul>';inUl=true}
      var c=line.slice(2);c=chk(c);
      html+='<li>'+inl(c)+'</li>';continue
    }
    if(line.match(/^\\d+\\.\\s/)){
      if(!inOl){cl();html+='<ol>';inOl=true}
      var c=line.replace(/^\\d+\\.\\s/,'');c=chk(c);
      html+='<li>'+inl(c)+'</li>';continue
    }
    cl();html+='<p>'+inl(line)+'</p>';
  }
  cl();
  if(inCode)html+='<pre><code>'+esc(codeBuf)+'</code></pre>';
  return html;
}
function chk(s){
  if(s.startsWith('[x] ')||s.startsWith('[X] '))return'<span class="md-check done">&#10003;</span> '+s.slice(4);
  if(s.startsWith('[ ] '))return'<span class="md-check">&#9675;</span> '+s.slice(4);
  return s;
}
function inl(s){
  return esc(s)
    .replace(/\\*\\*(.+?)\\*\\*/g,'<strong>$1</strong>')
    .replace(/\\*(.+?)\\*/g,'<em>$1</em>')
    .replace(/\`([^\`]+)\`/g,'<code>$1</code>');
}

/* ── Icons ────────────────────────────────────────────── */
var IC={
  code:'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/></svg>',
  file:'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>',
  doc:'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/></svg>',
  url:'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M18 13v6a2 2 0 01-2 2H5a2 2 0 01-2-2V8a2 2 0 012-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" y1="14" x2="21" y2="3"/></svg>',
  back:'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="19" y1="12" x2="5" y2="12"/><polyline points="12 19 5 12 12 5"/></svg>',
  chev:'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="6 9 12 15 18 9"/></svg>',
  agent:'<svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 2l2.4 7.2L22 12l-7.6 2.8L12 22l-2.4-7.2L2 12l7.6-2.8L12 2z"/></svg>',
  list:'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="8" y1="6" x2="21" y2="6"/><line x1="8" y1="12" x2="21" y2="12"/><line x1="8" y1="18" x2="21" y2="18"/><line x1="3" y1="6" x2="3.01" y2="6"/><line x1="3" y1="12" x2="3.01" y2="12"/><line x1="3" y1="18" x2="3.01" y2="18"/></svg>',
  grid:'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="14" y="14" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/></svg>',
  plus:'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="16"/><line x1="8" y1="12" x2="16" y2="12"/></svg>',
  pencil:'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M11 4H4a2 2 0 00-2 2v14a2 2 0 002 2h14a2 2 0 002-2v-7"/><path d="M18.5 2.5a2.12 2.12 0 013 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>',
  people:'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M17 21v-2a4 4 0 00-4-4H5a4 4 0 00-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 00-3-3.87"/><path d="M16 3.13a4 4 0 010 7.75"/></svg>',
  spark:'<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M12 2l2.4 7.2L22 12l-7.6 2.8L12 22l-2.4-7.2L2 12l7.6-2.8L12 2z"/></svg>',
};

function fileIcon(ct,name){
  if(!ct)ct='';
  var lang=langFromCt(ct,name);
  if(lang&&['TS','JS','Go','PY','RS','RB','JSON','YAML','TOML','HTML','CSS','SQL','SH','XML'].indexOf(lang)!==-1)return IC.code;
  if(lang==='MD')return IC.doc;
  if(lang==='PDF'||lang==='DOC'||lang==='DOCX')return IC.doc;
  return IC.file;
}

/* ── Format action (GitHub-style) ─────────────────────── */
function fmtAction(a){
  var action=a.action;
  var target=a.target;
  var icon='';
  var text='';

  if(action==='created'){
    icon=IC.spark;
    text='created <strong>'+esc(target)+'</strong>';
  } else if(action==='added_item'){
    icon=IC.plus;
    var lastTo=target.lastIndexOf(' to ');
    if(lastTo!==-1){
      text='added <strong>'+esc(target.substring(0,lastTo))+'</strong> to <strong>'+esc(target.substring(lastTo+4))+'</strong>';
    } else {
      text='added <strong>'+esc(target)+'</strong>';
    }
  } else if(action==='edited'){
    icon=IC.pencil;
    text='updated <strong>'+esc(target)+'</strong>';
  } else if(action==='shared'){
    icon=IC.people;
    var who=target.replace(/^with\\s+/,'');
    text='added <strong>'+esc(stripActor(who))+'</strong> as a member';
  } else if(action==='added_section'){
    icon=IC.plus;
    text='created section <strong>'+esc(target)+'</strong>';
  } else {
    text=esc(action).replace(/_/g,' ')+' <strong>'+esc(target)+'</strong>';
  }
  return{icon:icon,text:text};
}

/* ── Tab switching ────────────────────────────────────── */
function switchTab(t){
  document.querySelectorAll('.tab-btn').forEach(function(b){b.classList.toggle('active',b.dataset.tab===t)});
  document.querySelectorAll('.tab-panel').forEach(function(p){p.classList.toggle('active',p.id==='tab-'+t)});
}

/* ── View mode ────────────────────────────────────────── */
function setView(m){
  viewMode=m;
  document.querySelectorAll('.view-btn').forEach(function(b){b.classList.toggle('active',b.dataset.view===m)});
  if(spaceData)document.getElementById('assets-content').innerHTML=buildAssets(spaceData);
}

/* ── Section toggle ───────────────────────────────────── */
function toggleSec(id){
  var el=document.getElementById('sec-'+id);
  var ch=document.getElementById('chev-'+id);
  if(el.style.display==='none'){el.style.display='';ch.style.transform=''}
  else{el.style.display='none';ch.style.transform='rotate(-90deg)'}
}

/* ── Build Sidebar ────────────────────────────────────── */
function buildSidebar(data){
  var s=data.space;
  var allMembers=[
    {actor:(data.owner_info||{}).actor||s.owner,role:'owner',actor_type:(data.owner_info||{}).type||'human'}
  ].concat(data.members||[]);

  var html='';

  /* Members */
  html+='<div class="sidebar-card">';
  html+='<div class="sidebar-head">Members <span class="sidebar-count">'+allMembers.length+'</span></div>';
  html+='<div class="sb-members">';
  allMembers.forEach(function(m){
    var name=stripActor(m.actor);
    var isA=m.actor_type==='agent'||m.actor.startsWith('a/');
    html+='<div class="sb-member">';
    html+='<span class="sb-av'+(isA?' sb-av--agent':'')+'">'+
      (isA?'AI':initials(m.actor))+'</span>';
    html+='<span class="sb-name">'+esc(name)+'</span>';
    html+='<span class="sb-role">'+esc(m.role)+'</span>';
    if(isA)html+='<span class="sb-badge">'+IC.agent+'</span>';
    html+='</div>';
  });
  html+='</div></div>';

  /* Activity */
  html+='<div class="sidebar-card">';
  html+='<div class="sidebar-head">Activity</div>';
  html+='<div class="sb-activity-list">';
  (data.activity||[]).slice(0,8).forEach(function(a){
    var name=stripActor(a.actor);
    var fmt=fmtAction(a);
    html+='<div class="sb-act">';
    html+='<div class="sb-act-text"><strong>'+esc(name)+'</strong> '+fmt.text+'</div>';
    html+='<span class="sb-act-time">'+relTime(a.created_at)+'</span>';
    html+='</div>';
  });
  if(!(data.activity||[]).length){
    html+='<div class="sb-act"><div class="sb-act-text" style="color:var(--text-3)">No activity yet.</div></div>';
  }
  html+='</div></div>';

  return html;
}

/* ── Build Feed ───────────────────────────────────────── */
function buildFeed(data){
  var items=data.items||[];
  var activity=data.activity||[];

  var itemByTitle={};
  items.forEach(function(item){itemByTitle[item.title]=item});

  var html='<div class="feed-list">';
  activity.forEach(function(a){
    var name=stripActor(a.actor);
    var isA=(a.actor||'').startsWith('a/')||a.actor_type==='agent';
    var fmt=fmtAction(a);

    html+='<div class="feed-item">';

    /* Avatar */
    html+='<span class="feed-av'+(isA?' feed-av--agent':'')+'">'+
      (isA?'AI':initials(a.actor))+'</span>';

    /* Content */
    html+='<div class="feed-content">';
    html+='<div class="feed-line">';
    if(fmt.icon)html+='<span class="feed-ic">'+fmt.icon+'</span>';
    html+='<strong>'+esc(name)+'</strong> '+fmt.text;
    html+='</div>';
    html+='<div class="feed-time">'+relTime(a.created_at)+'</div>';

    /* Match activity to an item for rich content */
    var targetTitle;
    if(a.action==='added_item'){
      var lastTo=a.target.lastIndexOf(' to ');
      targetTitle=lastTo!==-1?a.target.substring(0,lastTo):a.target;
    } else {
      targetTitle=a.target;
    }
    var matched=itemByTitle[targetTitle]||itemByTitle[a.target]||null;

    if(matched){
      if(matched.item_type==='note'){
        html+='<div class="feed-note"><div class="note-body">'+renderMd(matched.note_body||matched.description||'')+'</div></div>';
      } else {
        html+=renderFeedAttach(matched);
      }
    }

    html+='</div>'; /* .feed-content */
    html+='</div>'; /* .feed-item */
  });
  html+='</div>';

  if(!activity.length){
    html='<div class="feed-empty">No activity yet.</div>';
  }
  return html;
}

/* ── Feed Attachment ──────────────────────────────────── */
function renderFeedAttach(item){
  if(item.item_type==='url'){
    var domain='';
    try{domain=new URL(item.url).hostname.replace('www.','');}catch(e){}
    return '<a class="feed-attach" href="'+esc(item.url)+'" target="_blank" rel="noopener">'+
      '<span class="feed-attach-ic">'+IC.url+'</span>'+
      '<span class="feed-attach-name">'+esc(item.title)+'</span>'+
      '<span class="feed-attach-meta">'+esc(domain)+'</span>'+
    '</a>';
  }
  if(item.item_type==='file'){
    var isImg=isImage(item.file_content_type);
    var lang=langFromCt(item.file_content_type,item.file_name||item.title);
    var name=item.file_name||item.title;
    var href=item.file_path?'/browse/'+encodeURI(item.file_path):null;

    if(isImg&&item.file_path){
      return '<div class="feed-img">'+
        (href?'<a href="'+esc(href)+'">':'')+
        '<img src="/files/'+encodeURI(item.file_path)+'" loading="lazy" alt="'+esc(name)+'">'+
        (href?'</a>':'')+
        '<div class="feed-img-cap">'+esc(name)+(item.file_size?' &middot; '+fmtSize(item.file_size):'')+'</div>'+
      '</div>';
    }

    var tag=href?'a':'div';
    return '<'+tag+' class="feed-attach"'+(href?' href="'+esc(href)+'"':'')+'>'+
      '<span class="feed-attach-ic">'+fileIcon(item.file_content_type,name)+'</span>'+
      '<span class="feed-attach-name">'+esc(name)+'</span>'+
      '<span class="feed-attach-meta">'+(lang||'')+(lang&&item.file_size?' &middot; ':'')+fmtSize(item.file_size)+'</span>'+
    '</'+tag+'>';
  }
  return '';
}

/* ── Build Assets ─────────────────────────────────────── */
function buildAssets(data){
  var sections=(data.sections||[]).slice().sort(function(a,b){return a.position-b.position});
  var secItems={};
  (data.items||[]).forEach(function(item){
    if(!secItems[item.section_id])secItems[item.section_id]=[];
    secItems[item.section_id].push(item);
  });

  var html='';
  sections.forEach(function(sec){
    var items=(secItems[sec.id]||[]).slice().sort(function(a,b){return a.position-b.position});
    if(!items.length)return;

    html+='<div class="asset-section">';
    html+='<div class="asset-sec-head" onclick="toggleSec(\''+esc(sec.id)+'\')"><span class="asset-sec-title">'+esc(sec.title)+'</span><span class="asset-sec-count">'+items.length+'</span><span class="asset-sec-chev" id="chev-'+esc(sec.id)+'">'+IC.chev+'</span></div>';
    html+='<div id="sec-'+esc(sec.id)+'">';

    if(viewMode==='grid'){
      html+='<div class="asset-grid">';
      items.forEach(function(item){html+=renderAssetCard(item)});
      html+='</div>';
    } else {
      html+='<div class="asset-list">';
      items.forEach(function(item){html+=renderAssetRow(item)});
      html+='</div>';
    }

    html+='</div></div>';
  });

  if(!html)html='<div class="feed-empty">No items yet.</div>';
  return html;
}

/* ── Asset: Compact Row ───────────────────────────────── */
function renderAssetRow(item){
  if(item.item_type==='note'){
    return '<div class="asset-row">'+
      '<span class="asset-row-icon">'+IC.doc+'</span>'+
      '<span class="asset-row-body"><span class="asset-row-lang">NOTE</span><span class="asset-row-name">'+esc(item.title)+'</span></span>'+
      '<span class="asset-row-size">\u2014</span>'+
      '<span class="asset-row-time">'+relTime(item.updated_at)+'</span>'+
    '</div>';
  }
  if(item.item_type==='url'){
    var domain='';
    try{domain=new URL(item.url).hostname.replace('www.','');}catch(e){}
    return '<a class="asset-row" href="'+esc(item.url)+'" target="_blank" rel="noopener">'+
      '<span class="asset-row-icon">'+IC.url+'</span>'+
      '<span class="asset-row-body"><span class="asset-row-name">'+esc(item.title)+'</span></span>'+
      '<span class="asset-row-size">\u2014</span>'+
      '<span class="asset-row-time">'+esc(domain)+'</span>'+
    '</a>';
  }
  var isImg=isImage(item.file_content_type);
  var lang=langFromCt(item.file_content_type,item.file_name||item.title);
  var name=item.file_name||item.title;
  var href=item.file_path?'/browse/'+encodeURI(item.file_path):null;
  var tag=href?'a':'div';
  var iconHtml;
  if(isImg&&item.file_path){
    iconHtml='<img src="/files/'+encodeURI(item.file_path)+'" loading="lazy" alt="">';
  } else {
    iconHtml=fileIcon(item.file_content_type,name);
  }

  return '<'+tag+' class="asset-row"'+(href?' href="'+esc(href)+'"':'')+'>'+
    '<span class="asset-row-icon">'+iconHtml+'</span>'+
    '<span class="asset-row-body">'+(lang?'<span class="asset-row-lang">'+lang+'</span>':'')+
      '<span class="asset-row-name">'+esc(name)+'</span></span>'+
    '<span class="asset-row-size">'+fmtSize(item.file_size)+'</span>'+
    '<span class="asset-row-time">'+relTime(item.updated_at)+'</span>'+
  '</'+tag+'>';
}

/* ── Asset: Grid Card ─────────────────────────────────── */
function renderAssetCard(item){
  if(item.item_type==='note'){
    return '<div class="asset-card">'+
      '<div class="asset-card-thumb">'+IC.doc+'<span style="font-family:var(--mono);font-size:10px;font-weight:600">NOTE</span></div>'+
      '<div class="asset-card-info"><div class="asset-card-name">'+esc(item.title)+'</div><div class="asset-card-meta">'+relTime(item.updated_at)+'</div></div>'+
    '</div>';
  }
  if(item.item_type==='url'){
    var domain='';
    try{domain=new URL(item.url).hostname.replace('www.','');}catch(e){}
    return '<a class="asset-card" href="'+esc(item.url)+'" target="_blank" rel="noopener">'+
      '<div class="asset-card-thumb">'+IC.url+'</div>'+
      '<div class="asset-card-info"><div class="asset-card-name">'+esc(item.title)+'</div><div class="asset-card-meta">'+esc(domain)+'</div></div>'+
    '</a>';
  }
  var isImg=isImage(item.file_content_type);
  var lang=langFromCt(item.file_content_type,item.file_name||item.title);
  var name=item.file_name||item.title;
  var href=item.file_path?'/browse/'+encodeURI(item.file_path):null;
  var tag=href?'a':'div';
  var thumbHtml;
  if(isImg&&item.file_path){
    thumbHtml='<img src="/files/'+encodeURI(item.file_path)+'" loading="lazy" alt="'+esc(name)+'">';
  } else {
    thumbHtml=fileIcon(item.file_content_type,name)+(lang?'<span style="font-family:var(--mono);font-size:10px;font-weight:600">'+lang+'</span>':'');
  }

  return '<'+tag+' class="asset-card"'+(href?' href="'+esc(href)+'"':'')+'>'+
    '<div class="asset-card-thumb">'+thumbHtml+'</div>'+
    '<div class="asset-card-info"><div class="asset-card-name">'+esc(name)+'</div><div class="asset-card-meta">'+fmtSize(item.file_size)+'</div></div>'+
  '</'+tag+'>';
}

/* ── Render Space ─────────────────────────────────────── */
function renderSpace(data){
  spaceData=data;
  var s=data.space;
  var root=document.getElementById('space-root');
  var totalItems=(data.items||[]).length;
  var totalSections=(data.sections||[]).length;

  var html='<div class="space-page">';

  /* Header */
  html+='<a href="/spaces" class="space-back">'+IC.back+' Spaces</a>';
  html+='<h1 class="space-title">'+esc(s.title)+'</h1>';
  if(s.description)html+='<p class="space-desc">'+esc(s.description)+'</p>';
  html+='<div class="space-meta">';
  html+='<span class="space-vis">'+esc(s.visibility)+'</span>';
  html+='<span>'+totalItems+' items</span>';
  html+='<span>'+totalSections+' sections</span>';
  html+='<span>Updated '+relTime(s.updated_at)+'</span>';
  html+='</div>';

  /* Two-column layout */
  html+='<div class="space-layout">';

  /* Main */
  html+='<div class="space-main">';

  html+='<div class="tab-bar">';
  html+='<button class="tab-btn active" data-tab="feed" onclick="switchTab(\'feed\')">Feed</button>';
  html+='<button class="tab-btn" data-tab="assets" onclick="switchTab(\'assets\')">Assets</button>';
  html+='</div>';

  html+='<div class="tab-panel active" id="tab-feed">';
  html+=buildFeed(data);
  html+='</div>';

  html+='<div class="tab-panel" id="tab-assets">';
  html+='<div class="view-bar">';
  html+='<span class="view-count">'+totalItems+' items &middot; '+totalSections+' sections</span>';
  html+='<div class="view-toggle">';
  html+='<button class="view-btn active" data-view="compact" onclick="setView(\'compact\')" title="List view">'+IC.list+'</button>';
  html+='<button class="view-btn" data-view="grid" onclick="setView(\'grid\')" title="Grid view">'+IC.grid+'</button>';
  html+='</div></div>';
  html+='<div id="assets-content">';
  html+=buildAssets(data);
  html+='</div></div>';

  html+='</div>'; /* .space-main */

  html+='<aside class="space-sidebar">';
  html+=buildSidebar(data);
  html+='</aside>';

  html+='</div>'; /* .space-layout */
  html+='</div>'; /* .space-page */

  root.innerHTML=html;
  document.title=esc(s.title)+' \u2014 storage.now';
}

/* ── Load ─────────────────────────────────────────────── */
async function loadSpace(){
  try{
    var res=await fetch('/spaces/'+SPACE_ID);
    if(!res.ok){
      var data=await res.json().catch(function(){return{}});
      document.getElementById('space-root').innerHTML=
        '<div class="space-page"><div class="space-error">'+
          '<h2>Space not found</h2>'+
          '<p>'+(data.error&&data.error.message||'This space does not exist or you do not have access.')+'</p>'+
          '<a href="/spaces" class="space-back">'+IC.back+' Back to Spaces</a>'+
        '</div></div>';
      return;
    }
    var data=await res.json();
    renderSpace(data);
  }catch(err){
    console.error(err);
    document.getElementById('space-root').innerHTML=
      '<div class="space-page"><div class="space-error"><h2>Failed to load</h2><p>'+esc(err.message)+'</p></div></div>';
  }
}

loadSpace();
</script>
</body>
</html>`;
}
