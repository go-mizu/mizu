import type { Context } from "hono";
import type { Env, Variables } from "./types";
import { humanAvatar, botAvatar } from "./avatar";
import { immersivePage, formatDate, relativeTime, esc } from "./layout";
import { getSessionActor } from "./session";

interface ChatRow {
  id: string;
  title: string;
  creator: string;
  visibility: string;
  created_at: number;
}

interface MemberRow {
  actor: string;
  role: string;
  joined_at: number;
}

interface MessageRow {
  id: string;
  actor: string;
  text: string;
  created_at: number;
}

export async function roomDetailPage(c: Context<{ Bindings: Env; Variables: Variables }>) {
  const roomId = c.req.param("room_id") || "";
  const sessionActor = await getSessionActor(c);

  const room = await c.env.DB.prepare(
    "SELECT id, title, creator, visibility, created_at FROM chats WHERE id = ? AND kind = 'room' AND visibility = 'public'"
  ).bind(roomId).first<ChatRow>();

  if (!room) {
    return c.html(immersivePage("Not found",
      `<div class="empty">Room not found or is private.</div>`,
      `<span class="h1"># Not found</span>\n\nRoom "${esc(roomId)}" does not exist or is private.`,
      sessionActor), 404);
  }

  // Fetch members and recent messages in parallel
  const [memberResult, messageResult] = await Promise.all([
    c.env.DB.prepare(
      "SELECT actor, role, joined_at FROM members WHERE chat_id = ? ORDER BY joined_at ASC"
    ).bind(roomId).all<MemberRow>(),
    c.env.DB.prepare(
      "SELECT id, actor, text, created_at FROM messages WHERE chat_id = ? ORDER BY created_at DESC LIMIT 20"
    ).bind(roomId).all<MessageRow>(),
  ]);

  const members = memberResult.results || [];
  const messages = (messageResult.results || []).reverse(); // oldest first
  const title = esc(room.title || "Untitled");
  const creator = esc(room.creator.slice(2));

  // Check membership
  const sessionIsMember = sessionActor
    ? members.some(m => m.actor === sessionActor)
    : false;

  // --- Member list ---
  let memberList = "";
  for (const m of members) {
    const mName = esc(m.actor.slice(2));
    const isAgent = m.actor.startsWith("a/");
    const avatar = isAgent ? botAvatar(m.actor, 28) : humanAvatar(m.actor, 28);
    const link = isAgent ? `/a/${encodeURIComponent(mName)}` : `/u/${encodeURIComponent(mName)}`;
    const roleTag = m.role === "admin" ? `<span class="room-role">admin</span>` : "";

    memberList += `
<a href="${link}" class="room-member">
  <div class="room-member-avatar">${avatar}</div>
  <span class="room-member-name">${mName}</span>
  <span class="dir-badge">${isAgent ? "agent" : "human"}</span>${roleTag}
</a>`;
  }

  // --- Recent messages ---
  let threadHtml = "";
  if (messages.length > 0) {
    let rows = "";
    for (const msg of messages) {
      const who = esc(msg.actor.slice(2));
      const text = esc(msg.text.length > 300 ? msg.text.slice(0, 300) + "..." : msg.text);
      rows += `
    <div class="room-msg">
      <div class="room-msg-who">${who}</div>
      <div class="room-msg-text">${text}</div>
    </div>`;
    }
    threadHtml = `
  <div class="pf-section">
    <div class="pf-section-label">RECENT MESSAGES</div>
    <div class="room-thread">${rows}
    </div>
  </div>`;
  }

  // --- Action / message form ---
  let actionHtml = "";
  if (!sessionActor) {
    actionHtml = `<div class="pf-section">
    <div class="pf-section-label">JOIN</div>
    <div class="signin-prompt"><a href="/">Sign in</a> to join and send messages in this room.</div>
  </div>`;
  } else if (!sessionIsMember) {
    actionHtml = `<div class="pf-section">
    <div class="pf-section-label">JOIN</div>
    <button onclick="joinRoom()" class="room-join-btn">Join this room</button>
    <div id="join-error" class="msg-error"></div>
  </div>
  <script>
  async function joinRoom(){
    try{
      const res=await fetch('/chats/${esc(roomId)}/join',{method:'POST'});
      if(!res.ok){const d=await res.json();throw new Error(d.error?.message||'Failed to join')}
      window.location.href='/chat/${esc(roomId)}';
    }catch(err){document.getElementById('join-error').textContent=err.message}
  }
  </script>`;
  } else {
    actionHtml = `<div class="pf-section">
    <div class="pf-section-label">MESSAGE</div>
    <a href="/chat/${esc(roomId)}" class="room-open-btn">
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15a2 2 0 01-2 2H7l-4 4V5a2 2 0 012-2h14a2 2 0 012 2z"/></svg>
      Open conversation
    </a>
    <div class="msg-form" id="msg-form" style="margin-top:16px">
      <textarea id="msg-text" placeholder="Message #${title}..." onkeydown="if(event.key==='Enter'&&!event.shiftKey){event.preventDefault();sendMsg()}"></textarea>
      <button onclick="sendMsg()">Send</button>
    </div>
    <div class="msg-error" id="msg-error"></div>
  </div>
  <script>
  async function sendMsg(){
    const text=document.getElementById('msg-text').value.trim();
    const errEl=document.getElementById('msg-error');
    errEl.textContent='';
    if(!text){errEl.textContent='Type a message.';return}
    try{
      const res=await fetch('/chats/${esc(roomId)}/messages',{
        method:'POST',
        headers:{'Content-Type':'application/json'},
        body:JSON.stringify({text:text})
      });
      const data=await res.json();
      if(!res.ok)throw new Error(data.error?.message||'Failed to send');
      document.getElementById('msg-text').value='';
      window.location.reload();
    }catch(err){errEl.textContent=err.message}
  }
  </script>`;
  }

  const humanContent = `
<a href="/rooms" class="back-link">&larr; Rooms</a>

<div class="pf-hero">
  <div class="pf-name" style="font-size:36px"># ${title}</div>
  <div class="pf-status">Created by ${creator} · ${members.length} member${members.length !== 1 ? "s" : ""} · ${messages.length} message${messages.length !== 1 ? "s" : ""}</div>
  <div class="pf-joined">${formatDate(room.created_at)}</div>
</div>

${threadHtml}

<div class="pf-section">
  <div class="pf-section-label">MEMBERS</div>
  <div class="room-members">${memberList}
  </div>
</div>

${actionHtml}`;

  const machineContent = `<span class="h1"># ${title}</span>

Room ID: ${esc(roomId)}
Created by: ${esc(room.creator)}
Members: ${members.length}
Messages: ${messages.length}

<span class="h2">## Join this room</span>

POST /chats/${esc(roomId)}/join
Authorization: Bearer &lt;token&gt;

<span class="h2">## Send a message</span>

POST /chats/${esc(roomId)}/messages
Authorization: Bearer &lt;token&gt;
{"text": "hello everyone"}

<span class="dim">&rarr; {"id":"m_...","chat_id":"${esc(roomId)}","text":"hello everyone",...}</span>

<span class="h2">## Read messages</span>

GET /chats/${esc(roomId)}/messages
Authorization: Bearer &lt;token&gt;

<span class="h2">## Members (${members.length})</span>

${members.map(m => `${m.actor}  ${m.role}`).join("\n")}`;

  return c.html(immersivePage(`#${title}`, humanContent, machineContent, sessionActor));
}
