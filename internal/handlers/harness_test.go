package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "github.com/mattn/go-sqlite3"

	"aist-bot/internal/config"
	"aist-bot/internal/database"
	"aist-bot/internal/services"
)

// sentMessage is one outgoing Bot API call, captured so tests can assert on
// what the parent or the admin actually saw.
type sentMessage struct {
	Method   string
	ChatID   string
	Text     string
	Caption  string
	Markup   string
	Filename string
	FileSize int
}

// Body returns whichever of text or caption the call carried.
func (m sentMessage) Body() string {
	if m.Text != "" {
		return m.Text
	}
	return m.Caption
}

// fakeTelegram is a stand-in Bot API server. It records every call and
// answers with the minimum the client library needs, which lets the whole
// bot run end to end in a test without touching the network.
type fakeTelegram struct {
	server *httptest.Server

	mu        sync.Mutex
	sent      []sentMessage
	messageID int
	// failDocuments makes every sendDocument call fail, standing in for a
	// Telegram outage while a parent is submitting.
	failDocuments bool
	// failChats makes every send to these chats fail, standing in for an
	// admin who has blocked the bot or never started it.
	failChats map[string]bool
}

// FailChat makes every delivery to one chat fail from now on.
func (f *fakeTelegram) FailChat(chatID int64) {
	f.mu.Lock()
	if f.failChats == nil {
		f.failChats = map[string]bool{}
	}
	f.failChats[fmt.Sprintf("%d", chatID)] = true
	f.mu.Unlock()
}

// FailDocuments makes document uploads fail from now on.
func (f *fakeTelegram) FailDocuments(fail bool) {
	f.mu.Lock()
	f.failDocuments = fail
	f.mu.Unlock()
}

func newFakeTelegram(t *testing.T) *fakeTelegram {
	t.Helper()

	f := &fakeTelegram{messageID: 1000}

	f.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method := filepath.Base(r.URL.Path)
		params := parseParams(r)

		f.mu.Lock()
		f.messageID++
		messageID := f.messageID
		failDocuments := f.failDocuments
		chatBlocked := f.failChats[params["chat_id"]]

		switch method {
		case "sendMessage", "sendDocument", "editMessageText", "sendPhoto":
			f.sent = append(f.sent, sentMessage{
				Method:   method,
				ChatID:   params["chat_id"],
				Text:     params["text"],
				Caption:  params["caption"],
				Markup:   params["reply_markup"],
				Filename: params["__filename"],
				FileSize: len(params["__filebytes"]),
			})
		}
		f.mu.Unlock()

		if chatBlocked && (method == "sendDocument" || method == "sendMessage") {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"ok":          false,
				"error_code":  403,
				"description": "Forbidden: bot was blocked by the user",
			})
			return
		}

		if method == "sendDocument" && failDocuments {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"ok":          false,
				"error_code":  400,
				"description": "simulated upload failure",
			})
			return
		}

		switch method {
		case "getMe":
			writeResult(w, tgbotapi.User{ID: 7, IsBot: true, UserName: "aist_test_bot"})

		case "sendDocument":
			writeResult(w, map[string]interface{}{
				"message_id": messageID,
				"date":       1,
				"chat":       map[string]interface{}{"id": 1, "type": "private"},
				"document": map[string]interface{}{
					"file_id":        fmt.Sprintf("pdf_file_%d", messageID),
					"file_unique_id": fmt.Sprintf("u%d", messageID),
					"file_name":      params["__filename"],
				},
			})

		case "sendMessage", "editMessageText", "sendPhoto":
			writeResult(w, map[string]interface{}{
				"message_id": messageID,
				"date":       1,
				"chat":       map[string]interface{}{"id": 1, "type": "private"},
				"text":       params["text"],
			})

		default:
			writeResult(w, true)
		}
	}))

	t.Cleanup(f.server.Close)

	return f
}

// parseParams reads both form-encoded and multipart requests, flattening the
// uploaded file into __filename and __filebytes.
func parseParams(r *http.Request) map[string]string {
	params := map[string]string{}

	contentType := r.Header.Get("Content-Type")

	if strings.HasPrefix(contentType, "multipart/") {
		_, boundaryParams, err := mime.ParseMediaType(contentType)
		if err != nil {
			return params
		}

		reader := multipart.NewReader(r.Body, boundaryParams["boundary"])

		for {
			part, err := reader.NextPart()
			if err != nil {
				break
			}

			data, _ := io.ReadAll(part)

			if part.FileName() != "" {
				params["__filename"] = part.FileName()
				params["__filebytes"] = string(data)
				continue
			}

			params[part.FormName()] = string(data)
		}

		return params
	}

	_ = r.ParseForm()
	for key := range r.Form {
		params[key] = r.Form.Get(key)
	}

	return params
}

func writeResult(w http.ResponseWriter, result interface{}) {
	encoded, _ := json.Marshal(result)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":     true,
		"result": json.RawMessage(encoded),
	})
}

// Sent returns everything captured so far.
func (f *fakeTelegram) Sent() []sentMessage {
	f.mu.Lock()
	defer f.mu.Unlock()

	out := make([]sentMessage, len(f.sent))
	copy(out, f.sent)

	return out
}

// Last returns the most recent captured call.
func (f *fakeTelegram) Last() sentMessage {
	sent := f.Sent()
	if len(sent) == 0 {
		return sentMessage{}
	}

	return sent[len(sent)-1]
}

// Reset clears the recorded calls between steps of a test.
func (f *fakeTelegram) Reset() {
	f.mu.Lock()
	f.sent = nil
	f.mu.Unlock()
}

// ToChat returns the calls addressed to one chat id.
func (f *fakeTelegram) ToChat(chatID int64) []sentMessage {
	want := fmt.Sprintf("%d", chatID)

	var out []sentMessage
	for _, m := range f.Sent() {
		if m.ChatID == want {
			out = append(out, m)
		}
	}

	return out
}

// testBot is a fully wired bot backed by a temporary database and the fake
// Telegram server.
type testBot struct {
	*services.BotService

	tg *fakeTelegram
	db *sql.DB
}

func newTestBot(t *testing.T, adminPhones ...string) *testBot {
	t.Helper()

	tg := newFakeTelegram(t)

	api, err := tgbotapi.NewBotAPIWithAPIEndpoint("test-token", tg.server.URL+"/bot%s/%s")
	if err != nil {
		t.Fatalf("failed to create stub bot: %v", err)
	}

	dbPath := filepath.Join(t.TempDir(), "test.db")

	cfg := &config.Config{
		Bot:      config.BotConfig{Token: "test-token"},
		Database: config.DatabaseConfig{Path: dbPath},
		Admin:    config.AdminConfig{PhoneNumbers: config.ParseAdminPhones(strings.Join(adminPhones, ","))},
		OrgName:  "AIST School",
	}

	if err := database.Connect(&cfg.Database); err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	db := database.DB
	t.Cleanup(func() { _ = db.Close() })

	service := services.NewBotServiceWithAPI(cfg, db, api)

	if err := service.SyncAdmins(); err != nil {
		t.Fatalf("failed to sync admins: %v", err)
	}

	// Photos are never fetched from the real Telegram in tests.
	service.Document.SetPhotoFetcher(func(string) ([]byte, string, error) {
		return testJPEG(), "JPG", nil
	})

	return &testBot{BotService: service, tg: tg, db: db}
}

// restart rebuilds the services against the same database file, dropping
// every in-memory cache exactly as a redeploy would.
func (b *testBot) restart(t *testing.T) {
	t.Helper()

	if err := database.Migrate(); err != nil {
		t.Fatalf("migrations failed on restart: %v", err)
	}

	service := services.NewBotServiceWithAPI(b.BotService.Config, b.db, b.BotService.Bot)

	if err := service.SyncAdmins(); err != nil {
		t.Fatalf("SyncAdmins failed on restart: %v", err)
	}

	service.Document.SetPhotoFetcher(func(string) ([]byte, string, error) {
		return testJPEG(), "JPG", nil
	})

	b.BotService = service
}

// ── Update builders ──────────────────────────────────────────────────

func (b *testBot) sendText(chatID int64, from tgbotapi.User, text string) {
	HandleUpdate(b.BotService, tgbotapi.Update{
		Message: &tgbotapi.Message{
			MessageID: 1,
			From:      &from,
			Chat:      &tgbotapi.Chat{ID: chatID, Type: "private"},
			Text:      text,
		},
	})
}

func (b *testBot) sendContact(chatID int64, from tgbotapi.User, phone string) {
	HandleUpdate(b.BotService, tgbotapi.Update{
		Message: &tgbotapi.Message{
			MessageID: 1,
			From:      &from,
			Chat:      &tgbotapi.Chat{ID: chatID, Type: "private"},
			Contact: &tgbotapi.Contact{
				PhoneNumber: phone,
				UserID:      from.ID,
			},
		},
	})
}

func (b *testBot) sendPhoto(chatID int64, from tgbotapi.User, fileID string) {
	HandleUpdate(b.BotService, tgbotapi.Update{
		Message: &tgbotapi.Message{
			MessageID: 1,
			From:      &from,
			Chat:      &tgbotapi.Chat{ID: chatID, Type: "private"},
			Photo: []tgbotapi.PhotoSize{
				{FileID: fileID + "_small", Width: 90, Height: 60},
				{FileID: fileID, FileUniqueID: fileID, Width: 1200, Height: 800, FileSize: 40000},
			},
		},
	})
}

func (b *testBot) tap(chatID int64, from tgbotapi.User, data string) {
	HandleUpdate(b.BotService, tgbotapi.Update{
		CallbackQuery: &tgbotapi.CallbackQuery{
			ID:   "cb1",
			From: &from,
			Message: &tgbotapi.Message{
				MessageID: 500,
				Chat:      &tgbotapi.Chat{ID: chatID, Type: "private"},
			},
			Data: data,
		},
	})
}

// callbackFor finds the callback_data of the button whose label contains
// the given text, in the most recent keyboard sent to the chat. Tests tap
// what a parent would see rather than hardcoding callback strings.
func (b *testBot) callbackFor(t *testing.T, label string) string {
	t.Helper()

	sent := b.tg.Sent()

	for i := len(sent) - 1; i >= 0; i-- {
		if sent[i].Markup == "" {
			continue
		}

		var markup struct {
			Inline [][]struct {
				Text string `json:"text"`
				Data string `json:"callback_data"`
			} `json:"inline_keyboard"`
		}

		if err := json.Unmarshal([]byte(sent[i].Markup), &markup); err != nil {
			continue
		}

		for _, row := range markup.Inline {
			for _, button := range row {
				if strings.Contains(button.Text, label) {
					return button.Data
				}
			}
		}
	}

	t.Fatalf("no button containing %q was offered; last markup: %s", label, b.tg.Last().Markup)
	return ""
}

// lastMarkupHas reports whether the most recent keyboard sent to this chat
// carries a button with the given label. Unlike hasButton it ignores older
// messages, which is what "the menu on screen right now" means.
func (b *testBot) lastMarkupHas(chatID int64, label string) bool {
	messages := b.tg.ToChat(chatID)

	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Markup == "" {
			continue
		}
		return strings.Contains(messages[i].Markup, label)
	}

	return false
}

// hasButton reports whether a button with this label is on offer.
func (b *testBot) hasButton(label string) bool {
	for _, m := range b.tg.Sent() {
		if strings.Contains(m.Markup, label) {
			return true
		}
	}
	return false
}

func user(id int64, username string) tgbotapi.User {
	return tgbotapi.User{ID: id, UserName: username, IsBot: false}
}

// assertContains fails the test when none of the captured messages contains
// the expected text.
func assertContains(t *testing.T, tg *fakeTelegram, want string) {
	t.Helper()

	for _, m := range tg.Sent() {
		if strings.Contains(m.Body(), want) {
			return
		}
	}

	t.Fatalf("expected a message containing %q\ngot:\n%s", want, dump(tg.Sent()))
}

func dump(messages []sentMessage) string {
	var b strings.Builder

	for _, m := range messages {
		b.WriteString(fmt.Sprintf("  [%s -> %s] %s\n", m.Method, m.ChatID, strings.ReplaceAll(m.Body(), "\n", " ⏎ ")))
	}

	if b.Len() == 0 {
		return "  (nothing was sent)"
	}

	return b.String()
}
