package models

import (
	"strings"
	"time"
	"unicode"
)

// Department is a topic area an appeal can be addressed to. Departments are
// stored in the database rather than hardcoded, because admins add, rename,
// hide and delete them from the admin panel.
type Department struct {
	ID        int64     `json:"id" db:"id"`
	Code      string    `json:"code" db:"code"`
	Emoji     string    `json:"emoji" db:"emoji"`
	NameUz    string    `json:"name_uz" db:"name_uz"`
	NameRu    string    `json:"name_ru" db:"name_ru"`
	NameEn    string    `json:"name_en" db:"name_en"`
	SortOrder int       `json:"sort_order" db:"sort_order"`
	IsActive  bool      `json:"is_active" db:"is_active"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// DefaultDepartmentEmoji is used when an admin adds a department without one.
const DefaultDepartmentEmoji = "📌"

// NameFor returns the department name in the given language code, falling
// back to Uzbek when a translation was left empty.
func (d *Department) NameFor(lang string) string {
	var name string

	switch lang {
	case "ru":
		name = d.NameRu
	case "en":
		name = d.NameEn
	default:
		name = d.NameUz
	}

	if strings.TrimSpace(name) == "" {
		return d.NameUz
	}

	return name
}

// LabelFor returns the emoji + name pair shown on buttons and in summaries.
func (d *Department) LabelFor(lang string) string {
	emoji := d.Emoji
	if emoji == "" {
		emoji = DefaultDepartmentEmoji
	}
	return emoji + " " + d.NameFor(lang)
}

// SplitLeadingEmoji separates an emoji typed at the start of a department
// name from the name itself, so an admin can simply type "🍽 Oshxona".
// The returned emoji is empty when the text starts with an ordinary letter.
func SplitLeadingEmoji(text string) (emoji, name string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", ""
	}

	var emojiRunes []rune
	rest := text

	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			break
		}
		if unicode.IsSpace(r) {
			break
		}
		emojiRunes = append(emojiRunes, r)
	}

	if len(emojiRunes) > 0 {
		prefix := string(emojiRunes)
		rest = strings.TrimSpace(strings.TrimPrefix(text, prefix))

		// A leading "-" or "." is punctuation, not an icon; and if nothing is
		// left after stripping, the whole text was the name.
		if rest == "" || isPlainPunctuation(prefix) {
			return "", text
		}

		return prefix, rest
	}

	return "", rest
}

func isPlainPunctuation(s string) bool {
	for _, r := range s {
		if r > 0x2000 {
			return false
		}
	}
	return true
}
