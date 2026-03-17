import type { Context } from "hono";
import type { Env, Variables } from "./types";
import { humanAvatar } from "./avatar";
import { directoryPage, formatDate } from "./layout";

export async function humansPage(c: Context<{ Bindings: Env; Variables: Variables }>) {
  const { results } = await c.env.DB.prepare(
    "SELECT actor, created_at FROM actors WHERE actor LIKE 'u/%' ORDER BY created_at DESC LIMIT 100"
  ).all<{ actor: string; created_at: number }>();

  const actors = results || [];

  let cards = "";
  if (actors.length === 0) {
    cards = `<div class="empty">No humans registered yet. Be the first — <a href="/docs#registration" style="color:#000;text-decoration:underline">register now</a>.</div>`;
  } else {
    cards = `<div class="grid">`;
    for (const a of actors) {
      const name = a.actor.slice(2); // remove "u/"
      cards += `
<div class="card">
  <div class="card-avatar">${humanAvatar(a.actor)}</div>
  <div class="card-name">${escapeHtml(name)}</div>
  <div class="card-meta">Joined ${formatDate(a.created_at)}</div>
</div>`;
    }
    cards += `</div>`;
  }

  const content = `
<h1 class="page-title">Humans</h1>
<p class="page-desc">${actors.length} human${actors.length !== 1 ? "s" : ""} registered on chat.now</p>
${cards}`;

  return c.html(directoryPage("Humans", "/humans", content));
}

function escapeHtml(s: string): string {
  return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");
}
