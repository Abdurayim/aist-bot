package utils

import (
	"strconv"
	"strings"
)

// Callback data is kept short and prefixed, because Telegram caps it at
// 64 bytes. The shape is "<action>" or "<action>:<id>".
const (
	CBLanguage  = "lang"  // lang:uz
	CBDept      = "dept"  // dept:<id>
	CBClass     = "class" // class:<id>
	CBPhotoAdd  = "ph_add"
	CBPhotoSkip = "ph_skip"
	CBPhotoDone = "ph_done"
	CBConfirm   = "send"
	CBCancel    = "cancel"
	CBBack      = "back" // back:<step>
	CBNoop      = "noop"

	// Settings
	CBSettingsLang = "set_lang"

	// Admin panel
	CBAdminPanel     = "a_panel"
	CBAdminClasses   = "a_cls"
	CBAdminClassAdd  = "a_cls_add"
	CBAdminClassOne  = "a_cls_one"  // a_cls_one:<id>
	CBAdminClassTgl  = "a_cls_tgl"  // a_cls_tgl:<id>
	CBAdminClassDel  = "a_cls_del"  // a_cls_del:<id>
	CBAdminClassDelY = "a_cls_dely" // a_cls_dely:<id>
	CBAdminDepts     = "a_dep"
	CBAdminDeptAdd   = "a_dep_add"
	CBAdminDeptOne   = "a_dep_one"  // a_dep_one:<id>
	CBAdminDeptEdit  = "a_dep_edit" // a_dep_edit:<id>
	CBAdminDeptTgl   = "a_dep_tgl"  // a_dep_tgl:<id>
	CBAdminDeptDel   = "a_dep_del"  // a_dep_del:<id>
	CBAdminDeptDelY  = "a_dep_dely" // a_dep_dely:<id>
	CBAdminStats     = "a_stats"
	CBAdminRecent    = "a_recent"
	CBAdminDoc       = "a_doc" // a_doc:<submission id>
	CBSkipStep       = "skip"
)

// Steps of the appeal flow. They double as the payload of a Back callback,
// so "go back" is simply "show this step again" and every step has exactly
// one place that renders it.
const (
	StepDepartment = "s_dept"
	StepClass      = "s_class"
	StepChildName  = "s_child"
	StepText       = "s_text"
	StepPhotoAsk   = "s_photoask"
	StepPhotos     = "s_photos"
	StepConfirm    = "s_confirm"
)

// PreviousStep returns the step a Back button should land on.
func PreviousStep(step string) string {
	switch step {
	case StepClass:
		return StepDepartment
	case StepChildName:
		return StepClass
	case StepText:
		return StepChildName
	case StepPhotoAsk:
		return StepText
	case StepPhotos, StepConfirm:
		return StepPhotoAsk
	default:
		return ""
	}
}

// Callback builds a callback payload out of an action and an id.
func Callback(action string, id int64) string {
	return action + ":" + strconv.FormatInt(id, 10)
}

// CallbackStr builds a callback payload out of an action and a string.
func CallbackStr(action, value string) string {
	return action + ":" + value
}

// ParseCallback splits a payload into its action and argument. A payload
// with no argument returns an empty value.
func ParseCallback(data string) (action, value string) {
	if idx := strings.Index(data, ":"); idx != -1 {
		return data[:idx], data[idx+1:]
	}
	return data, ""
}

// ParseCallbackID is ParseCallback for the common numeric-id case.
func ParseCallbackID(data string) (action string, id int64, ok bool) {
	action, value := ParseCallback(data)

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return action, 0, false
	}

	return action, parsed, true
}
