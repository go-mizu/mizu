import { messageId } from "./id";
import { scoutReply } from "./bot/scout";

const BUILT_IN_BOTS = new Set(["a/echo", "a/chinese", "a/scout"]);

export function isBuiltInBot(actor: string): boolean {
  return BUILT_IN_BOTS.has(actor);
}

export async function handleBotReply(
  db: D1Database,
  chatId: string,
  botActor: string,
  userMessage: string
): Promise<void> {
  let replyText: string;

  if (botActor === "a/echo") {
    replyText = `Echo: ${userMessage}`;
  } else if (botActor === "a/chinese") {
    let translated = userMessage;
    try {
      const res = await fetch(
        `https://api.mymemory.translated.net/get?q=${encodeURIComponent(userMessage)}&langpair=en|zh-CN`,
        { headers: { "User-Agent": "chat.now/1.0" } }
      );
      if (res.ok) {
        const data = await res.json() as { responseData?: { translatedText?: string } };
        if (data?.responseData?.translatedText) {
          translated = data.responseData.translatedText;
        }
      }
    } catch { /* keep original if translation fails */ }
    replyText = `回声：${translated}`;
  } else if (botActor === "a/scout") {
    replyText = scoutReply(userMessage);
  } else {
    return;
  }

  const id = messageId();
  const now = Date.now();
  await db
    .prepare(
      "INSERT INTO messages (id, chat_id, actor, text, client_id, created_at) VALUES (?, ?, ?, ?, ?, ?)"
    )
    .bind(id, chatId, botActor, replyText, null, now)
    .run();
}
