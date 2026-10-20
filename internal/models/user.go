package models

import "time"

// Roles a user can hold. The authority is always ADMIN_PHONES; these are
// what the bot last told the person they were.
const (
	RoleParent = "parent"
	RoleAdmin  = "admin"
)

// RoleFor maps the admin check onto a role name.
func RoleFor(isAdmin bool) string {
	if isAdmin {
		return RoleAdmin
	}
	return RoleParent
}

// User is a registered parent (or admin) of the bot.
type User struct {
	ID               int64     `json:"id" db:"id"`
	TelegramID       int64     `json:"telegram_id" db:"telegram_id"`
	TelegramUsername string    `json:"telegram_username" db:"telegram_username"`
	PhoneNumber      string    `json:"phone_number" db:"phone_number"`
	Language         string    `json:"language" db:"language"`
	Role             string    `json:"role" db:"role"`
	RegisteredAt     time.Time `json:"registered_at" db:"registered_at"`
}

// UsernameOrDash returns the @username, or a dash when the account has none.
func (u *User) UsernameOrDash() string {
	if u.TelegramUsername == "" {
		return "—"
	}
	return "@" + u.TelegramUsername
}
