package handlers

import (
	"strconv"
	"strings"
	"sync"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"aist-bot/internal/i18n"
	"aist-bot/internal/models"
	"aist-bot/internal/utils"
)

// TestForwardedContactIsRejected: sharing someone else's contact card must
// not register this account under that person's phone number.
func TestForwardedContactIsRejected(t *testing.T) {
	b := newTestBot(t, adminPhone)

	b.sendText(parent.ID, parent, "/start")
	b.tap(parent.ID, parent, "lang:uz")

	b.tg.Reset()
	HandleUpdate(b.BotService, tgbotapi.Update{
		Message: &tgbotapi.Message{
			MessageID: 1,
			From:      &parent,
			Chat:      &tgbotapi.Chat{ID: parent.ID, Type: "private"},
			Contact: &tgbotapi.Contact{
				PhoneNumber: adminPhone,
				UserID:      999999, // somebody else's contact card
			},
		},
	})

	if stored, _ := b.Users.GetByTelegramID(parent.ID); stored != nil {
		t.Fatalf("a forwarded contact registered the account as %q", stored.PhoneNumber)
	}

	assertContains(t, b.tg, "Raqam noto‘g‘ri")
}

// TestHTMLInUserTextIsEscaped: Telegram parses HTML in captions, so a name
// or complaint containing tags must not break the message.
func TestHTMLInUserTextIsEscaped(t *testing.T) {
	b := newTestBot(t, adminPhone)
	seedSchool(t, b)
	registerParent(t, b, i18n.Uzbek, admin, adminPhone)
	registerParent(t, b, i18n.Uzbek, parent, parentPhone)

	b.sendText(parent.ID, parent, i18n.Get(i18n.BtnComplaint, i18n.Uzbek))
	b.tap(parent.ID, parent, b.callbackFor(t, "Oshxona"))
	b.tap(parent.ID, parent, b.callbackFor(t, "9-B"))
	b.sendText(parent.ID, parent, "Ali <b>Vali</b>")

	b.sendText(parent.ID, parent, "Ovqat <script>alert(1)</script> sovuq & achchiq edi.")
	skip := b.callbackFor(t, "davom etish")

	b.tg.Reset()
	b.tap(parent.ID, parent, skip)

	// The confirmation summary must show the text inert, not as markup.
	var summary string
	for _, m := range b.tg.Sent() {
		if strings.Contains(m.Body(), "script") {
			summary = m.Body()
		}
	}

	if summary == "" {
		t.Fatalf("the complaint text never appeared:\n%s", dump(b.tg.Sent()))
	}

	if strings.Contains(summary, "<script>") {
		t.Fatalf("raw HTML reached the message body:\n%s", summary)
	}

	if !strings.Contains(summary, "&lt;script&gt;") {
		t.Fatalf("the text was not escaped:\n%s", summary)
	}
}

// TestAppealWithHTMLReachesTheAdminIntact drives the same through to
// delivery, where the caption is also HTML.
func TestAppealWithHTMLReachesTheAdminIntact(t *testing.T) {
	b := newTestBot(t, adminPhone)
	seedSchool(t, b)
	registerParent(t, b, i18n.Uzbek, admin, adminPhone)
	registerParent(t, b, i18n.Uzbek, parent, parentPhone)

	b.sendText(parent.ID, parent, i18n.Get(i18n.BtnComplaint, i18n.Uzbek))
	b.tap(parent.ID, parent, b.callbackFor(t, "Oshxona"))
	b.tap(parent.ID, parent, b.callbackFor(t, "9-B"))
	b.sendText(parent.ID, parent, "Ali <b>Vali</b>")
	b.sendText(parent.ID, parent, "Ovqat & nonushta <i>sovuq</i> edi, tekshiring.")
	b.tap(parent.ID, parent, b.callbackFor(t, "davom etish"))

	b.tg.Reset()
	b.tap(parent.ID, parent, "send")

	docs := documentsTo(b, admin.ID)
	if len(docs) != 1 {
		t.Fatalf("the admin got %d documents", len(docs))
	}

	if strings.Contains(docs[0].Caption, "<b>Vali</b>") {
		t.Fatalf("unescaped user HTML reached the admin caption:\n%s", docs[0].Caption)
	}

	if !strings.Contains(docs[0].Caption, "&lt;b&gt;Vali&lt;/b&gt;") {
		t.Fatalf("the child's name was not escaped:\n%s", docs[0].Caption)
	}
}

// TestUsernameIsKeptCurrent: people change their @username, and the stored
// one is printed on documents.
func TestUsernameIsKeptCurrent(t *testing.T) {
	b := newTestBot(t, adminPhone)
	registerParent(t, b, i18n.Uzbek, parent, parentPhone)

	renamed := user(parent.ID, "yangi_nom")
	b.sendText(renamed.ID, renamed, "salom")

	stored, _ := b.Users.GetByTelegramID(parent.ID)
	if stored.TelegramUsername != "yangi_nom" {
		t.Fatalf("stored username = %q, want the new one", stored.TelegramUsername)
	}
}

// TestUserWithNoUsernameIsHandled: plenty of parents have no @username.
func TestUserWithNoUsernameIsHandled(t *testing.T) {
	b := newTestBot(t, adminPhone)
	seedSchool(t, b)
	registerParent(t, b, i18n.Uzbek, admin, adminPhone)

	anonymous := user(555, "")
	registerParent(t, b, i18n.Uzbek, anonymous, "+998935550000")

	b.sendText(anonymous.ID, anonymous, i18n.Get(i18n.BtnComplaint, i18n.Uzbek))
	b.tap(anonymous.ID, anonymous, b.callbackFor(t, "Oshxona"))
	b.tap(anonymous.ID, anonymous, b.callbackFor(t, "9-B"))
	b.sendText(anonymous.ID, anonymous, "Aliyev Sardor")
	b.sendText(anonymous.ID, anonymous, "Ovqat sovuq berilmoqda, iltimos tekshiring.")
	b.tap(anonymous.ID, anonymous, b.callbackFor(t, "davom etish"))

	b.tg.Reset()
	b.tap(anonymous.ID, anonymous, "send")

	docs := documentsTo(b, admin.ID)
	if len(docs) != 1 {
		t.Fatalf("the admin got %d documents:\n%s", len(docs), dump(b.tg.Sent()))
	}

	if !strings.Contains(docs[0].Caption, "—") {
		t.Fatalf("a missing username should read as a dash:\n%s", docs[0].Caption)
	}
}

// TestDeletedClassMidFlowDoesNotCrash: an admin may delete a class while a
// parent is on the class step.
func TestDeletedClassMidFlowDoesNotCrash(t *testing.T) {
	b := newTestBot(t, adminPhone)
	seedSchool(t, b)
	registerParent(t, b, i18n.Uzbek, parent, parentPhone)

	b.sendText(parent.ID, parent, i18n.Get(i18n.BtnComplaint, i18n.Uzbek))
	b.tap(parent.ID, parent, b.callbackFor(t, "Oshxona"))

	chosen := b.callbackFor(t, "9-B")

	classes, _ := b.Classes.GetAll()
	if err := b.Classes.Delete(classes[0].ID); err != nil {
		t.Fatalf("failed to delete class: %v", err)
	}

	b.tg.Reset()
	b.tap(parent.ID, parent, chosen)

	// With no classes left the parent is told so and sent home, not left
	// staring at a step that cannot be answered.
	assertContains(t, b.tg, "sinflar ro‘yxati kiritilmagan")
}

// TestStateSurvivesARestart: SQLite-backed state means a redeploy must not
// strand a parent halfway through a form.
func TestStateSurvivesARestart(t *testing.T) {
	b := newTestBot(t, adminPhone)
	seedSchool(t, b)
	registerParent(t, b, i18n.Uzbek, parent, parentPhone)

	b.sendText(parent.ID, parent, i18n.Get(i18n.BtnComplaint, i18n.Uzbek))
	b.tap(parent.ID, parent, b.callbackFor(t, "Oshxona"))
	b.tap(parent.ID, parent, b.callbackFor(t, "9-B"))
	b.sendText(parent.ID, parent, "Aliyev Sardor")

	// A restart loses every in-memory cache but keeps the database.
	b.restart(t)

	b.tg.Reset()
	b.sendText(parent.ID, parent, "Ovqat sovuq berilmoqda, iltimos tekshiring.")
	b.tap(parent.ID, parent, b.callbackFor(t, "davom etish"))
	b.tap(parent.ID, parent, "send")

	stored, _ := b.Submissions.GetRecent(10)
	if len(stored) != 1 {
		t.Fatalf("the form did not survive a restart: %d appeals filed", len(stored))
	}

	if stored[0].ChildName != "Aliyev Sardor" {
		t.Fatalf("answers given before the restart were lost: %+v", stored[0])
	}
}

// TestMigrationsAreIdempotent: the bot restarts often, and migration 002
// adds a column, which fails if applied twice.
func TestMigrationsAreIdempotent(t *testing.T) {
	b := newTestBot(t, adminPhone)
	registerParent(t, b, i18n.Uzbek, parent, parentPhone)

	for i := 0; i < 3; i++ {
		b.restart(t)
	}

	stored, _ := b.Users.GetByTelegramID(parent.ID)
	if stored == nil || stored.PhoneNumber != parentPhone {
		t.Fatal("data did not survive repeated migrations")
	}
}

// TestUnknownCallbackIsIgnored: buttons from an older version of the bot
// are still sitting in people's chats.
func TestUnknownCallbackIsIgnored(t *testing.T) {
	b := newTestBot(t, adminPhone)
	registerParent(t, b, i18n.Uzbek, parent, parentPhone)

	for _, data := range []string{"", "garbage", "dept:", "dept:abc", "class:999999", "a_dep_one:abc", "send"} {
		b.tap(parent.ID, parent, data)
	}

	// Nothing crashed and nothing was filed.
	stored, _ := b.Submissions.GetRecent(10)
	if len(stored) != 0 {
		t.Fatalf("stray callbacks filed %d appeals", len(stored))
	}
}

// TestNonTextMessagesDoNotDerailAForm: stickers, voice notes and locations
// arrive with no Text at all.
func TestNonTextMessagesDoNotDerailAForm(t *testing.T) {
	b := newTestBot(t, adminPhone)
	seedSchool(t, b)
	registerParent(t, b, i18n.Uzbek, parent, parentPhone)

	b.sendText(parent.ID, parent, i18n.Get(i18n.BtnComplaint, i18n.Uzbek))
	b.tap(parent.ID, parent, b.callbackFor(t, "Oshxona"))
	b.tap(parent.ID, parent, b.callbackFor(t, "9-B"))

	// A voice note where a name was expected.
	HandleUpdate(b.BotService, tgbotapi.Update{
		Message: &tgbotapi.Message{
			MessageID: 1,
			From:      &parent,
			Chat:      &tgbotapi.Chat{ID: parent.ID, Type: "private"},
			Voice:     &tgbotapi.Voice{FileID: "voice1", Duration: 3},
		},
	})

	state, _, _ := b.State.Get(parent.ID)
	if state != models.StateAwaitingChildName {
		t.Fatalf("a voice note moved the form to %q", state)
	}

	// And the form still completes afterwards.
	b.sendText(parent.ID, parent, "Aliyev Sardor")

	state, _, _ = b.State.Get(parent.ID)
	if state != models.StateAwaitingText {
		t.Fatalf("the form could not continue: state = %q", state)
	}
}

// TestPhotoSentAsAFileIsAccepted: some phones send images as documents.
func TestPhotoSentAsAFileIsAccepted(t *testing.T) {
	b := newTestBot(t, adminPhone)
	seedSchool(t, b)
	registerParent(t, b, i18n.Uzbek, parent, parentPhone)

	b.sendText(parent.ID, parent, i18n.Get(i18n.BtnComplaint, i18n.Uzbek))
	b.tap(parent.ID, parent, b.callbackFor(t, "Oshxona"))
	b.tap(parent.ID, parent, b.callbackFor(t, "9-B"))
	b.sendText(parent.ID, parent, "Aliyev Sardor")
	b.sendText(parent.ID, parent, "Ovqat sovuq berilmoqda, iltimos tekshiring.")
	b.tap(parent.ID, parent, b.callbackFor(t, "rasm qo‘shaman"))

	HandleUpdate(b.BotService, tgbotapi.Update{
		Message: &tgbotapi.Message{
			MessageID: 1,
			From:      &parent,
			Chat:      &tgbotapi.Chat{ID: parent.ID, Type: "private"},
			Document: &tgbotapi.Document{
				FileID:   "doc_photo",
				MimeType: "image/jpeg",
				FileSize: 50000,
			},
		},
	})

	_, data, _ := b.State.Get(parent.ID)
	if len(data.Photos) != 1 {
		t.Fatalf("an image sent as a file was not accepted: %d photos", len(data.Photos))
	}

	// A PDF, however, is not a photo.
	HandleUpdate(b.BotService, tgbotapi.Update{
		Message: &tgbotapi.Message{
			MessageID: 1,
			From:      &parent,
			Chat:      &tgbotapi.Chat{ID: parent.ID, Type: "private"},
			Document: &tgbotapi.Document{
				FileID:   "doc_pdf",
				MimeType: "application/pdf",
				FileSize: 50000,
			},
		},
	})

	_, data, _ = b.State.Get(parent.ID)
	if len(data.Photos) != 1 {
		t.Fatalf("a PDF was accepted as a photo: %d photos", len(data.Photos))
	}
}

// TestLongTextIsNotTruncatedInTheRecord: Telegram captions cap at 1024
// characters, so the summary truncates for display — the stored appeal and
// the PDF must keep every word.
func TestLongTextIsNotTruncatedInTheRecord(t *testing.T) {
	b := newTestBot(t, adminPhone)
	seedSchool(t, b)
	registerParent(t, b, i18n.Uzbek, parent, parentPhone)

	long := strings.Repeat("Oshxonadagi ovqat sovuq berilmoqda. ", 70) // ~2500 chars

	b.sendText(parent.ID, parent, i18n.Get(i18n.BtnComplaint, i18n.Uzbek))
	b.tap(parent.ID, parent, b.callbackFor(t, "Oshxona"))
	b.tap(parent.ID, parent, b.callbackFor(t, "9-B"))
	b.sendText(parent.ID, parent, "Aliyev Sardor")
	b.sendText(parent.ID, parent, long)
	b.tap(parent.ID, parent, b.callbackFor(t, "davom etish"))
	b.tap(parent.ID, parent, "send")

	stored, _ := b.Submissions.GetRecent(1)
	if len(stored) != 1 {
		t.Fatal("the appeal was not filed")
	}

	if len([]rune(stored[0].Text)) != len([]rune(strings.TrimSpace(long))) {
		t.Fatalf("the stored text was truncated: %d runes stored, %d written",
			len([]rune(stored[0].Text)), len([]rune(strings.TrimSpace(long))))
	}
}

// TestCallbackDataFitsTelegramsLimit: Telegram rejects callback_data over
// 64 bytes, and department ids grow.
func TestCallbackDataFitsTelegramsLimit(t *testing.T) {
	payloads := []string{
		utils.Callback(utils.CBDept, 999999999),
		utils.Callback(utils.CBClass, 999999999),
		utils.Callback(utils.CBAdminDeptDelY, 999999999),
		utils.Callback(utils.CBAdminDoc, 9999999999),
		utils.CallbackStr(utils.CBBack, utils.StepPhotoAsk),
		utils.CallbackStr(utils.CBLanguage, "ru"),
	}

	for _, p := range payloads {
		if len(p) > 64 {
			t.Errorf("callback_data %q is %d bytes, over Telegram's 64-byte limit", p, len(p))
		}
	}
}

// TestConcurrentSendTapsFileOneAppeal: an impatient double tap arrives as
// two updates at once, each on its own goroutine.
func TestConcurrentSendTapsFileOneAppeal(t *testing.T) {
	b := newTestBot(t, adminPhone)
	seedSchool(t, b)
	registerParent(t, b, i18n.Uzbek, parent, parentPhone)

	b.sendText(parent.ID, parent, i18n.Get(i18n.BtnComplaint, i18n.Uzbek))
	b.tap(parent.ID, parent, b.callbackFor(t, "Oshxona"))
	b.tap(parent.ID, parent, b.callbackFor(t, "9-B"))
	b.sendText(parent.ID, parent, "Aliyev Sardor")
	b.sendText(parent.ID, parent, "Ovqat sovuq berilmoqda, iltimos tekshiring.")
	b.tap(parent.ID, parent, b.callbackFor(t, "davom etish"))

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b.tap(parent.ID, parent, "send")
		}()
	}
	wg.Wait()

	stored, _ := b.Submissions.GetRecent(10)
	if len(stored) != 1 {
		t.Fatalf("five concurrent Send taps filed %d appeals, want 1", len(stored))
	}
}

// TestConcurrentPhotosAreAllKept: an album arrives as several updates at
// once, and each one appends to the same list.
func TestConcurrentPhotosAreAllKept(t *testing.T) {
	b := newTestBot(t, adminPhone)
	seedSchool(t, b)
	registerParent(t, b, i18n.Uzbek, parent, parentPhone)

	b.sendText(parent.ID, parent, i18n.Get(i18n.BtnComplaint, i18n.Uzbek))
	b.tap(parent.ID, parent, b.callbackFor(t, "Oshxona"))
	b.tap(parent.ID, parent, b.callbackFor(t, "9-B"))
	b.sendText(parent.ID, parent, "Aliyev Sardor")
	b.sendText(parent.ID, parent, "Ovqat sovuq berilmoqda, iltimos tekshiring.")
	b.tap(parent.ID, parent, b.callbackFor(t, "rasm qo‘shaman"))

	var wg sync.WaitGroup
	for i := 0; i < models.MaxPhotos; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			b.sendPhoto(parent.ID, parent, "album_photo_"+strconv.Itoa(n))
		}(i)
	}
	wg.Wait()

	_, data, _ := b.State.Get(parent.ID)
	if len(data.Photos) != models.MaxPhotos {
		t.Fatalf("an album of %d photos was stored as %d — updates raced",
			models.MaxPhotos, len(data.Photos))
	}
}

// TestDifferentPeopleAreNotBlockedByEachOther: the per-person lock must not
// become a global one.
func TestDifferentPeopleAreNotBlockedByEachOther(t *testing.T) {
	b := newTestBot(t, adminPhone)
	seedSchool(t, b)

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()

			who := user(int64(700+n), "ota"+strconv.Itoa(n))
			b.sendText(who.ID, who, "/start")
			b.tap(who.ID, who, "lang:uz")
			b.sendContact(who.ID, who, "+99893555"+strconv.Itoa(1000+n))
		}(i)
	}
	wg.Wait()

	count, err := b.Users.Count()
	if err != nil {
		t.Fatalf("failed to count users: %v", err)
	}

	if count != 8 {
		t.Fatalf("%d of 8 concurrent registrations completed", count)
	}
}
