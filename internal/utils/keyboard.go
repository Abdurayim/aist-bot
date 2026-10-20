package utils

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"aist-bot/internal/i18n"
	"aist-bot/internal/models"
)

// The keyboards below follow two rules, both chosen because many parents
// using this bot are older and not fluent with Telegram:
//
//  1. Steps where the user picks something use inline buttons, one choice
//     per row whenever the label is long, so nothing is mistapped.
//  2. Steps where the user types or sends a photo use a reply keyboard, so
//     "Back" and "Cancel" sit next to the keyboard instead of scrolling
//     away above the photos.

// LanguageKeyboard is the very first thing a new user sees, so it carries
// no translated text at all.
func LanguageKeyboard() tgbotapi.InlineKeyboardMarkup {
	rows := make([][]tgbotapi.InlineKeyboardButton, 0, len(i18n.All))

	for _, lang := range i18n.All {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(lang.Flag(), CallbackStr(CBLanguage, lang.String())),
		))
	}

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// PhoneKeyboard shows Telegram's contact-sharing button, which is one tap
// and cannot be mistyped.
func PhoneKeyboard(lang i18n.Language) tgbotapi.ReplyKeyboardMarkup {
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButtonContact(i18n.Get(i18n.BtnSharePhone, lang)),
		),
	)
	keyboard.ResizeKeyboard = true
	keyboard.OneTimeKeyboard = false

	return keyboard
}

// MainMenu is the home screen.
//
// Admins get a different home screen entirely, not the parent one with an
// extra button: the administration *receives* appeals, it does not file
// them, so the three appeal types are absent from their menu.
func MainMenu(lang i18n.Language, isAdmin bool) tgbotapi.ReplyKeyboardMarkup {
	if isAdmin {
		return AdminMenu(lang)
	}

	rows := [][]tgbotapi.KeyboardButton{
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(i18n.Get(i18n.BtnComplaint, lang)),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(i18n.Get(i18n.BtnProposal, lang)),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(i18n.Get(i18n.BtnGratitude, lang)),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(i18n.Get(i18n.BtnMyAppeals, lang)),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(i18n.Get(i18n.BtnHelp, lang)),
			tgbotapi.NewKeyboardButton(i18n.Get(i18n.BtnSettings, lang)),
		),
	}

	keyboard := tgbotapi.NewReplyKeyboard(rows...)
	keyboard.ResizeKeyboard = true

	return keyboard
}

// DepartmentKeyboard lists the topics an appeal can be addressed to. Names
// are long, so each gets its own row.
func DepartmentKeyboard(departments []models.Department, lang i18n.Language) tgbotapi.InlineKeyboardMarkup {
	rows := make([][]tgbotapi.InlineKeyboardButton, 0, len(departments)+1)

	for i := range departments {
		d := &departments[i]
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(d.LabelFor(lang.String()), Callback(CBDept, d.ID)),
		))
	}

	rows = append(rows, NavRow(lang, ""))

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// ClassKeyboard lists classes three per row, since names such as "5-A" are
// short and a long single column would need scrolling.
func ClassKeyboard(classes []models.Class, lang i18n.Language) tgbotapi.InlineKeyboardMarkup {
	rows := make([][]tgbotapi.InlineKeyboardButton, 0)
	row := make([]tgbotapi.InlineKeyboardButton, 0, 3)

	for _, c := range classes {
		row = append(row, tgbotapi.NewInlineKeyboardButtonData(c.Name, Callback(CBClass, c.ID)))

		if len(row) == 3 {
			rows = append(rows, row)
			row = make([]tgbotapi.InlineKeyboardButton, 0, 3)
		}
	}

	if len(row) > 0 {
		rows = append(rows, row)
	}

	rows = append(rows, NavRow(lang, StepDepartment))

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// NavRow is the Back/Cancel pair that closes every inline keyboard in the
// appeal flow. Passing an empty backTarget drops the Back button, which is
// what the first step of a flow needs.
func NavRow(lang i18n.Language, backTarget string) []tgbotapi.InlineKeyboardButton {
	if backTarget == "" {
		return tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.Get(i18n.BtnCancel, lang), CBCancel),
		)
	}

	return tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(i18n.Get(i18n.BtnBack, lang), CallbackStr(CBBack, backTarget)),
		tgbotapi.NewInlineKeyboardButtonData(i18n.Get(i18n.BtnCancel, lang), CBCancel),
	)
}

// NavKeyboard is the whole keyboard for a step where the user types their
// answer: there is nothing to choose, only a way back out.
func NavKeyboard(lang i18n.Language, backTarget string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(NavRow(lang, backTarget))
}

// SkipNavKeyboard adds a Skip button above the navigation row, used by the
// admin department editor where a translation may be left out.
func SkipNavKeyboard(lang i18n.Language, backTarget string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.Get(i18n.BtnSkip, lang), CBSkipStep),
		),
		NavRow(lang, backTarget),
	)
}

// PhotoDoneKeyboard is re-sent after every photo, so "Done" is always the
// last thing in the chat instead of scrolled away above the attachments.
func PhotoDoneKeyboard(lang i18n.Language) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.Get(i18n.BtnPhotoDone, lang), CBPhotoDone),
		),
		NavRow(lang, StepPhotoAsk),
	)
}

// PhotoAskKeyboard offers the optional photo step.
func PhotoAskKeyboard(lang i18n.Language) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.Get(i18n.BtnAddPhoto, lang), CBPhotoAdd),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.Get(i18n.BtnNoPhoto, lang), CBPhotoSkip),
		),
		NavRow(lang, StepText),
	)
}

// PhotoCollectKeyboard stays visible while photos are being attached, so
// "Done" is always one tap away under the attachment picker.
func PhotoCollectKeyboard(lang i18n.Language) tgbotapi.ReplyKeyboardMarkup {
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(i18n.Get(i18n.BtnPhotoDone, lang)),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(i18n.Get(i18n.BtnCancel, lang)),
		),
	)
	keyboard.ResizeKeyboard = true

	return keyboard
}

// ConfirmKeyboard is the final check before sending.
func ConfirmKeyboard(lang i18n.Language) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.Get(i18n.BtnSend, lang), CBConfirm),
		),
		NavRow(lang, StepPhotoAsk),
	)
}

// SettingsKeyboard offers the settings a parent can change themselves.
func SettingsKeyboard(lang i18n.Language) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.Get(i18n.BtnChangeLang, lang), CBSettingsLang),
		),
	)
}

// RemoveKeyboard hides the reply keyboard.
func RemoveKeyboard() tgbotapi.ReplyKeyboardRemove {
	return tgbotapi.NewRemoveKeyboard(true)
}
