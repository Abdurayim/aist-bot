#!/usr/bin/env bash
#
# Makes the server survive another project running it out of memory.
#
#     sudo ./deploy/setup-resilience.sh
#
# Safe to re-run: every step checks whether it has already been done.
#
# Without swap, a VPS under memory pressure has only one move — the kernel
# OOM killer picks a victim and takes it. It often picks badly: sshd is
# cheap to kill and frees nothing, and once it is gone you cannot get in to
# fix anything. This script addresses that from three sides:
#
#   1. swap            — room to breathe instead of an instant kill
#   2. earlyoom        — kill the actual memory hog, early, while the box
#                        is still responsive, instead of freezing first
#   3. OOM scores      — make sshd and the bot the last things considered
#
# None of it is specific to this bot; it protects whatever else you run here.

set -euo pipefail

SWAP_FILE="${SWAP_FILE:-/swapfile}"
SERVICE="${SERVICE:-aist-bot}"

log()  { printf '\n\033[1m==> %s\033[0m\n' "$*"; }
note() { printf '    %s\n' "$*"; }

if [ "$(id -u)" -ne 0 ]; then
  echo "Run this as root: sudo $0" >&2
  exit 1
fi

# ── 1. Swap ──────────────────────────────────────────────────────────
log "Swap"

if [ "$(swapon --show --noheadings | wc -l)" -gt 0 ]; then
  note "Already configured:"
  swapon --show
else
  # Match RAM, within sane bounds. On a small VPS this is the difference
  # between a brief slowdown and a process being killed outright.
  ram_mb="$(free -m | awk '/^Mem:/ {print $2}')"
  swap_mb="$ram_mb"
  [ "$swap_mb" -lt 2048 ] && swap_mb=2048
  [ "$swap_mb" -gt 8192 ] && swap_mb=8192

  avail_mb="$(df -m --output=avail / | tail -1 | tr -d ' ')"
  if [ "$avail_mb" -lt $((swap_mb + 2048)) ]; then
    note "Only ${avail_mb}MB free on /; shrinking the swap file to fit."
    swap_mb=$(( avail_mb > 3072 ? avail_mb - 2048 : 1024 ))
  fi

  note "Creating ${swap_mb}MB at $SWAP_FILE"

  # fallocate is instant but produces a sparse file on some filesystems,
  # which swapon rejects; dd always works.
  if ! fallocate -l "${swap_mb}M" "$SWAP_FILE" 2>/dev/null; then
    dd if=/dev/zero of="$SWAP_FILE" bs=1M count="$swap_mb" status=none
  fi

  chmod 600 "$SWAP_FILE"
  mkswap "$SWAP_FILE" >/dev/null
  swapon "$SWAP_FILE"

  if ! grep -q "^${SWAP_FILE}[[:space:]]" /etc/fstab; then
    printf '%s none swap sw 0 0\n' "$SWAP_FILE" >> /etc/fstab
    note "Added to /etc/fstab so it survives a reboot"
  fi
fi

# ── 2. Kernel memory behaviour ───────────────────────────────────────
log "Kernel settings"

cat > /etc/sysctl.d/60-aist-memory.conf <<'SYSCTL'
# Swap only when genuinely short of memory. The default of 60 is tuned for
# desktops and pushes idle server processes to disk for no reason.
vm.swappiness = 10

# Keep directory and inode caches rather than dropping them under pressure.
vm.vfs_cache_pressure = 50

# Let the kernel overcommit heuristically, but refuse the wild allocations
# that cause a sudden unrecoverable kill.
vm.overcommit_memory = 0

# Reserve a little memory so root can still act when userland is starving.
vm.admin_reserve_kbytes = 65536
SYSCTL

sysctl -q --system
note "swappiness=$(cat /proc/sys/vm/swappiness), vfs_cache_pressure=$(cat /proc/sys/vm/vfs_cache_pressure)"

# ── 3. earlyoom ──────────────────────────────────────────────────────
log "earlyoom"

if ! dpkg -s earlyoom >/dev/null 2>&1; then
  note "Installing"
  DEBIAN_FRONTEND=noninteractive apt-get update -qq
  DEBIAN_FRONTEND=noninteractive apt-get install -y -qq earlyoom
fi

# -m 5 / -s 5: act once free RAM *and* swap fall under 5%.
# --avoid: never these if anything else will do.
# --prefer: the usual runaway memory hogs.
cat > /etc/default/earlyoom <<EARLYOOM
EARLYOOM_ARGS="-m 5 -s 5 -r 3600 --avoid '^(sshd|systemd|${SERVICE}|init)$' --prefer '^(node|java|python3?|chrome|mysqld|mongod|postgres)$'"
EARLYOOM
systemctl enable --now earlyoom >/dev/null 2>&1 || true
systemctl restart earlyoom
note "$(systemctl is-active earlyoom) — kills the biggest hog before the box freezes"

# ── 4. Protect the way back in ───────────────────────────────────────
log "Protecting sshd"

# If sshd is killed you cannot fix anything, and killing it frees almost
# nothing. Make it the last thing the kernel considers.
mkdir -p /etc/systemd/system/ssh.service.d
cat > /etc/systemd/system/ssh.service.d/oom.conf <<'SSHD'
[Service]
OOMScoreAdjust=-900
SSHD

systemctl daemon-reload
note "sshd OOMScoreAdjust=-900"

# ── 5. Report ────────────────────────────────────────────────────────
log "Done"

free -h
echo
swapon --show
echo
if systemctl list-unit-files "${SERVICE}.service" >/dev/null 2>&1; then
  printf '%-22s %s\n' "$SERVICE:" "$(systemctl is-active "$SERVICE") / $(systemctl is-enabled "$SERVICE") on boot"
  pid="$(systemctl show "$SERVICE" -p MainPID --value)"
  if [ "$pid" != "0" ] && [ -r "/proc/$pid/oom_score_adj" ]; then
    printf '%-22s %s\n' "its OOM score adj:" "$(cat "/proc/$pid/oom_score_adj")"
  fi
fi
printf '%-22s %s\n' "earlyoom:" "$(systemctl is-active earlyoom)"
