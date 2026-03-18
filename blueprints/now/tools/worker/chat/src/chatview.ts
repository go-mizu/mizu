import type { Context } from "hono";
import type { Env, Variables, MessageRow } from "./types";
import { humanAvatar, botAvatar } from "./avatar";
import { getSessionActor } from "./session";
import { isMember } from "./actor";
import { getBotProfile } from "./bots";

type AppContext = Context<{ Bindings: Env; Variables: Variables }>;

interface ChatMeta {
  id: string;
  kind: string;
  title: string;
  creator: string;
  visibility: string;
  created_at: number;
}

interface MemberRow {
  actor: string;
  role: string;
}

function esc(s: string): string {
  return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");
}

export async function chatViewPage(c: AppContext) {
  const chatId = c.req.param("chat_id") || "";
  const actor = await getSessionActor(c);
  if (!actor) return c.redirect("/");

  const chat = await c.env.DB.prepare(
    "SELECT id, kind, title, creator, visibility, created_at FROM chats WHERE id = ?"
  ).bind(chatId).first<ChatMeta>();

  if (!chat) return c.redirect("/inbox");

  const member = await isMember(c.env.DB, chatId, actor);
  if (!member) {
    if (chat.visibility === "private") return c.redirect("/inbox");
  }

  // Load members
  const { results: memberRows } = await c.env.DB.prepare(
    "SELECT actor, role FROM members WHERE chat_id = ? ORDER BY joined_at ASC"
  ).bind(chatId).all<MemberRow>();
  const members = memberRows || [];

  // Determine display name and peer (for DMs)
  let displayName: string;
  let peerActor: string | null = null;
  if (chat.kind === "direct") {
    peerActor = members.find(m => m.actor !== actor)?.actor || null;
    displayName = peerActor ? peerActor.slice(2) : "Direct message";
  } else {
    displayName = chat.title || "Untitled room";
  }

  // Load last 100 messages (ASC for display)
  const { results: msgRows } = await c.env.DB.prepare(
    `SELECT id, chat_id, actor, text, created_at FROM messages
     WHERE chat_id = ? ORDER BY created_at DESC LIMIT 100`
  ).bind(chatId).all<MessageRow>();
  const messages = (msgRows || []).reverse();

  // Build grouped message HTML (Discord/Slack-style)
  let msgHtml = "";
  if (messages.length > 0) {
    let prevActor = "";
    let prevTime = 0;
    let prevDateKey = "";

    for (const m of messages) {
      const d = new Date(m.created_at);
      const dateKey = d.toISOString().slice(0, 10);
      const isoTs = d.toISOString();
      const name = esc(m.actor.slice(2));
      const isMe = m.actor === actor;
      const isBot = m.actor.startsWith("a/");
      const cls = (isMe ? " mine" : "") + (isBot ? " bot" : "");

      // Day separator
      if (dateKey !== prevDateKey) {
        const label = d.toLocaleDateString("en-US", { month: "long", day: "numeric", year: "numeric" });
        msgHtml += `\n<div class="msg-day" data-date="${dateKey}">${label}</div>`;
        prevDateKey = dateKey;
        prevActor = "";
      }

      const sameAuthor = m.actor === prevActor;
      const withinWindow = (m.created_at - prevTime) < 300_000;
      const isCont = sameAuthor && withinWindow;

      if (isCont) {
        msgHtml += `\n<div class="msg msg-cont${cls}" data-ts="${isoTs}" data-actor="${esc(m.actor)}">
  <div class="msg-gutter"><span class="msg-hover-ts"></span></div>
  <div class="msg-content"><div class="msg-body" data-md="${esc(m.text)}"></div></div>
</div>`;
      } else {
        const av = isBot ? botAvatar(m.actor, 36) : humanAvatar(m.actor, 36);
        msgHtml += `\n<div class="msg${cls}" data-ts="${isoTs}" data-actor="${esc(m.actor)}">
  <div class="msg-gutter"><div class="msg-av">${av}</div></div>
  <div class="msg-content">
    <div class="msg-meta"><span class="msg-name">${name}</span><span class="msg-ts" data-ts="${isoTs}"></span></div>
    <div class="msg-body" data-md="${esc(m.text)}"></div>
  </div>
</div>`;
      }

      prevActor = m.actor;
      prevTime = m.created_at;
    }
  }

  // Header badge
  const memberCount = members.length;
  const isRoom = chat.kind === "room";
  const headerBadge = isRoom
    ? `<span class="cv-badge">${memberCount} members</span>`
    : peerActor?.startsWith("a/")
      ? `<span class="cv-badge">agent</span>`
      : `<span class="cv-badge">human</span>`;

  const myName = esc(actor.slice(2));
  const sendDisabled = !member ? `disabled title="Join to send messages"` : "";

  return c.html(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>${esc(displayName)} — chat.now</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800&family=JetBrains+Mono:wght@400;500;600;700&display=swap" rel="stylesheet">
<link rel="stylesheet" href="/chat.css">
<script src="https://cdn.jsdelivr.net/npm/marked@17.0.4/lib/marked.umd.js"></script>
</head>
<body>

<nav>
  <div class="nav-left">
    <a href="/inbox" class="back-btn">
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="15 18 9 12 15 6"/></svg>
      inbox
    </a>
    <span style="color:var(--border)">|</span>
    <span class="chat-name">${isRoom ? "# " : ""}${esc(displayName)}</span>
    ${headerBadge}
  </div>
  <div class="nav-right">
    <div class="sse-dot" id="sse-dot" title="Connecting..."></div>
    <span class="nav-user">${myName}</span>
    <button class="theme-toggle" onclick="toggleTheme()" aria-label="Toggle theme">
      <svg class="icon-moon" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 12.79A9 9 0 1111.21 3 7 7 0 0021 12.79z"/></svg>
      <svg class="icon-sun" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="5"/><line x1="12" y1="1" x2="12" y2="3"/><line x1="12" y1="21" x2="12" y2="23"/><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/><line x1="1" y1="12" x2="3" y2="12"/><line x1="21" y1="12" x2="23" y2="12"/><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/></svg>
    </button>
  </div>
</nav>

<div id="thread">
${(() => {
  if (msgHtml) return msgHtml;
  const profile = peerActor ? getBotProfile(peerActor) : null;
  if (peerActor && profile) {
    const chips = profile.examples
      .map(ex => `<button class="chip" onclick="fillInput(this)">${esc(ex)}</button>`)
      .join("");
    return `<div class="bot-welcome" id="bot-welcome">
  <div class="bot-welcome-avatar">${botAvatar(peerActor, 48)}</div>
  <div class="bot-welcome-name">${esc(peerActor.slice(2))}</div>
  <div class="bot-welcome-bio">${esc(profile.bio)}</div>
  <div class="bot-welcome-chips">${chips}</div>
</div>`;
  }
  return `<div style="flex:1;display:flex;align-items:center;justify-content:center;color:var(--text-3);font-size:14px">No messages yet. Say something!</div>`;
})()}
</div>

<div class="send-error" id="send-error"></div>
<div class="send-area">
  <textarea id="msg-input" placeholder="Message ${esc(displayName)}…" rows="1" ${sendDisabled}></textarea>
  <button id="send-btn" onclick="sendMsg()" ${sendDisabled}>Send</button>
</div>

<script>
var CHAT_ID=${JSON.stringify(chatId)};
var MY_ACTOR=${JSON.stringify(actor)};
var thread=document.getElementById('thread');
var sseDot=document.getElementById('sse-dot');

// Theme
function toggleTheme(){
  var d=document.documentElement.classList.toggle('dark');
  localStorage.setItem('theme',d?'dark':'light');
}
(function(){
  var s=localStorage.getItem('theme');
  if(s==='dark'||(!s&&window.matchMedia('(prefers-color-scheme:dark)').matches))
    document.documentElement.classList.add('dark');
})();

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

// Fill timestamps
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
</script>
</body>
</html>`);
}
