import { esc } from "./layout";

export function privacyPage(): string {
  return `<!DOCTYPE html>
<html lang="en" class="dark">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Privacy Policy — Storage</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800&family=JetBrains+Mono:wght@400;500;600;700&display=swap" rel="stylesheet">
<style>
*,*::before,*::after{box-sizing:border-box;margin:0;padding:0}
:root{--bg:#FAFAF9;--surface:#FFF;--text:#09090B;--text-2:#52525B;--text-3:#A1A1AA;--border:#E4E4E7;--ink:#09090B}
html.dark{--bg:#09090B;--surface:#111113;--text:#FAFAF9;--text-2:#A1A1AA;--text-3:#52525B;--border:#1E1E21;--ink:#FAFAF9}
html{scroll-behavior:smooth}
body{font-family:'Inter',system-ui,sans-serif;color:var(--text);background:var(--bg);-webkit-font-smoothing:antialiased;max-width:720px;margin:0 auto;padding:48px 24px 120px}
a{color:var(--text);text-decoration:underline;text-underline-offset:2px}
a:hover{color:var(--text-2)}
h1{font-size:32px;font-weight:800;letter-spacing:-1px;margin-bottom:8px}
h2{font-size:20px;font-weight:700;letter-spacing:-0.3px;margin:48px 0 16px;padding-top:24px;border-top:1px solid var(--border)}
h3{font-size:16px;font-weight:600;margin:24px 0 12px}
p,li{font-size:15px;color:var(--text-2);line-height:1.8;margin-bottom:12px}
ul{padding-left:24px;margin-bottom:16px}
li{margin-bottom:6px}
.meta{font-size:13px;color:var(--text-3);margin-bottom:48px}
code{font-family:'JetBrains Mono',monospace;font-size:0.85em;background:var(--surface);padding:2px 6px;border:1px solid var(--border)}
.back{display:inline-block;font-size:14px;color:var(--text-3);margin-bottom:32px;text-decoration:none}
.back:hover{color:var(--text)}
</style>
</head>
<body>

<a href="/" class="back">&larr; Storage</a>

<h1>Privacy Policy</h1>
<p class="meta">Last updated: March 20, 2026</p>

<p>This Privacy Policy explains how Storage ("we", "us", "our") collects, uses, and protects your personal information when you use our file storage service at <code>storage.liteio.dev</code>.</p>

<h2>1. Information We Collect</h2>

<h3>Account Information</h3>
<ul>
  <li><strong>Email address</strong> — used for authentication via magic sign-in links. We do not collect passwords.</li>
  <li><strong>Display name</strong> — derived from your email address for identification within the service.</li>
</ul>

<h3>Files and Content</h3>
<ul>
  <li><strong>Files you upload</strong> — stored in encrypted-at-rest object storage. We do not access, read, or analyze your file contents.</li>
  <li><strong>File metadata</strong> — file names, paths, sizes, MIME types, and modification timestamps for directory listing and search functionality.</li>
</ul>

<h3>Usage Metadata</h3>
<ul>
  <li><strong>API access logs</strong> — HTTP method, endpoint path, response status, and timestamp for operational monitoring. IP addresses are logged transiently for rate limiting and abuse prevention.</li>
  <li><strong>Authentication events</strong> — sign-in timestamps and session creation for security purposes.</li>
</ul>

<h3>Information We Do NOT Collect</h3>
<ul>
  <li>Payment card or financial information</li>
  <li>Government identifiers (social security numbers, etc.)</li>
  <li>Protected health information (PHI)</li>
  <li>Precise geolocation or GPS coordinates</li>
  <li>Chat histories or conversation logs from AI assistants</li>
  <li>Behavioral tracking or advertising profiles</li>
</ul>

<h2>2. How We Use Your Information</h2>

<ul>
  <li><strong>Provide the service</strong> — store, organize, retrieve, and share your files as you request.</li>
  <li><strong>Authentication</strong> — verify your identity via email magic links or API keys.</li>
  <li><strong>Service operation</strong> — monitor system health, enforce rate limits, prevent abuse.</li>
  <li><strong>Share links</strong> — when you create a share link, the linked file becomes accessible to anyone with the URL for the duration you specify.</li>
</ul>

<p>We do not sell your personal information. We do not use your data for advertising, profiling, or AI model training.</p>

<h2>3. Third-Party AI Assistants (MCP Integration)</h2>

<p>Storage integrates with AI assistants (Claude, ChatGPT) via the Model Context Protocol (MCP). When you connect Storage to an AI assistant:</p>

<ul>
  <li>The AI assistant can list, read, write, search, move, delete, and share files in your storage on your behalf.</li>
  <li>The AI assistant receives only the data necessary to fulfill your request — file names, paths, sizes, content types, and file contents for text files you ask to read.</li>
  <li>We do not send your files to any AI service proactively. The AI assistant initiates requests only in response to your instructions.</li>
  <li>Share links generated via AI are identical to those created through the web interface — time-limited and publicly accessible.</li>
</ul>

<h2>4. Data Sharing and Recipients</h2>

<ul>
  <li><strong>Share link recipients</strong> — anyone you share a link with can download the linked file. No authentication is required for share link access.</li>
  <li><strong>Infrastructure providers</strong> — we use Cloudflare (hosting, CDN, object storage) and Resend (email delivery). These providers process data as necessary to provide their services under their own privacy policies.</li>
  <li><strong>No other third parties</strong> — we do not sell, rent, or share your personal information with any other parties.</li>
</ul>

<h2>5. Data Retention</h2>

<ul>
  <li><strong>Files</strong> — retained until you delete them.</li>
  <li><strong>Account information</strong> — retained while your account is active.</li>
  <li><strong>Share links</strong> — automatically expire after the duration you set (1 hour to 7 days).</li>
  <li><strong>Authentication tokens</strong> — magic links expire in 15 minutes. Sessions expire in 2 hours. API keys expire in 90 days.</li>
  <li><strong>Access logs</strong> — retained for up to 30 days for operational purposes.</li>
</ul>

<h2>6. Your Rights and Controls</h2>

<ul>
  <li><strong>Access</strong> — view all your files and metadata through the web interface, CLI, or API.</li>
  <li><strong>Delete</strong> — delete any file or folder at any time through the web interface, CLI, API, or AI assistant.</li>
  <li><strong>Export</strong> — download all your files through the API or CLI.</li>
  <li><strong>Account deletion</strong> — contact us to permanently delete your account and all associated data.</li>
  <li><strong>Revoke AI access</strong> — disconnect Storage from your AI assistant at any time through the assistant's settings.</li>
</ul>

<h2>7. Security</h2>

<ul>
  <li>All data transmitted over HTTPS/TLS.</li>
  <li>Files stored in encrypted-at-rest object storage.</li>
  <li>Authentication via Ed25519 challenge-response (machine clients) or email magic links (human users) — no passwords stored.</li>
  <li>API keys are hashed before storage.</li>
  <li>SQL injection prevention via prepared statements.</li>
  <li>Path validation to prevent directory traversal.</li>
</ul>

<h2>8. Children's Privacy</h2>

<p>Storage is not directed at children under 13. We do not knowingly collect personal information from children under 13. If we learn that we have collected information from a child under 13, we will delete that information promptly.</p>

<h2>9. Changes to This Policy</h2>

<p>We may update this policy from time to time. Material changes will be communicated through the service. Continued use of Storage after changes constitutes acceptance of the updated policy.</p>

<h2>10. Contact</h2>

<p>For privacy questions, data requests, or account deletion, contact us at the email address listed on <a href="https://storage.liteio.dev">storage.liteio.dev</a>.</p>

<script>
(function(){
  const saved=localStorage.getItem('theme');
  if(saved==='light') document.documentElement.classList.remove('dark');
  else if(!saved&&!window.matchMedia('(prefers-color-scheme:dark)').matches) document.documentElement.classList.remove('dark');
})();
</script>
</body>
</html>`;
}
