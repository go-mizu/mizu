import { Hono } from "hono";
import { cors } from "hono/cors";
import { bearerAuth } from "./middleware/auth";
import { createChallenge, verifyChallenge, requestMagicLink, verifyMagicLink, logout, registerActor } from "./routes/auth";
import { getSessionActor } from "./pages/session";
import {
  createBucket, listBuckets, getBucket, updateBucket, deleteBucket, emptyBucket,
} from "./routes/buckets";
import {
  createObject, upsertObject, downloadObject, downloadPublicObject,
  headObject, objectInfo, deleteObjects, listObjects,
  moveObject, copyObject,
} from "./routes/objects";
import {
  createSignedUrl, createSignedUploadUrl, accessSignedUrl, uploadViaSignedUrl,
} from "./routes/signed";
import { createApiKey, listApiKeys, deleteApiKey } from "./routes/api-keys";
import { getAuditLog } from "./lib/audit";
import { uploadFile, downloadFile, deleteFile, headFile } from "./routes/files";
import { createFolder, listFolder, deleteFolder } from "./routes/folders";
import {
  toggleStar, renameItem, moveItems, copyFile,
  trashItems, restoreItems, emptyTrash, listTrash,
  listRecent, listStarred, searchFiles, driveStats, updateDescription,
} from "./routes/drive";
import { createShare, listShares, listSharedWithMe } from "./routes/shares";
import { accessPublicLink, accessPublicLinkFile } from "./routes/links";
import { homePage } from "./pages/home";
import { developersPage } from "./pages/developers";
import { docsPage } from "./pages/docs";
import { pricingPage } from "./pages/pricing";
import { browsePage } from "./pages/browse";
import { aiPage } from "./pages/ai";
import { cliPage } from "./pages/cli";
import { mcpHandler, mcpInfo } from "./routes/mcp";
import {
  protectedResourceMetadata, authorizationServerMetadata,
  registerClient, authorizeEndpoint, authorizeSubmit,
  oauthMagicCallback, tokenEndpoint,
} from "./routes/oauth";
import {
  authRateLimit, magicLinkRateLimit, registerRateLimit,
  uploadRateLimit, publicAccessRateLimit,
} from "./middleware/rate-limit";
import type { Env, Variables } from "./types";

const app = new Hono<{ Bindings: Env; Variables: Variables }>();

app.use("*", cors());

// ── Pages (no auth — session read optionally for signed-in state) ─────
app.get("/", async (c) => {
  const actor = await getSessionActor(c);
  return c.html(homePage(actor));
});
app.get("/developers", async (c) => {
  const actor = await getSessionActor(c);
  return c.html(developersPage(actor));
});
app.get("/api", (c) => c.html(docsPage()));
app.get("/pricing", (c) => c.html(pricingPage()));
app.get("/ai", (c) => c.html(aiPage()));
app.get("/cli", async (c) => {
  const actor = await getSessionActor(c);
  return c.html(cliPage(actor));
});
app.get("/browse", browsePage);
app.get("/browse/*", browsePage);

// ── Body size limit for small API routes ──────────────────────────────
const MAX_BODY_SIZE = 65_536;
const bodySizeLimit = async (c: any, next: any) => {
  const cl = c.req.header("Content-Length");
  if (cl && parseInt(cl, 10) > MAX_BODY_SIZE) {
    return c.json({ error: { code: "invalid_request", message: "Request body too large" } }, 413);
  }
  await next();
};

// ── Rate limiting on unauthenticated endpoints ────────────────────────
app.use("/auth/register", registerRateLimit);
app.use("/auth/challenge", authRateLimit);
app.use("/auth/verify", authRateLimit);
app.use("/auth/magic-link", magicLinkRateLimit);

// ── Auth (no auth required) ───────────────────────────────────────────
app.use("/auth/*", bodySizeLimit);
app.post("/auth/register", registerActor);
app.post("/auth/challenge", createChallenge);
app.post("/auth/verify", verifyChallenge);
app.post("/auth/magic-link", requestMagicLink);
app.get("/auth/magic/:token", verifyMagicLink);
app.post("/auth/logout", logout);
app.get("/auth/logout", logout);

// ── OAuth (no auth) ──────────────────────────────────────────────────
app.get("/.well-known/oauth-protected-resource", protectedResourceMetadata);
app.get("/.well-known/oauth-authorization-server", authorizationServerMetadata);
app.use("/oauth/*", bodySizeLimit);
app.post("/oauth/register", registerClient);
app.get("/oauth/authorize", authorizeEndpoint);
app.post("/oauth/authorize", authorizeSubmit);
app.get("/oauth/callback/:token", oauthMagicCallback);
app.post("/oauth/token", tokenEndpoint);

// ── MCP (JSON-RPC 2.0 with OAuth) ───────────────────────────────────
app.get("/mcp", mcpInfo);
app.use("/mcp", async (c, next) => {
  await next();
  if (c.res.status === 401) {
    const origin = new URL(c.req.url).origin;
    const body = await c.res.text();
    c.res = new Response(body, {
      status: 401,
      headers: {
        ...Object.fromEntries(c.res.headers.entries()),
        "WWW-Authenticate": `Bearer resource_metadata="${origin}/.well-known/oauth-protected-resource"`,
      },
    });
  }
});
app.use("/mcp", bearerAuth);
app.use("/mcp", bodySizeLimit);
app.post("/mcp", mcpHandler);

// ── Signed URL access (no auth) ─────────────────────────────────────
app.use("/sign/*", publicAccessRateLimit);
app.get("/sign/:token", accessSignedUrl);
app.use("/upload/sign/*", publicAccessRateLimit);
app.put("/upload/sign/:token", uploadViaSignedUrl);

// ── Public object access (no auth) ──────────────────────────────────
app.use("/object/public/*", publicAccessRateLimit);
app.get("/object/public/*", downloadPublicObject);

// ── Public links (no auth) ──────────────────────────────────────────
app.use("/p/*", publicAccessRateLimit);
app.get("/p/:token", accessPublicLink);
app.get("/p/:token/*", accessPublicLinkFile);

// ── Protected routes (Bearer token, session cookie, or API key) ─────
app.use("/files", bearerAuth);
app.use("/files/*", bearerAuth);
app.use("/folders", bearerAuth);
app.use("/folders/*", bearerAuth);
app.use("/drive/*", bearerAuth);
app.use("/shares", bearerAuth);
app.use("/shares/*", bearerAuth);
app.use("/shared", bearerAuth);
app.use("/bucket", bearerAuth);
app.use("/bucket/*", bearerAuth);
app.use("/object/*", bearerAuth);
app.use("/keys", bearerAuth);
app.use("/keys/*", bearerAuth);
app.use("/audit", bearerAuth);

// ── Drive (browse page API) ────────────────────────────────────────
app.get("/drive/search", searchFiles);
app.get("/drive/recent", listRecent);
app.get("/drive/starred", listStarred);
app.get("/drive/trash", listTrash);
app.get("/drive/stats", driveStats);
app.post("/drive/rename", bodySizeLimit, renameItem);
app.post("/drive/move", bodySizeLimit, moveItems);
app.post("/drive/copy", bodySizeLimit, copyFile);
app.post("/drive/trash", bodySizeLimit, trashItems);
app.post("/drive/restore", bodySizeLimit, restoreItems);
app.delete("/drive/trash", emptyTrash);
app.patch("/drive/star", bodySizeLimit, toggleStar);
app.patch("/drive/description", bodySizeLimit, updateDescription);

// ── Files (browse page upload/download) ─────────────────────────────
app.put("/files/*", uploadFile);
app.get("/files/*", downloadFile);
app.delete("/files/*", deleteFile);
app.on("HEAD", "/files/*", headFile);

// ── Folders ─────────────────────────────────────────────────────────
app.post("/folders", bodySizeLimit, createFolder);
app.get("/folders/*", listFolder);
app.get("/folders", listFolder);
app.delete("/folders/*", deleteFolder);

// ── Shares ─────────────────────────────────────────────────────────
app.post("/shares", bodySizeLimit, createShare);
app.get("/shares", listShares);
app.get("/shared", listSharedWithMe);

// ── Rate limiting on writes ─────────────────────────────────────────
app.use("/object/*", uploadRateLimit);

// ── Buckets ─────────────────────────────────────────────────────────
app.use("/bucket", bodySizeLimit);
app.use("/bucket/*", bodySizeLimit);
app.post("/bucket", createBucket);
app.get("/bucket", listBuckets);
app.get("/bucket/:id", getBucket);
app.patch("/bucket/:id", updateBucket);
app.delete("/bucket/:id", deleteBucket);
app.post("/bucket/:id/empty", emptyBucket);

// ── Objects ─────────────────────────────────────────────────────────
app.post("/object/list/:bucket", bodySizeLimit, listObjects);
app.post("/object/move", bodySizeLimit, moveObject);
app.post("/object/copy", bodySizeLimit, copyObject);
app.post("/object/sign/:bucket", bodySizeLimit, createSignedUrl);
app.post("/object/upload/sign/*", bodySizeLimit, createSignedUploadUrl);
app.get("/object/info/*", objectInfo);
app.post("/object/*", createObject);
app.put("/object/*", upsertObject);
app.get("/object/*", downloadObject);
app.on("HEAD", "/object/*", headObject);
app.delete("/object/:bucket", bodySizeLimit, deleteObjects);

// ── Keys ────────────────────────────────────────────────────────────
app.use("/keys", bodySizeLimit);
app.use("/keys/*", bodySizeLimit);
app.post("/keys", createApiKey);
app.get("/keys", listApiKeys);
app.delete("/keys/:id", deleteApiKey);

// ── Audit ───────────────────────────────────────────────────────────
app.get("/audit", getAuditLog);

// ── 404 fallback ────────────────────────────────────────────────────
app.notFound((c) => c.json({ error: { code: "not_found", message: "Not found" } }, 404));

// ── Error handler ───────────────────────────────────────────────────
app.onError((err, c) => {
  console.error("[storage-worker] unhandled error:", err);
  return c.json({ error: { code: "internal", message: "Internal server error" } }, 500);
});

export default {
  fetch: app.fetch,
} satisfies ExportedHandler<Env>;
