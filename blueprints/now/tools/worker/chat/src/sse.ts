import type { Context } from "hono";
import type { Env, Variables, MessageRow } from "./types";
import { isMember } from "./actor";

type AppContext = Context<{ Bindings: Env; Variables: Variables }>;

const POLL_INTERVAL_MS = 2000;
const PING_EVERY = 15; // send a ping comment every N polls (~30s)

export async function sseMessages(c: AppContext) {
  const actor = c.get("actor");
  const chatId = c.req.param("chat_id") || "";

  // Check chat exists and actor has access
  const chat = await c.env.DB.prepare(
    "SELECT id, visibility FROM chats WHERE id = ?"
  ).bind(chatId).first<{ id: string; visibility: string }>();

  if (!chat) {
    return c.json({ error: { code: "not_found", message: "Chat not found" } }, 404);
  }

  if (chat.visibility === "private" && !(await isMember(c.env.DB, chatId, actor))) {
    return c.json({ error: { code: "forbidden", message: "Not a member" } }, 403);
  }

  // Starting point: newest existing message
  const latest = await c.env.DB.prepare(
    "SELECT created_at FROM messages WHERE chat_id = ? ORDER BY created_at DESC LIMIT 1"
  ).bind(chatId).first<{ created_at: number }>();

  let lastSeen = latest?.created_at ?? (Date.now() - 1000);

  // Create a streaming response
  const { readable, writable } = new TransformStream<Uint8Array, Uint8Array>();
  const writer = writable.getWriter();
  const enc = new TextEncoder();

  let closed = false;

  const write = async (chunk: string): Promise<boolean> => {
    try {
      await writer.write(enc.encode(chunk));
      return true;
    } catch {
      closed = true;
      return false;
    }
  };

  // Send initial connected comment immediately
  await write(`: connected\n\n`);

  // Background poll loop
  const poll = async () => {
    let pingCount = 0;
    while (!closed) {
      try {
        const { results } = await c.env.DB.prepare(
          `SELECT id, chat_id, actor, text, created_at
           FROM messages WHERE chat_id = ? AND created_at > ?
           ORDER BY created_at ASC LIMIT 20`
        ).bind(chatId, lastSeen).all<MessageRow>();

        for (const row of results || []) {
          const ok = await write(
            `data: ${JSON.stringify({
              id: row.id,
              chat_id: row.chat_id,
              actor: row.actor,
              text: row.text,
              created_at: new Date(row.created_at).toISOString(),
            })}\n\n`
          );
          if (!ok) break;
          lastSeen = row.created_at;
        }

        // Periodic ping to keep connection alive through proxies
        pingCount++;
        if (pingCount % PING_EVERY === 0) {
          await write(": ping\n\n");
        }
      } catch {
        break;
      }

      if (closed) break;
      await new Promise<void>((r) => setTimeout(r, POLL_INTERVAL_MS));
    }

    try { await writer.close(); } catch { /* already closed */ }
  };

  poll(); // fire and forget — runs while client is connected

  return new Response(readable as unknown as BodyInit, {
    headers: {
      "Content-Type": "text/event-stream",
      "Cache-Control": "no-cache, no-store",
      "X-Accel-Buffering": "no",
      "Connection": "keep-alive",
    },
  });
}
