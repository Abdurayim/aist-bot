package models

import "time"

// Admin is a staff member listed in ADMIN_PHONES. TelegramID is filled in
// the moment the admin shares their contact with the bot, which is what
// makes it possible to forward appeals to them.
type Admin struct {
	ID          int64     `json:"id" db:"id"`
	PhoneNumber string    `json:"phone_number" db:"phone_number"`
	TelegramID  *int64    `json:"telegram_id" db:"telegram_id"`
	AddedAt     time.Time `json:"added_at" db:"added_at"`
}

// IsLinked reports whether this admin can already receive notifications.
func (a *Admin) IsLinked() bool {
	return a.TelegramID != nil && *a.TelegramID != 0
}
