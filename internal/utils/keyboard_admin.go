package utils

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"aist-bot/internal/i18n"
	"aist-bot/internal/models"
)

// AdminMenu is the admin home screen, and the only menu an admin ever sees.
// It is a reply keyboard so that it occupies the one menu slot at the
// bottom of the screen rather than stacking on top of a parent menu.
//
// It carries no appeal-creation buttons: the administration receives
// appeals, it does not write them.
func AdminMenu(lang i18n.Language) tgbotapi.ReplyKeyboardMarkup {
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(i18n.Get(i18n.BtnManageDepartments, lang)),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(i18n.Get(i18n.BtnManageClasses, lang)),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(i18n.Get(i18n.BtnRecent, lang)),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(i18n.Get(i18n.BtnStats, lang)),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(i18n.Get(i18n.BtnSettings, lang)),
		),
	)
	keyboard.ResizeKeyboard = true

	return keyboard
}

// AdminClassListKeyboard lists classes for editing, marking hidden ones.
func AdminClassListKeyboard(classes []models.Class, lang i18n.Language) tgbotapi.InlineKeyboardMarkup {
	rows := make([][]tgbotapi.InlineKeyboardButton, 0, len(classes)+2)
	row := make([]tgbotapi.InlineKeyboardButton, 0, 3)

	for _, c := range classes {
		label := c.Name
		if !c.IsActive {
			label = "🚫 " + label
		}

		row = append(row, tgbotapi.NewInlineKeyboardButtonData(label, Callback(CBAdminClassOne, c.ID)))

		if len(row) == 3 {
			rows = append(rows, row)
			row = make([]tgbotapi.InlineKeyboardButton, 0, 3)
		}
	}

	if len(row) > 0 {
		rows = append(rows, row)
	}

	rows = append(rows,
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.Get(i18n.BtnAddClass, lang), CBAdminClassAdd),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.Get(i18n.BtnBack, lang), CBAdminPanel),
		),
	)

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// AdminClassMenuKeyboard is what opens when an admin taps one class.
func AdminClassMenuKeyboard(class *models.Class, lang i18n.Language) tgbotapi.InlineKeyboardMarkup {
	toggleKey := i18n.BtnToggleOff
	if !class.IsActive {
		toggleKey = i18n.BtnToggleOn
	}

	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.Get(toggleKey, lang), Callback(CBAdminClassTgl, class.ID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.Get(i18n.BtnDelete, lang), Callback(CBAdminClassDel, class.ID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.Get(i18n.BtnBack, lang), CBAdminClasses),
		),
	)
}

// AdminDeptListKeyboard lists departments for editing, marking hidden ones.
func AdminDeptListKeyboard(departments []models.Department, lang i18n.Language) tgbotapi.InlineKeyboardMarkup {
	rows := make([][]tgbotapi.InlineKeyboardButton, 0, len(departments)+2)

	for i := range departments {
		d := &departments[i]

		label := d.LabelFor(lang.String())
		if !d.IsActive {
			label = "🚫 " + label
		}

		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(Truncate(label, 48), Callback(CBAdminDeptOne, d.ID)),
		))
	}

	rows = append(rows,
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.Get(i18n.BtnAddDept, lang), CBAdminDeptAdd),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.Get(i18n.BtnBack, lang), CBAdminPanel),
		),
	)

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// AdminDeptMenuKeyboard is what opens when an admin taps one department.
func AdminDeptMenuKeyboard(dept *models.Department, lang i18n.Language) tgbotapi.InlineKeyboardMarkup {
	toggleKey := i18n.BtnToggleOff
	if !dept.IsActive {
		toggleKey = i18n.BtnToggleOn
	}

	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.Get(i18n.BtnEditDept, lang), Callback(CBAdminDeptEdit, dept.ID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.Get(toggleKey, lang), Callback(CBAdminDeptTgl, dept.ID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.Get(i18n.BtnDelete, lang), Callback(CBAdminDeptDel, dept.ID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.Get(i18n.BtnBack, lang), CBAdminDepts),
		),
	)
}

// ConfirmDeleteKeyboard asks once before a destructive action.
func ConfirmDeleteKeyboard(confirmAction, cancelAction string, id int64, lang i18n.Language) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.Get(i18n.BtnYesDelete, lang), Callback(confirmAction, id)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.Get(i18n.BtnBack, lang), Callback(cancelAction, id)),
		),
	)
}

// AdminRecentKeyboard lists the newest appeals, each re-sending its PDF.
func AdminRecentKeyboard(submissions []models.SubmissionWithUser, lang i18n.Language) tgbotapi.InlineKeyboardMarkup {
	rows := make([][]tgbotapi.InlineKeyboardButton, 0, len(submissions)+1)

	for _, s := range submissions {
		label := i18n.TypeEmoji(string(s.Type)) + " " + s.Reference + " · " + s.ChildName
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(Truncate(label, 48), Callback(CBAdminDoc, s.ID)),
		))
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(i18n.Get(i18n.BtnBack, lang), CBAdminPanel),
	))

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// BackToAdminPanelKeyboard is the single-button footer of leaf admin screens.
func BackToAdminPanelKeyboard(lang i18n.Language) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(i18n.Get(i18n.BtnBack, lang), CBAdminPanel),
		),
	)
}
