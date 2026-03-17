import type { Context } from "hono";
import type { Env, Variables, Message, MessageRow, SendMessageRequest, SendMessageExplicitRequest } from "./types";
import { messageId, chatId } from "./id";
import { isMember, isValidActor } from "./actor";
import { errorResponse } from "./error";

const MAX_TEXT_LEN = 4000;

function rowToMessage(row: MessageRow): Message {
  return {
    id: row.id,
    chat_id: row.chat_id,
    actor: row.actor,
    text: row.text,
    created_at: new Date(row.created_at).toISOString(),
  };
}

function chatSummary(row: { id: string; kind: string; title: string; created_at: number }) {
  return {
    id: row.id,
    kind: row.kind,
    title: row.title,
    created_at: new Date(row.created_at).toISOString(),
  };
}

// POST /messages (unified send)
export async function sendMessageUnified(c: Context<{ Bindings: Env; Variables: Variables }>) {
  const actor = c.get("actor");

  let body: SendMessageRequest;
  try {
    body = await c.req.json<SendMessageRequest>();
  } catch {
    return errorResponse(c, "invalid_request", "Invalid JSON body");
  }

  if (!body.text || typeof body.text !== "string") {
    return errorResponse(c, "invalid_request", "text is required");
  }

  if (!body.to && !body.chat_id) {
    return errorResponse(c, "invalid_request", "Either 'to' or 'chat_id' is required");
  }

  if (body.to && body.chat_id) {
    return errorResponse(c, "invalid_request", "Provide either 'to' or 'chat_id', not both");
  }

  // Dedup by client_id
  if (body.client_id) {
    const existing = await c.env.DB.prepare(
      "SELECT * FROM messages WHERE client_id = ?"
    ).bind(body.client_id).first<MessageRow>();
    if (existing) {
      const chatRow = await c.env.DB.prepare("SELECT id, kind, title, created_at FROM chats WHERE id = ?")
        .bind(existing.chat_id).first<{ id: string; kind: string; title: string; created_at: number }>();
      return c.json({
        chat: chatRow ? chatSummary(chatRow) : null,
        message: rowToMessage(existing),
      });
    }
  }

  let targetChatId: string;

  if (body.to) {
    // Send to actor: find or create direct chat
    if (!isValidActor(body.to)) {
      return errorResponse(c, "invalid_request", "Invalid 'to' actor format");
    }

    if (body.to === actor) {
      return errorResponse(c, "invalid_request", "Cannot send to yourself");
    }

    const peerExists = await c.env.DB.prepare("SELECT 1 FROM actors WHERE actor = ?")
      .bind(body.to).first();
    if (!peerExists) {
      return errorResponse(c, "not_found", "Recipient not found");
    }

    // Find existing DM
    const existing = await c.env.DB.prepare(
      `SELECT c.id FROM chats c
       JOIN members m1 ON m1.chat_id = c.id AND m1.actor = ?
       JOIN members m2 ON m2.chat_id = c.id AND m2.actor = ?
       WHERE c.kind = 'direct' LIMIT 1`
    ).bind(actor, body.to).first<{ id: string }>();

    if (existing) {
      targetChatId = existing.id;
    } else {
      // Create new DM
      targetChatId = chatId();
      const now = Date.now();
      await c.env.DB.batch([
        c.env.DB.prepare(
          "INSERT INTO chats (id, kind, title, creator, visibility, created_at) VALUES (?, ?, ?, ?, ?, ?)"
        ).bind(targetChatId, "direct", "", actor, "private", now),
        c.env.DB.prepare(
          "INSERT INTO members (chat_id, actor, role, joined_at) VALUES (?, ?, ?, ?)"
        ).bind(targetChatId, actor, "member", now),
        c.env.DB.prepare(
          "INSERT INTO members (chat_id, actor, role, joined_at) VALUES (?, ?, ?, ?)"
        ).bind(targetChatId, body.to, "member", now),
      ]);
    }
  } else {
    // Send to chat_id
    targetChatId = body.chat_id!;
    const chat = await c.env.DB.prepare("SELECT id FROM chats WHERE id = ?")
      .bind(targetChatId).first();
    if (!chat) {
      return errorResponse(c, "not_found", "Chat not found");
    }
    if (!(await isMember(c.env.DB, targetChatId, actor))) {
      return errorResponse(c, "forbidden", "Not a member of this chat");
    }
  }

  const text = body.text.slice(0, MAX_TEXT_LEN);
  const id = messageId();
  const now = Date.now();

  await c.env.DB.prepare(
    "INSERT INTO messages (id, chat_id, actor, text, client_id, created_at) VALUES (?, ?, ?, ?, ?, ?)"
  ).bind(id, targetChatId, actor, text, body.client_id || null, now).run();

  const chatRow = await c.env.DB.prepare("SELECT id, kind, title, created_at FROM chats WHERE id = ?")
    .bind(targetChatId).first<{ id: string; kind: string; title: string; created_at: number }>();

  const msg: Message = { id, chat_id: targetChatId, actor, text, created_at: new Date(now).toISOString() };
  return c.json({
    chat: chatRow ? chatSummary(chatRow) : null,
    message: msg,
  }, 201);
}

// POST /chats/:chat_id/messages (explicit send)
export async function sendMessageExplicit(c: Context<{ Bindings: Env; Variables: Variables }>) {
  const actor = c.get("actor");
  const chatIdParam = c.req.param("chat_id")!;

  const chat = await c.env.DB.prepare("SELECT id FROM chats WHERE id = ?")
    .bind(chatIdParam).first();
  if (!chat) {
    return errorResponse(c, "not_found", "Chat not found");
  }

  if (!(await isMember(c.env.DB, chatIdParam, actor))) {
    return errorResponse(c, "forbidden", "Not a member of this chat");
  }

  let body: SendMessageExplicitRequest;
  try {
    body = await c.req.json<SendMessageExplicitRequest>();
  } catch {
    return errorResponse(c, "invalid_request", "Invalid JSON body");
  }

  if (!body.text || typeof body.text !== "string") {
    return errorResponse(c, "invalid_request", "text is required");
  }

  // Dedup by client_id
  if (body.client_id) {
    const existing = await c.env.DB.prepare(
      "SELECT * FROM messages WHERE client_id = ?"
    ).bind(body.client_id).first<MessageRow>();
    if (existing) {
      return c.json(rowToMessage(existing));
    }
  }

  const text = body.text.slice(0, MAX_TEXT_LEN);
  const id = messageId();
  const now = Date.now();

  await c.env.DB.prepare(
    "INSERT INTO messages (id, chat_id, actor, text, client_id, created_at) VALUES (?, ?, ?, ?, ?, ?)"
  ).bind(id, chatIdParam, actor, text, body.client_id || null, now).run();

  const msg: Message = { id, chat_id: chatIdParam, actor, text, created_at: new Date(now).toISOString() };
  return c.json(msg, 201);
}

// GET /chats/:chat_id/messages
export async function listMessages(c: Context<{ Bindings: Env; Variables: Variables }>) {
  const chatIdParam = c.req.param("chat_id")!;
  const limit = Math.min(parseInt(c.req.query("limit") || "50", 10), 100);
  const before = c.req.query("before");
  const actor = c.get("actor");

  const chat = await c.env.DB.prepare("SELECT id, visibility FROM chats WHERE id = ?")
    .bind(chatIdParam).first<{ id: string; visibility: string }>();
  if (!chat) {
    return errorResponse(c, "not_found", "Chat not found");
  }

  if (chat.visibility === "private" && !(await isMember(c.env.DB, chatIdParam, actor))) {
    return errorResponse(c, "not_found", "Chat not found");
  }

  let stmt;
  if (before) {
    const cursor = await c.env.DB.prepare("SELECT created_at FROM messages WHERE id = ? AND chat_id = ?")
      .bind(before, chatIdParam).first<{ created_at: number }>();
    if (!cursor) {
      return errorResponse(c, "invalid_request", "Cursor message not found");
    }
    stmt = c.env.DB.prepare(
      "SELECT * FROM messages WHERE chat_id = ? AND created_at < ? ORDER BY created_at DESC LIMIT ?"
    ).bind(chatIdParam, cursor.created_at, limit + 1);
  } else {
    stmt = c.env.DB.prepare(
      "SELECT * FROM messages WHERE chat_id = ? ORDER BY created_at DESC LIMIT ?"
    ).bind(chatIdParam, limit + 1);
  }

  const { results } = await stmt.all<MessageRow>();
  const rows = results || [];
  const hasMore = rows.length > limit;
  const items = rows.slice(0, limit).map(rowToMessage);
  const nextBefore = hasMore && items.length > 0 ? items[items.length - 1].id : undefined;

  return c.json({
    items,
    next_before: nextBefore,
    has_more: hasMore,
  });
}
