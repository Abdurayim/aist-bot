# Deployment

The bot runs on a Linux server as a systemd service. Pushing to `main`
triggers a GitHub Actions workflow that opens an SSH session to the server;
the **server** then pulls the new code from GitHub, builds it and restarts
itself. Nothing is uploaded from the workflow except the instruction to go.

```
 git push ──▶ GitHub ──▶ Actions ──▶ ssh ──▶ server
                 ▲                            │
                 └──────── git pull ──────────┘
                          build, restart, verify
```

If the new build fails to start, the server **rolls back** to the previous
binary on its own.

---

## 1. Prepare the server

On a fresh Ubuntu or Debian VPS, as root:

```bash
curl -fsSL https://raw.githubusercontent.com/Abdurayim/aist-bot/main/deploy/setup-server.sh | sudo bash
```

That installs Go, creates the `aistbot` service account, clones the repo to
`/opt/aist-bot`, installs the systemd unit, and allows the service account
to restart that one service without a password.

**If the repository is private**, the server cannot clone over HTTPS. Add a
read-only deploy key on GitHub (Settings → Deploy keys) for a key generated
on the server, and run the script against the SSH URL instead:

```bash
REPO=git@github.com:Abdurayim/aist-bot.git sudo -E bash deploy/setup-server.sh
```

Then write the configuration — this file is **not** in git and never will be:

```bash
sudo -u aistbot nano /opt/aist-bot/.env
```

```ini
BOT_TOKEN=123456:AA...
ADMIN_PHONES=+998901112233,+998935556677,+998977778899
DB_PATH=/opt/aist-bot/aist_bot.db
ORG_NAME=AIST School
```

Start it:

```bash
sudo -u aistbot /opt/aist-bot/deploy/deploy.sh
journalctl -u aist-bot -f
```

## 2. Give GitHub a key to the server

Create a key pair **for deployment only**, on your own machine:

```bash
ssh-keygen -t ed25519 -C "github-actions-aist-bot" -f ~/.ssh/aist_deploy -N ""
```

Put the public half on the server:

```bash
ssh-copy-id -i ~/.ssh/aist_deploy.pub aistbot@YOUR_SERVER_IP
```

Then add the repository secrets at
**Settings → Secrets and variables → Actions → New repository secret**:

| Secret | Value | Required |
|---|---|---|
| `SSH_HOST` | the server's IP or hostname | yes |
| `SSH_USER` | `aistbot` | yes |
| `SSH_KEY` | the **private** key: `cat ~/.ssh/aist_deploy` — all of it, including the BEGIN and END lines | yes |
| `SSH_PORT` | only if SSH is not on 22 | no |
| `DEPLOY_PATH` | only if the app is not at `/opt/aist-bot` | no |
| `SSH_KNOWN_HOSTS` | output of `ssh-keyscan YOUR_SERVER_IP` | recommended |

`SSH_KNOWN_HOSTS` pins the server's identity. Without it the workflow
accepts whatever host key answers, which is fine on a trusted network but
leaves the deploy open to being redirected.

## 3. Deploy

```bash
git push origin main
```

Watch it under the **Actions** tab. Tests run first and a failing test
blocks the deploy. You can also redeploy the current `main` without pushing:
**Actions → Deploy → Run workflow**.

---

## Surviving a server under pressure

The bot shares the server with whatever else you run, and the usual way a
small VPS dies is that one project exhausts the RAM, the kernel OOM killer
picks a victim badly, and you cannot even SSH in to fix it.

```bash
sudo /opt/aist-bot/deploy/setup-resilience.sh
```

Safe to re-run. It sets up four things:

| | |
|---|---|
| **Swap** | A swap file the size of RAM (2–8 GB), in `/etc/fstab` so it survives reboots. Without it the kernel's only response to pressure is to kill something. |
| **`vm.swappiness=10`** | The default of 60 is a desktop setting and pushes idle server processes to disk for no reason. |
| **earlyoom** | Kills the actual memory hog *early*, while the box still responds, instead of letting it freeze. Configured to avoid `sshd` and the bot, and to prefer the usual culprits (`node`, `java`, `python`, databases). |
| **OOM scores** | `sshd` at `-900` and the bot at `-500`. Killing either frees almost nothing, and killing `sshd` locks you out of your own server. |

### How the bot itself survives

| Failure | What happens |
|---|---|
| Bot crashes | `Restart=always`, back in 5s |
| Server reboots | `systemctl enable` — starts at boot |
| Telegram unreachable at boot | Six retries with backoff before the process gives up |
| Repeated failed starts | `StartLimitIntervalSec=0` — systemd never stops trying |
| Killed by the OOM killer | `OOMPolicy=stop` makes it a failure, so `Restart=always` applies |
| **Process alive but wedged** | `Type=notify` + `WatchdogSec=120`. The bot reports every 60s; miss two and systemd kills and restarts it. |

That last one is the case `Restart=always` cannot see on its own: a
deadlocked process looks perfectly healthy from the outside.

Check the watchdog is actually running:

```bash
systemctl show aist-bot -p WatchdogUSec -p Type -p NRestarts
journalctl -u aist-bot | grep watchdog
```

## Lock the server down

A VPS usually arrives with root password login enabled, and the provider
often sets the control-panel password and the root password to the same
value. Fix both before the bot carries real complaints:

```bash
# 1. Give the service account the deploy key, then confirm it works
ssh -i ~/.ssh/aist_deploy aistbot@YOUR_SERVER_IP 'echo ok'

# 2. Only once that prints ok — turn off password logins
sudo sed -i 's/^#*PermitRootLogin.*/PermitRootLogin prohibit-password/' /etc/ssh/sshd_config
sudo sed -i 's/^#*PasswordAuthentication.*/PasswordAuthentication no/' /etc/ssh/sshd_config
sudo systemctl restart ssh

# 3. Change the root password to something not shared with the panel
sudo passwd root
```

Keep a second terminal open while doing this, so a mistake in the SSH
config does not lock you out.

## Day-to-day

| | |
|---|---|
| Follow the log | `journalctl -u aist-bot -f` |
| Is it running? | `systemctl status aist-bot` |
| Restart it | `sudo systemctl restart aist-bot` |
| Deploy by hand | `cd /opt/aist-bot && ./deploy/deploy.sh` |
| Change who is an admin | edit `.env`, then restart |
| Back up everything that matters | `cp /opt/aist-bot/aist_bot.db ~/backup-$(date +%F).db` |

### Changing the admins

Edit `ADMIN_PHONES` in `/opt/aist-bot/.env` and restart. The bot syncs the
list at startup: numbers you add become admins, numbers you remove stop
being admins. Each person's menu switches the next time they use the bot,
and they are told their role changed.

An admin who has never opened the bot cannot be sent anything — the startup
log says so by name:

```
⚠ 1 admin(s) have not opened the bot yet and will NOT receive appeals:
    +998977778899 — ask them to send /start and share this number
```

### What is not in git

`.env` and `aist_bot.db` are untracked, and the deploy script's
`git reset --hard` leaves untracked files alone. Your token, your admin
numbers and every appeal ever filed stay on the server only.

**Back up `aist_bot.db`.** It is the only copy of the appeals, and nothing
in this pipeline backs it up for you. A daily copy is enough:

```bash
sudo crontab -e
# 0 3 * * * sqlite3 /opt/aist-bot/aist_bot.db ".backup /var/backups/aist-$(date +\%F).db"
```

## Backups

The database is the only copy of every complaint, proposal and thank-you
note the school receives. The PDFs live in Telegram's cloud and are
referenced by id, so a lost database cannot be rebuilt from them.

A timer takes a backup every night at 03:30 and `setup-server.sh` enables
it, so there is nothing to set up by hand.

```bash
systemctl list-timers aist-bot-backup.timer   # when it next runs
journalctl -u aist-bot-backup                 # what it did
sudo systemctl start aist-bot-backup          # take one right now
./deploy/restore.sh --list                    # what you have
```

Backups go to `/var/backups/aist-bot/`, gzipped, `600`, named by date.
Kept **daily for 14 days** and **first-of-month for a year**.

Two things make this more than a copy:

- The bot keeps SQLite in WAL mode while running, so a plain `cp` can catch
  a half-written transaction. `.backup` takes a consistent snapshot of a
  live database.
- Every backup is opened and integrity-checked, and its rows counted,
  **before** any older backup is pruned. A snapshot that fails is deleted
  and nothing else is touched.

### Restoring

```bash
sudo ./deploy/restore.sh                    # the newest backup
sudo ./deploy/restore.sh /var/backups/aist-bot/aist_bot-2026-10-20_0330.db.gz
```

It checks the backup before touching anything, shows you what is being
replaced against what is coming in, warns if the backup holds *fewer*
appeals than the live database, and saves the database it replaces next to
the original. Then it stops the bot, swaps the file, and starts it again.

**Try a restore once, deliberately, while nothing is wrong.** A backup
nobody has restored is a guess.

### Getting backups off the server

Everything above still lives on one disk. If the VPS is lost, so are the
backups. Set `BACKUP_REMOTE` in `/opt/aist-bot/.env` to an rsync
destination and each backup is copied there as well:

```ini
BACKUP_REMOTE=you@your-mac-or-other-host:/path/to/backups/
```

That needs an SSH key the `aistbot` user can use without a password. Or
pull them to your own machine on a schedule:

```bash
rsync -az aistbot@45.130.148.39:/var/backups/aist-bot/ ~/aist-backups/
```

### If a deploy fails

The script rolls back by itself and the Actions run goes red. To see why:

```bash
journalctl -u aist-bot --lines=50
```

The usual causes are a malformed `.env`, a revoked bot token, or two copies
of the bot polling the same token — Telegram allows only one, and the log
says `Conflict: terminated by other getUpdates request`.
