const ACTOR_PATTERN = /^[ua]\/[\w.@-]{1,64}$/;

/**
 * Validate actor format. Returns true if valid.
 */
export function isValidActor(actor: string): boolean {
  return actor.length <= 67 && ACTOR_PATTERN.test(actor);
}

/**
 * Check if actor is a member of the given chat.
 */
export async function isMember(db: D1Database, chatId: string, actor: string): Promise<boolean> {
  const row = await db.prepare("SELECT 1 FROM members WHERE chat_id = ? AND actor = ?")
    .bind(chatId, actor)
    .first();
  return row !== null;
}
