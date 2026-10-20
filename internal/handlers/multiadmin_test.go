package handlers

import (
	"strconv"
	"strings"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"aist-bot/internal/i18n"
)

// The school runs with three or four admins, and one person may list
// several of their own numbers. These tests cover that setup end to end.

type staff struct {
	user  tgbotapi.User
	phone string
}

// TestFourAdminsEachReceiveEveryAppeal is the school's actual setup.
func TestFourAdminsEachReceiveEveryAppeal(t *testing.T) {
	team := []staff{
		{user(401, "direktor"), "+998901112233"},
		{user(402, "orinbosar"), "+998935556677"},
		{user(403, "psixolog"), "+998977778899"},
		{user(404, "hisobchi"), "+998901234567"},
	}

	phones := make([]string, 0, len(team))
	for _, s := range team {
		phones = append(phones, s.phone)
	}

	b := newTestBot(t, phones...)
	seedSchool(t, b)

	for _, s := range team {
		registerParent(t, b, i18n.Uzbek, s.user, s.phone)
	}

	registerParent(t, b, i18n.Uzbek, parent, "+998935550001")

	b.tg.Reset()
	fileAppeal(t, b, "Oshxona", "Ovqat sovuq berilmoqda, iltimos tekshiring.")

	for _, s := range team {
		docs := documentsTo(b, s.user.ID)
		if len(docs) != 1 {
			t.Fatalf("%s received %d documents, want 1", s.user.UserName, len(docs))
		}

		if !strings.Contains(docs[0].Caption, "YANGI MUROJAAT") {
			t.Fatalf("%s got a document without the triage caption", s.user.UserName)
		}
	}

	// Every one of them can also open the panel.
	for _, s := range team {
		b.tg.Reset()
		b.sendText(s.user.ID, s.user, i18n.Get(i18n.BtnStats, i18n.Uzbek))
		assertContains(t, b.tg, "Statistika")
	}
}

// TestOnePersonWithFourNumbersGetsOneCopy: the same human may own several
// of the numbers in ADMIN_PHONES.
func TestOnePersonWithFourNumbersGetsOneCopy(t *testing.T) {
	phones := []string{"+998901112233", "+998935556677", "+998977778899", "+998901234567"}

	b := newTestBot(t, phones...)
	seedSchool(t, b)

	// They happen to register with the third of their numbers.
	registerParent(t, b, i18n.Uzbek, admin, phones[2])
	registerParent(t, b, i18n.Uzbek, parent, "+998935550001")

	b.tg.Reset()
	fileAppeal(t, b, "Mentorlar", "Mentor darsga kech kelmoqda, tekshiring.")

	if got := len(documentsTo(b, admin.ID)); got != 1 {
		t.Fatalf("one person with four listed numbers received %d copies, want 1", got)
	}
}

// TestABlockedAdminDoesNotStopTheOthers: if one admin blocked the bot, the
// delivery loop must carry on.
func TestABlockedAdminDoesNotStopTheOthers(t *testing.T) {
	blocked := user(501, "bloklagan")
	working := user(502, "ishlayotgan")

	b := newTestBot(t, "+998901112233", "+998935556677")
	seedSchool(t, b)

	registerParent(t, b, i18n.Uzbek, blocked, "+998901112233")
	registerParent(t, b, i18n.Uzbek, working, "+998935556677")
	registerParent(t, b, i18n.Uzbek, parent, "+998935550001")

	b.tg.FailChat(blocked.ID)

	b.tg.Reset()
	fileAppeal(t, b, "Moliya", "To‘lov kvitansiyasi kelmadi, tekshiring.")

	if got := len(documentsTo(b, working.ID)); got != 1 {
		t.Fatalf("a blocked admin stopped delivery: the working admin got %d documents", got)
	}

	// And the parent is still told it was sent.
	assertContains(t, b.tg, "Murojaatingiz yuborildi")

	// The appeal is on file regardless.
	stored, _ := b.Submissions.GetRecent(10)
	if len(stored) != 1 {
		t.Fatalf("the appeal was not stored: %d rows", len(stored))
	}
}

// TestUnregisteredAdminsDoNotBlockDelivery: admins are added to .env before
// they ever open the bot.
func TestUnregisteredAdminsDoNotBlockDelivery(t *testing.T) {
	b := newTestBot(t, "+998901112233", "+998935556677", "+998977778899")
	seedSchool(t, b)

	// Only one of the three has actually started the bot.
	registerParent(t, b, i18n.Uzbek, admin, "+998935556677")
	registerParent(t, b, i18n.Uzbek, parent, "+998935550001")

	b.tg.Reset()
	fileAppeal(t, b, "Kutubxona", "Kitoblar yetishmayapti, iltimos ko‘paytiring.")

	if got := len(documentsTo(b, admin.ID)); got != 1 {
		t.Fatalf("the connected admin received %d documents, want 1", got)
	}

	admins, _ := b.Admins.GetAll()
	var unlinked int
	for _, a := range admins {
		if !a.IsLinked() {
			unlinked++
		}
	}

	if unlinked != 2 {
		t.Fatalf("%d admins are unlinked, want 2", unlinked)
	}
}

// TestChangingToANonAdminNumberReleasesTheSeat: the phone number in
// ADMIN_PHONES is the only authority. A chat that claimed a seat with an
// old number must not keep it after moving to a number that is not listed.
func TestChangingToANonAdminNumberReleasesTheSeat(t *testing.T) {
	b := newTestBot(t, adminPhone)
	seedSchool(t, b)
	registerParent(t, b, i18n.Uzbek, admin, adminPhone)

	linked, _ := b.Admins.GetLinkedTelegramIDs()
	if len(linked) != 1 {
		t.Fatalf("setup failed: %d linked admins", len(linked))
	}

	// They re-register with a personal number that is not on the list.
	b.sendText(admin.ID, admin, "/start")
	b.tap(admin.ID, admin, "lang:uz")
	b.sendContact(admin.ID, admin, "+998935559999")

	linked, _ = b.Admins.GetLinkedTelegramIDs()
	for _, id := range linked {
		if id == admin.ID {
			t.Fatal("the admin seat was not released after moving to an unlisted number")
		}
	}

	// They are a parent now, in menu and in power.
	b.tg.Reset()
	b.sendText(admin.ID, admin, i18n.Get(i18n.BtnManageDepartments, i18n.Uzbek))

	if b.lastMarkupHas(admin.ID, i18n.Get(i18n.BtnAddDept, i18n.Uzbek)) {
		t.Fatal("an unlisted number still reaches the department editor")
	}
}

// TestKeepingOneOfSeveralOwnNumbersKeepsTheSeat is the opposite case, and
// must not be broken by the rule above.
func TestKeepingOneOfSeveralOwnNumbersKeepsTheSeat(t *testing.T) {
	first, second := "+998901112233", "+998935556677"

	b := newTestBot(t, first, second)
	seedSchool(t, b)
	registerParent(t, b, i18n.Uzbek, admin, first)

	// They switch to their other listed number.
	b.sendText(admin.ID, admin, "/start")
	b.tap(admin.ID, admin, "lang:uz")
	b.sendContact(admin.ID, admin, second)

	linked, _ := b.Admins.GetLinkedTelegramIDs()
	if len(linked) != 1 || linked[0] != admin.ID {
		t.Fatalf("switching between two listed numbers lost the seat: %v", linked)
	}

	// Still exactly one copy, not one per number.
	registerParent(t, b, i18n.Uzbek, parent, "+998935550001")

	b.tg.Reset()
	fileAppeal(t, b, "Oshxona", "Ovqat sovuq berilmoqda, iltimos tekshiring.")

	if got := len(documentsTo(b, admin.ID)); got != 1 {
		t.Fatalf("received %d copies after switching numbers, want 1", got)
	}
}

// TestAdminsReadNotificationsInTheirOwnLanguage: a school may have admins
// working in different languages.
func TestAdminsReadNotificationsInTheirOwnLanguage(t *testing.T) {
	uzbekAdmin := user(601, "uz_admin")
	russianAdmin := user(602, "ru_admin")

	b := newTestBot(t, "+998901112233", "+998935556677")
	seedSchool(t, b)

	registerParent(t, b, i18n.Uzbek, uzbekAdmin, "+998901112233")
	registerParent(t, b, i18n.Russian, russianAdmin, "+998935556677")
	registerParent(t, b, i18n.English, parent, "+998935550001")

	b.tg.Reset()

	// The parent writes in English.
	b.sendText(parent.ID, parent, i18n.Get(i18n.BtnComplaint, i18n.English))
	b.tap(parent.ID, parent, b.callbackFor(t, "Canteen"))
	b.tap(parent.ID, parent, b.callbackFor(t, "9-B"))
	b.sendText(parent.ID, parent, "Sardor Aliyev")
	b.sendText(parent.ID, parent, "The food in the canteen is served cold, please check.")
	b.tap(parent.ID, parent, b.callbackFor(t, "No, continue"))
	b.tap(parent.ID, parent, "send")

	uzDocs := documentsTo(b, uzbekAdmin.ID)
	ruDocs := documentsTo(b, russianAdmin.ID)

	if len(uzDocs) != 1 || len(ruDocs) != 1 {
		t.Fatalf("delivery counts: uz=%d ru=%d, want 1 each", len(uzDocs), len(ruDocs))
	}

	if !strings.Contains(uzDocs[0].Caption, "YANGI MUROJAAT") {
		t.Fatalf("the Uzbek admin's caption is not in Uzbek:\n%s", uzDocs[0].Caption)
	}

	if !strings.Contains(ruDocs[0].Caption, "НОВОЕ ОБРАЩЕНИЕ") {
		t.Fatalf("the Russian admin's caption is not in Russian:\n%s", ruDocs[0].Caption)
	}
}

// TestManyAdminsAllGetDelivered guards the delivery loop at a size no
// school will exceed.
func TestManyAdminsAllGetDelivered(t *testing.T) {
	const count = 10

	phones := make([]string, 0, count)
	people := make([]tgbotapi.User, 0, count)

	for i := 0; i < count; i++ {
		phones = append(phones, "+99890111"+strconv.Itoa(1000+i))
		people = append(people, user(int64(900+i), "admin"+strconv.Itoa(i)))
	}

	b := newTestBot(t, phones...)
	seedSchool(t, b)

	for i, who := range people {
		registerParent(t, b, i18n.Uzbek, who, phones[i])
	}

	registerParent(t, b, i18n.Uzbek, parent, "+998935550001")

	b.tg.Reset()
	fileAppeal(t, b, "Tadbirlar", "Olimpiada natijalari e’lon qilinmadi, ko‘ring.")

	for _, who := range people {
		if got := len(documentsTo(b, who.ID)); got != 1 {
			t.Fatalf("admin %s received %d documents, want 1", who.UserName, got)
		}
	}
}
