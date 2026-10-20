package repository

import (
	"database/sql"
	"fmt"
	"strings"

	"aist-bot/internal/models"
)

type DepartmentRepository struct{ db *sql.DB }

func NewDepartmentRepository(db *sql.DB) *DepartmentRepository {
	return &DepartmentRepository{db: db}
}

const departmentColumns = `id, code, emoji, name_uz, name_ru, name_en, sort_order, is_active, created_at`

// GetActive returns the departments parents can choose from, in admin order.
func (r *DepartmentRepository) GetActive() ([]models.Department, error) {
	return r.query(`SELECT ` + departmentColumns + ` FROM departments WHERE is_active = 1 ORDER BY sort_order, id`)
}

// GetAll returns every department, hidden ones included, for the admin panel.
func (r *DepartmentRepository) GetAll() ([]models.Department, error) {
	return r.query(`SELECT ` + departmentColumns + ` FROM departments ORDER BY sort_order, id`)
}

func (r *DepartmentRepository) query(sqlStr string) ([]models.Department, error) {
	rows, err := r.db.Query(sqlStr)
	if err != nil {
		return nil, fmt.Errorf("failed to list departments: %w", err)
	}
	defer rows.Close()

	departments := make([]models.Department, 0)
	for rows.Next() {
		var d models.Department
		if err := rows.Scan(&d.ID, &d.Code, &d.Emoji, &d.NameUz, &d.NameRu, &d.NameEn,
			&d.SortOrder, &d.IsActive, &d.CreatedAt); err != nil {
			return nil, err
		}
		departments = append(departments, d)
	}

	return departments, rows.Err()
}

// GetByID returns one department, or nil when it has been deleted.
func (r *DepartmentRepository) GetByID(id int64) (*models.Department, error) {
	var d models.Department
	err := r.db.QueryRow(`SELECT `+departmentColumns+` FROM departments WHERE id = ?`, id).
		Scan(&d.ID, &d.Code, &d.Emoji, &d.NameUz, &d.NameRu, &d.NameEn,
			&d.SortOrder, &d.IsActive, &d.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &d, nil
}

// GetByCode returns one department by its stable code.
func (r *DepartmentRepository) GetByCode(code string) (*models.Department, error) {
	var d models.Department
	err := r.db.QueryRow(`SELECT `+departmentColumns+` FROM departments WHERE code = ?`, code).
		Scan(&d.ID, &d.Code, &d.Emoji, &d.NameUz, &d.NameRu, &d.NameEn,
			&d.SortOrder, &d.IsActive, &d.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &d, nil
}

// ExistsByNameUz reports whether another department already uses this Uzbek
// name. excludeID lets an edit keep its own name.
func (r *DepartmentRepository) ExistsByNameUz(nameUz string, excludeID int64) (bool, error) {
	var count int
	err := r.db.QueryRow(
		`SELECT COUNT(*) FROM departments WHERE LOWER(name_uz) = LOWER(?) AND id != ?`,
		strings.TrimSpace(nameUz), excludeID,
	).Scan(&count)

	return count > 0, err
}

// Create adds a department and returns it. The code is derived from the
// Uzbek name and made unique, because codes are what appeals store.
func (r *DepartmentRepository) Create(emoji, nameUz, nameRu, nameEn string) (*models.Department, error) {
	code, err := r.uniqueCode(nameUz)
	if err != nil {
		return nil, err
	}

	var nextOrder int
	if err := r.db.QueryRow(`SELECT COALESCE(MAX(sort_order), 0) + 1 FROM departments`).Scan(&nextOrder); err != nil {
		return nil, err
	}

	res, err := r.db.Exec(
		`INSERT INTO departments (code, emoji, name_uz, name_ru, name_en, sort_order) VALUES (?, ?, ?, ?, ?, ?)`,
		code, emoji, nameUz, nameRu, nameEn, nextOrder,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create department: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}

	return r.GetByID(id)
}

// Update rewrites the names and emoji of a department. The code is left
// untouched so existing appeals keep pointing at it.
func (r *DepartmentRepository) Update(id int64, emoji, nameUz, nameRu, nameEn string) error {
	_, err := r.db.Exec(
		`UPDATE departments SET emoji = ?, name_uz = ?, name_ru = ?, name_en = ? WHERE id = ?`,
		emoji, nameUz, nameRu, nameEn, id,
	)
	if err != nil {
		return fmt.Errorf("failed to update department: %w", err)
	}
	return nil
}

// ToggleActive shows or hides a department from parents.
func (r *DepartmentRepository) ToggleActive(id int64) error {
	_, err := r.db.Exec(`UPDATE departments SET is_active = 1 - is_active WHERE id = ?`, id)
	return err
}

// Delete removes a department. Appeals keep their own snapshot of the name.
func (r *DepartmentRepository) Delete(id int64) error {
	_, err := r.db.Exec(`DELETE FROM departments WHERE id = ?`, id)
	return err
}

func (r *DepartmentRepository) Count() (int, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM departments`).Scan(&count)
	return count, err
}

// uniqueCode turns a name into a short ASCII slug, adding a numeric suffix
// until it is free.
func (r *DepartmentRepository) uniqueCode(nameUz string) (string, error) {
	base := slugify(nameUz)
	if base == "" {
		base = "dept"
	}

	code := base
	for i := 2; ; i++ {
		var count int
		if err := r.db.QueryRow(`SELECT COUNT(*) FROM departments WHERE code = ?`, code).Scan(&count); err != nil {
			return "", err
		}
		if count == 0 {
			return code, nil
		}
		code = fmt.Sprintf("%s_%d", base, i)
	}
}

// slugify keeps ASCII letters and digits, transliterating the Cyrillic that
// commonly shows up in department names.
func slugify(text string) string {
	var b strings.Builder

	for _, r := range strings.ToLower(text) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '_':
			if b.Len() > 0 && !strings.HasSuffix(b.String(), "_") {
				b.WriteRune('_')
			}
		default:
			if translit, ok := cyrillicMap[r]; ok {
				b.WriteString(translit)
			}
		}

		if b.Len() >= 24 {
			break
		}
	}

	return strings.Trim(b.String(), "_")
}

var cyrillicMap = map[rune]string{
	'а': "a", 'б': "b", 'в': "v", 'г': "g", 'д': "d", 'е': "e", 'ё': "yo",
	'ж': "j", 'з': "z", 'и': "i", 'й': "y", 'к': "k", 'л': "l", 'м': "m",
	'н': "n", 'о': "o", 'п': "p", 'р': "r", 'с': "s", 'т': "t", 'у': "u",
	'ф': "f", 'х': "h", 'ц': "ts", 'ч': "ch", 'ш': "sh", 'щ': "sch",
	'ъ': "", 'ы': "i", 'ь': "", 'э': "e", 'ю': "yu", 'я': "ya",
}
