# AIST Bot

A Telegram bot for a school, in **Uzbek, Russian and English**, that turns a
parent's message into a properly formatted PDF and delivers it to the
administration.

Parents can send three kinds of message:

| | | |
|---|---|---|
| 📝 | **Shikoyat** | a complaint |
| 💡 | **Taklif** | a proposal |
| 🙏 | **Minnatdorchilik** | a thank-you note |

Each one is addressed to a **topic department**, tagged with a **class** and a
**child's name**, and may carry **up to five optional photos**.

---

## What a parent does

The whole thing is five questions, each a single tap or one line of typing.
Every screen has a **⬅️ Back** and a **❌ Cancel** button, so nothing is a
dead end.

```
/start
  → choose language        🇺🇿 🇷🇺 🇬🇧   (one tap)
  → share phone number     📱            (one tap, Telegram's own button)

📝 Shikoyat
  1. Bo'lim        which department?     (one tap)
  2. Sinf          which class?          (one tap)
  3. Farzand       child's name          (type)
  4. Matn          the message itself    (type)
  5. Rasm          photo? optional       (one tap to skip)
  → check the summary → ✅ Yuborish
```

The parent then receives their own copy of the PDF with a document number
such as `AIST-2026-000042`, and every connected admin receives the same
document with a summary caption they can triage from without opening it.

## What the PDF looks like

A4, one accent colour per message type — red for complaints, blue for
proposals, green for thank-you notes:

- a header band with the school name and the type as a filled badge
- the document number and timestamp
- a details table: type, department, class, child, **phone number**,
  **Telegram username**, **date and time**
- the parent's own words in a quoted panel
- each attached photo, framed and captioned, two to a page

It is written **entirely in the language the parent chose**, so a Russian
speaker's complaint arrives as a Russian document.

---

## Setup

### 1. Configure

Copy `.env.example` to `.env` and fill it in:

```bash
cp .env.example .env
```

```ini
BOT_TOKEN=123456:AA...              # from @BotFather
ADMIN_PHONES=+998901112233,+998935556677,+998977778899,+998901234567
DB_PATH=./aist_bot.db
ORG_NAME=AIST School
```

**About `ADMIN_PHONES`** — list every number that should receive appeals,
separated by commas. There is no limit on how many: three, four or more all
work. One person may list several of their own numbers, and they will still
receive exactly one copy of each appeal.

These formats are all accepted and normalised:

```
+998901112233      +998 90 111 22 33      +998-90-111-22-33
998901112233       901112233
```

Editing `ADMIN_PHONES` and restarting is the whole admin workflow: numbers
you add become admins, numbers you remove stop being admins.

### 2. Run

```bash
go run ./cmd/bot
```

On start the bot prints which admins are connected, and warns loudly about
any who are not:

```
✓ database ready at ./aist_bot.db
✓ connected as @aist_school_bot
✓ 4 admin phone number(s) configured
⚠ 1 admin(s) have not opened the bot yet and will NOT receive appeals:
    +998977778899 — ask them to send /start and share this number
```

### 3. Each admin opens the bot once

An admin must send `/start` and share the contact matching one of the numbers
in `ADMIN_PHONES`. That is what links their chat to the bot — **until they do,
they receive nothing.** (If they had already registered as a parent before
their number was listed, restarting the bot connects them automatically.)

**Admins and parents see different menus.** The administration *receives*
appeals, so an admin's home screen is the management menu — there are no
Shikoyat / Taklif / Minnatdorchilik buttons on it, and an admin cannot file
an appeal even from an old button left in the chat:

```
Parent menu              Admin menu
📝 Shikoyat              📂 Bo'limlar
💡 Taklif                🏫 Sinflar
🙏 Minnatdorchilik       📬 Oxirgi murojaatlar
📂 Mening murojaatlarim  📊 Statistika
ℹ️ Yordam ⚙️ Sozlamalar  ⚙️ Sozlamalar
```

Exactly one of these is on screen at any time.

**Changing someone's role takes effect cleanly.** Add or remove a number in
`ADMIN_PHONES` and restart: the next thing that person does, the bot tells
them their role changed and hands them the new menu. If they were halfway
through a form that their new role may not send, it is discarded rather than
left armed. The role shown is always derived from `ADMIN_PHONES` — the
`users.role` column only records what they were last told, so the change can
be noticed.

### 4. Add the classes

Parents cannot send anything until at least one class exists.

> 🛠 Admin panel → 🏫 Sinflar → ➕ Sinf qo'shish → type `5-A`

The nine departments the school started with are already there.

---

## The admin panel

The admin menu is the panel — there is no separate screen to open.

```
📂 Bo'limlar          departments: add, rename, hide, delete
🏫 Sinflar            classes: add, hide, delete
📬 Oxirgi murojaatlar the latest appeals, tap to get the PDF again
📊 Statistika         totals per type
⚙️ Sozlamalar         interface language
```

### Managing departments

Departments are **fully editable** — the nine below are only a starting point.

| Seeded department | | |
|---|---|---|
| 🍽 | Oshxona | Canteen |
| 👩‍🏫 | Mentorlar | Mentors |
| 📚 | Akademik yo'nalish | Academics |
| 📖 | Kutubxona (kitob) | Library |
| 💰 | Moliya (to'lovlar) | Finance |
| 🏆 | Tadbirlar va olimpiadalar | Events and olympiads |
| 🔬 | Fan va sport to'garaklari | Science and sports clubs |
| 🛡 | Security va xavfsizlik | Security (Face ID, turnstile) |
| 🩺 | Tibbiy va psixologik | Medical and psychological |

Adding or renaming one asks for the name in each language in turn; Russian
and English can be skipped with **⏭ O'tkazib yuborish**, in which case the
Uzbek name is shown everywhere. Start the Uzbek name with an emoji
(`🚌 Transport`) and it becomes the button's icon automatically.

**Hide** (🚫) takes a department out of the parents' list while keeping it.
**Delete** (🗑) removes it for good. Neither one damages past appeals: every
appeal stores its own copy of the department name at the moment it was sent,
so an old complaint still reads `Oshxona` long after that department was
renamed or deleted.

---

## Deployment

**[DEPLOY.md](DEPLOY.md) has the full setup**: a one-command server script,
a systemd service, and a GitHub Actions workflow that makes a push to `main`
deploy itself — the server pulls from GitHub, rebuilds, restarts, and rolls
back on its own if the new build will not start.

Polling mode (the default) needs no public address, no domain and no TLS
certificate — it works from any machine with internet access. Leave
`WEBHOOK_URL` empty.

For webhook mode, set `WEBHOOK_URL` to your public HTTPS address and the bot
serves `POST /webhook` and `GET /health` on `SERVER_PORT`.

To build a single deployable binary:

```bash
go build -o aist-bot ./cmd/bot
```

The fonts and the database schema are embedded in that binary, so it can be
copied anywhere and run with only a `.env` beside it.

---

## Project layout

```
.github/workflows/       CI on every branch, deploy on main
deploy/                  systemd unit, server setup and deploy scripts
cmd/bot/                 entry point, polling and webhook modes
internal/
  config/                .env parsing, admin phone normalisation
  database/              SQLite connection + embedded migrations
  models/                domain types
  repository/            all SQL
  state/                 where each chat is in a conversation
  i18n/                  the three message catalogues
  utils/                 keyboards, callback payloads, formatting
  validator/             phone, name and text rules
  services/              Telegram calls, PDF rendering, submission
  handlers/              the conversation itself
```

### Notes on a few decisions

**Nothing personal is written to disk.** The PDF is built in memory and
handed straight to Telegram; photos are downloaded into memory and never
stored. The database keeps the text of appeals and Telegram file ids, not
files.

**An appeal is saved before it is rendered.** If the renderer or Telegram
fails at the wrong moment, the appeal is still stored and is delivered to
admins as text instead — what a parent wrote is never lost silently.

**Updates from one person are handled one at a time.** Each update runs on
its own goroutine so a slow PDF never blocks other parents, but two updates
from the same chat would otherwise race on the conversation state — a double
tap on Send filed the appeal twice, and a photo album arrived as several
updates that overwrote each other. A per-person lock serialises them while
leaving different people fully parallel.

**Old keyboards keep working.** Telegram leaves earlier keyboards in the chat
history, so every button is matched against all three languages. A parent who
switched to Russian can still tap an Uzbek button from yesterday. For the same
reason, role rules are enforced in the handlers and not only by hiding a
button: a stale appeal button cannot be used by an admin.

## Tests

```bash
go test ./...
```

The suite runs the whole bot against a stub Telegram server: registration,
promotion and demotion (including mid-form), concurrency under `-race`,
all three appeal types in all three languages, back and cancel at every step,
validation limits, the photo cap, the admin panel, the parent/admin menu
split, four admins receiving one appeal each, one admin with two numbers
receiving a single copy, and the document-failure fallback. The PDF renderer is tested separately across every
type, language, and page-break path.

To eyeball the generated documents:

```bash
AIST_PDF_SAMPLES=/tmp/pdfs go test ./internal/services -run TestWriteSamples
```
