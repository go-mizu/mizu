import type { Context } from "hono";
import type { Env, Variables, Member, AddMemberRequest } from "./types";
import { isMember, isValidActor } from "./actor";
import { errorResponse } from "./error";

// GET /chats/:chat_id/members
export async function listMembers(c: Context<{ Bindings: Env; Variables: Variables }>) {
  const chatId = c.req.param("chat_id")!;
  const actor = c.get("actor");

  const chat = await c.env.DB.prepare("SELECT id, visibility FROM chats WHERE id = ?")
    .bind(chatId).first<{ id: string; visibility: string }>();
  if (!chat) return errorResponse(c, "not_found", "Chat not found");

  if (chat.visibility === "private" && !(await isMember(c.env.DB, chatId, actor))) {
    return errorResponse(c, "not_found", "Chat not found");
  }

  const { results } = await c.env.DB.prepare(
    "SELECT actor, role FROM members WHERE chat_id = ?"
  ).bind(chatId).all<{ actor: string; role: string }>();

  const items: Member[] = (results || []).map(r => ({ actor: r.actor, role: r.role }));
  return c.json({ items });
}

// POST /chats/:chat_id/members
export async function addMember(c: Context<{ Bindings: Env; Variables: Variables }>) {
  const chatId = c.req.param("chat_id")!;
  const actor = c.get("actor");

  const chat = await c.env.DB.prepare("SELECT id, kind FROM chats WHERE id = ?")
    .bind(chatId).first<{ id: string; kind: string }>();
  if (!chat) return errorResponse(c, "not_found", "Chat not found");

  if (chat.kind === "direct") {
    return errorResponse(c, "forbidden", "Cannot add members to direct chat");
  }

  if (!(await isMember(c.env.DB, chatId, actor))) {
    return errorResponse(c, "forbidden", "Not a member of this chat");
  }

  let body: AddMemberRequest;
  try {
    body = await c.req.json<AddMemberRequest>();
  } catch {
    return errorResponse(c, "invalid_request", "Invalid JSON body");
  }

  if (!body.actor || !isValidActor(body.actor)) {
    return errorResponse(c, "invalid_request", "Valid actor is required");
  }

  const actorExists = await c.env.DB.prepare("SELECT 1 FROM actors WHERE actor = ?")
    .bind(body.actor).first();
  if (!actorExists) {
    return errorResponse(c, "not_found", "Actor not found");
  }

  await c.env.DB.prepare(
    "INSERT OR IGNORE INTO members (chat_id, actor, role, joined_at) VALUES (?, ?, ?, ?)"
  ).bind(chatId, body.actor, "member", Date.now()).run();

  return c.json({ actor: body.actor, role: "member" }, 201);
}

// DELETE /chats/:chat_id/members/:actor
export async function removeMember(c: Context<{ Bindings: Env; Variables: Variables }>) {
  const chatId = c.req.param("chat_id")!;
  const targetActor = c.req.param("actor")!;
  const actor = c.get("actor");

  const chat = await c.env.DB.prepare("SELECT id, kind FROM chats WHERE id = ?")
    .bind(chatId).first<{ id: string; kind: string }>();
  if (!chat) return errorResponse(c, "not_found", "Chat not found");

  if (chat.kind === "direct") {
    return errorResponse(c, "forbidden", "Cannot remove members from direct chat");
  }

  if (!(await isMember(c.env.DB, chatId, actor))) {
    return errorResponse(c, "forbidden", "Not a member of this chat");
  }

  await c.env.DB.prepare(
    "DELETE FROM members WHERE chat_id = ? AND actor = ?"
  ).bind(chatId, targetActor).run();

  return c.body(null, 204);
}

// POST /chats/:chat_id/join
export async function joinChat(c: Context<{ Bindings: Env; Variables: Variables }>) {
  const actor = c.get("actor");
  const chatId = c.req.param("chat_id")!;

  const chat = await c.env.DB.prepare("SELECT id, kind, visibility FROM chats WHERE id = ?")
    .bind(chatId).first<{ id: string; kind: string; visibility: string }>();
  if (!chat) return errorResponse(c, "not_found", "Chat not found");

  if (chat.kind === "direct") {
    return errorResponse(c, "forbidden", "Cannot join direct chat");
  }

  if (chat.visibility === "private" && !(await isMember(c.env.DB, chatId, actor))) {
    return errorResponse(c, "not_found", "Chat not found");
  }

  await c.env.DB.prepare(
    "INSERT OR IGNORE INTO members (chat_id, actor, role, joined_at) VALUES (?, ?, ?, ?)"
  ).bind(chatId, actor, "member", Date.now()).run();

  return c.body(null, 204);
}

// POST /chats/:chat_id/leave
export async function leaveChat(c: Context<{ Bindings: Env; Variables: Variables }>) {
  const actor = c.get("actor");
  const chatId = c.req.param("chat_id")!;

  const chat = await c.env.DB.prepare("SELECT id, kind FROM chats WHERE id = ?")
    .bind(chatId).first<{ id: string; kind: string }>();
  if (!chat) return errorResponse(c, "not_found", "Chat not found");

  if (chat.kind === "direct") {
    return errorResponse(c, "forbidden", "Cannot leave direct chat");
  }

  await c.env.DB.prepare(
    "DELETE FROM members WHERE chat_id = ? AND actor = ?"
  ).bind(chatId, actor).run();

  return c.body(null, 204);
}
