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
  const msgHtml = messages.map(m => {
    const t = new Date(m.created_at).toISOString(); // client JS will format
    const isMe = m.actor === actor;
    const name = esc(m.actor.slice(2));
    const text = esc(m.text);
    return `<div class="msg-row${isMe ? " mine" : ""}" data-ts="${t}">
  <span class="msg-time"></span>
  <span class="msg-author">${name}</span>
  <span class="msg-text">${text.replace(/\n/g, "<br>")}</span>
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
<style>
*,*::before,*::after{box-sizing:border-box;margin:0;padding:0}
:root{--bg:#FAFAF9;--surface:#FFF;--text:#111;--text-2:#666;--text-3:#999;--border:#DDD;--ink:#111;--err:#B91C1C}
html.dark{--bg:#0C0C0C;--surface:#161616;--text:#E5E5E5;--text-2:#888;--text-3:#555;--border:#2A2A2A;--ink:#E5E5E5;--err:#FCA5A5}
body{font-family:'DM Sans',system-ui,sans-serif;color:var(--text);background:var(--bg);
  -webkit-font-smoothing:antialiased;transition:background .3s,color .3s;
  display:flex;flex-direction:column;height:100dvh;overflow:hidden}
a{color:inherit;text-decoration:none}

/* Nav */
nav{padding:0 24px;height:52px;display:flex;align-items:center;justify-content:space-between;
  flex-shrink:0;border-bottom:1px solid var(--border);
  background:color-mix(in srgb,var(--bg) 95%,transparent);
  backdrop-filter:blur(12px);z-index:100}
.nav-left{display:flex;align-items:center;gap:12px}
.back-btn{font-family:'JetBrains Mono',monospace;font-size:12px;color:var(--text-3);
  display:flex;align-items:center;gap:6px;transition:color .15s}
.back-btn:hover{color:var(--text)}
.chat-name{font-weight:700;font-size:15px;letter-spacing:-0.3px}
.nav-right{display:flex;align-items:center;gap:10px}
.nav-user{font-family:'JetBrains Mono',monospace;font-size:11px;color:var(--text-3)}
.theme-toggle{background:none;border:1px solid var(--border);padding:5px 9px;
  cursor:pointer;color:var(--text-3);display:flex;align-items:center;transition:all .15s}
.theme-toggle:hover{color:var(--text);border-color:var(--text-3)}
.theme-toggle .icon-sun{display:none}.theme-toggle .icon-moon{display:block}
html.dark .theme-toggle .icon-sun{display:block}html.dark .theme-toggle .icon-moon{display:none}

/* Thread */
#thread{flex:1;overflow-y:auto;padding:16px 0;display:flex;flex-direction:column;gap:0}
.msg-row{display:grid;grid-template-columns:52px 120px 1fr;align-items:baseline;
  padding:2px 24px;min-height:24px;transition:background .1s}
.msg-row:hover{background:color-mix(in srgb,var(--text) 3%,transparent)}
.msg-time{font-family:'JetBrains Mono',monospace;font-size:11px;color:var(--text-3);
  flex-shrink:0;user-select:none}
.msg-author{font-family:'JetBrains Mono',monospace;font-size:12px;color:var(--text-3);
  font-weight:500;padding-right:12px;word-break:break-word}
.msg-row.mine .msg-author{color:var(--text);font-weight:700}
.msg-text{font-size:14px;line-height:1.6;color:var(--text);word-break:break-word;white-space:pre-wrap}

/* Day separator */
.day-sep{text-align:center;font-family:'JetBrains Mono',monospace;font-size:11px;
  color:var(--text-3);padding:12px 24px;display:flex;align-items:center;gap:12px}
.day-sep::before,.day-sep::after{content:'';flex:1;height:1px;background:var(--border)}

/* Send area */
.send-area{flex-shrink:0;border-top:1px solid var(--border);padding:12px 24px;display:flex;gap:10px;align-items:flex-end}
#msg-input{flex:1;font-family:'DM Sans',system-ui,sans-serif;font-size:14px;
  padding:10px 14px;border:1px solid var(--border);background:var(--bg);
  color:var(--text);outline:none;resize:none;min-height:44px;max-height:120px;
  line-height:1.5;transition:border-color .15s}
#msg-input:focus{border-color:var(--text-3)}
#msg-input::placeholder{color:var(--text-3)}
#send-btn{font-family:'JetBrains Mono',monospace;font-size:12px;
  padding:10px 20px;border:1px solid var(--ink);background:var(--ink);
  color:var(--bg);cursor:pointer;transition:opacity .15s;white-space:nowrap;flex-shrink:0;height:44px}
#send-btn:hover:not(:disabled){opacity:.8}
#send-btn:disabled{opacity:.35;cursor:not-allowed}
.send-error{font-family:'JetBrains Mono',monospace;font-size:11px;color:var(--err);
  padding:4px 24px 0;min-height:16px}

/* SSE status dot */
.sse-dot{width:6px;height:6px;border-radius:50%;background:var(--border);flex-shrink:0;transition:background .5s}
.sse-dot.live{background:#22c55e}
.sse-dot.error{background:var(--err)}

@media(max-width:640px){
  .msg-row{grid-template-columns:46px 88px 1fr;padding:2px 12px}
  .send-area{padding:8px 12px}
  .nav-user{display:none}
}
</style>
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

  const tx=document.createElement('span');
  tx.className='msg-text';
  tx.textContent=msg.text;

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
