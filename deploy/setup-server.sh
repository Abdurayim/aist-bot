#!/usr/bin/env bash
#
# One-time server setup. Run once, as root, on a fresh Ubuntu or Debian VPS:
#
#     curl -fsSL https://raw.githubusercontent.com/Abdurayim/aist-bot/main/deploy/setup-server.sh | sudo bash
#
# or, if you have the repo already:
#
#     sudo ./deploy/setup-server.sh
#
# Afterwards you still have to write /opt/aist-bot/.env — the script tells
# you exactly what to put in it.

set -euo pipefail

REPO="${REPO:-https://github.com/Abdurayim/aist-bot.git}"
APP_DIR="${APP_DIR:-/opt/aist-bot}"
APP_USER="${APP_USER:-aistbot}"
SERVICE="aist-bot"
GO_VERSION="${GO_VERSION:-1.24.5}"

log() { printf '\n\033[1m==> %s\033[0m\n' "$*"; }

if [ "$(id -u)" -ne 0 ]; then
  echo "Run this as root: sudo $0" >&2
  exit 1
fi

# ── Packages ─────────────────────────────────────────────────────────
log "Installing git, curl and build tools"
apt-get update -qq
apt-get install -y -qq git curl ca-certificates build-essential sqlite3

# ── Go ───────────────────────────────────────────────────────────────
if ! command -v go >/dev/null 2>&1 || [ "$(go version | awk '{print $3}')" != "go${GO_VERSION}" ]; then
  log "Installing Go ${GO_VERSION}"

  arch="$(dpkg --print-architecture)"
  case "$arch" in
    amd64) goarch=amd64 ;;
    arm64) goarch=arm64 ;;
    *) echo "Unsupported architecture: $arch" >&2; exit 1 ;;
  esac

  curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${goarch}.tar.gz" -o /tmp/go.tar.gz
  rm -rf /usr/local/go
  tar -C /usr/local -xzf /tmp/go.tar.gz
  rm -f /tmp/go.tar.gz

  # cgo is needed for the SQLite driver, so Go must be on the PATH of the
  # non-interactive SSH session the deploy workflow opens.
  ln -sf /usr/local/go/bin/go /usr/local/bin/go
fi

log "Go: $(go version)"

# ── Service account ──────────────────────────────────────────────────
if ! id -u "$APP_USER" >/dev/null 2>&1; then
  log "Creating the $APP_USER account"
  useradd --system --create-home --shell /bin/bash "$APP_USER"
fi

# ── Code ─────────────────────────────────────────────────────────────
if [ ! -d "$APP_DIR/.git" ]; then
  log "Cloning $REPO into $APP_DIR"
  mkdir -p "$APP_DIR"
  git clone "$REPO" "$APP_DIR"
else
  log "Repository already present, fetching"
  git -C "$APP_DIR" fetch --prune origin
fi

chown -R "$APP_USER:$APP_USER" "$APP_DIR"
git config --system --add safe.directory "$APP_DIR"

# ── Let the deploy user restart the service without a password ───────
log "Granting $APP_USER permission to restart $SERVICE"
cat > "/etc/sudoers.d/$SERVICE" <<SUDOERS
# The deploy script restarts the service, and reinstalls the unit file when
# it changes in the repo. Nothing else, and the install target is fixed so
# this cannot be turned into writing an arbitrary file.
$APP_USER ALL=(root) NOPASSWD: /bin/systemctl restart $SERVICE, /usr/bin/systemctl restart $SERVICE
$APP_USER ALL=(root) NOPASSWD: /bin/systemctl daemon-reload, /usr/bin/systemctl daemon-reload
$APP_USER ALL=(root) NOPASSWD: /usr/bin/install -m 644 -o root -g root $APP_DIR/deploy/$SERVICE.service /etc/systemd/system/$SERVICE.service
$APP_USER ALL=(root) NOPASSWD: /usr/bin/install -m 644 -o root -g root $APP_DIR/deploy/$SERVICE-backup.service /etc/systemd/system/$SERVICE-backup.service
$APP_USER ALL=(root) NOPASSWD: /usr/bin/install -m 644 -o root -g root $APP_DIR/deploy/$SERVICE-backup.timer /etc/systemd/system/$SERVICE-backup.timer
SUDOERS
chmod 440 "/etc/sudoers.d/$SERVICE"

# ── systemd ──────────────────────────────────────────────────────────
log "Installing the systemd units"
cp "$APP_DIR/deploy/$SERVICE.service" "/etc/systemd/system/$SERVICE.service"
cp "$APP_DIR/deploy/$SERVICE-backup.service" "/etc/systemd/system/$SERVICE-backup.service"
cp "$APP_DIR/deploy/$SERVICE-backup.timer" "/etc/systemd/system/$SERVICE-backup.timer"
systemctl daemon-reload
systemctl enable "$SERVICE" >/dev/null

log "Setting up daily database backups"
install -d -m 700 -o "$APP_USER" -g "$APP_USER" /var/backups/$SERVICE
systemctl enable --now "$SERVICE-backup.timer" >/dev/null
note_next="$(systemctl list-timers "$SERVICE-backup.timer" --no-pager --no-legend 2>/dev/null | awk '{print $1, $2, $3}')"
echo "    next run: ${note_next:-scheduled}"

# ── What is left for a human ─────────────────────────────────────────
cat <<DONE

$(printf '\033[1m')Server is ready. Two things left:$(printf '\033[0m')

  1. Write the configuration:

       sudo -u $APP_USER nano $APP_DIR/.env

     with at least:

       BOT_TOKEN=<token from @BotFather>
       ADMIN_PHONES=+998901112233,+998935556677
       DB_PATH=$APP_DIR/aist_bot.db
       ORG_NAME=AIST School

  2. Build and start it:

       sudo -u $APP_USER $APP_DIR/deploy/deploy.sh

Then watch it with:

       journalctl -u $SERVICE -f

For automatic deploys, add this server's details to the repository secrets
on GitHub — see DEPLOY.md.

DONE
