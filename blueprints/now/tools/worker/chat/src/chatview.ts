import type { Context } from "hono";
import type { Env, Variables, MessageRow } from "./types";
import { humanAvatar, botAvatar } from "./avatar";
import { getSessionActor } from "./session";
import { isMember } from "./actor";

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

  if (!chat) return c.redirect("/chats");

  const member = await isMember(c.env.DB, chatId, actor);
  if (!member) {
    // For public rooms, allow view but not send; for private, redirect
    if (chat.visibility === "private") return c.redirect("/chats");
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

  // Build initial message HTML
  // data-md holds the raw text; client JS renders it via marked.parse()
  const msgHtml = messages.map(m => {
    const t = new Date(m.created_at).toISOString();
    const isMe = m.actor === actor;
    const name = esc(m.actor.slice(2));
    const isBot = m.actor.startsWith("a/");
    return `<div class="msg-row${isMe ? " mine" : ""}${isBot ? " bot" : ""}" data-ts="${t}">
  <span class="msg-time"></span>
  <span class="msg-author">${name}</span>
  <div class="msg-text" data-md="${esc(m.text)}"></div>
</div>`;
  }).join("\n");

  // Header badge / back button
  const memberCount = members.length;
  const isRoom = chat.kind === "room";
  const headerBadge = isRoom
    ? `<span style="font-family:'JetBrains Mono',monospace;font-size:11px;color:var(--text-3);border:1px solid var(--border);padding:2px 8px">${memberCount} members</span>`
    : peerActor?.startsWith("a/")
      ? `<span style="font-family:'JetBrains Mono',monospace;font-size:11px;color:var(--text-3);border:1px solid var(--border);padding:2px 8px">agent</span>`
      : `<span style="font-family:'JetBrains Mono',monospace;font-size:11px;color:var(--text-3);border:1px solid var(--border);padding:2px 8px">human</span>`;

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
<link href="https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;600;700&family=DM+Sans:wght@400;500;600;700&display=swap" rel="stylesheet">
<link rel="stylesheet" href="/chat.css">
<script src="https://cdn.jsdelivr.net/npm/marked@17.0.4/lib/marked.umd.js"></script>
</head>
<body>

<nav>
  <div class="nav-left">
    <a href="/my-chats" class="back-btn">
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="15 18 9 12 15 6"/></svg>
      chats
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
${msgHtml || `<div style="flex:1;display:flex;align-items:center;justify-content:center;color:var(--text-3);font-size:14px">No messages yet. Say something!</div>`}
</div>

<div class="send-error" id="send-error"></div>
<div class="send-area">
  <textarea id="msg-input" placeholder="Message ${esc(displayName)}…" rows="1" ${sendDisabled}></textarea>
  <button id="send-btn" onclick="sendMsg()" ${sendDisabled}>Send</button>
</div>

<script>
const CHAT_ID = ${JSON.stringify(chatId)};
const MY_ACTOR = ${JSON.stringify(actor)};
const thread = document.getElementById('thread');
const sseDot = document.getElementById('sse-dot');

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

// Format timestamps on existing messages
function fmtTime(iso){
  const d=new Date(iso);
  return d.toLocaleTimeString([],{hour:'2-digit',minute:'2-digit'});
}
document.querySelectorAll('.msg-row[data-ts]').forEach(row=>{
  row.querySelector('.msg-time').textContent=fmtTime(row.dataset.ts);
});

// Parse markdown and wrap any bare tables in a scrollable div
function renderMd(text){
  const html=marked.parse(text||'');
  return html.split('<table').join('<div class="table-wrap"><table').split('</table>').join('</table></div>');
}

// Render markdown on server-rendered messages
document.querySelectorAll('.msg-text[data-md]').forEach(el=>{
  el.innerHTML=renderMd(el.dataset.md);
});

// Scroll to bottom initially
thread.scrollTop=thread.scrollHeight;

// Check if near bottom
function atBottom(){return thread.scrollHeight-thread.scrollTop-thread.clientHeight<120}

// Append a message object from SSE or send response
function appendMsg(msg){
  const wasBottom=atBottom();
  const row=document.createElement('div');
  row.className='msg-row'+(msg.actor===MY_ACTOR?' mine':'');
  row.dataset.ts=msg.created_at;

  const t=document.createElement('span');
  t.className='msg-time';
  t.textContent=fmtTime(msg.created_at);

  const a=document.createElement('span');
  a.className='msg-author';
  a.textContent=msg.actor.slice(2);

  const tx=document.createElement('div');
  tx.className='msg-text';
  tx.innerHTML=renderMd(msg.text);

  row.appendChild(t);row.appendChild(a);row.appendChild(tx);
  thread.appendChild(row);
  if(wasBottom) thread.scrollTop=thread.scrollHeight;
}

// SSE
let sse;
function connectSSE(){
  sse=new EventSource('/sse/chats/'+CHAT_ID);
  sse.onopen=()=>{sseDot.className='sse-dot live';sseDot.title='Live'};
  sse.onmessage=(e)=>{
    try{appendMsg(JSON.parse(e.data))}catch{}
  };
  sse.onerror=()=>{
    sseDot.className='sse-dot error';sseDot.title='Reconnecting…';
  };
}
connectSSE();

// Send
async function sendMsg(){
  const input=document.getElementById('msg-input');
  const errEl=document.getElementById('send-error');
  const btn=document.getElementById('send-btn');
  const text=input.value.trim();
  if(!text)return;

  btn.disabled=true;
  errEl.textContent='';
  try{
    const res=await fetch('/chats/'+CHAT_ID+'/messages',{
      method:'POST',
      headers:{'Content-Type':'application/json'},
      body:JSON.stringify({text})
    });
    if(!res.ok){const d=await res.json();throw new Error(d.error?.message||'Failed')}
    input.value='';
    input.style.height='auto';
  }catch(e){
    errEl.textContent=e.message;
    setTimeout(()=>{errEl.textContent=''},4000);
  }finally{
    btn.disabled=false;
    input.focus();
  }
}

// Auto-grow textarea
document.getElementById('msg-input').addEventListener('input',function(){
  this.style.height='auto';
  this.style.height=Math.min(this.scrollHeight,120)+'px';
});

// Enter to send
document.getElementById('msg-input').addEventListener('keydown',function(e){
  if(e.key==='Enter'&&!e.shiftKey){e.preventDefault();sendMsg()}
});
</script>
</body>
</html>`);
}
