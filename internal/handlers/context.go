package handlers

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"aist-bot/internal/i18n"
	"aist-bot/internal/models"
	"aist-bot/internal/services"
)

// ctx bundles everything a handler needs about the person it is answering,
// resolved once per update instead of being re-read in every handler.
type ctx struct {
	bot *services.BotService

	ChatID     int64
	TelegramID int64
	Username   string

	// User is nil until registration completes.
	User    *models.User
	Lang    i18n.Language
	IsAdmin bool

	State string
	Data  *models.StateData

	// RoleChanged is set when this person's role differs from the one they
	// were last shown, because an admin edited ADMIN_PHONES and restarted.
	RoleChanged bool

	// MessageID is the inline message a callback came from, so menus can be
	// edited in place. It is zero for plain messages.
	MessageID int
	// CallbackID is set when the update is an inline button tap.
	CallbackID string
}

// newContext resolves the sender of an update.
func newContext(bot *services.BotService, chatID int64, from *tgbotapi.User) *ctx {
	c := &ctx{
		bot:        bot,
		ChatID:     chatID,
		TelegramID: from.ID,
		Username:   from.UserName,
		Lang:       i18n.Uzbek,
		Data:       &models.StateData{},
	}

	user, err := bot.Users.GetByTelegramID(from.ID)
	if err != nil {
		log.Printf("warning: failed to load user %d: %v", from.ID, err)
	}

	if user != nil {
		c.User = user
		c.Lang = i18n.Parse(user.Language)
		c.IsAdmin = bot.IsAdmin(user.PhoneNumber, user.TelegramID)

		// Telegram usernames change; keep the stored one current so the
		// contact details printed on documents stay correct.
		if user.TelegramUsername != from.UserName {
			if err := bot.Users.UpdateUsername(from.ID, from.UserName); err == nil {
				user.TelegramUsername = from.UserName
			}
		}
	}

	state, data, err := bot.State.Get(from.ID)
	if err != nil {
		log.Printf("warning: failed to load state for %d: %v", from.ID, err)
	}

	c.State = state
	if data != nil {
		c.Data = data
	}

	c.detectRoleChange()

	// Before registration the chosen language lives in the state, because
	// there is no user row to read it from yet.
	if c.User == nil && c.Data.Language != "" {
		c.Lang = i18n.Parse(c.Data.Language)
	}

	return c
}

// detectRoleChange compares the role this person actually holds against the
// one they were last shown, and records the new one.
//
// A promotion or demotion happens between messages, while the person is
// looking at a keyboard drawn for their old role — and possibly halfway
// through filling in a form that their new role is not allowed to send. So
// the change is noticed here, the stale form is thrown away, and the caller
// hands them their new menu before anything else is routed.
func (c *ctx) detectRoleChange() {
	if c.User == nil {
		return
	}

	role := models.RoleFor(c.IsAdmin)
	if c.User.Role == role {
		return
	}

	if err := c.bot.Users.UpdateRole(c.TelegramID, role); err != nil {
		log.Printf("warning: failed to record role for %d: %v", c.TelegramID, err)
		return
	}

	c.User.Role = role
	c.RoleChanged = true

	// Whatever they were in the middle of belongs to the old role.
	if c.State != "" {
		if err := c.bot.State.Clear(c.TelegramID); err != nil {
			log.Printf("warning: failed to clear state for %d: %v", c.TelegramID, err)
		}
		c.State = ""
		c.Data = &models.StateData{}
	}
}

// T translates a key into this person's language.
func (c *ctx) T(key string) string {
	return i18n.Get(key, c.Lang)
}

// Tf translates and formats.
func (c *ctx) Tf(key string, args ...interface{}) string {
	return i18n.Getf(key, c.Lang, args...)
}

// Send sends a message to this chat.
func (c *ctx) Send(text string, keyboard interface{}) error {
	return c.bot.Telegram.Send(c.ChatID, text, keyboard)
}

// Reply answers an inline button: it edits the message the button belongs to
// when possible, and falls back to a new message otherwise. Editing keeps
// the chat short, which matters on the small screens most parents use.
func (c *ctx) Reply(text string, keyboard *tgbotapi.InlineKeyboardMarkup) error {
	if c.MessageID != 0 {
		if err := c.bot.Telegram.Edit(c.ChatID, c.MessageID, text, keyboard); err == nil {
			return nil
		}
	}

	if keyboard != nil {
		return c.Send(text, *keyboard)
	}

	return c.Send(text, nil)
}

// Ack acknowledges an inline button tap.
func (c *ctx) Ack(text string) {
	if c.CallbackID != "" {
		_ = c.bot.Telegram.AnswerCallback(c.CallbackID, text)
	}
}

// SetState stores the conversation position.
func (c *ctx) SetState(state string) error {
	c.State = state
	return c.bot.State.Set(c.TelegramID, state, c.Data)
}

// SaveData persists the collected form data without changing the state.
func (c *ctx) SaveData() error {
	return c.bot.State.Set(c.TelegramID, c.State, c.Data)
}

// ClearState drops the conversation state entirely.
func (c *ctx) ClearState() error {
	c.State = ""
	c.Data = &models.StateData{}
	return c.bot.State.Clear(c.TelegramID)
}

// Fail reports an unexpected error to the user in their own language and
// logs the detail for the operator.
func (c *ctx) Fail(where string, err error) error {
	log.Printf("error in %s for %d: %v", where, c.TelegramID, err)
	return c.Send(c.T(i18n.ErrGeneric), nil)
}

// matchesButton reports whether text is the given button in any language.
// People switch language mid-conversation, and old keyboards linger in the
// chat, so a button must keep working whichever language it was drawn in.
func matchesButton(text, key string) bool {
	for _, lang := range i18n.All {
		if text == i18n.Get(key, lang) {
			return true
		}
	}
	return false
}
