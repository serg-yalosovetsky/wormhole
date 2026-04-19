#!/usr/bin/env bash
# Deploy wormhole-relay to a VPS using nginx + systemd + certbot
# Usage: ./deploy.sh <domain> <your-email>
# Example: ./deploy.sh relay.example.com admin@example.com
set -euo pipefail

DOMAIN="${1:?Usage: $0 <domain> <email>}"
EMAIL="${2:?Usage: $0 <domain> <email>}"

APP_DIR="/opt/wormhole-relay"
DATA_DIR="/var/lib/wormhole-relay"
SERVICE_USER="wormhole"
SERVICE_NAME="wormhole-relay"
PORT=8000

BOLD='\033[1m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; RED='\033[0;31m'; NC='\033[0m'
info()    { echo -e "${GREEN}[INFO]${NC} $*"; }
warn()    { echo -e "${YELLOW}[WARN]${NC} $*"; }
die()     { echo -e "${RED}[ERR]${NC} $*" >&2; exit 1; }

[[ $EUID -eq 0 ]] || die "Run as root (sudo $0 $*)"

# ── 1. System packages ────────────────────────────────────────────────────────
info "Installing system packages..."
apt-get update -qq
apt-get install -qq python3 python3-venv python3-pip nginx certbot python3-certbot-nginx

# ── 2. System user ────────────────────────────────────────────────────────────
info "Creating service user '${SERVICE_USER}'..."
id "${SERVICE_USER}" &>/dev/null || useradd --system --no-create-home --shell /usr/sbin/nologin "${SERVICE_USER}"

# ── 3. App directory ──────────────────────────────────────────────────────────
info "Deploying app to ${APP_DIR}..."
mkdir -p "${APP_DIR}" "${DATA_DIR}"

# Copy app files from the directory this script lives in
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cp "${SCRIPT_DIR}/main.py"           "${APP_DIR}/"
cp "${SCRIPT_DIR}/requirements.txt"  "${APP_DIR}/"

# Firebase service account — must exist before deploy
if [[ -f "${SCRIPT_DIR}/serviceAccount.json" ]]; then
    cp "${SCRIPT_DIR}/serviceAccount.json" "${APP_DIR}/"
    chmod 600 "${APP_DIR}/serviceAccount.json"
else
    warn "serviceAccount.json not found in ${SCRIPT_DIR}"
    warn "Place it at ${APP_DIR}/serviceAccount.json before starting the service"
fi

# ── 4. Python venv ────────────────────────────────────────────────────────────
info "Setting up Python virtualenv..."
python3 -m venv "${APP_DIR}/.venv"
"${APP_DIR}/.venv/bin/pip" install --quiet --upgrade pip
"${APP_DIR}/.venv/bin/pip" install --quiet -r "${APP_DIR}/requirements.txt"

chown -R "${SERVICE_USER}:${SERVICE_USER}" "${APP_DIR}" "${DATA_DIR}"

# ── 5. Systemd service ────────────────────────────────────────────────────────
info "Writing systemd unit..."
cat > "/etc/systemd/system/${SERVICE_NAME}.service" <<EOF
[Unit]
Description=Wormhole Relay (FastAPI)
After=network.target

[Service]
Type=simple
User=${SERVICE_USER}
WorkingDirectory=${APP_DIR}
Environment="DB_PATH=${DATA_DIR}/devices.db"
Environment="GOOGLE_APPLICATION_CREDENTIALS=${APP_DIR}/serviceAccount.json"
ExecStart=${APP_DIR}/.venv/bin/uvicorn main:app --host 127.0.0.1 --port ${PORT} --workers 1
Restart=on-failure
RestartSec=5
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ReadWritePaths=${DATA_DIR}

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable "${SERVICE_NAME}"
systemctl restart "${SERVICE_NAME}"
info "Service started: $(systemctl is-active ${SERVICE_NAME})"

# ── 6. Nginx config ───────────────────────────────────────────────────────────
info "Writing nginx config for ${DOMAIN}..."
cat > "/etc/nginx/sites-available/${SERVICE_NAME}" <<EOF
server {
    listen 80;
    server_name ${DOMAIN};

    location / {
        proxy_pass         http://127.0.0.1:${PORT};
        proxy_set_header   Host \$host;
        proxy_set_header   X-Real-IP \$remote_addr;
        proxy_set_header   X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header   X-Forwarded-Proto \$scheme;
        proxy_read_timeout 60s;
    }
}
EOF

ln -sf "/etc/nginx/sites-available/${SERVICE_NAME}" "/etc/nginx/sites-enabled/${SERVICE_NAME}"
nginx -t
systemctl reload nginx
info "Nginx reloaded"

# ── 7. TLS via Certbot ────────────────────────────────────────────────────────
info "Obtaining TLS certificate for ${DOMAIN}..."
certbot --nginx \
    --non-interactive \
    --agree-tos \
    --email "${EMAIL}" \
    --redirect \
    --domain "${DOMAIN}"

systemctl reload nginx
info "Certbot done — HTTPS enabled"

# ── 8. Firewall ───────────────────────────────────────────────────────────────
if command -v ufw &>/dev/null; then
    info "Configuring ufw..."
    ufw allow 22/tcp   comment "SSH"   2>/dev/null || true
    ufw allow 80/tcp   comment "HTTP"  2>/dev/null || true
    ufw allow 443/tcp  comment "HTTPS" 2>/dev/null || true
    ufw --force enable 2>/dev/null || true
fi

# ── Done ──────────────────────────────────────────────────────────────────────
echo ""
echo -e "${BOLD}${GREEN}Deployment complete!${NC}"
echo -e "  API:     https://${DOMAIN}/docs"
echo -e "  Logs:    journalctl -u ${SERVICE_NAME} -f"
echo -e "  Status:  systemctl status ${SERVICE_NAME}"
echo -e "  DB:      ${DATA_DIR}/devices.db"
echo ""

if [[ ! -f "${APP_DIR}/serviceAccount.json" ]]; then
    warn "Don't forget to place serviceAccount.json at ${APP_DIR}/serviceAccount.json"
    warn "Then run: systemctl restart ${SERVICE_NAME}"
fi
