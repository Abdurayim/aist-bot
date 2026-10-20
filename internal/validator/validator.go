package validator

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// Limits applied to free-text input.
const (
	MinTextLength      = 10
	MaxTextLength      = 3000
	MinChildNameLength = 3
	MaxChildNameLength = 80
	MaxClassLength     = 20
	MinDeptNameLength  = 2
	MaxDeptNameLength  = 60
)

var phonePattern = regexp.MustCompile(`^\+998[0-9]{9}$`)

// NormalizePhone strips formatting and adds the +998 prefix when the user
// typed a bare national number.
func NormalizePhone(phone string) string {
	replacer := strings.NewReplacer(" ", "", "-", "", "(", "", ")", "", ".", "")
	cleaned := replacer.Replace(strings.TrimSpace(phone))

	if cleaned == "" {
		return ""
	}

	switch {
	case strings.HasPrefix(cleaned, "+"):
		// already international
	case strings.HasPrefix(cleaned, "998") && len(cleaned) == 12:
		cleaned = "+" + cleaned
	case len(cleaned) == 9:
		cleaned = "+998" + cleaned
	default:
		cleaned = "+" + cleaned
	}

	return cleaned
}

// ValidatePhone accepts an Uzbek mobile number in any common spelling and
// returns it normalised. Numbers shared through Telegram's contact button
// are accepted as they arrive, since Telegram already verified them.
func ValidatePhone(phone string) (string, error) {
	cleaned := NormalizePhone(phone)

	if !phonePattern.MatchString(cleaned) {
		return "", fmt.Errorf("invalid phone number")
	}

	return cleaned, nil
}

// ValidateText checks the body of an appeal and returns it trimmed.
// The returned counts let the caller build a translated error message.
func ValidateText(text string) (string, int, error) {
	cleaned := strings.TrimSpace(text)
	length := len([]rune(cleaned))

	if length < MinTextLength {
		return "", length, fmt.Errorf("text too short")
	}

	if length > MaxTextLength {
		return "", length, fmt.Errorf("text too long")
	}

	return cleaned, length, nil
}

// ValidateChildName checks that the name looks like a real name rather than
// a stray tap, and returns it with whitespace collapsed.
func ValidateChildName(name string) (string, error) {
	cleaned := strings.Join(strings.Fields(strings.TrimSpace(name)), " ")
	length := len([]rune(cleaned))

	if length < MinChildNameLength || length > MaxChildNameLength {
		return "", fmt.Errorf("invalid child name")
	}

	letters := 0
	for _, r := range cleaned {
		if unicode.IsLetter(r) {
			letters++
		}
	}

	if letters < MinChildNameLength {
		return "", fmt.Errorf("invalid child name")
	}

	return cleaned, nil
}

// ValidateClassName checks an admin-entered class name.
func ValidateClassName(name string) (string, error) {
	cleaned := strings.Join(strings.Fields(strings.TrimSpace(name)), " ")
	length := len([]rune(cleaned))

	if length < 1 || length > MaxClassLength {
		return "", fmt.Errorf("invalid class name")
	}

	return cleaned, nil
}

// ValidateDepartmentName checks an admin-entered department name.
func ValidateDepartmentName(name string) (string, error) {
	cleaned := strings.Join(strings.Fields(strings.TrimSpace(name)), " ")
	length := len([]rune(cleaned))

	if length < MinDeptNameLength || length > MaxDeptNameLength {
		return "", fmt.Errorf("invalid department name")
	}

	return cleaned, nil
}

// SanitizeFilename makes a string safe to use inside a filename.
func SanitizeFilename(text string) string {
	var b strings.Builder

	for _, r := range text {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-':
			// Hyphens are kept so a document number such as
			// AIST-2026-000042 stays readable in the filename.
			b.WriteRune('-')
		case r == ' ' || r == '_':
			b.WriteRune('_')
		}
	}

	cleaned := strings.Trim(b.String(), "_-")
	for strings.Contains(cleaned, "__") {
		cleaned = strings.ReplaceAll(cleaned, "__", "_")
	}

	if cleaned == "" {
		return "murojaat"
	}

	if len([]rune(cleaned)) > 40 {
		cleaned = string([]rune(cleaned)[:40])
	}

	return cleaned
}
