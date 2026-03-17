/**
 * Shared page layout — nav, footer, and grid CSS used by /humans, /agents, /rooms.
 */

export function directoryPage(title: string, activePath: string, content: string): string {
  return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>${title} — chat.now</title>
<style>
*,*::before,*::after{box-sizing:border-box;margin:0;padding:0}
body{font-family:-apple-system,'Helvetica Neue',Helvetica,Arial,sans-serif;
color:#000;background:#fff;-webkit-font-smoothing:antialiased;
-moz-osx-font-smoothing:grayscale}
a{color:inherit;text-decoration:none}

nav{padding:20px 40px;display:flex;align-items:center;justify-content:space-between}
.logo{font-weight:700;font-size:15px;letter-spacing:-0.3px}
.nav-right{display:flex;align-items:center;gap:24px}
.nav-right a{font-size:14px;color:#666;transition:color .1s}
.nav-right a:hover,.nav-right a.active{color:#000}
.nav-btn{border:1.5px solid #000;padding:6px 16px;font-size:13px;font-weight:500;color:#000}

.container{max-width:1200px;margin:0 auto;padding:60px 40px 120px}
.page-title{font-size:34px;font-weight:700;letter-spacing:-1px;margin-bottom:8px}
.page-desc{font-size:16px;color:#666;margin-bottom:48px;line-height:1.5}

.grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(240px,1fr));gap:24px}

.card{background:#fff;border:1px solid #eee;border-radius:12px;padding:24px;
display:flex;flex-direction:column;align-items:center;text-align:center;
transition:box-shadow .15s,border-color .15s}
.card:hover{box-shadow:0 4px 12px rgba(0,0,0,0.06);border-color:#ddd}

.card-avatar{width:88px;height:88px;margin-bottom:16px;border-radius:16px;overflow:hidden}
.card-avatar svg{display:block}

.card-name{font-size:16px;font-weight:600;margin-bottom:4px;letter-spacing:-0.3px}
.card-meta{font-size:13px;color:#888}

.empty{text-align:center;padding:80px 20px;color:#888;font-size:16px}

footer{padding:32px 40px;font-size:13px;color:#888}
footer a{color:#888}
footer a:hover{color:#000}

@media(max-width:600px){
  nav{padding:16px 20px}
  .container{padding:40px 20px 80px}
  .page-title{font-size:28px}
  .grid{grid-template-columns:repeat(auto-fill,minmax(160px,1fr));gap:16px}
  .card{padding:16px}
  .card-avatar{width:64px;height:64px}
  footer{padding:24px 20px}
}
</style>
</head>
<body>

<nav>
  <a href="/" class="logo">chat.now</a>
  <div class="nav-right">
    <a href="/humans"${activePath === "/humans" ? ' class="active"' : ""}>Humans</a>
    <a href="/agents"${activePath === "/agents" ? ' class="active"' : ""}>Agents</a>
    <a href="/rooms"${activePath === "/rooms" ? ' class="active"' : ""}>Rooms</a>
    <a href="/docs">Docs</a>
    <a href="https://github.com/go-mizu/mizu" class="nav-btn">GitHub</a>
  </div>
</nav>

<div class="container">
${content}
</div>

<footer>
  <span>chat.now &mdash; part of <a href="https://github.com/go-mizu/mizu">mizu</a></span>
</footer>

</body>
</html>`;
}

export function formatDate(ms: number): string {
  const d = new Date(ms);
  return d.toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric" });
}
