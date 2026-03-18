import type { Context } from "hono";
import type { Env, Variables } from "./types";
import { humanAvatar } from "./avatar";
import { immersivePage, formatDate, relativeTime, esc } from "./layout";
import { getSessionActor } from "./session";

interface HumanRow {
  actor: string;
  bio: string | null;
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

  // --- Human view ---
  let list = "";
  if (actors.length === 0) {
    list = `<div class="empty">No one here yet. <a href="/">Be the first to join</a>.</div>`;
  } else {
    list = `<div class="people-grid">`;
    for (const a of actors) {
      const name = a.actor.slice(2);
      const bio = a.bio || "";
      const truncBio = bio.length > 80 ? bio.slice(0, 80) + "..." : bio;
      const isActive = a.last_active && (Date.now() - a.last_active < 3_600_000);
      const statusHtml = a.last_active
        ? `<span class="status-dot${isActive ? " on" : ""}"></span> Active ${relativeTime(a.last_active)}`
        : `Joined ${formatDate(a.created_at)}`;

      const stats: string[] = [];
      if (a.room_count > 0) stats.push(`${a.room_count} room${a.room_count !== 1 ? "s" : ""}`);
      if (a.msg_count > 0) stats.push(`${a.msg_count} msg${a.msg_count !== 1 ? "s" : ""}`);

      list += `
<a href="/u/${encodeURIComponent(name)}" class="person-card" data-name="${esc(name)} ${esc(bio.toLowerCase())}">
  <div class="person-avatar">${humanAvatar(a.actor, 48)}</div>
  <div class="person-info">
    <div class="person-name">${esc(name)}</div>
    ${truncBio ? `<div class="person-bio">${esc(truncBio)}</div>` : ""}
    <div class="person-status">${statusHtml}</div>
    ${stats.length > 0 ? `<div class="person-meta">${stats.join(" · ")}</div>` : ""}
  </div>
  <span class="person-arrow">&rarr;</span>
</a>`;
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

<script>
function filterCards(q) {
  q = q.toLowerCase();
  document.querySelectorAll('.person-card').forEach(function(c) {
    c.style.display = c.dataset.name.toLowerCase().includes(q) ? '' : 'none';
  });
}
</script>`;

  // --- Machine view ---
  const machineContent = `<span class="h1"># Humans</span>

People registered on chat.now.

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

  return c.html(immersivePage("People", humanContent, machineContent, actor, true));
}
