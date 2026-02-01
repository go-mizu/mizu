package dashboard

// HTML is the full dashboard SPA HTML page.
// It embeds all CSS and JavaScript inline with no external dependencies.
const HTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>OpenClaw Control Dashboard</title>
<style>
*, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }

:root {
  --bg: #0d1117;
  --bg-card: #161b22;
  --bg-input: #0d1117;
  --bg-hover: #1c2333;
  --bg-sidebar: #0d1117;
  --border: #30363d;
  --text: #c9d1d9;
  --text-dim: #8b949e;
  --text-bright: #f0f6fc;
  --accent: #58a6ff;
  --accent-dim: #1f6feb;
  --green: #3fb950;
  --red: #f85149;
  --yellow: #d29922;
  --orange: #db6d28;
  --purple: #bc8cff;
  --font: -apple-system, BlinkMacSystemFont, 'Segoe UI', Helvetica, Arial, sans-serif;
  --mono: 'SFMono-Regular', Consolas, 'Liberation Mono', Menlo, monospace;
  --sidebar-w: 220px;
  --radius: 6px;
}

html, body { height: 100%; }

body {
  font-family: var(--font);
  background: var(--bg);
  color: var(--text);
  display: flex;
  overflow: hidden;
}

/* ---- Sidebar ---- */
.sidebar {
  width: var(--sidebar-w);
  min-width: var(--sidebar-w);
  background: var(--bg-sidebar);
  border-right: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  overflow-y: auto;
  padding: 0;
}

.sidebar-header {
  padding: 16px 16px 8px;
  font-size: 14px;
  font-weight: 600;
  color: var(--text-bright);
  display: flex;
  align-items: center;
  gap: 8px;
  border-bottom: 1px solid var(--border);
  margin-bottom: 4px;
}

.sidebar-header .logo {
  width: 20px;
  height: 20px;
  background: var(--accent);
  border-radius: 4px;
  display: inline-block;
  flex-shrink: 0;
}

.nav-group {
  padding: 4px 0;
}

.nav-group-label {
  padding: 6px 16px 4px;
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: var(--text-dim);
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 16px;
  font-size: 13px;
  color: var(--text);
  cursor: pointer;
  border-left: 2px solid transparent;
  transition: background 0.15s, border-color 0.15s;
  user-select: none;
}

.nav-item:hover {
  background: var(--bg-hover);
}

.nav-item.active {
  background: var(--bg-hover);
  color: var(--text-bright);
  border-left-color: var(--accent);
}

.nav-item .icon {
  font-size: 15px;
  width: 18px;
  text-align: center;
  flex-shrink: 0;
}

.sidebar-footer {
  margin-top: auto;
  padding: 12px 16px;
  border-top: 1px solid var(--border);
  font-size: 11px;
  color: var(--text-dim);
  display: flex;
  align-items: center;
  gap: 6px;
}

.ws-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--red);
  flex-shrink: 0;
}

.ws-dot.connected { background: var(--green); }

/* ---- Main Content ---- */
.main {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.topbar {
  height: 48px;
  border-bottom: 1px solid var(--border);
  display: flex;
  align-items: center;
  padding: 0 20px;
  gap: 12px;
  flex-shrink: 0;
}

.topbar-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--text-bright);
}

.topbar-right {
  margin-left: auto;
  display: flex;
  align-items: center;
  gap: 10px;
}

.content {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
}

/* ---- Cards ---- */
.card {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 16px;
  margin-bottom: 16px;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.card-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-bright);
}

.stat-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 12px;
}

.stat-card {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 14px;
}

.stat-label {
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: var(--text-dim);
  margin-bottom: 4px;
}

.stat-value {
  font-size: 22px;
  font-weight: 600;
  color: var(--text-bright);
}

/* ---- Badges ---- */
.badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 10px;
  font-size: 11px;
  font-weight: 500;
  line-height: 1.5;
}

.badge-green  { background: rgba(63,185,80,0.15); color: var(--green); }
.badge-red    { background: rgba(248,81,73,0.15); color: var(--red); }
.badge-yellow { background: rgba(210,153,34,0.15); color: var(--yellow); }
.badge-blue   { background: rgba(88,166,255,0.15); color: var(--accent); }
.badge-purple { background: rgba(188,140,255,0.15); color: var(--purple); }
.badge-dim    { background: rgba(139,148,158,0.15); color: var(--text-dim); }

/* ---- Tables ---- */
.table-wrap {
  overflow-x: auto;
}

table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}

th {
  text-align: left;
  padding: 8px 12px;
  font-weight: 600;
  color: var(--text-dim);
  border-bottom: 1px solid var(--border);
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  white-space: nowrap;
}

td {
  padding: 8px 12px;
  border-bottom: 1px solid var(--border);
  color: var(--text);
  vertical-align: middle;
}

tr:hover td {
  background: var(--bg-hover);
}

td.mono {
  font-family: var(--mono);
  font-size: 12px;
}

/* ---- Forms ---- */
input[type="text"], input[type="password"], input[type="number"],
textarea, select {
  background: var(--bg-input);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  color: var(--text);
  padding: 8px 12px;
  font-size: 13px;
  font-family: var(--font);
  outline: none;
  transition: border-color 0.15s;
  width: 100%;
}

input:focus, textarea:focus, select:focus {
  border-color: var(--accent);
}

textarea {
  resize: vertical;
  font-family: var(--mono);
  font-size: 12px;
  line-height: 1.5;
}

select {
  cursor: pointer;
  appearance: none;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='10' height='6'%3E%3Cpath d='M0 0l5 6 5-6z' fill='%238b949e'/%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right 10px center;
  padding-right: 28px;
}

label {
  display: block;
  font-size: 12px;
  font-weight: 500;
  color: var(--text-dim);
  margin-bottom: 4px;
}

.form-row {
  margin-bottom: 12px;
}

.form-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}

/* ---- Buttons ---- */
.btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 14px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--bg-card);
  color: var(--text);
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.15s, border-color 0.15s;
  font-family: var(--font);
  white-space: nowrap;
}

.btn:hover {
  background: var(--bg-hover);
  border-color: var(--text-dim);
}

.btn-primary {
  background: var(--accent-dim);
  border-color: var(--accent-dim);
  color: #fff;
}

.btn-primary:hover {
  background: var(--accent);
  border-color: var(--accent);
}

.btn-danger {
  color: var(--red);
}

.btn-danger:hover {
  background: rgba(248,81,73,0.1);
  border-color: var(--red);
}

.btn-sm {
  padding: 3px 8px;
  font-size: 11px;
}

.btn-icon {
  padding: 4px 6px;
  min-width: 28px;
  justify-content: center;
}

/* ---- Toggle switch ---- */
.toggle {
  position: relative;
  display: inline-block;
  width: 34px;
  height: 18px;
  cursor: pointer;
}

.toggle input { display: none; }

.toggle .slider {
  position: absolute;
  inset: 0;
  background: var(--border);
  border-radius: 9px;
  transition: background 0.2s;
}

.toggle .slider::before {
  content: '';
  position: absolute;
  width: 14px;
  height: 14px;
  left: 2px;
  top: 2px;
  background: var(--text-dim);
  border-radius: 50%;
  transition: transform 0.2s, background 0.2s;
}

.toggle input:checked + .slider {
  background: var(--accent-dim);
}

.toggle input:checked + .slider::before {
  transform: translateX(16px);
  background: #fff;
}

/* ---- Chat ---- */
.chat-container {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.chat-toolbar {
  display: flex;
  gap: 10px;
  align-items: center;
  margin-bottom: 12px;
  flex-shrink: 0;
}

.chat-messages {
  flex: 1;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 8px 0;
  min-height: 0;
}

.chat-msg {
  max-width: 80%;
  padding: 10px 14px;
  border-radius: 12px;
  font-size: 13px;
  line-height: 1.5;
  word-break: break-word;
  white-space: pre-wrap;
}

.chat-msg.user {
  align-self: flex-end;
  background: var(--accent-dim);
  color: #fff;
  border-bottom-right-radius: 4px;
}

.chat-msg.assistant {
  align-self: flex-start;
  background: var(--bg-card);
  border: 1px solid var(--border);
  color: var(--text);
  border-bottom-left-radius: 4px;
}

.chat-msg .msg-meta {
  font-size: 10px;
  color: var(--text-dim);
  margin-top: 4px;
}

.chat-msg.user .msg-meta { color: rgba(255,255,255,0.6); }

.chat-input-row {
  display: flex;
  gap: 8px;
  align-items: flex-end;
  flex-shrink: 0;
  padding-top: 12px;
  border-top: 1px solid var(--border);
}

.chat-input-row textarea {
  flex: 1;
  min-height: 40px;
  max-height: 120px;
  resize: none;
}

/* ---- Log Viewer ---- */
.log-viewer {
  background: var(--bg);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  font-family: var(--mono);
  font-size: 12px;
  line-height: 1.6;
  overflow-y: auto;
  max-height: calc(100vh - 260px);
  padding: 8px;
}

.log-entry {
  display: flex;
  gap: 8px;
  padding: 2px 4px;
  border-radius: 3px;
}

.log-entry:hover {
  background: var(--bg-hover);
}

.log-ts {
  color: var(--text-dim);
  flex-shrink: 0;
  min-width: 80px;
}

.log-level {
  flex-shrink: 0;
  min-width: 48px;
  font-weight: 600;
  text-transform: uppercase;
  font-size: 10px;
  padding-top: 2px;
}

.log-level.info  { color: var(--accent); }
.log-level.warn  { color: var(--yellow); }
.log-level.error { color: var(--red); }
.log-level.debug { color: var(--text-dim); }

.log-sub {
  color: var(--purple);
  flex-shrink: 0;
  min-width: 80px;
}

.log-msg {
  color: var(--text);
  flex: 1;
  word-break: break-all;
}

/* ---- RPC Console ---- */
.rpc-result {
  background: var(--bg);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 12px;
  font-family: var(--mono);
  font-size: 12px;
  line-height: 1.5;
  max-height: 400px;
  overflow-y: auto;
  white-space: pre-wrap;
  word-break: break-all;
  color: var(--text);
}

/* ---- Page containers ---- */
.page { display: none; height: 100%; }
.page.active { display: block; }
.page.active.flex-page { display: flex; flex-direction: column; }

/* ---- Placeholder ---- */
.placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 300px;
  color: var(--text-dim);
  gap: 12px;
}

.placeholder .ph-icon { font-size: 40px; }
.placeholder .ph-text { font-size: 15px; }
.placeholder .ph-sub  { font-size: 12px; }

/* ---- Filter row ---- */
.filter-row {
  display: flex;
  gap: 10px;
  align-items: center;
  margin-bottom: 14px;
  flex-wrap: wrap;
}

.filter-row input[type="text"],
.filter-row select {
  width: auto;
  min-width: 160px;
}

/* ---- Scrollbar ---- */
::-webkit-scrollbar { width: 8px; height: 8px; }
::-webkit-scrollbar-track { background: transparent; }
::-webkit-scrollbar-thumb { background: var(--border); border-radius: 4px; }
::-webkit-scrollbar-thumb:hover { background: var(--text-dim); }

/* ---- Utility ---- */
.flex-between { display: flex; align-items: center; justify-content: space-between; }
.gap-8 { gap: 8px; }
.mt-8 { margin-top: 8px; }
.mt-12 { margin-top: 12px; }
.mb-8 { margin-bottom: 8px; }
.text-dim { color: var(--text-dim); }
.text-sm { font-size: 12px; }
.text-xs { font-size: 11px; }
.mono { font-family: var(--mono); }
.hidden { display: none !important; }
.truncate { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.inline-flex { display: inline-flex; align-items: center; gap: 4px; }
</style>
</head>
<body>

<!-- ======== SIDEBAR ======== -->
<nav class="sidebar">
  <div class="sidebar-header">
    <span class="logo"></span> Control Dashboard
  </div>

  <div class="nav-group">
    <div class="nav-group-label">Chat</div>
    <div class="nav-item active" data-page="chat" onclick="showPage('chat')">
      <span class="icon">&#9993;</span> Chat
    </div>
  </div>

  <div class="nav-group">
    <div class="nav-group-label">Control</div>
    <div class="nav-item" data-page="overview" onclick="showPage('overview')">
      <span class="icon">&#9673;</span> Overview
    </div>
    <div class="nav-item" data-page="channels" onclick="showPage('channels')">
      <span class="icon">&#9878;</span> Channels
    </div>
    <div class="nav-item" data-page="instances" onclick="showPage('instances')">
      <span class="icon">&#8801;</span> Instances
    </div>
    <div class="nav-item" data-page="sessions" onclick="showPage('sessions')">
      <span class="icon">&#8596;</span> Sessions
    </div>
    <div class="nav-item" data-page="cron" onclick="showPage('cron')">
      <span class="icon">&#8986;</span> Cron Jobs
    </div>
  </div>

  <div class="nav-group">
    <div class="nav-group-label">Agent</div>
    <div class="nav-item" data-page="skills" onclick="showPage('skills')">
      <span class="icon">&#9881;</span> Skills
    </div>
    <div class="nav-item" data-page="nodes" onclick="showPage('nodes')">
      <span class="icon">&#9671;</span> Nodes
    </div>
  </div>

  <div class="nav-group">
    <div class="nav-group-label">Settings</div>
    <div class="nav-item" data-page="config" onclick="showPage('config')">
      <span class="icon">&#9881;</span> Config
    </div>
    <div class="nav-item" data-page="debug" onclick="showPage('debug')">
      <span class="icon">&#9888;</span> Debug
    </div>
    <div class="nav-item" data-page="logs" onclick="showPage('logs')">
      <span class="icon">&#9776;</span> Logs
    </div>
  </div>

  <div class="sidebar-footer">
    <span class="ws-dot" id="wsDot"></span>
    <span id="wsLabel">Disconnected</span>
  </div>
</nav>

<!-- ======== MAIN ======== -->
<div class="main">
  <div class="topbar">
    <span class="topbar-title" id="topbarTitle">Chat</span>
    <div class="topbar-right">
      <span class="text-xs text-dim" id="topbarStatus"></span>
    </div>
  </div>

  <div class="content">

    <!-- ======== CHAT PAGE ======== -->
    <div id="page-chat" class="page active flex-page">
      <div class="chat-container">
        <div class="chat-toolbar">
          <label class="text-xs" style="margin:0">Session:</label>
          <select id="chatSessionSelect" style="width:220px" onchange="loadChatHistory()">
            <option value="">New session</option>
          </select>
          <button class="btn btn-sm" onclick="loadChatSessions()">Refresh</button>
        </div>
        <div class="chat-messages" id="chatMessages"></div>
        <div class="chat-input-row">
          <textarea id="chatInput" rows="1" placeholder="Type a message..." onkeydown="chatKeydown(event)"></textarea>
          <button class="btn btn-primary" onclick="sendChat()" id="chatSendBtn">Send</button>
        </div>
      </div>
    </div>

    <!-- ======== OVERVIEW PAGE ======== -->
    <div id="page-overview" class="page">
      <div class="card">
        <div class="card-header">
          <span class="card-title">Gateway Connection</span>
          <button class="btn btn-sm" onclick="connectWs()" id="overviewConnBtn">Connect</button>
        </div>
        <div class="form-grid" style="max-width:600px">
          <div class="form-row">
            <label>WebSocket URL</label>
            <input type="text" id="gwUrl" readonly>
          </div>
          <div class="form-row">
            <label>Token (optional)</label>
            <input type="password" id="gwToken" placeholder="Enter token...">
          </div>
        </div>
      </div>
      <div class="stat-grid" id="overviewStats">
        <div class="stat-card"><div class="stat-label">Status</div><div class="stat-value" id="ovStatus">--</div></div>
        <div class="stat-card"><div class="stat-label">Uptime</div><div class="stat-value" id="ovUptime">--</div></div>
        <div class="stat-card"><div class="stat-label">Agents</div><div class="stat-value" id="ovAgents">--</div></div>
        <div class="stat-card"><div class="stat-label">Channels</div><div class="stat-value" id="ovChannels">--</div></div>
        <div class="stat-card"><div class="stat-label">Sessions</div><div class="stat-value" id="ovSessions">--</div></div>
        <div class="stat-card"><div class="stat-label">Messages</div><div class="stat-value" id="ovMessages">--</div></div>
      </div>
      <div class="card mt-12">
        <div class="card-header">
          <span class="card-title">Cron Scheduler</span>
        </div>
        <div id="ovCronStatus" class="text-dim text-sm">Loading...</div>
      </div>
    </div>

    <!-- ======== CHANNELS PAGE ======== -->
    <div id="page-channels" class="page">
      <div class="flex-between mb-8">
        <span></span>
        <button class="btn btn-sm" onclick="loadChannels()">Refresh</button>
      </div>
      <div id="channelsList"></div>
    </div>

    <!-- ======== INSTANCES PAGE ======== -->
    <div id="page-instances" class="page">
      <div class="flex-between mb-8">
        <span class="text-sm text-dim" id="instanceCount"></span>
        <button class="btn btn-sm" onclick="loadInstances()">Refresh</button>
      </div>
      <div class="card">
        <div class="table-wrap">
          <table>
            <thead>
              <tr>
                <th>ID</th>
                <th>Remote Address</th>
                <th>Connected</th>
                <th>User Agent</th>
              </tr>
            </thead>
            <tbody id="instancesBody"></tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- ======== SESSIONS PAGE ======== -->
    <div id="page-sessions" class="page">
      <div class="filter-row">
        <input type="text" id="sessionFilter" placeholder="Filter sessions..." oninput="filterSessions()">
        <button class="btn btn-sm" onclick="loadSessions()">Refresh</button>
      </div>
      <div class="card">
        <div class="table-wrap">
          <table>
            <thead>
              <tr>
                <th>ID</th>
                <th>Agent</th>
                <th>Channel</th>
                <th>Peer</th>
                <th>Origin</th>
                <th>Status</th>
                <th>Created</th>
                <th>Updated</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody id="sessionsBody"></tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- ======== CRON JOBS PAGE ======== -->
    <div id="page-cron" class="page">
      <div class="card" id="cronStatusCard">
        <div class="card-header">
          <span class="card-title">Scheduler Status</span>
          <button class="btn btn-sm" onclick="loadCron()">Refresh</button>
        </div>
        <div id="cronStatusInfo" class="text-sm text-dim">Loading...</div>
      </div>
      <div class="card">
        <div class="card-header">
          <span class="card-title">Add Cron Job</span>
        </div>
        <div class="form-grid">
          <div class="form-row">
            <label>Name</label>
            <input type="text" id="cronName" placeholder="Daily report">
          </div>
          <div class="form-row">
            <label>Description</label>
            <input type="text" id="cronDesc" placeholder="Optional description">
          </div>
          <div class="form-row">
            <label>Agent ID</label>
            <input type="text" id="cronAgentId" placeholder="agent-id">
          </div>
          <div class="form-row">
            <label>Schedule (JSON)</label>
            <input type="text" id="cronSchedule" placeholder='{"kind":"interval","interval":60,"unit":"minutes"}'>
          </div>
        </div>
        <div class="form-row">
          <label>Payload (JSON)</label>
          <textarea id="cronPayload" rows="3" placeholder='{"kind":"text","message":"Hello"}'></textarea>
        </div>
        <button class="btn btn-primary mt-8" onclick="addCronJob()">Add Job</button>
      </div>
      <div class="card">
        <div class="card-header">
          <span class="card-title">Jobs</span>
        </div>
        <div class="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Name</th>
                <th>Schedule</th>
                <th>Status</th>
                <th>Last Run</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody id="cronJobsBody"></tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- ======== SKILLS PAGE ======== -->
    <div id="page-skills" class="page">
      <div class="filter-row">
        <input type="text" id="skillFilter" placeholder="Search skills..." oninput="filterSkills()">
        <button class="btn btn-sm" onclick="loadSkills()">Refresh</button>
      </div>
      <div class="card">
        <div class="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Name</th>
                <th>Description</th>
                <th>Source</th>
                <th>Eligible</th>
                <th>Enabled</th>
              </tr>
            </thead>
            <tbody id="skillsBody"></tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- ======== NODES PAGE ======== -->
    <div id="page-nodes" class="page">
      <div class="placeholder">
        <div class="ph-icon">&#9671;</div>
        <div class="ph-text">Node management coming soon</div>
        <div class="ph-sub">Current binding routes from the REST API:</div>
      </div>
      <div class="card mt-12">
        <div class="card-header">
          <span class="card-title">Active Bindings</span>
          <button class="btn btn-sm" onclick="loadNodes()">Refresh</button>
        </div>
        <div id="nodesBindings" class="text-sm mono">Loading...</div>
      </div>
    </div>

    <!-- ======== CONFIG PAGE ======== -->
    <div id="page-config" class="page">
      <div class="card">
        <div class="card-header">
          <span class="card-title">Configuration (JSON)</span>
          <div class="inline-flex">
            <span id="configStatus" class="text-xs text-dim"></span>
            <button class="btn btn-primary btn-sm" onclick="saveConfig()">Save</button>
          </div>
        </div>
        <textarea id="configEditor" rows="24" spellcheck="false"></textarea>
      </div>
    </div>

    <!-- ======== DEBUG PAGE ======== -->
    <div id="page-debug" class="page">
      <div class="stat-grid mb-8">
        <div class="card" style="grid-column: span 2">
          <div class="card-header">
            <span class="card-title">System Status</span>
            <button class="btn btn-sm" onclick="loadDebug()">Refresh</button>
          </div>
          <div id="debugStatus" class="text-sm mono">Loading...</div>
        </div>
      </div>
      <div class="card mb-8">
        <div class="card-header">
          <span class="card-title">Health Check</span>
          <button class="btn btn-sm" onclick="runHealthCheck()">Run</button>
        </div>
        <div id="debugHealth" class="rpc-result" style="min-height:60px">Click "Run" to check health.</div>
      </div>
      <div class="card">
        <div class="card-header">
          <span class="card-title">RPC Console</span>
        </div>
        <div class="form-grid" style="max-width:800px">
          <div class="form-row">
            <label>Method</label>
            <input type="text" id="rpcMethod" placeholder="system.status">
          </div>
          <div class="form-row" style="display:flex;align-items:flex-end;gap:8px">
            <div style="flex:1">
              <label>&nbsp;</label>
              <button class="btn btn-primary" onclick="callRpc()">Call</button>
            </div>
          </div>
        </div>
        <div class="form-row">
          <label>Params (JSON, optional)</label>
          <textarea id="rpcParams" rows="3" placeholder="{}"></textarea>
        </div>
        <div class="form-row mt-8">
          <label>Result</label>
          <div class="rpc-result" id="rpcResult">No result yet.</div>
        </div>
      </div>
    </div>

    <!-- ======== LOGS PAGE ======== -->
    <div id="page-logs" class="page">
      <div class="filter-row">
        <select id="logLevel" onchange="loadLogs()" style="width:120px">
          <option value="">All Levels</option>
          <option value="debug">Debug</option>
          <option value="info">Info</option>
          <option value="warn">Warn</option>
          <option value="error">Error</option>
        </select>
        <input type="text" id="logSearch" placeholder="Search logs..." style="width:220px">
        <button class="btn btn-sm" onclick="loadLogs()">Refresh</button>
        <label class="toggle" title="Auto-scroll">
          <input type="checkbox" id="logAutoScroll" checked>
          <span class="slider"></span>
        </label>
        <span class="text-xs text-dim">Auto-scroll</span>
        <label class="toggle" title="Auto-poll">
          <input type="checkbox" id="logAutoPoll" onchange="toggleLogPoll()">
          <span class="slider"></span>
        </label>
        <span class="text-xs text-dim">Auto-poll</span>
      </div>
      <div class="log-viewer" id="logViewer"></div>
    </div>

  </div><!-- /content -->
</div><!-- /main -->

<script>
(function() {
  'use strict';

  // ============================================================
  // WebSocket client
  // ============================================================
  var ws = null;
  var wsConnected = false;
  var rpcId = 0;
  var rpcCallbacks = {};
  var reconnectTimer = null;
  var reconnectDelay = 800;
  var reconnectMultiplier = 1.7;
  var reconnectMax = 15000;
  var currentDelay = 800;

  // Pages data cache
  var sessionData = [];
  var skillData = [];
  var logPollTimer = null;

  function getWsUrl() {
    var proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
    return proto + '//' + location.host + '/ws';
  }

  function setWsStatus(connected) {
    wsConnected = connected;
    var dot = document.getElementById('wsDot');
    var label = document.getElementById('wsLabel');
    if (connected) {
      dot.className = 'ws-dot connected';
      label.textContent = 'Connected';
    } else {
      dot.className = 'ws-dot';
      label.textContent = 'Disconnected';
    }
    var topSt = document.getElementById('topbarStatus');
    topSt.textContent = connected ? 'WS Connected' : 'WS Disconnected';
    var connBtn = document.getElementById('overviewConnBtn');
    connBtn.textContent = connected ? 'Reconnect' : 'Connect';
  }

  function connect() {
    if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) {
      ws.close();
    }
    clearTimeout(reconnectTimer);
    currentDelay = reconnectDelay;

    var url = getWsUrl();
    document.getElementById('gwUrl').value = url;
    ws = new WebSocket(url);

    ws.onopen = function() {
      setWsStatus(true);
      currentDelay = reconnectDelay;
      // Send hello handshake
      var token = document.getElementById('gwToken').value || '';
      var hello = { type: 'hello' };
      if (token) hello.token = token;
      ws.send(JSON.stringify(hello));
    };

    ws.onclose = function() {
      setWsStatus(false);
      scheduleReconnect();
    };

    ws.onerror = function() {
      setWsStatus(false);
    };

    ws.onmessage = function(evt) {
      var data;
      try { data = JSON.parse(evt.data); } catch(e) { return; }

      // Hello response
      if (data.type === 'hello-ok') {
        setWsStatus(true);
        refreshCurrentPage();
        return;
      }

      if (data.type === 'hello-error') {
        setWsStatus(false);
        return;
      }

      // RPC response
      if (data.id && rpcCallbacks[data.id]) {
        var cb = rpcCallbacks[data.id];
        delete rpcCallbacks[data.id];
        if (data.error) {
          cb.reject(new Error(data.error));
        } else {
          cb.resolve(data.result);
        }
        return;
      }

      // Broadcast events
      if (data.type === 'event') {
        handleEvent(data.event, data.payload);
      }
    };
  }

  function scheduleReconnect() {
    clearTimeout(reconnectTimer);
    reconnectTimer = setTimeout(function() {
      currentDelay = Math.min(currentDelay * reconnectMultiplier, reconnectMax);
      connect();
    }, currentDelay);
  }

  // Alias for the connect button
  window.connectWs = connect;

  function rpc(method, params) {
    return new Promise(function(resolve, reject) {
      if (!ws || ws.readyState !== WebSocket.OPEN) {
        reject(new Error('WebSocket not connected'));
        return;
      }
      rpcId++;
      var id = 'rpc-' + rpcId;
      rpcCallbacks[id] = { resolve: resolve, reject: reject };
      var msg = { id: id, method: method };
      if (params !== undefined && params !== null) {
        msg.params = params;
      }
      ws.send(JSON.stringify(msg));
      // Timeout after 30 seconds
      setTimeout(function() {
        if (rpcCallbacks[id]) {
          delete rpcCallbacks[id];
          reject(new Error('RPC timeout: ' + method));
        }
      }, 30000);
    });
  }

  // ============================================================
  // Event handling
  // ============================================================
  function handleEvent(event, payload) {
    switch (event) {
      case 'session.updated':
        if (currentPage === 'sessions') loadSessions();
        break;
      case 'cron.updated':
        if (currentPage === 'cron') loadCron();
        break;
      case 'channel.updated':
        if (currentPage === 'channels') loadChannels();
        break;
      case 'log.entry':
        if (currentPage === 'logs') appendLogEntry(payload);
        break;
    }
  }

  // ============================================================
  // Navigation
  // ============================================================
  var currentPage = 'chat';
  var pageTitles = {
    chat: 'Chat',
    overview: 'Overview',
    channels: 'Channels',
    instances: 'Instances',
    sessions: 'Sessions',
    cron: 'Cron Jobs',
    skills: 'Skills',
    nodes: 'Nodes',
    config: 'Config',
    debug: 'Debug',
    logs: 'Logs'
  };

  function showPage(page) {
    currentPage = page;
    // Update nav
    var items = document.querySelectorAll('.nav-item');
    for (var i = 0; i < items.length; i++) {
      items[i].classList.toggle('active', items[i].getAttribute('data-page') === page);
    }
    // Update pages
    var pages = document.querySelectorAll('.page');
    for (var i = 0; i < pages.length; i++) {
      pages[i].classList.remove('active');
    }
    var el = document.getElementById('page-' + page);
    if (el) el.classList.add('active');
    // Update topbar
    document.getElementById('topbarTitle').textContent = pageTitles[page] || page;
    // Load page data
    refreshCurrentPage();
  }

  function refreshCurrentPage() {
    if (!wsConnected) return;
    switch (currentPage) {
      case 'chat': loadChatSessions(); break;
      case 'overview': loadOverview(); break;
      case 'channels': loadChannels(); break;
      case 'instances': loadInstances(); break;
      case 'sessions': loadSessions(); break;
      case 'cron': loadCron(); break;
      case 'skills': loadSkills(); break;
      case 'nodes': loadNodes(); break;
      case 'config': loadConfig(); break;
      case 'debug': loadDebug(); break;
      case 'logs': loadLogs(); break;
    }
  }

  // ============================================================
  // Chat page
  // ============================================================
  function loadChatSessions() {
    rpc('sessions.list', { limit: 50 }).then(function(r) {
      var sel = document.getElementById('chatSessionSelect');
      var cur = sel.value;
      sel.innerHTML = '<option value="">New session</option>';
      if (r && r.sessions) {
        r.sessions.forEach(function(s) {
          var opt = document.createElement('option');
          opt.value = s.id;
          opt.textContent = (s.displayName || s.id) + ' (' + s.origin + ')';
          sel.appendChild(opt);
        });
      }
      if (cur) sel.value = cur;
    }).catch(function() {});
  }

  function loadChatHistory() {
    var sessId = document.getElementById('chatSessionSelect').value;
    var container = document.getElementById('chatMessages');
    if (!sessId) {
      container.innerHTML = '';
      return;
    }
    rpc('sessions.preview', { key: sessId, limit: 50 }).then(function(r) {
      container.innerHTML = '';
      if (r && r.messages) {
        r.messages.forEach(function(m) {
          appendChatMessage(m.role === 'user' ? 'user' : 'assistant', m.content, m.createdAt);
        });
      }
      scrollChatBottom();
    }).catch(function() {});
  }

  function appendChatMessage(role, text, ts) {
    var container = document.getElementById('chatMessages');
    var div = document.createElement('div');
    div.className = 'chat-msg ' + role;
    div.textContent = text;
    if (ts) {
      var meta = document.createElement('div');
      meta.className = 'msg-meta';
      meta.textContent = formatTime(ts);
      div.appendChild(meta);
    }
    container.appendChild(div);
  }

  function scrollChatBottom() {
    var c = document.getElementById('chatMessages');
    c.scrollTop = c.scrollHeight;
  }

  function sendChat() {
    var input = document.getElementById('chatInput');
    var msg = input.value.trim();
    if (!msg) return;
    var sessId = document.getElementById('chatSessionSelect').value || '';
    input.value = '';

    appendChatMessage('user', msg);
    scrollChatBottom();

    var btn = document.getElementById('chatSendBtn');
    btn.disabled = true;
    btn.textContent = '...';

    rpc('chat.send', { sessionId: sessId, message: msg }).then(function(r) {
      if (r && r.response) {
        appendChatMessage('assistant', r.response);
        scrollChatBottom();
      }
    }).catch(function(err) {
      appendChatMessage('assistant', 'Error: ' + err.message);
      scrollChatBottom();
    }).finally(function() {
      btn.disabled = false;
      btn.textContent = 'Send';
    });
  }

  function chatKeydown(e) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      sendChat();
    }
  }

  // ============================================================
  // Overview page
  // ============================================================
  function loadOverview() {
    rpc('system.status', {}).then(function(r) {
      if (!r) return;
      var st = r.status || {};
      document.getElementById('ovStatus').textContent = st.status || 'unknown';
      document.getElementById('ovUptime').textContent = r.uptime || st.uptime || '--';
      document.getElementById('ovAgents').textContent = st.activeAgents != null ? st.activeAgents : '--';
      document.getElementById('ovChannels').textContent = st.channels ? st.channels.length : '--';
      document.getElementById('ovSessions').textContent = st.sessions != null ? st.sessions : '--';
      document.getElementById('ovMessages').textContent = st.messages != null ? st.messages : '--';
    }).catch(function() {});

    rpc('cron.status', {}).then(function(r) {
      var el = document.getElementById('ovCronStatus');
      if (!r) { el.textContent = 'Unavailable'; return; }
      el.innerHTML = 'Enabled: ' + badge(r.enabled ? 'Yes' : 'No', r.enabled ? 'green' : 'dim') +
        ' &nbsp; Jobs: <strong>' + r.jobs + '</strong>';
      if (r.nextWakeAtMs) {
        el.innerHTML += ' &nbsp; Next wake: ' + formatTime(r.nextWakeAtMs);
      }
    }).catch(function() {});
  }

  // ============================================================
  // Channels page
  // ============================================================
  function loadChannels() {
    rpc('channels.status', {}).then(function(r) {
      var el = document.getElementById('channelsList');
      if (!r || !r.channels || r.channels.length === 0) {
        el.innerHTML = '<div class="text-dim text-sm">No channels configured.</div>';
        return;
      }
      var html = '';
      r.channels.forEach(function(ch) {
        var statusBadge = channelStatusBadge(ch.status);
        html += '<div class="card">' +
          '<div class="card-header">' +
          '<span class="card-title inline-flex">' + channelIcon(ch.type) + ' ' + esc(ch.name || ch.id) + '</span>' +
          statusBadge +
          '</div>' +
          '<div class="text-xs text-dim mb-8">Type: ' + esc(ch.type) + ' &nbsp; ID: <span class="mono">' + esc(ch.id) + '</span></div>' +
          '<div class="text-xs text-dim">Config:</div>' +
          '<pre class="mono text-xs" style="margin-top:4px;color:var(--text);white-space:pre-wrap;word-break:break-all">' + esc(ch.config || '{}') + '</pre>' +
          '</div>';
      });
      el.innerHTML = html;
    }).catch(function(err) {
      document.getElementById('channelsList').innerHTML = '<div class="text-dim text-sm">Error: ' + esc(err.message) + '</div>';
    });
  }

  function channelIcon(type) {
    switch(type) {
      case 'discord': return '&#9830;';
      case 'telegram': return '&#9993;';
      case 'webhook': return '&#8631;';
      case 'slack': return '#';
      default: return '&#9878;';
    }
  }

  function channelStatusBadge(status) {
    switch(status) {
      case 'connected': return badge('Connected', 'green');
      case 'error': return badge('Error', 'red');
      case 'disconnected': return badge('Disconnected', 'dim');
      default: return badge(status || 'unknown', 'yellow');
    }
  }

  // ============================================================
  // Instances page
  // ============================================================
  function loadInstances() {
    rpc('system.presence', {}).then(function(r) {
      var tbody = document.getElementById('instancesBody');
      var countEl = document.getElementById('instanceCount');
      var instances = (r && r.instances) || [];
      countEl.textContent = instances.length + ' connected client' + (instances.length !== 1 ? 's' : '');
      if (instances.length === 0) {
        tbody.innerHTML = '<tr><td colspan="4" class="text-dim">No connected instances.</td></tr>';
        return;
      }
      var html = '';
      instances.forEach(function(inst) {
        html += '<tr>' +
          '<td class="mono">' + esc(inst.id) + '</td>' +
          '<td class="mono">' + esc(inst.remoteAddr) + '</td>' +
          '<td>' + formatTime(inst.connectedAt) + '</td>' +
          '<td class="truncate" style="max-width:300px">' + esc(inst.userAgent) + '</td>' +
          '</tr>';
      });
      tbody.innerHTML = html;
    }).catch(function() {});
  }

  // ============================================================
  // Sessions page
  // ============================================================
  function loadSessions() {
    rpc('sessions.list', { limit: 200 }).then(function(r) {
      sessionData = (r && r.sessions) || [];
      renderSessions(sessionData);
    }).catch(function() {});
  }

  function renderSessions(sessions) {
    var tbody = document.getElementById('sessionsBody');
    if (sessions.length === 0) {
      tbody.innerHTML = '<tr><td colspan="9" class="text-dim">No sessions.</td></tr>';
      return;
    }
    var html = '';
    sessions.forEach(function(s) {
      var statusBadge = sessionStatusBadge(s.status);
      html += '<tr>' +
        '<td class="mono" style="max-width:120px">' + esc(shortId(s.id)) + '</td>' +
        '<td>' + esc(s.agentId || '--') + '</td>' +
        '<td>' + esc(s.channelId || '--') + '</td>' +
        '<td>' + esc(s.peerId || '--') + '</td>' +
        '<td>' + esc(s.origin || '--') + '</td>' +
        '<td>' + statusBadge + '</td>' +
        '<td class="text-xs">' + formatTime(s.createdAt) + '</td>' +
        '<td class="text-xs">' + formatTime(s.updatedAt) + '</td>' +
        '<td><button class="btn btn-danger btn-sm" onclick="deleteSession(\'' + esc(s.id) + '\')">Delete</button></td>' +
        '</tr>';
    });
    tbody.innerHTML = html;
  }

  function filterSessions() {
    var q = document.getElementById('sessionFilter').value.toLowerCase();
    if (!q) { renderSessions(sessionData); return; }
    var filtered = sessionData.filter(function(s) {
      return (s.id + ' ' + (s.agentId||'') + ' ' + (s.channelId||'') + ' ' + (s.peerId||'') + ' ' + (s.origin||'')).toLowerCase().indexOf(q) !== -1;
    });
    renderSessions(filtered);
  }

  function deleteSession(key) {
    rpc('sessions.delete', { key: key }).then(function() {
      loadSessions();
    }).catch(function(err) {
      alert('Delete failed: ' + err.message);
    });
  }

  function sessionStatusBadge(status) {
    switch(status) {
      case 'active': return badge('Active', 'green');
      case 'expired': return badge('Expired', 'yellow');
      case 'closed': return badge('Closed', 'dim');
      default: return badge(status || 'unknown', 'blue');
    }
  }

  // ============================================================
  // Cron Jobs page
  // ============================================================
  function loadCron() {
    rpc('cron.status', {}).then(function(r) {
      var el = document.getElementById('cronStatusInfo');
      if (!r) { el.textContent = 'Unavailable'; return; }
      el.innerHTML = 'Enabled: ' + badge(r.enabled ? 'Yes' : 'No', r.enabled ? 'green' : 'dim') +
        ' &nbsp; Total Jobs: <strong>' + r.jobs + '</strong>';
      if (r.nextWakeAtMs) {
        el.innerHTML += ' &nbsp; Next Wake: ' + formatTime(r.nextWakeAtMs);
      }
    }).catch(function() {});

    rpc('cron.list', {}).then(function(r) {
      var tbody = document.getElementById('cronJobsBody');
      var jobs = (r && r.jobs) || [];
      if (jobs.length === 0) {
        tbody.innerHTML = '<tr><td colspan="5" class="text-dim">No cron jobs.</td></tr>';
        return;
      }
      var html = '';
      jobs.forEach(function(j) {
        var enabledBadge = j.enabled ? badge('Enabled', 'green') : badge('Disabled', 'dim');
        html += '<tr>' +
          '<td><strong>' + esc(j.name) + '</strong><div class="text-xs text-dim">' + esc(j.description || '') + '</div></td>' +
          '<td class="mono text-xs" style="max-width:200px">' + esc(j.schedule || '--') + '</td>' +
          '<td>' + enabledBadge + '</td>' +
          '<td class="text-xs">' + (j.lastRunAt ? formatTime(j.lastRunAt) + ' ' + badge(j.lastStatus || '', j.lastStatus === 'success' ? 'green' : 'red') : '--') + '</td>' +
          '<td class="inline-flex">' +
          '<button class="btn btn-sm" onclick="toggleCronJob(\'' + esc(j.id) + '\',' + !j.enabled + ')">' + (j.enabled ? 'Disable' : 'Enable') + '</button>' +
          '<button class="btn btn-sm" onclick="runCronJob(\'' + esc(j.id) + '\')">Run</button>' +
          '<button class="btn btn-sm btn-danger" onclick="deleteCronJob(\'' + esc(j.id) + '\')">Delete</button>' +
          '</td>' +
          '</tr>';
      });
      tbody.innerHTML = html;
    }).catch(function() {});
  }

  function addCronJob() {
    var name = document.getElementById('cronName').value.trim();
    var desc = document.getElementById('cronDesc').value.trim();
    var agentId = document.getElementById('cronAgentId').value.trim();
    var schedule = document.getElementById('cronSchedule').value.trim();
    var payload = document.getElementById('cronPayload').value.trim();

    if (!name) { alert('Name is required'); return; }

    rpc('cron.add', {
      name: name,
      description: desc,
      agentId: agentId,
      schedule: schedule,
      payload: payload
    }).then(function() {
      document.getElementById('cronName').value = '';
      document.getElementById('cronDesc').value = '';
      document.getElementById('cronAgentId').value = '';
      document.getElementById('cronSchedule').value = '';
      document.getElementById('cronPayload').value = '';
      loadCron();
    }).catch(function(err) {
      alert('Add failed: ' + err.message);
    });
  }

  function toggleCronJob(id, enabled) {
    rpc('cron.update', { id: id, enabled: enabled }).then(function() {
      loadCron();
    }).catch(function(err) {
      alert('Toggle failed: ' + err.message);
    });
  }

  function runCronJob(id) {
    rpc('cron.run', { id: id }).then(function() {
      loadCron();
    }).catch(function(err) {
      alert('Run failed: ' + err.message);
    });
  }

  function deleteCronJob(id) {
    if (!confirm('Delete this cron job?')) return;
    rpc('cron.remove', { id: id }).then(function() {
      loadCron();
    }).catch(function(err) {
      alert('Delete failed: ' + err.message);
    });
  }

  // ============================================================
  // Skills page
  // ============================================================
  function loadSkills() {
    rpc('skills.status', {}).then(function(r) {
      skillData = (r && r.skills) || [];
      renderSkills(skillData);
    }).catch(function() {});
  }

  function renderSkills(skills) {
    var tbody = document.getElementById('skillsBody');
    if (skills.length === 0) {
      tbody.innerHTML = '<tr><td colspan="5" class="text-dim">No skills found.</td></tr>';
      return;
    }
    var html = '';
    skills.forEach(function(s) {
      var eligibleBadge = s.eligible ? badge('Eligible', 'green') : badge('Ineligible', 'yellow');
      html += '<tr>' +
        '<td><strong>' + esc(s.emoji ? s.emoji + ' ' : '') + esc(s.name) + '</strong></td>' +
        '<td class="text-sm">' + esc(s.description || '--') + '</td>' +
        '<td>' + badge(s.source || 'unknown', s.source === 'bundled' ? 'blue' : s.source === 'workspace' ? 'purple' : 'dim') + '</td>' +
        '<td>' + eligibleBadge + '</td>' +
        '<td><label class="toggle"><input type="checkbox" ' + (s.enabled ? 'checked' : '') +
        ' onchange="toggleSkill(\'' + esc(s.key) + '\', this.checked)"><span class="slider"></span></label></td>' +
        '</tr>';
    });
    tbody.innerHTML = html;
  }

  function filterSkills() {
    var q = document.getElementById('skillFilter').value.toLowerCase();
    if (!q) { renderSkills(skillData); return; }
    var filtered = skillData.filter(function(s) {
      return (s.name + ' ' + (s.description||'') + ' ' + (s.source||'')).toLowerCase().indexOf(q) !== -1;
    });
    renderSkills(filtered);
  }

  function toggleSkill(key, enabled) {
    rpc('skills.toggle', { key: key, enabled: enabled }).catch(function(err) {
      alert('Toggle failed: ' + err.message);
    });
  }

  // ============================================================
  // Nodes page
  // ============================================================
  function loadNodes() {
    var el = document.getElementById('nodesBindings');
    // Fetch bindings via REST API
    fetch('/api/bindings').then(function(resp) {
      return resp.json();
    }).then(function(data) {
      if (!data || (Array.isArray(data) && data.length === 0)) {
        el.textContent = 'No bindings configured.';
        return;
      }
      var bindings = Array.isArray(data) ? data : (data.bindings || []);
      if (bindings.length === 0) {
        el.textContent = 'No bindings configured.';
        return;
      }
      var html = '';
      bindings.forEach(function(b) {
        html += '<div style="padding:4px 0;border-bottom:1px solid var(--border)">' +
          '<span class="badge badge-blue">' + esc(b.agentId || '--') + '</span> ' +
          esc(b.channelType || '') + '/' + esc(b.channelId || '') +
          ' &rarr; ' + esc(b.peerFilter || '*') +
          ' <span class="text-dim text-xs">(' + esc(b.id || '') + ')</span></div>';
      });
      el.innerHTML = html;
    }).catch(function(err) {
      el.textContent = 'Error loading bindings: ' + err.message;
    });
  }

  // ============================================================
  // Config page
  // ============================================================
  function loadConfig() {
    rpc('config.read', {}).then(function(r) {
      if (!r) return;
      var editor = document.getElementById('configEditor');
      var st = document.getElementById('configStatus');
      try {
        var obj = JSON.parse(r.raw);
        editor.value = JSON.stringify(obj, null, 2);
      } catch(e) {
        editor.value = r.raw || '{}';
      }
      st.textContent = r.valid ? 'Valid JSON' : 'Invalid JSON';
      st.style.color = r.valid ? 'var(--green)' : 'var(--red)';
    }).catch(function(err) {
      document.getElementById('configStatus').textContent = 'Error: ' + err.message;
    });
  }

  function saveConfig() {
    var raw = document.getElementById('configEditor').value;
    var st = document.getElementById('configStatus');
    try {
      JSON.parse(raw);
    } catch(e) {
      st.textContent = 'Invalid JSON - fix before saving';
      st.style.color = 'var(--red)';
      return;
    }
    rpc('config.write', { raw: raw }).then(function() {
      st.textContent = 'Saved';
      st.style.color = 'var(--green)';
    }).catch(function(err) {
      st.textContent = 'Save failed: ' + err.message;
      st.style.color = 'var(--red)';
    });
  }

  // ============================================================
  // Debug page
  // ============================================================
  function loadDebug() {
    rpc('system.status', {}).then(function(r) {
      var el = document.getElementById('debugStatus');
      if (!r) { el.textContent = 'Unavailable'; return; }
      el.textContent = JSON.stringify(r, null, 2);
    }).catch(function(err) {
      document.getElementById('debugStatus').textContent = 'Error: ' + err.message;
    });
  }

  function runHealthCheck() {
    var el = document.getElementById('debugHealth');
    el.textContent = 'Running health check...';
    rpc('health.check', {}).then(function(r) {
      el.textContent = JSON.stringify(r, null, 2);
    }).catch(function(err) {
      el.textContent = 'Error: ' + err.message;
    });
  }

  function callRpc() {
    var method = document.getElementById('rpcMethod').value.trim();
    var paramsStr = document.getElementById('rpcParams').value.trim();
    var el = document.getElementById('rpcResult');
    if (!method) { el.textContent = 'Enter a method name.'; return; }

    var params = null;
    if (paramsStr) {
      try {
        params = JSON.parse(paramsStr);
      } catch(e) {
        el.textContent = 'Invalid JSON params: ' + e.message;
        return;
      }
    }

    el.textContent = 'Calling ' + method + '...';
    rpc(method, params).then(function(r) {
      el.textContent = JSON.stringify(r, null, 2);
    }).catch(function(err) {
      el.textContent = 'Error: ' + err.message;
    });
  }

  // ============================================================
  // Logs page
  // ============================================================
  function loadLogs() {
    var level = document.getElementById('logLevel').value;
    var search = document.getElementById('logSearch').value.trim();

    var method = search ? 'logs.search' : 'logs.tail';
    var params = search ? { query: search, level: level } : { limit: 500, level: level };

    rpc(method, params).then(function(r) {
      var entries = (r && r.entries) || [];
      renderLogs(entries);
    }).catch(function(err) {
      document.getElementById('logViewer').innerHTML = '<div class="text-dim">Error: ' + esc(err.message) + '</div>';
    });
  }

  function renderLogs(entries) {
    var viewer = document.getElementById('logViewer');
    if (entries.length === 0) {
      viewer.innerHTML = '<div class="text-dim" style="padding:12px">No log entries.</div>';
      return;
    }
    var html = '';
    entries.forEach(function(e) {
      html += '<div class="log-entry">' +
        '<span class="log-ts">' + esc(formatLogTime(e.timestamp)) + '</span>' +
        '<span class="log-level ' + esc(e.level || 'info') + '">' + esc(e.level || 'info') + '</span>' +
        '<span class="log-sub">' + esc(e.subsystem || '') + '</span>' +
        '<span class="log-msg">' + esc(e.message || e.raw || '') + '</span>' +
        '</div>';
    });
    viewer.innerHTML = html;
    if (document.getElementById('logAutoScroll').checked) {
      viewer.scrollTop = viewer.scrollHeight;
    }
  }

  function appendLogEntry(entry) {
    if (!entry) return;
    var viewer = document.getElementById('logViewer');
    var div = document.createElement('div');
    div.className = 'log-entry';
    div.innerHTML =
      '<span class="log-ts">' + esc(formatLogTime(entry.timestamp)) + '</span>' +
      '<span class="log-level ' + esc(entry.level || 'info') + '">' + esc(entry.level || 'info') + '</span>' +
      '<span class="log-sub">' + esc(entry.subsystem || '') + '</span>' +
      '<span class="log-msg">' + esc(entry.message || entry.raw || '') + '</span>';
    viewer.appendChild(div);
    if (document.getElementById('logAutoScroll').checked) {
      viewer.scrollTop = viewer.scrollHeight;
    }
  }

  function toggleLogPoll() {
    if (document.getElementById('logAutoPoll').checked) {
      logPollTimer = setInterval(function() {
        if (currentPage === 'logs') loadLogs();
      }, 5000);
    } else {
      clearInterval(logPollTimer);
      logPollTimer = null;
    }
  }

  // ============================================================
  // Utilities
  // ============================================================
  function esc(s) {
    if (s == null) return '';
    var d = document.createElement('div');
    d.appendChild(document.createTextNode(String(s)));
    return d.innerHTML;
  }

  function badge(text, color) {
    return '<span class="badge badge-' + color + '">' + esc(text) + '</span>';
  }

  function shortId(id) {
    if (!id) return '--';
    if (id.length <= 12) return id;
    return id.substring(0, 8) + '...' + id.substring(id.length - 4);
  }

  function formatTime(val) {
    if (!val) return '--';
    var d;
    if (typeof val === 'number') {
      d = new Date(val > 1e12 ? val : val * 1000);
    } else if (typeof val === 'string') {
      d = new Date(val);
    } else {
      return '--';
    }
    if (isNaN(d.getTime())) return '--';
    return d.toLocaleString();
  }

  function formatLogTime(val) {
    if (!val) return '';
    var d = new Date(val);
    if (isNaN(d.getTime())) return val;
    return pad(d.getHours()) + ':' + pad(d.getMinutes()) + ':' + pad(d.getSeconds());
  }

  function pad(n) {
    return n < 10 ? '0' + n : '' + n;
  }

  // ============================================================
  // Expose functions to global scope for inline handlers
  // ============================================================
  window.showPage = showPage;
  window.sendChat = sendChat;
  window.chatKeydown = chatKeydown;
  window.loadChatSessions = loadChatSessions;
  window.loadChatHistory = loadChatHistory;
  window.loadOverview = loadOverview;
  window.loadChannels = loadChannels;
  window.loadInstances = loadInstances;
  window.loadSessions = loadSessions;
  window.filterSessions = filterSessions;
  window.deleteSession = deleteSession;
  window.loadCron = loadCron;
  window.addCronJob = addCronJob;
  window.toggleCronJob = toggleCronJob;
  window.runCronJob = runCronJob;
  window.deleteCronJob = deleteCronJob;
  window.loadSkills = loadSkills;
  window.filterSkills = filterSkills;
  window.toggleSkill = toggleSkill;
  window.loadNodes = loadNodes;
  window.loadConfig = loadConfig;
  window.saveConfig = saveConfig;
  window.loadDebug = loadDebug;
  window.runHealthCheck = runHealthCheck;
  window.callRpc = callRpc;
  window.loadLogs = loadLogs;
  window.toggleLogPoll = toggleLogPoll;

  // ============================================================
  // Initialize
  // ============================================================
  document.getElementById('gwUrl').value = getWsUrl();
  connect();

})();
</script>
</body>
</html>` + ""
