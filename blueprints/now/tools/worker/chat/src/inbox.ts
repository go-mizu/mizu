import type { Context } from "hono";
import type { Env, Variables, MessageRow } from "./types";
import { humanAvatar, botAvatar, roomIcon } from "./avatar";
import { formatDate, relativeTime, esc } from "./layout";
import { getSessionActor } from "./session";
import { isMember } from "./actor";
import { getBotProfile, listBotActors } from "./bots";
import { SITE_NAME } from "./constants";

type AppContext = Context<{ Bindings: Env; Variables: Variables }>;

interface ChatListRow {
  id: string;
  kind: string;
  title: string;
  creator: string;
  created_at: number;
  last_actor: string | null;
  last_text: string | null;
  last_at: number | null;
  member_count: number;
  peer_actor: string | null;
}

interface MemberRow {
  actor: string;
  role: string;
}

const FONT_LINK = `<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800&family=JetBrains+Mono:wght@400;500;600;700&display=swap" rel="stylesheet">`;

export async function inboxPage(c: AppContext) {
  const actor = await getSessionActor(c);
  if (!actor) return c.redirect("/");

  const chatId = c.req.param("chat_id") || null;

  // Always fetch conversation list
  const { results } = await c.env.DB.prepare(`
    SELECT c.id, c.kind, c.title, c.creator, c.created_at,
      (SELECT actor FROM messages WHERE chat_id=c.id ORDER BY created_at DESC LIMIT 1) as last_actor,
      (SELECT text  FROM messages WHERE chat_id=c.id ORDER BY created_at DESC LIMIT 1) as last_text,
      (SELECT created_at FROM messages WHERE chat_id=c.id ORDER BY created_at DESC LIMIT 1) as last_at,
      (SELECT COUNT(*) FROM members WHERE chat_id=c.id) as member_count,
      (CASE WHEN c.kind='direct'
        THEN (SELECT actor FROM members WHERE chat_id=c.id AND actor!=? LIMIT 1)
        ELSE NULL END) as peer_actor
    FROM chats c
    JOIN members m ON m.chat_id=c.id AND m.actor=?
    ORDER BY COALESCE(
      (SELECT created_at FROM messages WHERE chat_id=c.id ORDER BY created_at DESC LIMIT 1),
      c.created_at
    ) DESC
    LIMIT 50
  `).bind(actor, actor).all<ChatListRow>();

  const chats = results || [];

  // ── Build sidebar entries ──
  let entries = "";
  if (chats.length === 0) {
    entries = `<div class="ib-empty-list">No conversations yet</div>`;
  }
  for (const chat of chats) {
    const isDM = chat.kind === "direct";
    const peer = chat.peer_actor;

    // Display name
    let displayName: string;
    if (isDM && peer) displayName = peer.slice(2);
    else displayName = chat.title || "Untitled";

    // Avatar
    let avatar: string;
    if (isDM && peer?.startsWith("a/")) avatar = botAvatar(peer, 36);
    else if (isDM && peer?.startsWith("u/")) avatar = humanAvatar(peer, 36);
    else avatar = roomIcon(chat.title || "?", 36);

    // Type for filtering
    let entryType: string;
    if (isDM && peer?.startsWith("a/")) entryType = "agents";
    else if (isDM) entryType = "people";
    else entryType = "rooms";

    // Preview
    let preview = "";
    if (chat.last_text) {
      const who = chat.last_actor ? chat.last_actor.slice(2) + ": " : "";
      const text = chat.last_text.slice(0, 50) + (chat.last_text.length > 50 ? "..." : "");
      preview = esc(who + text);
    } else {
      preview = `<span style="color:var(--text-3)">No messages yet</span>`;
    }

    const ts = chat.last_at ? relativeTime(chat.last_at) : relativeTime(chat.created_at);
    const isActive = chat.id === chatId;
    const prefix = chat.kind === "room" ? "# " : "";

    entries += `
<a href="/inbox/${encodeURIComponent(chat.id)}" class="ib-entry${isActive ? " active" : ""}"
   data-type="${entryType}" data-name="${esc(displayName.toLowerCase())}">
  <div class="ib-entry-avatar">${avatar}</div>
  <div class="ib-entry-body">
    <div class="ib-entry-top">
      <span class="ib-entry-name">${prefix}${esc(displayName)}</span>
      <span class="ib-entry-time">${ts}</span>
    </div>
    <div class="ib-entry-preview">${preview}</div>
  </div>
</a>`;
  }

  // ── Build main panel ──
  let mainHtml: string;
  let activeDisplayName = "";

  if (chatId) {
    // Fetch chat metadata
    const chat = await c.env.DB.prepare(
      "SELECT id, kind, title, creator, visibility, created_at FROM chats WHERE id = ?"
    ).bind(chatId).first<{ id: string; kind: string; title: string; creator: string; visibility: string; created_at: number }>();

    if (!chat) {
      mainHtml = `<div class="ib-welcome"><div class="ib-welcome-title">Conversation not found</div></div>`;
    } else {
      const memberCheck = await isMember(c.env.DB, chatId, actor);
      if (!memberCheck && chat.visibility === "private") {
        return c.redirect("/inbox");
      }

      // Load members + messages in parallel
      const [memberResult, msgResult] = await Promise.all([
        c.env.DB.prepare("SELECT actor, role FROM members WHERE chat_id = ? ORDER BY joined_at ASC")
          .bind(chatId).all<MemberRow>(),
        c.env.DB.prepare("SELECT id, chat_id, actor, text, created_at FROM messages WHERE chat_id = ? ORDER BY created_at DESC LIMIT 100")
          .bind(chatId).all<MessageRow>(),
      ]);

      const members = memberResult.results || [];
      const messages = (msgResult.results || []).reverse();

      // Display name
      let peerActor: string | null = null;
      if (chat.kind === "direct") {
        peerActor = members.find(m => m.actor !== actor)?.actor || null;
        activeDisplayName = peerActor ? peerActor.slice(2) : "Direct message";
      } else {
        activeDisplayName = chat.title || "Untitled room";
      }

      const isRoom = chat.kind === "room";
      const badge = isRoom
        ? `<span class="ib-badge">${members.length} members</span>`
        : peerActor?.startsWith("a/")
          ? `<span class="ib-badge">agent</span>`
          : `<span class="ib-badge">human</span>`;

      // Messages HTML — Discord/Slack-style grouping
      let msgHtml = "";
      if (messages.length > 0) {
        let prevMsgActor = "";
        let prevMsgTime = 0;
        let prevDateKey = "";

        for (const m of messages) {
          const d = new Date(m.created_at);
          const dateKey = d.toISOString().slice(0, 10);
          const isoTs = d.toISOString();
          const name = esc(m.actor.slice(2));
          const letter = esc(m.actor.slice(2, 3).toUpperCase());
          const isMe = m.actor === actor;
          const isBot = m.actor.startsWith("a/");
          const cls = (isMe ? " mine" : "") + (isBot ? " bot" : "");

          // Day separator
          if (dateKey !== prevDateKey) {
            const label = d.toLocaleDateString("en-US", { month: "long", day: "numeric", year: "numeric" });
            msgHtml += `\n<div class="msg-day" data-date="${dateKey}">${label}</div>`;
            prevDateKey = dateKey;
            prevMsgActor = "";
          }

          // Group: same author within 5 minutes → continuation
          const sameAuthor = m.actor === prevMsgActor;
          const withinWindow = (m.created_at - prevMsgTime) < 300_000;
          const isCont = sameAuthor && withinWindow;

          if (isCont) {
            msgHtml += `\n<div class="msg msg-cont${cls}" data-ts="${isoTs}" data-actor="${esc(m.actor)}">
  <div class="msg-gutter"><span class="msg-hover-ts"></span></div>
  <div class="msg-content"><div class="msg-body" data-md="${esc(m.text)}"></div></div>
</div>`;
          } else {
            const av = isBot
              ? botAvatar(m.actor, 36)
              : humanAvatar(m.actor, 36);
            msgHtml += `\n<div class="msg${cls}" data-ts="${isoTs}" data-actor="${esc(m.actor)}">
  <div class="msg-gutter"><div class="msg-av">${av}</div></div>
  <div class="msg-content">
    <div class="msg-meta"><span class="msg-name">${name}</span><span class="msg-ts" data-ts="${isoTs}"></span></div>
    <div class="msg-body" data-md="${esc(m.text)}"></div>
  </div>
</div>`;
          }

          prevMsgActor = m.actor;
          prevMsgTime = m.created_at;
        }
      } else {
        // Empty state / bot welcome
        const profile = peerActor ? getBotProfile(peerActor) : null;
        if (peerActor && profile) {
          const chips = profile.examples
            .map(ex => `<button class="chip" onclick="fillInput(this)">${esc(ex)}</button>`)
            .join("");
          msgHtml = `<div class="bot-welcome" id="bot-welcome">
  <div class="bot-welcome-avatar">${botAvatar(peerActor, 48)}</div>
  <div class="bot-welcome-name">${esc(peerActor.slice(2))}</div>
  <div class="bot-welcome-bio">${esc(profile.bio)}</div>
  <div class="bot-welcome-chips">${chips}</div>
</div>`;
        } else {
          msgHtml = `<div class="ib-no-messages">No messages yet. Say something!</div>`;
        }
      }

      const sendDisabled = !memberCheck ? ` disabled title="Join to send"` : "";
      const chatPrefix = isRoom ? "# " : "";

      mainHtml = `
<div class="ib-chat-header">
  <a href="/inbox" class="ib-back">
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="15 18 9 12 15 6"/></svg>
  </a>
  <span class="ib-chat-name">${chatPrefix}${esc(activeDisplayName)}</span>
  ${badge}
  <div class="ib-chat-right">
    <div class="sse-dot" id="sse-dot" title="Connecting..."></div>
  </div>
</div>
<div id="thread" class="ib-thread">
  ${msgHtml}
</div>
<div class="ib-send-error" id="send-error"></div>
<div class="ib-send">
  <textarea id="msg-input" placeholder="Message ${esc(activeDisplayName)}..." rows="1"${sendDisabled}></textarea>
  <button id="send-btn" onclick="sendMsg()"${sendDisabled}>Send</button>
</div>`;
    }
  } else {
    // ── Welcome / compose panel ──
    const botActors = listBotActors();
    let agentCards = "";
    for (const ba of botActors) {
      const profile = getBotProfile(ba);
      if (!profile) continue;
      const bname = ba.slice(2);
      agentCards += `
<a href="/a/${encodeURIComponent(bname)}" class="ib-agent-card">
  <div class="ib-agent-card-avatar">${botAvatar(ba, 32)}</div>
  <div class="ib-agent-card-body">
    <div class="ib-agent-card-name">${esc(bname)}</div>
    <div class="ib-agent-card-bio">${esc(profile.bio.length > 60 ? profile.bio.slice(0, 60) + "..." : profile.bio)}</div>
  </div>
  <span class="ib-agent-card-arrow">&rarr;</span>
</a>`;
    }

    mainHtml = `
<div class="ib-welcome">
  <div class="ib-welcome-hero">
    <svg class="ib-welcome-icon" width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
      <path d="M4 4h16c1.1 0 2 .9 2 2v12c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V6c0-1.1.9-2 2-2z"/>
      <polyline points="22,6 12,13 2,6"/>
    </svg>
    <div class="ib-welcome-title">One inbox</div>
    <div class="ib-welcome-sub">Messages from people, agents, and rooms — all here.</div>
  </div>

  ${agentCards ? `
  <div class="ib-welcome-section">
    <div class="ib-welcome-label">TALK TO AN AGENT</div>
    <div class="ib-welcome-agents">${agentCards}</div>
  </div>` : ""}

  <div class="ib-welcome-section">
    <div class="ib-welcome-label">BROWSE</div>
    <div class="ib-welcome-nav">
      <a href="/humans" class="ib-welcome-link">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M20 21v-2a4 4 0 00-4-4H8a4 4 0 00-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>
        People
      </a>
      <a href="/agents" class="ib-welcome-link">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="4" y="4" width="16" height="16" rx="2"/><line x1="9" y1="9" x2="9.01" y2="9"/><line x1="15" y1="9" x2="15.01" y2="9"/><path d="M8 14s1.5 2 4 2 4-2 4-2"/></svg>
        Agents
      </a>
      <a href="/rooms" class="ib-welcome-link">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 15a2 2 0 01-2 2H7l-4 4V5a2 2 0 012-2h14a2 2 0 012 2z"/></svg>
        Rooms
      </a>
    </div>
  </div>
</div>`;
  }

  const myName = esc(actor.slice(2));
  const hasChatClass = chatId ? " has-chat" : "";
  const pageTitle = chatId && activeDisplayName
    ? `${activeDisplayName} — Inbox — ${SITE_NAME}`
    : `Inbox — ${SITE_NAME}`;

  return c.html(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>${esc(pageTitle)}</title>
${FONT_LINK}
<link rel="stylesheet" href="/inbox.css">
${chatId ? '<script src="https://cdn.jsdelivr.net/npm/marked@17.0.4/lib/marked.umd.js"></script>' : ""}
</head>
<body class="${hasChatClass}">

<header class="ib-header">
  <a href="/" class="ib-logo">${SITE_NAME}</a>
  <div class="ib-header-right">
    <span class="ib-user">${myName}</span>
    <a href="/auth/logout" class="ib-signout">sign out</a>
    <button class="ib-theme" onclick="toggleTheme()" aria-label="Toggle theme">
      <svg class="icon-moon" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 12.79A9 9 0 1111.21 3 7 7 0 0021 12.79z"/></svg>
      <svg class="icon-sun" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="5"/><line x1="12" y1="1" x2="12" y2="3"/><line x1="12" y1="21" x2="12" y2="23"/><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/><line x1="1" y1="12" x2="3" y2="12"/><line x1="21" y1="12" x2="23" y2="12"/><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/></svg>
    </button>
  </div>
</header>

<div class="ib-layout">
  <aside class="ib-sidebar">
    <div class="ib-sidebar-top">
      <div class="ib-sidebar-header">
        <span class="ib-sidebar-title">Inbox</span>
        <a href="/inbox" class="ib-new" title="New conversation">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
        </a>
      </div>
      <div class="ib-search">
        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
        <input type="text" placeholder="Search..." oninput="searchInbox(this.value)" autocomplete="off" spellcheck="false">
      </div>
      <div class="ib-filters">
        <button class="ib-filter active" onclick="filterInbox('all',this)">All</button>
        <button class="ib-filter" onclick="filterInbox('people',this)">People</button>
        <button class="ib-filter" onclick="filterInbox('agents',this)">Agents</button>
        <button class="ib-filter" onclick="filterInbox('rooms',this)">Rooms</button>
      </div>
    </div>
    <div class="ib-list">
      ${entries}
    </div>
  </aside>

  <main class="ib-main">
    ${mainHtml}
  </main>
</div>

<script>
// Theme
function toggleTheme(){
  const d=document.documentElement.classList.toggle('dark');
  localStorage.setItem('theme',d?'dark':'light');
}
(function(){
  const s=localStorage.getItem('theme');
  if(s==='dark'||(!s&&window.matchMedia('(prefers-color-scheme:dark)').matches))
    document.documentElement.classList.add('dark');
})();

// Search
function searchInbox(q){
  q=q.toLowerCase();
  document.querySelectorAll('.ib-entry').forEach(function(el){
    el.style.display=el.dataset.name.includes(q)?'':'none';
  });
}

// Filter
function filterInbox(type,btn){
  document.querySelectorAll('.ib-filter').forEach(function(b){b.classList.remove('active')});
  btn.classList.add('active');
  document.querySelectorAll('.ib-entry').forEach(function(el){
    el.style.display=(type==='all'||el.dataset.type===type)?'':'none';
  });
}

${chatId ? `
// ── Conversation JS ──
var CHAT_ID=${JSON.stringify(chatId)};
var MY_ACTOR=${JSON.stringify(actor)};
var thread=document.getElementById('thread');
var sseDot=document.getElementById('sse-dot');

// Time formatting
function fmtTime(iso){
  var d=new Date(iso);
  return d.toLocaleTimeString([],{hour:'numeric',minute:'2-digit'});
}
function fmtDayLabel(dateStr){
  var d=new Date(dateStr+'T12:00:00');
  var now=new Date();now.setHours(0,0,0,0);
  var t=new Date(d);t.setHours(0,0,0,0);
  var diff=now-t;
  if(diff>=0&&diff<86400000)return'Today';
  if(diff>=86400000&&diff<172800000)return'Yesterday';
  return d.toLocaleDateString([],{month:'long',day:'numeric',year:'numeric'});
}

// Fill timestamps on server-rendered messages
document.querySelectorAll('.msg-ts[data-ts]').forEach(function(el){
  el.textContent=fmtTime(el.dataset.ts);
});
document.querySelectorAll('.msg-cont[data-ts]').forEach(function(el){
  var h=el.querySelector('.msg-hover-ts');
  if(h)h.textContent=fmtTime(el.dataset.ts);
});
document.querySelectorAll('.msg-day[data-date]').forEach(function(el){
  el.textContent=fmtDayLabel(el.dataset.date);
});

// Markdown
function renderMd(text){
  var html=marked.parse(text||'');
  return html.split('<table').join('<div class="table-wrap"><table').split('</table>').join('</table></div>');
}
document.querySelectorAll('.msg-body[data-md]').forEach(function(el){
  el.innerHTML=renderMd(el.dataset.md);
});

// Scroll
thread.scrollTop=thread.scrollHeight;
function atBottom(){return thread.scrollHeight-thread.scrollTop-thread.clientHeight<120}

// Track last message for SSE grouping
var lastActor='',lastTime=0;
(function(){
  var msgs=document.querySelectorAll('.msg[data-actor]');
  if(msgs.length){
    var last=msgs[msgs.length-1];
    lastActor=last.dataset.actor;
    lastTime=new Date(last.dataset.ts).getTime();
  }
})();

// Append SSE message with grouping
function appendMsg(msg){
  var welcome=document.getElementById('bot-welcome');
  if(welcome)welcome.remove();
  var noMsg=document.querySelector('.ib-no-messages');
  if(noMsg)noMsg.remove();
  var wasBottom=atBottom();
  var ts=msg.created_at;
  var time=new Date(ts).getTime();
  var isMe=msg.actor===MY_ACTOR;
  var isBot=msg.actor.startsWith('a/');
  var name=msg.actor.slice(2);
  var letter=name.charAt(0).toUpperCase();
  var cls='msg'+(isMe?' mine':'')+(isBot?' bot':'');

  var sameAuthor=msg.actor===lastActor;
  var withinWindow=(time-lastTime)<300000;
  var isCont=sameAuthor&&withinWindow;

  var row=document.createElement('div');
  if(isCont){
    row.className=cls+' msg-cont';
    row.dataset.ts=ts;row.dataset.actor=msg.actor;
    row.innerHTML='<div class="msg-gutter"><span class="msg-hover-ts">'+fmtTime(ts)+'</span></div>'+
      '<div class="msg-content"><div class="msg-body"></div></div>';
  }else{
    row.className=cls;
    row.dataset.ts=ts;row.dataset.actor=msg.actor;
    row.innerHTML='<div class="msg-gutter"><div class="msg-av"><div class="msg-av-letter'+(isBot?' bot':'')+'">'+letter+'</div></div></div>'+
      '<div class="msg-content"><div class="msg-meta"><span class="msg-name">'+name+'</span><span class="msg-ts">'+fmtTime(ts)+'</span></div>'+
      '<div class="msg-body"></div></div>';
  }
  row.querySelector('.msg-body').innerHTML=renderMd(msg.text);
  thread.appendChild(row);
  if(wasBottom)thread.scrollTop=thread.scrollHeight;
  lastActor=msg.actor;lastTime=time;
}

function fillInput(btn){
  var input=document.getElementById('msg-input');
  input.value=btn.textContent;input.focus();
  input.style.height='auto';input.style.height=Math.min(input.scrollHeight,120)+'px';
}

// SSE
var sse;
function connectSSE(){
  sse=new EventSource('/sse/chats/'+CHAT_ID);
  sse.onopen=function(){sseDot.className='sse-dot live';sseDot.title='Live'};
  sse.onmessage=function(e){try{appendMsg(JSON.parse(e.data))}catch(x){}};
  sse.onerror=function(){sseDot.className='sse-dot error';sseDot.title='Reconnecting...'};
}
connectSSE();

// Send
async function sendMsg(){
  var input=document.getElementById('msg-input');
  var errEl=document.getElementById('send-error');
  var btn=document.getElementById('send-btn');
  var text=input.value.trim();
  if(!text)return;
  btn.disabled=true;errEl.textContent='';
  try{
    var res=await fetch('/chats/'+CHAT_ID+'/messages',{
      method:'POST',headers:{'Content-Type':'application/json'},
      body:JSON.stringify({text:text})
    });
    if(!res.ok){var d=await res.json();throw new Error(d.error?.message||'Failed')}
    input.value='';input.style.height='auto';
  }catch(e){
    errEl.textContent=e.message;
    setTimeout(function(){errEl.textContent=''},4000);
  }finally{btn.disabled=false;input.focus()}
}

// Auto-grow textarea
document.getElementById('msg-input').addEventListener('input',function(){
  this.style.height='auto';this.style.height=Math.min(this.scrollHeight,120)+'px';
});
document.getElementById('msg-input').addEventListener('keydown',function(e){
  if(e.key==='Enter'&&!e.shiftKey){e.preventDefault();sendMsg()}
});
` : ""}
</script>
</body>
</html>`);
}
