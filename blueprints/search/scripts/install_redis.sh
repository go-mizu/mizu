#!/usr/bin/env bash
# Idempotent Redis + Redis Insight install script for server2.
# Safe to run multiple times. Only changes what needs changing.
#
# Usage:
#   bash scripts/install_redis.sh              # local
#   make deploy-redis SERVER=2                 # remote via Makefile
#
set -euo pipefail

echo "=== Redis Install (idempotent) ==="

# ── 1. Install Redis if not present ──────────────────────────────────────────
if command -v redis-server &>/dev/null; then
    echo "  Redis already installed: $(redis-server --version | head -1)"
else
    echo "  Installing redis-server..."
    export DEBIAN_FRONTEND=noninteractive
    apt-get update -qq
    apt-get install -y -qq redis-server redis-tools
    echo "  Installed: $(redis-server --version | head -1)"
fi

# ── 2. Generate password (once) ──────────────────────────────────────────────
PW_FILE="/root/.redis_password"
if [ -f "$PW_FILE" ]; then
    REDIS_PW=$(cat "$PW_FILE")
    echo "  Password: using existing ($PW_FILE)"
else
    REDIS_PW=$(openssl rand -base64 32)
    echo "$REDIS_PW" > "$PW_FILE"
    chmod 600 "$PW_FILE"
    echo "  Password: generated → $PW_FILE"
fi

# ── 3. Configure Redis (overwrite config) ────────────────────────────────────
CONF="/etc/redis/redis.conf"
echo "  Writing config → $CONF"
cat > "$CONF" << REDISEOF
# Managed by scripts/install_redis.sh — do not edit manually.
bind 127.0.0.1 -::1
port 6379
protected-mode yes
requirepass $REDIS_PW

# Memory: 512 MB hard cap (server has ~12 GB; pipeline needs ~10 GB)
maxmemory 512mb
maxmemory-policy allkeys-lru

# Persistence: RDB snapshots (low overhead, state is reconstructible)
save 900 1
save 300 10
save 60 10000
dbfilename redis-cc.rdb
dir /var/lib/redis
appendonly no

# Limits
maxclients 100
timeout 300
tcp-keepalive 60

# Logging
loglevel notice
logfile /var/log/redis/redis-server.log
REDISEOF

# ── 4. Enable and restart Redis ──────────────────────────────────────────────
systemctl enable redis-server --quiet 2>/dev/null || true
systemctl restart redis-server
sleep 1

# Verify
if redis-cli -a "$REDIS_PW" --no-auth-warning ping | grep -q PONG; then
    echo "  Redis: PONG (OK)"
else
    echo "  ERROR: Redis not responding"
    exit 1
fi

# ── 5. Set REDIS_PASSWORD in environment (idempotent) ────────────────────────
BASHRC="/root/.bashrc"
if grep -q 'export REDIS_PASSWORD=' "$BASHRC" 2>/dev/null; then
    # Update existing line
    sed -i "s|^export REDIS_PASSWORD=.*|export REDIS_PASSWORD=\"$REDIS_PW\"|" "$BASHRC"
    echo "  .bashrc: updated REDIS_PASSWORD"
else
    echo "export REDIS_PASSWORD=\"$REDIS_PW\"" >> "$BASHRC"
    echo "  .bashrc: added REDIS_PASSWORD"
fi

# ── 6. Redis Insight (Docker, if available) ──────────────────────────────────
if command -v docker &>/dev/null; then
    if docker ps -a --format '{{.Names}}' | grep -q '^redis-insight$'; then
        echo "  Redis Insight: already exists"
        docker start redis-insight 2>/dev/null || true
    else
        echo "  Redis Insight: installing via Docker..."
        docker run -d --name redis-insight \
            -p 127.0.0.1:5540:5540 \
            --restart unless-stopped \
            redis/redisinsight:latest 2>/dev/null || echo "  Redis Insight: docker run failed (non-fatal)"
    fi
    if docker ps --format '{{.Names}}' | grep -q '^redis-insight$'; then
        echo "  Redis Insight: running at http://localhost:5540"
    fi
else
    echo "  Redis Insight: skipped (Docker not installed)"
fi

# ── 7. Summary ───────────────────────────────────────────────────────────────
echo ""
echo "=== Redis Ready ==="
redis-cli -a "$REDIS_PW" --no-auth-warning info server | grep redis_version
redis-cli -a "$REDIS_PW" --no-auth-warning info memory | grep used_memory_human
echo "  Password: $PW_FILE"
echo "  Config:   $CONF"
echo ""
echo "  Test: redis-cli -a \$(cat $PW_FILE) --no-auth-warning ping"
echo "  Go:   export REDIS_PASSWORD=\$(cat $PW_FILE)"
