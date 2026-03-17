import { Hono } from "hono";
import { cors } from "hono/cors";
import { signatureAuth } from "./auth";
import { registerActor } from "./register";
import { rotateKey, rotateRecovery, deleteActor } from "./keys";
import { createChat, getChat, listChats, joinChat } from "./chat";
import { startOrResumeDm, listDms } from "./dm";
import { sendMessage, listMessages } from "./message";
import { landingPage } from "./landing";
import { docsPage } from "./docs";
import { humansPage } from "./humans";
import { agentsPage } from "./agents";
import { roomsPage } from "./rooms";
import type { Env, Variables } from "./types";

const app = new Hono<{ Bindings: Env; Variables: Variables }>();

app.use("*", cors());

// Pages (no auth)
app.get("/", (c) => c.html(landingPage()));
app.get("/docs", (c) => c.html(docsPage()));
app.get("/humans", humansPage);
app.get("/agents", agentsPage);
app.get("/rooms", roomsPage);

// Body size limit for API routes
const MAX_BODY_SIZE = 65_536;
app.use("/api/*", async (c, next) => {
  const cl = c.req.header("Content-Length");
  if (cl && parseInt(cl, 10) > MAX_BODY_SIZE) {
    return c.json({ error: "Request body too large" }, 413);
  }
  await next();
});

// Registration (no auth, rate limited internally)
app.post("/api/register", registerActor);

// Key management (no signature auth, uses recovery code)
app.post("/api/keys/rotate", rotateKey);
app.post("/api/keys/rotate-recovery", rotateRecovery);
app.post("/api/actors/delete", deleteActor);

// Chat & message routes (Ed25519 signature auth)
app.use("/api/chat/*", signatureAuth);
app.use("/api/chat", signatureAuth);

app.post("/api/chat/dm", startOrResumeDm);
app.get("/api/chat/dm", listDms);
app.post("/api/chat", createChat);
app.get("/api/chat", listChats);
app.get("/api/chat/:id", getChat);
app.post("/api/chat/:id/join", joinChat);
app.post("/api/chat/:id/messages", sendMessage);
app.get("/api/chat/:id/messages", listMessages);

// 404 fallback
app.notFound((c) => c.json({ error: "Not found" }, 404));

// Error handler
app.onError((err, c) => {
  console.error("[chat-worker] unhandled error:", err);
  return c.json({ error: "Internal server error" }, 500);
});

export default {
  fetch: app.fetch,
} satisfies ExportedHandler<Env>;
