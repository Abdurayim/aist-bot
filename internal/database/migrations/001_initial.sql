-- ─────────────────────────────────────────────────────────────────────
--  AIST BOT — initial schema (SQLite)
--  Appeals: complaints, proposals and gratitude notes, rendered to PDF.
-- ─────────────────────────────────────────────────────────────────────

-- Parents (and admins, who are parents with an admin phone number).
CREATE TABLE IF NOT EXISTS users (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    telegram_id       INTEGER UNIQUE NOT NULL,
    telegram_username TEXT    NOT NULL DEFAULT '',
    phone_number      TEXT    NOT NULL,
    language          TEXT    NOT NULL DEFAULT 'uz' CHECK (language IN ('uz', 'ru', 'en')),
    registered_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_users_phone ON users(phone_number);
CREATE INDEX IF NOT EXISTS idx_users_registered ON users(registered_at DESC);

-- Classes, created and maintained by admins.
CREATE TABLE IF NOT EXISTS classes (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT UNIQUE NOT NULL,
    is_active  INTEGER NOT NULL DEFAULT 1 CHECK (is_active IN (0, 1)),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_classes_active ON classes(is_active, name);

-- Topic departments, created and maintained by admins. Seeded below with
-- the nine areas the school started with; admins may rename, hide, delete
-- and add to this list from the admin panel.
CREATE TABLE IF NOT EXISTS departments (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    code       TEXT UNIQUE NOT NULL,
    emoji      TEXT NOT NULL DEFAULT '',
    name_uz    TEXT NOT NULL,
    name_ru    TEXT NOT NULL DEFAULT '',
    name_en    TEXT NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0,
    is_active  INTEGER NOT NULL DEFAULT 1 CHECK (is_active IN (0, 1)),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_departments_active ON departments(is_active, sort_order);

-- Appeals. department_name and class_name are snapshots taken when the
-- appeal was sent, so renaming or deleting a department later never
-- rewrites history.
CREATE TABLE IF NOT EXISTS submissions (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    reference       TEXT UNIQUE NOT NULL DEFAULT '',
    user_id         INTEGER NOT NULL,
    type            TEXT NOT NULL CHECK (type IN ('complaint', 'proposal', 'gratitude')),
    department_code TEXT NOT NULL,
    department_name TEXT NOT NULL DEFAULT '',
    class_name      TEXT NOT NULL,
    child_name      TEXT NOT NULL,
    text            TEXT NOT NULL,
    photo_count     INTEGER NOT NULL DEFAULT 0,
    pdf_file_id     TEXT NOT NULL DEFAULT '',
    pdf_filename    TEXT NOT NULL DEFAULT '',
    language        TEXT NOT NULL DEFAULT 'uz',
    status          TEXT NOT NULL DEFAULT 'new' CHECK (status IN ('new', 'reviewed', 'archived')),
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_submissions_user ON submissions(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_submissions_created ON submissions(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_submissions_type ON submissions(type);
CREATE INDEX IF NOT EXISTS idx_submissions_department ON submissions(department_code);

-- Photos attached to an appeal, kept in Telegram's cloud by file_id.
CREATE TABLE IF NOT EXISTS submission_photos (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    submission_id  INTEGER NOT NULL,
    file_id        TEXT NOT NULL,
    file_unique_id TEXT NOT NULL DEFAULT '',
    file_size      INTEGER NOT NULL DEFAULT 0,
    order_index    INTEGER NOT NULL DEFAULT 0,
    FOREIGN KEY (submission_id) REFERENCES submissions(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_photos_submission ON submission_photos(submission_id, order_index);

-- Admins listed in ADMIN_PHONES. One person may own several rows, one per
-- phone number; telegram_id is filled once they share their contact.
CREATE TABLE IF NOT EXISTS admins (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    phone_number TEXT UNIQUE NOT NULL,
    telegram_id  INTEGER,
    added_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_admins_phone ON admins(phone_number);
CREATE INDEX IF NOT EXISTS idx_admins_telegram ON admins(telegram_id);

-- Conversation state, one row per chat.
CREATE TABLE IF NOT EXISTS user_states (
    telegram_id INTEGER PRIMARY KEY,
    state       TEXT NOT NULL,
    data        TEXT,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_user_states_updated ON user_states(updated_at);

-- ── Seed: the nine departments the school started with ───────────────
INSERT OR IGNORE INTO departments (code, emoji, name_uz, name_ru, name_en, sort_order) VALUES
    ('kitchen',  '🍽',   'Oshxona',                                    'Столовая',                             'Canteen',                       1),
    ('mentors',  '👩‍🏫', 'Mentorlar',                                  'Менторы',                              'Mentors',                       2),
    ('academic', '📚',   'Akademik yo''nalish',                        'Академическое направление',            'Academics',                     3),
    ('library',  '📖',   'Kutubxona (kitob)',                          'Библиотека (книги)',                   'Library (books)',               4),
    ('finance',  '💰',   'Moliya (to''lovlar)',                        'Финансы (оплаты)',                     'Finance (payments)',            5),
    ('events',   '🏆',   'Tadbirlar va olimpiadalar',                  'Мероприятия и олимпиады',              'Events and olympiads',          6),
    ('clubs',    '🔬',   'Fan va sport to''garaklari',                 'Научные и спортивные кружки',          'Science and sports clubs',      7),
    ('security', '🛡',   'Security va xavfsizlik (Face ID, turniket)', 'Служба безопасности (Face ID, турникет)', 'Security (Face ID, turnstile)', 8),
    ('medical',  '🩺',   'Tibbiy va psixologik',                       'Медицинская и психологическая служба', 'Medical and psychological',     9);
