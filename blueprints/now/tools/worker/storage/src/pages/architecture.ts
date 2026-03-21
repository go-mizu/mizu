import architectureMd from "../machine/architecture.md";
import { markdownToHtml } from "../machine/render";

export function architecturePage(): string {
  return `<!DOCTYPE html>
<html lang="en" class="dark">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Architecture — Storage</title>
<meta name="description" content="How Storage is built: edge-first, event-sourced, content-addressed. Purpose-built for AI agents and humans.">
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800;900&family=JetBrains+Mono:wght@400;500;600;700&display=swap" rel="stylesheet">
<link rel="stylesheet" href="/base.css">
<link rel="stylesheet" href="/architecture.css">
</head>
<body>

<div class="grid-bg"></div>

<nav>
  <div class="nav-inner">
    <a href="/" class="logo"><span class="logo-dot"></span> Storage</a>
    <button class="mobile-toggle" onclick="document.querySelector('.nav-links').classList.toggle('open')" aria-label="Menu">
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="3" y1="6" x2="21" y2="6"/><line x1="3" y1="12" x2="21" y2="12"/><line x1="3" y1="18" x2="21" y2="18"/></svg>
    </button>
    <div class="nav-links">
      <a href="/developers">developers</a>
      <a href="/api">api</a>
      <a href="/cli">cli</a>
      <a href="/architecture" class="active">architecture</a>
      <a href="/pricing">pricing</a>
    </div>
    <div class="nav-right">
      <button class="theme-toggle" onclick="toggleTheme()" aria-label="Toggle theme">
        <svg class="icon-moon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 12.79A9 9 0 1111.21 3 7 7 0 0021 12.79z"/></svg>
        <svg class="icon-sun" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="5"/><line x1="12" y1="1" x2="12" y2="3"/><line x1="12" y1="21" x2="12" y2="23"/><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/><line x1="1" y1="12" x2="3" y2="12"/><line x1="21" y1="12" x2="23" y2="12"/><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/></svg>
      </button>
    </div>
  </div>
</nav>

<!-- ===== HUMAN VIEW ===== -->
<div class="human-view" id="human-view">

<!-- ═══════════════════════════════════════════════════════════════════
     HERO
     ═══════════════════════════════════════════════════════════════════ -->
<div class="hero">
  <div class="hero-rings"></div>
  <div class="section-inner section-inner--center">
    <div class="hero-badge">SYSTEM DESIGN</div>
    <h1 class="hero-title">Built for the age of <span class="grad">AI agents</span></h1>
    <p class="hero-sub">Edge-first. Event-sourced. Content-addressed. Every layer designed so humans and AI agents collaborate on files through one unified API.</p>
  </div>
</div>

<!-- ═══════════════════════════════════════════════════════════════════
     SYSTEM OVERVIEW — layered stack diagram
     ═══════════════════════════════════════════════════════════════════ -->
<div class="section" id="overview">
  <div class="section-glow"></div>
  <div class="section-inner">
    <div class="arch-label">System Overview</div>
    <div class="arch-h">Four layers, one API</div>
    <div class="arch-sub">Requests flow from clients through the edge runtime, which handles auth and routing, down to the metadata and blob stores.</div>

    <div class="stack">
      <!-- Layer 1: Client -->
      <div class="stack-layer">
        <div class="stack-label">CLIENT</div>
        <div class="stack-content">
          <div class="stack-name">Humans &amp; AI Agents</div>
          <div class="stack-desc">Any HTTP client, MCP-compatible AI, CLI tool, or web browser. No SDK required.</div>
          <div class="stack-tech">
            <span>curl</span><span>fetch</span><span>MCP</span><span>CLI</span><span>Browser</span>
          </div>
        </div>
      </div>

      <!-- Connector -->
      <div class="stack-connector">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <circle class="stack-pulse" cx="12" cy="4" r="2"/>
          <circle class="stack-pulse" cx="12" cy="12" r="2"/>
          <circle class="stack-pulse" cx="12" cy="20" r="2"/>
        </svg>
      </div>

      <!-- Layer 2: Edge -->
      <div class="stack-layer">
        <div class="stack-label">EDGE</div>
        <div class="stack-content">
          <div class="stack-name">Edge Runtime</div>
          <div class="stack-desc">Runs in V8 isolates globally. Handles authentication, routing, rate limiting, presigned URL generation, and MCP tool dispatch.</div>
          <div class="stack-tech">
            <span>V8 isolates</span><span>OpenAPI</span><span>REST</span><span>MCP</span>
          </div>
        </div>
      </div>

      <!-- Connector -->
      <div class="stack-connector">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <circle class="stack-pulse" cx="12" cy="4" r="2"/>
          <circle class="stack-pulse" cx="12" cy="12" r="2"/>
          <circle class="stack-pulse" cx="12" cy="20" r="2"/>
        </svg>
      </div>

      <!-- Layer 3+4: Storage split -->
      <div class="stack-split">
        <div class="stack-split-item">
          <div class="stack-label">METADATA</div>
          <div class="stack-name">Meta Plane</div>
          <div class="stack-desc">File index, event log, session tokens, blob references, transaction counters. Strong consistency for all metadata reads.</div>
          <div class="stack-tech">
            <span>files</span><span>events</span><span>blobs</span><span>sessions</span>
          </div>
        </div>
        <div class="stack-split-item">
          <div class="stack-label">BLOBS</div>
          <div class="stack-name">Object Storage</div>
          <div class="stack-desc">S3-compatible object store. Durable, globally distributed, zero egress fees. Content-addressed by SHA-256 hash.</div>
          <div class="stack-tech">
            <span>SHA-256</span><span>presigned URLs</span><span>zero egress</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</div>

<!-- ═══════════════════════════════════════════════════════════════════
     META PLANE — inode-inspired data model
     ═══════════════════════════════════════════════════════════════════ -->
<div class="section meta-section" id="meta-plane">
  <div class="section-iso"></div>
  <div class="section-inner">
    <div class="arch-label">Meta Plane</div>
    <div class="arch-h">Paths and content live apart</div>
    <div class="arch-sub">File identity is decoupled from file content. A path points to a content hash, like how a filename points to an inode. This makes moves, renames, and dedup instant.</div>

    <div class="meta-split">
      <!-- Files table -->
      <div class="meta-diagram">
        <div class="meta-diagram-head"><span class="meta-dot"></span> files</div>
        <table class="meta-table">
          <tr><th>path</th><th>addr</th><th>size</th><th>tx</th></tr>
          <tr><td>report.pdf</td><td class="meta-highlight">a1b2c3d4...</td><td class="meta-dim">200 KB</td><td class="meta-dim">1</td></tr>
          <tr><td>data/config.json</td><td class="meta-highlight">d4e5f6a7...</td><td class="meta-dim">512 B</td><td class="meta-dim">2</td></tr>
          <tr><td>backup/report.pdf</td><td class="meta-highlight">a1b2c3d4...</td><td class="meta-dim">200 KB</td><td class="meta-dim">6</td></tr>
        </table>
      </div>

      <!-- Blobs ref table -->
      <div class="meta-diagram">
        <div class="meta-diagram-head"><span class="meta-dot"></span> blobs</div>
        <table class="meta-table">
          <tr><th>addr</th><th>ref_count</th></tr>
          <tr><td class="meta-highlight">a1b2c3d4...</td><td>2</td></tr>
          <tr><td class="meta-highlight">d4e5f6a7...</td><td>1</td></tr>
        </table>
      </div>
    </div>

    <!-- Operations enabled by this model -->
    <div class="meta-ops">
      <div class="meta-op">
        <div class="meta-op-name">Move / Rename <span class="meta-op-badge">O(1)</span></div>
        <div class="meta-op-desc">Update the path column. The blob stays in place. Zero bytes copied, zero storage operations. Instant regardless of file size.</div>
      </div>
      <div class="meta-op">
        <div class="meta-op-name">Deduplication <span class="meta-op-badge">automatic</span></div>
        <div class="meta-op-desc">Two files with identical content share one blob. The addr column matches, ref_count increments. No extra storage consumed.</div>
      </div>
      <div class="meta-op">
        <div class="meta-op-name">Delete <span class="meta-op-badge">ref counted</span></div>
        <div class="meta-op-desc">Remove the file entry, decrement ref_count. The blob is only garbage-collected when no files reference it anymore.</div>
      </div>
      <div class="meta-op">
        <div class="meta-op-name">Integrity <span class="meta-op-badge">built in</span></div>
        <div class="meta-op-desc">The content hash IS the address. If the data doesn't match the hash, corruption is self-evident. No separate checksums needed.</div>
      </div>
    </div>
  </div>
</div>

<!-- ═══════════════════════════════════════════════════════════════════
     REQUEST LIFECYCLE — upload + download flows
     ═══════════════════════════════════════════════════════════════════ -->
<div class="section flow-section" id="lifecycle">
  <div class="section-dots"></div>
  <div class="section-inner">
    <div class="arch-label">Request Lifecycle</div>
    <div class="arch-h">File bytes never touch the API</div>
    <div class="arch-sub">The API server only handles metadata. Uploads and downloads go directly to object storage via presigned URLs.</div>

    <div class="flow-tabs">
      <button class="flow-tab active" onclick="showFlow('upload')">UPLOAD</button>
      <button class="flow-tab" onclick="showFlow('download')">DOWNLOAD</button>
      <button class="flow-tab" onclick="showFlow('mcp-write')">MCP WRITE</button>
    </div>

    <!-- Upload flow -->
    <div class="flow-panel active" id="flow-upload">
      <div class="flow-steps">
        <div class="flow-step">
          <div class="flow-num">01</div>
          <div class="flow-step-content">
            <div class="flow-step-title">Initiate upload</div>
            <div class="flow-step-desc">Client sends path and content type. Edge validates auth, checks quotas, generates a presigned PUT URL.</div>
            <div class="flow-step-code">POST /files/uploads  {path: "report.pdf", content_type: "application/pdf"}</div>
            <div class="flow-step-badge">~30ms</div>
          </div>
        </div>
        <div class="flow-step">
          <div class="flow-num">02</div>
          <div class="flow-step-content">
            <div class="flow-step-title">Direct upload to storage</div>
            <div class="flow-step-desc">Client PUTs file bytes directly to the presigned URL. No proxy, no bandwidth through the API server.</div>
            <div class="flow-step-code">PUT {presigned_url}  --data-binary @report.pdf</div>
            <div class="flow-step-badge">direct to storage</div>
          </div>
        </div>
        <div class="flow-step">
          <div class="flow-num">03</div>
          <div class="flow-step-content">
            <div class="flow-step-title">Confirm &amp; index</div>
            <div class="flow-step-desc">Client confirms completion. Edge computes SHA-256, writes file entry + event to the Meta Plane, updates blob ref count.</div>
            <div class="flow-step-code">POST /files/uploads/complete  {path: "report.pdf"}</div>
            <div class="flow-step-badge">~40ms</div>
          </div>
        </div>
      </div>
    </div>

    <!-- Download flow -->
    <div class="flow-panel" id="flow-download">
      <div class="flow-steps">
        <div class="flow-step">
          <div class="flow-num">01</div>
          <div class="flow-step-content">
            <div class="flow-step-title">Request file</div>
            <div class="flow-step-desc">Client requests a file by path. Edge validates auth, looks up the file in the Meta Plane.</div>
            <div class="flow-step-code">GET /files/report.pdf  -H "Authorization: Bearer $TOKEN"</div>
            <div class="flow-step-badge">~20ms</div>
          </div>
        </div>
        <div class="flow-step">
          <div class="flow-num">02</div>
          <div class="flow-step-content">
            <div class="flow-step-title">Presigned redirect</div>
            <div class="flow-step-desc">Edge generates a time-limited presigned GET URL and returns a 302 redirect. The client follows it automatically.</div>
            <div class="flow-step-code">302 Location: https://storage.../blobs/alice/a1/b2/a1b2c3...</div>
            <div class="flow-step-badge">direct from storage</div>
          </div>
        </div>
      </div>
    </div>

    <!-- MCP write flow -->
    <div class="flow-panel" id="flow-mcp-write">
      <div class="flow-steps">
        <div class="flow-step">
          <div class="flow-num">01</div>
          <div class="flow-step-content">
            <div class="flow-step-title">AI calls storage_write</div>
            <div class="flow-step-desc">Claude, ChatGPT, or any MCP client invokes the storage_write tool with path and content.</div>
            <div class="flow-step-code">tool: storage_write  {path: "notes/summary.md", content: "..."}</div>
            <div class="flow-step-badge">MCP protocol</div>
          </div>
        </div>
        <div class="flow-step">
          <div class="flow-num">02</div>
          <div class="flow-step-content">
            <div class="flow-step-title">MCP server handles internally</div>
            <div class="flow-step-desc">The MCP server uses the same storage engine as REST. It uploads the blob, writes the file entry, and records the event in one step.</div>
            <div class="flow-step-code">engine.put(actor, "notes/summary.md", blob)</div>
            <div class="flow-step-badge">~50ms total</div>
          </div>
        </div>
        <div class="flow-step">
          <div class="flow-num">03</div>
          <div class="flow-step-content">
            <div class="flow-step-title">Result returned to AI</div>
            <div class="flow-step-desc">The tool returns success with a transaction number. The AI can immediately read the file back or share it.</div>
            <div class="flow-step-code">result: {tx: 42, path: "notes/summary.md", size: 1847}</div>
            <div class="flow-step-badge">immediately visible</div>
          </div>
        </div>
      </div>
    </div>
  </div>
</div>

<!-- ═══════════════════════════════════════════════════════════════════
     CONTENT ADDRESSING
     ═══════════════════════════════════════════════════════════════════ -->
<div class="section addr-section" id="content-addressing">
  <div class="section-crosshatch"></div>
  <div class="section-inner">
    <div class="arch-label">Content Addressing</div>
    <div class="arch-h">Same content, one blob</div>
    <div class="arch-sub">Files are stored by their SHA-256 hash. Upload the same file twice, only one copy is stored. Rename a file, zero bytes copied.</div>

    <div class="addr-visual">
      <div class="addr-row addr-row--header">
        <div>FILE PATH</div>
        <div></div>
        <div>BLOB ADDRESS</div>
      </div>
      <div class="addr-row">
        <div class="addr-file">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>
          report.pdf
        </div>
        <div class="addr-arrow">&rarr;</div>
        <div class="addr-hash">blobs/alice/<strong>a1</strong>/<strong>b2</strong>/a1b2c3d4e5f6789...</div>
      </div>
      <div class="addr-row">
        <div class="addr-file">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>
          data/config.json
        </div>
        <div class="addr-arrow">&rarr;</div>
        <div class="addr-hash">blobs/alice/<strong>d4</strong>/<strong>e5</strong>/d4e5f6a7b8c90ab...</div>
      </div>
      <div class="addr-row addr-dedup">
        <div class="addr-file">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>
          backup/report.pdf
        </div>
        <div class="addr-arrow">&rarr;</div>
        <div class="addr-hash">blobs/alice/<strong>a1</strong>/<strong>b2</strong>/a1b2c3d4e5f6789...</div>
      </div>
    </div>

    <div class="addr-props">
      <div class="addr-prop">
        <div class="addr-prop-name">Automatic dedup</div>
        <div class="addr-prop-desc">Same content = same blob. Always.</div>
      </div>
      <div class="addr-prop">
        <div class="addr-prop-name">Free integrity</div>
        <div class="addr-prop-desc">The hash IS the identifier. Corruption is self-evident.</div>
      </div>
      <div class="addr-prop">
        <div class="addr-prop-name">Zero-cost move</div>
        <div class="addr-prop-desc">Only the Meta Plane path changes. No blob copies.</div>
      </div>
      <div class="addr-prop">
        <div class="addr-prop-name">Collision risk ~2<sup>-128</sup></div>
        <div class="addr-prop-desc">Effectively zero. Would take 10<sup>21</sup> years at 10B files/sec.</div>
      </div>
    </div>
  </div>
</div>

<!-- ═══════════════════════════════════════════════════════════════════
     EVENT ARCHITECTURE — dedicated section
     ═══════════════════════════════════════════════════════════════════ -->
<div class="section events-section" id="events">
  <div class="section-scanlines"></div>
  <div class="section-inner">
    <div class="arch-label">Event Architecture</div>
    <div class="arch-h">Every mutation is an event</div>
    <div class="arch-sub">Not just logging. The event log is the primary source of truth for what happened, when, and by whom. Every version, every action, fully replayable.</div>

    <div class="events-timeline">
      <div class="events-header">
        <div>TX</div>
        <div>ACTION</div>
        <div>PATH</div>
        <div>ADDR</div>
        <div>TIME</div>
      </div>
      <div class="events-row">
        <div class="events-tx">1</div>
        <div class="events-action events-action--put">put</div>
        <div class="events-path">readme.md</div>
        <div class="events-hash">a1b2c3d4...</div>
        <div class="events-time">09:14:22</div>
      </div>
      <div class="events-row">
        <div class="events-tx">2</div>
        <div class="events-action events-action--put">put</div>
        <div class="events-path">data/config.json</div>
        <div class="events-hash">d4e5f6a7...</div>
        <div class="events-time">09:14:23</div>
      </div>
      <div class="events-row">
        <div class="events-tx">3</div>
        <div class="events-action events-action--move">move</div>
        <div class="events-path">docs/readme.md</div>
        <div class="events-hash">a1b2c3d4...</div>
        <div class="events-time">09:15:01</div>
      </div>
      <div class="events-row">
        <div class="events-tx">4</div>
        <div class="events-action events-action--delete">delete</div>
        <div class="events-path">old/draft.txt</div>
        <div class="events-hash">&mdash;</div>
        <div class="events-time">09:15:44</div>
      </div>
      <div class="events-row events-row--version">
        <div class="events-tx">5</div>
        <div class="events-action events-action--put">put</div>
        <div class="events-path">data/config.json</div>
        <div class="events-hash">f7g8h9i0...</div>
        <div class="events-time">09:16:12</div>
      </div>
    </div>

    <!-- Event capabilities grid -->
    <div class="events-capabilities">
      <div class="events-cap">
        <div class="events-cap-icon">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M12 8v4l3 3"/><circle cx="12" cy="12" r="10"/></svg>
        </div>
        <div class="events-cap-name">Smart Versioning</div>
        <div class="events-cap-desc">Every write to the same path creates a new event with a new content hash. Previous versions remain in the log. Reconstruct the full history of any file by reading its events.</div>
      </div>
      <div class="events-cap">
        <div class="events-cap-icon">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polyline points="1 4 1 10 7 10"/><path d="M3.51 15a9 9 0 102.13-9.36L1 10"/></svg>
        </div>
        <div class="events-cap-name">Replayable</div>
        <div class="events-cap-desc">Start from tx 0, apply events in order, and reconstruct the complete state of any actor's storage at any point in time. Perfect for disaster recovery and debugging.</div>
      </div>
      <div class="events-cap">
        <div class="events-cap-icon">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>
        </div>
        <div class="events-cap-name">Auditable</div>
        <div class="events-cap-desc">Every event records who did what, when. No action goes untracked. For AI agents operating autonomously, this provides complete accountability and traceability.</div>
      </div>
      <div class="events-cap">
        <div class="events-cap-icon">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/></svg>
        </div>
        <div class="events-cap-name">Realtime Change Tracking</div>
        <div class="events-cap-desc">Agents poll for events since their last known tx number. No need to list all files and diff. Just ask "what changed since tx 42?" and get exactly the new events.</div>
      </div>
    </div>

    <div class="events-props">
      <div class="events-prop">
        <div class="events-prop-name">APPEND-ONLY</div>
        <div class="events-prop-val">Immutable log</div>
        <div class="events-prop-desc">Events are never updated or deleted. Full history preserved.</div>
      </div>
      <div class="events-prop">
        <div class="events-prop-name">MONOTONIC</div>
        <div class="events-prop-val">Dense sequence</div>
        <div class="events-prop-desc">Transaction numbers have no gaps. Ordered by insertion time.</div>
      </div>
      <div class="events-prop">
        <div class="events-prop-name">PER-ACTOR</div>
        <div class="events-prop-val">Isolated counters</div>
        <div class="events-prop-desc">Each actor has its own tx sequence. No cross-actor contention.</div>
      </div>
      <div class="events-prop">
        <div class="events-prop-name">VERSIONED</div>
        <div class="events-prop-val">Every write tracked</div>
        <div class="events-prop-desc">Multiple puts to the same path create a version chain in the log.</div>
      </div>
    </div>
  </div>
</div>

<!-- ═══════════════════════════════════════════════════════════════════
     DUAL IDENTITY — humans + agents
     ═══════════════════════════════════════════════════════════════════ -->
<div class="section identity-section" id="identity">
  <div class="section-glow"></div>
  <div class="section-inner">
    <div class="arch-label">Dual Identity Model</div>
    <div class="arch-h">Humans and agents are equals</div>
    <div class="arch-sub">Both authenticate differently but get the same API surface. No "service accounts" or "bot modes", just actors.</div>

    <div class="identity-grid">
      <!-- Human column -->
      <div class="identity-col">
        <div class="identity-icon">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M20 21v-2a4 4 0 00-4-4H8a4 4 0 00-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>
          <span class="identity-icon-label">Human</span>
        </div>
        <div class="identity-title">Magic link authentication</div>
        <div class="identity-flow">
          <div class="identity-step"><span class="identity-step-num">1</span>Enter email address</div>
          <div class="identity-step"><span class="identity-step-num">2</span>Click magic link in inbox</div>
          <div class="identity-step"><span class="identity-step-num">3</span>Session cookie set automatically</div>
          <div class="identity-step"><span class="identity-step-num">4</span>Use browser, API, or CLI</div>
        </div>
      </div>

      <!-- Agent column -->
      <div class="identity-col">
        <div class="identity-icon">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="3" y="3" width="18" height="18"/><line x1="9" y1="9" x2="9" y2="9.01"/><line x1="15" y1="9" x2="15" y2="9.01"/><path d="M8 13h8a4 4 0 01-8 0"/></svg>
          <span class="identity-icon-label">AI Agent</span>
        </div>
        <div class="identity-title">Ed25519 challenge-response</div>
        <div class="identity-flow">
          <div class="identity-step"><span class="identity-step-num">1</span>Register public key via <code>/auth/register</code></div>
          <div class="identity-step"><span class="identity-step-num">2</span>Request challenge nonce via <code>/auth/challenge</code></div>
          <div class="identity-step"><span class="identity-step-num">3</span>Sign nonce with private key</div>
          <div class="identity-step"><span class="identity-step-num">4</span>Exchange signature for bearer token</div>
        </div>
      </div>
    </div>

    <!-- Unified capabilities table -->
    <div class="identity-unified">
      <div class="identity-unified-head">Same API, same capabilities</div>
      <div class="identity-unified-grid">
        <div class="identity-unified-cell identity-unified-cell--head">CAPABILITY</div>
        <div class="identity-unified-cell identity-unified-cell--head">HUMAN</div>
        <div class="identity-unified-cell identity-unified-cell--head">AGENT</div>

        <div class="identity-unified-cell">Upload files</div>
        <div class="identity-unified-cell">Yes</div>
        <div class="identity-unified-cell">Yes</div>

        <div class="identity-unified-cell">Download files</div>
        <div class="identity-unified-cell">Yes</div>
        <div class="identity-unified-cell">Yes</div>

        <div class="identity-unified-cell">Share files</div>
        <div class="identity-unified-cell">Yes</div>
        <div class="identity-unified-cell">Yes</div>

        <div class="identity-unified-cell">Search files</div>
        <div class="identity-unified-cell">Yes</div>
        <div class="identity-unified-cell">Yes</div>

        <div class="identity-unified-cell">Connect via MCP</div>
        <div class="identity-unified-cell">Yes</div>
        <div class="identity-unified-cell">Yes</div>
      </div>
    </div>
  </div>
</div>

<!-- ═══════════════════════════════════════════════════════════════════
     EDGE-FIRST DESIGN — global architecture
     ═══════════════════════════════════════════════════════════════════ -->
<div class="section edge-section" id="edge">
  <div class="section-rings"></div>
  <div class="section-inner">
    <div class="arch-label">Edge-First</div>
    <div class="arch-h">Global by default</div>
    <div class="arch-sub">No origin server. No single region. The entire application runs inside V8 isolates distributed across every continent.</div>

    <!-- Globe visualization: request path diagram -->
    <div class="edge-globe">
      <div class="edge-globe-title">How a request travels</div>
      <div class="edge-globe-flow">
        <div class="edge-globe-node">
          <div class="edge-globe-icon">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="5" y="2" width="14" height="20" rx="2"/><line x1="12" y1="18" x2="12" y2="18.01"/></svg>
          </div>
          <div class="edge-globe-label">Your client</div>
          <div class="edge-globe-region">Any location</div>
        </div>
        <div class="edge-globe-arrow">
          <svg width="40" height="24" viewBox="0 0 40 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="2" y1="12" x2="34" y2="12"/><polyline points="28 6 34 12 28 18"/></svg>
          <div class="edge-globe-arrow-label">nearest edge</div>
        </div>
        <div class="edge-globe-node edge-globe-node--active">
          <div class="edge-globe-icon">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 014 10 15.3 15.3 0 01-4 10 15.3 15.3 0 01-4-10 15.3 15.3 0 014-10z"/></svg>
          </div>
          <div class="edge-globe-label">Edge node</div>
          <div class="edge-globe-region">Auth + Meta Plane + Presign</div>
        </div>
        <div class="edge-globe-arrow">
          <svg width="40" height="24" viewBox="0 0 40 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="2" y1="12" x2="34" y2="12"/><polyline points="28 6 34 12 28 18"/></svg>
          <div class="edge-globe-arrow-label">presigned URL</div>
        </div>
        <div class="edge-globe-node">
          <div class="edge-globe-icon">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 16V8a2 2 0 00-1-1.73l-7-4a2 2 0 00-2 0l-7 4A2 2 0 003 8v8a2 2 0 001 1.73l7 4a2 2 0 002 0l7-4A2 2 0 0021 16z"/></svg>
          </div>
          <div class="edge-globe-label">Object store</div>
          <div class="edge-globe-region">Direct transfer</div>
        </div>
      </div>
    </div>

    <!-- Traditional vs Edge comparison -->
    <div class="edge-compare">
      <div class="edge-compare-col">
        <div class="edge-compare-head">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="2" y="2" width="20" height="20" rx="2"/><line x1="12" y1="2" x2="12" y2="22"/><line x1="2" y1="12" x2="22" y2="12"/></svg>
          Traditional (single origin)
        </div>
        <div class="edge-compare-items">
          <div class="edge-compare-item edge-compare-item--bad">All requests route to one region</div>
          <div class="edge-compare-item edge-compare-item--bad">File bytes proxy through the API</div>
          <div class="edge-compare-item edge-compare-item--bad">Cold starts from container spin-up</div>
          <div class="edge-compare-item edge-compare-item--bad">Latency scales with distance to origin</div>
        </div>
      </div>
      <div class="edge-compare-col edge-compare-col--good">
        <div class="edge-compare-head">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 014 10 15.3 15.3 0 01-4 10 15.3 15.3 0 01-4-10 15.3 15.3 0 014-10z"/></svg>
          Storage (global edge)
        </div>
        <div class="edge-compare-items">
          <div class="edge-compare-item edge-compare-item--good">Requests hit the nearest edge node</div>
          <div class="edge-compare-item edge-compare-item--good">Direct transfers via presigned URLs</div>
          <div class="edge-compare-item edge-compare-item--good">V8 isolate starts in milliseconds</div>
          <div class="edge-compare-item edge-compare-item--good">Consistent performance everywhere</div>
        </div>
      </div>
    </div>

    <!-- Why edge matters cards -->
    <div class="edge-why">
      <div class="edge-why-card">
        <div class="edge-why-icon">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="3" y="3" width="18" height="18"/><line x1="9" y1="9" x2="9" y2="9.01"/><line x1="15" y1="9" x2="15" y2="9.01"/><path d="M8 13h8a4 4 0 01-8 0"/></svg>
        </div>
        <div class="edge-why-name">For AI agents</div>
        <div class="edge-why-desc">Agents in cloud functions have their own cold starts. Adding network round-trips to a distant origin makes tool calls slow. Edge keeps the API fast wherever the agent runs.</div>
      </div>
      <div class="edge-why-card">
        <div class="edge-why-icon">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M20 21v-2a4 4 0 00-4-4H8a4 4 0 00-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>
        </div>
        <div class="edge-why-name">For humans</div>
        <div class="edge-why-desc">The web dashboard, file browser, and share links all load from the nearest edge. No matter where your team is located, the experience is the same.</div>
      </div>
      <div class="edge-why-card">
        <div class="edge-why-icon">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M17 21v-2a4 4 0 00-4-4H5a4 4 0 00-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 00-3-3.87"/><path d="M16 3.13a4 4 0 010 7.75"/></svg>
        </div>
        <div class="edge-why-name">For collaboration</div>
        <div class="edge-why-desc">When a human in Tokyo and an agent in Virginia share the same storage, both get fast responses. No single region becomes a bottleneck for the team.</div>
      </div>
    </div>
  </div>
</div>

<!-- ═══════════════════════════════════════════════════════════════════
     MCP — native AI integration
     ═══════════════════════════════════════════════════════════════════ -->
<div class="section mcp-section" id="mcp">
  <div class="section-dots"></div>
  <div class="section-inner">
    <div class="arch-label">MCP Integration</div>
    <div class="arch-h">AI-native, not AI-bolted</div>
    <div class="arch-sub">MCP tools map directly to storage operations. Same engine as REST. A file uploaded via API is instantly visible to Claude or ChatGPT.</div>

    <!-- Flow diagram -->
    <div class="mcp-flow">
      <div class="mcp-flow-node">
        <div class="mcp-flow-node-icon">
          <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="3" y="3" width="18" height="18"/><line x1="9" y1="9" x2="9" y2="9.01"/><line x1="15" y1="9" x2="15" y2="9.01"/><path d="M8 13h8a4 4 0 01-8 0"/></svg>
        </div>
        <div class="mcp-flow-node-name">AI Client</div>
        <div class="mcp-flow-node-desc">Claude, ChatGPT, custom</div>
      </div>
      <div class="mcp-flow-arrow">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="5" y1="12" x2="19" y2="12"/><polyline points="12 5 19 12 12 19"/></svg>
      </div>
      <div class="mcp-flow-node">
        <div class="mcp-flow-node-icon">
          <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/></svg>
        </div>
        <div class="mcp-flow-node-name">MCP Server</div>
        <div class="mcp-flow-node-desc">OAuth + 8 tools</div>
      </div>
      <div class="mcp-flow-arrow">
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="5" y1="12" x2="19" y2="12"/><polyline points="12 5 19 12 12 19"/></svg>
      </div>
      <div class="mcp-flow-node">
        <div class="mcp-flow-node-icon">
          <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>
        </div>
        <div class="mcp-flow-node-name">Storage Engine</div>
        <div class="mcp-flow-node-desc">Meta Plane + Object Store</div>
      </div>
    </div>

    <!-- Tool mapping table -->
    <div class="mcp-tools">
      <div class="mcp-tools-head">
        <div>MCP TOOL</div>
        <div>REST EQUIVALENT</div>
        <div>DESCRIPTION</div>
      </div>
      <div class="mcp-tool">
        <div class="mcp-tool-name">storage_read</div>
        <div class="mcp-tool-rest">GET /files/{path}</div>
        <div class="mcp-tool-desc">Read file contents</div>
      </div>
      <div class="mcp-tool">
        <div class="mcp-tool-name">storage_write</div>
        <div class="mcp-tool-rest">POST /files/uploads</div>
        <div class="mcp-tool-desc">Write or overwrite a file</div>
      </div>
      <div class="mcp-tool">
        <div class="mcp-tool-name">storage_list</div>
        <div class="mcp-tool-rest">GET /files?prefix=</div>
        <div class="mcp-tool-desc">List files in a folder</div>
      </div>
      <div class="mcp-tool">
        <div class="mcp-tool-name">storage_search</div>
        <div class="mcp-tool-rest">GET /files/search</div>
        <div class="mcp-tool-desc">Search files by name</div>
      </div>
      <div class="mcp-tool">
        <div class="mcp-tool-name">storage_share</div>
        <div class="mcp-tool-rest">POST /files/share</div>
        <div class="mcp-tool-desc">Create a temporary public link</div>
      </div>
      <div class="mcp-tool">
        <div class="mcp-tool-name">storage_move</div>
        <div class="mcp-tool-rest">POST /files/move</div>
        <div class="mcp-tool-desc">Move or rename a file</div>
      </div>
      <div class="mcp-tool">
        <div class="mcp-tool-name">storage_delete</div>
        <div class="mcp-tool-rest">DELETE /files/{path}</div>
        <div class="mcp-tool-desc">Delete a file</div>
      </div>
      <div class="mcp-tool">
        <div class="mcp-tool-name">storage_stats</div>
        <div class="mcp-tool-rest">GET /files/stats</div>
        <div class="mcp-tool-desc">Get storage usage</div>
      </div>
    </div>
  </div>
</div>

<!-- ═══════════════════════════════════════════════════════════════════
     WHY AI AGENTS — reasons grid
     ═══════════════════════════════════════════════════════════════════ -->
<div class="section why-section" id="why-ai">
  <div class="section-crosshatch"></div>
  <div class="section-inner">
    <div class="arch-label">Why AI Agents Love This</div>
    <div class="arch-h">Designed for autonomous operation</div>
    <div class="arch-sub">Every architectural decision was made with AI agents in mind, not as an afterthought but as a primary use case.</div>

    <div class="why-grid">
      <div class="why-item">
        <div class="why-num">01</div>
        <div class="why-content">
          <div class="why-item-title">No SDK required</div>
          <div class="why-item-desc">Plain HTTP with JSON. Any agent runtime can call Storage with zero dependencies.</div>
        </div>
      </div>
      <div class="why-item">
        <div class="why-num">02</div>
        <div class="why-content">
          <div class="why-item-title">Deterministic responses</div>
          <div class="why-item-desc">Consistent JSON schemas, typed with Zod, documented via OpenAPI. No HTML parsing, no brittle scraping.</div>
        </div>
      </div>
      <div class="why-item">
        <div class="why-num">03</div>
        <div class="why-content">
          <div class="why-item-title">MCP native</div>
          <div class="why-item-desc">8 tools map directly to storage operations. Connect once, then read, write, search, and share from any MCP-compatible AI.</div>
        </div>
      </div>
      <div class="why-item">
        <div class="why-num">04</div>
        <div class="why-content">
          <div class="why-item-title">Incremental sync</div>
          <div class="why-item-desc">Event sourcing means agents poll for changes since their last known tx. No full directory scans required.</div>
        </div>
      </div>
      <div class="why-item">
        <div class="why-num">05</div>
        <div class="why-content">
          <div class="why-item-title">Automatic deduplication</div>
          <div class="why-item-desc">Content addressing means agents don't waste storage re-uploading the same file. SHA-256 handles it.</div>
        </div>
      </div>
      <div class="why-item">
        <div class="why-num">06</div>
        <div class="why-content">
          <div class="why-item-title">Sub-50ms latency</div>
          <div class="why-item-desc">Fast enough for tool calls inside LLM inference loops. Edge runtime eliminates round-trip to a single origin.</div>
        </div>
      </div>
      <div class="why-item">
        <div class="why-num">07</div>
        <div class="why-content">
          <div class="why-item-title">First-class identity</div>
          <div class="why-item-desc">Agents are actors, not hacks on user accounts. Ed25519 key auth designed for programmatic access from day one.</div>
        </div>
      </div>
      <div class="why-item">
        <div class="why-num">08</div>
        <div class="why-content">
          <div class="why-item-title">Scoped permissions</div>
          <div class="why-item-desc">API keys with path-prefix restrictions. Give an agent access to <code>data/</code> but not <code>secrets/</code>. 90-day TTL auto-rotation.</div>
        </div>
      </div>
      <div class="why-item">
        <div class="why-num">09</div>
        <div class="why-content">
          <div class="why-item-title">Full audit trail</div>
          <div class="why-item-desc">Every agent action is logged with actor, resource, and timestamp. Debug agent behavior with complete history.</div>
        </div>
      </div>
      <div class="why-item">
        <div class="why-num">10</div>
        <div class="why-content">
          <div class="why-item-title">Zero egress fees</div>
          <div class="why-item-desc">Agents can read files as often as they need without cost anxiety. No bandwidth metering, ever.</div>
        </div>
      </div>
    </div>
  </div>
</div>

<!-- ═══════════════════════════════════════════════════════════════════
     TECHNOLOGY STACK — card grid
     ═══════════════════════════════════════════════════════════════════ -->
<div class="section stack-section" id="stack">
  <div class="section-dots"></div>
  <div class="section-inner">
    <div class="arch-label">Technology Stack</div>
    <div class="arch-h">Chosen for simplicity</div>
    <div class="arch-sub">Every component is the simplest technology that solves the problem well. No over-engineering.</div>

    <div class="tech-grid">
      <div class="tech-card">
        <div class="tech-card-cat">COMPUTE</div>
        <div class="tech-card-name">Edge Workers</div>
        <div class="tech-card-desc">V8 isolates distributed globally. Sub-5ms cold starts. No containers, no VMs, no orchestration.</div>
      </div>
      <div class="tech-card">
        <div class="tech-card-cat">FRAMEWORK</div>
        <div class="tech-card-name">Hono + OpenAPI</div>
        <div class="tech-card-desc">Type-safe routes with auto-generated API documentation. Zod schemas validate at runtime and generate OpenAPI specs.</div>
      </div>
      <div class="tech-card">
        <div class="tech-card-cat">METADATA</div>
        <div class="tech-card-name">Meta Plane</div>
        <div class="tech-card-desc">Fast reads, strong consistency, zero configuration. Stores file index, events, sessions, and blob references.</div>
      </div>
      <div class="tech-card">
        <div class="tech-card-cat">STORAGE</div>
        <div class="tech-card-name">Object Store</div>
        <div class="tech-card-desc">S3-compatible, globally distributed, zero egress fees. Content-addressed blobs with presigned URL access.</div>
      </div>
      <div class="tech-card">
        <div class="tech-card-cat">AUTH</div>
        <div class="tech-card-name">Ed25519 + Magic Links</div>
        <div class="tech-card-desc">Public-key challenge-response for agents. Email magic links for humans. No passwords to manage or leak.</div>
      </div>
      <div class="tech-card">
        <div class="tech-card-cat">PROTOCOL</div>
        <div class="tech-card-name">REST + MCP</div>
        <div class="tech-card-desc">Plain HTTP for universal access. Model Context Protocol for AI-native integration. Same engine, two interfaces.</div>
      </div>
      <div class="tech-card">
        <div class="tech-card-cat">VALIDATION</div>
        <div class="tech-card-name">Zod</div>
        <div class="tech-card-desc">Runtime type safety for all request and response schemas. Generates OpenAPI specs automatically from route definitions.</div>
      </div>
      <div class="tech-card">
        <div class="tech-card-cat">SECURITY</div>
        <div class="tech-card-name">OAuth 2.0 + PKCE</div>
        <div class="tech-card-desc">Standard flow for third-party apps and MCP clients. Dynamic client registration. Scoped API keys with TTL.</div>
      </div>
      <div class="tech-card">
        <div class="tech-card-cat">OBSERVABILITY</div>
        <div class="tech-card-name">Audit Logging</div>
        <div class="tech-card-desc">Every action logged with actor, resource, and timestamp. Event-sourced transaction log doubles as audit trail.</div>
      </div>
    </div>
  </div>
</div>

<!-- ═══════════════════════════════════════════════════════════════════
     CTA
     ═══════════════════════════════════════════════════════════════════ -->
<div class="section cta-section">
  <div class="section-inner section-inner--center">
    <div class="cta-title">Start building today</div>
    <div class="cta-sub">Free plan. No credit card. Agents welcome.</div>
    <div class="cta-actions">
      <a href="/developers" class="btn btn--primary">Developer Guide</a>
      <a href="/api" class="btn">API Reference</a>
      <a href="/cli" class="btn">CLI Docs</a>
    </div>
  </div>
</div>

</div><!-- /human-view -->

<!-- ===== MACHINE VIEW ===== -->
<div class="machine-view" id="machine-view">
  <div class="md" id="md-content"><button class="md-copy" onclick="copyMd()">copy</button>${markdownToHtml(architectureMd)}</div>
</div>

<!-- Floating mode switch -->
<div class="mode-switch">
  <button class="active" onclick="setMode('human')"><span class="dot"></span> HUMAN</button>
  <button onclick="setMode('machine')"><span class="dot"></span> MACHINE</button>
</div>

<script>
/* Theme */
function toggleTheme(){
  const isDark=document.documentElement.classList.toggle('dark');
  localStorage.setItem('theme',isDark?'dark':'light');
}
(function(){
  const saved=localStorage.getItem('theme');
  if(saved==='light'){
    document.documentElement.classList.remove('dark');
  } else if(!saved&&!window.matchMedia('(prefers-color-scheme:dark)').matches){
    document.documentElement.classList.remove('dark');
  }
})();

/* Mode switch */
function setMode(mode){
  var btns=document.querySelectorAll('.mode-switch button');
  btns.forEach(function(b){b.classList.remove('active')});
  if(mode==='human'){
    btns[0].classList.add('active');
    document.getElementById('human-view').classList.remove('hidden');
    document.getElementById('machine-view').classList.remove('active');
  } else {
    btns[1].classList.add('active');
    document.getElementById('human-view').classList.add('hidden');
    document.getElementById('machine-view').classList.add('active');
  }
}

/* Machine view copy */
function copyMd(){
  var el=document.getElementById('md-content');
  var text=el.innerText.replace(/^copy\\n/,'');
  navigator.clipboard.writeText(text).then(function(){
    var btn=el.querySelector('.md-copy');
    btn.textContent='copied';
    setTimeout(function(){btn.textContent='copy'},2000);
  });
}

/* Flow tabs */
function showFlow(id){
  document.querySelectorAll('.flow-tab').forEach(function(t){t.classList.remove('active')});
  document.querySelectorAll('.flow-panel').forEach(function(p){p.classList.remove('active')});
  event.target.classList.add('active');
  document.getElementById('flow-'+id).classList.add('active');
}

/* Scroll-triggered fade-up */
(function(){
  var sections=document.querySelectorAll('.section');
  var observer=new IntersectionObserver(function(entries){
    entries.forEach(function(e){
      if(e.isIntersecting){e.target.classList.add('visible');observer.unobserve(e.target)}
    });
  },{threshold:0.08});
  sections.forEach(function(s){observer.observe(s)});
})();
</script>
</body>
</html>`;
}
