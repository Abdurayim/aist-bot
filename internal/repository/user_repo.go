package repository

import (
	"database/sql"
	"fmt"

	"aist-bot/internal/models"
)

type UserRepository struct{ db *sql.DB }

func NewUserRepository(db *sql.DB) *UserRepository { return &UserRepository{db: db} }

// GetByTelegramID returns the user, or nil when they are not registered yet.
func (r *UserRepository) GetByTelegramID(telegramID int64) (*models.User, error) {
	const query = `
		SELECT id, telegram_id, telegram_username, phone_number, language, role, registered_at
		FROM users WHERE telegram_id = ?`

	var u models.User
	err := r.db.QueryRow(query, telegramID).Scan(
		&u.ID, &u.TelegramID, &u.TelegramUsername, &u.PhoneNumber, &u.Language, &u.Role, &u.RegisteredAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &u, nil
}

// GetByPhone returns the user registered with this number, or nil. Phone
// numbers are unique per person in practice but not enforced as unique in
// the schema, so the newest registration wins.
func (r *UserRepository) GetByPhone(phone string) (*models.User, error) {
	const query = `
		SELECT id, telegram_id, telegram_username, phone_number, language, role, registered_at
		FROM users WHERE phone_number = ? ORDER BY registered_at DESC LIMIT 1`

	var u models.User
	err := r.db.QueryRow(query, phone).Scan(
		&u.ID, &u.TelegramID, &u.TelegramUsername, &u.PhoneNumber, &u.Language, &u.Role, &u.RegisteredAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by phone: %w", err)
	}

	return &u, nil
}

// Upsert creates the user, or updates the mutable fields if they already exist.
func (r *UserRepository) Upsert(telegramID int64, username, phone, language string) (*models.User, error) {
	const query = `
		INSERT INTO users (telegram_id, telegram_username, phone_number, language)
		VALUES (?, ?, ?, ?)
		ON CONFLICT (telegram_id) DO UPDATE SET
			telegram_username = excluded.telegram_username,
			phone_number      = excluded.phone_number,
			language          = excluded.language`

	if _, err := r.db.Exec(query, telegramID, username, phone, language); err != nil {
		return nil, fmt.Errorf("failed to save user: %w", err)
	}

	return r.GetByTelegramID(telegramID)
}

// UpdateLanguage changes the interface language of a registered user.
func (r *UserRepository) UpdateLanguage(telegramID int64, language string) error {
	_, err := r.db.Exec(`UPDATE users SET language = ? WHERE telegram_id = ?`, language, telegramID)
	if err != nil {
		return fmt.Errorf("failed to update language: %w", err)
	}
	return nil
}

// UpdateUsername keeps the stored @username in sync, since people change it.
func (r *UserRepository) UpdateUsername(telegramID int64, username string) error {
	_, err := r.db.Exec(`UPDATE users SET telegram_username = ? WHERE telegram_id = ?`, username, telegramID)
	return err
}

// UpdateRole records the role the user was last shown, so a change can be
// noticed on their next message.
func (r *UserRepository) UpdateRole(telegramID int64, role string) error {
	_, err := r.db.Exec(`UPDATE users SET role = ? WHERE telegram_id = ?`, role, telegramID)
	if err != nil {
		return fmt.Errorf("failed to update role: %w", err)
	}
	return nil
}

// Count returns the number of registered users.
func (r *UserRepository) Count() (int, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count)
	return count, err
}
