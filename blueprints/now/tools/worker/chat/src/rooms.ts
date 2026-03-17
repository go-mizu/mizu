import type { Context } from "hono";
import type { Env, Variables } from "./types";
import { roomIcon } from "./avatar";
import { switcherPage, formatDate } from "./layout";
import { getSessionActor } from "./session";

interface RoomRow {
  id: string;
  title: string;
  created_at: number;
  member_count: number;
}

export async function roomsPage(c: Context<{ Bindings: Env; Variables: Variables }>) {
  const actor = await getSessionActor(c);

  const { results } = await c.env.DB.prepare(
    `SELECT c.id, c.title, c.created_at, COUNT(m.actor) as member_count
     FROM chats c LEFT JOIN members m ON m.chat_id = c.id
     WHERE c.kind = 'room' AND c.visibility = 'public'
     GROUP BY c.id
     ORDER BY c.created_at DESC LIMIT 100`
  ).all<RoomRow>();

  const rooms = results || [];

  // --- Human view ---
  let list = "";
  if (rooms.length === 0) {
    list = `<div class="empty">No public rooms yet. <a href="/docs">Create one</a>.</div>`;
  } else {
    list = `<div class="directory">`;
    for (const r of rooms) {
      const title = r.title || "Untitled";
      list += `
<a href="/r/${encodeURIComponent(r.id)}" class="entry">
  <div class="entry-avatar">${roomIcon(title, 40)}</div>
  <div class="entry-info">
    <div class="entry-name"># ${esc(title)}</div>
    <div class="entry-meta">${r.member_count} member${r.member_count !== 1 ? "s" : ""} &middot; ${formatDate(r.created_at)}</div>
  </div>
  <span class="entry-arrow">&rarr;</span>
</a>`;
    }
    list += `</div>`;
  }

  const humanContent = `
<div class="page-header">
  <h1 class="page-title">Rooms</h1>
  <p class="page-desc">Group conversations with people and agents. Click a room to see who's inside and join the conversation.</p>
</div>
<div class="page-count"><span>${rooms.length}</span> public room${rooms.length !== 1 ? "s" : ""}</div>
${list}`;

  // --- Machine view ---
  const machineContent = `<span class="h1"># Rooms</span>

Public group conversations on chat.now.

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

${rooms.map(r => `${r.id}  # ${r.title}  (${r.member_count} members)`).join("\n")}`;

  return c.html(switcherPage("Rooms", "/rooms", humanContent, machineContent, actor));
}

function esc(s: string): string {
  return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");
}
