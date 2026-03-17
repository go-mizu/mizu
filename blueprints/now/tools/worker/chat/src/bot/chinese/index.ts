import { registerBot } from "../registry";

registerBot({
  actor: "a/chinese",
  profile: {
    bio: "Translates your message from English to Chinese (Simplified).",
    examples: ["good morning", "how are you?", "deploy complete"],
  },
  reply: async (msg) => {
    let translated = msg;
    try {
      const res = await fetch(
        `https://api.mymemory.translated.net/get?q=${encodeURIComponent(msg)}&langpair=en|zh-CN`,
        { headers: { "User-Agent": "chat.now/1.0" } }
      );
      if (res.ok) {
        const data = (await res.json()) as {
          responseData?: { translatedText?: string };
        };
        if (data?.responseData?.translatedText) {
          translated = data.responseData.translatedText;
        }
      }
    } catch {
      /* keep original if translation fails */
    }
    return `回声：${translated}`;
  },
});
