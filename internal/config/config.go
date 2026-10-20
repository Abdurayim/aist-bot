package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds every runtime setting of the bot.
type Config struct {
	Bot      BotConfig
	Database DatabaseConfig
	Server   ServerConfig
	Admin    AdminConfig
	OrgName  string
}

type BotConfig struct {
	Token      string
	WebhookURL string
}

type DatabaseConfig struct {
	Path string
}

type ServerConfig struct {
	Port string
}

// AdminConfig holds the admin phone numbers read from .env.
// There is no upper limit: 3, 4 or more admins are allowed, and a single
// person may register several of their own numbers.
type AdminConfig struct {
	PhoneNumbers []string
}

// Load reads configuration from .env and the process environment.
func Load() (*Config, error) {
	_ = godotenv.Load(".env")

	cfg := &Config{
		Bot: BotConfig{
			Token:      getEnv("BOT_TOKEN", ""),
			WebhookURL: strings.TrimSpace(getEnv("WEBHOOK_URL", "")),
		},
		Database: DatabaseConfig{
			Path: getEnv("DB_PATH", "./aist_bot.db"),
		},
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "8083"),
		},
		Admin: AdminConfig{
			PhoneNumbers: ParseAdminPhones(getEnv("ADMIN_PHONES", "")),
		},
		OrgName: getEnv("ORG_NAME", "AIST School"),
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.Bot.Token == "" {
		return fmt.Errorf("BOT_TOKEN is required (get one from @BotFather)")
	}

	if len(c.Admin.PhoneNumbers) == 0 {
		return fmt.Errorf("ADMIN_PHONES is required: list at least one admin phone number")
	}

	for _, phone := range c.Admin.PhoneNumbers {
		if !strings.HasPrefix(phone, "+") {
			return fmt.Errorf("admin phone %q must be in international format, e.g. +998901234567", phone)
		}
	}

	return nil
}

// IsAdminPhone reports whether the given (already normalised) phone number
// belongs to an admin.
func (c *AdminConfig) IsAdminPhone(phone string) bool {
	for _, adminPhone := range c.PhoneNumbers {
		if adminPhone == phone {
			return true
		}
	}
	return false
}

// ParseAdminPhones splits a comma separated phone list and normalises each
// entry. Duplicates, spaces, dashes and parentheses are removed.
func ParseAdminPhones(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{}
	}

	// Allow both comma and semicolon as separators.
	raw = strings.ReplaceAll(raw, ";", ",")

	seen := make(map[string]bool)
	result := make([]string, 0)

	for _, part := range strings.Split(raw, ",") {
		phone := normalizePhone(part)
		if phone == "" || seen[phone] {
			continue
		}
		seen[phone] = true
		result = append(result, phone)
	}

	return result
}

func normalizePhone(phone string) string {
	replacer := strings.NewReplacer(" ", "", "-", "", "(", "", ")", "", "\t", "")
	cleaned := replacer.Replace(strings.TrimSpace(phone))

	if cleaned == "" {
		return ""
	}

	// Accept 998901234567 and 901234567 as well as +998901234567.
	if !strings.HasPrefix(cleaned, "+") {
		switch {
		case strings.HasPrefix(cleaned, "998") && len(cleaned) == 12:
			cleaned = "+" + cleaned
		case len(cleaned) == 9:
			cleaned = "+998" + cleaned
		default:
			cleaned = "+" + cleaned
		}
	}

	return cleaned
}

func getEnv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		// Strip inline comments such as `polling  # or webhook`.
		if idx := strings.Index(value, " #"); idx != -1 {
			value = strings.TrimSpace(value[:idx])
		}
		return value
	}
	return fallback
}
