package handlers

import (
	"fmt"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"aist-bot/internal/i18n"
	"aist-bot/internal/models"
	"aist-bot/internal/utils"
	"aist-bot/internal/validator"
)

// The appeal flow is a straight line of steps. Every step is rendered by
// showStep and nothing else, so "back" is simply "render the step before
// this one" and there is exactly one place to change a step's wording.
//
//	department → class → child name → text → photo? → [photos] → confirm

// startAppeal begins a new complaint, proposal or gratitude note.
func startAppeal(c *ctx, appealType models.SubmissionType) error {
	// Admins receive appeals rather than writing them. Their menu offers no
	// way in, but a button left in the chat from before they were made an
	// admin still would, so the rule is enforced here as well.
	if c.IsAdmin {
		return c.Send(c.T(i18n.MsgAdminCannotSubmit), utils.MainMenu(c.Lang, true))
	}

	c.Data = &models.StateData{
		Language: c.Lang.String(),
		Type:     appealType,
		Photos:   []models.PhotoData{},
	}

	return showStep(c, utils.StepDepartment)
}

// handleBack returns to an earlier step, or to the main menu when there is
// nothing left to go back to.
func handleBack(c *ctx, step string) error {
	if step == "" {
		return handleCancel(c)
	}

	return showStep(c, step)
}

// showStep renders one step of the flow and stores the matching state.
func showStep(c *ctx, step string) error {
	switch step {
	case utils.StepDepartment:
		return showDepartmentStep(c)
	case utils.StepClass:
		return showClassStep(c)
	case utils.StepChildName:
		return showChildNameStep(c)
	case utils.StepText:
		return showTextStep(c)
	case utils.StepPhotoAsk:
		return showPhotoAskStep(c)
	case utils.StepPhotos:
		return showPhotoCollectStep(c)
	case utils.StepConfirm:
		return showConfirmStep(c)
	default:
		return handleCancel(c)
	}
}

// ── Step 1: department ───────────────────────────────────────────────

func showDepartmentStep(c *ctx) error {
	departments, err := c.bot.Departments.GetActive()
	if err != nil {
		return c.Fail("showDepartmentStep", err)
	}

	if len(departments) == 0 {
		// Nothing to choose from is an administrative gap, not a user
		// error, so say so plainly and return home.
		if err := c.ClearState(); err != nil {
			return c.Fail("showDepartmentStep/clear", err)
		}
		return c.Send(c.T(i18n.MsgNoDepartments), utils.MainMenu(c.Lang, c.IsAdmin))
	}

	if err := c.SetState(models.StateAwaitingDepartment); err != nil {
		return c.Fail("showDepartmentStep/state", err)
	}

	keyboard := utils.DepartmentKeyboard(departments, c.Lang)
	return c.Reply(appealHeader(c)+c.T(i18n.MsgStepDepartment), &keyboard)
}

func handleDepartmentChoice(c *ctx, value string) error {
	// A button tapped from an older, already finished appeal has no type
	// behind it. Send the parent back to the menu instead of starting a
	// form that could never be submitted.
	if !c.Data.Type.IsValid() {
		c.Ack("")
		return showMainMenu(c, c.T(i18n.MsgMainMenu))
	}

	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		c.Ack("")
		return showStep(c, utils.StepDepartment)
	}

	department, err := c.bot.Departments.GetByID(id)
	if err != nil {
		return c.Fail("handleDepartmentChoice", err)
	}

	if department == nil {
		// The admin deleted it while this parent was choosing.
		c.Ack("")
		return showStep(c, utils.StepDepartment)
	}

	c.Data.DepartmentCode = department.Code
	c.Data.DepartmentName = department.NameFor(c.Lang.String())

	c.Ack("")
	return showStep(c, utils.StepClass)
}

// ── Step 2: class ────────────────────────────────────────────────────

func showClassStep(c *ctx) error {
	classes, err := c.bot.Classes.GetActive()
	if err != nil {
		return c.Fail("showClassStep", err)
	}

	if len(classes) == 0 {
		if err := c.ClearState(); err != nil {
			return c.Fail("showClassStep/clear", err)
		}
		return c.Send(c.T(i18n.MsgNoClasses), utils.MainMenu(c.Lang, c.IsAdmin))
	}

	if err := c.SetState(models.StateAwaitingClass); err != nil {
		return c.Fail("showClassStep/state", err)
	}

	keyboard := utils.ClassKeyboard(classes, c.Lang)
	return c.Reply(appealHeader(c)+c.T(i18n.MsgStepClass), &keyboard)
}

func handleClassChoice(c *ctx, value string) error {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		c.Ack("")
		return showStep(c, utils.StepClass)
	}

	class, err := c.bot.Classes.GetByID(id)
	if err != nil {
		return c.Fail("handleClassChoice", err)
	}

	if class == nil {
		c.Ack("")
		return showStep(c, utils.StepClass)
	}

	c.Data.ClassName = class.Name

	c.Ack("")
	return showStep(c, utils.StepChildName)
}

// ── Step 3: child's name ─────────────────────────────────────────────

func showChildNameStep(c *ctx) error {
	if err := c.SetState(models.StateAwaitingChildName); err != nil {
		return c.Fail("showChildNameStep/state", err)
	}

	keyboard := utils.NavKeyboard(c.Lang, utils.StepClass)
	return c.Reply(appealHeader(c)+c.T(i18n.MsgStepChildName), &keyboard)
}

func handleChildNameInput(c *ctx, text string) error {
	name, err := validator.ValidateChildName(text)
	if err != nil {
		return c.Send(c.T(i18n.MsgChildNameInvalid), utils.NavKeyboard(c.Lang, utils.StepClass))
	}

	c.Data.ChildName = name

	return showStep(c, utils.StepText)
}

// ── Step 4: the message itself ───────────────────────────────────────

func showTextStep(c *ctx) error {
	if err := c.SetState(models.StateAwaitingText); err != nil {
		return c.Fail("showTextStep/state", err)
	}

	var key string
	switch c.Data.Type {
	case models.TypeProposal:
		key = i18n.MsgStepTextProposal
	case models.TypeGratitude:
		key = i18n.MsgStepTextGratitude
	default:
		key = i18n.MsgStepTextComplaint
	}

	keyboard := utils.NavKeyboard(c.Lang, utils.StepChildName)
	return c.Reply(appealHeader(c)+c.T(key), &keyboard)
}

func handleAppealTextInput(c *ctx, text string) error {
	body, length, err := validator.ValidateText(text)
	if err != nil {
		keyboard := utils.NavKeyboard(c.Lang, utils.StepChildName)

		if length > validator.MaxTextLength {
			return c.Send(c.Tf(i18n.MsgTextTooLong, validator.MaxTextLength, length), keyboard)
		}

		return c.Send(c.Tf(i18n.MsgTextTooShort, validator.MinTextLength), keyboard)
	}

	c.Data.Text = body

	return showStep(c, utils.StepPhotoAsk)
}

// ── Step 5: the optional photo ───────────────────────────────────────

func showPhotoAskStep(c *ctx) error {
	if err := c.SetState(models.StateAwaitingPhotos); err != nil {
		return c.Fail("showPhotoAskStep/state", err)
	}

	keyboard := utils.PhotoAskKeyboard(c.Lang)
	return c.Reply(appealHeader(c)+c.T(i18n.MsgStepPhotoAsk), &keyboard)
}

func showPhotoCollectStep(c *ctx) error {
	if err := c.SetState(models.StateAwaitingPhotos); err != nil {
		return c.Fail("showPhotoCollectStep/state", err)
	}

	keyboard := utils.PhotoDoneKeyboard(c.Lang)
	return c.Reply(c.Tf(i18n.MsgPhotoHint, models.MaxPhotos), &keyboard)
}

// handlePhotoMessage collects an attached photo. The Done button is re-sent
// with every confirmation so it stays at the bottom of the chat, under the
// photos, instead of scrolling out of reach.
func handlePhotoMessage(c *ctx, message *tgbotapi.Message) error {
	photo := largestPhoto(message)
	if photo == nil {
		return c.Send(c.T(i18n.MsgPhotoWrongInput), utils.PhotoDoneKeyboard(c.Lang))
	}

	if len(c.Data.Photos) >= models.MaxPhotos {
		return c.Send(c.Tf(i18n.MsgPhotoMax, models.MaxPhotos), utils.PhotoDoneKeyboard(c.Lang))
	}

	c.Data.Photos = append(c.Data.Photos, models.PhotoData{
		FileID:       photo.FileID,
		FileUniqueID: photo.FileUniqueID,
		FileSize:     photo.FileSize,
	})

	if err := c.SaveData(); err != nil {
		return c.Fail("handlePhotoMessage/save", err)
	}

	return c.Send(
		c.Tf(i18n.MsgPhotoAdded, len(c.Data.Photos), models.MaxPhotos),
		utils.PhotoDoneKeyboard(c.Lang),
	)
}

// largestPhoto picks the highest resolution Telegram offers, and also
// accepts an image sent as a file, which is what some phones do by default.
func largestPhoto(message *tgbotapi.Message) *tgbotapi.PhotoSize {
	if len(message.Photo) > 0 {
		return &message.Photo[len(message.Photo)-1]
	}

	if message.Document != nil && isImageMimeType(message.Document.MimeType) {
		return &tgbotapi.PhotoSize{
			FileID:       message.Document.FileID,
			FileUniqueID: message.Document.FileUniqueID,
			FileSize:     message.Document.FileSize,
		}
	}

	return nil
}

func isImageMimeType(mimeType string) bool {
	switch mimeType {
	case "image/jpeg", "image/jpg", "image/png", "image/gif":
		return true
	default:
		return false
	}
}

// ── Step 6: confirm and send ─────────────────────────────────────────

func showConfirmStep(c *ctx) error {
	if err := c.SetState(models.StateConfirming); err != nil {
		return c.Fail("showConfirmStep/state", err)
	}

	keyboard := utils.ConfirmKeyboard(c.Lang)
	return c.Reply(buildSummary(c), &keyboard)
}

// buildSummary is the last thing a parent reads before sending, so it
// repeats every answer in full rather than summarising.
func buildSummary(c *ctx) string {
	photos := strconv.Itoa(len(c.Data.Photos))
	if len(c.Data.Photos) == 0 {
		photos = c.T(i18n.LblNone)
	}

	return fmt.Sprintf(
		"%s\n\n"+
			"%s %s: <b>%s</b>\n"+
			"📂 %s: <b>%s</b>\n"+
			"🏫 %s: <b>%s</b>\n"+
			"👶 %s: <b>%s</b>\n"+
			"📷 %s: <b>%s</b>\n\n"+
			"📝 <b>%s:</b>\n%s",
		c.T(i18n.MsgConfirmTitle),
		i18n.TypeEmoji(string(c.Data.Type)),
		utils.EscapeHTML(c.T(i18n.LblType)),
		utils.EscapeHTML(c.T(i18n.TypeKey(string(c.Data.Type)))),
		utils.EscapeHTML(c.T(i18n.LblDepartment)), utils.EscapeHTML(c.Data.DepartmentName),
		utils.EscapeHTML(c.T(i18n.LblClass)), utils.EscapeHTML(c.Data.ClassName),
		utils.EscapeHTML(c.T(i18n.LblChild)), utils.EscapeHTML(c.Data.ChildName),
		utils.EscapeHTML(c.T(i18n.LblPhotos)), utils.EscapeHTML(photos),
		utils.EscapeHTML(c.T(i18n.LblText)),
		utils.EscapeHTML(utils.Truncate(c.Data.Text, 1500)),
	)
}

// handleSubmit stores the appeal, builds the PDF and delivers it.
func handleSubmit(c *ctx) error {
	// Guard against a double tap on Send, which would otherwise file the
	// same appeal twice.
	if c.State != models.StateConfirming {
		c.Ack("")
		return nil
	}

	if !appealIsComplete(c.Data) {
		c.Ack("")
		return c.Send(c.T(i18n.ErrSessionLost), utils.MainMenu(c.Lang, c.IsAdmin))
	}

	c.Ack("")

	if err := c.SetState(models.StateIdle); err != nil {
		return c.Fail("handleSubmit/state", err)
	}

	if err := c.Reply(c.T(i18n.MsgSending), nil); err != nil {
		return err
	}

	result, err := c.bot.Appeals.Submit(c.User, c.Data, c.ChatID)
	if err != nil {
		// The appeal may already be stored; tell the parent plainly rather
		// than implying nothing happened.
		return c.Fail("handleSubmit/submit", err)
	}

	if err := c.ClearState(); err != nil {
		return c.Fail("handleSubmit/clear", err)
	}

	key := i18n.MsgSent
	switch {
	case result.AdminsLinked == 0:
		key = i18n.MsgSentNoAdmins
	case !result.DocumentSent:
		key = i18n.MsgSentNoDocument
	}

	return c.Send(c.Tf(key, result.Submission.Reference), utils.MainMenu(c.Lang, c.IsAdmin))
}

// appealIsComplete guards against a state that was cleaned up or corrupted
// between steps, which would otherwise produce a half-empty document.
func appealIsComplete(data *models.StateData) bool {
	return data.Type.IsValid() &&
		data.DepartmentCode != "" &&
		data.ClassName != "" &&
		data.ChildName != "" &&
		data.Text != ""
}

// appealHeader labels every step with the kind of appeal being written, so
// a parent always sees what they are in the middle of.
func appealHeader(c *ctx) string {
	if !c.Data.Type.IsValid() {
		return ""
	}

	return fmt.Sprintf("%s <b>%s</b>\n\n",
		i18n.TypeEmoji(string(c.Data.Type)),
		utils.EscapeHTML(c.T(i18n.TypeKey(string(c.Data.Type)))),
	)
}
