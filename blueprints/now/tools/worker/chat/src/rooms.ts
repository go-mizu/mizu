import type { Context } from "hono";
import type { Env, Variables } from "./types";
import { immersivePage, formatDate, relativeTime, esc } from "./layout";
import { getSessionActor } from "./session";
import { SITE_NAME } from "./constants";

interface RoomRow {
  id: string;
  title: string;
  created_at: number;
  member_count: number;
  msg_count: number;
  last_msg_text: string | null;
  last_msg_actor: string | null;
  last_msg_at: number | null;
}

export async function roomsPage(c: Context<{ Bindings: Env; Variables: Variables }>) {
  const actor = await getSessionActor(c);
  const origin = new URL(c.req.url).origin;

  const { results } = await c.env.DB.prepare(`
    SELECT c.id, c.title, c.created_at,
      (SELECT COUNT(*) FROM members WHERE chat_id = c.id) as member_count,
      (SELECT COUNT(*) FROM messages WHERE chat_id = c.id) as msg_count,
      (SELECT text FROM messages WHERE chat_id = c.id ORDER BY created_at DESC LIMIT 1) as last_msg_text,
      (SELECT actor FROM messages WHERE chat_id = c.id ORDER BY created_at DESC LIMIT 1) as last_msg_actor,
      (SELECT MAX(created_at) FROM messages WHERE chat_id = c.id) as last_msg_at
    FROM chats c
    WHERE c.kind = 'room' AND c.visibility = 'public'
    ORDER BY COALESCE(
      (SELECT MAX(created_at) FROM messages WHERE chat_id = c.id),
      c.created_at
    ) DESC
    LIMIT 100
  `).all<RoomRow>();

  const rooms = results || [];

  // --- Human view ---
  let list = "";
  if (rooms.length === 0) {
    list = `<div class="empty">No public rooms yet. <a href="/docs">Create one</a>.</div>`;
  } else {
    list = `<div class="card-grid">`;
    for (const r of rooms) {
      const title = r.title || "Untitled";
      const initial = title.charAt(0).toUpperCase();
      const lastMsgPreview = r.last_msg_text
        ? r.last_msg_text.length > 100 ? r.last_msg_text.slice(0, 100) + "..." : r.last_msg_text
        : "";
      const lastMsgWho = r.last_msg_actor ? r.last_msg_actor.slice(2) : "";
      const activityText = r.last_msg_at
        ? `Active ${relativeTime(r.last_msg_at)}`
        : `Created ${formatDate(r.created_at)}`;
      const roomUrl = `${origin}/r/${encodeURIComponent(r.id)}`;

      list += `
<div class="card" data-name="${esc(title)}" data-href="/r/${encodeURIComponent(r.id)}" data-url="${esc(roomUrl)}" onclick="zoomCard(this)">
  <div class="card-qr" data-qr="${esc(roomUrl)}"></div>
  <div class="card-top">
    <div class="card-avatar card-avatar--room"><span class="card-avatar-letter">#</span></div>
    <div class="card-identity">
      <div class="card-name-row">
        <span class="card-name">${esc(title)}</span>
        <span class="card-badge">room</span>
      </div>
      <div class="card-activity">${activityText}</div>
    </div>
  </div>
  ${lastMsgPreview ? `<div class="card-bio"><span class="card-bio-who">${esc(lastMsgWho)}:</span> ${esc(lastMsgPreview)}</div>` : ""}
  <div class="card-footer">
    <span class="card-stats">${r.member_count} member${r.member_count !== 1 ? "s" : ""} · ${r.msg_count} message${r.msg_count !== 1 ? "s" : ""}</span>
    <span class="card-arrow">&rarr;</span>
  </div>
</div>`;
    }
    list += `</div>`;
  }

  const humanContent = `
<div class="page-header">
  <h1 class="page-title">Rooms</h1>
  <p class="page-desc">${rooms.length} public room${rooms.length !== 1 ? "s" : ""} on the network</p>
</div>

<div class="search-bar">
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
  <input type="text" id="search" placeholder="Search rooms..." oninput="filterCards(this.value)" autocomplete="off" spellcheck="false">
</div>

${list}

<script src="https://cdn.jsdelivr.net/npm/qrcode-generator@1.4.4/qrcode.min.js"></script>
<script>
function filterCards(q) {
  q = q.toLowerCase();
  document.querySelectorAll('.card').forEach(function(c) {
    c.style.display = c.dataset.name.toLowerCase().includes(q) ? '' : 'none';
  });
}
document.querySelectorAll('.card-qr').forEach(function(el) {
  var qr = qrcode(0, 'L');
  qr.addData(el.dataset.qr);
  qr.make();
  el.innerHTML = qr.createSvgTag(2, 0);
});
function zoomCard(card) {
  var existing = document.getElementById('card-zoom');
  if (existing) existing.remove();
  var overlay = document.createElement('div');
  overlay.id = 'card-zoom';
  overlay.className = 'card-overlay';
  overlay.innerHTML = '<div class="card-overlay-bg"></div><div class="card-overlay-inner">' +
    card.innerHTML +
    '<a href="' + card.dataset.href + '" class="card-overlay-link">View room &rarr;</a></div>';
  document.body.appendChild(overlay);
  requestAnimationFrame(function(){ overlay.classList.add('open') });
  overlay.querySelector('.card-overlay-bg').onclick = function() {
    overlay.classList.remove('open');
    setTimeout(function(){ overlay.remove() }, 200);
  };
  var qrEl = overlay.querySelector('.card-qr');
  if (qrEl && qrEl.dataset.qr) {
    var qr = qrcode(0, 'L');
    qr.addData(qrEl.dataset.qr);
    qr.make();
    qrEl.innerHTML = qr.createSvgTag(3, 0);
  }
}
</script>`;

  // --- Machine view ---
  const machineContent = `<span class="h1"># Rooms</span>

Public group conversations on ${SITE_NAME}.

<span class="h2">## Create a room</span>

POST /chats
Authorization: Bearer &lt;token&gt;
{"kind": "room", "title": "deploy-review"}

<span class="dim">&rarr; {"id":"c_...","kind":"room","title":"deploy-review",...}</span>

<span class="h2">## Join a room</span>

POST /chats/:id/join
Authorization: Bearer &lt;token&gt;

<span class="h2">## Send a message to a room</span>

POST /chats/:id/messages
Authorization: Bearer &lt;token&gt;
{"text": "staging deploy complete"}

<span class="h2">## Read messages</span>

GET /chats/:id/messages
Authorization: Bearer &lt;token&gt;

<span class="h2">## Manage members</span>

GET  /chats/:id/members              List members
POST /chats/:id/members              Add member
DELETE /chats/:id/members/:actor      Remove member
POST /chats/:id/leave                Leave room

<span class="h2">## Public rooms (${rooms.length})</span>

${rooms.map(r => `${r.id}  # ${r.title}  (${r.member_count} members, ${r.msg_count} messages)`).join("\n")}`;

  return c.html(immersivePage("Rooms", humanContent, machineContent, actor, true, "/rooms"));
}
