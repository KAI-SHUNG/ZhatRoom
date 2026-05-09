#!/bin/bash
# ZhatRoom deployment script
#
# Usage:
#   sudo ./scripts/deploy.sh           # Full deploy (first time)
#   sudo ./scripts/deploy.sh --update  # Update only (build → install → restart)
#
# Full deploy will:
#   1. Create chat system user
#   2. Build Go binaries
#   3. Set up /opt/zhatroom/ directory (all owned by chat)
#   4. Start PostgreSQL via Docker
#   5. Install and enable systemd service (runs as chat)
#   6. Configure SSH for the chat user
#
# Update mode will:
#   1. Build Go binaries
#   2. Install to /opt/zhatroom/
#   3. Clean stale socket and restart service

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
INSTALL_DIR="/opt/zhatroom"
SOCKET_PATH="/tmp/zhatroom.sock"

UPDATE_ONLY=false
if [ "$1" = "--update" ] || [ "$1" = "-u" ]; then
    UPDATE_ONLY=true
fi

# ── Build & Install (shared by both modes) ─────────────────────────
build_and_install() {
    echo "[BUILD] Building Go binaries..."
    cd "$PROJECT_DIR"
    go build -o bin/server cmd/server/main.go
    go build -o bin/client cmd/client/main.go
    go build -o bin/zhatroom cmd/zhatroom/main.go
    echo "  Binaries built: server, client, zhatroom"

    echo "[INSTALL] Installing files to $INSTALL_DIR..."
    mkdir -p "$INSTALL_DIR/bin"
    cp bin/server "$INSTALL_DIR/bin/server"
    cp bin/client "$INSTALL_DIR/bin/client"
    cp bin/zhatroom /usr/local/bin/zhatroom
    cp scripts/entrypoint.sh "$INSTALL_DIR/entrypoint.sh"
    chmod +x "$INSTALL_DIR/entrypoint.sh"

    # Create empty authorized_keys if not exists
    if [ ! -f "$INSTALL_DIR/authorized_keys" ]; then
        touch "$INSTALL_DIR/authorized_keys"
        chmod 600 "$INSTALL_DIR/authorized_keys"
    fi

    chown -R chat:chat "$INSTALL_DIR"
    echo "  Files installed"
}

# ── Stop service ──────────────────────────────────────────────────
stop_service() {
    if systemctl is-active --quiet zhatroom 2>/dev/null; then
        echo "[STOP] Stopping zhatroom service..."
        systemctl stop zhatroom
        echo "  Service stopped"
    fi
}

# ── Clean stale socket & start service ────────────────────────────
start_service() {
    # Remove stale socket left by a crashed server
    if [ -e "$SOCKET_PATH" ]; then
        echo "[START] Removing stale socket: $SOCKET_PATH"
        rm -f "$SOCKET_PATH"
    fi

    echo "[START] Starting zhatroom service..."
    systemctl start zhatroom
    sleep 1
    systemctl status zhatroom --no-pager -l 2>/dev/null || true
    echo "  Service started"
}

# ════════════════════════════════════════════════════════════════════
# Update mode
# ════════════════════════════════════════════════════════════════════
if [ "$UPDATE_ONLY" = true ]; then
    echo "=== ZhatRoom Update ==="
    echo ""
    stop_service
    build_and_install
    start_service
    echo ""
    echo "=== Update Complete ==="
    exit 0
fi

# ════════════════════════════════════════════════════════════════════
# Full deploy
# ════════════════════════════════════════════════════════════════════
echo "=== ZhatRoom Deployment ==="
echo ""

# ── 1. System user ────────────────────────────────────────────────
echo "[1/7] Creating chat system user..."

if ! id chat &>/dev/null; then
    useradd -r -s /bin/sh -m -d /home/chat -G docker chat
    echo "  Created user: chat (shell=/bin/sh, docker group)"
else
    usermod -s /bin/sh chat
    usermod -aG docker chat
    echo "  User chat already exists, shell set to /bin/sh"
fi

# ── 2-3. Build & Install ──────────────────────────────────────────
echo "[2/7] & [3/7] Building and installing..."
build_and_install

# ── 4. PostgreSQL via Docker ─────────────────────────────────────
echo "[4/7] Starting PostgreSQL..."
if ! command -v docker &>/dev/null; then
    echo "  ERROR: Docker is not installed. Please install Docker first."
    exit 1
fi

cd "$PROJECT_DIR"
if ! docker compose ps | grep -q zhat_db; then
    docker compose up -d
    echo "  PostgreSQL started (container: zhat_db)"
else
    echo "  PostgreSQL already running"
fi

# ── 5. systemd service ───────────────────────────────────────────
echo "[5/7] Installing systemd service..."
cp scripts/zhatroom.service /etc/systemd/system/zhatroom.service
systemctl daemon-reload
systemctl enable zhatroom

# ── 6. SSH configuration ─────────────────────────────────────────
echo "[6/7] Configuring SSH for chat user..."

if [ -d /etc/ssh/sshd_config.d ]; then
    cat > /etc/ssh/sshd_config.d/99-zhatroom.conf <<EOF
Match User chat
    AuthorizedKeysFile $INSTALL_DIR/authorized_keys
    PermitTTY yes
EOF
    echo "  SSH configured via /etc/ssh/sshd_config.d/99-zhatroom.conf"
else
    SSHD_CONFIG="/etc/ssh/sshd_config"
    if ! grep -q "Match User chat" "$SSHD_CONFIG"; then
        echo "" >> "$SSHD_CONFIG"
        echo "Match User chat" >> "$SSHD_CONFIG"
        echo "    AuthorizedKeysFile $INSTALL_DIR/authorized_keys" >> "$SSHD_CONFIG"
        echo "    PermitTTY yes" >> "$SSHD_CONFIG"
    fi
    echo "  SSH configured via $SSHD_CONFIG"
fi

systemctl reload sshd 2>/dev/null || service ssh reload 2>/dev/null || true
echo "  SSH reloaded"

# ── 7. Start service ─────────────────────────────────────────────
echo "[7/7] Starting service..."
start_service

echo ""
echo "=== Deployment Complete ==="
echo ""
echo "  Next steps:"
echo "    1. Add a user:  sudo zhatroom user add <name> < their_key.pub"
echo "    2. Ask them to: ssh chat@$(hostname -I | awk '{print $1}')"
