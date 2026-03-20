/* ══════════════════════════════════════════════════════════════════════
   storage.now — Browse JS (Minimal Redesign)
   No sidebar. No navbar. Just files.
   Handles both demo (unauthenticated) and auth (logged-in) modes.
   Config injected via window.__BROWSE_CONFIG from the HTML template.
   ══════════════════════════════════════════════════════════════════════ */
(function(){
'use strict';

const CONFIG = window.__BROWSE_CONFIG || { mode: 'demo' };
const DEMO = CONFIG.mode === 'demo';
const ACTOR = CONFIG.actor || null;

/* ── Demo filesystem data ──────────────────────────────────────────── */
const now=Date.now(),h1=3600000,d1=86400000;
const DEMO_FS=DEMO?[
  {path:'documents/',name:'documents',is_folder:true,content_type:'',size:0,starred:true,description:'Work documents and notes',created_at:now-30*d1,updated_at:now-2*d1},
  {path:'images/',name:'images',is_folder:true,content_type:'',size:0,starred:false,description:'Photos and graphics',created_at:now-25*d1,updated_at:now-1*d1},
  {path:'projects/',name:'projects',is_folder:true,content_type:'',size:0,starred:true,description:'Code projects',created_at:now-20*d1,updated_at:now-3*h1},
  {path:'shared/',name:'shared',is_folder:true,content_type:'',size:0,starred:false,description:'Files shared with collaborators',created_at:now-15*d1,updated_at:now-5*h1},
  {path:'backups/',name:'backups',is_folder:true,content_type:'',size:0,starred:false,description:'System backups',created_at:now-10*d1,updated_at:now-7*d1},
  {path:'README.md',name:'README.md',is_folder:false,content_type:'text/markdown',size:2048,starred:true,description:'Project overview',created_at:now-30*d1,updated_at:now-1*d1},
  {path:'notes.txt',name:'notes.txt',is_folder:false,content_type:'text/plain',size:847,starred:false,description:'Quick notes',created_at:now-5*d1,updated_at:now-2*h1},
  {path:'budget-2026.csv',name:'budget-2026.csv',is_folder:false,content_type:'text/csv',size:15360,starred:false,description:'Annual budget planning',created_at:now-8*d1,updated_at:now-3*d1},
  {path:'documents/proposal.pdf',name:'proposal.pdf',is_folder:false,content_type:'application/pdf',size:2457600,starred:true,description:'Client proposal Q1',created_at:now-12*d1,updated_at:now-2*d1},
  {path:'documents/meeting-notes.md',name:'meeting-notes.md',is_folder:false,content_type:'text/markdown',size:4096,starred:false,description:'Weekly standup notes',created_at:now-3*d1,updated_at:now-3*h1},
  {path:'documents/contracts/',name:'contracts',is_folder:true,content_type:'',size:0,starred:false,description:'Legal contracts',created_at:now-20*d1,updated_at:now-5*d1},
  {path:'documents/templates/',name:'templates',is_folder:true,content_type:'',size:0,starred:false,description:'Document templates',created_at:now-18*d1,updated_at:now-8*d1},
  {path:'documents/report-final.docx',name:'report-final.docx',is_folder:false,content_type:'application/vnd.openxmlformats-officedocument.wordprocessingml.document',size:892416,starred:false,description:'Annual report',created_at:now-7*d1,updated_at:now-4*d1},
  {path:'documents/slides-keynote.pptx',name:'slides-keynote.pptx',is_folder:false,content_type:'application/vnd.openxmlformats-officedocument.presentationml.presentation',size:5242880,starred:false,description:'Conference presentation',created_at:now-6*d1,updated_at:now-2*d1},
  {path:'documents/contracts/nda-acme.pdf',name:'nda-acme.pdf',is_folder:false,content_type:'application/pdf',size:184320,starred:false,description:'NDA with Acme Corp',created_at:now-15*d1,updated_at:now-15*d1},
  {path:'documents/contracts/sow-2026.pdf',name:'sow-2026.pdf',is_folder:false,content_type:'application/pdf',size:256000,starred:false,description:'Statement of work',created_at:now-10*d1,updated_at:now-5*d1},
  {path:'documents/templates/invoice.docx',name:'invoice.docx',is_folder:false,content_type:'application/vnd.openxmlformats-officedocument.wordprocessingml.document',size:45056,starred:false,description:'Invoice template',created_at:now-18*d1,updated_at:now-18*d1},
  {path:'documents/templates/letterhead.docx',name:'letterhead.docx',is_folder:false,content_type:'application/vnd.openxmlformats-officedocument.wordprocessingml.document',size:32768,starred:false,description:'Company letterhead',created_at:now-18*d1,updated_at:now-12*d1},
  {path:'documents/analytics.xlsx',name:'analytics.xlsx',is_folder:false,content_type:'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',size:245760,starred:false,description:'Monthly analytics data',created_at:now-9*d1,updated_at:now-2*d1},
  {path:'images/hero-banner.png',name:'hero-banner.png',is_folder:false,content_type:'image/png',size:3145728,starred:false,description:'Website hero image',created_at:now-10*d1,updated_at:now-1*d1},
  {path:'images/team-photo.jpg',name:'team-photo.jpg',is_folder:false,content_type:'image/jpeg',size:4718592,starred:true,description:'Team offsite 2026',created_at:now-8*d1,updated_at:now-8*d1},
  {path:'images/logo.svg',name:'logo.svg',is_folder:false,content_type:'image/svg+xml',size:8192,starred:false,description:'Company logo vector',created_at:now-20*d1,updated_at:now-3*d1},
  {path:'images/screenshots/',name:'screenshots',is_folder:true,content_type:'',size:0,starred:false,description:'Product screenshots',created_at:now-7*d1,updated_at:now-1*d1},
  {path:'images/icons/',name:'icons',is_folder:true,content_type:'',size:0,starred:false,description:'App icons',created_at:now-12*d1,updated_at:now-6*d1},
  {path:'images/screenshots/dashboard-v2.png',name:'dashboard-v2.png',is_folder:false,content_type:'image/png',size:1572864,starred:false,description:'Dashboard redesign',created_at:now-5*d1,updated_at:now-1*d1},
  {path:'images/screenshots/mobile-app.png',name:'mobile-app.png',is_folder:false,content_type:'image/png',size:2097152,starred:false,description:'Mobile app capture',created_at:now-3*d1,updated_at:now-3*d1},
  {path:'projects/api-server/',name:'api-server',is_folder:true,content_type:'',size:0,starred:true,description:'Main API backend',created_at:now-20*d1,updated_at:now-3*h1},
  {path:'projects/landing-page/',name:'landing-page',is_folder:true,content_type:'',size:0,starred:false,description:'Marketing website',created_at:now-15*d1,updated_at:now-1*d1},
  {path:'projects/ml-pipeline/',name:'ml-pipeline',is_folder:true,content_type:'',size:0,starred:false,description:'ML training pipeline',created_at:now-8*d1,updated_at:now-2*d1},
  {path:'projects/api-server/main.go',name:'main.go',is_folder:false,content_type:'text/x-go',size:4096,starred:false,description:'Server entrypoint',created_at:now-20*d1,updated_at:now-3*h1},
  {path:'projects/api-server/go.mod',name:'go.mod',is_folder:false,content_type:'text/plain',size:512,starred:false,description:'Go module file',created_at:now-20*d1,updated_at:now-5*d1},
  {path:'projects/api-server/Dockerfile',name:'Dockerfile',is_folder:false,content_type:'text/plain',size:1024,starred:false,description:'Container build file',created_at:now-18*d1,updated_at:now-2*d1},
  {path:'projects/api-server/config.yaml',name:'config.yaml',is_folder:false,content_type:'text/yaml',size:2048,starred:false,description:'Service configuration',created_at:now-15*d1,updated_at:now-6*h1},
  {path:'projects/landing-page/index.html',name:'index.html',is_folder:false,content_type:'text/html',size:8192,starred:false,description:'Homepage HTML',created_at:now-15*d1,updated_at:now-1*d1},
  {path:'projects/landing-page/style.css',name:'style.css',is_folder:false,content_type:'text/css',size:12288,starred:false,description:'Main stylesheet',created_at:now-15*d1,updated_at:now-1*d1},
  {path:'projects/landing-page/app.js',name:'app.js',is_folder:false,content_type:'application/javascript',size:6144,starred:false,description:'Client JS bundle',created_at:now-10*d1,updated_at:now-2*d1},
  {path:'projects/ml-pipeline/train.py',name:'train.py',is_folder:false,content_type:'text/x-python',size:16384,starred:false,description:'Model training script',created_at:now-8*d1,updated_at:now-2*d1},
  {path:'projects/ml-pipeline/requirements.txt',name:'requirements.txt',is_folder:false,content_type:'text/plain',size:256,starred:false,description:'Python dependencies',created_at:now-8*d1,updated_at:now-5*d1},
  {path:'projects/ml-pipeline/model-v3.bin',name:'model-v3.bin',is_folder:false,content_type:'application/octet-stream',size:52428800,starred:true,description:'Trained model checkpoint',created_at:now-4*d1,updated_at:now-2*d1},
  {path:'shared/design-system.fig',name:'design-system.fig',is_folder:false,content_type:'application/octet-stream',size:8388608,starred:false,description:'Figma design file (from u/alice)',created_at:now-6*d1,updated_at:now-1*d1},
  {path:'shared/quarterly-review.pdf',name:'quarterly-review.pdf',is_folder:false,content_type:'application/pdf',size:1048576,starred:false,description:'Q4 review deck (from a/reports-bot)',created_at:now-3*d1,updated_at:now-3*d1},
  {path:'shared/api-spec.yaml',name:'api-spec.yaml',is_folder:false,content_type:'text/yaml',size:32768,starred:false,description:'OpenAPI spec (from u/bob)',created_at:now-10*d1,updated_at:now-5*d1},
  {path:'backups/db-2026-03-01.sql.gz',name:'db-2026-03-01.sql.gz',is_folder:false,content_type:'application/gzip',size:15728640,starred:false,description:'Database backup March 1',created_at:now-18*d1,updated_at:now-18*d1},
  {path:'backups/db-2026-03-15.sql.gz',name:'db-2026-03-15.sql.gz',is_folder:false,content_type:'application/gzip',size:16777216,starred:false,description:'Database backup March 15',created_at:now-4*d1,updated_at:now-4*d1},
  {path:'backups/config-snapshot.tar.gz',name:'config-snapshot.tar.gz',is_folder:false,content_type:'application/gzip',size:524288,starred:false,description:'Configuration snapshot',created_at:now-7*d1,updated_at:now-7*d1},
  {path:'media/',name:'media',is_folder:true,content_type:'',size:0,starred:false,description:'Audio and video files',created_at:now-12*d1,updated_at:now-2*d1},
  {path:'media/podcast-episode.mp3',name:'podcast-episode.mp3',is_folder:false,content_type:'audio/mpeg',size:8388608,starred:false,description:'Weekly podcast episode 12',created_at:now-5*d1,updated_at:now-5*d1},
  {path:'media/product-demo.mp4',name:'product-demo.mp4',is_folder:false,content_type:'video/mp4',size:26214400,starred:true,description:'Product demo recording',created_at:now-3*d1,updated_at:now-3*d1},
  {path:'media/notification.wav',name:'notification.wav',is_folder:false,content_type:'audio/wav',size:524288,starred:false,description:'App notification sound',created_at:now-8*d1,updated_at:now-8*d1},
  {path:'media/team-standup.webm',name:'team-standup.webm',is_folder:false,content_type:'video/webm',size:15728640,starred:false,description:'Weekly standup recording',created_at:now-2*d1,updated_at:now-2*d1},
  {path:'old-logo.png',name:'old-logo.png',is_folder:false,content_type:'image/png',size:1048576,starred:false,description:'Deprecated logo',created_at:now-30*d1,updated_at:now-20*d1,trashed_at:now-2*d1},
  {path:'draft-v1.md',name:'draft-v1.md',is_folder:false,content_type:'text/markdown',size:3072,starred:false,description:'First draft (superseded)',created_at:now-15*d1,updated_at:now-10*d1,trashed_at:now-1*d1},
]:[];

const DEMO_SHARED=DEMO?[
  {owner:'u/alice',path:'design-system.fig',name:'design-system.fig',is_folder:false,content_type:'application/octet-stream',size:8388608,permission:'editor',updated_at:now-1*d1},
  {owner:'a/reports-bot',path:'quarterly-review.pdf',name:'quarterly-review.pdf',is_folder:false,content_type:'application/pdf',size:1048576,permission:'viewer',updated_at:now-3*d1},
  {owner:'u/bob',path:'api-spec.yaml',name:'api-spec.yaml',is_folder:false,content_type:'text/yaml',size:32768,permission:'viewer',updated_at:now-5*d1},
  {owner:'u/alice',path:'wireframes/',name:'wireframes',is_folder:true,content_type:'',size:0,permission:'viewer',updated_at:now-2*d1},
]:[];

const DEMO_STATS=DEMO?{total_size:192923648,file_count:37,folder_count:13,trash_count:2,quota:5368709120}:null;

const DEMO_CONTENT=DEMO?{
'README.md':'# Storage Platform\n\nA modern file storage and sharing platform built for teams.\n\n## Features\n\n- **Fast uploads** with resumable chunked transfer\n- **Real-time collaboration** with shared folders\n- **Version history** for all file types\n- **Full-text search** across documents\n\n## Getting Started\n\n```bash\nnpm install\nnpm run dev\n```\n\nVisit http://localhost:3000 to open the dashboard.\n\n## Architecture\n\n- Frontend: TypeScript + Hono\n- Storage: R2 object storage\n- Auth: Session-based with OAuth\n\n## API\n\n| Endpoint | Method | Description |\n|----------|--------|-------------|\n| /files/* | GET | Download file |\n| /files/* | PUT | Upload file |\n| /folders/* | GET | List folder |\n| /drive/search | GET | Search files |\n\n> **Note**: All endpoints require authentication except /browse.\n\n---\n\n*Built with Mizu framework.*',
'notes.txt':'Meeting notes - March 2026\n\n- Discuss Q1 roadmap priorities\n- Review storage architecture proposal\n- Plan migration from legacy system\n- Budget allocation for cloud infra\n\nAction items:\n1. Draft RFC for new object storage layer\n2. Benchmark R2 vs S3 performance\n3. Set up CI/CD pipeline for staging\n4. Update documentation for API v2\n\nNext meeting: Friday 3pm',
'budget-2026.csv':'Category,Q1,Q2,Q3,Q4,Total\nInfrastructure,45000,48000,52000,55000,200000\nPersonnel,120000,120000,125000,130000,495000\nSoftware Licenses,15000,15000,18000,18000,66000\nMarketing,25000,30000,35000,40000,130000\nTravel,8000,10000,12000,8000,38000\nTraining,5000,8000,5000,8000,26000\nMiscellaneous,3000,3000,3000,3000,12000\nTotal,221000,234000,250000,262000,967000',
'documents/meeting-notes.md':'# Weekly Standup Notes\n\n## March 18, 2026\n\n### Completed\n- Migrated auth service to new OAuth provider\n- Fixed file upload timeout for files > 50MB\n- Updated API documentation\n\n### In Progress\n- Storage layer refactoring (70% complete)\n- Mobile responsive redesign\n- Performance optimization for large folders\n\n### Blockers\n- Waiting on security review for sharing feature\n- Need design approval for new file preview UI\n\n## March 11, 2026\n\n### Completed\n- Set up staging environment\n- Implemented file versioning backend\n- Code review for permission system',
'projects/api-server/main.go':'package main\n\nimport (\n\t"context"\n\t"log"\n\t"net/http"\n\t"os"\n\t"os/signal"\n\t"time"\n\n\t"github.com/example/storage/internal/api"\n\t"github.com/example/storage/internal/config"\n\t"github.com/example/storage/internal/storage"\n)\n\nfunc main() {\n\tcfg := config.Load()\n\n\tstore, err := storage.New(cfg.StorageBackend, cfg.StorageBucket)\n\tif err != nil {\n\t\tlog.Fatalf("storage init: %v", err)\n\t}\n\tdefer store.Close()\n\n\trouter := api.NewRouter(store, cfg)\n\n\tsrv := &http.Server{\n\t\tAddr:         cfg.ListenAddr,\n\t\tHandler:      router,\n\t\tReadTimeout:  30 * time.Second,\n\t\tWriteTimeout: 60 * time.Second,\n\t\tIdleTimeout:  120 * time.Second,\n\t}\n\n\tgo func() {\n\t\tlog.Printf("listening on %s", cfg.ListenAddr)\n\t\tif err := srv.ListenAndServe(); err != http.ErrServerClosed {\n\t\t\tlog.Fatalf("server error: %v", err)\n\t\t}\n\t}()\n\n\tquit := make(chan os.Signal, 1)\n\tsignal.Notify(quit, os.Interrupt)\n\t<-quit\n\n\tctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)\n\tdefer cancel()\n\tsrv.Shutdown(ctx)\n\tlog.Println("server stopped")\n}',
'projects/api-server/Dockerfile':'FROM golang:1.22-alpine AS builder\n\nWORKDIR /app\nCOPY go.mod go.sum ./\nRUN go mod download\nCOPY . .\nRUN CGO_ENABLED=0 GOOS=linux go build -o /bin/server ./cmd/server\n\nFROM alpine:3.19\nRUN apk --no-cache add ca-certificates\nCOPY --from=builder /bin/server /bin/server\nEXPOSE 8080\nCMD ["/bin/server"]',
'projects/api-server/config.yaml':'server:\n  listen: ":8080"\n  read_timeout: 30s\n  write_timeout: 60s\n\nstorage:\n  backend: r2\n  bucket: storage-files\n  max_upload_size: 104857600\n  presign_expiry: 3600\n\nauth:\n  session_secret: "${SESSION_SECRET}"\n  oauth_provider: github\n  allowed_origins:\n    - "https://storage.example.com"\n    - "http://localhost:3000"\n\ndatabase:\n  driver: d1\n  name: storage-db\n\nlogging:\n  level: info\n  format: json',
'projects/landing-page/index.html':'<!DOCTYPE html>\n<html lang="en">\n<head>\n  <meta charset="utf-8">\n  <meta name="viewport" content="width=device-width,initial-scale=1">\n  <title>Storage Platform</title>\n  <link rel="stylesheet" href="/style.css">\n</head>\n<body>\n  <nav class="nav">\n    <a href="/" class="logo">storage.now</a>\n    <a href="/docs">Docs</a>\n    <a href="/pricing">Pricing</a>\n    <a href="/login" class="btn">Sign In</a>\n  </nav>\n\n  <main class="hero">\n    <h1>File storage for developers</h1>\n    <p>Upload, organize, and share files with a simple API.</p>\n    <a href="/signup" class="btn btn--primary">Get Started Free</a>\n  </main>\n</body>\n</html>',
'projects/landing-page/style.css':'* { box-sizing: border-box; margin: 0; padding: 0; }\n\nbody {\n  font-family: system-ui, sans-serif;\n  color: #18181B;\n  background: #FAFAF9;\n}\n\n.nav {\n  display: flex;\n  align-items: center;\n  gap: 24px;\n  padding: 16px 32px;\n  border-bottom: 1px solid #E4E4E7;\n}\n\n.logo {\n  font-weight: 700;\n  font-size: 18px;\n  margin-right: auto;\n}\n\n.hero {\n  text-align: center;\n  padding: 120px 32px;\n}\n\n.hero h1 {\n  font-size: 48px;\n  font-weight: 800;\n  letter-spacing: -0.02em;\n}\n\n.hero p {\n  font-size: 18px;\n  color: #52525B;\n  margin: 16px 0 32px;\n}\n\n.btn {\n  padding: 10px 24px;\n  border: 1px solid #E4E4E7;\n  background: none;\n  font-size: 14px;\n  cursor: pointer;\n}\n\n.btn--primary {\n  background: #18181B;\n  color: #FAFAF9;\n  border-color: #18181B;\n}',
'projects/ml-pipeline/train.py':'import torch\nimport torch.nn as nn\nfrom torch.utils.data import DataLoader\nfrom pathlib import Path\nimport logging\n\nlogging.basicConfig(level=logging.INFO)\nlogger = logging.getLogger(__name__)\n\nclass StorageModel(nn.Module):\n    def __init__(self, input_dim=512, hidden_dim=256, output_dim=128):\n        super().__init__()\n        self.encoder = nn.Sequential(\n            nn.Linear(input_dim, hidden_dim),\n            nn.ReLU(),\n            nn.Dropout(0.2),\n            nn.Linear(hidden_dim, hidden_dim),\n            nn.ReLU(),\n            nn.Linear(hidden_dim, output_dim),\n        )\n\n    def forward(self, x):\n        return self.encoder(x)\n\ndef train_epoch(model, loader, optimizer, criterion, device):\n    model.train()\n    total_loss = 0\n    for batch_idx, (data, target) in enumerate(loader):\n        data, target = data.to(device), target.to(device)\n        optimizer.zero_grad()\n        output = model(data)\n        loss = criterion(output, target)\n        loss.backward()\n        optimizer.step()\n        total_loss += loss.item()\n    return total_loss / len(loader)\n\ndef main():\n    device = torch.device("cuda" if torch.cuda.is_available() else "cpu")\n    logger.info(f"Using device: {device}")\n\n    model = StorageModel().to(device)\n    optimizer = torch.optim.Adam(model.parameters(), lr=1e-3)\n    criterion = nn.MSELoss()\n\n    for epoch in range(100):\n        loss = train_epoch(model, train_loader, optimizer, criterion, device)\n        if epoch % 10 == 0:\n            logger.info(f"Epoch {epoch}: loss={loss:.4f}")\n\n    torch.save(model.state_dict(), "model-v3.bin")\n    logger.info("Model saved")\n\nif __name__ == "__main__":\n    main()',
}:null;

const DEMO_MEDIA=DEMO?{
  'media/podcast-episode.mp3':'https://upload.wikimedia.org/wikipedia/commons/b/bb/Test_ogg_mp3_48kbps.wav',
  'media/product-demo.mp4':'https://storage.googleapis.com/gtv-videos-bucket/sample/ForBiggerBlazes.mp4',
  'media/notification.wav':'https://upload.wikimedia.org/wikipedia/commons/b/bb/Test_ogg_mp3_48kbps.wav',
  'media/team-standup.webm':'https://storage.googleapis.com/gtv-videos-bucket/sample/ForBiggerBlazes.mp4',
}:null;

const DEMO_DOCS=DEMO?{
  'documents/proposal.pdf':{title:'Client Proposal Q1 2026',subtitle:'Storage Platform Solutions',author:'Engineering Team',date:'March 2026',pages:[
    {type:'title'},{type:'content',title:'Executive Summary',body:'<p>This proposal outlines our recommended approach for implementing a modern file storage and sharing platform. The solution leverages edge computing with Cloudflare Workers and R2 object storage to deliver sub-50ms response times globally.</p><h3>Key Benefits</h3><ul><li>99.99% uptime SLA with global edge distribution</li><li>End-to-end encryption for all stored files</li><li>Granular access controls with role-based permissions</li><li>AI-powered file organization and search</li></ul>'},
    {type:'content',title:'Technical Architecture',body:'<h3>Storage Layer</h3><p>Files are stored in R2 object storage with automatic replication across multiple regions. Metadata is managed through D1 SQLite databases co-located with Workers for minimal latency.</p><h3>API Design</h3><table><tr><th>Endpoint</th><th>Method</th><th>Description</th></tr><tr><td>/files/*</td><td>PUT</td><td>Upload file</td></tr><tr><td>/files/*</td><td>GET</td><td>Download file</td></tr><tr><td>/folders/*</td><td>GET</td><td>List contents</td></tr><tr><td>/shares</td><td>POST</td><td>Share file</td></tr></table>'},
    {type:'content',title:'Timeline & Budget',body:'<h3>Phase 1: Core Platform (4 weeks)</h3><ul><li>File upload/download with presigned URLs</li><li>Folder organization and metadata</li><li>Basic authentication and sessions</li></ul><h3>Phase 2: Collaboration (3 weeks)</h3><ul><li>File sharing with granular permissions</li><li>Shared folders and team workspaces</li></ul><h3>Phase 3: Intelligence (3 weeks)</h3><ul><li>Full-text search across documents</li><li>AI-powered file categorization</li><li>Smart duplicate detection</li></ul><h3>Budget Estimate</h3><table><tr><th>Item</th><th>Cost</th></tr><tr><td>Development</td><td>$45,000</td></tr><tr><td>Infrastructure (annual)</td><td>$2,400</td></tr><tr><td>Total First Year</td><td>$47,400</td></tr></table>'},
  ]},
  'documents/report-final.docx':{title:'Annual Report 2025',header:'Storage Platform Inc. — Confidential',footer:'Page {n} of {total}',body:'<h1>Annual Report 2025</h1><h2>Company Overview</h2><p>Storage Platform Inc. provides enterprise-grade file storage and collaboration tools built on edge computing infrastructure. Our platform serves over 10,000 active users across 45 countries.</p><h2>Key Metrics</h2><table><tr><th>Metric</th><th>2024</th><th>2025</th><th>Growth</th></tr><tr><td>Active Users</td><td>6,200</td><td>10,400</td><td>+67%</td></tr><tr><td>Files Stored</td><td>2.1M</td><td>5.8M</td><td>+176%</td></tr><tr><td>Storage Volume</td><td>4.2 TB</td><td>12.6 TB</td><td>+200%</td></tr><tr><td>API Requests/day</td><td>850K</td><td>2.4M</td><td>+182%</td></tr></table><h2>Product Highlights</h2><h3>Q1: File Versioning</h3><p>Launched automatic version history for all file types, allowing users to restore any previous version within 30 days.</p><h3>Q2: AI-Powered Search</h3><p>Introduced semantic search capabilities, enabling users to find files by describing their content in natural language.</p><h3>Q3: Team Workspaces</h3><p>Released collaborative workspaces with real-time presence indicators and granular permission controls.</p><h3>Q4: Edge Optimization</h3><p>Deployed to 200+ edge locations worldwide, reducing average response time from 120ms to 35ms.</p><h2>Financial Summary</h2><table><tr><th>Category</th><th>Amount</th></tr><tr><td>Revenue</td><td>$1,240,000</td></tr><tr><td>Operating Costs</td><td>$680,000</td></tr><tr><td>Net Income</td><td>$560,000</td></tr></table>'},
  'documents/analytics.xlsx':{sheets:[{name:'Overview',data:[['Month','Users','Files','Storage (GB)','API Calls'],['Jan',8200,4200000,9800,52000000],['Feb',8600,4500000,10200,58000000],['Mar',9100,4800000,10800,62000000],['Apr',9400,5000000,11200,65000000],['May',9800,5300000,11800,70000000],['Jun',10400,5800000,12600,76000000]]},{name:'Revenue',data:[['Month','Subscriptions','Enterprise','API','Total'],['Jan',68000,42000,8000,118000],['Feb',72000,42000,9200,123200],['Mar',78000,45000,10800,133800],['Apr',82000,48000,11000,141000],['May',88000,52000,12400,152400],['Jun',95000,58000,14000,167000]]}]},
  'documents/slides-keynote.pptx':{slides:[
    {title:'Storage Platform',subtitle:'Product Vision 2026',type:'title'},
    {title:'The Problem',body:'<ul><li>Files scattered across multiple services</li><li>No unified search or organization</li><li>Sharing is complex and insecure</li><li>AI tools cannot access your files</li></ul>'},
    {title:'Our Solution',body:'<ul><li>One platform for all file types</li><li>API-first design for integration</li><li>Granular sharing with audit trails</li><li>AI-native: files as context for agents</li></ul>'},
    {title:'Traction',body:'<div class="metric"><span class="metric-value">10,400</span><span class="metric-label">Active Users</span></div><div class="metric"><span class="metric-value">5.8M</span><span class="metric-label">Files Stored</span></div><div class="metric"><span class="metric-value">12.6 TB</span><span class="metric-label">Total Storage</span></div><div class="metric"><span class="metric-value">99.99%</span><span class="metric-label">Uptime</span></div>'},
    {title:'Roadmap 2026',body:'<ul><li><strong>Q1:</strong> Real-time collaboration</li><li><strong>Q2:</strong> AI assistant integration</li><li><strong>Q3:</strong> Enterprise SSO & compliance</li><li><strong>Q4:</strong> On-premise deployment option</li></ul>'},
  ]},
}:null;

/* ── Helpers ──────────────────────────────────────────────────────── */
function $(id){return document.getElementById(id)}
function h(s){return String(s).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;')}
function fmtSize(b){if(!b)return'\u2014';if(b<1024)return b+' B';if(b<1048576)return(b/1024).toFixed(1)+' KB';if(b<1073741824)return(b/1048576).toFixed(1)+' MB';return(b/1073741824).toFixed(1)+' GB'}
function fmtTime(ts){if(!ts)return'\u2014';var d=Date.now()-ts;if(d<60000)return'now';if(d<3600000)return Math.floor(d/60000)+'m';if(d<86400000)return Math.floor(d/3600000)+'h';if(d<2592000000)return Math.floor(d/86400000)+'d';return new Date(ts).toLocaleDateString('en',{month:'short',day:'numeric'})}

/* ── Icons ────────────────────────────────────────────────────────── */
var I={
  folder:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z"/></svg>',
  file:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>',
  image:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="3" y="3" width="18" height="18" rx="2"/><circle cx="8.5" cy="8.5" r="1.5"/><polyline points="21 15 16 10 5 21"/></svg>',
  video:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polygon points="23 7 16 12 23 17 23 7"/><rect x="1" y="5" width="15" height="14" rx="2"/></svg>',
  audio:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M9 18V5l12-2v13"/><circle cx="6" cy="18" r="3"/><circle cx="18" cy="16" r="3"/></svg>',
  doc:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/><line x1="10" y1="9" x2="8" y2="9"/></svg>',
  code:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/></svg>',
  archive:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polyline points="21 8 21 21 3 21 3 8"/><rect x="1" y="3" width="22" height="5"/><line x1="10" y1="12" x2="14" y2="12"/></svg>',
  sheet:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="3" y="3" width="18" height="18" rx="2"/><line x1="3" y1="9" x2="21" y2="9"/><line x1="3" y1="15" x2="21" y2="15"/><line x1="9" y1="3" x2="9" y2="21"/></svg>',
  text:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="17" y1="10" x2="3" y2="10"/><line x1="21" y1="6" x2="3" y2="6"/><line x1="21" y1="14" x2="3" y2="14"/><line x1="17" y1="18" x2="3" y2="18"/></svg>',
  star:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/></svg>',
  starFill:'<svg viewBox="0 0 24 24" fill="currentColor" stroke="currentColor" stroke-width="1.5"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/></svg>',
  trash:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/></svg>',
  download:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>',
  upload:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/></svg>',
  share:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="18" cy="5" r="3"/><circle cx="6" cy="12" r="3"/><circle cx="18" cy="19" r="3"/><line x1="8.59" y1="13.51" x2="15.42" y2="17.49"/><line x1="15.41" y1="6.51" x2="8.59" y2="10.49"/></svg>',
  rename:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M17 3a2.83 2.83 0 114 4L7.5 20.5 2 22l1.5-5.5L17 3z"/></svg>',
  move:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M5 12h14M12 5l7 7-7 7"/></svg>',
  copy:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1"/></svg>',
  home:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M3 9l9-7 9 7v11a2 2 0 01-2 2H5a2 2 0 01-2-2z"/></svg>',
  shared:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M17 21v-2a4 4 0 00-4-4H5a4 4 0 00-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 00-3-3.87"/><path d="M16 3.13a4 4 0 010 7.75"/></svg>',
  clock:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>',
  grid:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="14" y="14" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/></svg>',
  list:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="8" y1="6" x2="21" y2="6"/><line x1="8" y1="12" x2="21" y2="12"/><line x1="8" y1="18" x2="21" y2="18"/><line x1="3" y1="6" x2="3.01" y2="6"/><line x1="3" y1="12" x2="3.01" y2="12"/><line x1="3" y1="18" x2="3.01" y2="18"/></svg>',
  plus:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>',
  restore:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polyline points="1 4 1 10 7 10"/><path d="M3.51 15a9 9 0 102.13-9.36L1 10"/></svg>',
  info:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg>',
  x:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>',
  link:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M10 13a5 5 0 007.54.54l3-3a5 5 0 00-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 00-7.54-.54l-3 3a5 5 0 007.07 7.07l1.71-1.71"/></svg>',
  search:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>',
  arrowL:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="19" y1="12" x2="5" y2="12"/><polyline points="12 19 5 12 12 5"/></svg>',
  arrowR:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="5" y1="12" x2="19" y2="12"/><polyline points="12 5 19 12 12 19"/></svg>',
  more:'<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="1"/><circle cx="19" cy="12" r="1"/><circle cx="5" cy="12" r="1"/></svg>',
};

/* ── State ────────────────────────────────────────────────────────── */
var S={
  path:'',
  view:localStorage.getItem('bv')||'list',
  sort:JSON.parse(localStorage.getItem('bs')||'null')||{col:'name',asc:true},
  items:[],selected:new Set(),
  section:'files',
  clipboard:null,
  stats:DEMO_STATS,
  searchMode:false,searchQ:'',
  uploading:[],
  previewItem:null,previewContent:null,previewLoading:false,
  _mediaCleanup:null,
};
window.S=S;

/* ── File type detection ──────────────────────────────────────────── */
function fileType(item){
  if(item.is_folder)return'folder';
  var ct=item.content_type||'',n=(item.name||'').toLowerCase();
  if(ct.startsWith('image/')||/\.(png|jpe?g|gif|svg|webp|bmp|ico)$/.test(n))return'image';
  if(ct.startsWith('video/')||/\.(mp4|webm|mov|avi|mkv)$/.test(n))return'video';
  if(ct.startsWith('audio/')||/\.(mp3|wav|ogg|flac|aac|m4a)$/.test(n))return'audio';
  if(/\.(pdf)$/.test(n)||ct==='application/pdf')return'doc';
  if(/\.(docx?|odt|rtf)$/.test(n)||ct.includes('wordprocessing'))return'doc';
  if(/\.(xlsx?|ods)$/.test(n)||ct.includes('spreadsheet'))return'sheet';
  if(/\.(pptx?|odp)$/.test(n)||ct.includes('presentation'))return'doc';
  if(/\.(zip|tar|gz|rar|7z|bz2)$/.test(n)||ct.includes('zip')||ct.includes('gzip'))return'archive';
  if(/\.(js|ts|jsx|tsx|py|go|rs|rb|php|java|c|cpp|h|cs|swift|kt|sh|bash|zsh|fish|ps1|sql|r|m|scala|lua|pl|ex|exs|hs|ml|clj|erl|elm|v|zig|nim|d|f90|asm|wasm)$/.test(n))return'code';
  if(/\.(json|ya?ml|toml|ini|env|cfg|conf|xml|html?|css|scss|less|sass|vue|svelte|astro|mdx?)$/.test(n))return'code';
  if(/\.(csv|tsv)$/.test(n)||ct==='text/csv')return'sheet';
  if(/\.(txt|log|md|rst|tex)$/.test(n)||ct.startsWith('text/'))return'text';
  if(ct.includes('json')||ct.includes('xml')||ct.includes('yaml'))return'code';
  if(/Dockerfile|Makefile|Gemfile|Rakefile|Vagrantfile|Procfile|\.gitignore|\.dockerignore|\.editorconfig|go\.mod|go\.sum|Cargo\.toml|requirements\.txt|package\.json|tsconfig/.test(n))return'code';
  return'file';
}

function fileIconHtml(item){
  var t=fileType(item);
  var cls=t==='folder'?'file-icon file-icon--folder':'file-icon';
  var icon=I[t]||I.file;
  return '<div class="'+cls+'">'+icon+'</div>';
}

/* ── Syntax highlighting ──────────────────────────────────────────── */
function langFromName(n){
  var ext=(n||'').split('.').pop().toLowerCase();
  var map={js:'js',jsx:'js',ts:'js',tsx:'js',mjs:'js',cjs:'js',json:'json',py:'py',go:'go',rs:'rs',rb:'rb',java:'java',c:'c',cpp:'c',h:'c',cs:'cs',swift:'swift',kt:'kt',sh:'sh',bash:'sh',zsh:'sh',sql:'sql',html:'html',htm:'html',xml:'xml',svg:'xml',css:'css',scss:'css',less:'css',yaml:'yaml',yml:'yaml',toml:'toml',md:'md',dockerfile:'sh',makefile:'sh'};
  return map[ext]||(/Dockerfile/i.test(n)?'sh':/Makefile/i.test(n)?'sh':null);
}

function highlightCode(code,lang){
  var s=h(code);
  if(!lang)return s.split('\\n').map(function(l){return'<span class="line">'+l+'</span>'}).join('\\n');
  var rules=[];
  if(lang==='js'||lang==='ts'||lang==='json')rules=[
    [/\b(const|let|var|function|return|if|else|for|while|do|switch|case|break|continue|new|this|class|extends|import|export|from|default|async|await|try|catch|throw|typeof|instanceof|in|of|yield|void|delete|true|false|null|undefined)\b/g,'tok-kw'],
    [/(\/\/.*$|\/\*[\s\S]*?\*\/)/gm,'tok-cm'],
    [/("(?:\\\\[\\s\\S]|[^"\\\\])*"|'(?:\\\\[\\s\\S]|[^'\\\\])*'|`(?:\\\\[\\s\\S]|[^`\\\\])*`)/g,'tok-str'],
    [/\b(\d+\.?\d*(?:e[+-]?\d+)?)\b/gi,'tok-num'],
    [/\b([A-Z][a-zA-Z0-9]*)\b/g,'tok-type'],
    [/\b(\w+)(?=\s*\()/g,'tok-fn'],
  ];
  else if(lang==='py')rules=[
    [/\b(def|class|return|if|elif|else|for|while|break|continue|import|from|as|try|except|raise|with|yield|lambda|pass|True|False|None|and|or|not|in|is|global|nonlocal|async|await)\b/g,'tok-kw'],
    [/(#.*$)/gm,'tok-cm'],
    [/("""[\s\S]*?"""|'''[\s\S]*?'''|"(?:\\\\[\\s\\S]|[^"\\\\])*"|'(?:\\\\[\\s\\S]|[^'\\\\])*')/g,'tok-str'],
    [/\b(\d+\.?\d*(?:e[+-]?\d+)?)\b/gi,'tok-num'],
    [/\b(\w+)(?=\s*\()/g,'tok-fn'],
    [/@(\w+)/g,'tok-fn'],
  ];
  else if(lang==='go')rules=[
    [/\b(package|import|func|return|if|else|for|range|switch|case|default|break|continue|go|defer|chan|select|type|struct|interface|map|var|const|true|false|nil|string|int|int8|int16|int32|int64|uint|uint8|uint16|uint32|uint64|float32|float64|bool|byte|rune|error|make|new|len|cap|append|copy|delete|panic|recover)\b/g,'tok-kw'],
    [/(\/\/.*$|\/\*[\s\S]*?\*\/)/gm,'tok-cm'],
    [/("(?:\\\\[\\s\\S]|[^"\\\\])*"|`[^`]*`)/g,'tok-str'],
    [/\b(\d+\.?\d*(?:e[+-]?\d+)?)\b/gi,'tok-num'],
    [/\b([A-Z][a-zA-Z0-9]*)\b/g,'tok-type'],
    [/\b(\w+)(?=\s*\()/g,'tok-fn'],
  ];
  else if(lang==='html'||lang==='xml')rules=[
    [/(<!--[\s\S]*?-->)/g,'tok-cm'],
    [/(<\/?[a-zA-Z][a-zA-Z0-9-]*)/g,'tok-tag'],
    [/\b([a-zA-Z-]+)(=)/g,'tok-attr'],
    [/("(?:[^"\\\\]|\\\\.)*"|'(?:[^'\\\\]|\\\\.)*')/g,'tok-str'],
  ];
  else if(lang==='css'||lang==='scss')rules=[
    [/(\/\*[\s\S]*?\*\/)/g,'tok-cm'],
    [/([.#][a-zA-Z_][a-zA-Z0-9_-]*)/g,'tok-fn'],
    [/\b([\d.]+(?:px|em|rem|%|vh|vw|deg|s|ms)?)\b/g,'tok-num'],
    [/("(?:[^"\\\\]|\\\\.)*"|'(?:[^'\\\\]|\\\\.)*')/g,'tok-str'],
    [/@[a-zA-Z-]+/g,'tok-kw'],
    [/([a-zA-Z-]+)(?=\s*:)/g,'tok-attr'],
  ];
  else if(lang==='sh')rules=[
    [/(#.*$)/gm,'tok-cm'],
    [/("(?:\\\\[\\s\\S]|[^"\\\\])*"|'[^']*')/g,'tok-str'],
    [/\b(if|then|else|elif|fi|for|while|do|done|case|esac|in|function|return|exit|echo|export|source|local|readonly|set|unset|shift|trap|exec|eval|cd|pwd|test|true|false)\b/g,'tok-kw'],
    [/\$[a-zA-Z_][a-zA-Z0-9_]*/g,'tok-type'],
    [/\$\{[^}]*\}/g,'tok-type'],
  ];
  else if(lang==='sql')rules=[
    [/(--.*$)/gm,'tok-cm'],
    [/\b(SELECT|FROM|WHERE|INSERT|INTO|VALUES|UPDATE|SET|DELETE|CREATE|DROP|ALTER|TABLE|INDEX|VIEW|JOIN|LEFT|RIGHT|INNER|OUTER|ON|AND|OR|NOT|IN|IS|NULL|AS|ORDER|BY|GROUP|HAVING|LIMIT|OFFSET|UNION|ALL|DISTINCT|COUNT|SUM|AVG|MIN|MAX|BETWEEN|LIKE|EXISTS|CASE|WHEN|THEN|ELSE|END|PRIMARY|KEY|FOREIGN|REFERENCES|CONSTRAINT|DEFAULT|CHECK|UNIQUE|CASCADE|TRIGGER|FUNCTION|PROCEDURE|BEGIN|COMMIT|ROLLBACK|GRANT|REVOKE|INTEGER|TEXT|REAL|BLOB|VARCHAR|BOOLEAN|DATE|TIMESTAMP)\b/gi,'tok-kw'],
    [/('(?:[^'\\\\]|\\\\.)*')/g,'tok-str'],
    [/\b(\d+\.?\d*)\b/g,'tok-num'],
  ];
  else if(lang==='rs')rules=[
    [/\b(fn|let|mut|const|if|else|for|while|loop|match|return|break|continue|struct|enum|impl|trait|type|pub|use|mod|crate|self|super|as|in|ref|move|async|await|dyn|where|true|false|Some|None|Ok|Err|Self|Box|Vec|String|Option|Result|i8|i16|i32|i64|i128|u8|u16|u32|u64|u128|f32|f64|bool|char|str|usize|isize)\b/g,'tok-kw'],
    [/(\/\/.*$|\/\*[\s\S]*?\*\/)/gm,'tok-cm'],
    [/("(?:\\\\[\\s\\S]|[^"\\\\])*")/g,'tok-str'],
    [/\b(\d+\.?\d*(?:e[+-]?\d+)?)\b/gi,'tok-num'],
    [/\b([A-Z][a-zA-Z0-9]*)\b/g,'tok-type'],
    [/\b(\w+)(?=\s*[({<])/g,'tok-fn'],
  ];
  else rules=[
    [/(\/\/.*$|#.*$|\/\*[\s\S]*?\*\/)/gm,'tok-cm'],
    [/("(?:\\\\[\\s\\S]|[^"\\\\])*"|'(?:\\\\[\\s\\S]|[^'\\\\])*'|`[^`]*`)/g,'tok-str'],
    [/\b(\d+\.?\d*)\b/g,'tok-num'],
    [/\b(\w+)(?=\s*\()/g,'tok-fn'],
  ];
  var tokens=[];
  rules.forEach(function(r){s.replace(r[0],function(m){var i=arguments[arguments.length-2];tokens.push({start:i,end:i+m.length,cls:r[1],text:m});return m})});
  tokens.sort(function(a,b){return a.start-b.start||b.end-a.end});
  var out='',pos=0,used=[];
  tokens.forEach(function(t){if(t.start<pos)return;if(used.some(function(u){return t.start<u}))return;out+=s.slice(pos,t.start)+'<span class="'+t.cls+'">'+t.text+'</span>';pos=t.end;used.push(t.end)});
  out+=s.slice(pos);
  return out.split('\\n').map(function(l){return'<span class="line">'+l+'</span>'}).join('\\n');
}

/* ── Markdown renderer ────────────────────────────────────────────── */
function renderMarkdown(md){
  var html=md
    .replace(/```(\w*)\n([\s\S]*?)```/g,function(_,lang,code){return'<div class="md-code-block"><code>'+h(code.trim())+'</code></div>'})
    .replace(/`([^`]+)`/g,'<code>$1</code>')
    .replace(/^######\s+(.+)$/gm,'<h6>$1</h6>').replace(/^#####\s+(.+)$/gm,'<h5>$1</h5>')
    .replace(/^####\s+(.+)$/gm,'<h4>$1</h4>').replace(/^###\s+(.+)$/gm,'<h3>$1</h3>')
    .replace(/^##\s+(.+)$/gm,'<h2>$1</h2>').replace(/^#\s+(.+)$/gm,'<h1>$1</h1>')
    .replace(/^\*\*\*$|^---$|^___$/gm,'<hr>')
    .replace(/\*\*(.+?)\*\*/g,'<strong>$1</strong>')
    .replace(/\*(.+?)\*/g,'<em>$1</em>')
    .replace(/!\[([^\]]*)\]\(([^)]+)\)/g,'<img src="$2" alt="$1">')
    .replace(/\[([^\]]+)\]\(([^)]+)\)/g,'<a href="$2">$1</a>')
    .replace(/^\s*>\s+(.+)$/gm,'<blockquote>$1</blockquote>')
    .replace(/^- \[x\]\s+(.+)$/gm,'<li class="task-done"><input type="checkbox" checked disabled> $1</li>')
    .replace(/^- \[ \]\s+(.+)$/gm,'<li class="task"><input type="checkbox" disabled> $1</li>')
    .replace(/^\s*[-*]\s+(.+)$/gm,'<li>$1</li>')
    .replace(/(<li>[\s\S]*?<\/li>)/g,'<ul>$1</ul>')
    .replace(/<\/ul>\s*<ul>/g,'')
    .replace(/\n\|(.+)\|\s*\n\|[-\s|:]+\|\s*\n((?:\|.+\|\s*\n)*)/g,function(_,hdr,body){
      var ths=hdr.split('|').map(function(c){return'<th>'+c.trim()+'</th>'}).join('');
      var rows=body.trim().split('\\n').map(function(r){return'<tr>'+r.split('|').filter(Boolean).map(function(c){return'<td>'+c.trim()+'</td>'}).join('')+'</tr>'}).join('');
      return'<table><thead><tr>'+ths+'</tr></thead><tbody>'+rows+'</tbody></table>';
    })
    .replace(/\n{2,}/g,'</p><p>')
    .replace(/^(?!<[hublotip])/gm,function(m,o,s){return s[o-1]===undefined||s[o-1]==='\\n'?'<p>':m});
  return html;
}

function csvToTable(csv){
  var rows=csv.trim().split('\\n').map(function(r){
    var cols=[];var cur='';var inQ=false;
    for(var i=0;i<r.length;i++){
      if(r[i]==='"'){inQ=!inQ}else if(r[i]===','&&!inQ){cols.push(cur.trim());cur=''}else{cur+=r[i]}
    }
    cols.push(cur.trim());return cols;
  });
  if(!rows.length)return'<p>Empty file</p>';
  var hdr=rows[0],body=rows.slice(1);
  var out='<div class="preview-table"><table><thead><tr>'+hdr.map(function(c){return'<th>'+h(c)+'</th>'}).join('')+'</tr></thead><tbody>';
  body.forEach(function(r){out+='<tr>'+r.map(function(c){return'<td>'+h(c)+'</td>'}).join('')+'</tr>'});
  return out+'</tbody></table></div>';
}

/* ── Document renderers ───────────────────────────────────────────── */
function renderPdfPages(doc){
  return'<div class="preview-pdf">'+doc.pages.map(function(p,i){
    if(p.type==='title')return'<div class="pdf-page"><div class="pdf-page-title">'+h(doc.title)+'</div><div class="pdf-page-subtitle">'+h(doc.subtitle||'')+'</div><div class="pdf-page-meta">'+h(doc.author||'')+'</div><div class="pdf-page-meta">'+h(doc.date||'')+'</div><div class="pdf-page-num">'+(i+1)+'</div></div>';
    return'<div class="pdf-page"><h2>'+h(p.title||'')+'</h2>'+p.body+'<div class="pdf-page-num">'+(i+1)+'</div></div>';
  }).join('')+'</div>';
}

function renderDocxPage(doc){
  return'<div class="preview-docx"><div class="docx-ruler">1 &middot; 2 &middot; 3 &middot; 4 &middot; 5 &middot; 6</div><div class="docx-page">'+(doc.header?'<div class="docx-header">'+h(doc.header)+'</div>':'')+doc.body+(doc.footer?'<div class="docx-footer">'+doc.footer.replace('{n}','1').replace('{total}','1')+'</div>':'')+'</div></div>';
}

function renderXlsxSheet(doc){
  var sheets=doc.sheets;
  var tabsH=sheets.map(function(s,i){return'<div class="xlsx-tab'+(i===0?' active':'')+'" onclick="var ts=this.parentElement.children;for(var j=0;j<ts.length;j++){ts[j].classList.remove(\\x27active\\x27)};this.classList.add(\\x27active\\x27);var gs=this.parentElement.nextElementSibling.children;for(var j=0;j<gs.length;j++){gs[j].style.display=j==='+i+'?\\x27block\\x27:\\x27none\\x27}">'+h(s.name)+'</div>'}).join('');
  var grids=sheets.map(function(s,si){
    var d=s.data,hdr=d[0],body=d.slice(1);
    var t='<div style="'+(si>0?'display:none':'')+'" class="xlsx-grid"><table><thead><tr><th></th>'+hdr.map(function(c){return'<th>'+h(c)+'</th>'}).join('')+'</tr></thead><tbody>';
    body.forEach(function(r,ri){t+='<tr><td class="xlsx-row-num">'+(ri+1)+'</td>'+r.map(function(c){return'<td>'+h(c)+'</td>'}).join('')+'</tr>'});
    return t+'</tbody></table></div>';
  }).join('');
  return'<div class="preview-xlsx"><div class="xlsx-tabs">'+tabsH+'</div>'+grids+'</div>';
}

function renderPptxSlides(doc){
  var thumbs=doc.slides.map(function(s,i){return'<div class="pptx-thumb'+(i===0?' active':'')+'" onclick="B.showSlide('+i+')"><div class="pptx-thumb-num">'+(i+1)+'</div><div class="pptx-thumb-title">'+h(s.title)+'</div></div>'}).join('');
  var slides=doc.slides.map(function(s,i){
    var cls='pptx-slide'+(i===0?' active':'')+(s.type==='title'?' pptx-slide--title':'');
    return'<div class="'+cls+'" id="pptx-slide-'+i+'"><div class="pptx-slide-title">'+h(s.title)+'</div><div class="pptx-slide-body">'+(s.body||h(s.subtitle||''))+'</div></div>';
  }).join('');
  return'<div class="preview-pptx"><div class="pptx-thumbs">'+thumbs+'</div><div class="pptx-stage" id="pptx-stage">'+slides+'</div></div>';
}

/* ── Media player ─────────────────────────────────────────────────── */
function setupMedia(type,item){
  if(type==='audio')setupAudio(item);
  else if(type==='video')setupVideo(item);
}

function setupAudio(item){
  var src=DEMO?(DEMO_MEDIA[item.path]||''):'/files/'+encodeURIComponent(item.path);
  var el=$('mp-audio');if(!el)return;
  var audio=new Audio(src);
  var play=$('mp-play'),time=$('mp-time'),dur=$('mp-dur'),prog=$('mp-progress'),fill=$('mp-fill'),thumb=$('mp-thumb'),wave=$('mp-wave');
  var bars=wave?wave.children:[];
  function fmtT(s){if(isNaN(s))return'0:00';var m=Math.floor(s/60),ss=Math.floor(s%60);return m+':'+(ss<10?'0':'')+ss}
  function updateWave(){
    if(!bars.length)return;
    var pct=audio.duration?audio.currentTime/audio.duration:0;
    for(var i=0;i<bars.length;i++){
      var h=20+Math.sin(i*0.3+audio.currentTime*2)*18;
      bars[i].style.height=h+'px';
      bars[i].className=i/bars.length<=pct?'mp-wave-bar active':'mp-wave-bar';
    }
    if(!audio.paused)requestAnimationFrame(updateWave);
  }
  if(play)play.onclick=function(){audio.paused?audio.play():audio.pause()};
  audio.onplay=function(){if(play)play.innerHTML=I.x.replace('viewBox','viewBox');updateWave()};
  audio.onpause=function(){if(play)play.innerHTML='<svg viewBox="0 0 24 24" fill="currentColor"><polygon points="5 3 19 12 5 21 5 3"/></svg>'};
  audio.onended=function(){if(play)play.innerHTML='<svg viewBox="0 0 24 24" fill="currentColor"><polygon points="5 3 19 12 5 21 5 3"/></svg>'};
  audio.onloadedmetadata=function(){if(dur)dur.textContent=fmtT(audio.duration)};
  audio.ontimeupdate=function(){if(time)time.textContent=fmtT(audio.currentTime);if(fill&&audio.duration)fill.style.width=(audio.currentTime/audio.duration*100)+'%'};
  if(prog)prog.onclick=function(e){var r=prog.getBoundingClientRect();audio.currentTime=((e.clientX-r.left)/r.width)*audio.duration};
  var vol=$('mp-vol');
  if(vol)vol.onclick=function(e){var r=vol.getBoundingClientRect();audio.volume=Math.max(0,Math.min(1,(e.clientX-r.left)/r.width));vol.querySelector('.mp-vol-fill').style.width=(audio.volume*100)+'%'};
  S._mediaCleanup=function(){audio.pause();audio.src=''};
}

function setupVideo(item){
  var src=DEMO?(DEMO_MEDIA[item.path]||''):'/files/'+encodeURIComponent(item.path);
  var el=$('mp-video');if(!el)return;
  var viewport=$('mp-viewport');
  var video=document.createElement('video');video.src=src;video.preload='metadata';
  if(viewport){var first=viewport.firstChild;if(first)viewport.insertBefore(video,first);else viewport.appendChild(video)}
  var play=$('mp-play'),playBig=$('mp-play-big'),time=$('mp-time'),dur=$('mp-dur'),prog=$('mp-progress'),fill=$('mp-fill'),fs=$('mp-fs');
  function fmtT(s){if(isNaN(s))return'0:00';var m=Math.floor(s/60),ss=Math.floor(s%60);return m+':'+(ss<10?'0':'')+ss}
  function doPlay(){video.paused?video.play():video.pause()}
  if(play)play.onclick=doPlay;
  if(playBig)playBig.onclick=function(){video.play();playBig.style.display='none'};
  video.onplay=function(){if(play)play.innerHTML=I.x.replace('viewBox','viewBox');if(playBig)playBig.style.display='none'};
  video.onpause=function(){if(play)play.innerHTML='<svg viewBox="0 0 24 24" fill="currentColor"><polygon points="5 3 19 12 5 21 5 3"/></svg>';if(playBig)playBig.style.display='flex'};
  video.onended=function(){if(playBig)playBig.style.display='flex'};
  video.onloadedmetadata=function(){if(dur)dur.textContent=fmtT(video.duration)};
  video.ontimeupdate=function(){if(time)time.textContent=fmtT(video.currentTime);if(fill&&video.duration)fill.style.width=(video.currentTime/video.duration*100)+'%'};
  if(prog)prog.onclick=function(e){var r=prog.getBoundingClientRect();video.currentTime=((e.clientX-r.left)/r.width)*video.duration};
  if(fs)fs.onclick=function(){if(viewport.requestFullscreen)viewport.requestFullscreen();else if(viewport.webkitRequestFullscreen)viewport.webkitRequestFullscreen()};
  var vol=$('mp-vol');
  if(vol)vol.onclick=function(e){var r=vol.getBoundingClientRect();video.volume=Math.max(0,Math.min(1,(e.clientX-r.left)/r.width));vol.querySelector('.mp-vol-fill').style.width=(video.volume*100)+'%'};
  S._mediaCleanup=function(){video.pause();video.src=''};
}

/* ── API ──────────────────────────────────────────────────────────── */
var api=DEMO?null:{
  get:function(u){return fetch(u).then(function(r){return r.json()})},
  post:function(u,b){return fetch(u,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(b)}).then(function(r){return r.json()})},
  patch:function(u,b){return fetch(u,{method:'PATCH',headers:{'Content-Type':'application/json'},body:JSON.stringify(b)}).then(function(r){return r.json()})},
  del:function(u){return fetch(u,{method:'DELETE'}).then(function(r){return r.json()})},
};

/* ── Toast ────────────────────────────────────────────────────────── */
function toast(msg,type){
  var t=document.createElement('div');t.className='toast toast--'+(type||'info');t.textContent=msg;
  $('toasts').appendChild(t);setTimeout(function(){t.remove()},3000);
}
function requireSignup(){toast('Sign up free to use this feature','info')}

/* ── Data loading ─────────────────────────────────────────────────── */
function loadItems(){S.selected.clear();DEMO?loadDemoItems():loadApiItems()}

function loadDemoItems(){
  var sec=S.section;
  if(S.searchMode){S.items=DEMO_FS.filter(function(f){return!f.trashed_at&&f.name.toLowerCase().includes(S.searchQ.toLowerCase())});sortItems();render();return}
  if(sec==='files'){
    var p=S.path;
    S.items=DEMO_FS.filter(function(f){
      if(f.trashed_at)return false;if(!f.path.startsWith(p))return false;
      var rest=f.path.slice(p.length);if(!rest)return false;
      if(f.is_folder)return rest.replace(/\/$/,'').indexOf('/')===-1;
      return rest.indexOf('/')===-1;
    });
  }else if(sec==='shared'){S.items=DEMO_SHARED.slice()}
  else if(sec==='recent'){S.items=DEMO_FS.filter(function(f){return!f.is_folder&&!f.trashed_at}).sort(function(a,b){return b.updated_at-a.updated_at}).slice(0,10)}
  else if(sec==='starred'){S.items=DEMO_FS.filter(function(f){return f.starred&&!f.trashed_at})}
  else if(sec==='trash'){S.items=DEMO_FS.filter(function(f){return!!f.trashed_at})}
  sortItems();render();
}

function loadApiItems(){
  var sec=S.section,p=S.path;
  var url;
  if(S.searchMode)url='/drive/search?q='+encodeURIComponent(S.searchQ);
  else if(sec==='files')url=p?'/folders/'+encodeURIComponent(p):'/folders/';
  else if(sec==='shared')url='/shared';
  else if(sec==='recent')url='/drive/recent';
  else if(sec==='starred')url='/drive/starred';
  else if(sec==='trash')url='/drive/trash';
  api.get(url).then(function(d){S.items=d.items||d.files||[];sortItems();render()}).catch(function(){toast('Failed to load','err');S.items=[];render()});
}

function loadStats(){
  if(DEMO){S.stats=DEMO_STATS;renderFoot();return}
  api.get('/drive/stats').then(function(d){S.stats=d;renderFoot()}).catch(function(){});
}

function sortItems(){
  var col=S.sort.col,asc=S.sort.asc;
  S.items.sort(function(a,b){
    if(a.is_folder!==b.is_folder)return a.is_folder?-1:1;
    var va,vb;
    if(col==='name'){va=(a.name||'').toLowerCase();vb=(b.name||'').toLowerCase();return asc?va.localeCompare(vb):vb.localeCompare(va)}
    if(col==='size'){va=a.size||0;vb=b.size||0;}
    else{va=a.updated_at||0;vb=b.updated_at||0;}
    return asc?va-vb:vb-va;
  });
}

/* ══════════════════════════════════════════════════════════════════════
   RENDERING
   ══════════════════════════════════════════════════════════════════════ */
function render(){renderMain()}

/* ── Tabs ─────────────────────────────────────────────────────────── */
function renderTabs(){
  var t=$('tabs');if(!t)return;
  var sections=[
    {id:'files',label:'files',icon:I.folder},
    {id:'recent',label:'recent',icon:I.clock},
    {id:'starred',label:'starred',icon:I.star},
    {id:'shared',label:'shared',icon:I.shared},
    {id:'trash',label:'trash',icon:I.trash},
  ];
  var counts=S.stats||{};
  t.innerHTML=sections.map(function(s){
    var cls='tab'+(S.section===s.id?' active':'');
    var badge='';
    if(s.id==='trash'&&counts.trash_count)badge='<span class="tab-count">'+counts.trash_count+'</span>';
    return'<button class="'+cls+'" data-sec="'+s.id+'">'+s.label+badge+'</button>';
  }).join('');
}

/* ── Main ─────────────────────────────────────────────────────────── */
function renderMain(){
  if(S._mediaCleanup){S._mediaCleanup();S._mediaCleanup=null}
  var m=$('main');
  if(S.previewItem){
    m.innerHTML=buildPreviewPage();
    var ft=fileType(S.previewItem);
    if(ft==='audio'||ft==='video')setTimeout(function(){setupMedia(ft,S.previewItem)},0);
    return;
  }
  m.innerHTML='<div class="tabs" id="tabs"></div>'+
    '<div class="toolbar" id="toolbar"></div>'+
    (S.searchMode?'<div class="search-q">Searching for <strong>'+h(S.searchQ)+'</strong></div>':'')+
    '<div id="file-area" style="flex:1;overflow-y:auto;position:relative"></div>'+
    '<div id="foot"></div>'+
    '<div class="bulk-bar" id="bulk-bar"></div>'+
    (!DEMO?'<div class="drop-zone" id="drop-zone"><div class="drop-zone-icon">'+I.upload+'</div><div class="drop-zone-text">Drop files or folders</div><div class="drop-zone-sub">Upload to '+h(S.path||'/')+'</div></div>':'');
  renderTabs();renderToolbar();renderItems();renderBulk();renderFoot();
}

/* ── Toolbar ──────────────────────────────────────────────────────── */
function renderToolbar(){
  var t=$('toolbar');if(!t)return;
  var sec=S.section;
  // Breadcrumb
  var bc='<nav class="crumb"><button class="crumb-home" onclick="B.nav(\\x27\\x27)">'+I.home+'</button>';
  if(S.path){
    var parts=S.path.replace(/\/$/,'').split('/');
    var cur='';
    parts.forEach(function(p){
      cur+=p+'/';
      bc+='<span class="crumb-sep">/</span><button class="crumb-seg" onclick="B.nav(\\x27'+cur.replace(/'/g,"\\x27")+'\\x27)">'+h(p)+'</button>';
    });
  }
  bc+='</nav>';
  // Actions
  var acts='<div class="toolbar-actions">';
  if(sec==='files'){
    acts+='<button class="btn btn--icon'+(S.view==='list'?' btn--active':'')+'" onclick="B.setView(\\x27list\\x27)" title="List">'+I.list+'</button>';
    acts+='<button class="btn btn--icon'+(S.view==='grid'?' btn--active':'')+'" onclick="B.setView(\\x27grid\\x27)" title="Grid">'+I.grid+'</button>';
    acts+='<span class="toolbar-sep"></span>';
    if(DEMO){
      acts+='<button class="btn" onclick="requireSignup()">'+I.plus+' <span>Folder</span></button>';
      acts+='<button class="btn btn--primary" onclick="requireSignup()">'+I.upload+' <span>Upload</span></button>';
    }else{
      acts+='<button class="btn" onclick="B.newFolder()">'+I.plus+' <span>Folder</span></button>';
      acts+='<button class="btn btn--primary" onclick="$(\\x27file-input\\x27).click()">'+I.upload+' <span>Upload</span></button>';
    }
  }else if(sec==='trash'&&!DEMO){
    acts+='<button class="btn btn--danger" onclick="B.emptyTrash()">'+I.trash+' Empty</button>';
  }
  acts+='</div>';
  t.innerHTML=bc+acts;
}

/* ── Items ────────────────────────────────────────────────────────── */
function renderItems(){
  var fa=$('file-area');if(!fa)return;
  if(!S.items.length){
    var msg='No files here',sub='',action='';
    if(S.section==='files'&&!S.path){msg='Your drive is empty';sub='Drag files here or click Upload';if(!DEMO)action='<button class="btn btn--primary" onclick="$(\\x27file-input\\x27).click()">'+I.upload+' Upload</button>'}
    else if(S.section==='starred'){msg='No starred files';sub='Star files to find them quickly'}
    else if(S.section==='shared'){msg='Nothing shared';sub='Files shared with you appear here'}
    else if(S.section==='trash'){msg='Trash is empty'}
    else if(S.searchMode){msg='No results';sub='No files matching "'+h(S.searchQ)+'"'}
    fa.innerHTML='<div class="empty"><div class="empty-icon">'+I.file+'</div><div class="empty-title">'+msg+'</div>'+(sub?'<div class="empty-sub">'+sub+'</div>':'')+(action?'<div class="empty-action">'+action+'</div>':'')+'</div>';
    return;
  }
  S.view==='grid'?renderGrid(fa):renderList(fa);
}

function renderList(fa){
  var sec=S.section,isShared=sec==='shared';
  var sa=S.sort,arrow=sa.asc?'\u2191':'\u2193';
  var hd='<div class="list-head"><div class="col col-check"><input type="checkbox" class="row-check" onchange="B.selectAll(this.checked)"></div>';
  hd+='<div class="col col-name" onclick="B.setSort(\\x27name\\x27)">name'+(sa.col==='name'?' <span class="sort-arrow">'+arrow+'</span>':'')+'</div>';
  if(isShared){hd+='<div class="col col-size">owner</div><div class="col col-date">perm</div>'}
  else{hd+='<div class="col col-size" onclick="B.setSort(\\x27size\\x27)">size'+(sa.col==='size'?' <span class="sort-arrow">'+arrow+'</span>':'')+'</div>';
    hd+='<div class="col col-date" onclick="B.setSort(\\x27updated_at\\x27)">modified'+(sa.col==='updated_at'?' <span class="sort-arrow">'+arrow+'</span>':'')+'</div>'}
  hd+='</div>';
  var bd='<div class="list-body">';
  S.items.forEach(function(item){
    var p=item.path,sel=S.selected.has(p);
    var cls='row'+(item.is_folder?' row--folder':'')+(sel?' selected':'');
    bd+='<div class="'+cls+'" data-path="'+h(p)+'">';
    bd+='<div class="col-check"><input type="checkbox" class="row-check"'+(sel?' checked':'')+' onchange="B.toggleSel(\\x27'+p.replace(/'/g,"\\x27")+'\\x27,event)"></div>';
    bd+='<div class="col-name"><div class="col-name-inner">'+fileIconHtml(item)+'<span class="file-name">'+h(item.name)+'</span></div>';
    if(!isShared){
      var starCls='star-btn'+(item.starred?' starred':'');
      bd+='<button class="'+starCls+'" onclick="event.stopPropagation();B.star(\\x27'+p.replace(/'/g,"\\x27")+'\\x27,'+(item.starred?0:1)+')">'+(item.starred?I.starFill:I.star)+'</button>';
    }
    bd+='</div>';
    if(isShared){bd+='<div class="col-size"><span class="col-size-val">'+h(item.owner||'')+'</span></div><div class="col-date"><span class="perm perm--'+(item.permission||'viewer')+'">'+h(item.permission||'viewer')+'</span></div>'}
    else{bd+='<div class="col-size"><span class="col-size-val">'+(item.is_folder?'\u2014':fmtSize(item.size))+'</span></div><div class="col-date"><span class="col-date-val">'+fmtTime(item.updated_at)+'</span></div>'}
    bd+='</div>';
  });
  bd+='</div>';
  fa.innerHTML=hd+bd;
}

function renderGrid(fa){
  var g='<div class="file-grid">';
  S.items.forEach(function(item){
    var p=item.path,sel=S.selected.has(p);
    var cls='grid-card'+(item.is_folder?' grid-card--folder':'')+(sel?' selected':'');
    var thumb='';
    var ft=fileType(item);
    if(ft==='image'&&!DEMO)thumb='<img src="/files/'+encodeURIComponent(p)+'?thumb=1" alt="">';
    else thumb=(I[ft]||I.file);
    g+='<div class="'+cls+'" data-path="'+h(p)+'">';
    g+='<div class="grid-thumb">'+thumb+'</div>';
    g+='<div class="grid-check"><input type="checkbox" class="row-check"'+(sel?' checked':'')+' onchange="B.toggleSel(\\x27'+p.replace(/'/g,"\\x27")+'\\x27,event)"></div>';
    var starCls='grid-star'+(item.starred?' starred':'');
    g+='<button class="'+starCls+'" onclick="event.stopPropagation();B.star(\\x27'+p.replace(/'/g,"\\x27")+'\\x27,'+(item.starred?0:1)+')">'+(item.starred?I.starFill:I.star)+'</button>';
    g+='<div class="grid-info"><div class="grid-name">'+h(item.name)+'</div><div class="grid-meta">'+(item.is_folder?'folder':fmtSize(item.size)+' \u00b7 '+fmtTime(item.updated_at))+'</div></div></div>';
  });
  g+='</div>';
  fa.innerHTML=g;
}

/* ── Bulk ─────────────────────────────────────────────────────────── */
function renderBulk(){
  var b=$('bulk-bar');if(!b)return;
  if(!S.selected.size){b.classList.remove('open');return}
  b.classList.add('open');
  var n=S.selected.size;
  var acts='<div class="bulk-actions">';
  if(S.section==='trash'){
    if(DEMO){acts+='<button class="bulk-btn" onclick="requireSignup()">'+I.restore+' Restore</button>'}
    else{acts+='<button class="bulk-btn" onclick="B.bulkRestore()">'+I.restore+' Restore</button><button class="bulk-btn" onclick="B.permDelete(Array.from(S.selected))">'+I.trash+' Delete</button>'}
  }else{
    if(DEMO){acts+='<button class="bulk-btn" onclick="requireSignup()">'+I.download+' Download</button><button class="bulk-btn" onclick="requireSignup()">'+I.star+' Star</button><button class="bulk-btn" onclick="requireSignup()">'+I.trash+' Trash</button>'}
    else{acts+='<button class="bulk-btn" onclick="B.bulkDownload()">'+I.download+'</button><button class="bulk-btn" onclick="B.bulkStar()">'+I.star+'</button><button class="bulk-btn" onclick="B.bulkMove()">'+I.move+'</button><button class="bulk-btn" onclick="B.bulkTrash()">'+I.trash+'</button>'}
  }
  acts+='</div>';
  b.innerHTML='<span class="bulk-count">'+n+' selected</span>'+acts+'<button class="bulk-dismiss" onclick="B.clearSel()">&times;</button>';
}

/* ── Footer ───────────────────────────────────────────────────────── */
function renderFoot(){
  var f=$('foot');if(!f)return;
  var s=S.stats;
  if(!s){f.innerHTML='';return}
  f.innerHTML=s.file_count+' files \u00b7 '+s.folder_count+' folders \u00b7 '+fmtSize(s.total_size)+' / '+fmtSize(s.quota);
}

/* ── Preview page (lean: one header, content, one footer) ─────────── */
function buildPreviewPage(){
  var item=S.previewItem;if(!item)return'';
  var ft=fileType(item);
  var name=h(item.name);
  var meta=ft+' \u00b7 '+fmtSize(item.size)+' \u00b7 '+fmtTime(item.updated_at);

  // Content body
  var body='';
  if(DEMO){
    var dc=DEMO_CONTENT?DEMO_CONTENT[item.path]:null;
    var dd=DEMO_DOCS?DEMO_DOCS[item.path]:null;
    if(dc){
      if(ft==='code'){var lang=langFromName(item.name);body='<pre class="preview-code">'+highlightCode(dc,lang)+'</pre>'}
      else if(item.name.endsWith('.md')){body='<div class="preview-md">'+renderMarkdown(dc)+'</div>'}
      else if(ft==='sheet'){body=csvToTable(dc)}
      else{body='<pre class="preview-text">'+h(dc)+'</pre>'}
    }else if(dd){
      if(item.name.endsWith('.pdf'))body=renderPdfPages(dd);
      else if(item.name.endsWith('.docx'))body=renderDocxPage(dd);
      else if(item.name.endsWith('.xlsx'))body=renderXlsxSheet(dd);
      else if(item.name.endsWith('.pptx'))body=renderPptxSlides(dd);
    }else if(ft==='image'){
      body='<div style="text-align:center"><svg width="200" height="140" viewBox="0 0 200 140"><defs><linearGradient id="ig" x1="0" y1="0" x2="1" y2="1"><stop offset="0%" stop-color="var(--surface-2)"/><stop offset="100%" stop-color="var(--border)"/></linearGradient></defs><rect width="200" height="140" fill="url(#ig)"/><text x="100" y="76" text-anchor="middle" fill="var(--text-3)" font-size="12" font-family="JetBrains Mono">'+name+'</text></svg></div>';
    }else if(ft==='audio'){
      var bars='';for(var i=0;i<50;i++){bars+='<div class="mp-wave-bar" style="height:'+(10+Math.random()*28)+'px"></div>'}
      body='<div class="mp mp--audio" id="mp-audio"><div class="mp-art">'+I.audio+'</div><div class="mp-title">'+name+'</div><div class="mp-wave" id="mp-wave">'+bars+'</div><div class="mp-controls"><button class="mp-play-btn" id="mp-play"><svg viewBox="0 0 24 24" fill="currentColor"><polygon points="5 3 19 12 5 21 5 3"/></svg></button><span class="mp-time" id="mp-time">0:00</span><div class="mp-progress" id="mp-progress"><div class="mp-progress-fill" id="mp-fill" style="width:0"><div class="mp-progress-thumb" id="mp-thumb"></div></div></div><span class="mp-time" id="mp-dur">0:00</span><div class="mp-vol-wrap"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polygon points="11 5 6 9 2 9 2 15 6 15 11 19 11 5"/><path d="M19.07 4.93a10 10 0 010 14.14"/></svg><div class="mp-vol" id="mp-vol"><div class="mp-vol-fill" style="width:100%"></div></div></div></div></div>';
    }else if(ft==='video'){
      body='<div class="mp mp--video" id="mp-video"><div class="mp-viewport" id="mp-viewport"><div class="mp-play-overlay" id="mp-play-big"><svg viewBox="0 0 24 24" fill="currentColor"><polygon points="5 3 19 12 5 21 5 3"/></svg></div></div><div class="mp-controls"><button class="mp-play-btn" id="mp-play"><svg viewBox="0 0 24 24" fill="currentColor"><polygon points="5 3 19 12 5 21 5 3"/></svg></button><span class="mp-time" id="mp-time">0:00</span><div class="mp-progress" id="mp-progress"><div class="mp-progress-fill" id="mp-fill" style="width:0"><div class="mp-progress-thumb" id="mp-thumb"></div></div></div><span class="mp-time" id="mp-dur">0:00</span><button class="mp-fs-btn" id="mp-fs"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" width="14" height="14"><polyline points="15 3 21 3 21 9"/><polyline points="9 21 3 21 3 15"/><line x1="21" y1="3" x2="14" y2="10"/><line x1="3" y1="21" x2="10" y2="14"/></svg></button></div></div>';
    }else{
      body='<div class="preview-generic"><div class="file-icon" style="width:64px;height:64px">'+I.file+'</div><div class="preview-generic-name">'+name+'</div><div class="preview-generic-meta">'+h(item.content_type||'unknown')+' \u00b7 '+fmtSize(item.size)+'</div></div>';
    }
  }else{
    // Auth mode
    if(S.previewLoading){body='<div style="padding:60px;text-align:center"><div class="spinner"></div></div>'}
    else if(S.previewContent!==null){
      var c=S.previewContent;
      if(ft==='code'){var lang=langFromName(item.name);body='<pre class="preview-code">'+highlightCode(c,lang)+'</pre>'}
      else if(item.name.endsWith('.md')){body='<div class="preview-md">'+renderMarkdown(c)+'</div>'}
      else if(ft==='sheet'){body=csvToTable(c)}
      else{body='<pre class="preview-text">'+h(c)+'</pre>'}
    }else if(ft==='image'){
      body='<img class="preview-img" src="/files/'+encodeURIComponent(item.path)+'" alt="'+name+'">';
    }else if(ft==='audio'){
      var bars='';for(var i=0;i<50;i++){bars+='<div class="mp-wave-bar" style="height:'+(10+Math.random()*28)+'px"></div>'}
      body='<div class="mp mp--audio" id="mp-audio"><div class="mp-art">'+I.audio+'</div><div class="mp-title">'+name+'</div><div class="mp-wave" id="mp-wave">'+bars+'</div><div class="mp-controls"><button class="mp-play-btn" id="mp-play"><svg viewBox="0 0 24 24" fill="currentColor"><polygon points="5 3 19 12 5 21 5 3"/></svg></button><span class="mp-time" id="mp-time">0:00</span><div class="mp-progress" id="mp-progress"><div class="mp-progress-fill" id="mp-fill" style="width:0"><div class="mp-progress-thumb" id="mp-thumb"></div></div></div><span class="mp-time" id="mp-dur">0:00</span><div class="mp-vol-wrap"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polygon points="11 5 6 9 2 9 2 15 6 15 11 19 11 5"/><path d="M19.07 4.93a10 10 0 010 14.14"/></svg><div class="mp-vol" id="mp-vol"><div class="mp-vol-fill" style="width:100%"></div></div></div></div></div>';
    }else if(ft==='video'){
      body='<div class="mp mp--video" id="mp-video"><div class="mp-viewport" id="mp-viewport"><div class="mp-play-overlay" id="mp-play-big"><svg viewBox="0 0 24 24" fill="currentColor"><polygon points="5 3 19 12 5 21 5 3"/></svg></div></div><div class="mp-controls"><button class="mp-play-btn" id="mp-play"><svg viewBox="0 0 24 24" fill="currentColor"><polygon points="5 3 19 12 5 21 5 3"/></svg></button><span class="mp-time" id="mp-time">0:00</span><div class="mp-progress" id="mp-progress"><div class="mp-progress-fill" id="mp-fill" style="width:0"><div class="mp-progress-thumb" id="mp-thumb"></div></div></div><span class="mp-time" id="mp-dur">0:00</span><button class="mp-fs-btn" id="mp-fs"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" width="14" height="14"><polyline points="15 3 21 3 21 9"/><polyline points="9 21 3 21 3 15"/><line x1="21" y1="3" x2="14" y2="10"/><line x1="3" y1="21" x2="10" y2="14"/></svg></button></div></div>';
    }else if(/\.(pdf|docx?|xlsx?|pptx?)$/i.test(item.name)){
      body='<div class="preview-office"><div class="preview-office-icon">'+I.doc+'</div><div class="preview-office-name">'+name+'</div><div class="preview-office-type">'+h(item.content_type)+'</div><div class="preview-office-size">'+fmtSize(item.size)+'</div><button class="preview-office-dl" onclick="B.downloadCurrent()">'+I.download+' Download</button></div>';
    }else{
      body='<div class="preview-generic"><div class="file-icon" style="width:64px;height:64px">'+I.file+'</div><div class="preview-generic-name">'+name+'</div><div class="preview-generic-meta">'+h(item.content_type||'unknown')+' \u00b7 '+fmtSize(item.size)+'</div></div>';
    }
  }

  // Sibling nav
  var siblings=S.items.filter(function(f){return!f.is_folder});
  var idx=-1;siblings.forEach(function(f,i){if(f.path===item.path)idx=i});
  var prev=idx>0?siblings[idx-1]:null;
  var next=idx<siblings.length-1?siblings[idx+1]:null;
  var sibHtml='<div class="pv-sibling">';
  sibHtml+=prev?'<button class="pv-sib-link" onclick="B.previewNav(-1)">'+I.arrowL+' '+h(prev.name)+'</button>':'<span></span>';
  sibHtml+=next?'<button class="pv-sib-link" onclick="B.previewNav(1)">'+h(next.name)+' '+I.arrowR+'</button>':'<span></span>';
  sibHtml+='</div>';

  // Actions — just icon buttons, extremely lean
  var actHtml='<div class="pv-actions">';
  if(DEMO){
    actHtml+='<button class="btn btn--icon" onclick="requireSignup()" title="Download">'+I.download+'</button>';
    actHtml+='<button class="btn btn--icon" onclick="requireSignup()" title="Share">'+I.share+'</button>';
  }else{
    actHtml+='<button class="btn btn--icon" onclick="B.downloadCurrent()" title="Download">'+I.download+'</button>';
    actHtml+='<button class="btn btn--icon" onclick="B.copyPreviewLink()" title="Copy link">'+I.link+'</button>';
    actHtml+='<button class="btn btn--icon" onclick="B.showShareModal(\\x27'+item.path.replace(/'/g,"\\x27")+'\\x27)" title="Share">'+I.share+'</button>';
  }
  actHtml+='</div>';

  return '<div class="pv">'+
    '<div class="pv-top">'+
      '<button class="pv-back" onclick="B.closePreview()">'+I.arrowL+' <span>'+name+'</span></button>'+
      '<span class="pv-info">'+meta+'</span>'+
      actHtml+
    '</div>'+
    '<div class="pv-body">'+body+'</div>'+
    sibHtml+
  '</div>';
}

/* ── Context menu ─────────────────────────────────────────────────── */
function renderCtx(x,y,item){
  var m=$('ctx-menu');
  var w=window.innerWidth,ht=window.innerHeight;
  if(w<640){m.style.left='0';m.style.top='auto';m.style.bottom='0'}
  else{m.style.left=Math.min(x,w-200)+'px';m.style.top=Math.min(y,ht-300)+'px';m.style.bottom='auto'}
  var html='';
  if(!item){
    if(DEMO){html='<div class="ctx-item" onclick="requireSignup()">'+I.plus+' New folder</div><div class="ctx-item" onclick="requireSignup()">'+I.upload+' Upload</div>'}
    else{html='<div class="ctx-item" onclick="hideCtx();B.newFolder()">'+I.plus+' New folder</div><div class="ctx-item" onclick="hideCtx();$(\\x27file-input\\x27).click()">'+I.upload+' Upload</div>';
      if(S.clipboard)html+='<div class="ctx-sep"></div><div class="ctx-item" onclick="hideCtx();B.paste()">'+I.copy+' Paste</div>';
    }
  }else if(S.section==='trash'){
    if(DEMO){html='<div class="ctx-item" onclick="requireSignup()">'+I.restore+' Restore</div>'}
    else{html='<div class="ctx-item" onclick="hideCtx();B.restoreItems([\\x27'+item.path.replace(/'/g,"\\x27")+'\\x27])">'+I.restore+' Restore</div><div class="ctx-sep"></div><div class="ctx-item ctx-item--danger" onclick="hideCtx();B.permDelete([\\x27'+item.path.replace(/'/g,"\\x27")+'\\x27])">'+I.trash+' Delete forever</div>'}
  }else{
    if(item.is_folder){html='<div class="ctx-item" onclick="hideCtx();B.nav(\\x27'+item.path.replace(/'/g,"\\x27")+'\\x27)">'+I.folder+' Open</div>'}
    else{
      var dl=DEMO?'onclick="requireSignup()"':'onclick="hideCtx();B.downloadFile(S.items.find(function(f){return f.path===\\x27'+item.path.replace(/'/g,"\\x27")+'\\x27}))"';
      html='<div class="ctx-item" '+dl+'>'+I.download+' Download</div>';
    }
    html+='<div class="ctx-sep"></div>';
    if(DEMO){
      html+='<div class="ctx-item" onclick="requireSignup()">'+I.share+' Share</div>';
      html+='<div class="ctx-item" onclick="requireSignup()">'+I.rename+' Rename</div>';
      html+='<div class="ctx-item" onclick="requireSignup()">'+I.move+' Move</div>';
      if(!item.is_folder)html+='<div class="ctx-item" onclick="requireSignup()">'+I.copy+' Copy</div>';
      html+='<div class="ctx-sep"></div>';
      html+='<div class="ctx-item ctx-item--danger" onclick="requireSignup()">'+I.trash+' Trash</div>';
    }else{
      html+='<div class="ctx-item" onclick="hideCtx();B.startRename(\\x27'+item.path.replace(/'/g,"\\x27")+'\\x27)">'+I.rename+' Rename</div>';
      html+='<div class="ctx-item" onclick="hideCtx();B.showMoveModal([\\x27'+item.path.replace(/'/g,"\\x27")+'\\x27])">'+I.move+' Move</div>';
      if(!item.is_folder)html+='<div class="ctx-item" onclick="hideCtx();B.copyFile(\\x27'+item.path.replace(/'/g,"\\x27")+'\\x27)">'+I.copy+' Copy</div>';
      html+='<div class="ctx-item" onclick="hideCtx();B.showShareModal(\\x27'+item.path.replace(/'/g,"\\x27")+'\\x27)">'+I.share+' Share</div>';
      html+='<div class="ctx-item" onclick="hideCtx();B.star(\\x27'+item.path.replace(/'/g,"\\x27")+'\\x27,'+(item.starred?0:1)+')">'+(!item.starred?I.star:I.starFill)+' '+(item.starred?'Unstar':'Star')+'</div>';
      html+='<div class="ctx-sep"></div>';
      html+='<div class="ctx-item ctx-item--danger" onclick="hideCtx();B.trashItems([\\x27'+item.path.replace(/'/g,"\\x27")+'\\x27])">'+I.trash+' Trash</div>';
    }
  }
  m.innerHTML=html;
  m.classList.add('open');
}

/* ── Command palette ──────────────────────────────────────────────── */
function openCmdPalette(){
  var m=$('cmd-palette');
  m.innerHTML='<div class="cmd-box"><div class="cmd-input">'+I.search+'<input type="text" id="cmd-search" placeholder="Search files..." autocomplete="off"></div><div class="cmd-results" id="cmd-results"></div></div>';
  m.classList.add('open');
  var inp=$('cmd-search');inp.focus();
  var timer;
  inp.oninput=function(){clearTimeout(timer);timer=setTimeout(function(){updateCmdResults(inp.value.trim())},150)};
  inp.onkeydown=cmdKeydown;
}
function closeCmdPalette(){var m=$('cmd-palette');m.classList.remove('open');m.innerHTML=''}
function updateCmdResults(q){
  if(!q){$('cmd-results').innerHTML='';return}
  if(DEMO){
    var r=DEMO_FS.filter(function(f){return!f.trashed_at&&f.name.toLowerCase().includes(q.toLowerCase())}).slice(0,10);
    renderCmdItems(r);
  }else{
    api.get('/drive/search?q='+encodeURIComponent(q)).then(function(d){renderCmdItems(d.items||[])}).catch(function(){});
  }
}
function renderCmdItems(items){
  var r=$('cmd-results');
  if(!items.length){r.innerHTML='<div class="cmd-empty">No results</div>';return}
  r.innerHTML='<div class="cmd-group">Files</div>'+items.map(function(f,i){
    return'<div class="cmd-result'+(i===0?' active':'')+'" data-path="'+h(f.path)+'">'+fileIconHtml(f)+'<span>'+h(f.name)+'</span><span class="cmd-result-path">'+h(f.path)+'</span></div>';
  }).join('');
}
function selectCmdResult(path){
  closeCmdPalette();
  var item=DEMO?DEMO_FS.find(function(f){return f.path===path}):null;
  if(item&&item.is_folder){B.nav(path);return}
  if(item&&!item.is_folder){
    var parent=path.replace(/[^/]+$/,'');
    S.path=parent;
    loadItems();
    setTimeout(function(){B.openPreview(path,item)},50);
    return;
  }
  // Auth mode
  if(path.endsWith('/')){B.nav(path)}
  else{
    var parent=path.replace(/[^/]+$/,'');
    S.path=parent;S.section='files';
    loadItems();
    setTimeout(function(){B.openPreview(path)},100);
  }
}
function cmdKeydown(e){
  var res=$('cmd-results');if(!res)return;
  var items=res.querySelectorAll('.cmd-result');
  var idx=-1;items.forEach(function(el,i){if(el.classList.contains('active'))idx=i});
  if(e.key==='ArrowDown'){e.preventDefault();if(idx<items.length-1){if(idx>=0)items[idx].classList.remove('active');items[idx+1].classList.add('active');items[idx+1].scrollIntoView({block:'nearest'})}}
  else if(e.key==='ArrowUp'){e.preventDefault();if(idx>0){items[idx].classList.remove('active');items[idx-1].classList.add('active');items[idx-1].scrollIntoView({block:'nearest'})}}
  else if(e.key==='Enter'){e.preventDefault();if(idx>=0&&items[idx])selectCmdResult(items[idx].dataset.path);else{var q=$('cmd-search').value.trim();if(q){closeCmdPalette();B.search(q)}}}
  else if(e.key==='Escape'){closeCmdPalette()}
}

/* ══════════════════════════════════════════════════════════════════════
   CONTROLLER
   ══════════════════════════════════════════════════════════════════════ */
var B=window.B={
  nav:function(p){S.path=p;S.section='files';S.searchMode=false;S.selected.clear();S.previewItem=null;history.pushState(null,'','/browse/'+(p||''));loadItems()},
  setSection:function(sec){S.section=sec;S.path='';S.searchMode=false;S.selected.clear();S.previewItem=null;loadItems()},
  setView:function(v){S.view=v;localStorage.setItem('bv',v);renderItems();renderToolbar()},
  setSort:function(col){if(S.sort.col===col)S.sort.asc=!S.sort.asc;else{S.sort.col=col;S.sort.asc=true}localStorage.setItem('bs',JSON.stringify(S.sort));sortItems();renderToolbar();renderItems()},
  clickSel:function(path,e){
    if(e.ctrlKey||e.metaKey){S.selected.has(path)?S.selected.delete(path):S.selected.add(path)}
    else if(e.shiftKey){
      var all=S.items.map(function(f){return f.path});
      var last=Array.from(S.selected).pop()||all[0];
      var a=all.indexOf(last),b=all.indexOf(path);
      if(a>b){var t=a;a=b;b=t}
      for(var i=a;i<=b;i++)S.selected.add(all[i]);
    }else{S.selected.clear();S.selected.add(path)}
    renderBulk();renderItems();
  },
  toggleSel:function(path,e){e&&e.stopPropagation();S.selected.has(path)?S.selected.delete(path):S.selected.add(path);renderBulk();renderItems()},
  selectAll:function(c){if(c)S.items.forEach(function(f){S.selected.add(f.path)});else S.selected.clear();renderBulk();renderItems()},
  clearSel:function(){S.selected.clear();renderBulk();renderItems()},

  openPreview:function(path,itemOverride){
    var item=itemOverride||S.items.find(function(f){return f.path===path});
    if(!item||item.is_folder)return;
    S.previewItem=item;S.previewContent=null;S.previewLoading=false;
    history.pushState(null,'','/browse/'+path);
    if(!DEMO){
      var ft=fileType(item);
      if(ft==='code'||ft==='text'||ft==='sheet'||item.name.endsWith('.md')){
        S.previewLoading=true;renderMain();
        fetch('/files/'+encodeURIComponent(path)).then(function(r){return r.text()}).then(function(t){
          S.previewContent=t;S.previewLoading=false;renderMain();
        }).catch(function(){S.previewLoading=false;renderMain()});
        return;
      }
    }
    renderMain();
  },
  closePreview:function(){
    if(S._mediaCleanup){S._mediaCleanup();S._mediaCleanup=null}
    S.previewItem=null;S.previewContent=null;
    history.pushState(null,'','/browse/'+(S.path||''));
    renderMain();
  },
  copyPreviewLink:function(){navigator.clipboard.writeText(location.href).then(function(){toast('Link copied','ok')})},
  previewNav:function(dir){
    var siblings=S.items.filter(function(f){return!f.is_folder});
    var idx=-1;siblings.forEach(function(f,i){if(S.previewItem&&f.path===S.previewItem.path)idx=i});
    var next=siblings[idx+dir];if(next)B.openPreview(next.path,next);
  },
  downloadCurrent:function(){if(S.previewItem)B.downloadFile(S.previewItem)},
  downloadItem:function(path){var it=S.items.find(function(f){return f.path===path});if(it)B.downloadFile(it)},
  showSlide:function(idx){
    var stage=$('pptx-stage');if(!stage)return;
    var slides=stage.querySelectorAll('.pptx-slide');
    slides.forEach(function(s){s.classList.remove('active')});
    if(slides[idx])slides[idx].classList.add('active');
    var thumbs=stage.previousElementSibling;
    if(thumbs){var ts=thumbs.querySelectorAll('.pptx-thumb');ts.forEach(function(t){t.classList.remove('active')});if(ts[idx])ts[idx].classList.add('active')}
  },
  search:function(q){if(!q){S.searchMode=false;S.section='files';loadItems();return}S.searchQ=q;S.searchMode=true;loadItems()},

  // Auth-only operations
  star:function(path,v){
    if(DEMO){requireSignup();return}
    api.patch('/drive/star',{path:path,starred:v}).then(function(){
      var it=S.items.find(function(f){return f.path===path});if(it)it.starred=!!v;
      toast(v?'Starred':'Unstarred','ok');renderItems();loadStats();
    }).catch(function(){toast('Failed','err')});
  },
  startRename:function(path){
    var item=S.items.find(function(f){return f.path===path});if(!item)return;
    var newName=prompt('Rename',item.name);if(!newName||newName===item.name)return;
    api.post('/drive/rename',{path:path,new_name:newName}).then(function(){toast('Renamed','ok');loadItems();loadStats()}).catch(function(){toast('Failed','err')});
  },
  trashItems:function(paths){
    api.post('/drive/trash',{paths:paths}).then(function(){toast(paths.length+' trashed','ok');S.selected.clear();loadItems();loadStats()}).catch(function(){toast('Failed','err')});
  },
  bulkTrash:function(){B.trashItems(Array.from(S.selected))},
  restoreItems:function(paths){
    api.post('/drive/restore',{paths:paths}).then(function(){toast('Restored','ok');S.selected.clear();loadItems();loadStats()}).catch(function(){toast('Failed','err')});
  },
  bulkRestore:function(){B.restoreItems(Array.from(S.selected))},
  emptyTrash:function(){
    if(!confirm('Permanently delete all trashed items?'))return;
    api.del('/drive/trash').then(function(){toast('Trash emptied','ok');loadItems();loadStats()}).catch(function(){toast('Failed','err')});
  },
  downloadFile:function(item){
    if(DEMO){requireSignup();return}
    var a=document.createElement('a');a.href='/files/'+encodeURIComponent(item.path);a.download=item.name;document.body.appendChild(a);a.click();a.remove();
  },
  bulkDownload:function(){S.selected.forEach(function(p){var it=S.items.find(function(f){return f.path===p});if(it&&!it.is_folder)B.downloadFile(it)})},
  bulkStar:function(){
    var paths=Array.from(S.selected);
    Promise.all(paths.map(function(p){return api.patch('/drive/star',{path:p,starred:1})})).then(function(){toast(paths.length+' starred','ok');loadItems();loadStats()});
  },
  showMoveModal:function(paths){
    B._movePaths=paths;B._moveDest='';
    var body='<div class="folder-tree" id="move-tree"><div class="tree-node" onclick="B.selectMoveTarget(this,\\x27\\x27)"><div class="tree-icon">'+I.home+'</div> / (root)</div></div>';
    showModal('Move to',body,'<button class="btn" onclick="hideModal()">Cancel</button><button class="btn btn--primary" onclick="B.doMove()">Move</button>');
    api.get('/folders/').then(function(d){
      var tree=$('move-tree');if(!tree)return;
      (d.items||[]).filter(function(f){return f.is_folder}).forEach(function(f){
        tree.innerHTML+='<div class="tree-node" onclick="B.selectMoveTarget(this,\\x27'+f.path.replace(/'/g,"\\x27")+'\\x27)"><div class="tree-icon">'+I.folder+'</div> '+h(f.name)+'</div>';
      });
    });
  },
  selectMoveTarget:function(el,path){
    B._moveDest=path;
    el.parentElement.querySelectorAll('.tree-node').forEach(function(n){n.classList.remove('selected')});
    el.classList.add('selected');
  },
  doMove:function(){
    api.post('/drive/move',{paths:B._movePaths,destination:B._moveDest}).then(function(){hideModal();toast('Moved','ok');S.selected.clear();loadItems();loadStats()}).catch(function(){toast('Failed','err')});
  },
  bulkMove:function(){B.showMoveModal(Array.from(S.selected))},
  copyFile:function(path){
    api.post('/drive/copy',{path:path}).then(function(d){toast('Copied as '+(d.name||'copy'),'ok');loadItems();loadStats()}).catch(function(){toast('Failed','err')});
  },
  showShareModal:function(path){
    var body='<div class="share-row"><input type="text" id="share-actor" placeholder="Actor (e.g. u/alice)"><select id="share-perm"><option value="viewer">Viewer</option><option value="editor">Editor</option></select><button class="btn" onclick="B.doShare(\\x27'+path.replace(/'/g,"\\x27")+'\\x27)">Share</button></div><div id="share-list"></div><div class="share-link" style="margin-top:12px"><span>'+location.origin+'/files/'+h(path)+'</span><button onclick="B.copyLink(\\x27'+path.replace(/'/g,"\\x27")+'\\x27)">Copy</button></div>';
    showModal('Share',body);
    if(!DEMO)api.get('/shares').then(function(d){
      var sl=$('share-list');if(!sl)return;
      var shares=(d.shares||[]).filter(function(s){return s.path===path});
      sl.innerHTML=shares.map(function(s){return'<div style="display:flex;align-items:center;gap:8px;padding:6px 0;font-size:12px"><span>'+h(s.grantee)+'</span><span class="perm perm--'+s.permission+'">'+s.permission+'</span></div>'}).join('');
    });
  },
  doShare:function(path){
    var actor=$('share-actor').value.trim(),perm=$('share-perm').value;
    if(!actor){toast('Enter an actor','err');return}
    api.post('/shares',{path:path,grantee:actor,permission:perm}).then(function(){toast('Shared with '+actor,'ok');B.showShareModal(path)}).catch(function(){toast('Failed','err')});
  },
  copyLink:function(path){navigator.clipboard.writeText(location.origin+'/files/'+path).then(function(){toast('Link copied','ok')})},
  saveDesc:function(path,desc){if(!DEMO)api.patch('/drive/description',{path:path,description:desc})},
  newFolder:function(){
    if(DEMO){requireSignup();return}
    var name=prompt('Folder name');if(!name)return;
    api.post('/folders',{path:S.path+name}).then(function(){toast('Folder created','ok');loadItems();loadStats()}).catch(function(){toast('Failed','err')});
  },
  uploadFiles:function(files){
    if(DEMO){requireSignup();return}
    var panel=$('upload-panel');if(panel)panel.classList.add('open');
    var arr=Array.from(files);
    // Merge into existing uploading list (don't clobber in-flight items)
    arr.forEach(function(file){
      S.uploading.push({name:file.name,size:file.size,progress:0,loaded:0,status:'pending',id:Math.random().toString(36).slice(2),file:file,tusId:null,retries:0});
    });
    renderUploadList();
    // Process up to 3 concurrent uploads
    function processQueue(){
      var active=S.uploading.filter(function(u){return u.status==='uploading'}).length;
      while(active<3){
        var next=S.uploading.find(function(u){return u.status==='pending'});
        if(!next)break;
        next.status='uploading';active++;
        tusUpload(next);
      }
      renderUploadList();
    }
    function onDone(u,ok){
      u.status=ok?'done':'error';
      u.progress=ok?100:u.progress;
      renderUploadList();
      // Check if all done
      var allDone=S.uploading.every(function(u){return u.status==='done'||u.status==='error'});
      if(allDone){
        var ok_count=S.uploading.filter(function(u){return u.status==='done'}).length;
        if(ok_count)toast(ok_count+' uploaded','ok');
        loadItems();loadStats();
      }else{processQueue()}
    }
    function tusUpload(u){
      var file=u.file;
      var path=S.path+(file._relativePath||file.name);
      var CHUNK=5*1024*1024; // 5MB chunks
      var meta='bucketName '+btoa('default')+',objectName '+btoa(path);
      if(file.type)meta+=',contentType '+btoa(file.type);
      // If we have a tusId, resume; otherwise create
      var resumeOrCreate=u.tusId
        ?fetch('/upload/resumable/'+u.tusId,{method:'HEAD',headers:{'Tus-Resumable':'1.0.0'}})
          .then(function(r){
            if(!r.ok)throw new Error('resume-failed');
            u.loaded=parseInt(r.headers.get('Upload-Offset')||'0',10);
            u.progress=u.size?Math.round(u.loaded/u.size*100):0;
            renderUploadList();
            return u.tusId;
          })
        :fetch('/upload/resumable',{
            method:'POST',
            headers:{
              'Tus-Resumable':'1.0.0',
              'Upload-Length':String(file.size),
              'Upload-Metadata':meta,
              'X-Upsert':'true',
            }
          }).then(function(r){
            if(!r.ok)throw new Error('create-failed');
            var loc=r.headers.get('Location')||'';
            var id=loc.split('/').pop();
            u.tusId=id;
            u.loaded=0;
            return id;
          });
      resumeOrCreate.then(function(tusId){
        // Send chunks sequentially
        function sendChunk(){
          if(u.loaded>=file.size){onDone(u,true);return}
          var end=Math.min(u.loaded+CHUNK,file.size);
          var blob=file.slice(u.loaded,end);
          fetch('/upload/resumable/'+tusId,{
            method:'PATCH',
            headers:{
              'Tus-Resumable':'1.0.0',
              'Upload-Offset':String(u.loaded),
              'Content-Type':'application/offset+octet-stream',
            },
            body:blob,
          }).then(function(r){
            if(!r.ok)throw new Error('patch-'+r.status);
            var newOffset=parseInt(r.headers.get('Upload-Offset')||String(end),10);
            u.loaded=newOffset;
            u.progress=u.size?Math.round(u.loaded/u.size*100):0;
            renderUploadList();
            sendChunk();
          }).catch(function(err){
            // Retry up to 3 times with backoff
            if(u.retries<3){
              u.retries++;u.status='retrying';renderUploadList();
              setTimeout(function(){
                // HEAD to get real offset before retrying
                fetch('/upload/resumable/'+tusId,{method:'HEAD',headers:{'Tus-Resumable':'1.0.0'}})
                  .then(function(r){
                    if(r.ok)u.loaded=parseInt(r.headers.get('Upload-Offset')||String(u.loaded),10);
                    u.status='uploading';sendChunk();
                  }).catch(function(){u.status='uploading';sendChunk()});
              },1000*u.retries);
            }else{onDone(u,false)}
          });
        }
        sendChunk();
      }).catch(function(){onDone(u,false)});
    }
    // Expose retry for individual items
    B._retryUpload=function(id){
      var u=S.uploading.find(function(x){return x.id===id});
      if(!u||u.status==='uploading')return;
      u.status='pending';u.retries=0;processQueue();
    };
    processQueue();
  },
  paste:function(){
    if(DEMO||!S.clipboard)return;
    var cb=S.clipboard;
    var chain=Promise.resolve();
    cb.paths.forEach(function(p){
      chain=chain.then(function(){return cb.action==='copy'?api.post('/drive/copy',{path:p}):api.post('/drive/move',{paths:[p],destination:S.path})});
    });
    chain.then(function(){S.clipboard=null;toast('Pasted','ok');loadItems();loadStats()}).catch(function(){toast('Failed','err')});
  },
  permDelete:function(paths){
    if(!confirm('Permanently delete '+paths.length+' item(s)?'))return;
    Promise.all(paths.map(function(p){return api.del('/drive/trash?path='+encodeURIComponent(p))})).then(function(){toast('Deleted','ok');S.selected.clear();loadItems();loadStats()}).catch(function(){toast('Failed','err')});
  },
};

/* ── Upload list ──────────────────────────────────────────────────── */
function renderUploadList(){
  var ul=$('upload-list');if(!ul)return;
  var title=$('upload-title');
  if(title){
    var done=S.uploading.filter(function(u){return u.status==='done'}).length;
    var total=S.uploading.length;
    var active=S.uploading.filter(function(u){return u.status==='uploading'||u.status==='retrying'}).length;
    if(active)title.textContent='Uploading '+done+'/'+total+'...';
    else if(done===total)title.textContent=total+' uploaded';
    else title.textContent=done+'/'+total+' uploaded';
  }
  ul.innerHTML=S.uploading.map(function(u){
    var icon,extra='';
    if(u.status==='done')icon='<span class="upload-ok">\u2713</span>';
    else if(u.status==='error'){icon='<span class="upload-err">\u2717</span>';extra='<button class="upload-retry" onclick="B._retryUpload(\\x27'+u.id+'\\x27)">retry</button>'}
    else if(u.status==='retrying')icon='<span class="upload-retry-spin"><div class="spinner"></div></span>';
    else if(u.status==='uploading')icon='<span class="upload-pct">'+u.progress+'%</span>';
    else icon='<span class="upload-pct">\u2022</span>';
    var bar='';
    if(u.status==='uploading'||u.status==='retrying'||u.status==='done'){
      bar='<div class="upload-item-bar"><div class="upload-item-fill'+(u.status==='done'?' fill--done':u.status==='retrying'?' fill--retry':'')+'" style="width:'+u.progress+'%"></div></div>';
    }else if(u.status==='error'){
      bar='<div class="upload-item-bar"><div class="upload-item-fill fill--err" style="width:'+u.progress+'%"></div></div>';
    }
    var sizeStr=u.size?fmtSize(u.loaded||0)+' / '+fmtSize(u.size):'';
    return'<div class="upload-item"><div class="upload-item-top"><span class="upload-item-name">'+h(u.name)+'</span>'+icon+extra+'</div>'+bar+'<div class="upload-item-meta">'+sizeStr+'</div></div>';
  }).join('');
}

/* ── Modal helpers ────────────────────────────────────────────────── */
function showModal(title,body,footer){
  var m=$('modal-root');
  m.innerHTML='<div class="modal"><div class="modal-head"><span class="modal-title">'+title+'</span><button class="modal-close" onclick="hideModal()">&times;</button></div><div class="modal-body">'+body+'</div>'+(footer?'<div class="modal-foot">'+footer+'</div>':'')+'</div>';
  m.classList.add('open');
}
function hideModal(){var m=$('modal-root');m.classList.remove('open');m.innerHTML=''}
function hideCtx(){$('ctx-menu').classList.remove('open')}

function showShortcuts(){
  var body='<div class="shortcuts">'+[
    ['/','/  Search'],['Esc','Close'],['Enter','Open'],['Del','Trash'],
    ['\u2191 \u2193','Navigate'],['Space','Preview'],['\u2190 \u2192','Prev/Next'],['F2','Rename'],
    ['S','Star'],['Ctrl+A','Select all'],['Ctrl+C','Copy'],['Ctrl+V','Paste'],
    ['?','Shortcuts'],
  ].map(function(s){return'<div class="shortcut"><span>'+s[1]+'</span><kbd>'+s[0]+'</kbd></div>'}).join('')+'</div>';
  showModal('Keyboard Shortcuts',body);
}

/* ══════════════════════════════════════════════════════════════════════
   EVENTS
   ══════════════════════════════════════════════════════════════════════ */

// Close menus on any click
document.addEventListener('click',function(){hideCtx()});

// Delegated click on main
document.addEventListener('click',function(e){
  var main=$('main');if(!main||!main.contains(e.target))return;
  // Tab click
  var tab=e.target.closest('.tab');
  if(tab&&tab.dataset.sec){B.setSection(tab.dataset.sec);return}
  // File row/card click
  var row=e.target.closest('.row,.grid-card');
  if(!row||!row.dataset.path)return;
  if(e.target.closest('.row-check,.star-btn,.grid-star,.grid-check'))return;
  var path=row.dataset.path;
  var item=S.items.find(function(f){return f.path===path});if(!item)return;
  if(e.ctrlKey||e.metaKey||e.shiftKey){B.clickSel(path,e);return}
  if(item.is_folder){B.nav(path)}else{B.openPreview(path,item)}
});

// Context menu
document.addEventListener('contextmenu',function(e){
  var main=$('main');if(!main||!main.contains(e.target))return;
  var row=e.target.closest('.row,.grid-card');
  if(row&&row.dataset.path){
    e.preventDefault();
    var p=row.dataset.path;
    if(!S.selected.has(p)){S.selected.clear();S.selected.add(p);renderBulk();renderItems()}
    var item=S.items.find(function(f){return f.path===p});
    renderCtx(e.clientX,e.clientY,item);
    return;
  }
  var fa=$('file-area');
  if(fa&&fa.contains(e.target)){e.preventDefault();renderCtx(e.clientX,e.clientY,null)}
});

// Command palette clicks
document.addEventListener('mousedown',function(e){
  var cp=$('cmd-palette');if(!cp||!cp.classList.contains('open'))return;
  var result=e.target.closest('.cmd-result');
  if(result){e.preventDefault();selectCmdResult(result.dataset.path);return}
  if(!e.target.closest('.cmd-box')){closeCmdPalette()}
});

// Search trigger
document.addEventListener('click',function(e){
  if(e.target.closest('#search-trigger')){openCmdPalette()}
});

// Theme toggle
document.addEventListener('click',function(e){
  if(e.target.closest('#theme-btn')){
    document.documentElement.classList.toggle('dark');
    localStorage.setItem('theme',document.documentElement.classList.contains('dark')?'dark':'light');
  }
});

// Init theme
(function(){
  var t=localStorage.getItem('theme');
  if(t==='light')document.documentElement.classList.remove('dark');
  else if(t==='dark')document.documentElement.classList.add('dark');
  else if(!matchMedia('(prefers-color-scheme:dark)').matches)document.documentElement.classList.remove('dark');
})();

/* ── Keyboard shortcuts ───────────────────────────────────────────── */
document.addEventListener('keydown',function(e){
  var tag=e.target.tagName;
  if(tag==='INPUT'||tag==='TEXTAREA'||tag==='SELECT')return;
  // Preview mode
  if(S.previewItem){
    if(e.key==='Escape'){B.closePreview();return}
    if(e.key==='ArrowLeft'){B.previewNav(-1);return}
    if(e.key==='ArrowRight'){B.previewNav(1);return}
  }
  if(e.key==='/'||(e.key==='k'&&(e.metaKey||e.ctrlKey))){e.preventDefault();openCmdPalette();return}
  if(e.key==='?'){showShortcuts();return}
  if(e.key==='Escape'){B.clearSel();hideCtx();hideModal();closeCmdPalette();render();return}
  if((e.key==='a'||e.key==='A')&&(e.metaKey||e.ctrlKey)){e.preventDefault();B.selectAll(true);return}
  if(e.key==='Enter'&&S.selected.size){
    var p=Array.from(S.selected)[0];var it=S.items.find(function(f){return f.path===p});
    if(it){it.is_folder?B.nav(p):B.openPreview(p,it)}return;
  }
  if(e.key===' '&&!S.previewItem&&S.selected.size){
    e.preventDefault();var p=Array.from(S.selected)[0];var it=S.items.find(function(f){return f.path===p});
    if(it&&!it.is_folder)B.openPreview(p,it);return;
  }
  if(e.key==='ArrowDown'||e.key==='ArrowUp'){
    e.preventDefault();
    var paths=S.items.map(function(f){return f.path});
    var cur=Array.from(S.selected).pop();
    var idx=cur?paths.indexOf(cur):-1;
    var ni=e.key==='ArrowDown'?Math.min(idx+1,paths.length-1):Math.max(idx-1,0);
    S.selected.clear();S.selected.add(paths[ni]);
    renderBulk();renderItems();
    var row=document.querySelector('[data-path="'+CSS.escape(paths[ni])+'"]');
    if(row)row.scrollIntoView({block:'nearest'});
    return;
  }
  if(!DEMO){
    if((e.key==='Delete'||e.key==='Backspace')&&S.selected.size){B.trashItems(Array.from(S.selected));return}
    if(e.key==='F2'&&S.selected.size===1){B.startRename(Array.from(S.selected)[0]);return}
    if((e.key==='s'||e.key==='S')&&S.selected.size===1){var p=Array.from(S.selected)[0];var it=S.items.find(function(f){return f.path===p});if(it)B.star(p,it.starred?0:1);return}
    if(e.key==='c'&&(e.metaKey||e.ctrlKey)&&S.selected.size){e.preventDefault();S.clipboard={action:'copy',paths:Array.from(S.selected)};toast('Copied to clipboard','ok');return}
    if(e.key==='x'&&(e.metaKey||e.ctrlKey)&&S.selected.size){e.preventDefault();S.clipboard={action:'cut',paths:Array.from(S.selected)};toast('Cut to clipboard','ok');return}
    if(e.key==='v'&&(e.metaKey||e.ctrlKey)){B.paste();return}
  }
});

/* ── Drag & drop ──────────────────────────────────────────────────── */
if(!DEMO){
  var dragCount=0;
  // Collect files from DataTransferItem entries (supports folder drops)
  function collectDropFiles(dt){
    return new Promise(function(resolve){
      if(!dt.items||!dt.items.length){resolve(dt.files?Array.from(dt.files):[]);return}
      var files=[],pending=0,done=false;
      function finish(){if(!done&&pending===0){done=true;resolve(files)}}
      function readEntry(entry,pathPrefix){
        if(entry.isFile){
          pending++;
          entry.file(function(f){
            // Preserve relative path for nested folder uploads
            var fullPath=pathPrefix?pathPrefix+'/'+f.name:f.name;
            Object.defineProperty(f,'_relativePath',{value:fullPath});
            files.push(f);pending--;finish();
          },function(){pending--;finish()});
        }else if(entry.isDirectory){
          pending++;
          var reader=entry.createReader();
          reader.readEntries(function(entries){
            pending--;
            entries.forEach(function(e){readEntry(e,pathPrefix?pathPrefix+'/'+entry.name:entry.name)});
            finish();
          },function(){pending--;finish()});
        }
      }
      for(var i=0;i<dt.items.length;i++){
        var item=dt.items[i];
        if(item.webkitGetAsEntry){
          var entry=item.webkitGetAsEntry();
          if(entry){readEntry(entry,'');continue}
        }
        var f=item.getAsFile();
        if(f)files.push(f);
      }
      // Fallback timeout in case entries API is flaky
      setTimeout(function(){if(!done){done=true;resolve(files)}},3000);
      finish();
    });
  }
  document.addEventListener('dragenter',function(e){e.preventDefault();dragCount++;var dz=$('drop-zone');if(dz){dz.classList.add('open');dz.querySelector('.drop-zone-sub').textContent='Upload to '+(S.path||'/')}});
  document.addEventListener('dragleave',function(e){e.preventDefault();dragCount--;if(dragCount<=0){dragCount=0;var dz=$('drop-zone');if(dz)dz.classList.remove('open')}});
  document.addEventListener('dragover',function(e){e.preventDefault();e.dataTransfer.dropEffect='copy'});
  document.addEventListener('drop',function(e){
    e.preventDefault();dragCount=0;
    var dz=$('drop-zone');if(dz)dz.classList.remove('open');
    if(!e.dataTransfer)return;
    collectDropFiles(e.dataTransfer).then(function(files){if(files.length)B.uploadFiles(files)});
  });
  var fi=$('file-input');if(fi)fi.addEventListener('change',function(){if(fi.files.length)B.uploadFiles(fi.files);fi.value=''});
  var uc=$('upload-close');if(uc)uc.addEventListener('click',function(){var p=$('upload-panel');if(p)p.classList.remove('open')});
}

/* ── Touch handlers ───────────────────────────────────────────────── */
var touchStart=null,touchTimer=null;
document.addEventListener('touchstart',function(e){
  var row=e.target.closest('.row,.grid-card');
  if(!row)return;
  touchStart={x:e.touches[0].clientX,y:e.touches[0].clientY,path:row.dataset.path};
  touchTimer=setTimeout(function(){
    if(!touchStart)return;
    var item=S.items.find(function(f){return f.path===touchStart.path});
    renderCtx(touchStart.x,touchStart.y,item);
    touchStart=null;
  },500);
},{passive:true});
document.addEventListener('touchmove',function(e){
  if(!touchStart)return;
  var dx=e.touches[0].clientX-touchStart.x,dy=e.touches[0].clientY-touchStart.y;
  if(Math.abs(dx)>10||Math.abs(dy)>10){clearTimeout(touchTimer);touchTimer=null}
},{passive:true});
document.addEventListener('touchend',function(e){
  clearTimeout(touchTimer);
  if(S.previewItem&&touchStart){
    var dx=e.changedTouches[0].clientX-touchStart.x;
    if(Math.abs(dx)>60){B.previewNav(dx>0?-1:1)}
  }
  touchStart=null;
},{passive:true});

/* ── History ──────────────────────────────────────────────────────── */
window.addEventListener('popstate',function(){
  var p=decodeURIComponent(location.pathname.replace(/^\/browse\/?/,''));
  if(DEMO){
    var file=DEMO_FS.find(function(f){return f.path===p&&!f.is_folder});
    if(file){
      var parent=p.replace(/[^/]+$/,'');
      S.path=parent;S.section='files';S.previewItem=null;
      loadItems();
      setTimeout(function(){B.openPreview(p,file)},50);
      return;
    }
    S.path=p;S.section='files';S.previewItem=null;loadItems();
  }else{
    if(p&&!p.endsWith('/')&&p.includes('.')){
      var parent=p.replace(/[^/]+$/,'');
      S.path=parent;S.section='files';S.previewItem=null;
      loadItems();
      setTimeout(function(){B.openPreview(p)},100);
      return;
    }
    S.path=p;S.section='files';S.previewItem=null;loadItems();
  }
});

/* ── Init ─────────────────────────────────────────────────────────── */
(function(){
  var p=decodeURIComponent(location.pathname.replace(/^\/browse\/?/,''));
  if(DEMO){
    var file=DEMO_FS.find(function(f){return f.path===p&&!f.is_folder});
    if(file){
      var parent=p.replace(/[^/]+$/,'');
      S.path=parent;
      loadItems();
      setTimeout(function(){B.openPreview(p,file)},50);
      return;
    }
    S.path=p;loadItems();loadStats();
  }else{
    if(p&&!p.endsWith('/')&&p.includes('.')){
      var parent=p.replace(/[^/]+$/,'');
      S.path=parent;
      loadItems();
      setTimeout(function(){B.openPreview(p)},100);
      loadStats();
      return;
    }
    S.path=p;loadItems();loadStats();
  }
})();

})();
