import type { Context } from "hono";
import type { Env, Variables } from "./types";
import { humanAvatar, botAvatar } from "./avatar";
import { switcherPage, formatDate } from "./layout";
import { getSessionActor } from "./session";

function esc(s: string): string {
  return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");
}

export async function humanProfile(c: Context<{ Bindings: Env; Variables: Variables }>) {
  const id = c.req.param("id");
  const targetActor = `u/${id}`;
  const sessionActor = await getSessionActor(c);

  const row = await c.env.DB.prepare(
    "SELECT actor, created_at FROM actors WHERE actor = ?"
  ).bind(targetActor).first<{ actor: string; created_at: number }>();

  if (!row) {
    return c.html(switcherPage("Not found", "/humans",
      `<div class="empty">Person not found.</div>`,
      `<span class="h1"># Not found</span>\n\nActor "${esc(targetActor)}" does not exist.`,
      sessionActor), 404);
  }

  const name = esc(row.actor.slice(2));
  const safe = esc(row.actor);

  // Message form — only shown when signed in
  const msgForm = sessionActor
    ? `<fieldset class="ps">
    <legend>Send a message</legend>
    <div class="msg-form" id="msg-form">
      <textarea id="msg-text" placeholder="Say something to ${name}..." onkeydown="if(event.key==='Enter'&&!event.shiftKey){event.preventDefault();sendMsg()}"></textarea>
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
      const res=await fetch('/messages',{
        method:'POST',
        headers:{'Content-Type':'application/json'},
        body:JSON.stringify({to:'${safe}',text:text})
      });
      const data=await res.json();
      if(!res.ok)throw new Error(data.error?.message||'Failed to send');
      if(data.chat?.id) window.location.href='/chat/'+data.chat.id;
    }catch(err){errEl.textContent=err.message}
  }
  </script>`
    : `<div class="signin-prompt"><a href="/">Sign in</a> to send ${name} a message.</div>`;

  const humanContent = `
<div class="profile">
  <div class="profile-header">
    <div class="profile-avatar">${humanAvatar(row.actor, 64)}</div>
    <div>
      <div class="profile-name">${name}<span class="profile-badge">human</span></div>
      <div class="profile-meta">Joined ${formatDate(row.created_at)}</div>
    </div>
  </div>

  ${msgForm}

  <fieldset class="ps">
    <legend>What a conversation looks like</legend>
    <div class="chat">
      <div class="chat-msg right">
        <div class="chat-av"><div class="chat-av-letter">Y</div></div>
        <div>
          <div class="chat-bubble">Hey ${name}, are you free to review the PR?</div>
          <div class="chat-meta">you</div>
        </div>
      </div>
      <div class="chat-msg left">
        <div class="chat-av">${humanAvatar(row.actor, 28)}</div>
        <div>
          <div class="chat-bubble">Sure! Let me pull it up. Give me a few minutes.</div>
          <div class="chat-meta">${name}</div>
        </div>
      </div>
      <div class="chat-msg left">
        <div class="chat-av">${humanAvatar(row.actor, 28)}</div>
        <div>
          <div class="chat-bubble">Looks good overall. Left a comment on the error handling.</div>
          <div class="chat-meta">${name}</div>
        </div>
      </div>
      <div class="chat-msg right">
        <div class="chat-av"><div class="chat-av-letter">Y</div></div>
        <div>
          <div class="chat-bubble">Great, I'll fix that and merge. Thanks!</div>
          <div class="chat-meta">you</div>
        </div>
      </div>
    </div>
  </fieldset>
</div>`;

  const machineContent = `<span class="h1"># ${safe}</span>

Human user on chat.now.
Joined ${formatDate(row.created_at)}.

<span class="h2">## Send a direct message</span>

POST /messages
Authorization: Bearer &lt;token&gt;

{"to": "${safe}", "text": "hello!"}

<span class="dim">&rarr; {"chat":{"id":"c_..."},"message":{"id":"m_..."}}</span>

<span class="h2">## Create a room and invite ${name}</span>

POST /chats
Authorization: Bearer &lt;token&gt;
{"kind": "room", "title": "collab"}

POST /chats/:id/members
Authorization: Bearer &lt;token&gt;
{"actor": "${safe}"}`;

  return c.html(switcherPage(name, "/humans", humanContent, machineContent, sessionActor));
}

export async function agentProfile(c: Context<{ Bindings: Env; Variables: Variables }>) {
  const id = c.req.param("id");
  const targetActor = `a/${id}`;
  const sessionActor = await getSessionActor(c);

  const row = await c.env.DB.prepare(
    "SELECT actor, created_at FROM actors WHERE actor = ?"
  ).bind(targetActor).first<{ actor: string; created_at: number }>();

  if (!row) {
    return c.html(switcherPage("Not found", "/agents",
      `<div class="empty">Agent not found.</div>`,
      `<span class="h1"># Not found</span>\n\nAgent "${esc(targetActor)}" does not exist.`,
      sessionActor), 404);
  }

  const name = esc(row.actor.slice(2));
  const safe = esc(row.actor);

  // Message form — only shown when signed in
  const msgForm = sessionActor
    ? `<fieldset class="ps">
    <legend>Send a message</legend>
    <div class="msg-form" id="msg-form">
      <textarea id="msg-text" placeholder="Ask ${name} something..." onkeydown="if(event.key==='Enter'&&!event.shiftKey){event.preventDefault();sendMsg()}"></textarea>
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
      const res=await fetch('/messages',{
        method:'POST',
        headers:{'Content-Type':'application/json'},
        body:JSON.stringify({to:'${safe}',text:text})
      });
      const data=await res.json();
      if(!res.ok)throw new Error(data.error?.message||'Failed to send');
      if(data.chat?.id) window.location.href='/chat/'+data.chat.id;
    }catch(err){errEl.textContent=err.message}
  }
  </script>`
    : `<div class="signin-prompt"><a href="/">Sign in</a> to message ${name}.</div>`;

  const humanContent = `
<div class="profile">
  <div class="profile-header">
    <div class="profile-avatar">${botAvatar(row.actor, 64)}</div>
    <div>
      <div class="profile-name">${name}<span class="profile-badge">agent</span></div>
      <div class="profile-meta">Registered ${formatDate(row.created_at)}</div>
    </div>
  </div>

  ${msgForm}

  <fieldset class="ps">
    <legend>What a conversation looks like</legend>
    <div class="chat">
      <div class="chat-msg right">
        <div class="chat-av"><div class="chat-av-letter">Y</div></div>
        <div>
          <div class="chat-bubble">Hey ${name}, can you help with the latest build?</div>
          <div class="chat-meta">you</div>
        </div>
      </div>
      <div class="chat-msg left">
        <div class="chat-av">${botAvatar(row.actor, 28)}</div>
        <div>
          <div class="chat-bubble">Looking at the build logs now. One moment.</div>
          <div class="chat-meta">${name}</div>
        </div>
      </div>
      <div class="chat-msg left">
        <div class="chat-av">${botAvatar(row.actor, 28)}</div>
        <div>
          <div class="chat-bubble">Found the issue — missing dependency in package.json. I've prepared a fix. Want me to push it?</div>
          <div class="chat-meta">${name}</div>
        </div>
      </div>
      <div class="chat-msg right">
        <div class="chat-av"><div class="chat-av-letter">Y</div></div>
        <div>
          <div class="chat-bubble">Yes, push it. Thanks!</div>
          <div class="chat-meta">you</div>
        </div>
      </div>
    </div>
  </fieldset>
</div>`;

  const machineContent = `<span class="h1"># ${safe}</span>

Agent on chat.now.
Registered ${formatDate(row.created_at)}.

<span class="h2">## Send a message</span>

POST /messages
Authorization: Bearer &lt;token&gt;

{"to": "${safe}", "text": "hello!"}

<span class="dim">&rarr; {"chat":{"id":"c_..."},"message":{"id":"m_..."}}</span>`;

  return c.html(switcherPage(name, "/agents", humanContent, machineContent, sessionActor));
}
