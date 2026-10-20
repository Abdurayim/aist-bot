package handlers

import (
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"aist-bot/internal/i18n"
	"aist-bot/internal/models"
	"aist-bot/internal/services"
	"aist-bot/internal/utils"
)

// HandleUpdate is the single entry point for everything Telegram sends.
func HandleUpdate(bot *services.BotService, update tgbotapi.Update) {
	defer func() {
		// One malformed update must never take the bot down for everyone.
		if r := recover(); r != nil {
			log.Printf("recovered from panic while handling update: %v", r)
		}
	}()

	// Updates from one person are handled one at a time; see locks.go.
	if from := updateSender(update); from != nil {
		defer updateLocks.lock(from.ID)()
	}

	switch {
	case update.Message != nil:
		c := newContext(bot, update.Message.Chat.ID, update.Message.From)

		if c.RoleChanged {
			if err := announceRoleChange(c); err != nil {
				log.Printf("error announcing role change to %d: %v", c.TelegramID, err)
			}
			return
		}

		if err := handleMessage(c, update.Message); err != nil {
			log.Printf("error handling message from %d: %v", c.TelegramID, err)
		}

	case update.CallbackQuery != nil:
		callback := update.CallbackQuery

		c := newContext(bot, callback.Message.Chat.ID, callback.From)
		c.MessageID = callback.Message.MessageID
		c.CallbackID = callback.ID

		if c.RoleChanged {
			c.Ack("")

			if err := announceRoleChange(c); err != nil {
				log.Printf("error announcing role change to %d: %v", c.TelegramID, err)
			}
			return
		}

		if err := handleCallback(c, callback.Data); err != nil {
			log.Printf("error handling callback from %d: %v", c.TelegramID, err)
		}
	}
}

// updateSender returns the person an update came from, or nil for update
// kinds this bot does not handle.
func updateSender(update tgbotapi.Update) *tgbotapi.User {
	switch {
	case update.Message != nil:
		return update.Message.From
	case update.CallbackQuery != nil:
		return update.CallbackQuery.From
	default:
		return nil
	}
}

// announceRoleChange tells someone their role changed and replaces their
// keyboard with the one for the new role. Nothing else is routed on this
// update: the button they just pressed belonged to the menu they no longer
// have.
func announceRoleChange(c *ctx) error {
	key := i18n.MsgRoleNowParent
	if c.IsAdmin {
		key = i18n.MsgRoleNowAdmin
	}

	return c.Send(c.T(key), utils.MainMenu(c.Lang, c.IsAdmin))
}

// handleMessage routes a plain message: commands first, then the shared
// buttons, then whatever the current step expects.
func handleMessage(c *ctx, message *tgbotapi.Message) error {
	// A shared contact only ever means "here is my phone number".
	if message.Contact != nil {
		return handleContact(c, message.Contact)
	}

	text := strings.TrimSpace(message.Text)

	if strings.HasPrefix(text, "/") {
		return handleCommand(c, text)
	}

	// Registration has to finish before anything else is reachable.
	if c.User == nil {
		switch c.State {
		case models.StateAwaitingPhone:
			return handlePhoneText(c, text)
		default:
			return handleStart(c)
		}
	}

	// The main menu keyboard stays visible during a flow, so a parent who
	// taps it mid-form is taken at their word and the flow restarts there
	// rather than their tap being swallowed as form input.
	// The admin keyboard is checked first: it is the one on screen while an
	// admin is in the panel, and it shares no labels with the parent menu.
	if handled, err := handleAdminMenuButton(c, text); handled {
		return err
	}

	if handled, err := handleMainMenuButton(c, text); handled {
		return err
	}

	switch c.State {
	case models.StateAwaitingChildName:
		return handleChildNameInput(c, text)

	case models.StateAwaitingText:
		return handleAppealTextInput(c, text)

	case models.StateAwaitingPhotos:
		return handlePhotoMessage(c, message)

	case models.StateAwaitingClassName:
		return handleClassNameInput(c, text)

	case models.StateAwaitingDeptNameUz, models.StateAwaitingDeptNameRu, models.StateAwaitingDeptNameEn:
		return handleDeptNameInput(c, text)

	case models.StateAwaitingDepartment, models.StateAwaitingClass,
		models.StateConfirming:
		// These steps are answered with inline buttons. Re-show the step so
		// a parent who typed instead of tapping is not left stuck.
		return showStep(c, stepForState(c.State))

	default:
		return showMainMenu(c, c.T(i18n.MsgMainMenu))
	}
}

// handleCommand handles the few slash commands the bot understands.
func handleCommand(c *ctx, text string) error {
	command := strings.ToLower(strings.TrimSpace(strings.Split(text, " ")[0]))
	if idx := strings.Index(command, "@"); idx != -1 {
		command = command[:idx]
	}

	switch command {
	case "/start":
		return handleStart(c)

	case "/help":
		if c.User == nil {
			return handleStart(c)
		}
		return showMainMenu(c, c.T(i18n.MsgHelp))

	case "/cancel":
		return handleCancel(c)

	case "/admin":
		if !c.IsAdmin {
			return c.Send(c.T(i18n.MsgAdminOnly), nil)
		}
		return showAdminPanel(c, false)

	default:
		if c.User == nil {
			return handleStart(c)
		}
		return showMainMenu(c, c.T(i18n.MsgMainMenu))
	}
}

// handleMainMenuButton reacts to the persistent bottom keyboard. It reports
// whether the text was one of those buttons.
func handleMainMenuButton(c *ctx, text string) (bool, error) {
	switch {
	case matchesButton(text, i18n.BtnComplaint):
		return true, startAppeal(c, models.TypeComplaint)

	case matchesButton(text, i18n.BtnProposal):
		return true, startAppeal(c, models.TypeProposal)

	case matchesButton(text, i18n.BtnGratitude):
		return true, startAppeal(c, models.TypeGratitude)

	case matchesButton(text, i18n.BtnMyAppeals):
		return true, leaveFlow(c, handleMyAppeals)

	case matchesButton(text, i18n.BtnHelp):
		return true, leaveFlow(c, func(c *ctx) error {
			return c.Send(c.T(i18n.MsgHelp), nil)
		})

	case matchesButton(text, i18n.BtnSettings):
		return true, leaveFlow(c, handleSettings)

	case matchesButton(text, i18n.BtnAdminPanel):
		if !c.IsAdmin {
			return true, c.Send(c.T(i18n.MsgAdminOnly), nil)
		}
		return true, showAdminPanel(c, false)

	case matchesButton(text, i18n.BtnMainMenu):
		return true, showMainMenu(c, c.T(i18n.MsgMainMenu))

	// Old keyboards linger in the chat history, so the navigation buttons
	// a previous version of this bot drew are still honoured.
	case matchesButton(text, i18n.BtnCancel):
		return true, handleCancel(c)

	case matchesButton(text, i18n.BtnBack):
		return true, handleBack(c, utils.PreviousStep(stepForState(c.State)))

	case matchesButton(text, i18n.BtnPhotoDone):
		if c.State == models.StateAwaitingPhotos {
			return true, showStep(c, utils.StepConfirm)
		}
	}

	return false, nil
}

// leaveFlow abandons any half-filled form before running an action that is
// not part of it. Without this, opening the help text mid-form would leave
// the next thing the parent typed being read as an answer.
func leaveFlow(c *ctx, action func(*ctx) error) error {
	if c.State != "" {
		if err := c.ClearState(); err != nil {
			return c.Fail("leaveFlow", err)
		}
	}

	return action(c)
}

// handleCallback routes an inline button tap.
func handleCallback(c *ctx, data string) error {
	action, value := utils.ParseCallback(data)

	switch action {
	case utils.CBLanguage:
		return handleLanguageChoice(c, value)

	case utils.CBCancel:
		c.Ack("")
		return handleCancel(c)

	case utils.CBBack:
		c.Ack("")
		return handleBack(c, value)

	case utils.CBNoop:
		c.Ack("")
		return nil
	}

	// Everything past this point needs a registered user.
	if c.User == nil {
		c.Ack("")
		return handleStart(c)
	}

	switch action {
	case utils.CBDept:
		return handleDepartmentChoice(c, value)

	case utils.CBClass:
		return handleClassChoice(c, value)

	case utils.CBPhotoAdd:
		c.Ack("")
		return showStep(c, utils.StepPhotos)

	case utils.CBPhotoSkip, utils.CBPhotoDone:
		c.Ack("")
		return showStep(c, utils.StepConfirm)

	case utils.CBConfirm:
		return handleSubmit(c)

	case utils.CBSettingsLang:
		c.Ack("")
		return handleChangeLanguage(c)

	case utils.CBSkipStep:
		c.Ack("")
		return handleDeptNameSkip(c)
	}

	// The remaining actions all belong to the admin panel.
	if strings.HasPrefix(action, "a_") {
		if !c.IsAdmin {
			c.Ack(c.T(i18n.MsgAdminOnly))
			return nil
		}
		return handleAdminCallback(c, action, value)
	}

	c.Ack("")
	return nil
}

// stepForState maps a stored state back to the step that renders it.
func stepForState(state string) string {
	switch state {
	case models.StateAwaitingDepartment:
		return utils.StepDepartment
	case models.StateAwaitingClass:
		return utils.StepClass
	case models.StateAwaitingChildName:
		return utils.StepChildName
	case models.StateAwaitingText:
		return utils.StepText
	case models.StateAwaitingPhotos:
		return utils.StepPhotos
	case models.StateConfirming:
		return utils.StepConfirm
	default:
		return ""
	}
}
