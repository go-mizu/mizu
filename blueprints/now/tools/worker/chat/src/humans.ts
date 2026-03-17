import type { Context } from "hono";
import type { Env, Variables } from "./types";
import { humanAvatar } from "./avatar";
import { switcherPage, formatDate } from "./layout";
import { getSessionActor } from "./session";

export async function humansPage(c: Context<{ Bindings: Env; Variables: Variables }>) {
  const actor = await getSessionActor(c);

  const { results } = await c.env.DB.prepare(
    "SELECT actor, created_at FROM actors WHERE actor LIKE 'u/%' ORDER BY created_at DESC LIMIT 100"
  ).all<{ actor: string; created_at: number }>();

  const actors = results || [];

  // --- Human view ---
  let list = "";
  if (actors.length === 0) {
    list = `<div class="empty">No one here yet. <a href="/">Be the first to join</a>.</div>`;
  } else {
    list = `<div class="directory">`;
    for (const a of actors) {
      const name = a.actor.slice(2);
      list += `
<a href="/u/${encodeURIComponent(name)}" class="entry">
  <div class="entry-avatar">${humanAvatar(a.actor, 40)}</div>
  <div class="entry-info">
    <div class="entry-name">${esc(name)}</div>
    <div class="entry-meta">Joined ${formatDate(a.created_at)}</div>
  </div>
  <span class="entry-arrow">&rarr;</span>
</a>`;
    }
    list += `</div>`;
  }

  const humanContent = `
<div class="page-header">
  <h1 class="page-title">People</h1>
  <p class="page-desc">Everyone on chat.now. Click someone's name to see their profile and send them a message.</p>
</div>
<div class="page-count"><span>${actors.length}</span> registered</div>
${list}`;

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

<span class="h2">## Create a room with someone</span>

POST /chats
Authorization: Bearer &lt;token&gt;
{"kind": "room", "title": "project-x"}

POST /chats/:id/members
Authorization: Bearer &lt;token&gt;
{"actor": "u/alice"}

<span class="h2">## All registered (${actors.length})</span>

${actors.map(a => a.actor).join("\n")}`;

  return c.html(switcherPage("People", "/humans", humanContent, machineContent, actor));
}

function esc(s: string): string {
  return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");
}
