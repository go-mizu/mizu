# 0779: Secure Redis + Redis Insight on server2

**Goal:** Install Redis 8 (2026 LTS) with password auth and TLS-optional config on server2
(6-core, 12 GB RAM, Ubuntu 24.04). Provide Redis Insight 2.x with secure auth.

---

## 1. Why Redis

The CC pipeline currently tracks state via:
- **stats.csv** — committed shard list (read/written by watcher + scheduler)
- **watcher_status.json** — latest HF commit info (written by watcher, read by scheduler)
- **file counting** — `os.ReadDir` to count `.parquet` and `.warc.gz` files
- **pgrep/pkill** — process detection for screen sessions

Problems:
1. **stats.csv** is read-modify-write with no locking — races between watcher and HF merge
2. **watcher_status.json** is polled every 45s by scheduler — delayed, one-way
3. File counting is O(n) disk I/O every round
4. No cross-process visibility — scheduler can't see pipeline progress until parquet appears
5. No real-time monitoring — must `tail` log files in screen sessions

Redis solves all of these: atomic counters, pub/sub for events, sorted sets for
rate windows, hashes for per-shard state, and Redis Insight for visual dashboards.

---

## 2. Installation

### 2.1 Redis Server (via apt)

```bash
# Add Redis official apt repo (Ubuntu 24.04)
curl -fsSL https://packages.redis.io/gpg | sudo gpg --dearmor -o /usr/share/keyrings/redis-archive-keyring.gpg
echo "deb [signed-by=/usr/share/keyrings/redis-archive-keyring.gpg] https://packages.redis.io/deb $(lsb_release -cs) main" | sudo tee /etc/apt/sources.list.d/redis.list
sudo apt update && sudo apt install -y redis-server
```

### 2.2 Secure Configuration

Edit `/etc/redis/redis.conf`:

```conf
# --- Network ---
bind 127.0.0.1 -::1          # localhost only (no external access)
port 6379
protected-mode yes

# --- Auth ---
requirepass <REDIS_PASSWORD>   # generate: openssl rand -base64 32

# --- Memory ---
maxmemory 512mb               # hard cap (server has 12 GB, pipeline needs ~10 GB)
maxmemory-policy allkeys-lru  # evict least-recently-used when full

# --- Persistence ---
# RDB snapshots: save state periodically (low overhead)
save 900 1                    # save if 1+ key changed in 15 min
save 300 10                   # save if 10+ keys changed in 5 min
save 60 10000                 # save if 10000+ keys changed in 1 min
dbfilename redis-cc.rdb
dir /var/lib/redis

# AOF: disable (we don't need durability — pipeline state is reconstructible from HF)
appendonly no

# --- Security hardening ---
rename-command FLUSHALL ""    # disable dangerous commands
rename-command FLUSHDB ""
rename-command DEBUG ""
rename-command CONFIG "REDIS_CONFIG_a8f3"  # rename CONFIG to obscure name

# --- Limits ---
maxclients 100                # more than enough for pipeline + insight
timeout 300                   # close idle connections after 5 min
tcp-keepalive 60

# --- Logging ---
loglevel notice
logfile /var/log/redis/redis-server.log
```

### 2.3 Enable and Start

```bash
sudo systemctl enable redis-server
sudo systemctl restart redis-server
sudo systemctl status redis-server

# Verify auth works
redis-cli -a '<REDIS_PASSWORD>' ping
# → PONG
```

### 2.4 Redis Insight (Web UI — Secured)

Redis Insight 2.x supports built-in authentication. Secure it with:

```bash
# Docker install with auth + TLS
docker run -d --name redis-insight \
  -p 127.0.0.1:5540:5540 \
  --restart unless-stopped \
  -e RI_PROXY_PATH="/insight" \
  -e RI_ENCRYPTION_KEY="$(openssl rand -hex 32)" \
  redis/redisinsight:2

# Bind to 127.0.0.1 only — no external access without SSH tunnel.
```

**Nginx reverse proxy with basic auth** (recommended for remote access):

```bash
sudo apt install -y nginx apache2-utils

# Create auth credentials
sudo htpasswd -c /etc/nginx/.htpasswd admin
# Enter a strong password when prompted

# Nginx config: /etc/nginx/sites-available/redis-insight
cat <<'EOF' | sudo tee /etc/nginx/sites-available/redis-insight
server {
    listen 8443 ssl;
    server_name _;

    # Self-signed TLS cert (or use Let's Encrypt for a real domain)
    ssl_certificate     /etc/nginx/ssl/insight.crt;
    ssl_certificate_key /etc/nginx/ssl/insight.key;
    ssl_protocols       TLSv1.2 TLSv1.3;

    location /insight/ {
        auth_basic "Redis Insight";
        auth_basic_user_file /etc/nginx/.htpasswd;

        proxy_pass http://127.0.0.1:5540/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
EOF

# Generate self-signed cert
sudo mkdir -p /etc/nginx/ssl
sudo openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout /etc/nginx/ssl/insight.key \
  -out /etc/nginx/ssl/insight.crt \
  -subj "/CN=server2"

sudo ln -sf /etc/nginx/sites-available/redis-insight /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx
```

Access: `https://server2:8443/insight/` (with basic auth + TLS).

**Alternative: SSH tunnel only** (simpler, no nginx needed):

```bash
# From local machine:
ssh -L 5540:localhost:5540 server2
# Then open http://localhost:5540
```

### 2.5 Firewall

```bash
# On server2: lock down all Redis-related ports
sudo ufw allow ssh
sudo ufw deny 6379       # Redis: localhost only
sudo ufw deny 5540       # Insight: localhost only (behind nginx or SSH tunnel)
sudo ufw allow 8443/tcp  # Nginx TLS proxy (if using nginx approach)
sudo ufw enable
```

---

## 3. Go Client Setup

Add `github.com/redis/go-redis/v9` (latest 2026 release) to the search module:

```bash
cd blueprints/search
go get github.com/redis/go-redis/v9@latest  # v9.18.0 (2026)
```

Connection helper in `cli/cc_redis.go`:

```go
package cli

import (
    "context"
    "os"
    "time"

    "github.com/redis/go-redis/v9"
)

// ccRedisClient returns a Redis client if REDIS_URL or REDIS_PASSWORD is set.
// Returns nil if Redis is not configured (graceful degradation).
func ccRedisClient() *redis.Client {
    addr := os.Getenv("REDIS_URL")
    if addr == "" {
        addr = "localhost:6379"
    }
    password := os.Getenv("REDIS_PASSWORD")
    if password == "" {
        return nil // Redis not configured — fall back to file-based state
    }
    return redis.NewClient(&redis.Options{
        Addr:         addr,
        Password:     password,
        DB:           0,
        DialTimeout:  5 * time.Second,
        ReadTimeout:  3 * time.Second,
        WriteTimeout: 3 * time.Second,
        PoolSize:     10,
    })
}

// ccRedisAvailable checks if a Redis client is connected and responsive.
func ccRedisAvailable(ctx context.Context, rdb *redis.Client) bool {
    if rdb == nil {
        return false
    }
    ctx2, cancel := context.WithTimeout(ctx, 2*time.Second)
    defer cancel()
    return rdb.Ping(ctx2).Err() == nil
}
```

---

## 4. Environment Variable

Set on server2 (in `~/.bashrc` or screen session env):

```bash
export REDIS_PASSWORD="<generated-password>"
# Optional: export REDIS_URL="localhost:6379"
```

---

## 5. Verification Checklist

- [ ] `redis-cli -a $REDIS_PASSWORD ping` returns PONG
- [ ] `redis-cli -a $REDIS_PASSWORD info memory` shows maxmemory=512MB
- [ ] Redis Insight accessible via SSH tunnel at localhost:5540
- [ ] Go client connects: `ccRedisAvailable(ctx, ccRedisClient())` returns true
- [ ] `FLUSHALL` is disabled (returns error)
- [ ] External connections refused (port 6379 not accessible from outside)

---

## 6. Resource Impact

| Resource | Before | After |
|----------|--------|-------|
| RAM | 0 | ~50-100 MB (Redis idle + 512 MB cap) |
| Disk | 0 | ~5-50 MB (RDB snapshots) |
| CPU | 0 | negligible (single-threaded, event-driven) |
| Ports | — | 6379 (Redis), 5540 (Insight) |

On a 12 GB server, 512 MB Redis is well within budget. The `maxmemory 512mb`
hard cap ensures Redis never competes with pipeline sessions for memory.
