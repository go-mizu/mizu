import { Hono } from "hono";
import { cors } from "hono/cors";
import { bearerAuth, createChallenge, verifyChallenge } from "./auth";
import { requestMagicLink, verifyMagicLink, logout } from "./magic";
import { getSessionActor } from "./session";
import { registerActor } from "./register";
import { uploadFile, downloadFile, deleteFile, headFile } from "./files";
import { createFolder, listFolder, deleteFolder } from "./folders";
import {
  createShare, listShares, updateShare, deleteShare,
  listSharedWithMe, downloadSharedFile, uploadSharedFile, deleteSharedFile,
} from "./shares";
import { presignUpload, presignDownload, presignComplete } from "./presign";
import {
  toggleStar, renameItem, moveItems, copyFile,
  trashItems, restoreItems, emptyTrash, listTrash,
  listRecent, listStarred, searchFiles, driveStats, updateDescription,
} from "./drive";
import { createLink, listLinks, deleteLink, accessPublicLink, accessPublicLinkFile } from "./links";
import { createApiKey, listApiKeys, deleteApiKey } from "./api-keys";
import { getAuditLog } from "./audit";
import { landingPage } from "./landing";
import { docsPage } from "./docs";
import { pricingPage } from "./pricing";
import { browsePage } from "./browse";
import { aiPage } from "./ai";
import { mcpHandler, mcpInfo } from "./mcp";
import {
  protectedResourceMetadata, authorizationServerMetadata,
  registerClient, authorizeEndpoint, authorizeSubmit,
  oauthMagicCallback, tokenEndpoint,
} from "./oauth";
import {
  authRateLimit, magicLinkRateLimit, registerRateLimit,
  uploadRateLimit, shareRateLimit, linkRateLimit, publicAccessRateLimit,
} from "./rate-limit";
import type { Env, Variables } from "./types";

const app = new Hono<{ Bindings: Env; Variables: Variables }>();

app.use("*", cors());

// Pages (no auth — session read optionally for signed-in state)
app.get("/", async (c) => {
  const actor = await getSessionActor(c);
  return c.html(landingPage(actor));
});
app.get("/docs", (c) => c.html(docsPage()));
app.get("/pricing", (c) => c.html(pricingPage()));
app.get("/ai", (c) => c.html(aiPage()));
app.get("/browse", browsePage);
app.get("/browse/*", browsePage);

// Body size limit for small API routes
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
app.use("/folders", bodySizeLimit);
app.use("/folders/*", bodySizeLimit);
app.use("/shares", bodySizeLimit);
app.use("/shares/*", bodySizeLimit);
app.use("/presign/*", bodySizeLimit);
app.use("/drive/*", bodySizeLimit);
app.use("/links", bodySizeLimit);
app.use("/links/*", bodySizeLimit);
app.use("/api-keys", bodySizeLimit);
app.use("/api-keys/*", bodySizeLimit);

// Rate limiting on unauthenticated endpoints
app.use("/actors", registerRateLimit);
app.use("/auth/challenge", authRateLimit);
app.use("/auth/verify", authRateLimit);
app.use("/auth/magic-link", magicLinkRateLimit);

// Registration (no auth)
app.post("/actors", registerActor);

// Auth (no auth required)
app.post("/auth/challenge", createChallenge);
app.post("/auth/verify", verifyChallenge);
app.post("/auth/magic-link", requestMagicLink);
app.get("/auth/magic/:token", verifyMagicLink);
app.post("/auth/logout", logout);
app.get("/auth/logout", logout);

// OAuth well-known metadata (no auth)
app.get("/.well-known/oauth-protected-resource", protectedResourceMetadata);
app.get("/.well-known/oauth-authorization-server", authorizationServerMetadata);

// OAuth endpoints (no auth, rate limited)
app.use("/oauth/*", bodySizeLimit);
app.post("/oauth/register", registerClient);
app.get("/oauth/authorize", authorizeEndpoint);
app.post("/oauth/authorize", authorizeSubmit);
app.get("/oauth/callback/:token", oauthMagicCallback);
app.post("/oauth/token", tokenEndpoint);

// MCP endpoint (JSON-RPC 2.0)
app.get("/mcp", mcpInfo);
// Add WWW-Authenticate header on 401 for OAuth discovery (RFC 9728)
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

// Public links (no auth required, rate limited)
app.use("/p/*", publicAccessRateLimit);
app.get("/p/:token", accessPublicLink);
app.get("/p/:token/*", accessPublicLinkFile);

// Protected routes (Bearer token or session cookie or API key)
app.use("/files/*", bearerAuth);
app.use("/folders", bearerAuth);
app.use("/folders/*", bearerAuth);
app.use("/shares", bearerAuth);
app.use("/shares/*", bearerAuth);
app.use("/shared", bearerAuth);
app.use("/shared/*", bearerAuth);
app.use("/drive/*", bearerAuth);
app.use("/presign/*", bearerAuth);
app.use("/links", bearerAuth);
app.use("/links/*", bearerAuth);
app.use("/api-keys", bearerAuth);
app.use("/api-keys/*", bearerAuth);
app.use("/audit", bearerAuth);

// Rate limiting on authenticated write endpoints
app.use("/files/*", uploadRateLimit);
app.use("/shares", shareRateLimit);
app.use("/links", linkRateLimit);

// Drive features
app.patch("/drive/star", toggleStar);
app.post("/drive/rename", renameItem);
app.post("/drive/move", moveItems);
app.post("/drive/copy", copyFile);
app.post("/drive/trash", trashItems);
app.post("/drive/restore", restoreItems);
app.delete("/drive/trash", emptyTrash);
app.get("/drive/trash", listTrash);
app.get("/drive/recent", listRecent);
app.get("/drive/starred", listStarred);
app.get("/drive/search", searchFiles);
app.get("/drive/stats", driveStats);
app.patch("/drive/description", updateDescription);

// Presigned URLs (direct-to-storage)
app.post("/presign/upload", presignUpload);
app.post("/presign/download", presignDownload);
app.post("/presign/complete", presignComplete);

// Files
app.put("/files/*", uploadFile);
app.get("/files/*", downloadFile);
app.delete("/files/*", deleteFile);
app.on("HEAD", "/files/*", headFile);

// Folders
app.post("/folders", createFolder);
app.get("/folders", listFolder);
app.get("/folders/*", listFolder);
app.delete("/folders/*", deleteFolder);

// Shares
app.post("/shares", createShare);
app.get("/shares", listShares);
app.patch("/shares/:id", updateShare);
app.delete("/shares/:id", deleteShare);

// Shared files (download, upload, delete)
app.get("/shared", listSharedWithMe);
app.get("/shared/:owner/*", downloadSharedFile);
app.put("/shared/:owner/*", uploadSharedFile);
app.delete("/shared/:owner/*", deleteSharedFile);

// Public links management
app.post("/links", createLink);
app.get("/links", listLinks);
app.delete("/links/:id", deleteLink);

// API keys management
app.post("/api-keys", createApiKey);
app.get("/api-keys", listApiKeys);
app.delete("/api-keys/:id", deleteApiKey);

// Audit log
app.get("/audit", getAuditLog);

// 404 fallback
app.notFound((c) => c.json({ error: { code: "not_found", message: "Not found" } }, 404));

// Error handler
app.onError((err, c) => {
  console.error("[storage-worker] unhandled error:", err);
  return c.json({ error: { code: "internal", message: "Internal server error" } }, 500);
});

export default {
  fetch: app.fetch,
} satisfies ExportedHandler<Env>;
