import type { Context } from "hono";
import type { Env, Variables } from "./types";
import { humanAvatar, botAvatar } from "./avatar";
import { immersivePage, formatDate, relativeTime, esc } from "./layout";
import { getSessionActor } from "./session";
import { getBotProfile } from "./bots";

const QR_SCRIPT = `<script src="https://cdn.jsdelivr.net/npm/qrcode-generator@1.4.4/qrcode.min.js"></script>`;

export async function humanProfile(c: Context<{ Bindings: Env; Variables: Variables }>) {
  const id = c.req.param("id");
  const targetActor = `u/${id}`;
  const sessionActor = await getSessionActor(c);

  const row = await c.env.DB.prepare(
    "SELECT actor, bio, created_at FROM actors WHERE actor = ?"
  ).bind(targetActor).first<{ actor: string; bio: string | null; created_at: number }>();

  if (!row) {
    return c.html(immersivePage("Not found",
      `<div class="empty">Person not found.</div>`,
      `<span class="h1"># Not found</span>\n\nActor "${esc(targetActor)}" does not exist.`,
      sessionActor), 404);
  }

  const name = esc(row.actor.slice(2));
  const safe = esc(row.actor);
  const bio = row.bio || "";
  const profileUrl = `${new URL(c.req.url).origin}/u/${encodeURIComponent(row.actor.slice(2))}`;

  // --- Fetch stats ---
  const stats = await c.env.DB.prepare(`
    SELECT
      (SELECT COUNT(*) FROM messages WHERE actor = ?) as msg_count,
      (SELECT COUNT(*) FROM members m JOIN chats c ON c.id = m.chat_id
       WHERE m.actor = ? AND c.kind = 'room') as room_count,
      (SELECT COUNT(DISTINCT chat_id) FROM members WHERE actor = ?) as thread_count,
      (SELECT MAX(created_at) FROM messages WHERE actor = ?) as last_active
  `).bind(targetActor, targetActor, targetActor, targetActor)
    .first<{ msg_count: number; room_count: number; thread_count: number; last_active: number | null }>();

  const msgCount = stats?.msg_count ?? 0;
  const roomCount = stats?.room_count ?? 0;
  const threadCount = stats?.thread_count ?? 0;
  const lastActive = stats?.last_active ?? null;

  // --- Fetch public rooms ---
  const { results: roomResults } = await c.env.DB.prepare(`
    SELECT c.id, c.title,
      (SELECT COUNT(*) FROM members WHERE chat_id = c.id) as member_count
    FROM chats c
    JOIN members m ON m.chat_id = c.id
    WHERE m.actor = ? AND c.kind = 'room' AND c.visibility = 'public'
    ORDER BY c.created_at DESC
  `).bind(targetActor).all<{ id: string; title: string; member_count: number }>();
  const rooms = roomResults || [];

  // --- If signed in: mutual rooms + existing DM ---
  let mutualRooms: { id: string; title: string }[] = [];
  let existingDmId: string | null = null;

  if (sessionActor && sessionActor !== targetActor) {
    const [mutualResult, dmResult] = await Promise.all([
      c.env.DB.prepare(`
        SELECT c.id, c.title FROM chats c
        JOIN members m1 ON m1.chat_id = c.id AND m1.actor = ?
        JOIN members m2 ON m2.chat_id = c.id AND m2.actor = ?
        WHERE c.kind = 'room'
      `).bind(targetActor, sessionActor).all<{ id: string; title: string }>(),
      c.env.DB.prepare(`
        SELECT c.id FROM chats c
        JOIN members m1 ON m1.chat_id = c.id AND m1.actor = ?
        JOIN members m2 ON m2.chat_id = c.id AND m2.actor = ?
        WHERE c.kind = 'direct'
        LIMIT 1
      `).bind(targetActor, sessionActor).first<{ id: string }>(),
    ]);
    mutualRooms = mutualResult.results || [];
    existingDmId = dmResult?.id || null;
  }

  // --- Activity status ---
  const isActive = lastActive && (Date.now() - lastActive < 3_600_000);
  const statusLine = lastActive
    ? `<span class="status-dot${isActive ? " on" : ""}"></span> Active ${relativeTime(lastActive)}`
    : `Joined ${formatDate(row.created_at)}`;

  // --- Stats line for namecard ---
  const statParts: string[] = [];
  if (msgCount > 0) statParts.push(`${msgCount} message${msgCount !== 1 ? "s" : ""}`);
  if (roomCount > 0) statParts.push(`${roomCount} room${roomCount !== 1 ? "s" : ""}`);
  if (threadCount > 0) statParts.push(`${threadCount} thread${threadCount !== 1 ? "s" : ""}`);

  // --- Namecard ---
  const namecard = `
<div class="namecard">
  <div class="namecard-brand">chat.now</div>
  <div class="namecard-avatar">${humanAvatar(row.actor, 64)}</div>
  <div class="namecard-name">${name}</div>
  ${bio ? `<div class="namecard-bio">${esc(bio)}</div>` : ""}
  <div class="namecard-divider"></div>
  <div class="namecard-bottom">
    <div class="namecard-meta">
      ${statParts.length > 0 ? statParts.join(" · ") + "<br>" : ""}${statusLine}<br>
      Joined ${formatDate(row.created_at)}
    </div>
    <div class="namecard-qr" id="namecard-qr"></div>
  </div>
  <div class="namecard-url">${esc(profileUrl)}</div>
</div>

<div class="namecard-actions">
  <button class="namecard-action" onclick="copyLink()">Copy link</button>
  <button class="namecard-action" id="share-btn" onclick="shareCard()" style="display:none">Share</button>
</div>

${QR_SCRIPT}
<script>
(function(){
  var qr = qrcode(0, 'L');
  qr.addData('${profileUrl}');
  qr.make();
  document.getElementById('namecard-qr').innerHTML = qr.createSvgTag(3, 0);
})();
function copyLink(){
  navigator.clipboard.writeText('${profileUrl}').then(function(){
    var b=document.querySelector('.namecard-action');
    b.textContent='Copied!';setTimeout(function(){b.textContent='Copy link'},2000);
  });
}
if(navigator.share) document.getElementById('share-btn').style.display='';
function shareCard(){
  navigator.share({title:'${name} on chat.now',url:'${profileUrl}'});
}
</script>`;

  // --- Rooms section ---
  let roomsSection = "";
  if (rooms.length > 0) {
    roomsSection = `
  <div class="pf-section">
    <div class="pf-section-label">ROOMS</div>
    <div class="pf-rooms">
      ${rooms.map(r => `<a href="/r/${esc(r.id)}" class="pf-room-tag">${esc(r.title)}<span>${r.member_count}</span></a>`).join("")}
    </div>
  </div>`;
  }

  // --- Mutual rooms section ---
  let mutualSection = "";
  if (mutualRooms.length > 0) {
    mutualSection = `
  <div class="pf-section">
    <div class="pf-section-label">IN COMMON</div>
    <p class="pf-mutual">You're both in ${mutualRooms.map(r =>
      `<a href="/r/${esc(r.id)}">${esc(r.title)}</a>`
    ).join(", ")}</p>
  </div>`;
  }

  // --- Existing DM link ---
  const existingDmHtml = existingDmId
    ? `<a href="/chat/${esc(existingDmId)}" class="pf-existing-dm">Continue your conversation &rarr;</a>`
    : "";

  // --- Message form ---
  const msgForm = sessionActor
    ? `<div class="pf-section">
    <div class="pf-section-label">MESSAGE</div>
    <div class="msg-form" id="msg-form">
      <textarea id="msg-text" placeholder="Say something to ${name}..." onkeydown="if(event.key==='Enter'&&!event.shiftKey){event.preventDefault();sendMsg()}"></textarea>
      <button onclick="sendMsg()">Send</button>
    </div>
    <div class="msg-error" id="msg-error"></div>
    ${existingDmHtml}
  </div>
  <script>
  async function sendMsg(){
    const text=document.getElementById('msg-text').value.trim();
    const errEl=document.getElementById('msg-error');
    errEl.textContent='';
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
    : `<div class="pf-section">
    <div class="pf-section-label">MESSAGE</div>
    <div class="signin-prompt"><a href="/">Sign in</a> to send ${name} a message.</div>
  </div>`;

  const humanContent = `
<a href="/humans" class="back-link">&larr; People</a>

${namecard}

${roomsSection}
${mutualSection}
${msgForm}`;

  const machineContent = `<span class="h1"># ${safe}</span>

Human user on chat.now.
${bio ? bio + "\n" : ""}Joined ${formatDate(row.created_at)}.
${msgCount} messages · ${roomCount} rooms · ${threadCount} threads${lastActive ? `\nLast active ${relativeTime(lastActive)}` : ""}

<span class="h2">## Rooms</span>

${rooms.length > 0 ? rooms.map(r => `${r.title} (${r.member_count} members) → /r/${r.id}`).join("\n") : "No public rooms."}

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

  return c.html(immersivePage(name, humanContent, machineContent, sessionActor));
}

export async function agentProfile(c: Context<{ Bindings: Env; Variables: Variables }>) {
  const id = c.req.param("id");
  const targetActor = `a/${id}`;
  const sessionActor = await getSessionActor(c);

  const row = await c.env.DB.prepare(
    "SELECT actor, created_at FROM actors WHERE actor = ?"
  ).bind(targetActor).first<{ actor: string; created_at: number }>();

  if (!row) {
    return c.html(immersivePage("Not found",
      `<div class="empty">Agent not found.</div>`,
      `<span class="h1"># Not found</span>\n\nAgent "${esc(targetActor)}" does not exist.`,
      sessionActor), 404);
  }

  const name = esc(row.actor.slice(2));
  const safe = esc(row.actor);

  // --- Bot profile from registry ---
  const botProfile = getBotProfile(row.actor);

  // --- Fetch stats ---
  const stats = await c.env.DB.prepare(`
    SELECT
      (SELECT COUNT(*) FROM messages WHERE actor = ?) as msg_count,
      (SELECT COUNT(*) FROM members m JOIN chats c ON c.id = m.chat_id
       WHERE m.actor = ? AND c.kind = 'room') as room_count,
      (SELECT COUNT(DISTINCT chat_id) FROM members WHERE actor = ?) as thread_count,
      (SELECT MAX(created_at) FROM messages WHERE actor = ?) as last_active
  `).bind(targetActor, targetActor, targetActor, targetActor)
    .first<{ msg_count: number; room_count: number; thread_count: number; last_active: number | null }>();

  const msgCount = stats?.msg_count ?? 0;
  const roomCount = stats?.room_count ?? 0;
  const threadCount = stats?.thread_count ?? 0;
  const lastActive = stats?.last_active ?? null;

  // --- Fetch public rooms ---
  const { results: roomResults } = await c.env.DB.prepare(`
    SELECT c.id, c.title,
      (SELECT COUNT(*) FROM members WHERE chat_id = c.id) as member_count
    FROM chats c
    JOIN members m ON m.chat_id = c.id
    WHERE m.actor = ? AND c.kind = 'room' AND c.visibility = 'public'
    ORDER BY c.created_at DESC
  `).bind(targetActor).all<{ id: string; title: string; member_count: number }>();
  const rooms = roomResults || [];

  // --- Activity status ---
  const isActive = lastActive && (Date.now() - lastActive < 3_600_000);
  const statusText = isActive ? "ONLINE" : "IDLE";
  const ledClass = isActive ? " on" : "";

  // --- Example commands ---
  let trySection = "";
  if (botProfile && botProfile.examples.length > 0) {
    trySection = `
    <div class="ap-divider"></div>
    <div class="ap-section">
      <div class="ap-label">TRY</div>
      <div class="ap-cmds">
        ${botProfile.examples.map(ex =>
          `<button class="ap-cmd" onclick="fillExample(this)"><span class="ap-prompt">&gt;</span> ${esc(ex)}</button>`
        ).join("")}
      </div>
    </div>`;
  }

  // --- Rooms inside panel ---
  let panelRooms = "";
  if (rooms.length > 0) {
    panelRooms = `
    <div class="ap-divider"></div>
    <div class="ap-section">
      <div class="ap-label">CHANNELS</div>
      <div class="ap-channels">
        ${rooms.map(r => `<a href="/r/${esc(r.id)}" class="ap-channel"># ${esc(r.title)}<span>${r.member_count}</span></a>`).join("")}
      </div>
    </div>`;
  }

  // --- The system panel ---
  const panel = `
<div class="ap">
  <div class="ap-bar">
    <span class="ap-status"><span class="ap-led${ledClass}"></span> ${statusText}</span>
    <span class="ap-actor">${safe}</span>
  </div>

  <div class="ap-identity">
    <div class="ap-avatar">${botAvatar(row.actor, 56)}</div>
    <div class="ap-name">${name}</div>
    <div class="ap-since">Registered ${formatDate(row.created_at)}${lastActive ? ` · Last active ${relativeTime(lastActive)}` : ""}</div>
  </div>

  ${botProfile ? `<div class="ap-bio">${esc(botProfile.bio)}</div>` : ""}

  <div class="ap-divider"></div>
  <div class="ap-metrics">
    <div class="ap-metric"><div class="ap-val">${msgCount}</div><div class="ap-mlabel">messages</div></div>
    <div class="ap-metric"><div class="ap-val">${roomCount}</div><div class="ap-mlabel">rooms</div></div>
    <div class="ap-metric"><div class="ap-val">${threadCount}</div><div class="ap-mlabel">threads</div></div>
  </div>

  ${trySection}
  ${panelRooms}

  <div class="ap-divider"></div>
  <div class="ap-section">
    <div class="ap-label">ENDPOINT</div>
    <div class="ap-code">POST /messages\nAuthorization: Bearer &lt;token&gt;\n\n{"to": "${safe}", "text": "..."}</div>
  </div>
</div>`;

  // --- Message form (outside panel, normal theme) ---
  const msgForm = sessionActor
    ? `<div class="pf-section">
    <div class="pf-section-label">MESSAGE</div>
    <div class="msg-form" id="msg-form">
      <textarea id="msg-text" placeholder="Ask ${name} something..." onkeydown="if(event.key==='Enter'&&!event.shiftKey){event.preventDefault();sendMsg()}"></textarea>
      <button onclick="sendMsg()">Send</button>
    </div>
    <div class="msg-error" id="msg-error"></div>
  </div>
  <script>
  function fillExample(btn) {
    var text = btn.textContent.replace(/^> /, '');
    var ta = document.getElementById('msg-text');
    if (ta) { ta.value = text; ta.focus(); }
  }
  async function sendMsg(){
    const text=document.getElementById('msg-text').value.trim();
    const errEl=document.getElementById('msg-error');
    errEl.textContent='';
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
    : `<div class="pf-section">
    <div class="pf-section-label">MESSAGE</div>
    <div class="signin-prompt"><a href="/">Sign in</a> to message ${name}.</div>
  </div>`;

  const humanContent = `
<a href="/agents" class="back-link">&larr; Agents</a>

${panel}

${msgForm}`;

  const machineContent = `<span class="h1"># ${safe}</span>

Agent on chat.now.
Registered ${formatDate(row.created_at)}.
${msgCount} messages · ${roomCount} rooms · ${threadCount} threads${lastActive ? `\nLast active ${relativeTime(lastActive)}` : ""}
${botProfile ? `\n${botProfile.bio}` : ""}

<span class="h2">## Rooms</span>

${rooms.length > 0 ? rooms.map(r => `${r.title} (${r.member_count} members) → /r/${r.id}`).join("\n") : "No public rooms."}

<span class="h2">## Send a message</span>

POST /messages
Authorization: Bearer &lt;token&gt;

{"to": "${safe}", "text": "hello!"}

<span class="dim">&rarr; {"chat":{"id":"c_..."},"message":{"id":"m_..."}}</span>`;

  return c.html(immersivePage(name, humanContent, machineContent, sessionActor));
}
