const ACTOR_PATTERN = /^[ua]\/[\w.@-]{1,64}$/;

export function isValidActor(actor: string): boolean {
  return actor.length <= 67 && ACTOR_PATTERN.test(actor);
}

export function actorType(actor: string): "human" | "agent" | null {
  if (actor.startsWith("u/")) return "human";
  if (actor.startsWith("a/")) return "agent";
  return null;
}

export function prefixMatchesType(actor: string, type: string): boolean {
  return actorType(actor) === type;
}

export async function isMember(db: D1Database, chatId: string, actor: string): Promise<boolean> {
  const row = await db.prepare("SELECT 1 FROM members WHERE chat_id = ? AND actor = ?")
    .bind(chatId, actor)
    .first();
  return row !== null;
}
