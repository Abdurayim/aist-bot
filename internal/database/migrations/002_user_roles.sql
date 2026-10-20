-- Remember which role each user was last seen with.
--
-- The role itself is always derived from ADMIN_PHONES, never from this
-- column: this is only a record of what the person was last *told* they
-- were, so the bot can notice a promotion or demotion between two messages
-- and swap their menu instead of leaving a stale one on their screen.
ALTER TABLE users ADD COLUMN role TEXT NOT NULL DEFAULT 'parent';

UPDATE users SET role = 'parent' WHERE role IS NULL OR role = '';
