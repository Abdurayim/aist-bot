package models

import (
	"encoding/json"
	"time"
)

// UserState is the persisted position of a user inside a conversation flow.
type UserState struct {
	TelegramID int64           `json:"telegram_id" db:"telegram_id"`
	State      string          `json:"state" db:"state"`
	Data       json.RawMessage `json:"data" db:"data"`
	UpdatedAt  time.Time       `json:"updated_at" db:"updated_at"`
}

// PhotoData is one photo attached to an appeal, kept in Telegram's cloud.
type PhotoData struct {
	FileID       string `json:"file_id"`
	FileUniqueID string `json:"file_unique_id"`
	FileSize     int    `json:"file_size"`
}

// StateData carries everything collected so far in the current flow.
type StateData struct {
	Language string `json:"language,omitempty"`

	// Appeal being composed.
	Type           SubmissionType `json:"type,omitempty"`
	DepartmentCode string         `json:"department_code,omitempty"`
	DepartmentName string         `json:"department_name,omitempty"`
	ClassName      string         `json:"class_name,omitempty"`
	ChildName      string         `json:"child_name,omitempty"`
	Text           string         `json:"text,omitempty"`
	Photos         []PhotoData    `json:"photos,omitempty"`

	// Admin department editor: the department being created or edited, and
	// the names collected so far.
	EditDepartmentID int64  `json:"edit_department_id,omitempty"`
	DraftNameUz      string `json:"draft_name_uz,omitempty"`
	DraftNameRu      string `json:"draft_name_ru,omitempty"`
	DraftEmoji       string `json:"draft_emoji,omitempty"`

	// Id of the last inline message shown, so it can be edited in place
	// instead of flooding the chat with new messages.
	PromptMessageID int `json:"prompt_message_id,omitempty"`
}

// MaxPhotos is the number of photos a single appeal may carry.
const MaxPhotos = 5

// Conversation states.
const (
	StateAwaitingLanguage = "awaiting_language"
	StateAwaitingPhone    = "awaiting_phone"
	StateIdle             = "idle"

	// Appeal flow, in order.
	StateAwaitingDepartment = "awaiting_department"
	StateAwaitingClass      = "awaiting_class"
	StateAwaitingChildName  = "awaiting_child_name"
	StateAwaitingText       = "awaiting_text"
	StateAwaitingPhotos     = "awaiting_photos"
	StateConfirming         = "confirming"

	// Admin flow.
	StateAwaitingClassName = "awaiting_class_name"

	// Admin department editor, one language per step.
	StateAwaitingDeptNameUz = "awaiting_dept_name_uz"
	StateAwaitingDeptNameRu = "awaiting_dept_name_ru"
	StateAwaitingDeptNameEn = "awaiting_dept_name_en"
)
