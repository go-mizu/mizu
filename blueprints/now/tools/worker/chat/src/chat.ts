import type { Context } from "hono";
import type { Env, Variables, Chat, ChatRow, CreateChatRequest } from "./types";
import { chatId } from "./id";
import { isMember, isValidActor } from "./actor";
import { errorResponse } from "./error";

const MAX_TITLE_LEN = 200;

function rowToChat(row: ChatRow): Chat {
  return {
    id: row.id,
    kind: row.kind as "direct" | "room",
    title: row.title,
    created_at: new Date(row.created_at).toISOString(),
  };
}

// POST /chats
export async function createChat(c: Context<{ Bindings: Env; Variables: Variables }>) {
  const actor = c.get("actor");

  let body: CreateChatRequest;
  try {
    body = await c.req.json<CreateChatRequest>();
  } catch {
    return errorResponse(c, "invalid_request", "Invalid JSON body");
  }

  if (!body.kind || (body.kind !== "direct" && body.kind !== "room")) {
    return errorResponse(c, "invalid_request", "kind must be 'direct' or 'room'");
  }

  if (body.kind === "direct") {
    return createDirectChat(c, actor, body);
  }

  // Room creation
  const title = (body.title || "").slice(0, MAX_TITLE_LEN);
  const id = chatId();
  const now = Date.now();

  await c.env.DB.batch([
    c.env.DB.prepare(
      "INSERT INTO chats (id, kind, title, creator, visibility, created_at) VALUES (?, ?, ?, ?, ?, ?)"
    ).bind(id, "room", title, actor, "private", now),
    c.env.DB.prepare(
      "INSERT INTO members (chat_id, actor, role, joined_at) VALUES (?, ?, ?, ?)"
    ).bind(id, actor, "owner", now),
  ]);

  const chat: Chat = { id, kind: "room", title, created_at: new Date(now).toISOString() };
  return c.json(chat, 201);
}

async function createDirectChat(c: Context<{ Bindings: Env; Variables: Variables }>, actor: string, body: CreateChatRequest) {
  if (!body.peer || typeof body.peer !== "string") {
    return errorResponse(c, "invalid_request", "peer is required for direct chats");
  }

  if (!isValidActor(body.peer)) {
    return errorResponse(c, "invalid_request", "Invalid peer format");
  }

  if (body.peer === actor) {
    return errorResponse(c, "invalid_request", "Cannot create direct chat with yourself");
  }

  // Verify peer exists
  const peerExists = await c.env.DB.prepare("SELECT 1 FROM actors WHERE actor = ?")
    .bind(body.peer).first();
  if (!peerExists) {
    return errorResponse(c, "not_found", "Peer not found");
  }

  // Check for existing direct chat
  const existing = await c.env.DB.prepare(
    `SELECT c.* FROM chats c
     JOIN members m1 ON m1.chat_id = c.id AND m1.actor = ?
     JOIN members m2 ON m2.chat_id = c.id AND m2.actor = ?
     WHERE c.kind = 'direct' LIMIT 1`
  ).bind(actor, body.peer).first<ChatRow>();

  if (existing) {
    return c.json(rowToChat(existing), 200);
  }

  const id = chatId();
  const now = Date.now();

  await c.env.DB.batch([
    c.env.DB.prepare(
      "INSERT INTO chats (id, kind, title, creator, visibility, created_at) VALUES (?, ?, ?, ?, ?, ?)"
    ).bind(id, "direct", "", actor, "private", now),
    c.env.DB.prepare(
      "INSERT INTO members (chat_id, actor, role, joined_at) VALUES (?, ?, ?, ?)"
    ).bind(id, actor, "member", now),
    c.env.DB.prepare(
      "INSERT INTO members (chat_id, actor, role, joined_at) VALUES (?, ?, ?, ?)"
    ).bind(id, body.peer, "member", now),
  ]);

  const chat: Chat = { id, kind: "direct", title: body.peer, created_at: new Date(now).toISOString() };
  return c.json(chat, 201);
}

// GET /chats
export async function listChats(c: Context<{ Bindings: Env; Variables: Variables }>) {
  const actor = c.get("actor");
  const limit = Math.min(parseInt(c.req.query("limit") || "50", 10), 100);
  const cursor = c.req.query("cursor");

  let sql = `SELECT c.* FROM chats c
    WHERE (c.visibility = 'public' OR EXISTS (SELECT 1 FROM members m WHERE m.chat_id = c.id AND m.actor = ?))`;
  const binds: any[] = [actor];

  if (cursor) {
    const cursorRow = await c.env.DB.prepare("SELECT created_at FROM chats WHERE id = ?")
      .bind(cursor).first<{ created_at: number }>();
    if (cursorRow) {
      sql += " AND c.created_at < ?";
      binds.push(cursorRow.created_at);
    }
  }

  sql += " ORDER BY c.created_at DESC LIMIT ?";
  binds.push(limit + 1);

  const { results } = await c.env.DB.prepare(sql).bind(...binds).all<ChatRow>();

  const rows = results || [];
  const hasMore = rows.length > limit;
  const items = rows.slice(0, limit).map(rowToChat);
  const nextCursor = hasMore && items.length > 0 ? items[items.length - 1].id : undefined;

  return c.json({
    items,
    next_cursor: nextCursor,
    has_more: hasMore,
  });
}

// GET /chats/:chat_id
export async function getChat(c: Context<{ Bindings: Env; Variables: Variables }>) {
  const id = c.req.param("chat_id")!;
  const actor = c.get("actor");

  const row = await c.env.DB.prepare("SELECT * FROM chats WHERE id = ?")
    .bind(id).first<ChatRow>();

  if (!row) {
    return errorResponse(c, "not_found", "Chat not found");
  }

  if (row.visibility === "private" && !(await isMember(c.env.DB, id, actor))) {
    return errorResponse(c, "not_found", "Chat not found");
  }

  return c.json(rowToChat(row));
}
