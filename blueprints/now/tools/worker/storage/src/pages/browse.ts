import type { Context } from "hono";
import type { Env, Variables } from "../types";
import { getSessionActor } from "./session";

type C = Context<{ Bindings: Env; Variables: Variables }>;

function esc(s: string): string {
  return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");
}

export async function browsePage(c: C) {
  const actor = await getSessionActor(c);
  if (!actor) return c.html(browseLanding());
  return c.html(browseApp(actor));
}

const HEAD = `<meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1,viewport-fit=cover">
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800&family=JetBrains+Mono:wght@400;500;600;700&display=swap" rel="stylesheet">
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/katex@0.16.21/dist/katex.min.css">
<link rel="stylesheet" href="/browse.css">`;

function browseLanding(): string {
  return `<!DOCTYPE html>
<html lang="en" class="dark">
<head>
${HEAD}
<title>Browse — storage.now</title>
</head>
<body>
<div class="dot-bg"></div>
<main id="main"></main>
<div id="ctx-menu"></div>
<div id="cmd-palette"></div>
<div id="modal-root"></div>
<div class="toast-wrap" id="toasts"></div>
<script>window.__BROWSE_CONFIG={mode:'demo'}</script>
<script defer src="https://cdn.jsdelivr.net/npm/katex@0.16.21/dist/katex.min.js"></script>
<script defer src="https://cdn.jsdelivr.net/npm/katex@0.16.21/dist/contrib/auto-render.min.js"></script>
<script defer src="/browse.js"></script>
</body></html>`;
}

function browseApp(actor: string): string {
  return `<!DOCTYPE html>
<html lang="en" class="dark">
<head>
${HEAD}
<title>Browse — storage.now</title>
</head>
<body>
<div class="dot-bg"></div>
<main id="main"></main>
<div id="ctx-menu"></div>
<div id="cmd-palette"></div>
<div id="modal-root"></div>
<div class="upload-panel" id="upload-panel">
  <div class="upload-head"><span id="upload-title">Uploading...</span><button id="upload-close">&times;</button></div>
  <div class="upload-list" id="upload-list"></div>
</div>
<div class="toast-wrap" id="toasts"></div>
<input type="file" id="file-input" multiple style="display:none">
<script>window.__BROWSE_CONFIG={mode:'auth',actor:${JSON.stringify(actor)}}</script>
<script defer src="https://cdn.jsdelivr.net/npm/katex@0.16.21/dist/katex.min.js"></script>
<script defer src="https://cdn.jsdelivr.net/npm/katex@0.16.21/dist/contrib/auto-render.min.js"></script>
<script defer src="/browse.js"></script>
</body></html>`;
}
