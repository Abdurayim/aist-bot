package repository

import (
	"database/sql"
	"fmt"

	"aist-bot/internal/models"
)

type SubmissionRepository struct{ db *sql.DB }

func NewSubmissionRepository(db *sql.DB) *SubmissionRepository {
	return &SubmissionRepository{db: db}
}

// Create inserts the appeal and its photos in one transaction and returns
// the stored row. The PDF is attached afterwards with AttachPDF, once the
// document has been generated and uploaded to Telegram.
func (r *SubmissionRepository) Create(s *models.Submission, photos []models.PhotoData) (*models.Submission, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.Exec(`
		INSERT INTO submissions
			(user_id, type, department_code, department_name, class_name, child_name, text, photo_count, language)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.UserID, string(s.Type), s.DepartmentCode, s.DepartmentName,
		s.ClassName, s.ChildName, s.Text, len(photos), s.Language,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert submission: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}

	for i, photo := range photos {
		if _, err := tx.Exec(`
			INSERT INTO submission_photos (submission_id, file_id, file_unique_id, file_size, order_index)
			VALUES (?, ?, ?, ?, ?)`,
			id, photo.FileID, photo.FileUniqueID, photo.FileSize, i,
		); err != nil {
			return nil, fmt.Errorf("failed to insert photo: %w", err)
		}
	}

	// The reference number is derived from the row id, so it is sequential
	// and unique without a separate counter.
	reference := models.BuildReference(id)
	if _, err := tx.Exec(`UPDATE submissions SET reference = ? WHERE id = ?`, reference, id); err != nil {
		return nil, fmt.Errorf("failed to set reference: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit submission: %w", err)
	}

	return r.GetByID(id)
}

// AttachPDF records the generated document against the appeal.
func (r *SubmissionRepository) AttachPDF(id int64, fileID, filename string) error {
	_, err := r.db.Exec(`UPDATE submissions SET pdf_file_id = ?, pdf_filename = ? WHERE id = ?`,
		fileID, filename, id)
	if err != nil {
		return fmt.Errorf("failed to attach pdf: %w", err)
	}
	return nil
}

const submissionColumns = `id, reference, user_id, type, department_code, department_name,
	class_name, child_name, text, photo_count, pdf_file_id, pdf_filename, language, status, created_at`

func (r *SubmissionRepository) GetByID(id int64) (*models.Submission, error) {
	var s models.Submission
	var submissionType string

	err := r.db.QueryRow(`SELECT `+submissionColumns+` FROM submissions WHERE id = ?`, id).Scan(
		&s.ID, &s.Reference, &s.UserID, &submissionType, &s.DepartmentCode, &s.DepartmentName,
		&s.ClassName, &s.ChildName, &s.Text, &s.PhotoCount, &s.PDFFileID, &s.PDFFilename,
		&s.Language, &s.Status, &s.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get submission: %w", err)
	}

	s.Type = models.SubmissionType(submissionType)
	return &s, nil
}

// GetByUser returns the most recent appeals of one parent.
func (r *SubmissionRepository) GetByUser(userID int64, limit int) ([]models.Submission, error) {
	rows, err := r.db.Query(
		`SELECT `+submissionColumns+` FROM submissions WHERE user_id = ? ORDER BY created_at DESC LIMIT ?`,
		userID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list submissions: %w", err)
	}
	defer rows.Close()

	submissions := make([]models.Submission, 0)
	for rows.Next() {
		var s models.Submission
		var submissionType string

		if err := rows.Scan(
			&s.ID, &s.Reference, &s.UserID, &submissionType, &s.DepartmentCode, &s.DepartmentName,
			&s.ClassName, &s.ChildName, &s.Text, &s.PhotoCount, &s.PDFFileID, &s.PDFFilename,
			&s.Language, &s.Status, &s.CreatedAt,
		); err != nil {
			return nil, err
		}

		s.Type = models.SubmissionType(submissionType)
		submissions = append(submissions, s)
	}

	return submissions, rows.Err()
}

// GetWithUserByID returns one appeal joined with its author, which is what
// the admin panel needs to resend a document.
func (r *SubmissionRepository) GetWithUserByID(id int64) (*models.SubmissionWithUser, error) {
	var s models.SubmissionWithUser
	var submissionType string

	err := r.db.QueryRow(`
		SELECT s.id, s.reference, s.user_id, s.type, s.department_code, s.department_name,
		       s.class_name, s.child_name, s.text, s.photo_count, s.pdf_file_id, s.pdf_filename,
		       s.language, s.status, s.created_at,
		       u.phone_number, u.telegram_username, u.telegram_id
		FROM submissions s
		INNER JOIN users u ON u.id = s.user_id
		WHERE s.id = ?`, id).Scan(
		&s.ID, &s.Reference, &s.UserID, &submissionType, &s.DepartmentCode, &s.DepartmentName,
		&s.ClassName, &s.ChildName, &s.Text, &s.PhotoCount, &s.PDFFileID, &s.PDFFilename,
		&s.Language, &s.Status, &s.CreatedAt,
		&s.PhoneNumber, &s.TelegramUsername, &s.UserTelegramID,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get submission: %w", err)
	}

	s.Type = models.SubmissionType(submissionType)
	return &s, nil
}

// GetRecent returns the newest appeals across all parents, for admins.
func (r *SubmissionRepository) GetRecent(limit int) ([]models.SubmissionWithUser, error) {
	rows, err := r.db.Query(`
		SELECT s.id, s.reference, s.user_id, s.type, s.department_code, s.department_name,
		       s.class_name, s.child_name, s.text, s.photo_count, s.pdf_file_id, s.pdf_filename,
		       s.language, s.status, s.created_at,
		       u.phone_number, u.telegram_username, u.telegram_id
		FROM submissions s
		INNER JOIN users u ON u.id = s.user_id
		ORDER BY s.created_at DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list recent submissions: %w", err)
	}
	defer rows.Close()

	result := make([]models.SubmissionWithUser, 0)
	for rows.Next() {
		var s models.SubmissionWithUser
		var submissionType string

		if err := rows.Scan(
			&s.ID, &s.Reference, &s.UserID, &submissionType, &s.DepartmentCode, &s.DepartmentName,
			&s.ClassName, &s.ChildName, &s.Text, &s.PhotoCount, &s.PDFFileID, &s.PDFFilename,
			&s.Language, &s.Status, &s.CreatedAt,
			&s.PhoneNumber, &s.TelegramUsername, &s.UserTelegramID,
		); err != nil {
			return nil, err
		}

		s.Type = models.SubmissionType(submissionType)
		result = append(result, s)
	}

	return result, rows.Err()
}

// CountByType returns how many appeals of each type exist, keyed by type.
func (r *SubmissionRepository) CountByType() (map[string]int, error) {
	rows, err := r.db.Query(`SELECT type, COUNT(*) FROM submissions GROUP BY type`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := map[string]int{}
	for rows.Next() {
		var submissionType string
		var count int
		if err := rows.Scan(&submissionType, &count); err != nil {
			return nil, err
		}
		counts[submissionType] = count
	}

	return counts, rows.Err()
}
