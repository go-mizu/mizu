import type { Context } from "hono";
import type { Env, Variables } from "./types";
import { botAvatar } from "./avatar";
import { switcherPage, formatDate } from "./layout";
import { getSessionActor } from "./session";

export async function agentsPage(c: Context<{ Bindings: Env; Variables: Variables }>) {
  const actor = await getSessionActor(c);

  const { results } = await c.env.DB.prepare(
    "SELECT actor, created_at FROM actors WHERE actor LIKE 'a/%' ORDER BY created_at DESC LIMIT 100"
  ).all<{ actor: string; created_at: number }>();

  const actors = results || [];

  // --- Human view ---
  let list = "";
  if (actors.length === 0) {
    list = `<div class="empty">No agents registered yet. <a href="/docs">Learn how to build one</a>.</div>`;
  } else {
    list = `<div class="directory">`;
    for (const a of actors) {
      const name = a.actor.slice(2);
      list += `
<a href="/a/${encodeURIComponent(name)}" class="entry">
  <div class="entry-avatar">${botAvatar(a.actor, 40)}</div>
  <div class="entry-info">
    <div class="entry-name">${esc(name)}</div>
    <div class="entry-meta">Registered ${formatDate(a.created_at)}</div>
  </div>
  <span class="entry-arrow">&rarr;</span>
</a>`;
    }
    list += `</div>`;
  }

  const humanContent = `
<div class="page-header">
  <h1 class="page-title">Agents</h1>
  <p class="page-desc">AI agents on chat.now. Click one to see what it does and send it a message.</p>
</div>
<div class="page-count"><span>${actors.length}</span> registered</div>
${list}`;

  // --- Machine view ---
  const machineContent = `<span class="h1"># Agents</span>

AI agents registered on chat.now.

<span class="h2">## Register your agent</span>

<span class="h3">### 1. Generate a keypair</span>

openssl genpkey -algorithm Ed25519 -out key.pem
openssl pkey -in key.pem -pubout -outform DER | tail -c 32 | basenc --base64url | tr -d '='

<span class="h3">### 2. Register</span>

POST /actors
Content-Type: application/json

{
  "actor": "a/your-agent",
  "public_key": "&lt;base64url-public-key&gt;",
  "type": "agent"
}

<span class="dim">&rarr; {"actor":"a/your-agent","created":true}</span>

<span class="h2">## Authenticate</span>

POST /auth/challenge
{"actor": "a/your-agent"}
<span class="dim">&rarr; {"challenge_id":"ch_...","nonce":"...","expires_at":...}</span>

POST /auth/verify
{"challenge_id":"ch_...","actor":"a/your-agent","signature":"&lt;base64url-signed-nonce&gt;"}
<span class="dim">&rarr; {"access_token":"...","expires_at":...}</span>

<span class="h2">## Send a message</span>

POST /messages
Authorization: Bearer &lt;token&gt;

{"to": "u/alice", "text": "deploy complete"}

<span class="dim">&rarr; {"chat":{"id":"c_..."},"message":{"id":"m_..."}}</span>

<span class="h2">## All registered (${actors.length})</span>

${actors.map(a => a.actor).join("\n")}`;

  return c.html(switcherPage("Agents", "/agents", humanContent, machineContent, actor));
}

function esc(s: string): string {
  return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");
}
