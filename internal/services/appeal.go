package services

import (
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"aist-bot/internal/i18n"
	"aist-bot/internal/models"
	"aist-bot/internal/utils"
)

// AppealService turns a completed form into a stored appeal, a PDF and a
// notification to every admin.
type AppealService struct {
	bot *BotService
}

func NewAppealService(bot *BotService) *AppealService {
	return &AppealService{bot: bot}
}

// SubmitResult reports what happened, so the handler can tell the parent
// whether their appeal actually reached anyone.
type SubmitResult struct {
	Submission     *models.Submission
	AdminsNotified int
	AdminsLinked   int
	// DocumentSent is false when the PDF could not be produced or uploaded
	// and the appeal was delivered as plain text instead.
	DocumentSent bool
}

// Submit stores the appeal, renders its PDF, sends a copy to the parent and
// forwards it to every admin who has connected to the bot.
//
// The order matters. The appeal is written to the database first, so a
// failure while rendering or sending can never lose what a parent wrote.
// And if the document cannot be produced at all, the appeal is still
// delivered to the admins as text rather than quietly disappearing.
func (s *AppealService) Submit(user *models.User, data *models.StateData, chatID int64) (*SubmitResult, error) {
	lang := i18n.Parse(user.Language)

	submission, err := s.bot.Submissions.Create(&models.Submission{
		UserID:         user.ID,
		Type:           data.Type,
		DepartmentCode: data.DepartmentCode,
		DepartmentName: data.DepartmentName,
		ClassName:      data.ClassName,
		ChildName:      data.ChildName,
		Text:           data.Text,
		Language:       user.Language,
	}, data.Photos)
	if err != nil {
		return nil, fmt.Errorf("failed to save appeal: %w", err)
	}

	fileID := s.buildAndDeliverDocument(submission, user, data, chatID, lang)

	// The submitter has just received their own copy, so they are skipped
	// here even when they are an admin themselves — otherwise an admin
	// writing about their own child gets the same document twice.
	notified, linked := s.notifyAdmins(submission, user, fileID, chatID)

	return &SubmitResult{
		Submission:     submission,
		AdminsNotified: notified,
		AdminsLinked:   linked,
		DocumentSent:   fileID != "",
	}, nil
}

// buildAndDeliverDocument renders the PDF and sends the parent their copy.
// It returns the Telegram file_id of the uploaded document, or an empty
// string when the document could not be produced — the caller then falls
// back to delivering the appeal as text.
func (s *AppealService) buildAndDeliverDocument(
	submission *models.Submission,
	user *models.User,
	data *models.StateData,
	chatID int64,
	lang i18n.Language,
) string {
	s.bot.Telegram.SendChatAction(chatID, tgbotapi.ChatUploadDocument)

	pdf, err := s.bot.Document.Generate(AppealDocument{
		Type:           submission.Type,
		Reference:      submission.Reference,
		DepartmentName: submission.DepartmentName,
		ClassName:      submission.ClassName,
		ChildName:      submission.ChildName,
		ParentPhone:    user.PhoneNumber,
		ParentUsername: user.TelegramUsername,
		Text:           submission.Text,
		Photos:         data.Photos,
		Language:       lang,
		CreatedAt:      submission.CreatedAt,
	})
	if err != nil {
		log.Printf("error: failed to render document for %s: %v", submission.Reference, err)
		return ""
	}

	filename := utils.PDFFilename(submission.Type, submission.ChildName, submission.Reference)

	// The parent's own copy doubles as the upload: Telegram hands back a
	// file_id that lets the same document be forwarded to every admin
	// without uploading it again.
	fileID, err := s.bot.Telegram.UploadDocument(chatID, filename, pdf, s.parentCaption(submission, lang), nil)
	if err != nil {
		log.Printf("error: failed to deliver document for %s: %v", submission.Reference, err)
		return ""
	}

	if err := s.bot.Submissions.AttachPDF(submission.ID, fileID, filename); err != nil {
		log.Printf("warning: failed to record pdf for appeal %s: %v", submission.Reference, err)
	}

	submission.PDFFileID = fileID
	submission.PDFFilename = filename

	return fileID
}

// notifyAdmins forwards the document to every admin who has shared their
// contact, except skipChatID, which has already been sent a copy. One admin
// listing several phone numbers receives a single copy, because the
// repository returns distinct chat ids.
//
// linked counts every connected admin including the skipped one, so the
// caller can tell "nobody is connected" apart from "the only admin is the
// person who just wrote in".
func (s *AppealService) notifyAdmins(
	submission *models.Submission,
	user *models.User,
	fileID string,
	skipChatID int64,
) (notified, linked int) {
	chatIDs, err := s.bot.Admins.GetLinkedTelegramIDs()
	if err != nil {
		log.Printf("warning: failed to list admin chats: %v", err)
		return 0, 0
	}

	linked = len(chatIDs)

	for _, chatID := range chatIDs {
		if chatID == skipChatID {
			continue
		}

		// Admins read the notification in their own language where we know
		// it, otherwise in the language the parent wrote in.
		lang := i18n.Parse(submission.Language)
		if admin, err := s.bot.Users.GetByTelegramID(chatID); err == nil && admin != nil {
			lang = i18n.Parse(admin.Language)
		}

		caption := s.adminCaption(submission, user, lang)

		var err error
		if fileID != "" {
			err = s.bot.Telegram.SendDocumentByFileID(chatID, fileID, caption)
		} else {
			// No document could be produced, so the appeal itself goes in
			// the message body. Losing it is not an option.
			err = s.bot.Telegram.Send(chatID,
				caption+"\n\n📝 <b>"+utils.EscapeHTML(i18n.Get(i18n.LblText, lang))+":</b>\n"+
					utils.EscapeHTML(utils.Truncate(submission.Text, 2500)), nil)
		}

		if err != nil {
			log.Printf("warning: failed to notify admin %d about %s: %v", chatID, submission.Reference, err)
			continue
		}

		notified++
	}

	return notified, linked
}

// parentCaption is the short summary attached to the parent's own copy.
func (s *AppealService) parentCaption(submission *models.Submission, lang i18n.Language) string {
	return fmt.Sprintf("%s <b>%s</b>\n%s: <code>%s</code>",
		i18n.TypeEmoji(string(submission.Type)),
		utils.EscapeHTML(i18n.Get(i18n.TypeKey(string(submission.Type)), lang)),
		utils.EscapeHTML(i18n.Get(i18n.LblReference, lang)),
		utils.EscapeHTML(submission.Reference),
	)
}

// adminCaption is the fuller summary admins see above the document, so they
// can triage without opening the PDF.
func (s *AppealService) adminCaption(submission *models.Submission, user *models.User, lang i18n.Language) string {
	username := user.UsernameOrDash()

	return fmt.Sprintf(
		"%s\n\n"+
			"%s %s\n"+
			"%s: <code>%s</code>\n\n"+
			"📂 %s: <b>%s</b>\n"+
			"🏫 %s: <b>%s</b>\n"+
			"👶 %s: <b>%s</b>\n"+
			"📱 %s: <code>%s</code>\n"+
			"💬 %s: %s\n"+
			"🕒 %s: %s\n"+
			"📷 %s: %d",
		i18n.Get(i18n.MsgNewAppealHeader, lang),
		i18n.TypeEmoji(string(submission.Type)),
		"<b>"+utils.EscapeHTML(i18n.Get(i18n.TypeKey(string(submission.Type)), lang))+"</b>",
		utils.EscapeHTML(i18n.Get(i18n.LblReference, lang)), utils.EscapeHTML(submission.Reference),
		utils.EscapeHTML(i18n.Get(i18n.LblDepartment, lang)), utils.EscapeHTML(submission.DepartmentName),
		utils.EscapeHTML(i18n.Get(i18n.LblClass, lang)), utils.EscapeHTML(submission.ClassName),
		utils.EscapeHTML(i18n.Get(i18n.LblChild, lang)), utils.EscapeHTML(submission.ChildName),
		utils.EscapeHTML(i18n.Get(i18n.LblPhone, lang)), utils.EscapeHTML(utils.FormatPhone(user.PhoneNumber)),
		utils.EscapeHTML(i18n.Get(i18n.LblUsername, lang)), utils.EscapeHTML(username),
		utils.EscapeHTML(i18n.Get(i18n.LblDate, lang)), utils.FormatDateTime(submission.CreatedAt),
		utils.EscapeHTML(i18n.Get(i18n.LblPhotos, lang)), submission.PhotoCount,
	)
}

// ResendDocument sends a stored appeal's PDF again, used by the admin panel.
func (s *AppealService) ResendDocument(chatID int64, submission *models.SubmissionWithUser, lang i18n.Language) error {
	if submission.PDFFileID == "" {
		return fmt.Errorf("appeal %s has no document", submission.Reference)
	}

	user := &models.User{
		PhoneNumber:      submission.PhoneNumber,
		TelegramUsername: submission.TelegramUsername,
	}

	return s.bot.Telegram.SendDocumentByFileID(chatID, submission.PDFFileID,
		s.adminCaption(&submission.Submission, user, lang))
}

// CleanupStates drops conversation states abandoned more than a day ago.
func (s *AppealService) CleanupStates() {
	if err := s.bot.State.CleanOlderThan(24); err != nil {
		log.Printf("warning: failed to clean old states: %v", err)
	}
}
