package repository

import (
	"database/sql"
	"fmt"

	"aist-bot/internal/models"
)

type ClassRepository struct{ db *sql.DB }

func NewClassRepository(db *sql.DB) *ClassRepository { return &ClassRepository{db: db} }

// GetActive returns the classes parents can choose from.
func (r *ClassRepository) GetActive() ([]models.Class, error) {
	return r.query(`SELECT id, name, is_active, created_at FROM classes WHERE is_active = 1 ORDER BY name COLLATE NOCASE`)
}

// GetAll returns every class, including hidden ones, for the admin panel.
func (r *ClassRepository) GetAll() ([]models.Class, error) {
	return r.query(`SELECT id, name, is_active, created_at FROM classes ORDER BY is_active DESC, name COLLATE NOCASE`)
}

func (r *ClassRepository) query(sqlStr string) ([]models.Class, error) {
	rows, err := r.db.Query(sqlStr)
	if err != nil {
		return nil, fmt.Errorf("failed to list classes: %w", err)
	}
	defer rows.Close()

	classes := make([]models.Class, 0)
	for rows.Next() {
		var c models.Class
		if err := rows.Scan(&c.ID, &c.Name, &c.IsActive, &c.CreatedAt); err != nil {
			return nil, err
		}
		classes = append(classes, c)
	}

	return classes, rows.Err()
}

// GetByID returns one class, or nil when it no longer exists.
func (r *ClassRepository) GetByID(id int64) (*models.Class, error) {
	var c models.Class
	err := r.db.QueryRow(`SELECT id, name, is_active, created_at FROM classes WHERE id = ?`, id).
		Scan(&c.ID, &c.Name, &c.IsActive, &c.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &c, nil
}

// Create adds a class. The bool reports false when the name already exists.
func (r *ClassRepository) Create(name string) (bool, error) {
	res, err := r.db.Exec(`INSERT OR IGNORE INTO classes (name) VALUES (?)`, name)
	if err != nil {
		return false, fmt.Errorf("failed to create class: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}

	return affected > 0, nil
}

// ToggleActive flips the visibility of a class and returns the new state.
func (r *ClassRepository) ToggleActive(id int64) error {
	_, err := r.db.Exec(`UPDATE classes SET is_active = 1 - is_active WHERE id = ?`, id)
	return err
}

// Delete removes a class. Past appeals keep their class name snapshot.
func (r *ClassRepository) Delete(id int64) error {
	_, err := r.db.Exec(`DELETE FROM classes WHERE id = ?`, id)
	return err
}

func (r *ClassRepository) Count() (int, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM classes`).Scan(&count)
	return count, err
}
