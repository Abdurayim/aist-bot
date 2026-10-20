#!/usr/bin/env bash
#
# Backs up the bot's database.
#
#     ./deploy/backup.sh            # one backup, now
#
# Normally run by the aist-bot-backup.timer once a day.
#
# The database is the only copy of every complaint, proposal and thank-you
# note the school has ever received. Nothing else in this project stores
# them: the PDFs live in Telegram's cloud and are referenced by id, so a
# lost database cannot be rebuilt from anywhere.
#
# Two things make this more than a `cp`:
#
#   * SQLite is running in WAL mode while the bot is live, so a plain copy
#     can catch a half-written transaction. `.backup` takes a consistent
#     snapshot of a database that is being written to.
#   * A backup nobody has ever restored is a guess. Each one is opened and
#     integrity-checked before any older backup is pruned.

set -euo pipefail

APP_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

BACKUP_DIR="${BACKUP_DIR:-/var/backups/aist-bot}"
KEEP_DAYS="${KEEP_DAYS:-14}"
KEEP_MONTHLY_DAYS="${KEEP_MONTHLY_DAYS:-365}"

log()  { printf '\n\033[1m==> %s\033[0m\n' "$*"; }
note() { printf '    %s\n' "$*"; }
fail() { printf '\n\033[31m!! %s\033[0m\n' "$*" >&2; exit 1; }

# ── Where is the database? ───────────────────────────────────────────
# Read DB_PATH from .env rather than guessing, so the two cannot drift.
DB_PATH="$APP_DIR/aist_bot.db"
if [ -f "$APP_DIR/.env" ]; then
  from_env="$(grep -E '^DB_PATH=' "$APP_DIR/.env" | tail -1 | cut -d= -f2- | tr -d '"' | tr -d "'" || true)"
  [ -n "$from_env" ] && DB_PATH="$from_env"
fi

# A relative DB_PATH is relative to the working directory, which for the
# service is APP_DIR.
case "$DB_PATH" in
  /*) ;;
  *) DB_PATH="$APP_DIR/${DB_PATH#./}" ;;
esac

[ -f "$DB_PATH" ] || fail "No database at $DB_PATH"

command -v sqlite3 >/dev/null 2>&1 || fail "sqlite3 is not installed (apt-get install sqlite3)"

mkdir -p "$BACKUP_DIR"
chmod 700 "$BACKUP_DIR"

stamp="$(date +%Y-%m-%d_%H%M)"
day_of_month="$(date +%d)"
target="$BACKUP_DIR/aist_bot-$stamp.db"

# First of the month is kept far longer; mark it in the name so pruning
# can tell them apart.
if [ "$day_of_month" = "01" ]; then
  target="$BACKUP_DIR/aist_bot-$stamp-monthly.db"
fi

# ── Snapshot ─────────────────────────────────────────────────────────
log "Backing up $DB_PATH"

# .backup is safe against a live, actively written database; cp is not.
if ! sqlite3 "$DB_PATH" ".backup '$target'"; then
  rm -f "$target"
  fail "The snapshot failed — nothing was pruned, existing backups are untouched."
fi

# ── Verify before trusting it ────────────────────────────────────────
log "Verifying"

integrity="$(sqlite3 "$target" "PRAGMA integrity_check;" 2>&1 || true)"
if [ "$integrity" != "ok" ]; then
  rm -f "$target"
  fail "The backup did not pass its integrity check: $integrity"
fi

# A structurally valid but empty file would also pass the check above.
submissions="$(sqlite3 "$target" "SELECT COUNT(*) FROM submissions;" 2>/dev/null || echo "?")"
users="$(sqlite3 "$target" "SELECT COUNT(*) FROM users;" 2>/dev/null || echo "?")"

if [ "$submissions" = "?" ] || [ "$users" = "?" ]; then
  rm -f "$target"
  fail "The backup is unreadable — its tables are missing."
fi

note "integrity ok · $users users · $submissions appeals"

gzip -f "$target"
target="$target.gz"
chmod 600 "$target"

note "$(du -h "$target" | cut -f1) → $target"

# ── Prune, only now that a good backup exists ────────────────────────
log "Pruning"

before="$(find "$BACKUP_DIR" -name 'aist_bot-*.db.gz' | wc -l)"

# Daily backups older than KEEP_DAYS, except the monthly ones.
find "$BACKUP_DIR" -name 'aist_bot-*.db.gz' ! -name '*-monthly.db.gz' \
  -mtime "+$KEEP_DAYS" -delete

# Monthly backups older than a year.
find "$BACKUP_DIR" -name 'aist_bot-*-monthly.db.gz' \
  -mtime "+$KEEP_MONTHLY_DAYS" -delete

after="$(find "$BACKUP_DIR" -name 'aist_bot-*.db.gz' | wc -l)"

note "removed $((before - after)), keeping $after"
note "daily for $KEEP_DAYS days, first-of-month for $KEEP_MONTHLY_DAYS days"

# ── Off-server copy, if one is configured ────────────────────────────
# BACKUP_REMOTE is an rsync destination, e.g. user@host:/path/ — backups
# that live only on this disk do not survive losing this server.
if [ -n "${BACKUP_REMOTE:-}" ]; then
  log "Copying off the server"

  if rsync -az --chmod=600 "$target" "$BACKUP_REMOTE"; then
    note "sent to $BACKUP_REMOTE"
  else
    # Worth a loud warning, but the local backup is still good.
    printf '\033[33m    warning: could not reach %s — the local backup is fine\033[0m\n' "$BACKUP_REMOTE" >&2
  fi
fi

log "Done"
ls -lh "$BACKUP_DIR" | tail -5
