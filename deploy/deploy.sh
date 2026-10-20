#!/usr/bin/env bash
#
# Pulls the latest code from GitHub, builds it and restarts the bot.
#
# Run on the server, either by the GitHub Actions deploy workflow or by hand:
#
#     cd /opt/aist-bot && ./deploy/deploy.sh
#
# It is written to be safe to interrupt: the running binary is only replaced
# once a new one has built successfully, and if the service fails to come
# back up the previous binary is restored.

set -euo pipefail

SERVICE="${SERVICE:-aist-bot}"
BRANCH="${BRANCH:-main}"
EXPECTED_SHA="${1:-}"

APP_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$APP_DIR"

BIN="$APP_DIR/bin/aist-bot"
PREVIOUS="$APP_DIR/bin/aist-bot.previous"

log() { printf '\n\033[1m==> %s\033[0m\n' "$*"; }
fail() { printf '\n\033[31m!! %s\033[0m\n' "$*" >&2; exit 1; }

# ── 1. Fetch the new code ────────────────────────────────────────────
log "Fetching $BRANCH from GitHub"

git fetch --prune origin "$BRANCH"

# A hard reset rather than a merge: the server is a copy of GitHub, never a
# place where edits are made. Untracked files (.env, the database) are left
# alone, which is why neither is tracked.
git reset --hard "origin/$BRANCH"

CURRENT_SHA="$(git rev-parse HEAD)"
log "Now at ${CURRENT_SHA:0:7} — $(git log -1 --pretty=%s)"

if [ -n "$EXPECTED_SHA" ] && [ "$CURRENT_SHA" != "$EXPECTED_SHA" ]; then
  echo "note: expected ${EXPECTED_SHA:0:7}, got ${CURRENT_SHA:0:7} (a newer push probably landed first)"
fi

# ── 2. Build before touching the running bot ─────────────────────────
log "Building"

mkdir -p "$APP_DIR/bin"

# Build to a scratch path so a compile error leaves the running bot alone.
if ! go build -trimpath -o "$BIN.new" ./cmd/bot; then
  fail "Build failed — the running bot was left untouched."
fi

# ── 3. Swap the binary and restart ───────────────────────────────────
log "Restarting $SERVICE"

if [ -f "$BIN" ]; then
  cp -f "$BIN" "$PREVIOUS"
fi

mv -f "$BIN.new" "$BIN"

sudo systemctl restart "$SERVICE"

# ── 4. Make sure it actually came back ───────────────────────────────
log "Checking the service"

for _ in $(seq 1 15); do
  sleep 1
  if systemctl is-active --quiet "$SERVICE"; then
    # Give it a moment to get past config loading and the Telegram handshake,
    # which is where a bad .env or a revoked token shows up.
    sleep 3
    if systemctl is-active --quiet "$SERVICE"; then
      log "Deployed ${CURRENT_SHA:0:7} successfully"
      systemctl --no-pager --lines=10 status "$SERVICE" || true
      exit 0
    fi
  fi
done

# ── 5. Roll back ─────────────────────────────────────────────────────
printf '\n\033[31m!! The service did not come back up. Rolling back.\033[0m\n' >&2
journalctl -u "$SERVICE" --no-pager --lines=30 || true

if [ -f "$PREVIOUS" ]; then
  mv -f "$PREVIOUS" "$BIN"
  sudo systemctl restart "$SERVICE"
  sleep 3

  if systemctl is-active --quiet "$SERVICE"; then
    fail "Rolled back to the previous build, which is running. The new build is broken."
  fi

  fail "Rolled back, but the service is still down. Check the log above."
fi

fail "The service is down and there is no previous build to roll back to."
