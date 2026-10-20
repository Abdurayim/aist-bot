package services

import (
	"database/sql"
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"aist-bot/internal/config"
	"aist-bot/internal/i18n"
	"aist-bot/internal/repository"
	"aist-bot/internal/state"
)

// BotService is the single dependency handlers receive. It owns the bot
// connection, the repositories and the services built on top of them.
type BotService struct {
	Bot    *tgbotapi.BotAPI
	Config *config.Config

	Users       *repository.UserRepository
	Classes     *repository.ClassRepository
	Departments *repository.DepartmentRepository
	Submissions *repository.SubmissionRepository
	Admins      *repository.AdminRepository

	State    *state.Manager
	Telegram *TelegramService
	Document *DocumentService
	Appeals  *AppealService
}

// NewBotService connects to Telegram and wires everything together.
func NewBotService(cfg *config.Config, db *sql.DB) (*BotService, error) {
	bot, err := tgbotapi.NewBotAPI(cfg.Bot.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Telegram: %w", err)
	}

	return NewBotServiceWithAPI(cfg, db, bot), nil
}

// NewBotServiceWithAPI wires the service around an existing bot connection.
// Tests use it to drive the whole bot against a stub Telegram server.
func NewBotServiceWithAPI(cfg *config.Config, db *sql.DB, bot *tgbotapi.BotAPI) *BotService {
	s := &BotService{
		Bot:         bot,
		Config:      cfg,
		Users:       repository.NewUserRepository(db),
		Classes:     repository.NewClassRepository(db),
		Departments: repository.NewDepartmentRepository(db),
		Submissions: repository.NewSubmissionRepository(db),
		Admins:      repository.NewAdminRepository(db),
		State:       state.NewManager(db),
		Telegram:    NewTelegramService(bot),
		Document:    NewDocumentService(bot, cfg.OrgName),
	}

	s.Appeals = NewAppealService(s)

	return s
}

// SyncAdmins makes the admins table match ADMIN_PHONES. It runs at every
// startup, so changing .env and restarting is the whole admin workflow.
//
// It also connects any admin who had already registered as an ordinary
// parent before their number was added to the list. Without this they would
// be reported as unreachable until they happened to message the bot again,
// which reads as a broken setup when it is not.
func (s *BotService) SyncAdmins() error {
	if err := s.Admins.SyncFromConfig(s.Config.Admin.PhoneNumbers); err != nil {
		return err
	}

	for _, phone := range s.Config.Admin.PhoneNumbers {
		user, err := s.Users.GetByPhone(phone)
		if err != nil {
			log.Printf("warning: failed to look up admin %s: %v", phone, err)
			continue
		}

		if user == nil {
			continue
		}

		if err := s.Admins.LinkTelegramID(phone, user.TelegramID); err != nil {
			log.Printf("warning: failed to connect admin %s: %v", phone, err)
		}
	}

	return nil
}

// IsAdmin reports whether this person is an admin, and links their chat id
// to the matching admin row the first time they are recognised, so appeals
// can be delivered to them from then on.
func (s *BotService) IsAdmin(phone string, telegramID int64) bool {
	if phone != "" {
		if s.Config.Admin.IsAdminPhone(phone) {
			// Linking is best effort: a failure here must not lock an admin
			// out of the panel, it only delays notifications until their
			// next visit.
			_ = s.Admins.LinkTelegramID(phone, telegramID)
			return true
		}

		// This phone number is not on the list. If the chat is still
		// holding an admin seat claimed with an earlier number, release it.
		// The phone number in ADMIN_PHONES is the only authority, and a
		// stale link would keep delivering appeals to someone who is no
		// longer an admin — and keep letting them into the panel through
		// the telegram_id match below.
		if released, err := s.Admins.UnlinkTelegramID(telegramID); err != nil {
			log.Printf("warning: failed to release admin seat for %d: %v", telegramID, err)
		} else if released {
			log.Printf("released the admin seat held by %d: %s is not in ADMIN_PHONES", telegramID, phone)
		}

		return false
	}

	isAdmin, err := s.Admins.IsAdmin(phone, telegramID)
	if err != nil {
		return false
	}

	return isAdmin
}

// SetWebhook points Telegram at the public URL from the configuration.
func (s *BotService) SetWebhook(url string) error {
	webhook, err := tgbotapi.NewWebhook(url)
	if err != nil {
		return fmt.Errorf("failed to build webhook: %w", err)
	}

	if _, err := s.Bot.Request(webhook); err != nil {
		return fmt.Errorf("failed to set webhook: %w", err)
	}

	return nil
}

// RemoveWebhook switches Telegram back to long polling.
func (s *BotService) RemoveWebhook() error {
	if _, err := s.Bot.Request(tgbotapi.DeleteWebhookConfig{DropPendingUpdates: false}); err != nil {
		return fmt.Errorf("failed to remove webhook: %w", err)
	}
	return nil
}

// RegisterCommands publishes the command list Telegram shows behind the
// "Menu" button next to the message box, in each supported language. It is
// the only in-app signpost a parent has before they type anything, so it is
// refreshed at every startup.
func (s *BotService) RegisterCommands() error {
	commands := map[i18n.Language][]tgbotapi.BotCommand{
		i18n.Uzbek: {
			{Command: "start", Description: "Boshlash / Asosiy menyu"},
			{Command: "help", Description: "Bot qanday ishlaydi?"},
			{Command: "cancel", Description: "Joriy murojaatni bekor qilish"},
		},
		i18n.Russian: {
			{Command: "start", Description: "Начать / Главное меню"},
			{Command: "help", Description: "Как работает бот?"},
			{Command: "cancel", Description: "Отменить текущее обращение"},
		},
		i18n.English: {
			{Command: "start", Description: "Start / Main menu"},
			{Command: "help", Description: "How does the bot work?"},
			{Command: "cancel", Description: "Cancel the current message"},
		},
	}

	// The Uzbek list is also the default, shown to anyone whose Telegram
	// language is none of the three.
	config := tgbotapi.NewSetMyCommands(commands[i18n.Uzbek]...)
	if _, err := s.Bot.Request(config); err != nil {
		return fmt.Errorf("failed to set default commands: %w", err)
	}

	for lang, list := range commands {
		scoped := tgbotapi.NewSetMyCommandsWithScopeAndLanguage(
			tgbotapi.NewBotCommandScopeDefault(), lang.String(), list...,
		)

		if _, err := s.Bot.Request(scoped); err != nil {
			return fmt.Errorf("failed to set %s commands: %w", lang, err)
		}
	}

	return nil
}
