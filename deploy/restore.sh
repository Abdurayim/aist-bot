#!/usr/bin/env bash
#
# Restores the database from a backup.
#
#     sudo ./deploy/restore.sh                       # newest backup
#     sudo ./deploy/restore.sh /var/backups/aist-bot/aist_bot-2026-10-20_0330.db.gz
#     ./deploy/restore.sh --list                     # what is available
#
# A backup you have never restored is a guess, so this exists to be run —
# ideally once now, on purpose, rather than for the first time during an
# emergency.
#
# The database currently in place is itself backed up before anything is
# overwritten, so a restore from the wrong file is recoverable.

set -euo pipefail

APP_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

BACKUP_DIR="${BACKUP_DIR:-/var/backups/aist-bot}"
SERVICE="${SERVICE:-aist-bot}"
APP_USER="${APP_USER:-aistbot}"

log()  { printf '\n\033[1m==> %s\033[0m\n' "$*"; }
note() { printf '    %s\n' "$*"; }
fail() { printf '\n\033[31m!! %s\033[0m\n' "$*" >&2; exit 1; }

DB_PATH="$APP_DIR/aist_bot.db"
if [ -f "$APP_DIR/.env" ]; then
  from_env="$(grep -E '^DB_PATH=' "$APP_DIR/.env" | tail -1 | cut -d= -f2- | tr -d '"' | tr -d "'" || true)"
  [ -n "$from_env" ] && DB_PATH="$from_env"
fi
case "$DB_PATH" in
  /*) ;;
  *) DB_PATH="$APP_DIR/${DB_PATH#./}" ;;
esac

# ── --list ───────────────────────────────────────────────────────────
if [ "${1:-}" = "--list" ]; then
  log "Backups in $BACKUP_DIR"

  if ! ls "$BACKUP_DIR"/aist_bot-*.db.gz >/dev/null 2>&1; then
    note "none yet"
    exit 0
  fi

  for file in $(ls -1t "$BACKUP_DIR"/aist_bot-*.db.gz); do
    rows="$(zcat "$file" > /tmp/.peek.db 2>/dev/null && sqlite3 /tmp/.peek.db "SELECT COUNT(*) FROM submissions;" 2>/dev/null || echo '?')"
    rm -f /tmp/.peek.db
    printf '    %-12s %-22s %s appeals\n' "$(du -h "$file" | cut -f1)" "$(basename "$file")" "$rows"
  done

  exit 0
fi

[ "$(id -u)" -eq 0 ] || fail "Run this as root: sudo $0 $*"

# ── Pick the backup ──────────────────────────────────────────────────
SOURCE="${1:-}"

if [ -z "$SOURCE" ]; then
  SOURCE="$(ls -1t "$BACKUP_DIR"/aist_bot-*.db.gz 2>/dev/null | head -1 || true)"
  [ -n "$SOURCE" ] || fail "No backups found in $BACKUP_DIR"
  note "Using the newest backup: $(basename "$SOURCE")"
fi

[ -f "$SOURCE" ] || fail "No such backup: $SOURCE"

# ── Unpack and check it before touching anything ─────────────────────
log "Checking $(basename "$SOURCE")"

staged="$(mktemp /tmp/aist-restore-XXXXXX.db)"
trap 'rm -f "$staged"' EXIT

if [[ "$SOURCE" == *.gz ]]; then
  zcat "$SOURCE" > "$staged"
else
  cp "$SOURCE" "$staged"
fi

integrity="$(sqlite3 "$staged" "PRAGMA integrity_check;" 2>&1 || true)"
[ "$integrity" = "ok" ] || fail "That backup is damaged: $integrity"

in_users="$(sqlite3 "$staged" "SELECT COUNT(*) FROM users;")"
in_appeals="$(sqlite3 "$staged" "SELECT COUNT(*) FROM submissions;")"

note "integrity ok · $in_users users · $in_appeals appeals"

# ── Show what is being replaced ──────────────────────────────────────
if [ -f "$DB_PATH" ]; then
  now_users="$(sqlite3 "$DB_PATH" "SELECT COUNT(*) FROM users;" 2>/dev/null || echo '?')"
  now_appeals="$(sqlite3 "$DB_PATH" "SELECT COUNT(*) FROM submissions;" 2>/dev/null || echo '?')"

  log "Replacing the current database"
  note "now:     $now_users users · $now_appeals appeals"
  note "restore: $in_users users · $in_appeals appeals"

  if [ "$now_appeals" != "?" ] && [ "$in_appeals" -lt "$now_appeals" ]; then
    printf '\033[33m    warning: the backup holds FEWER appeals than the live database.\033[0m\n'
    printf '\033[33m             %s would be lost.\033[0m\n' "$((now_appeals - in_appeals))"
  fi

  if [ -t 0 ] && [ "${FORCE:-}" != "1" ]; then
    read -r -p "    Continue? [y/N] " answer
    case "$answer" in
      [yY]*) ;;
      *) fail "Cancelled; nothing was changed." ;;
    esac
  fi
fi

# ── Do it ────────────────────────────────────────────────────────────
log "Stopping $SERVICE"
systemctl stop "$SERVICE" || true

if [ -f "$DB_PATH" ]; then
  safety="$DB_PATH.replaced-$(date +%Y-%m-%d_%H%M)"
  cp "$DB_PATH" "$safety"
  note "the database being replaced was saved to $safety"
fi

log "Restoring"

# The write-ahead log and shared-memory files belong to the old database;
# leaving them behind would corrupt the restored one.
rm -f "$DB_PATH-wal" "$DB_PATH-shm"

cp "$staged" "$DB_PATH"
chown "$APP_USER:$APP_USER" "$DB_PATH"
chmod 640 "$DB_PATH"

log "Starting $SERVICE"
systemctl start "$SERVICE"

sleep 5

if systemctl is-active --quiet "$SERVICE"; then
  log "Restored"
  note "$(sqlite3 "$DB_PATH" 'SELECT COUNT(*) FROM users;') users · $(sqlite3 "$DB_PATH" 'SELECT COUNT(*) FROM submissions;') appeals"
  systemctl --no-pager --lines=5 status "$SERVICE" || true
else
  fail "Restored the file, but the service did not start. Check: journalctl -u $SERVICE -n 30"
fi
