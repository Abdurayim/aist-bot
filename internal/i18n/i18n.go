package i18n

import "fmt"

// Language is a supported interface language.
type Language string

const (
	Uzbek   Language = "uz"
	Russian Language = "ru"
	English Language = "en"
)

// All is every language the bot speaks, in menu order.
var All = []Language{Uzbek, Russian, English}

// Flag returns the button label used on the language picker.
func (l Language) Flag() string {
	switch l {
	case Russian:
		return "🇷🇺 Русский"
	case English:
		return "🇬🇧 English"
	default:
		return "🇺🇿 O‘zbekcha"
	}
}

// Parse turns a stored language code into a Language, defaulting to Uzbek.
func Parse(code string) Language {
	switch code {
	case "ru":
		return Russian
	case "en":
		return English
	default:
		return Uzbek
	}
}

// String returns the code stored in the database.
func (l Language) String() string { return string(l) }

var catalogs = map[Language]map[string]string{
	Uzbek:   uzbek,
	Russian: russian,
	English: english,
}

// Get returns the translation of key, falling back to Uzbek and then to the
// key itself so a missing string is visible rather than silently empty.
func Get(key string, lang Language) string {
	if catalog, ok := catalogs[lang]; ok {
		if msg, ok := catalog[key]; ok {
			return msg
		}
	}

	if msg, ok := uzbek[key]; ok {
		return msg
	}

	return key
}

// Getf is Get followed by fmt.Sprintf.
func Getf(key string, lang Language, args ...interface{}) string {
	return fmt.Sprintf(Get(key, lang), args...)
}

// Message keys. Keeping them as constants means a typo is a compile error.
const (
	// Language + registration
	MsgChooseLanguage = "choose_language"
	MsgWelcome        = "welcome"
	MsgRequestPhone   = "request_phone"
	MsgPhoneHint      = "phone_hint"
	MsgPhoneInvalid   = "phone_invalid"
	MsgRegistered     = "registered"

	// Main menu
	MsgMainMenu = "main_menu"
	MsgHelp     = "help"

	// Appeal flow
	MsgStepDepartment    = "step_department"
	MsgStepClass         = "step_class"
	MsgNoClasses         = "no_classes"
	MsgStepChildName     = "step_child_name"
	MsgChildNameInvalid  = "child_name_invalid"
	MsgStepTextComplaint = "step_text_complaint"
	MsgStepTextProposal  = "step_text_proposal"
	MsgStepTextGratitude = "step_text_gratitude"
	MsgTextTooShort      = "text_too_short"
	MsgTextTooLong       = "text_too_long"
	MsgStepPhotoAsk      = "step_photo_ask"
	MsgPhotoHint         = "photo_hint"
	MsgPhotoAdded        = "photo_added"
	MsgPhotoMax          = "photo_max"
	MsgPhotoWrongInput   = "photo_wrong_input"
	MsgConfirmTitle      = "confirm_title"
	MsgSending           = "sending"
	MsgSent              = "sent"
	MsgSentNoAdmins      = "sent_no_admins"
	MsgSentNoDocument    = "sent_no_document"
	MsgCancelled         = "cancelled"

	// My appeals
	MsgMyAppealsTitle = "my_appeals_title"
	MsgNoAppeals      = "no_appeals"

	// Settings
	MsgSettingsTitle   = "settings_title"
	MsgLanguageChanged = "language_changed"

	// Labels reused in summaries
	LblType       = "lbl_type"
	LblDepartment = "lbl_department"
	LblClass      = "lbl_class"
	LblChild      = "lbl_child"
	LblPhone      = "lbl_phone"
	LblUsername   = "lbl_username"
	LblDate       = "lbl_date"
	LblText       = "lbl_text"
	LblPhotos     = "lbl_photos"
	LblReference  = "lbl_reference"
	LblNone       = "lbl_none"

	// Appeal type names
	TypeComplaint = "type_complaint"
	TypeProposal  = "type_proposal"
	TypeGratitude = "type_gratitude"

	// Buttons
	BtnSharePhone = "btn_share_phone"
	BtnComplaint  = "btn_complaint"
	BtnProposal   = "btn_proposal"
	BtnGratitude  = "btn_gratitude"
	BtnMyAppeals  = "btn_my_appeals"
	BtnSettings   = "btn_settings"
	BtnHelp       = "btn_help"
	BtnAdminPanel = "btn_admin_panel"
	BtnChangeLang = "btn_change_lang"
	BtnAddPhoto   = "btn_add_photo"
	BtnNoPhoto    = "btn_no_photo"
	BtnPhotoDone  = "btn_photo_done"
	BtnSend       = "btn_send"
	BtnCancel     = "btn_cancel"
	BtnBack       = "btn_back"
	BtnMainMenu   = "btn_main_menu"

	// Admin
	MsgAdminPanel        = "admin_panel"
	MsgAdminOnly         = "admin_only"
	MsgAdminCannotSubmit = "admin_cannot_submit"
	MsgRoleNowAdmin      = "role_now_admin"
	MsgRoleNowParent     = "role_now_parent"
	MsgClassPrompt       = "class_prompt"
	MsgClassAdded        = "class_added"
	MsgClassExists       = "class_exists"
	MsgClassInvalid      = "class_invalid"
	MsgClassList         = "class_list"
	MsgClassListEmpty    = "class_list_empty"
	MsgClassDeleted      = "class_deleted"
	MsgClassToggled      = "class_toggled"
	MsgAdminStats        = "admin_stats"
	MsgAdminRecent       = "admin_recent"
	MsgAdminNoRecent     = "admin_no_recent"
	MsgNewAppealHeader   = "new_appeal_header"

	BtnAddClass      = "btn_add_class"
	BtnManageClasses = "btn_manage_classes"
	BtnStats         = "btn_stats"
	BtnRecent        = "btn_recent"

	// Admin: department management
	MsgDeptList          = "dept_list"
	MsgDeptListEmpty     = "dept_list_empty"
	MsgDeptMenu          = "dept_menu"
	MsgDeptPromptUz      = "dept_prompt_uz"
	MsgDeptPromptRu      = "dept_prompt_ru"
	MsgDeptPromptEn      = "dept_prompt_en"
	MsgDeptAdded         = "dept_added"
	MsgDeptUpdated       = "dept_updated"
	MsgDeptDeleted       = "dept_deleted"
	MsgDeptToggled       = "dept_toggled"
	MsgDeptExists        = "dept_exists"
	MsgDeptInvalid       = "dept_invalid"
	MsgDeptDeleteConfirm = "dept_delete_confirm"
	MsgNoDepartments     = "no_departments"

	BtnManageDepartments = "btn_manage_departments"
	BtnAddDept           = "btn_add_dept"
	BtnEditDept          = "btn_edit_dept"
	BtnDelete            = "btn_delete"
	BtnToggleOn          = "btn_toggle_on"
	BtnToggleOff         = "btn_toggle_off"
	BtnSkip              = "btn_skip"
	BtnYesDelete         = "btn_yes_delete"

	// Errors
	ErrGeneric       = "err_generic"
	ErrNotRegistered = "err_not_registered"
	ErrSessionLost   = "err_session_lost"

	// PDF
	PDFTitleComplaint = "pdf_title_complaint"
	PDFTitleProposal  = "pdf_title_proposal"
	PDFTitleGratitude = "pdf_title_gratitude"
	PDFSectionDetails = "pdf_section_details"
	PDFSectionText    = "pdf_section_text"
	PDFSectionPhotos  = "pdf_section_photos"
	PDFPhotoNo        = "pdf_photo_no"
	PDFFooter         = "pdf_footer"
	PDFPage           = "pdf_page"
)

// TypeKey maps an appeal type code to its translation key.
func TypeKey(submissionType string) string {
	switch submissionType {
	case "proposal":
		return TypeProposal
	case "gratitude":
		return TypeGratitude
	default:
		return TypeComplaint
	}
}

// TypeEmoji returns the icon shown next to an appeal type.
func TypeEmoji(submissionType string) string {
	switch submissionType {
	case "proposal":
		return "💡"
	case "gratitude":
		return "🙏"
	default:
		return "📝"
	}
}

// PDFTitleKey maps an appeal type code to its PDF heading key.
func PDFTitleKey(submissionType string) string {
	switch submissionType {
	case "proposal":
		return PDFTitleProposal
	case "gratitude":
		return PDFTitleGratitude
	default:
		return PDFTitleComplaint
	}
}
