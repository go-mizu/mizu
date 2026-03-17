import type { Context } from "hono";
import type { Env, Variables, Message, MessageRow, SendMessageRequest } from "./types";
import { messageId } from "./id";
import { isMember } from "./actor";

const MAX_TEXT_LEN = 4000;

function rowToMessage(row: MessageRow): Message {
  return {
    id: row.id,
    chat: row.chat_id,
    actor: row.actor,
    text: row.text,
    created_at: new Date(row.created_at).toISOString(),
  };
}

export async function sendMessage(c: Context<{ Bindings: Env; Variables: Variables }>) {
  const actor = c.get("actor");
  const chatId = c.req.param("id")!;

  const chat = await c.env.DB.prepare("SELECT id FROM chats WHERE id = ?")
    .bind(chatId)
    .first();
  if (!chat) {
    return c.json({ error: "Chat not found" }, 404);
  }

  if (!(await isMember(c.env.DB, chatId, actor))) {
    return c.json({ error: "Not a member of this chat" }, 403);
  }

  let body: SendMessageRequest;
  try {
    body = await c.req.json<SendMessageRequest>();
  } catch {
    return c.json({ error: "Invalid JSON body" }, 400);
  }

  if (!body.text || typeof body.text !== "string") {
    return c.json({ error: "text is required" }, 400);
  }

  const text = body.text.slice(0, MAX_TEXT_LEN);
  const id = messageId();
  const now = Date.now();

  await c.env.DB.prepare(
    "INSERT INTO messages (id, chat_id, actor, text, created_at) VALUES (?, ?, ?, ?, ?)"
  ).bind(id, chatId, actor, text, now).run();

  const msg: Message = {
    id,
    chat: chatId,
    actor,
    text,
    created_at: new Date(now).toISOString(),
  };

  return c.json(msg, 201);
}

export async function listMessages(c: Context<{ Bindings: Env; Variables: Variables }>) {
  const chatId = c.req.param("id")!;
  const limit = Math.min(parseInt(c.req.query("limit") || "50", 10), 100);
  const before = c.req.query("before");

  const chat = await c.env.DB.prepare("SELECT id, visibility FROM chats WHERE id = ?")
    .bind(chatId)
    .first<{ id: string; visibility: string }>();
  if (!chat) {
    return c.json({ error: "Chat not found" }, 404);
  }

  if (chat.visibility === "private") {
    const actor = c.get("actor");
    if (!(await isMember(c.env.DB, chatId, actor))) {
      return c.json({ error: "Chat not found" }, 404);
    }
  }

  let stmt;
  if (before) {
    const cursor = await c.env.DB.prepare("SELECT created_at FROM messages WHERE id = ? AND chat_id = ?")
      .bind(before, chatId)
      .first<{ created_at: number }>();

    if (!cursor) {
      return c.json({ error: "Cursor message not found" }, 400);
    }

    stmt = c.env.DB.prepare(
      "SELECT * FROM messages WHERE chat_id = ? AND created_at < ? ORDER BY created_at DESC LIMIT ?"
    ).bind(chatId, cursor.created_at, limit);
  } else {
    stmt = c.env.DB.prepare(
      "SELECT * FROM messages WHERE chat_id = ? ORDER BY created_at DESC LIMIT ?"
    ).bind(chatId, limit);
  }

  const { results } = await stmt.all<MessageRow>();
  return c.json({ items: (results || []).map(rowToMessage) });
}
