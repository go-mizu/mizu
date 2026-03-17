import type { Context } from "hono";
import type { Env, Variables } from "./types";
import { humanAvatar, botAvatar } from "./avatar";
import { switcherPage, formatDate } from "./layout";
import { getSessionActor } from "./session";

function esc(s: string): string {
  return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");
}

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

export async function roomDetailPage(c: Context<{ Bindings: Env; Variables: Variables }>) {
  const roomId = c.req.param("room_id") || "";
  const sessionActor = await getSessionActor(c);

  const room = await c.env.DB.prepare(
    "SELECT id, title, creator, visibility, created_at FROM chats WHERE id = ? AND kind = 'room' AND visibility = 'public'"
  ).bind(roomId).first<ChatRow>();

  if (!room) {
    return c.html(switcherPage("Not found", "/rooms",
      `<div class="empty">Room not found or is private.</div>`,
      `<span class="h1"># Not found</span>\n\nRoom "${esc(roomId)}" does not exist or is private.`,
      sessionActor), 404);
  }

  const { results: memberRows } = await c.env.DB.prepare(
    "SELECT actor, role, joined_at FROM members WHERE chat_id = ? ORDER BY joined_at ASC"
  ).bind(roomId).all<MemberRow>();

  const members = memberRows || [];
  const title = esc(room.title || "Untitled");
  const creator = esc(room.creator.slice(2));
  const humans = members.filter(m => m.actor.startsWith("u/"));
  const agents = members.filter(m => m.actor.startsWith("a/"));

  // Member list
  let memberList = "";
  for (const m of members) {
    const name = esc(m.actor.slice(2));
    const isAgent = m.actor.startsWith("a/");
    const avatar = isAgent ? botAvatar(m.actor, 28) : humanAvatar(m.actor, 28);
    const badge = isAgent ? "agent" : "human";
    const link = isAgent ? `/a/${encodeURIComponent(name)}` : `/u/${encodeURIComponent(name)}`;
    const roleTag = m.role === "admin" ? ` <span style="color:var(--text-3);font-style:italic;font-size:11px">admin</span>` : "";

    memberList += `
<a href="${link}" style="display:flex;align-items:center;gap:10px;padding:8px 0;border-bottom:1px solid var(--border);transition:opacity .15s;text-decoration:none;color:inherit">
  <div style="width:28px;height:28px;flex-shrink:0;overflow:hidden">${avatar}</div>
  <span style="font-size:14px;font-weight:600">${name}</span>
  <span style="font-family:'JetBrains Mono',monospace;font-size:10px;color:var(--text-3);border:1px solid var(--border);padding:1px 6px">${badge}</span>${roleTag}
</a>`;
  }

  // Mock conversation using actual members
  const mockHuman = humans.length > 0 ? esc(humans[0].actor.slice(2)) : "alice";
  const mockAgent = agents.length > 0 ? esc(agents[0].actor.slice(2)) : "assistant";
  const mockHumanActor = humans.length > 0 ? humans[0].actor : "u/alice";
  const mockAgentActor = agents.length > 0 ? agents[0].actor : "a/assistant";

  // Check if sessionActor is a member of this room
  const sessionIsMember = sessionActor
    ? members.some(m => m.actor === sessionActor)
    : false;

  // Actions / message form based on auth + membership state
  let openChat = "";
  let msgForm = "";

  if (!sessionActor) {
    msgForm = `<div class="signin-prompt"><a href="/">Sign in</a> to join and send messages in this room.</div>`;
  } else if (!sessionIsMember) {
    msgForm = `<div style="margin-bottom:16px">
  <button onclick="joinRoom()" style="font-family:'JetBrains Mono',monospace;font-size:12px;padding:10px 24px;border:1px solid var(--ink,#111);background:var(--ink,#111);color:var(--bg,#fff);cursor:pointer;transition:opacity .15s" onmouseover="this.style.opacity='.8'" onmouseout="this.style.opacity='1'">Join room</button>
  <div id="join-error" style="font-family:'JetBrains Mono',monospace;font-size:11px;color:var(--err,#B91C1C);margin-top:8px"></div>
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
    openChat = `<a href="/chat/${esc(roomId)}" style="display:inline-flex;align-items:center;gap:8px;font-family:'JetBrains Mono',monospace;font-size:12px;padding:10px 20px;border:1px solid var(--ink,#111);background:var(--ink,#111);color:var(--bg,#fff);text-decoration:none;margin-bottom:16px;transition:opacity .15s" onmouseover="this.style.opacity='.8'" onmouseout="this.style.opacity='1'">
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15a2 2 0 01-2 2H7l-4 4V5a2 2 0 012-2h14a2 2 0 012 2z"/></svg>
  Open conversation
</a>`;
    msgForm = `<fieldset class="ps">
    <legend>Send a message</legend>
    <div style="font-size:13px;color:var(--text-3);margin-bottom:12px">
      Use <strong style="color:var(--text)">@name</strong> to tag someone in the room.
    </div>
    <div class="msg-form" id="msg-form">
      <textarea id="msg-text" placeholder="Message #${title}..." onkeydown="if(event.key==='Enter'&&!event.shiftKey){event.preventDefault();sendMsg()}"></textarea>
      <button onclick="sendMsg()">Send</button>
    </div>
    <div class="msg-sent" id="msg-sent">Message sent!</div>
    <div class="msg-error" id="msg-error"></div>
  </fieldset>
  <script>
  async function sendMsg(){
    const text=document.getElementById('msg-text').value.trim();
    const errEl=document.getElementById('msg-error');
    const sentEl=document.getElementById('msg-sent');
    errEl.textContent='';sentEl.classList.remove('show');
    if(!text){errEl.textContent='Type a message.';return}
    try{
      const res=await fetch('/chats/${esc(roomId)}/messages',{
        method:'POST',
        headers:{'Content-Type':'application/json'},
        body:JSON.stringify({text:text})
      });
      const data=await res.json();
      if(!res.ok)throw new Error(data.error?.message||'Failed to send');
      sentEl.classList.add('show');
      document.getElementById('msg-text').value='';
    }catch(err){errEl.textContent=err.message}
  }
  </script>`;
  }

  const humanContent = `
<div class="profile">
  <div style="margin-bottom:40px">
    <h1 style="font-size:28px;font-weight:700;letter-spacing:-0.5px;margin-bottom:8px"># ${title}</h1>
    <div style="font-family:'JetBrains Mono',monospace;font-size:12px;color:var(--text-3);display:flex;gap:16px;flex-wrap:wrap">
      <span>by ${creator}</span>
      <span>${members.length} member${members.length !== 1 ? "s" : ""}</span>
      <span>${formatDate(room.created_at)}</span>
    </div>
  </div>

  ${openChat}
  ${msgForm}

  <fieldset class="ps">
    <legend>Members</legend>
    <div style="border-top:1px solid var(--border)">${memberList}</div>
  </fieldset>

  <fieldset class="ps">
    <legend>Activity preview</legend>
    <div class="chat">
      <div class="chat-msg left">
        <div class="chat-av">${humanAvatar(mockHumanActor, 28)}</div>
        <div>
          <div class="chat-bubble">Can we get a status update on the latest changes?</div>
          <div class="chat-meta">${mockHuman}</div>
        </div>
      </div>
      <div class="chat-msg left">
        <div class="chat-av">${botAvatar(mockAgentActor, 28)}</div>
        <div>
          <div class="chat-bubble">All tests passing. 3 files changed, ready for review.</div>
          <div class="chat-meta">${mockAgent}</div>
        </div>
      </div>
      <div class="chat-msg left">
        <div class="chat-av">${humanAvatar(mockHumanActor, 28)}</div>
        <div>
          <div class="chat-bubble">Looks good, let's merge it.</div>
          <div class="chat-meta">${mockHuman}</div>
        </div>
      </div>
    </div>
  </fieldset>
</div>`;

  const machineContent = `<span class="h1"># ${title}</span>

Room ID: ${esc(roomId)}
Created by: ${esc(room.creator)}
Members: ${members.length}

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

  return c.html(switcherPage(`#${title}`, "/rooms", humanContent, machineContent, sessionActor));
}
