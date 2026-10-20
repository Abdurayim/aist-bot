package services

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// TelegramService wraps the handful of Bot API calls this bot makes, so
// handlers stay readable and every message goes out with the same settings.
type TelegramService struct {
	bot *tgbotapi.BotAPI
}

func NewTelegramService(bot *tgbotapi.BotAPI) *TelegramService {
	return &TelegramService{bot: bot}
}

// Send sends an HTML message, optionally with a keyboard.
func (s *TelegramService) Send(chatID int64, text string, keyboard interface{}) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.DisableWebPagePreview = true

	if keyboard != nil {
		msg.ReplyMarkup = keyboard
	}

	if _, err := s.bot.Send(msg); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

// SendAndGetID is Send, returning the id of the message just sent.
func (s *TelegramService) SendAndGetID(chatID int64, text string, keyboard interface{}) (int, error) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.DisableWebPagePreview = true

	if keyboard != nil {
		msg.ReplyMarkup = keyboard
	}

	sent, err := s.bot.Send(msg)
	if err != nil {
		return 0, fmt.Errorf("failed to send message: %w", err)
	}

	return sent.MessageID, nil
}

// Edit replaces the text and keyboard of an existing message. Editing in
// place keeps menus from piling up in the chat, which matters a lot on the
// small screens most parents use.
func (s *TelegramService) Edit(chatID int64, messageID int, text string, keyboard *tgbotapi.InlineKeyboardMarkup) error {
	var msg tgbotapi.EditMessageTextConfig

	if keyboard != nil {
		msg = tgbotapi.NewEditMessageTextAndMarkup(chatID, messageID, text, *keyboard)
	} else {
		msg = tgbotapi.NewEditMessageText(chatID, messageID, text)
	}

	msg.ParseMode = tgbotapi.ModeHTML
	msg.DisableWebPagePreview = true

	if _, err := s.bot.Send(msg); err != nil {
		return fmt.Errorf("failed to edit message: %w", err)
	}

	return nil
}

// AnswerCallback acknowledges an inline button tap, which stops the button
// from showing a loading spinner.
func (s *TelegramService) AnswerCallback(callbackID, text string) error {
	if _, err := s.bot.Request(tgbotapi.NewCallback(callbackID, text)); err != nil {
		return fmt.Errorf("failed to answer callback: %w", err)
	}
	return nil
}

// DeleteMessage removes a message, ignoring the common case where it is
// already gone.
func (s *TelegramService) DeleteMessage(chatID int64, messageID int) {
	_, _ = s.bot.Request(tgbotapi.NewDeleteMessage(chatID, messageID))
}

// UploadDocument sends a document built in memory and returns the file_id
// Telegram assigned to it, which is how the same file is later forwarded to
// admins without re-uploading or storing it on our side.
func (s *TelegramService) UploadDocument(chatID int64, filename string, data []byte, caption string, keyboard interface{}) (string, error) {
	doc := tgbotapi.NewDocument(chatID, tgbotapi.FileBytes{Name: filename, Bytes: data})
	doc.Caption = caption
	doc.ParseMode = tgbotapi.ModeHTML

	if keyboard != nil {
		doc.ReplyMarkup = keyboard
	}

	sent, err := s.bot.Send(doc)
	if err != nil {
		return "", fmt.Errorf("failed to upload document: %w", err)
	}

	if sent.Document == nil {
		return "", fmt.Errorf("telegram returned no document")
	}

	return sent.Document.FileID, nil
}

// SendDocumentByFileID forwards an already uploaded document.
func (s *TelegramService) SendDocumentByFileID(chatID int64, fileID, caption string) error {
	doc := tgbotapi.NewDocument(chatID, tgbotapi.FileID(fileID))
	doc.Caption = caption
	doc.ParseMode = tgbotapi.ModeHTML

	if _, err := s.bot.Send(doc); err != nil {
		return fmt.Errorf("failed to send document: %w", err)
	}

	return nil
}

// SendChatAction shows "sending document..." while a PDF is being built.
func (s *TelegramService) SendChatAction(chatID int64, action string) {
	_, _ = s.bot.Request(tgbotapi.NewChatAction(chatID, action))
}
