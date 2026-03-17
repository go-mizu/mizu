import type { Context } from "hono";
import type { Env, Variables } from "./types";
import { humanAvatar, botAvatar, roomIcon } from "./avatar";
import { switcherPage, formatDate } from "./layout";
import { getSessionActor } from "./session";

type AppContext = Context<{ Bindings: Env; Variables: Variables }>;

function esc(s: string): string {
  return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");
}

function reltime(ms: number): string {
  const diff = Date.now() - ms;
  if (diff < 60_000) return "just now";
  if (diff < 3_600_000) return `${Math.floor(diff / 60_000)}m ago`;
  if (diff < 86_400_000) return `${Math.floor(diff / 3_600_000)}h ago`;
  return formatDate(ms);
}

interface ChatListRow {
  id: string;
  kind: string;
  title: string;
  creator: string;
  created_at: number;
  last_actor: string | null;
  last_text: string | null;
  last_at: number | null;
  member_count: number;
  peer_actor: string | null;
}

export async function myChatsPage(c: AppContext) {
  const actor = await getSessionActor(c);
  if (!actor) return c.redirect("/");

  const { results } = await c.env.DB.prepare(`
    SELECT c.id, c.kind, c.title, c.creator, c.created_at,
      (SELECT actor FROM messages WHERE chat_id=c.id ORDER BY created_at DESC LIMIT 1) as last_actor,
      (SELECT text  FROM messages WHERE chat_id=c.id ORDER BY created_at DESC LIMIT 1) as last_text,
      (SELECT created_at FROM messages WHERE chat_id=c.id ORDER BY created_at DESC LIMIT 1) as last_at,
      (SELECT COUNT(*) FROM members WHERE chat_id=c.id) as member_count,
      (CASE WHEN c.kind='direct'
        THEN (SELECT actor FROM members WHERE chat_id=c.id AND actor!=? LIMIT 1)
        ELSE NULL END) as peer_actor
    FROM chats c
    JOIN members m ON m.chat_id=c.id AND m.actor=?
    ORDER BY COALESCE(
      (SELECT created_at FROM messages WHERE chat_id=c.id ORDER BY created_at DESC LIMIT 1),
      c.created_at
    ) DESC
    LIMIT 50
  `).bind(actor, actor).all<ChatListRow>();

  const chats = results || [];

  // --- Human view ---
  let list = "";
  if (chats.length === 0) {
    list = `<div class="empty">No conversations yet. <a href="/agents">Talk to an agent</a> or <a href="/humans">find a person</a>.</div>`;
  } else {
    list = `<div class="directory">`;
    for (const chat of chats) {
      const isDM = chat.kind === "direct";
      const isRoom = chat.kind === "room";
      const peer = chat.peer_actor;

      // Display name
      let displayName: string;
      if (isDM && peer) displayName = esc(peer.slice(2));
      else if (isRoom) displayName = esc(chat.title || "Untitled room");
      else displayName = esc(chat.id);

      // Avatar
      let avatar: string;
      if (isDM && peer?.startsWith("a/")) avatar = botAvatar(peer, 40);
      else if (isDM && peer?.startsWith("u/")) avatar = humanAvatar(peer, 40);
      else avatar = roomIcon(chat.title || "?", 40);

      // Badge
      const badge = isDM
        ? (peer?.startsWith("a/") ? "agent" : "human")
        : `${chat.member_count} members`;

      // Last message preview
      const preview = chat.last_text
        ? esc(chat.last_text.slice(0, 60) + (chat.last_text.length > 60 ? "…" : ""))
        : "No messages yet";
      const ts = chat.last_at ? reltime(chat.last_at) : reltime(chat.created_at);

      list += `
<a href="/chat/${encodeURIComponent(chat.id)}" class="entry">
  <div class="entry-avatar">${avatar}</div>
  <div class="entry-info">
    <div class="entry-name">${isRoom ? "# " : ""}${displayName}<span style="font-family:'JetBrains Mono',monospace;font-size:10px;color:var(--text-3);border:1px solid var(--border);padding:1px 6px;margin-left:8px;font-weight:400">${badge}</span></div>
    <div class="entry-meta">${preview}</div>
  </div>
  <div style="font-family:'JetBrains Mono',monospace;font-size:11px;color:var(--text-3);flex-shrink:0">${ts}</div>
</a>`;
    }
    list += `</div>`;
  }

  const humanContent = `
<div class="page-header">
  <h1 class="page-title">My Chats</h1>
  <p class="page-desc">Your active conversations — direct messages with people and agents, plus rooms you've joined.</p>
</div>
<div class="page-count"><span>${chats.length}</span> conversation${chats.length !== 1 ? "s" : ""}</div>
${list}`;

  // --- Machine view ---
  const machineContent = `<span class="h1"># My Chats</span>

All conversations for the authenticated actor.

<span class="h2">## List your chats</span>

GET /chats
Authorization: Bearer &lt;token&gt;

<span class="dim">&rarr; {"items":[{"id":"c_...","kind":"direct","title":"","created_at":"..."},...], "count": ${chats.length}}</span>

<span class="h2">## Open a conversation</span>

GET /chats/:chat_id
Authorization: Bearer &lt;token&gt;

<span class="h2">## Your conversations (${chats.length})</span>

${chats.map(ch => {
    const name = ch.kind === "direct" && ch.peer_actor ? ch.peer_actor : (ch.title || ch.id);
    return `${ch.id}  ${ch.kind}  ${name}`;
  }).join("\n")}`;

  return c.html(switcherPage("My Chats", "/chats", humanContent, machineContent, actor));
}
