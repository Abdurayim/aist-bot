package handlers

import (
	"strings"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"aist-bot/internal/i18n"
	"aist-bot/internal/models"
)

const (
	adminPhone  = "+998901112233"
	parentPhone = "+998935556677"
)

var (
	parent = user(111, "sardor_ota")
	admin  = user(222, "direktor")
)

// registerParent walks a new user through language choice and phone sharing.
func registerParent(t *testing.T, b *testBot, lang i18n.Language, who tgbotapi.User, phone string) {
	t.Helper()

	b.sendText(who.ID, who, "/start")
	b.tap(who.ID, who, "lang:"+lang.String())
	b.sendContact(who.ID, who, phone)

	stored, err := b.Users.GetByTelegramID(who.ID)
	if err != nil || stored == nil {
		t.Fatalf("registration did not create a user: %v", err)
	}

	if stored.PhoneNumber != phone {
		t.Fatalf("stored phone = %q, want %q", stored.PhoneNumber, phone)
	}

	if stored.Language != lang.String() {
		t.Fatalf("stored language = %q, want %q", stored.Language, lang)
	}
}

// seedSchool gives the bot one class to choose from; departments are seeded
// by the migration.
func seedSchool(t *testing.T, b *testBot) {
	t.Helper()

	if _, err := b.Classes.Create("9-B"); err != nil {
		t.Fatalf("failed to seed class: %v", err)
	}
}

func TestRegistrationFlow(t *testing.T) {
	b := newTestBot(t, adminPhone)

	b.sendText(parent.ID, parent, "/start")

	// The very first prompt must be readable by everyone, so it repeats in
	// all three languages.
	first := b.tg.Last()
	for _, lang := range i18n.All {
		if !strings.Contains(first.Text, i18n.Get(i18n.MsgChooseLanguage, lang)) {
			t.Fatalf("language prompt is missing %s:\n%s", lang, first.Text)
		}
	}

	b.tap(parent.ID, parent, "lang:uz")
	assertContains(t, b.tg, "Raqamni yuborish")

	b.sendContact(parent.ID, parent, parentPhone)
	assertContains(t, b.tg, "ro‘yxatdan o‘tdingiz")

	// The home keyboard must offer all three kinds of appeal.
	for _, key := range []string{i18n.BtnComplaint, i18n.BtnProposal, i18n.BtnGratitude} {
		if !b.hasButton(i18n.Get(key, i18n.Uzbek)) {
			t.Fatalf("main menu is missing %q", i18n.Get(key, i18n.Uzbek))
		}
	}
}

func TestRegistrationRejectsAnInvalidTypedNumber(t *testing.T) {
	b := newTestBot(t, adminPhone)

	b.sendText(parent.ID, parent, "/start")
	b.tap(parent.ID, parent, "lang:uz")

	b.tg.Reset()
	b.sendText(parent.ID, parent, "12345")

	assertContains(t, b.tg, "Raqam noto‘g‘ri")

	if stored, _ := b.Users.GetByTelegramID(parent.ID); stored != nil {
		t.Fatal("an invalid number must not create a user")
	}
}

// TestFullComplaintFlow is the path almost every parent takes: five steps,
// a photo, and a PDF delivered to both the parent and the admin.
func TestFullComplaintFlow(t *testing.T) {
	b := newTestBot(t, adminPhone)
	seedSchool(t, b)

	// The admin registers first, so they are connected when the appeal lands.
	registerParent(t, b, i18n.Uzbek, admin, adminPhone)
	registerParent(t, b, i18n.Uzbek, parent, parentPhone)

	b.tg.Reset()

	// Step 1: department.
	b.sendText(parent.ID, parent, i18n.Get(i18n.BtnComplaint, i18n.Uzbek))
	assertContains(t, b.tg, "Bo‘lim")

	b.tap(parent.ID, parent, b.callbackFor(t, "Oshxona"))

	// Step 2: class.
	assertContains(t, b.tg, "Sinf")
	b.tap(parent.ID, parent, b.callbackFor(t, "9-B"))

	// Step 3: the child's name.
	assertContains(t, b.tg, "Farzandingizning")
	b.sendText(parent.ID, parent, "Aliyev Sardor")

	// Step 4: the complaint itself.
	assertContains(t, b.tg, "Shikoyat matni")
	b.sendText(parent.ID, parent, "Oshxonadagi ovqat sovuq berilmoqda, iltimos tekshiring.")

	// Step 5: the optional photo.
	assertContains(t, b.tg, "Rasm")
	b.tap(parent.ID, parent, b.callbackFor(t, "rasm qo‘shaman"))
	b.sendPhoto(parent.ID, parent, "photo_abc")
	assertContains(t, b.tg, "Rasm qabul qilindi (1/5)")

	b.tap(parent.ID, parent, b.callbackFor(t, "Tayyor"))

	// The summary repeats every answer before anything is sent.
	summary := b.tg.Last()
	for _, want := range []string{"Oshxona", "9-B", "Aliyev Sardor", "sovuq"} {
		if !strings.Contains(summary.Text, want) {
			t.Fatalf("the confirmation summary is missing %q:\n%s", want, summary.Text)
		}
	}

	b.tg.Reset()
	b.tap(parent.ID, parent, "send")

	// The appeal is stored with everything the form collected.
	stored, err := b.Submissions.GetRecent(10)
	if err != nil {
		t.Fatalf("failed to read submissions: %v", err)
	}

	if len(stored) != 1 {
		t.Fatalf("expected exactly one submission, got %d", len(stored))
	}

	appeal := stored[0]

	switch {
	case appeal.Type != models.TypeComplaint:
		t.Fatalf("type = %q, want complaint", appeal.Type)
	case appeal.DepartmentCode != "kitchen":
		t.Fatalf("department = %q, want kitchen", appeal.DepartmentCode)
	case appeal.DepartmentName != "Oshxona":
		t.Fatalf("department name = %q, want Oshxona", appeal.DepartmentName)
	case appeal.ClassName != "9-B":
		t.Fatalf("class = %q, want 9-B", appeal.ClassName)
	case appeal.ChildName != "Aliyev Sardor":
		t.Fatalf("child = %q", appeal.ChildName)
	case appeal.PhotoCount != 1:
		t.Fatalf("photo count = %d, want 1", appeal.PhotoCount)
	case appeal.PDFFileID == "":
		t.Fatal("the generated PDF was not recorded against the appeal")
	case !strings.HasPrefix(appeal.Reference, "AIST-"):
		t.Fatalf("reference = %q, want an AIST- prefix", appeal.Reference)
	}

	// The parent gets a copy of the document and a confirmation.
	parentDocs := documentsTo(b, parent.ID)
	if len(parentDocs) != 1 {
		t.Fatalf("expected the parent to receive one document, got %d:\n%s", len(parentDocs), dump(b.tg.Sent()))
	}

	if !strings.HasSuffix(parentDocs[0].Filename, ".pdf") {
		t.Fatalf("filename = %q, want a .pdf", parentDocs[0].Filename)
	}

	if parentDocs[0].FileSize < 2000 {
		t.Fatalf("the delivered PDF is only %d bytes", parentDocs[0].FileSize)
	}

	// The admin gets the same document with the triage details.
	adminDocs := documentsTo(b, admin.ID)
	if len(adminDocs) != 1 {
		t.Fatalf("expected the admin to receive one document, got %d:\n%s", len(adminDocs), dump(b.tg.Sent()))
	}

	caption := adminDocs[0].Caption
	for _, want := range []string{"YANGI MUROJAAT", "Oshxona", "9-B", "Aliyev Sardor", "+998 93 555 66 77", "@sardor_ota"} {
		if !strings.Contains(caption, want) {
			t.Fatalf("the admin caption is missing %q:\n%s", want, caption)
		}
	}

	assertContains(t, b.tg, appeal.Reference)
}

// documentsTo returns the documents delivered to one chat.
func documentsTo(b *testBot, chatID int64) []sentMessage {
	var docs []sentMessage

	for _, m := range b.tg.ToChat(chatID) {
		if m.Method == "sendDocument" {
			docs = append(docs, m)
		}
	}

	return docs
}

func TestProposalAndGratitudeUseTheirOwnWording(t *testing.T) {
	b := newTestBot(t, adminPhone)
	seedSchool(t, b)
	registerParent(t, b, i18n.Uzbek, parent, parentPhone)

	cases := []struct {
		button string
		prompt string
	}{
		{i18n.Get(i18n.BtnProposal, i18n.Uzbek), "Taklif matni"},
		{i18n.Get(i18n.BtnGratitude, i18n.Uzbek), "Minnatdorchilik matni"},
	}

	for _, c := range cases {
		b.tg.Reset()

		b.sendText(parent.ID, parent, c.button)
		b.tap(parent.ID, parent, b.callbackFor(t, "Mentorlar"))
		b.tap(parent.ID, parent, b.callbackFor(t, "9-B"))
		b.sendText(parent.ID, parent, "Aliyev Sardor")

		assertContains(t, b.tg, c.prompt)
	}
}

// TestPhotoIsOptional covers the path an older parent is most likely to
// take: skipping the attachment entirely.
func TestPhotoIsOptional(t *testing.T) {
	b := newTestBot(t, adminPhone)
	seedSchool(t, b)
	registerParent(t, b, i18n.Uzbek, parent, parentPhone)

	b.sendText(parent.ID, parent, i18n.Get(i18n.BtnGratitude, i18n.Uzbek))
	b.tap(parent.ID, parent, b.callbackFor(t, "Mentorlar"))
	b.tap(parent.ID, parent, b.callbackFor(t, "9-B"))
	b.sendText(parent.ID, parent, "Aliyev Sardor")
	b.sendText(parent.ID, parent, "Mentorimizga katta rahmat, juda yaxshi ishlamoqda.")

	b.tap(parent.ID, parent, b.callbackFor(t, "davom etish"))
	b.tap(parent.ID, parent, "send")

	stored, _ := b.Submissions.GetRecent(10)
	if len(stored) != 1 {
		t.Fatalf("expected one submission, got %d", len(stored))
	}

	if stored[0].PhotoCount != 0 {
		t.Fatalf("photo count = %d, want 0", stored[0].PhotoCount)
	}

	if stored[0].PDFFileID == "" {
		t.Fatal("a photoless appeal must still produce a document")
	}
}

// TestBackStepsThroughTheWholeFlow checks that every Back button lands on
// the previous question rather than dropping the parent out of the form.
func TestBackStepsThroughTheWholeFlow(t *testing.T) {
	b := newTestBot(t, adminPhone)
	seedSchool(t, b)
	registerParent(t, b, i18n.Uzbek, parent, parentPhone)

	b.sendText(parent.ID, parent, i18n.Get(i18n.BtnComplaint, i18n.Uzbek))
	b.tap(parent.ID, parent, b.callbackFor(t, "Kutubxona"))
	b.tap(parent.ID, parent, b.callbackFor(t, "9-B"))
	b.sendText(parent.ID, parent, "Aliyev Sardor")
	b.sendText(parent.ID, parent, "Kutubxonada kitoblar yetishmayapti, iltimos ko‘paytiring.")

	// Now at the photo question; walk all the way back to the department.
	steps := []struct {
		expect string
		state  string
	}{
		{"Shikoyat matni", models.StateAwaitingText},
		{"Farzandingizning", models.StateAwaitingChildName},
		{"Sinf", models.StateAwaitingClass},
		{"Bo‘lim", models.StateAwaitingDepartment},
	}

	for _, step := range steps {
		back := b.callbackFor(t, i18n.Get(i18n.BtnBack, i18n.Uzbek))

		b.tg.Reset()
		b.tap(parent.ID, parent, back)

		assertContains(t, b.tg, step.expect)

		state, _, err := b.State.Get(parent.ID)
		if err != nil {
			t.Fatalf("failed to read state: %v", err)
		}

		if state != step.state {
			t.Fatalf("after Back the state is %q, want %q", state, step.state)
		}
	}

	// Answers already given are still there, so going back and forward
	// again does not make the parent retype anything.
	_, data, _ := b.State.Get(parent.ID)
	if data.ChildName != "Aliyev Sardor" || data.Text == "" {
		t.Fatalf("going back lost earlier answers: %+v", data)
	}
}

func TestCancelReturnsToTheMainMenu(t *testing.T) {
	b := newTestBot(t, adminPhone)
	seedSchool(t, b)
	registerParent(t, b, i18n.Uzbek, parent, parentPhone)

	b.sendText(parent.ID, parent, i18n.Get(i18n.BtnComplaint, i18n.Uzbek))
	b.tap(parent.ID, parent, b.callbackFor(t, "Moliya"))

	b.tg.Reset()
	b.tap(parent.ID, parent, "cancel")

	state, _, _ := b.State.Get(parent.ID)
	if state != "" {
		t.Fatalf("state after cancel = %q, want it cleared", state)
	}

	stored, _ := b.Submissions.GetRecent(10)
	if len(stored) != 0 {
		t.Fatal("cancelling must not file an appeal")
	}
}

// TestSendIsIgnoredTwice guards the case where a parent taps Send again
// because the first tap seemed slow.
func TestSendIsIgnoredTwice(t *testing.T) {
	b := newTestBot(t, adminPhone)
	seedSchool(t, b)
	registerParent(t, b, i18n.Uzbek, parent, parentPhone)

	b.sendText(parent.ID, parent, i18n.Get(i18n.BtnComplaint, i18n.Uzbek))
	b.tap(parent.ID, parent, b.callbackFor(t, "Oshxona"))
	b.tap(parent.ID, parent, b.callbackFor(t, "9-B"))
	b.sendText(parent.ID, parent, "Aliyev Sardor")
	b.sendText(parent.ID, parent, "Ovqat sifati pasayib ketdi, iltimos e'tibor bering.")
	b.tap(parent.ID, parent, b.callbackFor(t, "davom etish"))

	b.tap(parent.ID, parent, "send")
	b.tap(parent.ID, parent, "send")

	stored, _ := b.Submissions.GetRecent(10)
	if len(stored) != 1 {
		t.Fatalf("a double tap filed %d appeals, want 1", len(stored))
	}
}

// TestTextLengthIsValidated keeps a stray "ok" from becoming a document.
func TestTextLengthIsValidated(t *testing.T) {
	b := newTestBot(t, adminPhone)
	seedSchool(t, b)
	registerParent(t, b, i18n.Uzbek, parent, parentPhone)

	b.sendText(parent.ID, parent, i18n.Get(i18n.BtnComplaint, i18n.Uzbek))
	b.tap(parent.ID, parent, b.callbackFor(t, "Oshxona"))
	b.tap(parent.ID, parent, b.callbackFor(t, "9-B"))
	b.sendText(parent.ID, parent, "Aliyev Sardor")

	b.tg.Reset()
	b.sendText(parent.ID, parent, "ok")
	assertContains(t, b.tg, "juda qisqa")

	b.tg.Reset()
	b.sendText(parent.ID, parent, strings.Repeat("a", 3100))
	assertContains(t, b.tg, "juda uzun")

	state, _, _ := b.State.Get(parent.ID)
	if state != models.StateAwaitingText {
		t.Fatalf("state = %q, want the bot to still be waiting for the text", state)
	}
}

// TestChildNameIsValidated rejects a stray tap as a name.
func TestChildNameIsValidated(t *testing.T) {
	b := newTestBot(t, adminPhone)
	seedSchool(t, b)
	registerParent(t, b, i18n.Uzbek, parent, parentPhone)

	b.sendText(parent.ID, parent, i18n.Get(i18n.BtnComplaint, i18n.Uzbek))
	b.tap(parent.ID, parent, b.callbackFor(t, "Oshxona"))
	b.tap(parent.ID, parent, b.callbackFor(t, "9-B"))

	b.tg.Reset()
	b.sendText(parent.ID, parent, ".")
	assertContains(t, b.tg, "juda qisqa yoki noto‘g‘ri")
}

// TestPhotoLimit stops at five attachments rather than growing forever.
func TestPhotoLimit(t *testing.T) {
	b := newTestBot(t, adminPhone)
	seedSchool(t, b)
	registerParent(t, b, i18n.Uzbek, parent, parentPhone)

	b.sendText(parent.ID, parent, i18n.Get(i18n.BtnComplaint, i18n.Uzbek))
	b.tap(parent.ID, parent, b.callbackFor(t, "Oshxona"))
	b.tap(parent.ID, parent, b.callbackFor(t, "9-B"))
	b.sendText(parent.ID, parent, "Aliyev Sardor")
	b.sendText(parent.ID, parent, "Oshxonada tozalik yetarli emas, iltimos tekshiring.")
	b.tap(parent.ID, parent, b.callbackFor(t, "rasm qo‘shaman"))

	for i := 0; i < models.MaxPhotos+2; i++ {
		b.sendPhoto(parent.ID, parent, "photo")
	}

	_, data, _ := b.State.Get(parent.ID)
	if len(data.Photos) != models.MaxPhotos {
		t.Fatalf("collected %d photos, want the limit of %d", len(data.Photos), models.MaxPhotos)
	}

	assertContains(t, b.tg, "maksimal")
}

// TestFlowStopsWhenNoClassExists tells the parent what is wrong instead of
// showing an empty keyboard.
func TestFlowStopsWhenNoClassExists(t *testing.T) {
	b := newTestBot(t, adminPhone)
	registerParent(t, b, i18n.Uzbek, parent, parentPhone)

	b.sendText(parent.ID, parent, i18n.Get(i18n.BtnComplaint, i18n.Uzbek))
	b.tap(parent.ID, parent, b.callbackFor(t, "Oshxona"))

	assertContains(t, b.tg, "sinflar ro‘yxati kiritilmagan")

	state, _, _ := b.State.Get(parent.ID)
	if state != "" {
		t.Fatalf("state = %q, want the parent returned to the menu", state)
	}
}

// TestAppealSurvivesADocumentFailure is the worst realistic case: Telegram
// or the renderer fails just as a parent sends. What they wrote must still
// reach the administration.
func TestAppealSurvivesADocumentFailure(t *testing.T) {
	b := newTestBot(t, adminPhone)
	seedSchool(t, b)
	registerParent(t, b, i18n.Uzbek, admin, adminPhone)
	registerParent(t, b, i18n.Uzbek, parent, parentPhone)

	// Telegram refuses the upload just as the parent taps Send.
	b.tg.FailDocuments(true)

	b.sendText(parent.ID, parent, i18n.Get(i18n.BtnComplaint, i18n.Uzbek))
	b.tap(parent.ID, parent, b.callbackFor(t, "Oshxona"))
	b.tap(parent.ID, parent, b.callbackFor(t, "9-B"))
	b.sendText(parent.ID, parent, "Aliyev Sardor")
	b.sendText(parent.ID, parent, "Oshxonada tozalik yetarli emas, iltimos tekshiring.")
	b.tap(parent.ID, parent, b.callbackFor(t, "rasm qo‘shaman"))
	b.sendPhoto(parent.ID, parent, "broken_photo")

	done := b.callbackFor(t, "Tayyor")

	b.tg.Reset()
	b.tap(parent.ID, parent, done)
	b.tap(parent.ID, parent, "send")

	// The appeal is stored whatever happened to the document.
	stored, _ := b.Submissions.GetRecent(10)
	if len(stored) != 1 {
		t.Fatalf("the appeal was lost: %d rows stored", len(stored))
	}

	// And the admin was told about it one way or another.
	var reachedAdmin bool
	for _, m := range b.tg.ToChat(admin.ID) {
		if strings.Contains(m.Body(), "YANGI MUROJAAT") {
			reachedAdmin = true
			break
		}
	}

	if !reachedAdmin {
		t.Fatalf("the admin was never told about the appeal:\n%s", dump(b.tg.Sent()))
	}

	// The text the parent wrote travels in the message body instead.
	var carriedTheText bool
	for _, m := range b.tg.ToChat(admin.ID) {
		if strings.Contains(m.Body(), "tozalik yetarli emas") {
			carriedTheText = true
			break
		}
	}

	if !carriedTheText {
		t.Fatalf("the appeal text did not reach the admin:\n%s", dump(b.tg.ToChat(admin.ID)))
	}

	// And the parent is told plainly what happened rather than just "sent".
	var parentTold bool
	for _, m := range b.tg.ToChat(parent.ID) {
		if strings.Contains(m.Body(), "Hujjatni tayyorlashda muammo") {
			parentTold = true
			break
		}
	}

	if !parentTold {
		t.Fatalf("the parent was not told the document failed:\n%s", dump(b.tg.ToChat(parent.ID)))
	}
}

// TestTappingAnotherTypeMidFormRestartsCleanly: the home keyboard stays
// visible during a form, and an unsure parent will tap it. Doing so should
// start that kind of message afresh rather than silently doing nothing.
func TestTappingAnotherTypeMidFormRestartsCleanly(t *testing.T) {
	b := newTestBot(t, adminPhone)
	seedSchool(t, b)
	registerParent(t, b, i18n.Uzbek, parent, parentPhone)

	b.sendText(parent.ID, parent, i18n.Get(i18n.BtnComplaint, i18n.Uzbek))
	b.tap(parent.ID, parent, b.callbackFor(t, "Oshxona"))
	b.tap(parent.ID, parent, b.callbackFor(t, "9-B"))
	b.sendText(parent.ID, parent, "Aliyev Sardor")

	// Half way through a complaint, they decide it is a thank-you instead.
	b.tg.Reset()
	b.sendText(parent.ID, parent, i18n.Get(i18n.BtnGratitude, i18n.Uzbek))

	assertContains(t, b.tg, "Bo‘lim")

	state, data, _ := b.State.Get(parent.ID)
	if state != models.StateAwaitingDepartment {
		t.Fatalf("state = %q, want the flow restarted at the department step", state)
	}

	if data.Type != models.TypeGratitude {
		t.Fatalf("type = %q, want gratitude", data.Type)
	}

	// Nothing from the abandoned complaint is carried over.
	if data.ChildName != "" || data.ClassName != "" || data.DepartmentCode != "" {
		t.Fatalf("the abandoned complaint leaked into the new message: %+v", data)
	}
}
