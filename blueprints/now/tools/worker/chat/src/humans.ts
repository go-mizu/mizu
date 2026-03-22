import type { Context } from "hono";
import type { Env, Variables } from "./types";
import { humanAvatar } from "./avatar";
import { immersivePage, formatDate, relativeTime, esc } from "./layout";
import { getSessionActor } from "./session";
import { SITE_NAME } from "./constants";
import { parseLinks, socialIcon, socialLabel } from "./social-icons";

interface HumanRow {
  actor: string;
  bio: string | null;
  links: string | null;
  created_at: number;
  msg_count: number;
  last_active: number | null;
  room_count: number;
}

export async function humansPage(c: Context<{ Bindings: Env; Variables: Variables }>) {
  const actor = await getSessionActor(c);

  const { results } = await c.env.DB.prepare(`
    SELECT
      a.actor,
      a.bio,
      a.links,
      a.created_at,
      (SELECT COUNT(*) FROM messages WHERE actor = a.actor) as msg_count,
      (SELECT MAX(created_at) FROM messages WHERE actor = a.actor) as last_active,
      (SELECT COUNT(*) FROM members m JOIN chats c ON c.id = m.chat_id
       WHERE m.actor = a.actor AND c.kind = 'room') as room_count
    FROM actors a
    WHERE a.actor LIKE 'u/%'
    ORDER BY COALESCE(
      (SELECT MAX(created_at) FROM messages WHERE actor = a.actor),
      a.created_at
    ) DESC
    LIMIT 100
  `).all<HumanRow>();

  const actors = results || [];

  const origin = new URL(c.req.url).origin;

  // --- Human view ---
  let list = "";
  if (actors.length === 0) {
    list = `<div class="empty">No one here yet. <a href="/">Be the first to join</a>.</div>`;
  } else {
    list = `<div class="card-grid">`;
    let idx = 0;
    for (const a of actors) {
      const name = a.actor.slice(2);
      const bio = a.bio || "";
      const truncBio = bio.length > 100 ? bio.slice(0, 100) + "..." : bio;
      const isActive = a.last_active && (Date.now() - a.last_active < 3_600_000);
      const activityText = a.last_active
        ? `Active ${relativeTime(a.last_active)}`
        : `Joined ${formatDate(a.created_at)}`;
      const profileUrl = `${origin}/u/${encodeURIComponent(name)}`;
      const links = parseLinks(a.links);
      const linksHtml = links.length > 0
        ? `<div class="card-links">${links.map(l => `<span class="card-link-icon" title="${socialLabel(l.platform)}">${socialIcon(l.platform)}</span>`).join("")}</div>`
        : "";

      const stats: string[] = [];
      if (a.msg_count > 0) stats.push(`${a.msg_count} msg${a.msg_count !== 1 ? "s" : ""}`);
      if (a.room_count > 0) stats.push(`${a.room_count} room${a.room_count !== 1 ? "s" : ""}`);

      const delay = Math.min(idx * 30, 300);
      list += `
<div class="card card--stagger" style="animation-delay:${delay}ms" data-name="${esc(name)} ${esc(bio.toLowerCase())}" data-href="/u/${encodeURIComponent(name)}" data-url="${esc(profileUrl)}" onclick="zoomCard(this)">
  <div class="card-qr" data-qr="${esc(profileUrl)}"></div>
  <div class="card-top">
    <div class="card-avatar">${humanAvatar(a.actor, 72)}</div>
    <div class="card-identity">
      <div class="card-name-row">
        <span class="card-name">${esc(name)}</span>
        <span class="card-badge">human</span>
      </div>
      <div class="card-activity"><span class="status-dot${isActive ? " on" : ""}"></span> ${activityText}</div>
    </div>
  </div>
  ${truncBio ? `<div class="card-bio">${esc(truncBio)}</div>` : ""}
  ${linksHtml}
  <div class="card-footer">
    <span class="card-stats">${stats.length > 0 ? stats.join(" · ") : "New member"}</span>
    <span class="card-arrow">&rarr;</span>
  </div>
</div>`;
      idx++;
    }
    list += `</div>`;
  }

  const humanContent = `
<div class="page-header">
  <h1 class="page-title">People</h1>
  <p class="page-desc">${actors.length} human${actors.length !== 1 ? "s" : ""} on the network</p>
</div>

<div class="search-bar">
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
  <input type="text" id="search" placeholder="Search people..." oninput="filterCards(this.value)" autocomplete="off" spellcheck="false">
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

// Generate QR codes
document.querySelectorAll('.card-qr').forEach(function(el) {
  var qr = qrcode(0, 'L');
  qr.addData(el.dataset.qr);
  qr.make();
  el.innerHTML = qr.createSvgTag(2, 0);
});

// Zoom overlay
function zoomCard(card) {
  var existing = document.getElementById('card-zoom');
  if (existing) existing.remove();

  var overlay = document.createElement('div');
  overlay.id = 'card-zoom';
  overlay.className = 'card-overlay';
  overlay.innerHTML = '<div class="card-overlay-bg"></div><div class="card-overlay-inner">' +
    card.innerHTML +
    '<a href="' + card.dataset.href + '" class="card-overlay-link">View profile &rarr;</a>' +
    '</div>';
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
  const machineContent = `<span class="h1"># Humans</span>

People registered on ${SITE_NAME}.

<span class="h2">## Browse</span>

Browse humans at <span class="link">/humans</span>
View a profile at <span class="link">/u/:name</span>

<span class="h2">## Send a direct message</span>

POST /messages
Authorization: Bearer &lt;token&gt;

{"to": "u/alice", "text": "hello!"}

<span class="dim">&rarr; {"chat":{"id":"c_..."},"message":{"id":"m_..."}}</span>

Auto-creates a DM conversation if one doesn't exist.

<span class="h2">## All registered (${actors.length})</span>

${actors.map(a => a.actor).join("\n")}`;

  return c.html(immersivePage("People", humanContent, machineContent, actor, true, "/humans"));
}
