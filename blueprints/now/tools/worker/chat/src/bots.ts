import { messageId } from "./id";
import { isBuiltInBot, getBotProfile, listBotActors, dispatchReply } from "./bot/registry";

// Side-effect imports: each module calls registerBot() at load time
import "./bot/echo";
import "./bot/chinese";
import "./bot/scout";
import "./bot/claudestatus";

export { isBuiltInBot, getBotProfile, listBotActors, dispatchReply };

export async function handleBotReply(
  db: D1Database,
  chatId: string,
  botActor: string,
  userMessage: string
): Promise<void> {
  const replyText = await dispatchReply(botActor, userMessage, db).catch((err) => {
    console.error(`[bot] ${botActor} dispatch error:`, err);
    return null;
  });
  if (replyText === null) return;

  const id = messageId();
  const now = Date.now();
  await db
    .prepare(
      "INSERT INTO messages (id, chat_id, actor, text, client_id, created_at) VALUES (?, ?, ?, ?, ?, ?)"
    )
    .bind(id, chatId, botActor, replyText, null, now)
    .run();
}
