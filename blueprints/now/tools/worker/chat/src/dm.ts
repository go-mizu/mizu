import type { Context } from "hono";
import type { Env, Variables, Chat, ChatRow, DmRequest } from "./types";
import { chatId } from "./id";
import { isValidActor } from "./actor";

type AppContext = Context<{ Bindings: Env; Variables: Variables }>;

function rowToDmChat(row: ChatRow, caller: string): Chat {
  return {
    id: row.id,
    kind: row.kind,
    title: row.title,
    creator: row.creator,
    peer: row.creator === caller ? "" : row.creator, // placeholder, resolved below
    created_at: new Date(row.created_at).toISOString(),
  };
}

async function findExistingDm(db: D1Database, actor1: string, actor2: string): Promise<ChatRow | null> {
  return db.prepare(
    `SELECT c.* FROM chats c
     JOIN members m1 ON m1.chat_id = c.id AND m1.actor = ?
     JOIN members m2 ON m2.chat_id = c.id AND m2.actor = ?
     WHERE c.kind = 'direct'
     LIMIT 1`
  ).bind(actor1, actor2).first<ChatRow>();
}

async function getPeer(db: D1Database, chatId: string, caller: string): Promise<string> {
  const row = await db.prepare(
    "SELECT actor FROM members WHERE chat_id = ? AND actor != ? LIMIT 1"
  ).bind(chatId, caller).first<{ actor: string }>();
  return row?.actor ?? "";
}

export async function startOrResumeDm(c: AppContext) {
  const actor = c.get("actor");

  let body: DmRequest;
  try {
    body = await c.req.json<DmRequest>();
  } catch {
    return c.json({ error: "Invalid JSON body" }, 400);
  }

  if (!body.peer || typeof body.peer !== "string") {
    return c.json({ error: "peer is required" }, 400);
  }

  if (!isValidActor(body.peer)) {
    return c.json({ error: "Invalid peer format (use u/<name> or a/<name>)" }, 400);
  }

  if (body.peer === actor) {
    return c.json({ error: "Cannot DM yourself" }, 400);
  }

  // Verify peer is registered
  const peerExists = await c.env.DB.prepare("SELECT 1 FROM actors WHERE actor = ?")
    .bind(body.peer).first();
  if (!peerExists) {
    return c.json({ error: "Peer not found" }, 404);
  }

  // Check for existing DM
  const existing = await findExistingDm(c.env.DB, actor, body.peer);
  if (existing) {
    const chat: Chat = {
      id: existing.id,
      kind: existing.kind,
      title: existing.title,
      creator: existing.creator,
      peer: body.peer,
      created_at: new Date(existing.created_at).toISOString(),
    };
    return c.json(chat, 200);
  }

  // Create new DM
  const id = chatId();
  const now = Date.now();

  await c.env.DB.batch([
    c.env.DB.prepare(
      "INSERT INTO chats (id, kind, title, creator, visibility, created_at) VALUES (?, ?, ?, ?, ?, ?)"
    ).bind(id, "direct", "", actor, "private", now),
    c.env.DB.prepare(
      "INSERT INTO members (chat_id, actor, joined_at) VALUES (?, ?, ?)"
    ).bind(id, actor, now),
    c.env.DB.prepare(
      "INSERT INTO members (chat_id, actor, joined_at) VALUES (?, ?, ?)"
    ).bind(id, body.peer, now),
  ]);

  const chat: Chat = {
    id,
    kind: "direct",
    title: "",
    creator: actor,
    peer: body.peer,
    created_at: new Date(now).toISOString(),
  };

  return c.json(chat, 201);
}

export async function listDms(c: AppContext) {
  const actor = c.get("actor");
  const limit = Math.min(parseInt(c.req.query("limit") || "50", 10), 100);

  const { results } = await c.env.DB.prepare(
    `SELECT c.* FROM chats c
     JOIN members m ON m.chat_id = c.id AND m.actor = ?
     WHERE c.kind = 'direct'
     ORDER BY c.created_at DESC
     LIMIT ?`
  ).bind(actor, limit).all<ChatRow>();

  const items: Chat[] = [];
  for (const row of results || []) {
    const peer = await getPeer(c.env.DB, row.id, actor);
    items.push({
      id: row.id,
      kind: row.kind,
      title: row.title,
      creator: row.creator,
      peer,
      created_at: new Date(row.created_at).toISOString(),
    });
  }

  return c.json({ items });
}
