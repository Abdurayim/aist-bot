package repository

import (
	"database/sql"
	"fmt"

	"aist-bot/internal/models"
)

type AdminRepository struct{ db *sql.DB }

func NewAdminRepository(db *sql.DB) *AdminRepository { return &AdminRepository{db: db} }

// SyncFromConfig makes the admins table match ADMIN_PHONES: new numbers are
// added and numbers no longer listed are removed, so editing .env and
// restarting is all it takes to change who is an admin.
func (r *AdminRepository) SyncFromConfig(phones []string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	keep := make(map[string]bool, len(phones))
	for _, phone := range phones {
		keep[phone] = true

		if _, err := tx.Exec(`INSERT OR IGNORE INTO admins (phone_number) VALUES (?)`, phone); err != nil {
			return fmt.Errorf("failed to add admin %s: %w", phone, err)
		}
	}

	rows, err := tx.Query(`SELECT phone_number FROM admins`)
	if err != nil {
		return err
	}

	var stale []string
	for rows.Next() {
		var phone string
		if err := rows.Scan(&phone); err != nil {
			rows.Close()
			return err
		}
		if !keep[phone] {
			stale = append(stale, phone)
		}
	}
	rows.Close()

	for _, phone := range stale {
		if _, err := tx.Exec(`DELETE FROM admins WHERE phone_number = ?`, phone); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetAll returns every admin row.
func (r *AdminRepository) GetAll() ([]models.Admin, error) {
	rows, err := r.db.Query(`SELECT id, phone_number, telegram_id, added_at FROM admins ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("failed to list admins: %w", err)
	}
	defer rows.Close()

	admins := make([]models.Admin, 0)
	for rows.Next() {
		var a models.Admin
		if err := rows.Scan(&a.ID, &a.PhoneNumber, &a.TelegramID, &a.AddedAt); err != nil {
			return nil, err
		}
		admins = append(admins, a)
	}

	return admins, rows.Err()
}

// GetLinkedTelegramIDs returns the chat ids of admins who have shared their
// contact, deduplicated — one person may own several admin phone numbers and
// must still receive exactly one copy of each appeal.
func (r *AdminRepository) GetLinkedTelegramIDs() ([]int64, error) {
	rows, err := r.db.Query(`SELECT DISTINCT telegram_id FROM admins WHERE telegram_id IS NOT NULL`)
	if err != nil {
		return nil, fmt.Errorf("failed to list admin chats: %w", err)
	}
	defer rows.Close()

	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	return ids, rows.Err()
}

// LinkTelegramID attaches a chat id to an admin phone number.
//
// This is called on every update from an admin, so it first checks whether
// anything would actually change and writes nothing when the link is
// already in place.
func (r *AdminRepository) LinkTelegramID(phone string, telegramID int64) error {
	var linked sql.NullInt64

	err := r.db.QueryRow(`SELECT telegram_id FROM admins WHERE phone_number = ?`, phone).Scan(&linked)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}

	if linked.Valid && linked.Int64 == telegramID {
		return nil
	}

	// A chat id belongs to one person, so clear it from any other row: an
	// admin may have re-registered with a different one of their numbers.
	if _, err := r.db.Exec(
		`UPDATE admins SET telegram_id = NULL WHERE telegram_id = ? AND phone_number != ?`,
		telegramID, phone,
	); err != nil {
		return err
	}

	_, err = r.db.Exec(`UPDATE admins SET telegram_id = ? WHERE phone_number = ?`, telegramID, phone)
	return err
}

// UnlinkTelegramID releases any admin seat still attached to this chat.
// Returns whether anything was actually released.
func (r *AdminRepository) UnlinkTelegramID(telegramID int64) (bool, error) {
	res, err := r.db.Exec(`UPDATE admins SET telegram_id = NULL WHERE telegram_id = ?`, telegramID)
	if err != nil {
		return false, fmt.Errorf("failed to unlink admin chat: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}

	return affected > 0, nil
}

// IsAdmin reports whether the phone number or chat id belongs to an admin.
func (r *AdminRepository) IsAdmin(phone string, telegramID int64) (bool, error) {
	var count int
	err := r.db.QueryRow(
		`SELECT COUNT(*) FROM admins WHERE (phone_number = ? AND ? != '') OR (telegram_id = ? AND ? != 0)`,
		phone, phone, telegramID, telegramID,
	).Scan(&count)

	if err == sql.ErrNoRows {
		return false, nil
	}

	return count > 0, err
}
