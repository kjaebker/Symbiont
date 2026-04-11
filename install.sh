#!/usr/bin/env bash
# Symbiont installer
# Usage: bash <(curl -fsSL https://raw.githubusercontent.com/kjaebker/Symbiont/main/install.sh)
set -euo pipefail

REPO="kjaebker/Symbiont"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/symbiont"
DATA_DIR="/var/lib/symbiont"
SERVICE_NAME="symbiont"

# ── Detect platform ───────────────────────────────────────────────────────────

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  arm64)   ARCH="arm64" ;;
  *)
    echo "Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

case "$OS" in
  linux|darwin) ;;
  *)
    echo "Unsupported OS: $OS"
    exit 1
    ;;
esac

ASSET="symbiont-${OS}-${ARCH}.tar.gz"

# ── Fetch latest release ──────────────────────────────────────────────────────

echo "==> Fetching latest release from github.com/${REPO}..."

if command -v curl &>/dev/null; then
  LATEST=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')
  URL="https://github.com/${REPO}/releases/download/${LATEST}/${ASSET}"
  echo "    Version: ${LATEST}"
  echo "    Asset:   ${ASSET}"
  curl -fsSL "$URL" -o "/tmp/${ASSET}"
elif command -v wget &>/dev/null; then
  LATEST=$(wget -qO- "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')
  URL="https://github.com/${REPO}/releases/download/${LATEST}/${ASSET}"
  echo "    Version: ${LATEST}"
  echo "    Asset:   ${ASSET}"
  wget -q "$URL" -O "/tmp/${ASSET}"
else
  echo "curl or wget is required"
  exit 1
fi

# ── Install binary ────────────────────────────────────────────────────────────

echo "==> Installing symbiont to ${INSTALL_DIR}..."
tar -xzf "/tmp/${ASSET}" -C /tmp ./symbiont

if [ -w "$INSTALL_DIR" ]; then
  mv /tmp/symbiont "${INSTALL_DIR}/symbiont"
else
  sudo mv /tmp/symbiont "${INSTALL_DIR}/symbiont"
fi
chmod +x "${INSTALL_DIR}/symbiont"
rm -f "/tmp/${ASSET}"

echo "    Installed: $(symbiont --version 2>/dev/null || echo "${INSTALL_DIR}/symbiont")"

# ── Config ────────────────────────────────────────────────────────────────────

if [ -f "${CONFIG_DIR}/env" ]; then
  echo "==> Config already exists at ${CONFIG_DIR}/env — skipping."
else
  echo ""
  echo "==> Configure your Apex controller"
  echo "    (Press Enter to accept the default shown in brackets)"
  echo ""

  # Apex URL
  read -rp "    Apex controller IP or URL [http://192.168.1.100]: " APEX_URL
  APEX_URL="${APEX_URL:-http://192.168.1.100}"
  # Strip trailing slash
  APEX_URL="${APEX_URL%/}"
  # Prepend http:// if no scheme given
  case "$APEX_URL" in
    http://*|https://*) ;;
    *) APEX_URL="http://${APEX_URL}" ;;
  esac

  # Username
  read -rp "    Apex username [admin]: " APEX_USER
  APEX_USER="${APEX_USER:-admin}"

  # Password (hidden input)
  read -rsp "    Apex password: " APEX_PASS
  echo ""

  # Data dir
  if [ "$OS" = "darwin" ]; then
    DEFAULT_DATA_DIR="$HOME/.symbiont"
  else
    DEFAULT_DATA_DIR="/var/lib/symbiont"
  fi

  echo ""
  echo "==> Writing config to ${CONFIG_DIR}/env..."

  if [ -w "$(dirname "$CONFIG_DIR")" ]; then
    mkdir -p "$CONFIG_DIR"
  else
    sudo mkdir -p "$CONFIG_DIR"
  fi

  CONFIG_CONTENT="SYMBIONT_APEX_URL=${APEX_URL}
SYMBIONT_APEX_USER=${APEX_USER}
SYMBIONT_APEX_PASS=${APEX_PASS}
SYMBIONT_DB_PATH=${DEFAULT_DATA_DIR}/telemetry.db
SYMBIONT_SQLITE_PATH=${DEFAULT_DATA_DIR}/app.db
SYMBIONT_POLL_INTERVAL=10s
SYMBIONT_API_PORT=8420
SYMBIONT_TOKEN=
SYMBIONT_LOG_LEVEL=info
SYMBIONT_RETENTION_DAYS=365
"

  if [ -w "$CONFIG_DIR" ]; then
    printf '%s' "$CONFIG_CONTENT" > "${CONFIG_DIR}/env"
    chmod 600 "${CONFIG_DIR}/env"
  else
    printf '%s' "$CONFIG_CONTENT" | sudo tee "${CONFIG_DIR}/env" > /dev/null
    sudo chmod 600 "${CONFIG_DIR}/env"
  fi

  echo "    Saved."
fi

# ── Linux: systemd service ────────────────────────────────────────────────────

if [ "$OS" = "linux" ] && command -v systemctl &>/dev/null; then
  UNIT_FILE="/etc/systemd/system/${SERVICE_NAME}.service"

  if systemctl is-active --quiet "$SERVICE_NAME"; then
    echo "==> Restarting ${SERVICE_NAME} service..."
    sudo systemctl restart "$SERVICE_NAME"
    echo "    Restarted."
  fi

  if [ ! -f "$UNIT_FILE" ]; then
    echo "==> Installing systemd service..."

    UNIT="[Unit]
Description=Symbiont Neptune Apex Dashboard
After=network.target

[Service]
ExecStart=${INSTALL_DIR}/symbiont serve
EnvironmentFile=${CONFIG_DIR}/env
WorkingDirectory=${DATA_DIR}
Restart=always
RestartSec=5s
PrivateTmp=true
NoNewPrivileges=true
StateDirectory=symbiont

[Install]
WantedBy=multi-user.target"

    if [ -w "/etc/systemd/system" ]; then
      echo "$UNIT" > "$UNIT_FILE"
      systemctl daemon-reload
    else
      echo "$UNIT" | sudo tee "$UNIT_FILE" > /dev/null
      sudo systemctl daemon-reload
    fi

    echo "    Service installed. Start with:"
    echo "      sudo systemctl enable --now symbiont"
  else
    echo "    systemd service already exists — skipping."
  fi
fi

# ── macOS: launchd plist ──────────────────────────────────────────────────────

if [ "$OS" = "darwin" ]; then
  PLIST="$HOME/Library/LaunchAgents/com.symbiont.plist"

  if [ ! -f "$PLIST" ]; then
    echo "==> Installing launchd agent..."
    mkdir -p "$HOME/Library/LaunchAgents"
    mkdir -p "$DATA_DIR" 2>/dev/null || sudo mkdir -p "$DATA_DIR"

    cat > "$PLIST" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>com.symbiont</string>
  <key>ProgramArguments</key>
  <array>
    <string>${INSTALL_DIR}/symbiont</string>
    <string>serve</string>
  </array>
  <key>EnvironmentVariables</key>
  <dict>
    <key>SYMBIONT_DB_PATH</key>
    <string>${HOME}/.symbiont/telemetry.db</string>
    <key>SYMBIONT_SQLITE_PATH</key>
    <string>${HOME}/.symbiont/app.db</string>
  </dict>
  <key>WorkingDirectory</key>
  <string>${HOME}/.symbiont</string>
  <key>StandardOutPath</key>
  <string>${HOME}/.symbiont/symbiont.log</string>
  <key>StandardErrorPath</key>
  <string>${HOME}/.symbiont/symbiont.log</string>
  <key>RunAtLoad</key>
  <false/>
  <key>KeepAlive</key>
  <true/>
</dict>
</plist>
EOF
    mkdir -p "$HOME/.symbiont"
    echo "    launchd agent installed."
    echo "    Edit ${CONFIG_DIR}/env (or set vars in the plist) then start with:"
    echo "      launchctl load ~/Library/LaunchAgents/com.symbiont.plist"
  else
    echo "    launchd agent already exists — skipping."
  fi
fi

# ── Done ──────────────────────────────────────────────────────────────────────

echo ""
echo "==> Symbiont installed!"
echo ""
echo "Next steps:"
if [ "$OS" = "linux" ] && command -v systemctl &>/dev/null; then
echo "  1. sudo systemctl enable --now symbiont"
echo "  2. sudo journalctl -u symbiont -f    (token is printed on first run — save it)"
elif [ "$OS" = "darwin" ]; then
echo "  1. launchctl load ~/Library/LaunchAgents/com.symbiont.plist"
echo "  2. tail -f ~/.symbiont/symbiont.log  (token is printed on first run — save it)"
else
echo "  1. symbiont serve                    (token is printed on first run — save it)"
fi
echo ""
echo "Config: ${CONFIG_DIR}/env  (edit anytime to change settings)"
echo ""
echo "Dashboard: http://localhost:8420"
