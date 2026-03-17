import type { Context } from "hono";
import type { Env, Variables } from "./types";
import { roomIcon } from "./avatar";
import { directoryPage, formatDate } from "./layout";

interface RoomRow {
  id: string;
  title: string;
  created_at: number;
  member_count: number;
}

export async function roomsPage(c: Context<{ Bindings: Env; Variables: Variables }>) {
  const { results } = await c.env.DB.prepare(
    `SELECT c.id, c.title, c.created_at, COUNT(m.actor) as member_count
     FROM chats c LEFT JOIN members m ON m.chat_id = c.id
     WHERE c.kind = 'room' AND c.visibility = 'public'
     GROUP BY c.id
     ORDER BY c.created_at DESC LIMIT 100`
  ).all<RoomRow>();

  const rooms = results || [];

  let cards = "";
  if (rooms.length === 0) {
    cards = `<div class="empty">No public rooms yet. <a href="/docs#chats" style="color:#000;text-decoration:underline">Create one</a>.</div>`;
  } else {
    cards = `<div class="grid">`;
    for (const r of rooms) {
      const title = r.title || "Untitled";
      cards += `
<div class="card">
  <div class="card-avatar">${roomIcon(title)}</div>
  <div class="card-name">${escapeHtml(title)}</div>
  <div class="card-meta">${r.member_count} member${r.member_count !== 1 ? "s" : ""} · Created ${formatDate(r.created_at)}</div>
</div>`;
    }
    cards += `</div>`;
  }

  const content = `
<h1 class="page-title">Rooms</h1>
<p class="page-desc">${rooms.length} public room${rooms.length !== 1 ? "s" : ""} on chat.now</p>
${cards}`;

  return c.html(directoryPage("Rooms", "/rooms", content));
}

function escapeHtml(s: string): string {
  return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");
}
