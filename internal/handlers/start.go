package handlers

import (
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"aist-bot/internal/i18n"
	"aist-bot/internal/models"
	"aist-bot/internal/utils"
	"aist-bot/internal/validator"
)

// handleStart greets the user. A returning parent goes straight to the main
// menu; a new one is asked for a language first.
func handleStart(c *ctx) error {
	if c.User != nil {
		if err := c.ClearState(); err != nil {
			return c.Fail("handleStart/clear", err)
		}
		return showMainMenu(c, c.T(forRole(c, i18n.MsgWelcome, i18n.MsgWelcomeAdmin)))
	}

	if err := c.SetState(models.StateAwaitingLanguage); err != nil {
		return c.Fail("handleStart/state", err)
	}

	// Before a language is chosen the prompt is shown in all three, since
	// there is no way to know which one the reader understands.
	prompts := make([]string, 0, len(i18n.All))
	for _, lang := range i18n.All {
		prompts = append(prompts, i18n.Get(i18n.MsgChooseLanguage, lang))
	}

	return c.Send(strings.Join(prompts, "\n"), utils.LanguageKeyboard())
}

// handleLanguageChoice stores the chosen language and moves on. It serves
// both first-time registration and the language switch in settings.
func handleLanguageChoice(c *ctx, code string) error {
	lang := i18n.Parse(code)
	c.Lang = lang
	c.Data.Language = lang.String()

	c.Ack("")

	if c.User != nil {
		if err := c.bot.Users.UpdateLanguage(c.TelegramID, lang.String()); err != nil {
			return c.Fail("handleLanguageChoice/update", err)
		}

		c.User.Language = lang.String()

		if err := c.Reply(c.T(i18n.MsgLanguageChanged), nil); err != nil {
			return err
		}

		return showMainMenu(c, c.T(i18n.MsgMainMenu))
	}

	if err := c.SetState(models.StateAwaitingPhone); err != nil {
		return c.Fail("handleLanguageChoice/state", err)
	}

	if err := c.Reply(c.T(i18n.MsgWelcome), nil); err != nil {
		return err
	}

	return c.Send(
		c.T(i18n.MsgRequestPhone)+"\n\n"+c.T(i18n.MsgPhoneHint),
		utils.PhoneKeyboard(c.Lang),
	)
}

// handleContact completes registration from a shared contact, which is the
// path almost every parent takes.
func handleContact(c *ctx, contact *tgbotapi.Contact) error {
	// Only accept the sender's own contact; a forwarded one would register
	// the wrong phone number against this account.
	if contact.UserID != 0 && contact.UserID != c.TelegramID {
		return c.Send(c.T(i18n.MsgPhoneInvalid), utils.PhoneKeyboard(c.Lang))
	}

	return completeRegistration(c, validator.NormalizePhone(contact.PhoneNumber))
}

// handlePhoneText completes registration from a typed number, the fallback
// for people whose client does not show the contact button.
func handlePhoneText(c *ctx, text string) error {
	phone, err := validator.ValidatePhone(text)
	if err != nil {
		return c.Send(c.T(i18n.MsgPhoneInvalid), utils.PhoneKeyboard(c.Lang))
	}

	return completeRegistration(c, phone)
}

// completeRegistration stores the user and opens the main menu.
func completeRegistration(c *ctx, phone string) error {
	if phone == "" {
		return c.Send(c.T(i18n.MsgPhoneInvalid), utils.PhoneKeyboard(c.Lang))
	}

	language := c.Data.Language
	if language == "" {
		language = c.Lang.String()
	}

	user, err := c.bot.Users.Upsert(c.TelegramID, c.Username, phone, language)
	if err != nil {
		return c.Fail("completeRegistration/upsert", err)
	}

	c.User = user
	c.Lang = i18n.Parse(user.Language)
	c.IsAdmin = c.bot.IsAdmin(user.PhoneNumber, user.TelegramID)

	// Record the role now. The menu they are about to be shown is the one
	// for this role, so without this they would be told on their very next
	// message that their role had just changed.
	role := models.RoleFor(c.IsAdmin)
	if user.Role != role {
		if err := c.bot.Users.UpdateRole(user.TelegramID, role); err != nil {
			return c.Fail("completeRegistration/role", err)
		}
		c.User.Role = role
	}

	if err := c.ClearState(); err != nil {
		return c.Fail("completeRegistration/clear", err)
	}

	return showMainMenu(c, c.T(forRole(c, i18n.MsgRegistered, i18n.MsgRegisteredAdmin)))
}

// forRole picks between the parent wording and the admin wording.
//
// The two roles use the bot for opposite purposes, so greeting an admin
// with "you can send a complaint through this bot" is not just untidy — it
// describes something they are not allowed to do.
func forRole(c *ctx, parentKey, adminKey string) string {
	if c.IsAdmin {
		return adminKey
	}
	return parentKey
}

// showMainMenu sends text with the home keyboard and resets the state, so
// it doubles as the way back out of any flow.
func showMainMenu(c *ctx, text string) error {
	if c.State != "" {
		if err := c.ClearState(); err != nil {
			return c.Fail("showMainMenu/clear", err)
		}
	}

	return c.Send(text, utils.MainMenu(c.Lang, c.IsAdmin))
}

// handleCancel abandons whatever flow is in progress.
func handleCancel(c *ctx) error {
	if c.User == nil {
		return handleStart(c)
	}

	// Remove the inline keyboard of the step being abandoned, so its
	// buttons cannot be tapped again from the chat history.
	if c.MessageID != 0 {
		_ = c.bot.Telegram.Edit(c.ChatID, c.MessageID, c.T(i18n.MsgCancelled), nil)
		if err := c.ClearState(); err != nil {
			return c.Fail("handleCancel/clear", err)
		}
		return c.Send(c.T(i18n.MsgMainMenu), utils.MainMenu(c.Lang, c.IsAdmin))
	}

	return showMainMenu(c, c.T(i18n.MsgCancelled))
}

// handleSettings opens the settings a parent can change themselves.
func handleSettings(c *ctx) error {
	return c.Send(c.T(i18n.MsgSettingsTitle), utils.SettingsKeyboard(c.Lang))
}

// handleChangeLanguage re-opens the language picker for a registered user.
func handleChangeLanguage(c *ctx) error {
	keyboard := utils.LanguageKeyboard()
	return c.Reply(c.T(i18n.MsgChooseLanguage), &keyboard)
}

// handleMyAppeals lists what this parent has sent, newest first.
func handleMyAppeals(c *ctx) error {
	if c.IsAdmin {
		return c.Send(c.T(i18n.MsgAdminCannotSubmit), utils.MainMenu(c.Lang, true))
	}

	const limit = 10

	submissions, err := c.bot.Submissions.GetByUser(c.User.ID, limit)
	if err != nil {
		return c.Fail("handleMyAppeals", err)
	}

	if len(submissions) == 0 {
		return c.Send(c.T(i18n.MsgNoAppeals), utils.MainMenu(c.Lang, c.IsAdmin))
	}

	var b strings.Builder
	b.WriteString(c.Tf(i18n.MsgMyAppealsTitle, len(submissions)))
	b.WriteString("\n")

	for _, s := range submissions {
		b.WriteString(fmt.Sprintf(
			"\n%s <b>%s</b>\n<code>%s</code>\n📂 %s · 🏫 %s\n🕒 %s\n",
			i18n.TypeEmoji(string(s.Type)),
			utils.EscapeHTML(i18n.Get(i18n.TypeKey(string(s.Type)), c.Lang)),
			utils.EscapeHTML(s.Reference),
			utils.EscapeHTML(s.DepartmentName),
			utils.EscapeHTML(s.ClassName),
			utils.FormatDateTime(s.CreatedAt),
		))
	}

	return c.Send(b.String(), utils.MainMenu(c.Lang, c.IsAdmin))
}
