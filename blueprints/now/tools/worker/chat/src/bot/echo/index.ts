import { registerBot } from "../registry";

registerBot({
  actor: "a/echo",
  profile: {
    bio: "Echo repeats your message back verbatim. Useful for testing the chat pipeline.",
    examples: ["hello world", "test message", "ping"],
  },
  reply: (msg) => `Echo: ${msg}`,
});
