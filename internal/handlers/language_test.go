package handlers

import (
	"strings"
	"testing"

	"aist-bot/internal/i18n"
	"aist-bot/internal/utils"
)

// TestEveryLanguageCoversEveryKey stops a half-translated release: a missing
// key would silently fall back to Uzbek for a Russian or English speaker.
func TestEveryLanguageCoversEveryKey(t *testing.T) {
	keys := []string{
		i18n.MsgChooseLanguage, i18n.MsgWelcome, i18n.MsgRequestPhone, i18n.MsgPhoneHint,
		i18n.MsgPhoneInvalid, i18n.MsgRegistered, i18n.MsgMainMenu, i18n.MsgHelp,
		i18n.MsgStepDepartment, i18n.MsgStepClass, i18n.MsgNoClasses, i18n.MsgNoDepartments,
		i18n.MsgStepChildName, i18n.MsgChildNameInvalid,
		i18n.MsgStepTextComplaint, i18n.MsgStepTextProposal, i18n.MsgStepTextGratitude,
		i18n.MsgTextTooShort, i18n.MsgTextTooLong,
		i18n.MsgStepPhotoAsk, i18n.MsgPhotoHint, i18n.MsgPhotoAdded, i18n.MsgPhotoMax,
		i18n.MsgPhotoWrongInput, i18n.MsgConfirmTitle, i18n.MsgSending, i18n.MsgSent,
		i18n.MsgSentNoAdmins, i18n.MsgCancelled, i18n.MsgMyAppealsTitle, i18n.MsgNoAppeals,
		i18n.MsgSettingsTitle, i18n.MsgLanguageChanged,
		i18n.LblType, i18n.LblDepartment, i18n.LblClass, i18n.LblChild, i18n.LblPhone,
		i18n.LblUsername, i18n.LblDate, i18n.LblText, i18n.LblPhotos, i18n.LblReference, i18n.LblNone,
		i18n.TypeComplaint, i18n.TypeProposal, i18n.TypeGratitude,
		i18n.BtnSharePhone, i18n.BtnComplaint, i18n.BtnProposal, i18n.BtnGratitude,
		i18n.BtnMyAppeals, i18n.BtnSettings, i18n.BtnHelp, i18n.BtnAdminPanel,
		i18n.BtnChangeLang, i18n.BtnAddPhoto, i18n.BtnNoPhoto, i18n.BtnPhotoDone,
		i18n.BtnSend, i18n.BtnCancel, i18n.BtnBack, i18n.BtnMainMenu,
		i18n.MsgAdminPanel, i18n.MsgAdminOnly, i18n.MsgClassPrompt, i18n.MsgClassAdded,
		i18n.MsgClassExists, i18n.MsgClassInvalid, i18n.MsgClassList, i18n.MsgClassListEmpty,
		i18n.MsgClassDeleted, i18n.MsgClassToggled, i18n.MsgAdminStats, i18n.MsgAdminRecent,
		i18n.MsgAdminNoRecent, i18n.MsgNewAppealHeader,
		i18n.BtnAddClass, i18n.BtnManageClasses, i18n.BtnStats, i18n.BtnRecent,
		i18n.MsgDeptList, i18n.MsgDeptListEmpty, i18n.MsgDeptMenu,
		i18n.MsgDeptPromptUz, i18n.MsgDeptPromptRu, i18n.MsgDeptPromptEn,
		i18n.MsgDeptAdded, i18n.MsgDeptUpdated, i18n.MsgDeptDeleted, i18n.MsgDeptToggled,
		i18n.MsgDeptExists, i18n.MsgDeptInvalid, i18n.MsgDeptDeleteConfirm,
		i18n.BtnManageDepartments, i18n.BtnAddDept, i18n.BtnEditDept, i18n.BtnDelete,
		i18n.BtnToggleOn, i18n.BtnToggleOff, i18n.BtnSkip, i18n.BtnYesDelete,
		i18n.ErrGeneric, i18n.ErrNotRegistered, i18n.ErrSessionLost,
		i18n.PDFTitleComplaint, i18n.PDFTitleProposal, i18n.PDFTitleGratitude,
		i18n.PDFSectionDetails, i18n.PDFSectionText, i18n.PDFSectionPhotos,
		i18n.PDFPhotoNo, i18n.PDFFooter, i18n.PDFPage,
	}

	for _, lang := range i18n.All {
		for _, key := range keys {
			value := i18n.Get(key, lang)

			if value == key {
				t.Errorf("%s has no translation for %q", lang, key)
			}

			if strings.TrimSpace(value) == "" {
				t.Errorf("%s has an empty translation for %q", lang, key)
			}
		}
	}
}

// TestButtonLabelsAreDistinct guards against two menu buttons sharing a
// label, which would make one of them unreachable.
func TestButtonLabelsAreDistinct(t *testing.T) {
	buttons := []string{
		i18n.BtnComplaint, i18n.BtnProposal, i18n.BtnGratitude,
		i18n.BtnMyAppeals, i18n.BtnSettings, i18n.BtnHelp, i18n.BtnAdminPanel,
	}

	for _, lang := range i18n.All {
		seen := map[string]string{}

		for _, key := range buttons {
			label := i18n.Get(key, lang)

			if other, ok := seen[label]; ok {
				t.Errorf("%s: %q and %q share the label %q", lang, other, key, label)
			}

			seen[label] = key
		}
	}
}

// TestRussianFlow runs a whole appeal in Russian and checks the stored
// record carries the language, so its PDF is written in Russian too.
func TestRussianFlow(t *testing.T) {
	b := newTestBot(t, adminPhone)
	seedSchool(t, b)
	registerParent(t, b, i18n.Russian, parent, parentPhone)

	b.tg.Reset()
	b.sendText(parent.ID, parent, i18n.Get(i18n.BtnComplaint, i18n.Russian))
	assertContains(t, b.tg, "Отдел")

	b.tap(parent.ID, parent, b.callbackFor(t, "Столовая"))
	assertContains(t, b.tg, "Класс")

	b.tap(parent.ID, parent, b.callbackFor(t, "9-B"))
	assertContains(t, b.tg, "ребёнка")

	b.sendText(parent.ID, parent, "Алиев Сардор")
	assertContains(t, b.tg, "Текст жалобы")

	b.sendText(parent.ID, parent, "Еда в столовой подаётся холодной, пожалуйста, проверьте.")
	b.tap(parent.ID, parent, b.callbackFor(t, "продолжить"))
	b.tap(parent.ID, parent, "send")

	stored, _ := b.Submissions.GetRecent(1)
	if len(stored) != 1 {
		t.Fatal("the Russian appeal was not stored")
	}

	if stored[0].Language != "ru" {
		t.Fatalf("stored language = %q, want ru", stored[0].Language)
	}

	// The department name is snapshotted in the language the parent used.
	if stored[0].DepartmentName != "Столовая" {
		t.Fatalf("department name = %q, want the Russian name", stored[0].DepartmentName)
	}
}

// TestEnglishFlow does the same in English.
func TestEnglishFlow(t *testing.T) {
	b := newTestBot(t, adminPhone)
	seedSchool(t, b)
	registerParent(t, b, i18n.English, parent, parentPhone)

	b.tg.Reset()
	b.sendText(parent.ID, parent, i18n.Get(i18n.BtnGratitude, i18n.English))
	assertContains(t, b.tg, "Department")

	b.tap(parent.ID, parent, b.callbackFor(t, "Mentors"))
	b.tap(parent.ID, parent, b.callbackFor(t, "9-B"))

	assertContains(t, b.tg, "child's first and last name")
	b.sendText(parent.ID, parent, "Sardor Aliyev")

	assertContains(t, b.tg, "thank-you note")
	b.sendText(parent.ID, parent, "Thank you very much to our mentor for the extra lessons.")

	b.tap(parent.ID, parent, b.callbackFor(t, "No, continue"))
	b.tap(parent.ID, parent, "send")

	stored, _ := b.Submissions.GetRecent(1)
	if len(stored) != 1 || stored[0].Language != "en" {
		t.Fatalf("the English appeal was not stored correctly: %+v", stored)
	}

	if stored[0].DepartmentName != "Mentors" {
		t.Fatalf("department name = %q, want the English name", stored[0].DepartmentName)
	}
}

// TestChangingLanguageSwitchesTheWholeInterface covers the settings path.
func TestChangingLanguageSwitchesTheWholeInterface(t *testing.T) {
	b := newTestBot(t, adminPhone)
	registerParent(t, b, i18n.Uzbek, parent, parentPhone)

	b.sendText(parent.ID, parent, i18n.Get(i18n.BtnSettings, i18n.Uzbek))
	b.tap(parent.ID, parent, utils.CBSettingsLang)

	b.tg.Reset()
	b.tap(parent.ID, parent, "lang:ru")

	stored, _ := b.Users.GetByTelegramID(parent.ID)
	if stored.Language != "ru" {
		t.Fatalf("stored language = %q, want ru", stored.Language)
	}

	if !b.hasButton(i18n.Get(i18n.BtnComplaint, i18n.Russian)) {
		t.Fatalf("the menu was not redrawn in Russian:\n%s", dump(b.tg.Sent()))
	}
}

// TestAButtonFromAnOtherLanguageStillWorks: old keyboards linger in the chat
// history, and a parent who switched language may well tap one of them.
func TestAButtonFromAnOtherLanguageStillWorks(t *testing.T) {
	b := newTestBot(t, adminPhone)
	seedSchool(t, b)
	registerParent(t, b, i18n.Russian, parent, parentPhone)

	b.tg.Reset()
	// Tapping the Uzbek-labelled button from before the language switch.
	b.sendText(parent.ID, parent, i18n.Get(i18n.BtnComplaint, i18n.Uzbek))

	// The flow starts, and answers in the language now selected.
	assertContains(t, b.tg, "Отдел")
}
