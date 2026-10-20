package utils

import (
	"fmt"
	"strings"
	"time"

	"aist-bot/internal/models"
	"aist-bot/internal/validator"
)

// Tashkent is the timezone every date shown to users and printed on
// documents is rendered in, regardless of the server's own clock setting.
var Tashkent = mustLoadTashkent()

func mustLoadTashkent() *time.Location {
	if loc, err := time.LoadLocation("Asia/Tashkent"); err == nil {
		return loc
	}
	// Fall back to a fixed +05:00 offset when the host has no tzdata.
	return time.FixedZone("UTC+5", 5*60*60)
}

// FormatDateTime renders a timestamp as 15.09.2026 14:30.
func FormatDateTime(t time.Time) string {
	return t.In(Tashkent).Format("02.01.2006 15:04")
}

// FormatDate renders a timestamp as 15.09.2026.
func FormatDate(t time.Time) string {
	return t.In(Tashkent).Format("02.01.2006")
}

// PDFFilename builds the name of the generated document. It stays ASCII so
// the file downloads cleanly on every phone and desktop.
func PDFFilename(submissionType models.SubmissionType, childName, reference string) string {
	prefix := "Murojaat"
	switch submissionType {
	case models.TypeComplaint:
		prefix = "Shikoyat"
	case models.TypeProposal:
		prefix = "Taklif"
	case models.TypeGratitude:
		prefix = "Minnatdorchilik"
	}

	name := validator.SanitizeFilename(childName)
	ref := validator.SanitizeFilename(reference)

	return fmt.Sprintf("%s_%s_%s.pdf", prefix, name, ref)
}

// PDFSafe prepares text for the PDF. The DejaVu fonts cover Latin, Cyrillic
// and punctuation but not emoji, so unsupported runes are dropped and the
// several apostrophe shapes used in Uzbek are folded into one.
func PDFSafe(text string) string {
	var b strings.Builder

	for _, r := range text {
		switch r {
		case '\u02bb', '\u02bc', '\u2018', '\u2019', '\u00b4', '\u0060':
			b.WriteRune('\'')
			continue
		case '“', '”', '«', '»':
			b.WriteRune('"')
			continue
		case '\n', '\r', '\t':
			b.WriteRune(r)
			continue
		}

		switch {
		case r >= 0x0020 && r <= 0x007E, // Basic Latin
			r >= 0x00A0 && r <= 0x024F, // Latin-1 Supplement, Extended-A and -B
			r >= 0x0400 && r <= 0x04FF, // Cyrillic
			r >= 0x2010 && r <= 0x2027, // dashes and quotes
			r >= 0x2030 && r <= 0x205E, // misc punctuation
			r >= 0x20A0 && r <= 0x20BF: // currency
			b.WriteRune(r)
		default:
			// Emoji and other glyphs the font cannot draw are skipped.
		}
	}

	return strings.TrimSpace(b.String())
}

// Truncate shortens text to maxRunes characters, counting runes rather than
// bytes so Cyrillic is not cut mid-character.
func Truncate(text string, maxRunes int) string {
	runes := []rune(text)
	if len(runes) <= maxRunes {
		return text
	}
	return strings.TrimSpace(string(runes[:maxRunes])) + "…"
}

// EscapeHTML escapes the three characters Telegram's HTML parse mode treats
// as markup, so user-typed text can never break a message.
func EscapeHTML(text string) string {
	replacer := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return replacer.Replace(text)
}

// FormatPhone renders +998901234567 as +998 90 123 45 67.
func FormatPhone(phone string) string {
	if len(phone) != 13 || !strings.HasPrefix(phone, "+998") {
		return phone
	}

	return fmt.Sprintf("%s %s %s %s %s",
		phone[:4], phone[4:6], phone[6:9], phone[9:11], phone[11:])
}
