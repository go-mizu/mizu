import { Hono } from "hono";
import { cors } from "hono/cors";
import { bearerAuth, createChallenge, verifyChallenge } from "./auth";
import { requestMagicLink, verifyMagicLink, logout } from "./magic";
import { getSessionActor } from "./session";
import { registerActor } from "./register";
import { createChat, getChat, listChats } from "./chat";
import { sendMessageUnified, sendMessageExplicit, listMessages } from "./message";
import { listMembers, addMember, removeMember, joinChat, leaveChat } from "./member";
import { landingPage } from "./landing";
import { docsPage } from "./docs";
import { humansPage } from "./humans";
import { agentsPage } from "./agents";
import { roomsPage } from "./rooms";
import { humanProfile, agentProfile } from "./profile";
import { roomDetailPage } from "./room";
import { myChatsPage } from "./mychats";
import { chatViewPage } from "./chatview";
import { sseMessages } from "./sse";
import type { Env, Variables } from "./types";

const app = new Hono<{ Bindings: Env; Variables: Variables }>();

app.use("*", cors());

// Pages (no auth — session read optionally for signed-in state)
app.get("/", async (c) => {
  const actor = await getSessionActor(c);
  return c.html(landingPage(actor));
});
app.get("/docs", (c) => c.html(docsPage()));
app.get("/humans", humansPage);
app.get("/agents", agentsPage);
app.get("/rooms", roomsPage);
app.get("/u/:id", humanProfile);
app.get("/a/:id", agentProfile);
app.get("/r/:room_id", roomDetailPage);
app.get("/my-chats", myChatsPage);
app.get("/chat/:chat_id", chatViewPage);

// Body size limit for API routes
const MAX_BODY_SIZE = 65_536;
const bodySizeLimit = async (c: any, next: any) => {
  const cl = c.req.header("Content-Length");
  if (cl && parseInt(cl, 10) > MAX_BODY_SIZE) {
    return c.json({ error: { code: "invalid_request", message: "Request body too large" } }, 413);
  }
  await next();
};

app.use("/actors", bodySizeLimit);
app.use("/auth/*", bodySizeLimit);
app.use("/chats", bodySizeLimit);
app.use("/chats/*", bodySizeLimit);
app.use("/messages", bodySizeLimit);

// Registration (no auth)
app.post("/actors", registerActor);

// Auth (no auth required)
app.post("/auth/challenge", createChallenge);
app.post("/auth/verify", verifyChallenge);
app.post("/auth/magic-link", requestMagicLink);
app.get("/auth/magic/:token", verifyMagicLink);
app.post("/auth/logout", logout);
app.get("/auth/logout", logout);

// Protected routes (Bearer token or session cookie)
app.use("/chats", bearerAuth);
app.use("/chats/*", bearerAuth);
app.use("/messages", bearerAuth);
app.use("/sse/*", bearerAuth);

// Chats
app.post("/chats", createChat);
app.get("/chats", listChats);
app.get("/chats/:chat_id", getChat);

// Messages
app.post("/messages", sendMessageUnified);
app.post("/chats/:chat_id/messages", sendMessageExplicit);
app.get("/chats/:chat_id/messages", listMessages);

// Members
app.get("/chats/:chat_id/members", listMembers);
app.post("/chats/:chat_id/members", addMember);
app.delete("/chats/:chat_id/members/:actor", removeMember);
app.post("/chats/:chat_id/join", joinChat);
app.post("/chats/:chat_id/leave", leaveChat);

// SSE
app.get("/sse/chats/:chat_id", sseMessages);

// 404 fallback
app.notFound((c) => c.json({ error: { code: "not_found", message: "Not found" } }, 404));

// Error handler
app.onError((err, c) => {
  console.error("[chat-worker] unhandled error:", err);
  return c.json({ error: { code: "internal", message: "Internal server error" } }, 500);
});

export default {
  fetch: app.fetch,
} satisfies ExportedHandler<Env>;
