package models

import (
	"fmt"
	"time"
)

// SubmissionType is one of the three kinds of appeal the bot accepts.
type SubmissionType string

const (
	TypeComplaint SubmissionType = "complaint" // Shikoyat
	TypeProposal  SubmissionType = "proposal"  // Taklif
	TypeGratitude SubmissionType = "gratitude" // Minnatdorchilik
)

// IsValid reports whether t is one of the three supported types.
func (t SubmissionType) IsValid() bool {
	switch t {
	case TypeComplaint, TypeProposal, TypeGratitude:
		return true
	}
	return false
}

// Submission is a single appeal sent by a parent.
type Submission struct {
	ID int64 `json:"id" db:"id"`
	// Reference is the human-readable document number printed on the PDF.
	Reference      string         `json:"reference" db:"reference"`
	UserID         int64          `json:"user_id" db:"user_id"`
	Type           SubmissionType `json:"type" db:"type"`
	DepartmentCode string         `json:"department_code" db:"department_code"`
	// DepartmentName is a snapshot taken at submission time, in the language
	// the parent used. Admins may later rename or delete the department, and
	// past appeals must keep reading correctly.
	DepartmentName string    `json:"department_name" db:"department_name"`
	ClassName      string    `json:"class_name" db:"class_name"`
	ChildName      string    `json:"child_name" db:"child_name"`
	Text           string    `json:"text" db:"text"`
	PhotoCount     int       `json:"photo_count" db:"photo_count"`
	PDFFileID      string    `json:"pdf_file_id" db:"pdf_file_id"`
	PDFFilename    string    `json:"pdf_filename" db:"pdf_filename"`
	Language       string    `json:"language" db:"language"`
	Status         string    `json:"status" db:"status"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

// SubmissionWithUser is the admin-facing view that joins in the author.
type SubmissionWithUser struct {
	Submission
	PhoneNumber      string `json:"phone_number" db:"phone_number"`
	TelegramUsername string `json:"telegram_username" db:"telegram_username"`
	UserTelegramID   int64  `json:"user_telegram_id" db:"user_telegram_id"`
}

// BuildReference turns a row id into the document number shown to parents
// and printed on the PDF, for example AIST-2026-000042.
func BuildReference(id int64) string {
	return fmt.Sprintf("AIST-%d-%06d", time.Now().Year(), id)
}
