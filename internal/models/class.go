package models

import "time"

// Class is a school class created by an admin. Parents pick one of these
// when submitting an appeal.
type Class struct {
	ID        int64     `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	IsActive  bool      `json:"is_active" db:"is_active"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}
