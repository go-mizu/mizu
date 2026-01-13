package assets

import (
	"embed"
	"io/fs"
)

//go:embed static views
var content embed.FS

// Static returns the static file system.
func Static() fs.FS {
	static, _ := fs.Sub(content, "static")
	return static
}

// IndexHTML returns the index.html content for the SPA.
func IndexHTML() []byte {
	// Try to read from static/dist/index.html first
	if data, err := content.ReadFile("static/dist/index.html"); err == nil {
		return data
	}

	// Fallback to embedded HTML if frontend not built
	return []byte(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Localflare Dashboard</title>
    <link rel="icon" type="image/svg+xml" href="data:image/svg+xml,<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 100 100'><text y='.9em' font-size='90'>🔥</text></svg>">
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #1a1a2e; color: #fff; min-height: 100vh; }
        .container { max-width: 1200px; margin: 0 auto; padding: 2rem; }
        .header { display: flex; align-items: center; gap: 1rem; margin-bottom: 2rem; padding-bottom: 1rem; border-bottom: 1px solid #333; }
        .logo { font-size: 2rem; font-weight: bold; color: #f6821f; }
        .nav { display: flex; gap: 1rem; margin-left: auto; }
        .nav a { color: #888; text-decoration: none; padding: 0.5rem 1rem; border-radius: 0.5rem; transition: all 0.2s; }
        .nav a:hover, .nav a.active { color: #fff; background: #333; }
        .hero { text-align: center; padding: 4rem 0; }
        .hero h1 { font-size: 3rem; margin-bottom: 1rem; background: linear-gradient(135deg, #f6821f, #ffcc00); -webkit-background-clip: text; -webkit-text-fill-color: transparent; }
        .hero p { color: #888; font-size: 1.2rem; max-width: 600px; margin: 0 auto 2rem; }
        .features { display: grid; grid-template-columns: repeat(auto-fit, minmax(280px, 1fr)); gap: 1.5rem; margin-top: 3rem; }
        .feature { background: #252540; padding: 1.5rem; border-radius: 1rem; border: 1px solid #333; }
        .feature h3 { color: #f6821f; margin-bottom: 0.5rem; display: flex; align-items: center; gap: 0.5rem; }
        .feature p { color: #888; font-size: 0.9rem; }
        .status { display: inline-flex; align-items: center; gap: 0.5rem; padding: 0.5rem 1rem; background: #1a3a1a; color: #4ade80; border-radius: 2rem; font-size: 0.9rem; }
        .status::before { content: ''; width: 8px; height: 8px; background: #4ade80; border-radius: 50%; }
        .cta { display: inline-block; margin-top: 1.5rem; padding: 0.75rem 2rem; background: #f6821f; color: #fff; text-decoration: none; border-radius: 0.5rem; font-weight: 500; transition: all 0.2s; }
        .cta:hover { background: #ff9933; transform: translateY(-2px); }
        .note { margin-top: 3rem; padding: 1rem; background: #2a2a4a; border-radius: 0.5rem; border-left: 4px solid #f6821f; }
        .note code { background: #1a1a2e; padding: 0.2rem 0.5rem; border-radius: 0.25rem; font-family: monospace; }
    </style>
</head>
<body>
    <div class="container">
        <header class="header">
            <div class="logo">🔥 Localflare</div>
            <nav class="nav">
                <a href="/" class="active">Dashboard</a>
                <a href="/zones">Zones</a>
                <a href="/workers">Workers</a>
                <a href="/r2">R2</a>
                <a href="/kv">KV</a>
                <a href="/d1">D1</a>
            </nav>
        </header>

        <section class="hero">
            <span class="status">All Systems Operational</span>
            <h1>Localflare Dashboard</h1>
            <p>A comprehensive, offline-first implementation of Cloudflare's core features. Run your own edge infrastructure locally.</p>
            <a href="/zones" class="cta">Manage Zones</a>
        </section>

        <div class="features">
            <div class="feature">
                <h3>🌐 DNS Management</h3>
                <p>Full DNS server with zone management, record types (A, AAAA, CNAME, MX, TXT), and proxy support.</p>
            </div>
            <div class="feature">
                <h3>🔒 SSL/TLS</h3>
                <p>Self-signed certificate generation, Origin CA certificates, and flexible SSL modes.</p>
            </div>
            <div class="feature">
                <h3>🛡️ Security</h3>
                <p>Web Application Firewall, IP access rules, rate limiting, and bot management.</p>
            </div>
            <div class="feature">
                <h3>⚡ Workers</h3>
                <p>JavaScript runtime for serverless functions with KV, R2, and D1 bindings.</p>
            </div>
            <div class="feature">
                <h3>💾 KV Storage</h3>
                <p>Key-value storage with namespaces, TTL support, and metadata.</p>
            </div>
            <div class="feature">
                <h3>📦 R2 Storage</h3>
                <p>S3-compatible object storage with bucket management and multipart uploads.</p>
            </div>
            <div class="feature">
                <h3>🗄️ D1 Database</h3>
                <p>SQLite-based databases with SQL query interface and prepared statements.</p>
            </div>
            <div class="feature">
                <h3>📊 Analytics</h3>
                <p>Traffic, security, and cache analytics with real-time metrics.</p>
            </div>
        </div>

        <div class="note">
            <strong>Note:</strong> This is the fallback UI. To get the full React dashboard, run:
            <br><br>
            <code>cd app/frontend && pnpm install && pnpm run build</code>
            <br><br>
            Then restart the server.
        </div>
    </div>

    <script>
        // Simple SPA routing for the fallback UI
        document.querySelectorAll('.nav a').forEach(link => {
            if (link.pathname === window.location.pathname) {
                link.classList.add('active');
            } else {
                link.classList.remove('active');
            }
        });
    </script>
</body>
</html>`)
}
