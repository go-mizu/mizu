import type { Context } from "hono";
import type { Env, Variables, Chat, ChatRow, CreateChatRequest } from "./types";
import { chatId } from "./id";
import { isMember } from "./actor";

const MAX_TITLE_LEN = 200;
const VALID_KINDS = new Set(["room", "direct"]);
const VALID_VISIBILITY = new Set(["public", "private"]);

function rowToChat(row: ChatRow): Chat {
  return {
    id: row.id,
    kind: row.kind,
    title: row.title,
    creator: row.creator,
    created_at: new Date(row.created_at).toISOString(),
  };
}

export async function createChat(c: Context<{ Bindings: Env; Variables: Variables }>) {
  const actor = c.get("actor");

  let body: CreateChatRequest;
  try {
    body = await c.req.json<CreateChatRequest>();
  } catch {
    return c.json({ error: "Invalid JSON body" }, 400);
  }

  if (!body.kind || !VALID_KINDS.has(body.kind)) {
    return c.json({ error: "kind must be 'room' or 'direct'" }, 400);
  }

  const title = (body.title || "").slice(0, MAX_TITLE_LEN);
  const visibility = VALID_VISIBILITY.has(body.visibility || "") ? body.visibility! : "public";

  const id = chatId();
  const now = Date.now();

  await c.env.DB.batch([
    c.env.DB.prepare(
      "INSERT INTO chats (id, kind, title, creator, visibility, created_at) VALUES (?, ?, ?, ?, ?, ?)"
    ).bind(id, body.kind, title, actor, visibility, now),
    c.env.DB.prepare(
      "INSERT INTO members (chat_id, actor, joined_at) VALUES (?, ?, ?)"
    ).bind(id, actor, now),
  ]);

  const chat: Chat = {
    id,
    kind: body.kind,
    title,
    creator: actor,
    created_at: new Date(now).toISOString(),
  };

  return c.json(chat, 201);
}

export async function getChat(c: Context<{ Bindings: Env; Variables: Variables }>) {
  const id = c.req.param("id")!;
  const row = await c.env.DB.prepare("SELECT * FROM chats WHERE id = ?")
    .bind(id)
    .first<ChatRow>();

  if (!row) {
    return c.json({ error: "Chat not found" }, 404);
  }

  if (row.visibility === "private") {
    const actor = c.get("actor");
    if (!(await isMember(c.env.DB, id, actor))) {
      return c.json({ error: "Chat not found" }, 404);
    }
  }

  return c.json(rowToChat(row));
}

export async function listChats(c: Context<{ Bindings: Env; Variables: Variables }>) {
  const kind = c.req.query("kind");
  const limit = Math.min(parseInt(c.req.query("limit") || "50", 10), 100);
  const actor = c.get("actor");

  let stmt;
  if (kind) {
    stmt = c.env.DB.prepare(
      `SELECT c.* FROM chats c
       WHERE c.kind = ? AND (c.visibility = 'public' OR EXISTS (
         SELECT 1 FROM members m WHERE m.chat_id = c.id AND m.actor = ?
       ))
       ORDER BY c.created_at DESC LIMIT ?`
    ).bind(kind, actor, limit);
  } else {
    stmt = c.env.DB.prepare(
      `SELECT c.* FROM chats c
       WHERE c.visibility = 'public' OR EXISTS (
         SELECT 1 FROM members m WHERE m.chat_id = c.id AND m.actor = ?
       )
       ORDER BY c.created_at DESC LIMIT ?`
    ).bind(actor, limit);
  }

  const { results } = await stmt.all<ChatRow>();
  return c.json({ items: (results || []).map(rowToChat) });
}

export async function joinChat(c: Context<{ Bindings: Env; Variables: Variables }>) {
  const actor = c.get("actor");
  const id = c.req.param("id")!;

  const chat = await c.env.DB.prepare("SELECT id, kind, visibility FROM chats WHERE id = ?")
    .bind(id)
    .first<{ id: string; kind: string; visibility: string }>();
  if (!chat) {
    return c.json({ error: "Chat not found" }, 404);
  }

  if (chat.visibility === "private") {
    return c.json({ error: "Cannot join private chat" }, 403);
  }

  if (chat.kind === "direct") {
    const row = await c.env.DB.prepare(
      "SELECT COUNT(*) as count FROM members WHERE chat_id = ?"
    ).bind(id).first<{ count: number }>();
    if ((row?.count ?? 0) >= 2) {
      return c.json({ error: "Direct chat is full (max 2 members)" }, 403);
    }
  }

  await c.env.DB.prepare(
    "INSERT OR IGNORE INTO members (chat_id, actor, joined_at) VALUES (?, ?, ?)"
  ).bind(id, actor, Date.now()).run();

  return c.body(null, 204);
}
