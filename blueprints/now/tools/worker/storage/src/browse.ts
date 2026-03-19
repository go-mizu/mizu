import type { Context } from "hono";
import type { Env, Variables } from "./types";
import { getSessionActor } from "./session";

type AppContext = Context<{ Bindings: Env; Variables: Variables }>;

function esc(s: string): string {
  return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");
}

export async function browsePage(c: AppContext) {
  const actor = await getSessionActor(c);
  if (!actor) return c.html(browseLanding());
  return c.html(browseApp(actor));
}

/* ═══════════════════════════════════════════════════════════════════════
   BROWSE DEMO (unauthenticated — interactive demo with sample data)
   Shows a full working file browser with preview for all file types.
   Single-click opens preview (like Google Drive). All write ops → sign up.
   ═══════════════════════════════════════════════════════════════════════ */
function browseLanding(): string {
  return `<!DOCTYPE html>
<html lang="en" class="dark">
<head>
<meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Demo — storage.now</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800&family=JetBrains+Mono:wght@400;500;600;700&display=swap" rel="stylesheet">
<link rel="stylesheet" href="/browse.css">
</head>
<body>

<div class="demo-banner" id="demo-banner">
  <span class="demo-banner-icon"><svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg></span>
  <span>You're exploring a <strong>live demo</strong> — this is what your drive looks like. <a href="/" class="demo-banner-link">Sign up free</a> to create your own.</span>
  <button class="demo-banner-close" onclick="document.getElementById('demo-banner').remove()">&times;</button>
</div>

<nav>
  <a href="/" class="logo"><span class="logo-dot"></span> storage.now</a>
  <button class="mobile-toggle" id="mobile-toggle" aria-label="Menu">
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="3" y1="6" x2="21" y2="6"/><line x1="3" y1="12" x2="21" y2="12"/><line x1="3" y1="18" x2="21" y2="18"/></svg>
  </button>
  <div class="nav-search" id="nav-search">
    <span class="search-icon"><svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg></span>
    <input type="text" id="search-input" placeholder="Search files..." autocomplete="off" spellcheck="false">
  </div>
  <div class="nav-right">
    <span class="nav-user" style="opacity:.6">demo</span>
    <a href="/" class="nav-signout">sign up</a>
    <button class="theme-toggle" id="theme-btn" aria-label="Toggle theme">
      <svg class="icon-moon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 12.79A9 9 0 1111.21 3 7 7 0 0021 12.79z"/></svg>
      <svg class="icon-sun" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="5"/><line x1="12" y1="1" x2="12" y2="3"/><line x1="12" y1="21" x2="12" y2="23"/><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/><line x1="1" y1="12" x2="3" y2="12"/><line x1="21" y1="12" x2="23" y2="12"/><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/></svg>
    </button>
  </div>
</nav>

<div id="app">
  <div class="sidebar-backdrop" id="sidebar-backdrop"></div>
  <aside id="sidebar">
    <div class="sidebar-nav" id="sidebar-nav"></div>
    <div class="sidebar-quota" id="quota"></div>
  </aside>

  <main id="main">
    <div id="toolbar"></div>
    <div id="bulk-bar"></div>
    <div class="file-content" id="file-content"></div>
  </main>

  <aside id="detail-panel" class="hidden"></aside>
</div>

<div id="ctx-menu"></div>
<div id="modal-overlay"></div>
<div id="preview-overlay"></div>
<div class="toast-container" id="toasts"></div>

<script>
(function(){
'use strict';
const DEMO=true;

/* ── Demo filesystem ─────────────────────────────────────────────── */
const now=Date.now();
const h1=3600000,d1=86400000;
const DEMO_FS=[
  // Root folders
  {path:'documents/',name:'documents',is_folder:true,content_type:'',size:0,starred:true,description:'Work documents and notes',created_at:now-30*d1,updated_at:now-2*d1},
  {path:'images/',name:'images',is_folder:true,content_type:'',size:0,starred:false,description:'Photos and graphics',created_at:now-25*d1,updated_at:now-1*d1},
  {path:'projects/',name:'projects',is_folder:true,content_type:'',size:0,starred:true,description:'Code projects',created_at:now-20*d1,updated_at:now-3*h1},
  {path:'shared/',name:'shared',is_folder:true,content_type:'',size:0,starred:false,description:'Files shared with collaborators',created_at:now-15*d1,updated_at:now-5*h1},
  {path:'backups/',name:'backups',is_folder:true,content_type:'',size:0,starred:false,description:'System backups',created_at:now-10*d1,updated_at:now-7*d1},

  // Root files
  {path:'README.md',name:'README.md',is_folder:false,content_type:'text/markdown',size:2048,starred:true,description:'Project overview',created_at:now-30*d1,updated_at:now-1*d1},
  {path:'notes.txt',name:'notes.txt',is_folder:false,content_type:'text/plain',size:847,starred:false,description:'Quick notes',created_at:now-5*d1,updated_at:now-2*h1},
  {path:'budget-2026.csv',name:'budget-2026.csv',is_folder:false,content_type:'text/csv',size:15360,starred:false,description:'Annual budget planning',created_at:now-8*d1,updated_at:now-3*d1},

  // documents/
  {path:'documents/proposal.pdf',name:'proposal.pdf',is_folder:false,content_type:'application/pdf',size:2457600,starred:true,description:'Client proposal Q1',created_at:now-12*d1,updated_at:now-2*d1},
  {path:'documents/meeting-notes.md',name:'meeting-notes.md',is_folder:false,content_type:'text/markdown',size:4096,starred:false,description:'Weekly standup notes',created_at:now-3*d1,updated_at:now-3*h1},
  {path:'documents/contracts/',name:'contracts',is_folder:true,content_type:'',size:0,starred:false,description:'Legal contracts',created_at:now-20*d1,updated_at:now-5*d1},
  {path:'documents/templates/',name:'templates',is_folder:true,content_type:'',size:0,starred:false,description:'Document templates',created_at:now-18*d1,updated_at:now-8*d1},
  {path:'documents/report-final.docx',name:'report-final.docx',is_folder:false,content_type:'application/vnd.openxmlformats-officedocument.wordprocessingml.document',size:892416,starred:false,description:'Annual report',created_at:now-7*d1,updated_at:now-4*d1},
  {path:'documents/slides-keynote.pptx',name:'slides-keynote.pptx',is_folder:false,content_type:'application/vnd.openxmlformats-officedocument.presentationml.presentation',size:5242880,starred:false,description:'Conference presentation',created_at:now-6*d1,updated_at:now-2*d1},

  // documents/contracts/
  {path:'documents/contracts/nda-acme.pdf',name:'nda-acme.pdf',is_folder:false,content_type:'application/pdf',size:184320,starred:false,description:'NDA with Acme Corp',created_at:now-15*d1,updated_at:now-15*d1},
  {path:'documents/contracts/sow-2026.pdf',name:'sow-2026.pdf',is_folder:false,content_type:'application/pdf',size:256000,starred:false,description:'Statement of work',created_at:now-10*d1,updated_at:now-5*d1},

  // documents/templates/
  {path:'documents/templates/invoice.docx',name:'invoice.docx',is_folder:false,content_type:'application/vnd.openxmlformats-officedocument.wordprocessingml.document',size:45056,starred:false,description:'Invoice template',created_at:now-18*d1,updated_at:now-18*d1},
  {path:'documents/templates/letterhead.docx',name:'letterhead.docx',is_folder:false,content_type:'application/vnd.openxmlformats-officedocument.wordprocessingml.document',size:32768,starred:false,description:'Company letterhead',created_at:now-18*d1,updated_at:now-12*d1},
  {path:'documents/analytics.xlsx',name:'analytics.xlsx',is_folder:false,content_type:'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',size:245760,starred:false,description:'Monthly analytics data',created_at:now-9*d1,updated_at:now-2*d1},

  // images/
  {path:'images/hero-banner.png',name:'hero-banner.png',is_folder:false,content_type:'image/png',size:3145728,starred:false,description:'Website hero image',created_at:now-10*d1,updated_at:now-1*d1},
  {path:'images/team-photo.jpg',name:'team-photo.jpg',is_folder:false,content_type:'image/jpeg',size:4718592,starred:true,description:'Team offsite 2026',created_at:now-8*d1,updated_at:now-8*d1},
  {path:'images/logo.svg',name:'logo.svg',is_folder:false,content_type:'image/svg+xml',size:8192,starred:false,description:'Company logo vector',created_at:now-20*d1,updated_at:now-3*d1},
  {path:'images/screenshots/',name:'screenshots',is_folder:true,content_type:'',size:0,starred:false,description:'Product screenshots',created_at:now-7*d1,updated_at:now-1*d1},
  {path:'images/icons/',name:'icons',is_folder:true,content_type:'',size:0,starred:false,description:'App icons',created_at:now-12*d1,updated_at:now-6*d1},

  // images/screenshots/
  {path:'images/screenshots/dashboard-v2.png',name:'dashboard-v2.png',is_folder:false,content_type:'image/png',size:1572864,starred:false,description:'Dashboard redesign',created_at:now-5*d1,updated_at:now-1*d1},
  {path:'images/screenshots/mobile-app.png',name:'mobile-app.png',is_folder:false,content_type:'image/png',size:2097152,starred:false,description:'Mobile app capture',created_at:now-3*d1,updated_at:now-3*d1},

  // projects/
  {path:'projects/api-server/',name:'api-server',is_folder:true,content_type:'',size:0,starred:true,description:'Main API backend',created_at:now-20*d1,updated_at:now-3*h1},
  {path:'projects/landing-page/',name:'landing-page',is_folder:true,content_type:'',size:0,starred:false,description:'Marketing website',created_at:now-15*d1,updated_at:now-1*d1},
  {path:'projects/ml-pipeline/',name:'ml-pipeline',is_folder:true,content_type:'',size:0,starred:false,description:'ML training pipeline',created_at:now-8*d1,updated_at:now-2*d1},

  // projects/api-server/
  {path:'projects/api-server/main.go',name:'main.go',is_folder:false,content_type:'text/x-go',size:4096,starred:false,description:'Server entrypoint',created_at:now-20*d1,updated_at:now-3*h1},
  {path:'projects/api-server/go.mod',name:'go.mod',is_folder:false,content_type:'text/plain',size:512,starred:false,description:'Go module file',created_at:now-20*d1,updated_at:now-5*d1},
  {path:'projects/api-server/Dockerfile',name:'Dockerfile',is_folder:false,content_type:'text/plain',size:1024,starred:false,description:'Container build file',created_at:now-18*d1,updated_at:now-2*d1},
  {path:'projects/api-server/config.yaml',name:'config.yaml',is_folder:false,content_type:'text/yaml',size:2048,starred:false,description:'Service configuration',created_at:now-15*d1,updated_at:now-6*h1},

  // projects/landing-page/
  {path:'projects/landing-page/index.html',name:'index.html',is_folder:false,content_type:'text/html',size:8192,starred:false,description:'Homepage HTML',created_at:now-15*d1,updated_at:now-1*d1},
  {path:'projects/landing-page/style.css',name:'style.css',is_folder:false,content_type:'text/css',size:12288,starred:false,description:'Main stylesheet',created_at:now-15*d1,updated_at:now-1*d1},
  {path:'projects/landing-page/app.js',name:'app.js',is_folder:false,content_type:'application/javascript',size:6144,starred:false,description:'Client JS bundle',created_at:now-10*d1,updated_at:now-2*d1},

  // projects/ml-pipeline/
  {path:'projects/ml-pipeline/train.py',name:'train.py',is_folder:false,content_type:'text/x-python',size:16384,starred:false,description:'Model training script',created_at:now-8*d1,updated_at:now-2*d1},
  {path:'projects/ml-pipeline/requirements.txt',name:'requirements.txt',is_folder:false,content_type:'text/plain',size:256,starred:false,description:'Python dependencies',created_at:now-8*d1,updated_at:now-5*d1},
  {path:'projects/ml-pipeline/model-v3.bin',name:'model-v3.bin',is_folder:false,content_type:'application/octet-stream',size:52428800,starred:true,description:'Trained model checkpoint',created_at:now-4*d1,updated_at:now-2*d1},

  // shared/
  {path:'shared/design-system.fig',name:'design-system.fig',is_folder:false,content_type:'application/octet-stream',size:8388608,starred:false,description:'Figma design file (from u/alice)',created_at:now-6*d1,updated_at:now-1*d1},
  {path:'shared/quarterly-review.pdf',name:'quarterly-review.pdf',is_folder:false,content_type:'application/pdf',size:1048576,starred:false,description:'Q4 review deck (from a/reports-bot)',created_at:now-3*d1,updated_at:now-3*d1},
  {path:'shared/api-spec.yaml',name:'api-spec.yaml',is_folder:false,content_type:'text/yaml',size:32768,starred:false,description:'OpenAPI spec (from u/bob)',created_at:now-10*d1,updated_at:now-5*d1},

  // backups/
  {path:'backups/db-2026-03-01.sql.gz',name:'db-2026-03-01.sql.gz',is_folder:false,content_type:'application/gzip',size:15728640,starred:false,description:'Database backup March 1',created_at:now-18*d1,updated_at:now-18*d1},
  {path:'backups/db-2026-03-15.sql.gz',name:'db-2026-03-15.sql.gz',is_folder:false,content_type:'application/gzip',size:16777216,starred:false,description:'Database backup March 15',created_at:now-4*d1,updated_at:now-4*d1},
  {path:'backups/config-snapshot.tar.gz',name:'config-snapshot.tar.gz',is_folder:false,content_type:'application/gzip',size:524288,starred:false,description:'Configuration snapshot',created_at:now-7*d1,updated_at:now-7*d1},

  // media/
  {path:'media/',name:'media',is_folder:true,content_type:'',size:0,starred:false,description:'Audio and video files',created_at:now-12*d1,updated_at:now-2*d1},
  {path:'media/podcast-episode.mp3',name:'podcast-episode.mp3',is_folder:false,content_type:'audio/mpeg',size:8388608,starred:false,description:'Weekly podcast episode 12',created_at:now-5*d1,updated_at:now-5*d1},
  {path:'media/product-demo.mp4',name:'product-demo.mp4',is_folder:false,content_type:'video/mp4',size:26214400,starred:true,description:'Product demo recording',created_at:now-3*d1,updated_at:now-3*d1},
  {path:'media/notification.wav',name:'notification.wav',is_folder:false,content_type:'audio/wav',size:524288,starred:false,description:'App notification sound',created_at:now-8*d1,updated_at:now-8*d1},
  {path:'media/team-standup.webm',name:'team-standup.webm',is_folder:false,content_type:'video/webm',size:15728640,starred:false,description:'Weekly standup recording',created_at:now-2*d1,updated_at:now-2*d1},

  // Trash items
  {path:'old-logo.png',name:'old-logo.png',is_folder:false,content_type:'image/png',size:1048576,starred:false,description:'Deprecated logo',created_at:now-30*d1,updated_at:now-20*d1,trashed_at:now-2*d1},
  {path:'draft-v1.md',name:'draft-v1.md',is_folder:false,content_type:'text/markdown',size:3072,starred:false,description:'First draft (superseded)',created_at:now-15*d1,updated_at:now-10*d1,trashed_at:now-1*d1},
];

// Shared-with-me items
const DEMO_SHARED=[
  {owner:'u/alice',path:'design-system.fig',name:'design-system.fig',is_folder:false,content_type:'application/octet-stream',size:8388608,permission:'editor',updated_at:now-1*d1},
  {owner:'a/reports-bot',path:'quarterly-review.pdf',name:'quarterly-review.pdf',is_folder:false,content_type:'application/pdf',size:1048576,permission:'viewer',updated_at:now-3*d1},
  {owner:'u/bob',path:'api-spec.yaml',name:'api-spec.yaml',is_folder:false,content_type:'text/yaml',size:32768,permission:'viewer',updated_at:now-5*d1},
  {owner:'u/alice',path:'wireframes/',name:'wireframes',is_folder:true,content_type:'',size:0,permission:'viewer',updated_at:now-2*d1},
];

const DEMO_STATS={total_size:192923648,file_count:37,folder_count:13,trash_count:2,quota:5368709120};

const DEMO_CONTENT={
'README.md':'# Storage Platform\\n\\nA modern file storage and sharing platform built for teams.\\n\\n## Features\\n\\n- **Fast uploads** with resumable chunked transfer\\n- **Real-time collaboration** with shared folders\\n- **Version history** for all file types\\n- **Full-text search** across documents\\n\\n## Getting Started\\n\\nInstall and run:\\n  npm install\\n  npm run dev\\n\\nVisit http://localhost:3000 to open the dashboard.\\n\\n## Architecture\\n\\n- Frontend: TypeScript + Hono\\n- Storage: R2 object storage\\n- Auth: Session-based with OAuth\\n\\n## API\\n\\n| Endpoint | Method | Description |\\n|----------|--------|-------------|\\n| /files/* | GET | Download file |\\n| /files/* | PUT | Upload file |\\n| /folders/* | GET | List folder |\\n| /drive/search | GET | Search files |\\n\\n> **Note**: All endpoints require authentication except /browse.\\n\\n---\\n\\n*Built with Mizu framework.*',
'notes.txt':'Quick Notes\\n===========\\n\\nTODO for this week:\\n- Review pull request #142 (storage refactor)\\n- Update API docs for v2 endpoints\\n- Fix thumbnail generation for large PNGs\\n- Schedule 1:1 with design team\\n\\nMeeting notes 3/15:\\n  Alice suggested moving to R2 for cost savings.\\n  Bob will prototype the migration script.\\n  Timeline: 2 weeks for staging, 1 week for prod.\\n\\nIdeas:\\n- Add drag-and-drop folder upload\\n- Implement file versioning UI\\n- Add keyboard shortcut help modal',
'budget-2026.csv':'Department,Q1,Q2,Q3,Q4,Total\\nEngineering,125000,130000,135000,140000,530000\\nDesign,45000,47000,48000,50000,190000\\nMarketing,60000,65000,70000,75000,270000\\nSales,80000,85000,90000,95000,350000\\nOperations,35000,36000,37000,38000,146000\\nHR,25000,26000,27000,28000,106000\\nLegal,20000,21000,22000,23000,86000\\nExecutive,55000,55000,55000,55000,220000',
'documents/meeting-notes.md':'# Weekly Standup Notes\\n\\n## March 15, 2026\\n\\n### Attendees\\n- Alice (Engineering Lead)\\n- Bob (Backend)\\n- Carol (Frontend)\\n- Dave (DevOps)\\n\\n### Updates\\n\\n**Alice**: Finished the storage migration planning doc. PR #142 ready for review.\\n\\n**Bob**: Working on the search indexing pipeline. Found a bottleneck in the\\nfull-text search — switching to trigram indexes.\\n\\n**Carol**: Shipped the new file preview overlay. Next up: drag-and-drop upload.\\n\\n**Dave**: Deployed monitoring dashboards. Alert thresholds set for:\\n- API latency > 500ms\\n- Error rate > 1%\\n- Storage usage > 80%\\n\\n### Action Items\\n\\n- [ ] Alice: Merge storage migration PR by Wednesday\\n- [ ] Bob: Benchmark new search indexes\\n- [ ] Carol: Add keyboard navigation to preview\\n- [ ] Dave: Set up staging environment for R2 migration',
'projects/api-server/main.go':'package main\\n\\nimport (\\n\\t\\"fmt\\"\\n\\t\\"log\\"\\n\\t\\"net/http\\"\\n\\t\\"os\\"\\n\\t\\"time\\"\\n)\\n\\nfunc main() {\\n\\tmux := http.NewServeMux()\\n\\n\\tmux.HandleFunc(\\"GET /health\\", func(w http.ResponseWriter, r *http.Request) {\\n\\t\\tw.Header().Set(\\"Content-Type\\", \\"application/json\\")\\n\\t\\tfmt.Fprintf(w, \\"{\\\\\\"status\\\\\\":\\\\\\"ok\\\\\\",\\\\\\"time\\\\\\":\\\\\\"%s\\\\\\"}\\" , time.Now().Format(time.RFC3339))\\n\\t})\\n\\n\\tmux.HandleFunc(\\"GET /api/v1/files/\\", handleListFiles)\\n\\tmux.HandleFunc(\\"PUT /api/v1/files/\\", handleUploadFile)\\n\\tmux.HandleFunc(\\"DELETE /api/v1/files/\\", handleDeleteFile)\\n\\n\\tport := os.Getenv(\\"PORT\\")\\n\\tif port == \\"\\" {\\n\\t\\tport = \\"8080\\"\\n\\t}\\n\\n\\tlog.Printf(\\"Starting server on :%s\\", port)\\n\\tlog.Fatal(http.ListenAndServe(\\":\\"+port, mux))\\n}',
'projects/api-server/go.mod':'module github.com/acme/api-server\\n\\ngo 1.22.0\\n\\nrequire (\\n\\tgithub.com/go-mizu/mizu v0.12.0\\n\\tgithub.com/rs/cors v1.11.0\\n)',
'projects/api-server/Dockerfile':'FROM golang:1.22-alpine AS builder\\n\\nWORKDIR /app\\nCOPY go.mod go.sum ./\\nRUN go mod download\\n\\nCOPY . .\\nRUN CGO_ENABLED=0 go build -o /bin/server .\\n\\nFROM alpine:3.19\\nRUN apk add --no-cache ca-certificates\\nCOPY --from=builder /bin/server /bin/server\\n\\nEXPOSE 8080\\nENTRYPOINT [\\"/bin/server\\"]',
'projects/api-server/config.yaml':'server:\\n  port: 8080\\n  read_timeout: 30s\\n  write_timeout: 60s\\n  max_upload_size: 100MB\\n\\nstorage:\\n  driver: r2\\n  bucket: acme-files-prod\\n  region: auto\\n  public_url: https://cdn.acme.dev\\n\\nauth:\\n  session_ttl: 24h\\n  cookie_name: sid\\n  secure: true\\n\\nlogging:\\n  level: info\\n  format: json\\n\\nrate_limit:\\n  requests_per_minute: 120\\n  burst: 20',
'projects/landing-page/index.html':'<!DOCTYPE html>\\n<html lang=\\"en\\">\\n<head>\\n  <meta charset=\\"utf-8\\">\\n  <meta name=\\"viewport\\" content=\\"width=device-width, initial-scale=1\\">\\n  <title>Acme Cloud Storage</title>\\n  <link rel=\\"stylesheet\\" href=\\"/style.css\\">\\n</head>\\n<body>\\n  <nav class=\\"navbar\\">\\n    <a href=\\"/\\" class=\\"logo\\">Acme Storage</a>\\n    <div class=\\"nav-links\\">\\n      <a href=\\"/features\\">Features</a>\\n      <a href=\\"/pricing\\">Pricing</a>\\n      <a href=\\"/login\\" class=\\"btn btn-primary\\">Sign In</a>\\n    </div>\\n  </nav>\\n\\n  <main class=\\"hero\\">\\n    <h1>Store, share, and collaborate.</h1>\\n    <p>Secure cloud storage for modern teams.</p>\\n    <button class=\\"btn btn-lg\\" onclick=\\"app.start()\\">\\n      Get Started Free\\n    </button>\\n  </main>\\n\\n  <script src=\\"/app.js\\"><\\/script>\\n</body>\\n</html>',
'projects/landing-page/style.css':':root {\\n  --primary: #667eea;\\n  --primary-dark: #5a67d8;\\n  --bg: #0f172a;\\n  --surface: #1e293b;\\n  --text: #f8fafc;\\n  --text-muted: #94a3b8;\\n  --radius: 8px;\\n}\\n\\n* { margin: 0; padding: 0; box-sizing: border-box; }\\n\\nbody {\\n  font-family: Inter, system-ui, sans-serif;\\n  background: var(--bg);\\n  color: var(--text);\\n  line-height: 1.6;\\n}\\n\\n.navbar {\\n  display: flex;\\n  align-items: center;\\n  justify-content: space-between;\\n  padding: 16px 24px;\\n  border-bottom: 1px solid rgba(255,255,255,.06);\\n}\\n\\n.hero {\\n  text-align: center;\\n  padding: 120px 24px;\\n  max-width: 640px;\\n  margin: 0 auto;\\n}\\n\\n.hero h1 {\\n  font-size: 48px;\\n  font-weight: 800;\\n  letter-spacing: -0.03em;\\n  margin-bottom: 16px;\\n}\\n\\n.btn {\\n  display: inline-flex;\\n  padding: 10px 20px;\\n  border-radius: var(--radius);\\n  background: var(--primary);\\n  color: white;\\n  border: none;\\n  cursor: pointer;\\n  font-weight: 600;\\n}',
'projects/landing-page/app.js':'// Landing page interactions\\nconst app = {\\n  start() {\\n    window.location.href = \\"/register\\";\\n  },\\n\\n  async loadTestimonials() {\\n    const res = await fetch(\\"/api/testimonials\\");\\n    const data = await res.json();\\n    const container = document.getElementById(\\"testimonials\\");\\n    if (!container) return;\\n\\n    container.innerHTML = data.items\\n      .map(function(t) {\\n        return \\"<div class=testimonial-card>\\"\\n          + \\"<p class=quote>\\" + t.text + \\"</p>\\"\\n          + \\"<div class=author>\\" + t.author + \\"</div>\\"\\n          + \\"</div>\\";\\n      })\\n      .join(\\"\\");\\n  },\\n\\n  initScrollAnimations() {\\n    const observer = new IntersectionObserver(entries => {\\n      entries.forEach(e => {\\n        if (e.isIntersecting) e.target.classList.add(\\"visible\\");\\n      });\\n    }, { threshold: 0.1 });\\n\\n    document.querySelectorAll(\\".animate-in\\")\\n      .forEach(el => observer.observe(el));\\n  }\\n};\\n\\ndocument.addEventListener(\\"DOMContentLoaded\\", () => {\\n  app.loadTestimonials();\\n  app.initScrollAnimations();\\n});',
'projects/ml-pipeline/train.py':'import torch\\nimport torch.nn as nn\\nfrom torch.utils.data import DataLoader\\nfrom pathlib import Path\\nimport json\\nimport time\\n\\n# Hyperparameters\\nBATCH_SIZE = 64\\nLEARNING_RATE = 3e-4\\nEPOCHS = 50\\nHIDDEN_DIM = 256\\n\\nclass SimpleModel(nn.Module):\\n    def __init__(self, input_dim, hidden_dim, output_dim):\\n        super().__init__()\\n        self.net = nn.Sequential(\\n            nn.Linear(input_dim, hidden_dim),\\n            nn.ReLU(),\\n            nn.Dropout(0.2),\\n            nn.Linear(hidden_dim, hidden_dim),\\n            nn.ReLU(),\\n            nn.Linear(hidden_dim, output_dim),\\n        )\\n\\n    def forward(self, x):\\n        return self.net(x)\\n\\ndef train():\\n    device = torch.device(\\"cuda\\" if torch.cuda.is_available() else \\"cpu\\")\\n    print(\\"Training on \\"+str(device))\\n\\n    model = SimpleModel(768, HIDDEN_DIM, 10).to(device)\\n    optimizer = torch.optim.AdamW(model.parameters(), lr=LEARNING_RATE)\\n    criterion = nn.CrossEntropyLoss()\\n\\n    for epoch in range(EPOCHS):\\n        model.train()\\n        total_loss = 0.0\\n        # Training loop placeholder\\n        print(\\"Epoch %d/%d  loss=%.4f\\" % (epoch+1, EPOCHS, total_loss))\\n\\n    # Save model\\n    Path(\\"checkpoints\\").mkdir(exist_ok=True)\\n    torch.save(model.state_dict(), \\"checkpoints/model-v3.bin\\")\\n    print(\\"Model saved.\\")\\n\\nif __name__ == \\"__main__\\":\\n    train()',
'projects/ml-pipeline/requirements.txt':'torch>=2.2.0\\nnumpy>=1.26.0\\nscikit-learn>=1.4.0\\npandas>=2.2.0\\nmatplotlib>=3.8.0\\ntqdm>=4.66.0\\nwandb>=0.16.0',
'shared/api-spec.yaml':'openapi: 3.0.3\\ninfo:\\n  title: Acme Storage API\\n  version: 2.0.0\\n  description: File storage and sharing API\\n\\nservers:\\n  - url: https://api.acme.dev/v2\\n\\npaths:\\n  /files/{path}:\\n    get:\\n      summary: Download a file\\n      parameters:\\n        - name: path\\n          in: path\\n          required: true\\n          schema:\\n            type: string\\n      responses:\\n        200:\\n          description: File content\\n    put:\\n      summary: Upload a file\\n      requestBody:\\n        content:\\n          application/octet-stream:\\n            schema:\\n              type: string\\n              format: binary\\n      responses:\\n        201:\\n          description: File created\\n\\n  /folders/{path}:\\n    get:\\n      summary: List folder contents\\n      responses:\\n        200:\\n          description: Folder listing\\n          content:\\n            application/json:\\n              schema:\\n                type: object\\n                properties:\\n                  items:\\n                    type: array',
};

const DEMO_DOCS={
  'documents/proposal.pdf':{type:'pdf',pages:[
    '<div class="pdf-page-title">Project Proposal</div><div class="pdf-page-subtitle">Cloud Storage Platform</div><div class="pdf-page-meta">Acme Corp \\u2014 Q1 2026</div><div class="pdf-page-meta">Prepared by Engineering Team</div>',
    '<h2>Executive Summary</h2><p>This proposal outlines the architecture and implementation plan for a next-generation cloud storage platform. The system will provide secure, scalable file storage with real-time collaboration features.</p><h3>Objectives</h3><ul><li>99.99% uptime SLA</li><li>Sub-100ms API latency globally</li><li>End-to-end encryption at rest and in transit</li><li>Support for 10M+ concurrent users</li></ul><h3>Key Technologies</h3><p>The platform leverages Cloudflare Workers for edge computing, R2 for object storage, and D1 for metadata. This architecture eliminates cold starts and ensures data locality.</p>',
    '<h2>Timeline &amp; Budget</h2><table><thead><tr><th>Phase</th><th>Duration</th><th>Cost</th></tr></thead><tbody><tr><td>Discovery &amp; Design</td><td>4 weeks</td><td>$48,000</td></tr><tr><td>Core Development</td><td>12 weeks</td><td>$180,000</td></tr><tr><td>Testing &amp; QA</td><td>4 weeks</td><td>$52,000</td></tr><tr><td>Deployment</td><td>2 weeks</td><td>$24,000</td></tr></tbody></table><h3>Total: $304,000</h3><p>Payment schedule: 30% upfront, 40% at midpoint, 30% on delivery.</p>'
  ]},
  'documents/contracts/nda-acme.pdf':{type:'pdf',pages:[
    '<h2>Non-Disclosure Agreement</h2><div class="pdf-page-meta" style="text-align:left">Agreement No. NDA-2026-0042</div><p>This Non-Disclosure Agreement ("Agreement") is entered into as of January 15, 2026, by and between <strong>Acme Corporation</strong> ("Disclosing Party") and <strong>Storage Platform Inc.</strong> ("Receiving Party").</p><h3>1. Definition of Confidential Information</h3><p>For purposes of this Agreement, "Confidential Information" means any data or information that is proprietary to the Disclosing Party, including but not limited to trade secrets, technical data, product plans, customer lists, and financial information.</p><h3>2. Obligations of Receiving Party</h3><p>The Receiving Party agrees to hold and maintain the Confidential Information in strict confidence for the sole benefit of the Disclosing Party. The Receiving Party shall not, without prior written approval, disclose any Confidential Information to third parties.</p>',
    '<h3>3. Time Period</h3><p>This Agreement and the obligations herein shall be effective for a period of two (2) years from the date of execution.</p><h3>4. Return of Materials</h3><p>Upon termination of this Agreement, or upon request by the Disclosing Party, the Receiving Party shall promptly return all documents, notes, and other materials containing Confidential Information.</p><h3>5. Governing Law</h3><p>This Agreement shall be governed by and construed in accordance with the laws of the State of Delaware.</p><div style="margin-top:40px"><table><tbody><tr><td style="width:50%;border:none;padding:24px"><p><strong>Disclosing Party</strong></p><p style="border-bottom:1px solid #ccc;height:40px"></p><p>Acme Corporation</p><p>Date: _______________</p></td><td style="width:50%;border:none;padding:24px"><p><strong>Receiving Party</strong></p><p style="border-bottom:1px solid #ccc;height:40px"></p><p>Storage Platform Inc.</p><p>Date: _______________</p></td></tr></tbody></table></div>'
  ]},
  'documents/contracts/sow-2026.pdf':{type:'pdf',pages:[
    '<h2>Statement of Work</h2><div class="pdf-page-meta" style="text-align:left">SOW-2026-0018 \\u2014 Cloud Storage Platform Build</div><p>This Statement of Work ("SOW") defines the project scope, deliverables, and timeline for the Cloud Storage Platform project between Acme Corp and Storage Platform Inc.</p><h3>1. Project Scope</h3><p>The project encompasses the design, development, testing, and deployment of a cloud-native file storage platform with the following capabilities:</p><ul><li>Multi-tenant file storage with configurable quotas</li><li>Real-time file synchronization across devices</li><li>Role-based access control and sharing permissions</li><li>Full-text search indexing of document contents</li><li>RESTful API with comprehensive documentation</li></ul>',
    '<h3>2. Deliverables</h3><table><thead><tr><th>Deliverable</th><th>Description</th><th>Due Date</th></tr></thead><tbody><tr><td>Architecture Document</td><td>System design and data flow diagrams</td><td>Feb 14, 2026</td></tr><tr><td>API Specification</td><td>OpenAPI 3.0 spec with examples</td><td>Feb 28, 2026</td></tr><tr><td>MVP Release</td><td>Core upload/download/share features</td><td>Apr 30, 2026</td></tr><tr><td>Beta Release</td><td>Full feature set with integrations</td><td>Jun 15, 2026</td></tr><tr><td>Production Release</td><td>Hardened, load-tested final build</td><td>Jul 31, 2026</td></tr></tbody></table><h3>3. Acceptance Criteria</h3><p>Each deliverable will be reviewed within 5 business days. Acceptance requires sign-off from both the Project Manager and Technical Lead.</p>'
  ]},
  'shared/quarterly-review.pdf':{type:'pdf',pages:[
    '<div class="pdf-page-title">Quarterly Business Review</div><div class="pdf-page-subtitle">Q4 2025 Performance Summary</div><div class="pdf-page-meta">Prepared by a/reports-bot \\u2014 January 2026</div><h3>Key Metrics</h3><table><thead><tr><th>Metric</th><th>Q3 2025</th><th>Q4 2025</th><th>Change</th></tr></thead><tbody><tr><td>Monthly Active Users</td><td>284,000</td><td>342,000</td><td>+20.4%</td></tr><tr><td>Files Stored</td><td>18.2M</td><td>23.7M</td><td>+30.2%</td></tr><tr><td>API Requests/day</td><td>4.1M</td><td>5.8M</td><td>+41.5%</td></tr><tr><td>P99 Latency</td><td>142ms</td><td>89ms</td><td>-37.3%</td></tr></tbody></table>',
    '<h2>Key Findings</h2><h3>Growth Drivers</h3><ul><li>Enterprise plan adoption increased 45% after launching team workspaces</li><li>Developer API usage grew 3x following SDK releases for Python and Go</li><li>Mobile upload volume doubled with the new background sync feature</li></ul><h3>Areas for Improvement</h3><ul><li>Search latency exceeds 500ms for queries spanning 10M+ documents</li><li>Onboarding completion rate dropped to 62% \\u2014 needs UX review</li><li>Support ticket volume increased 28%, primarily around sharing permissions</li></ul><h3>Q1 2026 Priorities</h3><p>Focus on search performance optimization, onboarding flow redesign, and launching the Cloudflare R2 migration to reduce storage costs by an estimated 40%.</p>'
  ]},
  'documents/report-final.docx':{type:'docx',body:'<h1>Annual Report 2025</h1><div class="docx-header">Storage Platform Inc. \\u2014 Confidential</div><h2>1. Company Overview</h2><p>Storage Platform Inc. provides cloud-native file storage and collaboration tools for businesses of all sizes. Founded in 2023, the company has grown to serve over 12,000 organizations worldwide.</p><h2>2. Financial Highlights</h2><p>The fiscal year 2025 marked a significant milestone with the company achieving profitability in Q3. Key financial metrics demonstrate strong year-over-year growth across all segments.</p><table><thead><tr><th>Category</th><th>FY 2024</th><th>FY 2025</th><th>Growth</th></tr></thead><tbody><tr><td>Revenue</td><td>$8.2M</td><td>$14.6M</td><td>+78%</td></tr><tr><td>Gross Margin</td><td>68%</td><td>74%</td><td>+6pp</td></tr><tr><td>Net Income</td><td>-$1.2M</td><td>$0.8M</td><td>N/A</td></tr><tr><td>Customers</td><td>7,200</td><td>12,400</td><td>+72%</td></tr></tbody></table><h2>3. Product Development</h2><p>Major product milestones achieved during the year include:</p><ul><li>Launched real-time collaboration with conflict resolution</li><li>Released mobile apps for iOS and Android</li><li>Introduced team workspaces with granular permissions</li><li>Deployed edge caching across 200+ global PoPs</li><li>Achieved SOC 2 Type II compliance</li></ul><h2>4. Outlook</h2><p>The company is well-positioned for continued growth in 2026, with a strong product roadmap focused on AI-powered search, enhanced developer APIs, and expansion into the Asia-Pacific market.</p><div class="docx-footer">Page 1 of 1 \\u2014 Annual Report 2025</div>'},
  'documents/templates/invoice.docx':{type:'docx',body:'<div class="docx-header" style="text-align:left"><strong style="font-size:20px">INVOICE</strong><br>Storage Platform Inc.<br>123 Cloud Avenue, Suite 400<br>San Francisco, CA 94105<br>billing@storageplatform.io</div><table><tbody><tr><td style="border:none;width:50%"><strong>Bill To:</strong><br>Acme Corporation<br>456 Enterprise Blvd<br>New York, NY 10001<br>accounts@acme.dev</td><td style="border:none;text-align:right"><strong>Invoice #:</strong> INV-2026-0089<br><strong>Date:</strong> March 1, 2026<br><strong>Due Date:</strong> March 31, 2026<br><strong>Terms:</strong> Net 30</td></tr></tbody></table><table><thead><tr><th>Description</th><th style="text-align:right">Qty</th><th style="text-align:right">Rate</th><th style="text-align:right">Amount</th></tr></thead><tbody><tr><td>Enterprise Plan \\u2014 Annual License</td><td style="text-align:right">1</td><td style="text-align:right">$24,000.00</td><td style="text-align:right">$24,000.00</td></tr><tr><td>Additional Storage (500 GB)</td><td style="text-align:right">2</td><td style="text-align:right">$1,200.00</td><td style="text-align:right">$2,400.00</td></tr><tr><td>Priority Support Add-on</td><td style="text-align:right">1</td><td style="text-align:right">$3,600.00</td><td style="text-align:right">$3,600.00</td></tr><tr><td>API Rate Limit Increase</td><td style="text-align:right">1</td><td style="text-align:right">$1,800.00</td><td style="text-align:right">$1,800.00</td></tr></tbody></table><table><tbody><tr><td style="border:none;width:60%"></td><td style="text-align:right"><strong>Subtotal:</strong></td><td style="text-align:right">$31,800.00</td></tr><tr><td style="border:none"></td><td style="text-align:right"><strong>Tax (8.5%):</strong></td><td style="text-align:right">$2,703.00</td></tr><tr><td style="border:none"></td><td style="text-align:right;border-top:2px solid #111"><strong>Total Due:</strong></td><td style="text-align:right;border-top:2px solid #111"><strong>$34,503.00</strong></td></tr></tbody></table><div class="docx-footer">Thank you for your business. \\u2014 Storage Platform Inc.</div>'},
  'documents/templates/letterhead.docx':{type:'docx',body:'<div class="docx-header" style="text-align:left"><strong style="font-size:18px">Storage Platform Inc.</strong><br>123 Cloud Avenue, Suite 400 \\u2014 San Francisco, CA 94105<br>Tel: (415) 555-0192 \\u2014 hello@storageplatform.io \\u2014 storageplatform.io</div><p style="margin-top:32px">March 15, 2026</p><p>Dear Valued Partner,</p><p>We are writing to inform you of exciting updates to our platform that will be rolling out over the coming quarter. Our engineering team has been working diligently to deliver features that our customers have requested most frequently.</p><p>The upcoming release includes significant improvements to our file sharing infrastructure, including real-time collaborative editing, enhanced access controls, and a completely redesigned search experience powered by vector embeddings.</p><p>We believe these enhancements will substantially improve your team\\u2019s productivity and make Storage Platform an even more integral part of your daily workflow.</p><p>Please do not hesitate to reach out if you have any questions or would like a preview demonstration of the new features.</p><p>Best regards,</p><p><strong>Jane Chen</strong><br>VP of Product<br>Storage Platform Inc.</p><div class="docx-footer">Storage Platform Inc. \\u2014 Confidential</div>'},
  'documents/analytics.xlsx':{type:'xlsx',sheets:[
    {name:'Monthly Traffic',headers:['Date','Visitors','Pageviews','Bounce Rate','Avg. Session'],rows:[
      ['2026-01-01','142,300','489,200','34.2%','4m 12s'],
      ['2026-01-08','156,800','534,100','32.8%','4m 38s'],
      ['2026-01-15','148,900','501,400','33.5%','4m 21s'],
      ['2026-01-22','163,200','562,800','31.9%','4m 45s'],
      ['2026-02-01','171,400','598,300','30.1%','5m 02s'],
      ['2026-02-08','168,900','584,600','30.8%','4m 55s'],
      ['2026-02-15','179,300','621,400','29.4%','5m 11s'],
      ['2026-02-22','185,100','645,800','28.7%','5m 22s'],
      ['2026-03-01','192,600','672,100','27.9%','5m 34s'],
      ['2026-03-08','201,400','708,900','26.5%','5m 48s']
    ]},
    {name:'Revenue',headers:['Month','MRR','ARR','Churn Rate'],rows:[
      ['Jan 2026','$1,218,000','$14,616,000','1.8%'],
      ['Feb 2026','$1,284,000','$15,408,000','1.6%'],
      ['Mar 2026','$1,342,000','$16,104,000','1.5%']
    ]}
  ]},
  'documents/slides-keynote.pptx':{type:'pptx',slides:[
    {title:'Cloud Storage Platform',body:'<p style="font-size:18px;color:#666">Annual Technology Conference 2026</p><p style="margin-top:24px;color:#888">Storage Platform Inc. \\u2014 Engineering Division</p>'},
    {title:'Agenda',body:'<ul><li>Platform overview &amp; key metrics</li><li>Architecture deep-dive</li><li>Performance benchmarks</li><li>Roadmap &amp; upcoming features</li><li>Q&amp;A</li></ul>'},
    {title:'Key Metrics',body:'<div style="display:flex;gap:32px;flex-wrap:wrap;justify-content:center;margin:16px 0"><div class="metric"><span class="metric-value">342K</span><span class="metric-label">Monthly Active Users</span></div><div class="metric"><span class="metric-value">23.7M</span><span class="metric-label">Files Stored</span></div><div class="metric"><span class="metric-value">89ms</span><span class="metric-label">P99 Latency</span></div><div class="metric"><span class="metric-value">99.99%</span><span class="metric-label">Uptime SLA</span></div></div>'},
    {title:'Architecture Overview',body:'<p>The platform is built on a globally distributed edge architecture:</p><ul><li><strong>Edge Layer</strong> \\u2014 Cloudflare Workers handle routing, auth, and caching at 300+ PoPs</li><li><strong>Storage Layer</strong> \\u2014 R2 object storage with automatic replication and versioning</li><li><strong>Metadata Layer</strong> \\u2014 D1 SQLite databases for fast, consistent metadata queries</li><li><strong>Search Layer</strong> \\u2014 Vectorize indexes for semantic and full-text search</li></ul><p style="margin-top:12px;color:#888;font-size:13px">All components are serverless with zero cold starts and automatic scaling.</p>'},
    {title:'Thank You',body:'<p style="font-size:18px;color:#666;text-align:center">Questions &amp; Discussion</p><p style="text-align:center;margin-top:20px;color:#888">engineering@storageplatform.io</p>'}
  ]}
};

/* ── Icons (same as authenticated app) ───────────────────────────── */
const I={
  folder:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M22 19a2 2 0 01-2 2H4a2 2 0 01-2-2V5a2 2 0 012-2h5l2 3h9a2 2 0 012 2z"/></svg>',
  file:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>',
  image:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="3" y="3" width="18" height="18" rx="2"/><circle cx="8.5" cy="8.5" r="1.5"/><polyline points="21 15 16 10 5 21"/></svg>',
  video:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polygon points="23 7 16 12 23 17 23 7"/><rect x="1" y="5" width="15" height="14" rx="2"/></svg>',
  audio:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M9 18V5l12-2v13"/><circle cx="6" cy="18" r="3"/><circle cx="18" cy="16" r="3"/></svg>',
  doc:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/><polyline points="10 9 9 9 8 9"/></svg>',
  code:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/></svg>',
  archive:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polyline points="21 8 21 21 3 21 3 8"/><rect x="1" y="3" width="22" height="5"/><line x1="10" y1="12" x2="14" y2="12"/></svg>',
  sheet:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="3" y="3" width="18" height="18" rx="2"/><line x1="3" y1="9" x2="21" y2="9"/><line x1="3" y1="15" x2="21" y2="15"/><line x1="9" y1="3" x2="9" y2="21"/></svg>',
  text:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/></svg>',
  star:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/></svg>',
  starFill:'<svg viewBox="0 0 24 24" fill="currentColor" stroke="currentColor" stroke-width="1.5"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/></svg>',
  trash:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/></svg>',
  download:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>',
  upload:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/></svg>',
  share:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="18" cy="5" r="3"/><circle cx="6" cy="12" r="3"/><circle cx="18" cy="19" r="3"/><line x1="8.59" y1="13.51" x2="15.42" y2="17.49"/><line x1="15.41" y1="6.51" x2="8.59" y2="10.49"/></svg>',
  home:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M3 9l9-7 9 7v11a2 2 0 01-2 2H5a2 2 0 01-2-2z"/></svg>',
  shared:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M17 21v-2a4 4 0 00-4-4H5a4 4 0 00-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 00-3-3.87"/><path d="M16 3.13a4 4 0 010 7.75"/></svg>',
  clock:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>',
  grid:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/><rect x="14" y="14" width="7" height="7"/></svg>',
  list:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="8" y1="6" x2="21" y2="6"/><line x1="8" y1="12" x2="21" y2="12"/><line x1="8" y1="18" x2="21" y2="18"/><line x1="3" y1="6" x2="3.01" y2="6"/><line x1="3" y1="12" x2="3.01" y2="12"/><line x1="3" y1="18" x2="3.01" y2="18"/></svg>',
  plus:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>',
  restore:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polyline points="1 4 1 10 7 10"/><path d="M3.51 15a9 9 0 102.13-9.36L1 10"/></svg>',
  info:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg>',
  x:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>',
  link:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M10 13a5 5 0 007.54.54l3-3a5 5 0 00-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 00-7.54-.54l-3 3a5 5 0 007.07 7.07l1.71-1.71"/></svg>',
};

/* ── State ────────────────────────────────────────────────────────── */
const S={
  path:'',view:localStorage.getItem('sv')||'list',
  sort:{col:localStorage.getItem('sc')||'name',asc:localStorage.getItem('sa')!=='0'},
  items:[],selected:new Set(),section:'files',
  detail:null,detailOpen:false,detailTab:'details',
  searchMode:false,searchQ:'',
  previewItem:null,previewIdx:-1,
};

const $=id=>document.getElementById(id);

/* ── Helpers ──────────────────────────────────────────────────────── */
function h(s){return (s||'').replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;')}
function fmtSize(b){if(!b)return '—';if(b<1024)return b+' B';if(b<1048576)return (b/1024).toFixed(1)+' KB';if(b<1073741824)return (b/1048576).toFixed(1)+' MB';return (b/1073741824).toFixed(1)+' GB'}
function fmtTime(ts){if(!ts)return '—';const d=Date.now()-ts,s=d/1000;if(s<60)return 'just now';if(s<3600)return Math.floor(s/60)+'m ago';if(s<86400)return Math.floor(s/3600)+'h ago';if(s<2592000)return Math.floor(s/86400)+'d ago';return new Date(ts).toLocaleDateString()}

function fileType(item){
  if(item.is_folder)return 'folder';
  const ct=item.content_type||'';const n=item.name||'';
  if(ct.startsWith('image/'))return 'image';
  if(ct.startsWith('video/'))return 'video';
  if(ct.startsWith('audio/'))return 'audio';
  if(ct==='application/pdf'||ct.includes('document')||ct.includes('msword'))return 'doc';
  if(ct.includes('spreadsheet')||ct.includes('csv')||ct==='text/csv')return 'sheet';
  if(ct.includes('zip')||ct.includes('tar')||ct.includes('gzip')||ct.includes('compressed'))return 'archive';
  const ext=n.split('.').pop()?.toLowerCase()||'';
  if(['js','ts','py','go','rs','java','c','cpp','h','rb','php','sh','bash','yaml','yml','toml','json','xml','html','css','sql','md'].includes(ext))return 'code';
  if(['txt','log','text'].includes(ext)||ct.startsWith('text/'))return 'text';
  return 'generic';
}
function fileIconHtml(item){const t=fileType(item);return '<div class="file-icon file-icon--'+t+'">'+I[t==='generic'?'file':t]+'</div>'}

/* ── Language detection ─────────────────────────────────────────── */
function langFromName(name){
  const ext=(name||'').split('.').pop()?.toLowerCase()||'';
  const map={go:'go',py:'py',js:'js',ts:'js',jsx:'js',tsx:'js',css:'css',html:'html',htm:'html',yaml:'yaml',yml:'yaml',toml:'toml',json:'json',sql:'sql',sh:'sh',bash:'sh',rs:'rs',java:'java',c:'c',cpp:'cpp',h:'c',rb:'rb',php:'php',xml:'html',md:'md',dockerfile:'sh',Dockerfile:'sh'};
  return map[ext]||'text';
}

/* ── Syntax highlighter ────────────────────────────────────────── */
function highlightCode(code,lang){
  const lines=code.split('\\n');
  return lines.map(raw=>{
    let ln=h(raw);
    // Strings (double and single quoted)
    ln=ln.replace(/(&quot;(?:[^&]|&(?!quot;))*?&quot;)/g,'<span class="tok-str">$1</span>');
    ln=ln.replace(/(&#x27;(?:[^&]|&(?!#x27;))*?&#x27;)/g,'<span class="tok-str">$1</span>');
    ln=ln.replace(/(\\x60[^\\x60]*\\x60)/g,'<span class="tok-str">$1</span>');
    // Comments
    if(['go','js','java','c','cpp','rs','css'].includes(lang)){
      ln=ln.replace(/(^|\\s)(\\/{2}.*)$/,'$1<span class="tok-cm">$2</span>');
    }
    if(['py','yaml','toml','sh','dockerfile'].includes(lang)){
      ln=ln.replace(/(^|\\s)(#.*)$/,'$1<span class="tok-cm">$2</span>');
    }
    if(lang==='html'){
      ln=ln.replace(/(&lt;!--.*?--&gt;)/g,'<span class="tok-cm">$1</span>');
    }
    // Numbers
    ln=ln.replace(/\\b(\\d+\\.?\\d*)\\b/g,'<span class="tok-num">$1</span>');
    // Keywords per language
    const kw={
      go:'package|import|func|var|const|type|struct|interface|return|if|else|for|range|switch|case|default|defer|go|chan|select|map|make|new|nil|true|false|break|continue|error|string|int|bool|byte|fmt',
      py:'import|from|def|class|return|if|elif|else|for|while|in|not|and|or|is|None|True|False|with|as|try|except|finally|raise|pass|break|continue|lambda|yield|async|await|self|print',
      js:'import|export|from|const|let|var|function|return|if|else|for|while|do|switch|case|default|break|continue|new|this|class|extends|async|await|try|catch|finally|throw|typeof|instanceof|null|undefined|true|false|of|in|console|require|module',
      css:'@media|@keyframes|@import|@font-face|@layer|@supports|@property|:root|:hover|:focus|:active|:visited|!important|inherit|initial|unset',
      html:'DOCTYPE|html|head|body|meta|link|title|script|style|div|span|a|p|h1|h2|h3|h4|img|ul|ol|li|table|tr|td|th|form|input|button|nav|main|aside|section|article|header|footer',
      sh:'if|then|else|fi|for|do|done|while|case|esac|function|return|exit|echo|export|source|cd|mkdir|rm|cp|mv|chmod|chown|grep|awk|sed|cat|FROM|RUN|CMD|COPY|WORKDIR|EXPOSE|ENV|ARG|ENTRYPOINT',
      sql:'SELECT|FROM|WHERE|AND|OR|NOT|INSERT|INTO|VALUES|UPDATE|SET|DELETE|CREATE|TABLE|DROP|ALTER|INDEX|JOIN|LEFT|RIGHT|INNER|OUTER|ON|GROUP|BY|ORDER|ASC|DESC|LIMIT|OFFSET|IF|EXISTS|PRIMARY|KEY|UNIQUE|NOT|NULL|DEFAULT|INTEGER|TEXT|REAL|BLOB',
      yaml:'true|false|null|yes|no',
      json:'true|false|null',
    };
    const kwList=kw[lang];
    if(kwList){
      ln=ln.replace(new RegExp('\\\\b('+kwList+')\\\\b','g'),'<span class="tok-kw">$1</span>');
    }
    return '<span class="line">'+ln+'</span>';
  }).join('\\n');
}

/* ── Markdown renderer (basic) ─────────────────────────────────── */
function renderMarkdown(md){
  let html=h(md);
  // Code blocks
  html=html.replace(/\\x60\\x60\\x60(\\w*)\\n([\\s\\S]*?)\\x60\\x60\\x60/g,(m,lang,code)=>'<pre><code>'+code+'</code></pre>');
  // Inline code
  html=html.replace(/\\x60([^\\x60]+)\\x60/g,'<code>$1</code>');
  // Headers
  html=html.replace(/^#### (.+)$/gm,'<h4>$1</h4>');
  html=html.replace(/^### (.+)$/gm,'<h3>$1</h3>');
  html=html.replace(/^## (.+)$/gm,'<h2>$1</h2>');
  html=html.replace(/^# (.+)$/gm,'<h1>$1</h1>');
  // Bold and italic
  html=html.replace(/\\*\\*(.+?)\\*\\*/g,'<strong>$1</strong>');
  html=html.replace(/\\*(.+?)\\*/g,'<em>$1</em>');
  // Links
  html=html.replace(/\\[([^\\]]+)\\]\\(([^)]+)\\)/g,'<a href="$2">$1</a>');
  // Horizontal rule
  html=html.replace(/^---$/gm,'<hr>');
  // Unordered list items
  html=html.replace(/^[\\-\\*] (.+)$/gm,'<li>$1</li>');
  // Blockquotes
  html=html.replace(/^&gt; (.+)$/gm,'<blockquote>$1</blockquote>');
  // Paragraphs (wrap loose lines)
  html=html.replace(/^(?!<[hluobpc]|<\\/|<hr|<li|<block|<pre|<code)(.+)$/gm,'<p>$1</p>');
  // Wrap consecutive li in ul
  html=html.replace(/(<li>.*?<\\/li>\\n?)+/g,'<ul>$&</ul>');
  return html;
}

/* ── CSV to table ──────────────────────────────────────────────── */
function csvToTable(csv){
  const rows=csv.trim().split('\\n').map(r=>r.split(',').map(c=>c.trim()));
  if(!rows.length)return '';
  let t='<table><thead><tr>'+rows[0].map(c=>'<th>'+h(c)+'</th>').join('')+'</tr></thead><tbody>';
  for(let i=1;i<rows.length;i++){
    t+='<tr>'+rows[i].map(c=>'<td>'+h(c)+'</td>').join('')+'</tr>';
  }
  return t+'</tbody></table>';
}

/* ── Document preview renderers ────────────────────────────────── */
function renderPdfPages(doc,item){
  let html='<div class="preview-pdf">';
  doc.pages.forEach((pg,i)=>{
    html+='<div class="pdf-page"><div class="pdf-page-content">'+pg+'</div><div class="pdf-page-num">Page '+(i+1)+' of '+doc.pages.length+'</div></div>';
  });
  html+='</div>';
  return html;
}

function renderDocxPage(doc,item){
  return '<div class="preview-docx"><div class="docx-ruler"><span>1</span><span>2</span><span>3</span><span>4</span><span>5</span><span>6</span><span>7</span></div><div class="docx-page">'+doc.body+'</div></div>';
}

function renderXlsxSheet(doc,item){
  const sheet=doc.sheets[0];
  const colLetters='ABCDEFGHIJKLMNOPQRSTUVWXYZ';
  let html='<div class="preview-xlsx">';
  html+='<div class="xlsx-tabs">';
  doc.sheets.forEach((s,i)=>{html+='<div class="xlsx-tab'+(i===0?' active':'')+'">'+h(s.name)+'</div>'});
  html+='</div>';
  html+='<div class="xlsx-grid"><table><thead><tr><th class="xlsx-row-num"></th>';
  for(let c=0;c<sheet.headers.length;c++){html+='<th>'+colLetters[c]+'</th>'}
  html+='</tr><tr class="xlsx-header-row"><th class="xlsx-row-num">1</th>';
  sheet.headers.forEach(hdr=>{html+='<td class="xlsx-header-cell">'+h(hdr)+'</td>'});
  html+='</tr></thead><tbody>';
  sheet.rows.forEach((row,r)=>{
    html+='<tr><th class="xlsx-row-num">'+(r+2)+'</th>';
    row.forEach(cell=>{html+='<td>'+h(String(cell))+'</td>'});
    html+='</tr>';
  });
  html+='</tbody></table></div></div>';
  return html;
}

function renderPptxSlides(doc,item){
  let html='<div class="preview-pptx">';
  html+='<div class="pptx-thumbs">';
  doc.slides.forEach((s,i)=>{html+='<div class="pptx-thumb'+(i===0?' active':'')+'" onclick="B.showSlide('+i+')"><div class="pptx-thumb-num">'+(i+1)+'</div><div class="pptx-thumb-title">'+h(s.title)+'</div></div>'});
  html+='</div>';
  html+='<div class="pptx-stage" id="pptx-stage">';
  doc.slides.forEach((s,i)=>{
    html+='<div class="pptx-slide'+(i===0?' active':'')+'" data-slide="'+i+'"><div class="pptx-slide-title">'+h(s.title)+'</div><div class="pptx-slide-body">'+s.body+'</div></div>';
  });
  html+='</div></div>';
  return html;
}

/* ── Generate waveform bars HTML ───────────────────────────────── */
function waveformBars(){
  let bars='';
  for(let i=0;i<60;i++){
    const h=8+Math.floor(Math.random()*32);
    bars+='<div class="bar" style="height:'+h+'px"></div>';
  }
  return bars;
}

/* ── Toast ────────────────────────────────────────────────────────── */
function toast(msg,type='info'){
  const el=document.createElement('div');
  el.className='toast toast--'+type;
  el.innerHTML='<span class="toast-msg">'+h(msg)+'</span>';
  $('toasts').appendChild(el);
  setTimeout(()=>{el.classList.add('fade-out');setTimeout(()=>el.remove(),300)},3000);
}

/* ── Sign-up prompt for write actions ────────────────────────────── */
function requireSignup(action){
  showModal(
    'Sign up to '+action,
    '<div style="text-align:center;padding:16px 0"><div style="font-size:48px;margin-bottom:16px">🔒</div><p style="color:var(--text-2);margin-bottom:20px">This is a read-only demo.<br>Create a free account to '+action+'.</p><a href="/" class="tool-btn tool-btn--primary" style="display:inline-flex;padding:10px 24px;font-size:14px;text-decoration:none">Sign up free</a></div>',
    ''
  );
}

/* ── Data loading (from static demo filesystem) ──────────────────── */
function loadItems(){
  S.selected.clear();
  const prefix=S.path;
  if(S.searchMode){
    const q=S.searchQ.toLowerCase();
    S.items=DEMO_FS.filter(f=>!f.trashed_at&&f.name.toLowerCase().includes(q));
  } else if(S.section==='files'){
    S.items=DEMO_FS.filter(f=>{
      if(f.trashed_at)return false;
      if(!f.path.startsWith(prefix))return false;
      const rest=f.path.slice(prefix.length);
      if(f.is_folder){return rest.replace(/\\/$/,'').indexOf('/')===-1&&rest!==''}
      return rest.indexOf('/')===-1&&rest!=='';
    });
  } else if(S.section==='shared'){
    S.items=DEMO_SHARED.map(i=>({...i}));
  } else if(S.section==='recent'){
    S.items=DEMO_FS.filter(f=>!f.trashed_at&&!f.is_folder).sort((a,b)=>b.updated_at-a.updated_at).slice(0,10);
  } else if(S.section==='starred'){
    S.items=DEMO_FS.filter(f=>!f.trashed_at&&f.starred);
  } else if(S.section==='trash'){
    S.items=DEMO_FS.filter(f=>f.trashed_at);
  }
  sortItems();
  render();
}

function sortItems(){
  const {col,asc}=S.sort;
  S.items.sort((a,b)=>{
    if(a.is_folder!==b.is_folder)return b.is_folder?1:-1;
    let v=0;
    if(col==='name')v=a.name.localeCompare(b.name);
    else if(col==='updated_at')v=(a.updated_at||0)-(b.updated_at||0);
    else if(col==='size')v=(a.size||0)-(b.size||0);
    return asc?v:-v;
  });
}

/* ── Render: Sidebar ─────────────────────────────────────────────── */
function renderSidebar(){
  const items=[
    {id:'files',icon:I.folder,label:'My Files'},
    {id:'shared',icon:I.shared,label:'Shared with me'},
    {id:'recent',icon:I.clock,label:'Recent'},
    {id:'starred',icon:I.star,label:'Starred'},
    {id:'trash',icon:I.trash,label:'Trash',badge:DEMO_STATS.trash_count},
  ];
  let html='';
  for(const it of items){
    const active=S.section===it.id?' active':'';
    const badge=it.badge?'<span class="item-badge">'+it.badge+'</span>':'';
    html+='<div class="sidebar-item'+active+'" data-section="'+it.id+'"><span class="item-icon">'+it.icon+'</span><span class="item-label">'+it.label+'</span>'+badge+'</div>';
  }
  $('sidebar-nav').innerHTML=html;
}

function renderQuota(){
  const pct=Math.min(100,DEMO_STATS.total_size/DEMO_STATS.quota*100);
  const cls=pct>80?'danger':pct>50?'warn':'ok';
  $('quota').innerHTML='<div class="quota-bar"><div class="quota-fill quota-fill--'+cls+'" style="width:'+pct.toFixed(1)+'%"></div></div><div class="quota-text">'+fmtSize(DEMO_STATS.total_size)+' of '+fmtSize(DEMO_STATS.quota)+' used</div>';
}

/* ── Render: Toolbar ─────────────────────────────────────────────── */
function renderToolbar(){
  const parts=S.path.replace(/\\/$/,'').split('/').filter(Boolean);
  let bc='<span class="breadcrumb-home" onclick="B.nav(\\'\\')">' + I.home.replace('viewBox','width="16" height="16" viewBox') + '</span>';
  let acc='';
  for(const p of parts){
    acc+=p+'/';
    const a=acc;
    bc+='<span class="breadcrumb-sep">/</span><span class="breadcrumb-segment" onclick="B.nav(\\''+h(a)+'\\')">'+h(p)+'</span>';
  }
  if(S.searchMode)bc='<span class="breadcrumb-segment">Search results for "'+h(S.searchQ)+'"</span>';

  const lActive=S.view==='list'?' active':'';
  const gActive=S.view==='grid'?' active':'';
  const sectionIsTrash=S.section==='trash';

  let right='';
  if(!S.searchMode&&S.section==='files'){
    right='<div class="toolbar-group"><button class="tool-btn'+lActive+'" onclick="B.setView(\\'list\\')" title="List view">'+I.list+'</button><button class="tool-btn'+gActive+'" onclick="B.setView(\\'grid\\')" title="Grid view">'+I.grid+'</button></div>';
    right+='<div class="toolbar-divider"></div>';
    right+='<div class="sort-dropdown" id="sort-dd"><button class="tool-btn" onclick="B.toggleSort()">Sort</button><div class="sort-menu" id="sort-menu"><div class="sort-option'+(S.sort.col==='name'?' active':'')+'" onclick="B.setSort(\\'name\\')">Name <span class="sort-dir">'+(S.sort.col==='name'?(S.sort.asc?'A→Z':'Z→A'):'')+'</span></div><div class="sort-option'+(S.sort.col==='updated_at'?' active':'')+'" onclick="B.setSort(\\'updated_at\\')">Modified <span class="sort-dir">'+(S.sort.col==='updated_at'?(S.sort.asc?'Old→New':'New→Old'):'')+'</span></div><div class="sort-option'+(S.sort.col==='size'?' active':'')+'" onclick="B.setSort(\\'size\\')">Size <span class="sort-dir">'+(S.sort.col==='size'?(S.sort.asc?'Small→Big':'Big→Small'):'')+'</span></div></div></div>';
    right+='<div class="toolbar-divider"></div>';
    right+='<button class="tool-btn" onclick="requireSignup(\\'create folders\\')" title="New folder">'+I.plus+' Folder</button>';
    right+='<button class="tool-btn tool-btn--primary" onclick="requireSignup(\\'upload files\\')" title="Upload">'+I.upload+' Upload</button>';
  }
  if(sectionIsTrash){
    right='<button class="tool-btn bulk-btn--danger" onclick="requireSignup(\\'manage trash\\')">'+I.trash+' Empty trash</button>';
  }

  $('toolbar').innerHTML='<div class="breadcrumb">'+bc+'</div>'+right;
}

/* ── Render: Bulk bar ────────────────────────────────────────────── */
function renderBulk(){
  const n=S.selected.size;
  const el=$('bulk-bar');
  if(!n){el.className='';el.innerHTML='';return}
  el.className='visible';
  el.innerHTML='<span class="bulk-count">'+n+' selected</span><div class="bulk-actions"><button class="bulk-btn" onclick="requireSignup(\\'download files\\')">'+I.download+' Download</button><button class="bulk-btn" onclick="requireSignup(\\'star files\\')">'+I.star+' Star</button><button class="bulk-btn bulk-btn--danger" onclick="requireSignup(\\'move files to trash\\')">'+I.trash+' Trash</button></div><button class="bulk-dismiss" onclick="B.clearSel()">'+I.x+'</button>';
}

/* ── Render: File list ───────────────────────────────────────────── */
function renderItems(){
  const fc=$('file-content');
  if(!S.items.length){
    const msgs={files:'This folder is empty',shared:'No files shared with you yet',recent:'No recently accessed files',starred:'No starred files',trash:'Trash is empty'};
    fc.innerHTML='<div class="empty-state"><div class="empty-icon"><svg width="56" height="56" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1"><path d="M22 19a2 2 0 01-2 2H4a2 2 0 01-2-2V5a2 2 0 012-2h5l2 3h9a2 2 0 012 2z"/></svg></div><div class="empty-title">'+(msgs[S.section]||'No files')+'</div></div>';
    return;
  }
  if(S.view==='grid')renderGrid(fc);
  else renderList(fc);
}

function renderList(fc){
  const allSel=S.items.length>0&&S.items.every(i=>S.selected.has(i.path));
  const isShared=S.section==='shared';
  let html='<div id="file-list"><div class="list-header"><input type="checkbox" class="row-check col--check"'+(allSel?' checked':'')+' onchange="B.selectAll(this.checked)"><span class="col col--icon"></span><span class="col col--name" onclick="B.setSort(\\'name\\')">Name'+(S.sort.col==='name'?'<span class="sort-arrow">'+(S.sort.asc?' ↑':' ↓')+'</span>':'')+'</span>'+(isShared?'<span class="col col--owner">Owner</span><span class="col col--perm">Access</span>':'<span class="col col--modified" onclick="B.setSort(\\'updated_at\\')">Modified'+(S.sort.col==='updated_at'?'<span class="sort-arrow">'+(S.sort.asc?' ↑':' ↓')+'</span>':'')+'</span><span class="col col--size" onclick="B.setSort(\\'size\\')">Size'+(S.sort.col==='size'?'<span class="sort-arrow">'+(S.sort.asc?' ↑':' ↓')+'</span>':'')+'</span>')+'</div>';
  for(const item of S.items){
    const sel=S.selected.has(item.path)?' selected':'';
    const isF=item.is_folder;
    const starCls=item.starred?' starred':'';
    html+='<div class="file-row'+(isF?' file-row--folder':'')+sel+'" data-path="'+h(item.path)+'" oncontextmenu="B.ctx(event,\\''+h(item.path)+'\\')"><input type="checkbox" class="row-check col--check"'+(sel?' checked':'')+' onclick="event.stopPropagation()" onchange="B.toggleSel(\\''+h(item.path)+'\\',event)">'+fileIconHtml(item)+'<span class="file-name"><span class="file-name-text">'+h(item.name)+(isF?'/':'')+'</span>'+(isShared?'':'<button class="file-star'+starCls+'" onclick="event.stopPropagation();requireSignup(\\'star files\\')">'+(item.starred?I.starFill:I.star)+'</button>')+'</span>';
    if(isShared){
      html+='<span class="file-modified" style="font-family:JetBrains Mono,monospace;font-size:12px;opacity:.7">'+h(item.owner||'')+'</span>';
      html+='<span class="file-size"><span class="perm-badge perm-badge--'+h(item.permission||'viewer')+'">'+h(item.permission||'viewer')+'</span></span>';
    } else {
      html+='<span class="file-modified">'+fmtTime(item.updated_at)+'</span>';
      html+='<span class="file-size">'+fmtSize(item.is_folder?0:item.size)+'</span>';
    }
    html+='</div>';
  }
  html+='</div>';
  fc.innerHTML=html;
  fc.querySelectorAll('.file-row').forEach(el=>{
    el.addEventListener('click',e=>{
      if(e.target.closest('.row-check')||e.target.closest('.file-star')||e.target.closest('.grid-star'))return;
      const p=el.dataset.path;
      if(e.ctrlKey||e.metaKey||e.shiftKey){B.clickSel(p,e);return}
      const item=S.items.find(i=>i.path===p);
      if(item?.is_folder)B.nav(p);
      else if(item)B.openPreview(p);
    });
  });
}

function renderGrid(fc){
  let html='<div id="file-grid">';
  for(const item of S.items){
    const sel=S.selected.has(item.path)?' selected':'';
    const isF=item.is_folder;
    const t=fileType(item);
    const starCls=item.starred?' starred':'';
    html+='<div class="grid-card'+(isF?' grid-card--folder':'')+sel+'" data-path="'+h(item.path)+'" oncontextmenu="B.ctx(event,\\''+h(item.path)+'\\')"><div class="grid-check"><input type="checkbox" class="row-check"'+(sel?' checked':'')+' onclick="event.stopPropagation()" onchange="B.toggleSel(\\''+h(item.path)+'\\',event)"></div><button class="grid-star'+starCls+'" onclick="event.stopPropagation();requireSignup(\\'star files\\')">'+(item.starred?I.starFill:I.star)+'</button><div class="grid-thumb"><div class="file-icon file-icon--'+t+'">'+I[t==='generic'?'file':t]+'</div></div><div class="grid-name">'+h(item.name)+(isF?'/':'')+'</div></div>';
  }
  html+='</div>';
  fc.innerHTML=html;
  fc.querySelectorAll('.grid-card').forEach(el=>{
    el.addEventListener('click',e=>{
      if(e.target.closest('.row-check')||e.target.closest('.file-star')||e.target.closest('.grid-star'))return;
      const p=el.dataset.path;
      if(e.ctrlKey||e.metaKey||e.shiftKey){B.clickSel(p,e);return}
      const item=S.items.find(i=>i.path===p);
      if(item?.is_folder)B.nav(p);
      else if(item)B.openPreview(p);
    });
  });
}

/* ── Render: Detail panel ────────────────────────────────────────── */
function renderDetail(){
  const dp=$('detail-panel');
  if(!S.detailOpen||!S.detail){dp.className='hidden';return}
  dp.className='';
  const it=S.detail;
  const t=fileType(it);
  const detTab=S.detailTab==='details'?' active':'';
  const actTab=S.detailTab==='activity'?' active':'';
  let body='';
  if(S.detailTab==='details'){
    body='<div class="detail-icon file-icon--'+t+'">'+I[t==='generic'?'file':t]+'</div>';
    body+='<div class="detail-filename">'+h(it.name)+'</div>';
    body+='<div class="detail-type">'+(it.is_folder?'Folder':h(it.content_type||'Unknown'))+'</div>';
    if(!it.is_folder)body+='<div class="detail-field"><div class="detail-label">Size</div><div class="detail-value">'+fmtSize(it.size)+'</div></div>';
    body+='<div class="detail-field"><div class="detail-label">Location</div><div class="detail-value">'+h(it.path)+'</div></div>';
    body+='<div class="detail-field"><div class="detail-label">Created</div><div class="detail-value">'+fmtTime(it.created_at)+'</div></div>';
    body+='<div class="detail-field"><div class="detail-label">Modified</div><div class="detail-value">'+fmtTime(it.updated_at)+'</div></div>';
    if(it.description)body+='<div class="detail-field"><div class="detail-label">Description</div><div class="detail-value" style="color:var(--text-2)">'+h(it.description)+'</div></div>';
    if(it.owner)body+='<div class="detail-field"><div class="detail-label">Owner</div><div class="detail-value" style="font-family:JetBrains Mono,monospace;font-size:12px">'+h(it.owner)+'</div></div>';
    if(it.permission)body+='<div class="detail-field"><div class="detail-label">Access</div><div class="detail-value"><span class="perm-badge perm-badge--'+h(it.permission)+'">'+h(it.permission)+'</span></div></div>';
  } else {
    body='<div class="activity-list"><div class="activity-item"><div class="activity-avatar">D</div><div class="activity-content"><div class="activity-text"><strong>demo</strong> uploaded this file</div><div class="activity-time">'+fmtTime(it.created_at)+'</div></div></div>';
    if(it.updated_at!==it.created_at)body+='<div class="activity-item"><div class="activity-avatar">D</div><div class="activity-content"><div class="activity-text"><strong>demo</strong> modified this file</div><div class="activity-time">'+fmtTime(it.updated_at)+'</div></div></div>';
    body+='</div>';
  }
  dp.innerHTML='<div class="detail-header"><div class="detail-tabs"><span class="detail-tab'+detTab+'" onclick="B.detailTab(\\'details\\')">Details</span><span class="detail-tab'+actTab+'" onclick="B.detailTab(\\'activity\\')">Activity</span></div><button class="detail-close" onclick="B.closeDetail()">'+I.x+'</button></div><div class="detail-body">'+body+'</div>';
}

/* ── Render: Context menu ────────────────────────────────────────── */
function renderCtx(x,y,item){
  const m=$('ctx-menu');
  let html='';
  if(!item){
    html+='<div class="ctx-item" onclick="requireSignup(\\'create folders\\')">'+I.plus+' New folder</div>';
    html+='<div class="ctx-item" onclick="requireSignup(\\'upload files\\')">'+I.upload+' Upload files</div>';
  } else if(S.section==='trash'){
    html+='<div class="ctx-item" onclick="requireSignup(\\'restore files\\')">'+I.restore+' Restore</div>';
  } else {
    if(item.is_folder)html+='<div class="ctx-item" onclick="hideCtx();B.nav(\\''+h(item.path)+'\\')">'+I.folder+' Open</div>';
    else html+='<div class="ctx-item" onclick="hideCtx();requireSignup(\\'download files\\')">'+I.download+' Download</div>';
    html+='<div class="ctx-divider"></div>';
    html+='<div class="ctx-item" onclick="hideCtx();B.showDetail(\\''+h(item.path)+'\\')">'+I.info+' Details</div>';
    html+='<div class="ctx-divider"></div>';
    html+='<div class="ctx-item" style="opacity:.5" onclick="hideCtx();requireSignup(\\'share files\\')">'+I.share+' Share</div>';
    html+='<div class="ctx-item" style="opacity:.5" onclick="hideCtx();requireSignup(\\'manage files\\')">'+I.trash+' Move to trash</div>';
  }
  const vw=window.innerWidth,vh=window.innerHeight;
  m.innerHTML=html;
  m.style.left=Math.min(x,vw-200)+'px';
  m.style.top=Math.min(y,vh-m.scrollHeight-10)+'px';
  m.classList.add('visible');
}

/* ── Render: Preview overlay ───────────────────────────────────── */
function renderPreview(){
  const el=$('preview-overlay');
  if(!S.previewItem){el.classList.remove('visible');el.innerHTML='';return}
  el.classList.add('visible');
  const item=S.previewItem;
  const t=fileType(item);
  const files=S.items.filter(i=>!i.is_folder);
  const idx=files.findIndex(i=>i.path===item.path);
  const hasPrev=idx>0;
  const hasNext=idx<files.length-1;
  const content=DEMO_CONTENT[item.path];
  const lang=langFromName(item.name);
  const isMd=item.name.endsWith('.md')||item.content_type==='text/markdown';

  let body='';
  if(isMd&&content){
    body='<div class="preview-markdown">'+renderMarkdown(content)+'</div>';
  } else if(t==='code'&&content){
    body='<pre class="preview-code"><code>'+highlightCode(content,lang)+'</code></pre>';
  } else if(t==='text'&&content){
    body='<pre class="preview-text">'+h(content)+'</pre>';
  } else if(t==='sheet'&&content){
    body='<div class="preview-table">'+csvToTable(content)+'</div>';
  } else if(t==='image'){
    const colors=['#667eea,#764ba2','#f093fb,#f5576c','#4facfe,#00f2fe','#43e97b,#38f9d7','#fa709a,#fee140'];
    const c=colors[Math.abs(item.name.length)%colors.length];
    const svg='<svg xmlns="http://www.w3.org/2000/svg" width="800" height="500"><defs><linearGradient id="g" x1="0%" y1="0%" x2="100%" y2="100%"><stop offset="0%" stop-color="'+c.split(',')[0]+'"/><stop offset="100%" stop-color="'+c.split(',')[1]+'"/></linearGradient></defs><rect width="800" height="500" fill="url(#g)"/><text x="400" y="240" text-anchor="middle" fill="rgba(255,255,255,.85)" font-family="Inter,sans-serif" font-size="24" font-weight="600">'+h(item.name)+'</text><text x="400" y="275" text-anchor="middle" fill="rgba(255,255,255,.5)" font-family="Inter,sans-serif" font-size="14">'+fmtSize(item.size)+' \\u2014 '+h(item.content_type)+'</text></svg>';
    body='<div class="preview-image"><img src="data:image/svg+xml,'+encodeURIComponent(svg)+'" alt="'+h(item.name)+'"></div>';
  } else if(t==='audio'){
    body='<div class="preview-audio"><div class="preview-audio-art">'+I.audio+'</div><div class="preview-audio-name">'+h(item.name)+'</div><div class="preview-audio-meta">'+fmtSize(item.size)+' \\u2014 '+h(item.content_type)+'</div><div class="preview-waveform">'+waveformBars()+'</div><div style="width:100%;display:flex;align-items:center;gap:12px;color:var(--text-2);font-size:13px;font-family:JetBrains Mono,monospace"><span>0:00</span><div style="flex:1;height:4px;background:var(--border);border-radius:2px"><div style="width:0%;height:100%;background:var(--text);border-radius:2px"></div></div><span>'+(item.size>5e6?'3:42':'0:48')+'</span></div></div>';
  } else if(t==='video'){
    const svg='<svg xmlns="http://www.w3.org/2000/svg" width="854" height="480"><rect width="854" height="480" fill="#0f172a"/><circle cx="427" cy="240" r="36" fill="none" stroke="rgba(255,255,255,.5)" stroke-width="2"/><polygon points="420,222 420,258 446,240" fill="rgba(255,255,255,.5)"/><text x="427" y="310" text-anchor="middle" fill="rgba(255,255,255,.4)" font-family="Inter,sans-serif" font-size="14">'+h(item.name)+' \\u2014 '+fmtSize(item.size)+'</text></svg>';
    body='<div class="preview-video"><div class="preview-video-poster"><img src="data:image/svg+xml,'+encodeURIComponent(svg)+'" alt="'+h(item.name)+'" style="width:100%;border-radius:4px"></div></div>';
  } else if(DEMO_DOCS[item.path]){
    const doc=DEMO_DOCS[item.path];
    if(doc.type==='pdf')body=renderPdfPages(doc,item);
    else if(doc.type==='docx')body=renderDocxPage(doc,item);
    else if(doc.type==='xlsx')body=renderXlsxSheet(doc,item);
    else if(doc.type==='pptx')body=renderPptxSlides(doc,item);
  } else if(t==='doc'){
    body='<div class="preview-doc"><div class="preview-doc-icon">'+I.doc+'</div><div class="preview-doc-name">'+h(item.name)+'</div><div class="preview-doc-meta">'+h(item.content_type)+'</div><div class="preview-doc-meta">'+fmtSize(item.size)+'</div><div class="preview-doc-desc">'+(item.description?h(item.description):'Document preview')+'</div></div>';
  } else if(t==='archive'){
    body='<div class="preview-doc"><div class="preview-doc-icon">'+I.archive+'</div><div class="preview-doc-name">'+h(item.name)+'</div><div class="preview-doc-meta">'+h(item.content_type)+'</div><div class="preview-doc-meta">'+fmtSize(item.size)+'</div></div>';
  } else {
    body='<div class="preview-generic"><div class="preview-generic-icon">'+fileIconHtml(item)+'</div><div class="preview-generic-name">'+h(item.name)+'</div><div class="preview-generic-meta">'+h(item.content_type||'Unknown')+' \\u2014 '+fmtSize(item.size)+'</div></div>';
  }

  const navL=hasPrev?'<button class="preview-nav preview-nav--prev" onclick="B.previewNav(-1)"><svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="15 18 9 12 15 6"/></svg></button>':'';
  const navR=hasNext?'<button class="preview-nav preview-nav--next" onclick="B.previewNav(1)"><svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="9 18 15 12 9 6"/></svg></button>':'';
  el.innerHTML='<div class="preview-header"><div class="preview-filename">'+fileIconHtml(item)+'<span>'+h(item.name)+'</span></div><div class="preview-info"><span>'+fmtSize(item.size)+'</span><span>'+fmtTime(item.updated_at)+'</span></div><div class="preview-actions"><button class="preview-btn" onclick="requireSignup(\\'download files\\')">'+I.download+'</button><button class="preview-btn" onclick="B.closePreview()">'+I.x+'</button></div></div><div class="preview-body">'+body+'</div>'+navL+navR;
}

function render(){
  renderSidebar();renderToolbar();renderBulk();renderItems();renderDetail();renderQuota();
}

/* ── App controller ──────────────────────────────────────────────── */
const B=window.B={
  nav(path){
    S.path=path;S.section='files';S.searchMode=false;S.selected.clear();S.detail=null;S.detailOpen=false;
    history.pushState(null,'','/browse/'+(path||''));
    loadItems();
  },
  setSection(sec){
    S.section=sec;S.path='';S.searchMode=false;S.selected.clear();S.detail=null;S.detailOpen=false;
    $('sidebar').classList.remove('open');$('sidebar-backdrop').classList.remove('visible');
    loadItems();
  },
  setView(v){S.view=v;localStorage.setItem('sv',v);renderItems()},
  setSort(col){
    if(S.sort.col===col)S.sort.asc=!S.sort.asc;
    else{S.sort.col=col;S.sort.asc=true}
    localStorage.setItem('sc',col);localStorage.setItem('sa',S.sort.asc?'1':'0');
    sortItems();renderToolbar();renderItems();
  },
  toggleSort(){$('sort-dd')?.classList.toggle('open')},

  clickSel(path,e){
    if(e.ctrlKey||e.metaKey){
      if(S.selected.has(path))S.selected.delete(path);else S.selected.add(path);
    } else if(e.shiftKey&&S.selected.size){
      const paths=S.items.map(i=>i.path);
      const last=[...S.selected].pop();
      const a=paths.indexOf(last),b=paths.indexOf(path);
      const[lo,hi]=[Math.min(a,b),Math.max(a,b)];
      for(let i=lo;i<=hi;i++)S.selected.add(paths[i]);
    } else {
      S.selected.clear();S.selected.add(path);
    }
    if(S.selected.size===1){
      S.detail=S.items.find(i=>i.path===[...S.selected][0])||null;
    }
    renderBulk();renderItems();renderDetail();
  },
  toggleSel(path,e){
    if(e.target.checked)S.selected.add(path);else S.selected.delete(path);
    renderBulk();renderItems();
  },
  selectAll(checked){
    if(checked)S.items.forEach(i=>S.selected.add(i.path));else S.selected.clear();
    renderBulk();renderItems();
  },
  clearSel(){S.selected.clear();renderBulk();renderItems()},

  ctx(e,path){
    e.preventDefault();
    const item=S.items.find(i=>i.path===path)||null;
    if(item&&!S.selected.has(path)){S.selected.clear();S.selected.add(path);renderItems()}
    renderCtx(e.clientX,e.clientY,item);
  },

  showDetail(path){
    hideCtx();
    S.detail=S.items.find(i=>i.path===path)||null;
    S.detailOpen=!!S.detail;S.detailTab='details';
    renderDetail();
  },
  closeDetail(){S.detailOpen=false;renderDetail()},
  detailTab(tab){S.detailTab=tab;renderDetail()},

  openPreview(path){
    const item=S.items.find(i=>i.path===path);
    if(!item||item.is_folder)return;
    S.previewItem=item;
    renderPreview();
  },
  closePreview(){
    S.previewItem=null;
    renderPreview();
  },
  previewNav(dir){
    const files=S.items.filter(i=>!i.is_folder);
    const idx=files.findIndex(i=>i.path===S.previewItem?.path);
    if(idx===-1)return;
    const next=files[idx+dir];
    if(next)B.openPreview(next.path);
  },
  showSlide(idx){
    document.querySelectorAll('.pptx-slide').forEach((s,i)=>s.classList.toggle('active',i===idx));
    document.querySelectorAll('.pptx-thumb').forEach((t,i)=>t.classList.toggle('active',i===idx));
  },

  search(q){
    S.searchQ=q;
    if(!q){S.searchMode=false;S.section='files';loadItems();return}
    S.searchMode=true;loadItems();
  },
};

/* ── Modal helpers ───────────────────────────────────────────────── */
function showModal(title,body,footer){
  const m=$('modal-overlay');
  m.innerHTML='<div class="modal"><div class="modal-header"><span class="modal-title">'+title+'</span><button class="modal-close" onclick="hideModal()">'+I.x+'</button></div><div class="modal-body">'+body+'</div>'+(footer?'<div class="modal-footer">'+footer+'</div>':'')+'</div>';
  m.classList.add('visible');
}
function hideModal(){$('modal-overlay').classList.remove('visible');$('modal-overlay').innerHTML=''}
window.showModal=showModal;window.hideModal=hideModal;window.requireSignup=requireSignup;

function hideCtx(){$('ctx-menu').classList.remove('visible')}
window.hideCtx=hideCtx;
document.addEventListener('click',()=>{hideCtx();$('sort-dd')?.classList.remove('open')});

/* ── Keyboard shortcuts ──────────────────────────────────────────── */
document.addEventListener('keydown',e=>{
  const tag=e.target.tagName;
  if(tag==='INPUT'||tag==='TEXTAREA'||tag==='SELECT')return;
  // Preview navigation
  if(S.previewItem){
    if(e.key==='Escape'){B.closePreview();return}
    if(e.key==='ArrowLeft'){B.previewNav(-1);return}
    if(e.key==='ArrowRight'){B.previewNav(1);return}
  }
  if(e.key==='/'||(e.key==='f'&&(e.ctrlKey||e.metaKey))){e.preventDefault();$('search-input').focus();return}
  if(e.key==='Escape'){S.selected.clear();S.detailOpen=false;hideCtx();hideModal();render();return}
  const sel=[...S.selected];
  if(e.key==='i'||e.key==='I'){if(sel.length===1){B.showDetail(sel[0])}else{S.detailOpen=false;renderDetail()}return}
  if(e.key==='a'&&(e.ctrlKey||e.metaKey)){e.preventDefault();B.selectAll(true);return}
  if(e.key==='Enter'){if(sel.length===1){const it=S.items.find(i=>i.path===sel[0]);if(it?.is_folder)B.nav(it.path);else if(it)B.openPreview(it.path)}return}
  if(e.key===' '&&!S.previewItem){
    e.preventDefault();
    if(sel.length===1){const it=S.items.find(i=>i.path===sel[0]);if(it&&!it.is_folder)B.openPreview(it.path)}
    return;
  }
  if(e.key==='ArrowDown'||e.key==='ArrowUp'){
    e.preventDefault();
    const paths=S.items.map(i=>i.path);
    const cur=sel.length?paths.indexOf(sel[sel.length-1]):-1;
    const next=e.key==='ArrowDown'?Math.min(cur+1,paths.length-1):Math.max(cur-1,0);
    S.selected.clear();S.selected.add(paths[next]);
    S.detail=S.items[next];
    renderBulk();renderItems();renderDetail();
    document.querySelector('.file-row.selected,.grid-card.selected')?.scrollIntoView({block:'nearest'});
    return;
  }
});

/* ── Search ───────────────────────────────────────────────────────── */
let searchTimer=null;
$('search-input').addEventListener('input',e=>{
  clearTimeout(searchTimer);
  searchTimer=setTimeout(()=>B.search(e.target.value),300);
});
$('search-input').addEventListener('keydown',e=>{if(e.key==='Escape'){e.target.value='';B.search('')}});

/* ── Sidebar clicks ──────────────────────────────────────────────── */
$('sidebar-nav').addEventListener('click',e=>{
  const el=e.target.closest('.sidebar-item');
  if(el)B.setSection(el.dataset.section);
});

/* ── Mobile sidebar ──────────────────────────────────────────────── */
$('mobile-toggle')?.addEventListener('click',()=>{
  $('sidebar').classList.toggle('open');
  $('sidebar-backdrop').classList.toggle('visible');
});
$('sidebar-backdrop').addEventListener('click',()=>{
  $('sidebar').classList.remove('open');
  $('sidebar-backdrop').classList.remove('visible');
});

/* ── Theme toggle ────────────────────────────────────────────────── */
$('theme-btn').addEventListener('click',()=>{
  const d=document.documentElement.classList.toggle('dark');
  localStorage.setItem('theme',d?'dark':'light');
});
(function(){const s=localStorage.getItem('theme');
  if(s==='light')document.documentElement.classList.remove('dark');
  else if(!s&&!window.matchMedia('(prefers-color-scheme:dark)').matches)document.documentElement.classList.remove('dark')})();

/* ── History ─────────────────────────────────────────────────────── */
window.addEventListener('popstate',()=>{
  const p=decodeURIComponent(location.pathname.replace('/browse/','').replace('/browse',''));
  S.path=p;S.section='files';S.searchMode=false;loadItems();
});

/* ── Init ────────────────────────────────────────────────────────── */
const initPath=decodeURIComponent(location.pathname.replace('/browse/','').replace('/browse',''));
S.path=initPath==='browse'?'':initPath;
loadItems();

})();
</script>
</body></html>`;
}

/* ═══════════════════════════════════════════════════════════════════════
   BROWSE APP (authenticated)
   ═══════════════════════════════════════════════════════════════════════ */
function browseApp(actor: string): string {
  const displayName = esc(actor.slice(2));
  return `<!DOCTYPE html>
<html lang="en" class="dark">
<head>
<meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Browse — storage.now</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800&family=JetBrains+Mono:wght@400;500;600;700&display=swap" rel="stylesheet">
<link rel="stylesheet" href="/browse.css">
</head>
<body>

<nav>
  <a href="/" class="logo"><span class="logo-dot"></span> storage.now</a>
  <button class="mobile-toggle" id="mobile-toggle" aria-label="Menu">
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="3" y1="6" x2="21" y2="6"/><line x1="3" y1="12" x2="21" y2="12"/><line x1="3" y1="18" x2="21" y2="18"/></svg>
  </button>
  <div class="nav-search" id="nav-search">
    <span class="search-icon"><svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg></span>
    <input type="text" id="search-input" placeholder="Search files..." autocomplete="off" spellcheck="false">
  </div>
  <div class="nav-right">
    <span class="nav-user">${displayName}</span>
    <a href="/auth/logout" class="nav-signout">sign out</a>
    <button class="theme-toggle" id="theme-btn" aria-label="Toggle theme">
      <svg class="icon-moon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 12.79A9 9 0 1111.21 3 7 7 0 0021 12.79z"/></svg>
      <svg class="icon-sun" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="5"/><line x1="12" y1="1" x2="12" y2="3"/><line x1="12" y1="21" x2="12" y2="23"/><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/><line x1="1" y1="12" x2="3" y2="12"/><line x1="21" y1="12" x2="23" y2="12"/><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/></svg>
    </button>
  </div>
</nav>

<div id="app">
  <div class="sidebar-backdrop" id="sidebar-backdrop"></div>
  <aside id="sidebar">
    <div class="sidebar-nav" id="sidebar-nav"></div>
    <div class="sidebar-quota" id="quota"></div>
  </aside>

  <main id="main">
    <div id="toolbar"></div>
    <div id="bulk-bar"></div>
    <div class="file-content" id="file-content"></div>
    <div class="drop-overlay" id="drop-overlay">
      <div class="drop-overlay-icon"><svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/></svg></div>
      <div class="drop-overlay-text">Drop files to upload</div>
      <div class="drop-overlay-sub">Files will be uploaded to the current folder</div>
    </div>
  </main>

  <aside id="detail-panel" class="hidden"></aside>
</div>

<div id="ctx-menu"></div>
<div id="modal-overlay"></div>
<div id="preview-overlay"></div>
<div class="upload-panel" id="upload-panel">
  <div class="upload-header"><span class="upload-title" id="upload-title">Uploading...</span><button class="upload-close" id="upload-close">&times;</button></div>
  <div class="upload-list" id="upload-list"></div>
</div>
<div class="toast-container" id="toasts"></div>
<input type="file" id="file-input" multiple style="display:none">

<script>
(function(){
'use strict';
const ACTOR=${JSON.stringify(actor)};

/* ── Icons ────────────────────────────────────────────────────────── */
const I={
  folder:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M22 19a2 2 0 01-2 2H4a2 2 0 01-2-2V5a2 2 0 012-2h5l2 3h9a2 2 0 012 2z"/></svg>',
  file:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>',
  image:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="3" y="3" width="18" height="18" rx="2"/><circle cx="8.5" cy="8.5" r="1.5"/><polyline points="21 15 16 10 5 21"/></svg>',
  video:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polygon points="23 7 16 12 23 17 23 7"/><rect x="1" y="5" width="15" height="14" rx="2"/></svg>',
  audio:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M9 18V5l12-2v13"/><circle cx="6" cy="18" r="3"/><circle cx="18" cy="16" r="3"/></svg>',
  doc:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/><polyline points="10 9 9 9 8 9"/></svg>',
  code:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/></svg>',
  archive:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polyline points="21 8 21 21 3 21 3 8"/><rect x="1" y="3" width="22" height="5"/><line x1="10" y1="12" x2="14" y2="12"/></svg>',
  sheet:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="3" y="3" width="18" height="18" rx="2"/><line x1="3" y1="9" x2="21" y2="9"/><line x1="3" y1="15" x2="21" y2="15"/><line x1="9" y1="3" x2="9" y2="21"/></svg>',
  text:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/></svg>',
  star:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/></svg>',
  starFill:'<svg viewBox="0 0 24 24" fill="currentColor" stroke="currentColor" stroke-width="1.5"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/></svg>',
  trash:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/></svg>',
  download:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>',
  upload:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/></svg>',
  share:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="18" cy="5" r="3"/><circle cx="6" cy="12" r="3"/><circle cx="18" cy="19" r="3"/><line x1="8.59" y1="13.51" x2="15.42" y2="17.49"/><line x1="15.41" y1="6.51" x2="8.59" y2="10.49"/></svg>',
  rename:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M11 4H4a2 2 0 00-2 2v14a2 2 0 002 2h14a2 2 0 002-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 013 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>',
  move:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M22 19a2 2 0 01-2 2H4a2 2 0 01-2-2V5a2 2 0 012-2h5l2 3h9a2 2 0 012 2z"/><line x1="12" y1="11" x2="12" y2="17"/><polyline points="9 14 12 17 15 14"/></svg>',
  copy:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1"/></svg>',
  info:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg>',
  x:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>',
  home:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M3 9l9-7 9 7v11a2 2 0 01-2 2H5a2 2 0 01-2-2z"/></svg>',
  shared:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M17 21v-2a4 4 0 00-4-4H5a4 4 0 00-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 00-3-3.87"/><path d="M16 3.13a4 4 0 010 7.75"/></svg>',
  clock:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>',
  grid:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/><rect x="14" y="14" width="7" height="7"/></svg>',
  list:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="8" y1="6" x2="21" y2="6"/><line x1="8" y1="12" x2="21" y2="12"/><line x1="8" y1="18" x2="21" y2="18"/><line x1="3" y1="6" x2="3.01" y2="6"/><line x1="3" y1="12" x2="3.01" y2="12"/><line x1="3" y1="18" x2="3.01" y2="18"/></svg>',
  plus:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>',
  restore:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polyline points="1 4 1 10 7 10"/><path d="M3.51 15a9 9 0 102.13-9.36L1 10"/></svg>',
  link:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M10 13a5 5 0 007.54.54l3-3a5 5 0 00-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 00-7.54-.54l-3 3a5 5 0 007.07 7.07l1.71-1.71"/></svg>',
};

/* ── State ─────────────────────────────────────────────────────────── */
const S={
  path:'',view:localStorage.getItem('sv')||'list',
  sort:{col:localStorage.getItem('sc')||'name',asc:localStorage.getItem('sa')!=='0'},
  items:[],selected:new Set(),section:'files',
  detail:null,detailOpen:false,detailTab:'details',
  clipboard:null,stats:null,searchMode:false,searchQ:'',
  uploading:[],
  previewItem:null,previewIdx:-1,previewContent:null,previewLoading:false,
};

const $=id=>document.getElementById(id);

/* ── Helpers ───────────────────────────────────────────────────────── */
function h(s){return (s||'').replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;')}
function fmtSize(b){if(!b)return '—';if(b<1024)return b+' B';if(b<1048576)return (b/1024).toFixed(1)+' KB';if(b<1073741824)return (b/1048576).toFixed(1)+' MB';return (b/1073741824).toFixed(1)+' GB'}
function fmtTime(ts){if(!ts)return '—';const d=Date.now()-ts,s=d/1000;if(s<60)return 'just now';if(s<3600)return Math.floor(s/60)+'m ago';if(s<86400)return Math.floor(s/3600)+'h ago';if(s<2592000)return Math.floor(s/86400)+'d ago';return new Date(ts).toLocaleDateString()}

function fileType(item){
  if(item.is_folder)return 'folder';
  const ct=item.content_type||'';const n=item.name||'';
  if(ct.startsWith('image/'))return 'image';
  if(ct.startsWith('video/'))return 'video';
  if(ct.startsWith('audio/'))return 'audio';
  if(ct==='application/pdf'||ct.includes('document')||ct.includes('msword'))return 'doc';
  if(ct.includes('spreadsheet')||ct.includes('csv')||ct==='text/csv')return 'sheet';
  if(ct.includes('zip')||ct.includes('tar')||ct.includes('gzip')||ct.includes('compressed'))return 'archive';
  const ext=n.split('.').pop()?.toLowerCase()||'';
  if(['js','ts','py','go','rs','java','c','cpp','h','rb','php','sh','bash','yaml','yml','toml','json','xml','html','css','sql','md'].includes(ext))return 'code';
  if(['txt','log','text'].includes(ext)||ct.startsWith('text/'))return 'text';
  return 'generic';
}
function fileIconHtml(item){const t=fileType(item);return '<div class="file-icon file-icon--'+t+'">'+I[t==='generic'?'file':t]+'</div>'}

/* ── Language detection ─────────────────────────────────────────── */
function langFromName(name){
  const ext=(name||'').split('.').pop()?.toLowerCase()||'';
  const map={go:'go',py:'py',js:'js',ts:'js',jsx:'js',tsx:'js',css:'css',html:'html',htm:'html',yaml:'yaml',yml:'yaml',toml:'toml',json:'json',sql:'sql',sh:'sh',bash:'sh',rs:'rs',java:'java',c:'c',cpp:'cpp',h:'c',rb:'rb',php:'php',xml:'html',md:'md',dockerfile:'sh',Dockerfile:'sh'};
  return map[ext]||'text';
}

/* ── Syntax highlighter ────────────────────────────────────────── */
function highlightCode(code,lang){
  const lines=code.split('\\n');
  return lines.map(raw=>{
    let ln=h(raw);
    ln=ln.replace(/(&quot;(?:[^&]|&(?!quot;))*?&quot;)/g,'<span class="tok-str">$1</span>');
    ln=ln.replace(/(&#x27;(?:[^&]|&(?!#x27;))*?&#x27;)/g,'<span class="tok-str">$1</span>');
    ln=ln.replace(/(\\x60[^\\x60]*\\x60)/g,'<span class="tok-str">$1</span>');
    if(['go','js','java','c','cpp','rs','css'].includes(lang)){
      ln=ln.replace(/(^|\\s)(\\/{2}.*)$/,'$1<span class="tok-cm">$2</span>');
    }
    if(['py','yaml','toml','sh','dockerfile'].includes(lang)){
      ln=ln.replace(/(^|\\s)(#.*)$/,'$1<span class="tok-cm">$2</span>');
    }
    if(lang==='html'){
      ln=ln.replace(/(&lt;!--.*?--&gt;)/g,'<span class="tok-cm">$1</span>');
    }
    ln=ln.replace(/\\b(\\d+\\.?\\d*)\\b/g,'<span class="tok-num">$1</span>');
    const kw={
      go:'package|import|func|var|const|type|struct|interface|return|if|else|for|range|switch|case|default|defer|go|chan|select|map|make|new|nil|true|false|break|continue|error|string|int|bool|byte|fmt',
      py:'import|from|def|class|return|if|elif|else|for|while|in|not|and|or|is|None|True|False|with|as|try|except|finally|raise|pass|break|continue|lambda|yield|async|await|self|print',
      js:'import|export|from|const|let|var|function|return|if|else|for|while|do|switch|case|default|break|continue|new|this|class|extends|async|await|try|catch|finally|throw|typeof|instanceof|null|undefined|true|false|of|in|console|require|module',
      css:'@media|@keyframes|@import|@font-face|@layer|@supports|@property|:root|:hover|:focus|:active|:visited|!important|inherit|initial|unset',
      html:'DOCTYPE|html|head|body|meta|link|title|script|style|div|span|a|p|h1|h2|h3|h4|img|ul|ol|li|table|tr|td|th|form|input|button|nav|main|aside|section|article|header|footer',
      sh:'if|then|else|fi|for|do|done|while|case|esac|function|return|exit|echo|export|source|cd|mkdir|rm|cp|mv|chmod|chown|grep|awk|sed|cat|FROM|RUN|CMD|COPY|WORKDIR|EXPOSE|ENV|ARG|ENTRYPOINT',
      sql:'SELECT|FROM|WHERE|AND|OR|NOT|INSERT|INTO|VALUES|UPDATE|SET|DELETE|CREATE|TABLE|DROP|ALTER|INDEX|JOIN|LEFT|RIGHT|INNER|OUTER|ON|GROUP|BY|ORDER|ASC|DESC|LIMIT|OFFSET|IF|EXISTS|PRIMARY|KEY|UNIQUE|NOT|NULL|DEFAULT|INTEGER|TEXT|REAL|BLOB',
      yaml:'true|false|null|yes|no',
      json:'true|false|null',
    };
    const kwList=kw[lang];
    if(kwList){
      ln=ln.replace(new RegExp('\\\\b('+kwList+')\\\\b','g'),'<span class="tok-kw">$1</span>');
    }
    return '<span class="line">'+ln+'</span>';
  }).join('\\n');
}

/* ── Markdown renderer (basic) ─────────────────────────────────── */
function renderMarkdown(md){
  let html=h(md);
  html=html.replace(/\\x60\\x60\\x60(\\w*)\\n([\\s\\S]*?)\\x60\\x60\\x60/g,(m,lang,code)=>'<pre><code>'+code+'</code></pre>');
  html=html.replace(/\\x60([^\\x60]+)\\x60/g,'<code>$1</code>');
  html=html.replace(/^#### (.+)$/gm,'<h4>$1</h4>');
  html=html.replace(/^### (.+)$/gm,'<h3>$1</h3>');
  html=html.replace(/^## (.+)$/gm,'<h2>$1</h2>');
  html=html.replace(/^# (.+)$/gm,'<h1>$1</h1>');
  html=html.replace(/\\*\\*(.+?)\\*\\*/g,'<strong>$1</strong>');
  html=html.replace(/\\*(.+?)\\*/g,'<em>$1</em>');
  html=html.replace(/\\[([^\\]]+)\\]\\(([^)]+)\\)/g,'<a href="$2">$1</a>');
  html=html.replace(/^---$/gm,'<hr>');
  html=html.replace(/^[\\-\\*] (.+)$/gm,'<li>$1</li>');
  html=html.replace(/^&gt; (.+)$/gm,'<blockquote>$1</blockquote>');
  html=html.replace(/^(?!<[hluobpc]|<\\/|<hr|<li|<block|<pre|<code)(.+)$/gm,'<p>$1</p>');
  html=html.replace(/(<li>.*?<\\/li>\\n?)+/g,'<ul>$&</ul>');
  return html;
}

/* ── CSV to table ──────────────────────────────────────────────── */
function csvToTable(csv){
  const rows=csv.trim().split('\\n').map(r=>r.split(',').map(c=>c.trim()));
  if(!rows.length)return '';
  let t='<table><thead><tr>'+rows[0].map(c=>'<th>'+h(c)+'</th>').join('')+'</tr></thead><tbody>';
  for(let i=1;i<rows.length;i++){
    t+='<tr>'+rows[i].map(c=>'<td>'+h(c)+'</td>').join('')+'</tr>';
  }
  return t+'</tbody></table>';
}

/* ── Generate waveform bars HTML ───────────────────────────────── */
function waveformBars(){
  let bars='';
  for(let i=0;i<60;i++){
    const h=8+Math.floor(Math.random()*32);
    bars+='<div class="bar" style="height:'+h+'px"></div>';
  }
  return bars;
}

/* ── API ───────────────────────────────────────────────────────────── */
const api={
  get:u=>fetch(u).then(r=>r.json()),
  post:(u,b)=>fetch(u,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(b)}).then(r=>r.json()),
  patch:(u,b)=>fetch(u,{method:'PATCH',headers:{'Content-Type':'application/json'},body:JSON.stringify(b)}).then(r=>r.json()),
  del:u=>fetch(u,{method:'DELETE'}).then(r=>r.json()),
};

/* ── Toast ─────────────────────────────────────────────────────────── */
function toast(msg,type='info'){
  const el=document.createElement('div');
  el.className='toast toast--'+type;
  el.innerHTML='<span class="toast-msg">'+h(msg)+'</span>';
  $('toasts').appendChild(el);
  setTimeout(()=>{el.classList.add('fade-out');setTimeout(()=>el.remove(),300)},3000);
}

/* ── Render: Sidebar ──────────────────────────────────────────────── */
function renderSidebar(){
  const items=[
    {id:'files',icon:I.folder,label:'My Files'},
    {id:'shared',icon:I.shared,label:'Shared with me'},
    {id:'recent',icon:I.clock,label:'Recent'},
    {id:'starred',icon:I.star,label:'Starred'},
    {id:'trash',icon:I.trash,label:'Trash',badge:S.stats?.trash_count||0},
  ];
  let html='';
  for(const it of items){
    const active=S.section===it.id?' active':'';
    const badge=it.badge?'<span class="item-badge">'+it.badge+'</span>':'';
    html+='<div class="sidebar-item'+active+'" data-section="'+it.id+'"><span class="item-icon">'+it.icon+'</span><span class="item-label">'+it.label+'</span>'+badge+'</div>';
  }
  $('sidebar-nav').innerHTML=html;
}

function renderQuota(){
  if(!S.stats)return;
  const pct=Math.min(100,S.stats.total_size/S.stats.quota*100);
  const cls=pct>80?'danger':pct>50?'warn':'ok';
  $('quota').innerHTML='<div class="quota-bar"><div class="quota-fill quota-fill--'+cls+'" style="width:'+pct.toFixed(1)+'%"></div></div><div class="quota-text">'+fmtSize(S.stats.total_size)+' of '+fmtSize(S.stats.quota)+' used</div>';
}

/* ── Render: Toolbar ──────────────────────────────────────────────── */
function renderToolbar(){
  const parts=S.path.replace(/\\/$/,'').split('/').filter(Boolean);
  let bc='<span class="breadcrumb-home" onclick="B.nav(\\'\\')"><svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M3 9l9-7 9 7v11a2 2 0 01-2 2H5a2 2 0 01-2-2z"/></svg></span>';
  let acc='';
  for(const p of parts){
    acc+=p+'/';
    const a=acc;
    bc+='<span class="breadcrumb-sep">/</span><span class="breadcrumb-segment" onclick="B.nav(\\''+h(a)+'\\')">'+h(p)+'</span>';
  }
  if(S.searchMode)bc='<span class="breadcrumb-segment">Search results for "'+h(S.searchQ)+'"</span>';

  const lActive=S.view==='list'?' active':'';
  const gActive=S.view==='grid'?' active':'';
  const sectionIsTrash=S.section==='trash';

  let right='';
  if(!S.searchMode&&S.section==='files'){
    right='<div class="toolbar-group"><button class="tool-btn'+lActive+'" onclick="B.setView(\\'list\\')" title="List view">'+I.list+'</button><button class="tool-btn'+gActive+'" onclick="B.setView(\\'grid\\')" title="Grid view">'+I.grid+'</button></div>';
    right+='<div class="toolbar-divider"></div>';
    right+='<div class="sort-dropdown" id="sort-dd"><button class="tool-btn" onclick="B.toggleSort()">Sort</button><div class="sort-menu" id="sort-menu"><div class="sort-option'+(S.sort.col==='name'?' active':'')+'" onclick="B.setSort(\\'name\\')">Name <span class="sort-dir">'+(S.sort.col==='name'?(S.sort.asc?'A→Z':'Z→A'):'')+'</span></div><div class="sort-option'+(S.sort.col==='updated_at'?' active':'')+'" onclick="B.setSort(\\'updated_at\\')">Modified <span class="sort-dir">'+(S.sort.col==='updated_at'?(S.sort.asc?'Old→New':'New→Old'):'')+'</span></div><div class="sort-option'+(S.sort.col==='size'?' active':'')+'" onclick="B.setSort(\\'size\\')">Size <span class="sort-dir">'+(S.sort.col==='size'?(S.sort.asc?'Small→Big':'Big→Small'):'')+'</span></div></div></div>';
    right+='<div class="toolbar-divider"></div>';
    right+='<button class="tool-btn" onclick="B.newFolder()" title="New folder">'+I.plus+' Folder</button>';
    right+='<button class="tool-btn tool-btn--primary" onclick="$(\\'file-input\\').click()" title="Upload">'+I.upload+' Upload</button>';
  }
  if(sectionIsTrash){
    right='<button class="tool-btn bulk-btn--danger" onclick="B.emptyTrash()">'+I.trash+' Empty trash</button>';
  }

  $('toolbar').innerHTML='<div class="breadcrumb">'+bc+'</div>'+right;
}

/* ── Render: Bulk bar ─────────────────────────────────────────────── */
function renderBulk(){
  const n=S.selected.size;
  const el=$('bulk-bar');
  if(!n){el.className='';el.innerHTML='';return}
  el.className='visible';
  const isTrash=S.section==='trash';
  let btns='';
  if(isTrash){
    btns='<button class="bulk-btn" onclick="B.bulkRestore()">'+I.restore+' Restore</button>';
  } else {
    btns='<button class="bulk-btn" onclick="B.bulkDownload()">'+I.download+' Download</button>';
    btns+='<button class="bulk-btn" onclick="B.bulkStar()">'+I.star+' Star</button>';
    btns+='<button class="bulk-btn" onclick="B.bulkMove()">'+I.move+' Move</button>';
    btns+='<button class="bulk-btn bulk-btn--danger" onclick="B.bulkTrash()">'+I.trash+' Trash</button>';
  }
  el.innerHTML='<span class="bulk-count">'+n+' selected</span><div class="bulk-actions">'+btns+'</div><button class="bulk-dismiss" onclick="B.clearSel()">'+I.x+'</button>';
}

/* ── Render: File list ────────────────────────────────────────────── */
function renderItems(){
  const fc=$('file-content');
  if(!S.items.length){
    const msgs={files:'This folder is empty',shared:'No files shared with you yet',recent:'No recently accessed files',starred:'No starred files',trash:'Trash is empty'};
    fc.innerHTML='<div class="empty-state"><div class="empty-icon"><svg width="56" height="56" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1"><path d="M22 19a2 2 0 01-2 2H4a2 2 0 01-2-2V5a2 2 0 012-2h5l2 3h9a2 2 0 012 2z"/></svg></div><div class="empty-title">'+(msgs[S.section]||'No files')+'</div><div class="empty-sub">'+(S.section==='files'?'Upload files or create a folder to get started.':'')+'</div>'+(S.section==='files'?'<button class="empty-action" onclick="$(\\'file-input\\').click()">'+I.upload+' Upload files</button>':'')+'</div>';
    return;
  }
  if(S.view==='grid')renderGrid(fc);
  else renderList(fc);
}

function renderList(fc){
  const allSel=S.items.length>0&&S.items.every(i=>S.selected.has(i.path));
  let html='<div id="file-list"><div class="list-header"><input type="checkbox" class="row-check col--check"'+(allSel?' checked':'')+' onchange="B.selectAll(this.checked)"><span class="col col--icon"></span><span class="col col--name" onclick="B.setSort(\\'name\\')">Name'+(S.sort.col==='name'?'<span class="sort-arrow">'+(S.sort.asc?' ↑':' ↓')+'</span>':'')+'</span><span class="col col--modified" onclick="B.setSort(\\'updated_at\\')">Modified'+(S.sort.col==='updated_at'?'<span class="sort-arrow">'+(S.sort.asc?' ↑':' ↓')+'</span>':'')+'</span><span class="col col--size" onclick="B.setSort(\\'size\\')">Size'+(S.sort.col==='size'?'<span class="sort-arrow">'+(S.sort.asc?' ↑':' ↓')+'</span>':'')+'</span></div>';
  for(const item of S.items){
    const sel=S.selected.has(item.path)?' selected':'';
    const isF=item.is_folder;
    const starCls=item.starred?' starred':'';
    html+='<div class="file-row'+(isF?' file-row--folder':'')+sel+'" data-path="'+h(item.path)+'" oncontextmenu="B.ctx(event,\\''+h(item.path)+'\\')"><input type="checkbox" class="row-check col--check"'+(sel?' checked':'')+' onclick="event.stopPropagation()" onchange="B.toggleSel(\\''+h(item.path)+'\\',event)">'+fileIconHtml(item)+'<span class="file-name"><span class="file-name-text">'+h(item.name)+(isF?'/':'')+'</span><button class="file-star'+starCls+'" onclick="event.stopPropagation();B.star(\\''+h(item.path)+'\\','+(!item.starred?1:0)+')">'+(item.starred?I.starFill:I.star)+'</button></span><span class="file-modified">'+fmtTime(item.updated_at)+'</span><span class="file-size">'+fmtSize(item.is_folder?0:item.size)+'</span></div>';
  }
  html+='</div>';
  fc.innerHTML=html;
  // Click handlers
  fc.querySelectorAll('.file-row').forEach(el=>{
    el.addEventListener('click',e=>{
      if(e.target.closest('.row-check')||e.target.closest('.file-star')||e.target.closest('.grid-star'))return;
      const p=el.dataset.path;
      if(e.ctrlKey||e.metaKey||e.shiftKey){B.clickSel(p,e);return}
      const item=S.items.find(i=>i.path===p);
      if(item?.is_folder)B.nav(p);
      else if(item)B.openPreview(p);
    });
  });
}

function renderGrid(fc){
  let html='<div id="file-grid">';
  for(const item of S.items){
    const sel=S.selected.has(item.path)?' selected':'';
    const isF=item.is_folder;
    const t=fileType(item);
    const starCls=item.starred?' starred':'';
    html+='<div class="grid-card'+(isF?' grid-card--folder':'')+sel+'" data-path="'+h(item.path)+'" oncontextmenu="B.ctx(event,\\''+h(item.path)+'\\')"><div class="grid-check"><input type="checkbox" class="row-check"'+(sel?' checked':'')+' onclick="event.stopPropagation()" onchange="B.toggleSel(\\''+h(item.path)+'\\',event)"></div><button class="grid-star'+starCls+'" onclick="event.stopPropagation();B.star(\\''+h(item.path)+'\\','+(!item.starred?1:0)+')">'+(item.starred?I.starFill:I.star)+'</button><div class="grid-thumb"><div class="file-icon file-icon--'+t+'">'+I[t==='generic'?'file':t]+'</div></div><div class="grid-name">'+h(item.name)+(isF?'/':'')+'</div></div>';
  }
  html+='</div>';
  fc.innerHTML=html;
  fc.querySelectorAll('.grid-card').forEach(el=>{
    el.addEventListener('click',e=>{
      if(e.target.closest('.row-check')||e.target.closest('.file-star')||e.target.closest('.grid-star'))return;
      const p=el.dataset.path;
      if(e.ctrlKey||e.metaKey||e.shiftKey){B.clickSel(p,e);return}
      const item=S.items.find(i=>i.path===p);
      if(item?.is_folder)B.nav(p);
      else if(item)B.openPreview(p);
    });
  });
}

/* ── Render: Detail panel ─────────────────────────────────────────── */
function renderDetail(){
  const dp=$('detail-panel');
  if(!S.detailOpen||!S.detail){dp.className='hidden';return}
  dp.className='';
  const it=S.detail;
  const t=fileType(it);
  const detTab=S.detailTab==='details'?' active':'';
  const actTab=S.detailTab==='activity'?' active':'';
  let body='';
  if(S.detailTab==='details'){
    body='<div class="detail-icon file-icon--'+t+'">'+I[t==='generic'?'file':t]+'</div>';
    body+='<div class="detail-filename">'+h(it.name)+'</div>';
    body+='<div class="detail-type">'+(it.is_folder?'Folder':h(it.content_type||'Unknown'))+'</div>';
    if(!it.is_folder)body+='<div class="detail-field"><div class="detail-label">Size</div><div class="detail-value">'+fmtSize(it.size)+'</div></div>';
    body+='<div class="detail-field"><div class="detail-label">Location</div><div class="detail-value">'+h(it.path)+'</div></div>';
    body+='<div class="detail-field"><div class="detail-label">Created</div><div class="detail-value">'+fmtTime(it.created_at)+'</div></div>';
    body+='<div class="detail-field"><div class="detail-label">Modified</div><div class="detail-value">'+fmtTime(it.updated_at)+'</div></div>';
    body+='<div class="detail-field"><div class="detail-label">Description</div><textarea class="detail-description" placeholder="Add a description..." onblur="B.saveDesc(\\''+h(it.path)+'\\',this.value)">'+(it.description?h(it.description):'')+'</textarea></div>';
  } else {
    body='<div class="activity-list"><div class="activity-item"><div class="activity-avatar">U</div><div class="activity-content"><div class="activity-text"><strong>You</strong> uploaded this file</div><div class="activity-time">'+fmtTime(it.created_at)+'</div></div></div>';
    if(it.updated_at!==it.created_at)body+='<div class="activity-item"><div class="activity-avatar">U</div><div class="activity-content"><div class="activity-text"><strong>You</strong> modified this file</div><div class="activity-time">'+fmtTime(it.updated_at)+'</div></div></div>';
    body+='</div>';
  }
  dp.innerHTML='<div class="detail-header"><div class="detail-tabs"><span class="detail-tab'+detTab+'" onclick="B.detailTab(\\'details\\')">Details</span><span class="detail-tab'+actTab+'" onclick="B.detailTab(\\'activity\\')">Activity</span></div><button class="detail-close" onclick="B.closeDetail()">'+I.x+'</button></div><div class="detail-body">'+body+'</div>';
}

/* ── Render: Context menu ─────────────────────────────────────────── */
function renderCtx(x,y,item){
  const m=$('ctx-menu');
  let html='';
  if(!item){
    // Empty area context
    html+='<div class="ctx-item" onclick="B.newFolder()">'+I.plus+' New folder</div>';
    html+='<div class="ctx-item" onclick="$(\\'file-input\\').click()">'+I.upload+' Upload files</div>';
    if(S.clipboard){html+='<div class="ctx-divider"></div><div class="ctx-item" onclick="B.paste()">'+I.copy+' Paste</div>'}
    m.innerHTML=html;
  } else if(S.section==='trash'){
    html+='<div class="ctx-item" onclick="B.restoreItems([\\''+h(item.path)+'\\'])">'+I.restore+' Restore</div>';
    html+='<div class="ctx-divider"></div>';
    html+='<div class="ctx-item ctx-item--danger" onclick="B.permDelete([\\''+h(item.path)+'\\'])">'+I.trash+' Delete forever</div>';
    m.innerHTML=html;
  } else {
    if(item.is_folder)html+='<div class="ctx-item" onclick="B.nav(\\''+h(item.path)+'\\')">'+I.folder+' Open</div>';
    else html+='<div class="ctx-item" onclick="B.downloadFile(S.items.find(i=>i.path===\\''+h(item.path)+'\\'))">'+I.download+' Download</div>';
    html+='<div class="ctx-divider"></div>';
    html+='<div class="ctx-item" onclick="B.startRename(\\''+h(item.path)+'\\')">'+I.rename+' Rename <span class="ctx-shortcut">F2</span></div>';
    html+='<div class="ctx-item" onclick="B.showMoveModal([\\''+h(item.path)+'\\'])">'+I.move+' Move to...</div>';
    if(!item.is_folder)html+='<div class="ctx-item" onclick="B.copyFile(\\''+h(item.path)+'\\')">'+I.copy+' Copy</div>';
    html+='<div class="ctx-divider"></div>';
    html+='<div class="ctx-item" onclick="B.showShareModal(\\''+h(item.path)+'\\')">'+I.share+' Share</div>';
    html+='<div class="ctx-item" onclick="B.star(\\''+h(item.path)+'\\','+(item.starred?0:1)+')">'+(!item.starred?I.star:I.starFill)+' '+(item.starred?'Unstar':'Star')+' <span class="ctx-shortcut">S</span></div>';
    html+='<div class="ctx-item" onclick="B.showDetail(\\''+h(item.path)+'\\')">'+I.info+' Details <span class="ctx-shortcut">I</span></div>';
    html+='<div class="ctx-divider"></div>';
    html+='<div class="ctx-item ctx-item--danger" onclick="B.trashItems([\\''+h(item.path)+'\\'])">'+I.trash+' Move to trash <span class="ctx-shortcut">Del</span></div>';
    m.innerHTML=html;
  }
  // Position
  const vw=window.innerWidth,vh=window.innerHeight;
  m.style.left=Math.min(x,vw-200)+'px';
  m.style.top=Math.min(y,vh-m.scrollHeight-10)+'px';
  m.classList.add('visible');
}

/* ── Render: Preview overlay ───────────────────────────────────── */
function renderPreview(){
  const el=$('preview-overlay');
  if(!S.previewItem){el.classList.remove('visible');el.innerHTML='';return}
  el.classList.add('visible');
  const item=S.previewItem;
  const t=fileType(item);
  const files=S.items.filter(i=>!i.is_folder);
  const idx=files.findIndex(i=>i.path===item.path);
  const hasPrev=idx>0;
  const hasNext=idx<files.length-1;
  const content=S.previewContent;
  const lang=langFromName(item.name);
  const isMd=item.name.endsWith('.md')||item.content_type==='text/markdown';

  let body='';
  if(S.previewLoading){
    body='<div class="spinner"></div>';
  } else if(isMd&&content){
    body='<div class="preview-markdown">'+renderMarkdown(content)+'</div>';
  } else if(t==='code'&&content){
    body='<pre class="preview-code"><code>'+highlightCode(content,lang)+'</code></pre>';
  } else if(t==='text'&&content){
    body='<pre class="preview-text">'+h(content)+'</pre>';
  } else if(t==='sheet'&&content){
    body='<div class="preview-table">'+csvToTable(content)+'</div>';
  } else if(t==='image'){
    body='<div class="preview-image"><img src="/files/'+h(item.path)+'" alt="'+h(item.name)+'"></div>';
  } else if(t==='audio'){
    body='<div class="preview-audio"><div class="preview-audio-art">'+I.audio+'</div><div class="preview-audio-name">'+h(item.name)+'</div><div class="preview-audio-meta">'+fmtSize(item.size)+' \\u2014 '+h(item.content_type)+'</div><div class="preview-waveform">'+waveformBars()+'</div><div style="width:100%;display:flex;align-items:center;gap:12px;color:var(--text-2);font-size:13px;font-family:JetBrains Mono,monospace"><span>0:00</span><div style="flex:1;height:4px;background:var(--border);border-radius:2px"><div style="width:0%;height:100%;background:var(--text);border-radius:2px"></div></div><span>'+(item.size>5e6?'3:42':'0:48')+'</span></div><audio controls src="/files/'+h(item.path)+'" style="width:100%;margin-top:12px"></audio></div>';
  } else if(t==='video'){
    body='<div class="preview-video"><video controls src="/files/'+h(item.path)+'" style="width:100%;border-radius:4px"></video></div>';
  } else if(t==='doc'&&item.content_type==='application/pdf'){
    body='<iframe class="preview-iframe" src="/files/'+h(item.path)+'"></iframe>';
  } else if(t==='doc'){
    const ext=(item.name||'').split('.').pop()?.toLowerCase()||'';
    const labels={docx:'Word Document',xlsx:'Excel Spreadsheet',pptx:'PowerPoint Presentation',doc:'Word Document (Legacy)',xls:'Excel Spreadsheet (Legacy)',ppt:'PowerPoint (Legacy)'};
    const icons={docx:'<svg viewBox="0 0 24 24" fill="none" stroke="#4285F4" stroke-width="1.5"><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="9" y1="13" x2="15" y2="13"/><line x1="9" y1="17" x2="13" y2="17"/></svg>',xlsx:'<svg viewBox="0 0 24 24" fill="none" stroke="#34A853" stroke-width="1.5"><rect x="3" y="3" width="18" height="18" rx="2"/><line x1="3" y1="9" x2="21" y2="9"/><line x1="3" y1="15" x2="21" y2="15"/><line x1="9" y1="3" x2="9" y2="21"/><line x1="15" y1="3" x2="15" y2="21"/></svg>',pptx:'<svg viewBox="0 0 24 24" fill="none" stroke="#EA4335" stroke-width="1.5"><rect x="2" y="3" width="20" height="14" rx="2"/><line x1="8" y1="21" x2="16" y2="21"/><line x1="12" y1="17" x2="12" y2="21"/></svg>'};
    const typeLabel=labels[ext]||'Document';
    const typeIcon=icons[ext]||I.doc;
    body='<div class="preview-office"><div class="preview-office-icon">'+typeIcon+'</div><div class="preview-office-name">'+h(item.name)+'</div><div class="preview-office-type">'+typeLabel+'</div><div class="preview-office-size">'+fmtSize(item.size)+'</div><div class="preview-office-desc">'+(item.description?h(item.description):'')+'</div><button class="preview-office-dl" onclick="B.downloadFile(S.previewItem)">'+I.download+' Download to view</button></div>';
  } else if(t==='archive'){
    body='<div class="preview-doc"><div class="preview-doc-icon">'+I.archive+'</div><div class="preview-doc-name">'+h(item.name)+'</div><div class="preview-doc-meta">'+h(item.content_type)+'</div><div class="preview-doc-meta">'+fmtSize(item.size)+'</div></div>';
  } else {
    body='<div class="preview-generic"><div class="preview-generic-icon">'+fileIconHtml(item)+'</div><div class="preview-generic-name">'+h(item.name)+'</div><div class="preview-generic-meta">'+h(item.content_type||'Unknown')+' \\u2014 '+fmtSize(item.size)+'</div></div>';
  }

  const navL=hasPrev?'<button class="preview-nav preview-nav--prev" onclick="B.previewNav(-1)"><svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="15 18 9 12 15 6"/></svg></button>':'';
  const navR=hasNext?'<button class="preview-nav preview-nav--next" onclick="B.previewNav(1)"><svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="9 18 15 12 9 6"/></svg></button>':'';
  el.innerHTML='<div class="preview-header"><div class="preview-filename">'+fileIconHtml(item)+'<span>'+h(item.name)+'</span></div><div class="preview-info"><span>'+fmtSize(item.size)+'</span><span>'+fmtTime(item.updated_at)+'</span></div><div class="preview-actions"><button class="preview-btn" onclick="B.downloadFile(S.previewItem)">'+I.download+'</button><button class="preview-btn" onclick="B.closePreview()">'+I.x+'</button></div></div><div class="preview-body">'+body+'</div>'+navL+navR;
}

/* ── Data loading ─────────────────────────────────────────────────── */
async function loadItems(){
  S.selected.clear();
  let data;
  try{
    if(S.searchMode){
      data=await api.get('/drive/search?q='+encodeURIComponent(S.searchQ));
      S.items=data.items||[];
    } else if(S.section==='files'){
      data=await api.get('/folders/'+(S.path||''));
      S.items=data.items||[];
    } else if(S.section==='shared'){
      data=await api.get('/shared');
      S.items=(data.items||[]).map(i=>({...i,is_folder:false}));
    } else if(S.section==='recent'){
      data=await api.get('/drive/recent');
      S.items=data.items||[];
    } else if(S.section==='starred'){
      data=await api.get('/drive/starred');
      S.items=data.items||[];
    } else if(S.section==='trash'){
      data=await api.get('/drive/trash');
      S.items=data.items||[];
    }
  }catch(e){toast('Failed to load files','error');S.items=[]}
  sortItems();
  render();
}

async function loadStats(){
  try{S.stats=await api.get('/drive/stats')}catch{}
  renderSidebar();renderQuota();
}

function sortItems(){
  const {col,asc}=S.sort;
  S.items.sort((a,b)=>{
    // Folders first
    if(a.is_folder!==b.is_folder)return b.is_folder?1:-1;
    let v=0;
    if(col==='name')v=a.name.localeCompare(b.name);
    else if(col==='updated_at')v=(a.updated_at||0)-(b.updated_at||0);
    else if(col==='size')v=(a.size||0)-(b.size||0);
    return asc?v:-v;
  });
}

function render(){
  renderSidebar();renderToolbar();renderBulk();renderItems();renderDetail();renderQuota();
}

/* ── App controller (exposed as window.B) ─────────────────────────── */
const B=window.B={
  nav(path){
    S.path=path;S.section='files';S.searchMode=false;S.selected.clear();S.detail=null;S.detailOpen=false;
    history.pushState(null,'','/browse/'+(path||''));
    loadItems();
  },
  setSection(sec){
    S.section=sec;S.path='';S.searchMode=false;S.selected.clear();S.detail=null;S.detailOpen=false;
    // Close mobile sidebar
    $('sidebar').classList.remove('open');$('sidebar-backdrop').classList.remove('visible');
    loadItems();
  },
  setView(v){S.view=v;localStorage.setItem('sv',v);renderItems()},
  setSort(col){
    if(S.sort.col===col)S.sort.asc=!S.sort.asc;
    else{S.sort.col=col;S.sort.asc=true}
    localStorage.setItem('sc',col);localStorage.setItem('sa',S.sort.asc?'1':'0');
    sortItems();renderToolbar();renderItems();
  },
  toggleSort(){$('sort-dd')?.classList.toggle('open')},

  // Selection
  clickSel(path,e){
    if(e.ctrlKey||e.metaKey){
      if(S.selected.has(path))S.selected.delete(path);else S.selected.add(path);
    } else if(e.shiftKey&&S.selected.size){
      const paths=S.items.map(i=>i.path);
      const last=[...S.selected].pop();
      const a=paths.indexOf(last),b=paths.indexOf(path);
      const[lo,hi]=[Math.min(a,b),Math.max(a,b)];
      for(let i=lo;i<=hi;i++)S.selected.add(paths[i]);
    } else {
      S.selected.clear();S.selected.add(path);
    }
    // Show detail for single selection
    if(S.selected.size===1){
      S.detail=S.items.find(i=>i.path===[...S.selected][0])||null;
    }
    renderBulk();renderItems();renderDetail();
  },
  toggleSel(path,e){
    if(e.target.checked)S.selected.add(path);else S.selected.delete(path);
    renderBulk();renderItems();
  },
  selectAll(checked){
    if(checked)S.items.forEach(i=>S.selected.add(i.path));else S.selected.clear();
    renderBulk();renderItems();
  },
  clearSel(){S.selected.clear();renderBulk();renderItems()},

  // Context menu
  ctx(e,path){
    e.preventDefault();
    const item=S.items.find(i=>i.path===path)||null;
    if(item&&!S.selected.has(path)){S.selected.clear();S.selected.add(path);renderItems()}
    renderCtx(e.clientX,e.clientY,item);
  },

  // Star
  async star(path,v){
    await api.patch('/drive/star',{path,starred:v});
    const it=S.items.find(i=>i.path===path);
    if(it)it.starred=!!v;
    if(S.detail?.path===path)S.detail.starred=!!v;
    toast(v?'Starred':'Unstarred','success');
    renderItems();renderDetail();loadStats();
  },

  // Rename
  startRename(path){
    hideCtx();
    const item=S.items.find(i=>i.path===path);if(!item)return;
    const newName=prompt('Rename to:',item.name);
    if(!newName||newName===item.name)return;
    B.doRename(path,newName);
  },
  async doRename(path,newName){
    const res=await api.post('/drive/rename',{path,new_name:newName});
    if(res.error){toast(res.error.message||'Rename failed','error');return}
    toast('Renamed to '+newName,'success');loadItems();loadStats();
  },

  // Trash
  async trashItems(paths){
    hideCtx();
    await api.post('/drive/trash',{paths});
    toast(paths.length+' item(s) trashed','success');
    S.selected.clear();loadItems();loadStats();
  },
  async bulkTrash(){B.trashItems([...S.selected])},

  // Restore
  async restoreItems(paths){
    hideCtx();
    await api.post('/drive/restore',{paths});
    toast(paths.length+' item(s) restored','success');
    S.selected.clear();loadItems();loadStats();
  },
  async bulkRestore(){B.restoreItems([...S.selected])},

  // Empty trash
  async emptyTrash(){
    if(!confirm('Permanently delete all items in trash?'))return;
    await api.del('/drive/trash');
    toast('Trash emptied','success');loadItems();loadStats();
  },

  // Download
  downloadFile(item){
    if(!item||item.is_folder)return;
    const a=document.createElement('a');
    a.href='/files/'+item.path;a.download=item.name;a.click();
  },
  bulkDownload(){
    for(const p of S.selected){const it=S.items.find(i=>i.path===p);if(it&&!it.is_folder)B.downloadFile(it)}
  },

  // Star bulk
  async bulkStar(){
    for(const p of S.selected){await api.patch('/drive/star',{path:p,starred:1})}
    toast('Starred '+S.selected.size+' items','success');loadItems();
  },

  // Move
  showMoveModal(paths){
    hideCtx();
    showModal('Move to...','<div class="folder-tree" id="move-tree"><div class="tree-node selected" data-path="" onclick="B.selectMoveTarget(this,\\'\\')">/  (root)</div></div>','<button class="modal-btn modal-btn--secondary" onclick="hideModal()">Cancel</button><button class="modal-btn modal-btn--primary" onclick="B.doMove()">Move</button>');
    B._movePaths=paths;B._moveDest='';
    // Load folders
    api.get('/folders/').then(data=>{
      const tree=$('move-tree');
      if(!tree)return;
      let html='<div class="tree-node selected" data-path="" onclick="B.selectMoveTarget(this,\\'\\')">/  (root)</div>';
      for(const f of (data.items||[]).filter(i=>i.is_folder)){
        html+='<div class="tree-node" data-path="'+h(f.path)+'" onclick="B.selectMoveTarget(this,\\''+h(f.path)+'\\')"><span class="tree-icon">'+I.folder+'</span> '+h(f.name)+'</div>';
      }
      tree.innerHTML=html;
    });
  },
  selectMoveTarget(el,path){
    document.querySelectorAll('.tree-node').forEach(n=>n.classList.remove('selected'));
    el.classList.add('selected');B._moveDest=path;
  },
  async doMove(){
    hideModal();
    const res=await api.post('/drive/move',{paths:B._movePaths,destination:B._moveDest});
    if(res.error){toast(res.error.message||'Move failed','error');return}
    toast('Moved '+B._movePaths.length+' item(s)','success');
    S.selected.clear();loadItems();loadStats();
  },
  bulkMove(){B.showMoveModal([...S.selected])},

  // Copy
  async copyFile(path){
    hideCtx();
    const res=await api.post('/drive/copy',{path});
    if(res.error){toast(res.error.message||'Copy failed','error');return}
    toast('Copied as '+res.name,'success');loadItems();
  },

  // Share
  showShareModal(path){
    hideCtx();
    showModal('Share "'+h(path.split('/').pop())+'"','<div class="share-input-row"><input class="share-input" id="share-actor" placeholder="Actor name (e.g. a/agent-name)"><select class="share-perm-select" id="share-perm"><option value="read">Read</option><option value="write">Write</option></select><button class="modal-btn modal-btn--primary" onclick="B.doShare(\\''+h(path)+'\\')">Share</button></div><div class="share-list" id="share-list">Loading...</div><div style="margin-top:16px"><button class="copy-link-btn" onclick="B.copyLink(\\''+h(path)+'\\')">'+I.link+' Copy file path</button></div>','');
    B._sharePath=path;
    api.get('/shares').then(data=>{
      const list=$('share-list');if(!list)return;
      const given=(data.given||[]).filter(s=>s.path===path||s.object_path===path);
      if(!given.length){list.innerHTML='<div style="font-size:13px;color:var(--text-3)">Not shared with anyone yet</div>';return}
      list.innerHTML=given.map(s=>'<div class="share-entry"><div class="share-avatar">'+h((s.grantee||'?')[0].toUpperCase())+'</div><div class="share-name">'+h(s.grantee)+'</div><div class="share-role">'+h(s.permission)+'</div></div>').join('');
    });
  },
  async doShare(path){
    const actor=$('share-actor')?.value?.trim();
    const perm=$('share-perm')?.value;
    if(!actor){toast('Enter an actor name','error');return}
    const res=await api.post('/shares',{path,grantee:actor,permission:perm});
    if(res.error){toast(res.error.message||'Share failed','error');return}
    toast('Shared with '+actor,'success');
    B.showShareModal(path);// Refresh
  },
  copyLink(path){
    navigator.clipboard.writeText(window.location.origin+'/files/'+path);
    toast('Path copied to clipboard','success');
  },

  // Detail panel
  showDetail(path){
    hideCtx();
    S.detail=S.items.find(i=>i.path===path)||null;
    S.detailOpen=!!S.detail;S.detailTab='details';
    renderDetail();
  },
  closeDetail(){S.detailOpen=false;renderDetail()},
  detailTab(tab){S.detailTab=tab;renderDetail()},
  async saveDesc(path,desc){
    await api.patch('/drive/description',{path,description:desc});
  },

  async openPreview(path){
    const item=S.items.find(i=>i.path===path);
    if(!item||item.is_folder)return;
    S.previewItem=item;S.previewContent=null;S.previewLoading=true;
    renderPreview();
    const t=fileType(item);
    if(['code','text','sheet'].includes(t)||item.name.endsWith('.md')||item.content_type==='text/markdown'||(item.content_type||'').startsWith('text/')){
      try{const r=await fetch('/files/'+item.path);S.previewContent=await r.text()}catch{S.previewContent='Failed to load'}
    }
    S.previewLoading=false;
    renderPreview();
  },
  closePreview(){
    S.previewItem=null;S.previewContent=null;S.previewLoading=false;
    renderPreview();
  },
  previewNav(dir){
    const files=S.items.filter(i=>!i.is_folder);
    const idx=files.findIndex(i=>i.path===S.previewItem?.path);
    if(idx===-1)return;
    const next=files[idx+dir];
    if(next)B.openPreview(next.path);
  },

  // New folder
  async newFolder(){
    const name=prompt('Folder name:');if(!name)return;
    const res=await api.post('/folders',{path:S.path+name});
    if(res.error){toast(res.error.message||'Failed','error');return}
    toast('Folder created','success');loadItems();loadStats();
  },

  // Upload
  async uploadFiles(files){
    const panel=$('upload-panel');
    panel.classList.add('visible');
    const list=$('upload-list');
    S.uploading=[];
    for(const f of files){
      const entry={name:f.name,size:f.size,progress:0,status:'uploading',id:Math.random()};
      S.uploading.push(entry);
    }
    renderUploadList();
    $('upload-title').textContent='Uploading '+files.length+' file(s)...';

    for(let i=0;i<files.length;i++){
      const f=files[i];
      const path=S.path+f.name;
      try{
        await fetch('/files/'+path,{
          method:'PUT',
          headers:{'Content-Type':f.type||'application/octet-stream'},
          body:f,
        });
        S.uploading[i].status='done';S.uploading[i].progress=100;
      }catch{
        S.uploading[i].status='error';
      }
      renderUploadList();
    }
    $('upload-title').textContent='Upload complete';
    toast(files.length+' file(s) uploaded','success');
    loadItems();loadStats();
  },

  // Search
  search(q){
    S.searchQ=q;
    if(!q){S.searchMode=false;S.section='files';loadItems();return}
    S.searchMode=true;loadItems();
  },

  // Paste (for clipboard operations)
  async paste(){
    hideCtx();
    if(!S.clipboard)return;
    if(S.clipboard.action==='copy'){
      for(const p of S.clipboard.paths)await api.post('/drive/copy',{path:p,destination:S.path});
    } else {
      await api.post('/drive/move',{paths:S.clipboard.paths,destination:S.path});
    }
    S.clipboard=null;toast('Pasted','success');loadItems();
  },
};

/* ── Upload list render ───────────────────────────────────────────── */
function renderUploadList(){
  const list=$('upload-list');
  list.innerHTML=S.uploading.map(u=>{
    const st=u.status==='done'?'upload-item-status--done':u.status==='error'?'upload-item-status--error':'';
    const fill=u.status==='error'?'upload-item-fill--error':'';
    return '<div class="upload-item"><div class="upload-item-info"><div class="upload-item-name">'+h(u.name)+'</div><div class="upload-item-bar"><div class="upload-item-fill '+fill+'" style="width:'+(u.status==='done'?100:u.status==='error'?100:30)+'%"></div></div></div><span class="upload-item-status '+st+'">'+(u.status==='done'?'✓':u.status==='error'?'✗':'...')+'</span></div>';
  }).join('');
}

/* ── Modal helpers ────────────────────────────────────────────────── */
function showModal(title,body,footer){
  const m=$('modal-overlay');
  m.innerHTML='<div class="modal"><div class="modal-header"><span class="modal-title">'+title+'</span><button class="modal-close" onclick="hideModal()">'+I.x+'</button></div><div class="modal-body">'+body+'</div>'+(footer?'<div class="modal-footer">'+footer+'</div>':'')+'</div>';
  m.classList.add('visible');
}
function hideModal(){$('modal-overlay').classList.remove('visible');$('modal-overlay').innerHTML=''}
window.showModal=showModal;window.hideModal=hideModal;

/* ── Context menu helpers ─────────────────────────────────────────── */
function hideCtx(){$('ctx-menu').classList.remove('visible')}
document.addEventListener('click',()=>hideCtx());
document.addEventListener('click',()=>$('sort-dd')?.classList.remove('open'));

// Empty-area context menu
$('file-content').addEventListener('contextmenu',e=>{
  if(!e.target.closest('.file-row')&&!e.target.closest('.grid-card')){
    e.preventDefault();renderCtx(e.clientX,e.clientY,null);
  }
});

/* ── Keyboard shortcuts ───────────────────────────────────────────── */
document.addEventListener('keydown',e=>{
  const tag=e.target.tagName;
  if(tag==='INPUT'||tag==='TEXTAREA'||tag==='SELECT')return;

  // Preview navigation
  if(S.previewItem){
    if(e.key==='Escape'){B.closePreview();return}
    if(e.key==='ArrowLeft'){B.previewNav(-1);return}
    if(e.key==='ArrowRight'){B.previewNav(1);return}
  }

  if(e.key==='?'){e.preventDefault();showShortcuts();return}
  if(e.key==='/'||e.key==='f'&&(e.ctrlKey||e.metaKey)){e.preventDefault();$('search-input').focus();return}
  if(e.key==='Escape'){S.selected.clear();S.detailOpen=false;hideCtx();hideModal();render();return}

  const sel=[...S.selected];
  if(e.key==='Delete'||e.key==='Backspace'){if(sel.length&&S.section!=='trash'){B.trashItems(sel)}return}
  if(e.key==='F2'&&sel.length===1){B.startRename(sel[0]);return}
  if(e.key==='s'||e.key==='S'){if(sel.length===1){const it=S.items.find(i=>i.path===sel[0]);if(it)B.star(sel[0],it.starred?0:1)}return}
  if(e.key==='i'||e.key==='I'){if(sel.length===1){B.showDetail(sel[0])}else{S.detailOpen=false;renderDetail()}return}
  if(e.key==='a'&&(e.ctrlKey||e.metaKey)){e.preventDefault();B.selectAll(true);return}
  if(e.key==='c'&&(e.ctrlKey||e.metaKey)){if(sel.length){S.clipboard={action:'copy',paths:sel};toast('Copied to clipboard','info')}return}
  if(e.key==='x'&&(e.ctrlKey||e.metaKey)){if(sel.length){S.clipboard={action:'cut',paths:sel};toast('Cut to clipboard','info')}return}
  if(e.key==='v'&&(e.ctrlKey||e.metaKey)){if(S.clipboard)B.paste();return}
  if(e.key==='Enter'){if(sel.length===1){const it=S.items.find(i=>i.path===sel[0]);if(it?.is_folder)B.nav(it.path);else if(it)B.openPreview(it.path)}return}
  if(e.key===' '&&!S.previewItem){
    e.preventDefault();
    if(sel.length===1){const it=S.items.find(i=>i.path===sel[0]);if(it&&!it.is_folder)B.openPreview(it.path)}
    return;
  }

  // Arrow navigation
  if(e.key==='ArrowDown'||e.key==='ArrowUp'){
    e.preventDefault();
    const paths=S.items.map(i=>i.path);
    const cur=sel.length?paths.indexOf(sel[sel.length-1]):-1;
    const next=e.key==='ArrowDown'?Math.min(cur+1,paths.length-1):Math.max(cur-1,0);
    S.selected.clear();S.selected.add(paths[next]);
    S.detail=S.items[next];
    renderBulk();renderItems();renderDetail();
    // Scroll into view
    document.querySelector('.file-row.selected,.grid-card.selected')?.scrollIntoView({block:'nearest'});
    return;
  }
});

function showShortcuts(){
  const shortcuts=[
    ['?','Show shortcuts'],['/', 'Search'],['Esc','Deselect / Close'],['↑ ↓','Navigate'],
    ['Enter','Open'],['Space','Preview'],['← →','Prev/Next preview'],['Del','Trash'],['F2','Rename'],['S','Star/Unstar'],
    ['I','Details panel'],['⌘A','Select all'],['⌘C','Copy'],['⌘X','Cut'],
    ['⌘V','Paste'],['',''],
  ];
  const grid=shortcuts.map(([k,d])=>'<div class="shortcut-item"><span class="shortcut-desc">'+d+'</span><span class="shortcut-keys"><span class="kbd">'+k+'</span></span></div>').join('');
  showModal('Keyboard Shortcuts','<div class="shortcuts-grid">'+grid+'</div>','');
}

/* ── Search ────────────────────────────────────────────────────────── */
let searchTimer=null;
$('search-input').addEventListener('input',e=>{
  clearTimeout(searchTimer);
  searchTimer=setTimeout(()=>B.search(e.target.value),300);
});
$('search-input').addEventListener('keydown',e=>{if(e.key==='Escape'){e.target.value='';B.search('')}});

/* ── Drag and drop ────────────────────────────────────────────────── */
let dragCount=0;
const main=$('main');
main.addEventListener('dragenter',e=>{e.preventDefault();dragCount++;$('drop-overlay').classList.add('active')});
main.addEventListener('dragleave',e=>{e.preventDefault();dragCount--;if(dragCount<=0){$('drop-overlay').classList.remove('active');dragCount=0}});
main.addEventListener('dragover',e=>e.preventDefault());
main.addEventListener('drop',e=>{
  e.preventDefault();$('drop-overlay').classList.remove('active');dragCount=0;
  if(e.dataTransfer?.files?.length)B.uploadFiles(e.dataTransfer.files);
});

/* ── File input ───────────────────────────────────────────────────── */
$('file-input').addEventListener('change',e=>{if(e.target.files.length)B.uploadFiles(e.target.files);e.target.value=''});

/* ── Upload panel close ───────────────────────────────────────────── */
$('upload-close').addEventListener('click',()=>$('upload-panel').classList.remove('visible'));

/* ── Sidebar clicks ───────────────────────────────────────────────── */
$('sidebar-nav').addEventListener('click',e=>{
  const el=e.target.closest('.sidebar-item');
  if(el)B.setSection(el.dataset.section);
});

/* ── Mobile sidebar toggle ────────────────────────────────────────── */
$('mobile-toggle')?.addEventListener('click',()=>{
  $('sidebar').classList.toggle('open');
  $('sidebar-backdrop').classList.toggle('visible');
});
$('sidebar-backdrop').addEventListener('click',()=>{
  $('sidebar').classList.remove('open');
  $('sidebar-backdrop').classList.remove('visible');
});

/* ── Theme toggle ─────────────────────────────────────────────────── */
$('theme-btn').addEventListener('click',()=>{
  const d=document.documentElement.classList.toggle('dark');
  localStorage.setItem('theme',d?'dark':'light');
});
(function(){const s=localStorage.getItem('theme');
  if(s==='light')document.documentElement.classList.remove('dark');
  else if(!s&&!window.matchMedia('(prefers-color-scheme:dark)').matches)document.documentElement.classList.remove('dark')})();

/* ── History ──────────────────────────────────────────────────────── */
window.addEventListener('popstate',()=>{
  const p=decodeURIComponent(location.pathname.replace('/browse/','').replace('/browse',''));
  S.path=p;S.section='files';S.searchMode=false;loadItems();
});

/* ── Init ─────────────────────────────────────────────────────────── */
const initPath=decodeURIComponent(location.pathname.replace('/browse/','').replace('/browse',''));
S.path=initPath==='browse'?'':initPath;
loadItems();loadStats();

})();
</script>
</body></html>`;
}
