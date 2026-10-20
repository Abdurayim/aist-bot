package handlers

import (
	"strings"
	"testing"

	"aist-bot/internal/i18n"
	"aist-bot/internal/models"
	"aist-bot/internal/utils"
)

// fileAppeal drives a whole appeal, used by tests that care about what the
// admin side does with it afterwards.
func fileAppeal(t *testing.T, b *testBot, department, text string) {
	t.Helper()

	b.sendText(parent.ID, parent, i18n.Get(i18n.BtnComplaint, i18n.Uzbek))
	b.tap(parent.ID, parent, b.callbackFor(t, department))
	b.tap(parent.ID, parent, b.callbackFor(t, "9-B"))
	b.sendText(parent.ID, parent, "Aliyev Sardor")
	b.sendText(parent.ID, parent, text)
	b.tap(parent.ID, parent, b.callbackFor(t, "davom etish"))
	b.tap(parent.ID, parent, "send")
}

func TestAdminPanelIsClosedToParents(t *testing.T) {
	b := newTestBot(t, adminPhone)
	registerParent(t, b, i18n.Uzbek, parent, parentPhone)

	// The button is not even drawn for a parent.
	if b.hasButton(i18n.Get(i18n.BtnAdminPanel, i18n.Uzbek)) {
		t.Fatal("the admin button must not appear for a parent")
	}

	b.tg.Reset()
	b.sendText(parent.ID, parent, "/admin")
	assertContains(t, b.tg, "faqat ma’muriyat uchun")

	// Nor can they reach it by replaying an admin callback.
	b.tg.Reset()
	b.tap(parent.ID, parent, utils.CBAdminDeptAdd)

	state, _, _ := b.State.Get(parent.ID)
	if state == models.StateAwaitingDeptNameUz {
		t.Fatal("a parent must not be able to start the department editor")
	}
}

// TestAnAdminsHomeIsTheAdminMenu: an admin never sees the parent menu. The
// administration receives appeals, so the three appeal types are absent.
func TestAnAdminsHomeIsTheAdminMenu(t *testing.T) {
	b := newTestBot(t, adminPhone)
	registerParent(t, b, i18n.Uzbek, admin, adminPhone)

	for _, key := range []string{i18n.BtnManageDepartments, i18n.BtnManageClasses, i18n.BtnStats, i18n.BtnRecent} {
		if !b.lastMarkupHas(admin.ID, i18n.Get(key, i18n.Uzbek)) {
			t.Fatalf("the admin home menu is missing %q:\n%s", i18n.Get(key, i18n.Uzbek), b.tg.Last().Markup)
		}
	}

	for _, key := range []string{i18n.BtnComplaint, i18n.BtnProposal, i18n.BtnGratitude, i18n.BtnMyAppeals} {
		if b.lastMarkupHas(admin.ID, i18n.Get(key, i18n.Uzbek)) {
			t.Fatalf("an admin was offered %q; admins receive appeals, they do not write them",
				i18n.Get(key, i18n.Uzbek))
		}
	}
}

// TestAdminAddsAClass covers the only setup step a school must do before
// parents can use the bot.
func TestAdminAddsAClass(t *testing.T) {
	b := newTestBot(t, adminPhone)
	registerParent(t, b, i18n.Uzbek, admin, adminPhone)

	b.sendText(admin.ID, admin, i18n.Get(i18n.BtnAdminPanel, i18n.Uzbek))
	b.tap(admin.ID, admin, utils.CBAdminClasses)
	b.tap(admin.ID, admin, utils.CBAdminClassAdd)

	b.tg.Reset()
	b.sendText(admin.ID, admin, "11-A")

	classes, err := b.Classes.GetActive()
	if err != nil {
		t.Fatalf("failed to read classes: %v", err)
	}

	if len(classes) != 1 || classes[0].Name != "11-A" {
		t.Fatalf("classes = %+v, want one named 11-A", classes)
	}

	assertContains(t, b.tg, "sinfi qo‘shildi")

	// Adding it a second time is refused rather than duplicated.
	b.tap(admin.ID, admin, utils.CBAdminClassAdd)
	b.tg.Reset()
	b.sendText(admin.ID, admin, "11-A")

	assertContains(t, b.tg, "allaqachon mavjud")

	all, _ := b.Classes.GetAll()
	if len(all) != 1 {
		t.Fatalf("a duplicate name created %d classes", len(all))
	}
}

func TestAdminHidesAndDeletesAClass(t *testing.T) {
	b := newTestBot(t, adminPhone)
	seedSchool(t, b)
	registerParent(t, b, i18n.Uzbek, admin, adminPhone)

	classes, _ := b.Classes.GetAll()
	id := classes[0].ID

	b.tap(admin.ID, admin, utils.Callback(utils.CBAdminClassTgl, id))

	active, _ := b.Classes.GetActive()
	if len(active) != 0 {
		t.Fatal("hiding a class must remove it from the parent's list")
	}

	b.tap(admin.ID, admin, utils.Callback(utils.CBAdminClassDel, id))
	b.tap(admin.ID, admin, utils.Callback(utils.CBAdminClassDelY, id))

	all, _ := b.Classes.GetAll()
	if len(all) != 0 {
		t.Fatalf("the class was not deleted: %+v", all)
	}
}

// TestAdminAddsADepartment walks the three-language editor, including the
// emoji picked up from the front of the Uzbek name.
func TestAdminAddsADepartment(t *testing.T) {
	b := newTestBot(t, adminPhone)
	registerParent(t, b, i18n.Uzbek, admin, adminPhone)

	before, _ := b.Departments.GetAll()

	b.sendText(admin.ID, admin, i18n.Get(i18n.BtnAdminPanel, i18n.Uzbek))
	b.tap(admin.ID, admin, utils.CBAdminDepts)
	b.tap(admin.ID, admin, utils.CBAdminDeptAdd)

	assertContains(t, b.tg, "O‘zbekcha nomi")
	b.sendText(admin.ID, admin, "🚌 Transport")

	assertContains(t, b.tg, "Ruscha nomi")
	b.sendText(admin.ID, admin, "Транспорт")

	assertContains(t, b.tg, "Inglizcha nomi")
	b.sendText(admin.ID, admin, "Transport")

	after, _ := b.Departments.GetAll()
	if len(after) != len(before)+1 {
		t.Fatalf("department count went from %d to %d", len(before), len(after))
	}

	created := after[len(after)-1]

	switch {
	case created.NameUz != "Transport":
		t.Fatalf("uzbek name = %q, want the emoji stripped off", created.NameUz)
	case created.Emoji != "🚌":
		t.Fatalf("emoji = %q, want it taken from the front of the name", created.Emoji)
	case created.NameRu != "Транспорт":
		t.Fatalf("russian name = %q", created.NameRu)
	case created.NameEn != "Transport":
		t.Fatalf("english name = %q", created.NameEn)
	case created.Code == "":
		t.Fatal("a department must get a stable code")
	case !created.IsActive:
		t.Fatal("a new department should be visible to parents")
	}

	// And a parent can now pick it.
	seedSchool(t, b)
	registerParent(t, b, i18n.Uzbek, parent, parentPhone)

	b.tg.Reset()
	b.sendText(parent.ID, parent, i18n.Get(i18n.BtnComplaint, i18n.Uzbek))

	if !b.hasButton("Transport") {
		t.Fatalf("the new department is missing from the parent's list:\n%s", b.tg.Last().Markup)
	}
}

// TestAdminSkipsTranslations leaves Russian and English empty, which must
// fall back to the Uzbek name rather than showing a blank button.
func TestAdminSkipsTranslations(t *testing.T) {
	b := newTestBot(t, adminPhone)
	registerParent(t, b, i18n.Uzbek, admin, adminPhone)

	b.tap(admin.ID, admin, utils.CBAdminDeptAdd)
	b.sendText(admin.ID, admin, "Nazorat")
	b.tap(admin.ID, admin, utils.CBSkipStep)
	b.tap(admin.ID, admin, utils.CBSkipStep)

	all, _ := b.Departments.GetAll()
	created := all[len(all)-1]

	if created.NameUz != "Nazorat" {
		t.Fatalf("uzbek name = %q", created.NameUz)
	}

	if got := created.NameFor("ru"); got != "Nazorat" {
		t.Fatalf("russian name falls back to %q, want the Uzbek name", got)
	}

	if got := created.NameFor("en"); got != "Nazorat" {
		t.Fatalf("english name falls back to %q, want the Uzbek name", got)
	}

	if created.Emoji != models.DefaultDepartmentEmoji {
		t.Fatalf("emoji = %q, want the default", created.Emoji)
	}
}

// TestAdminRenamesADepartment checks the edit path keeps the code, which is
// what keeps past appeals pointing at the same department.
func TestAdminRenamesADepartment(t *testing.T) {
	b := newTestBot(t, adminPhone)
	registerParent(t, b, i18n.Uzbek, admin, adminPhone)

	original, err := b.Departments.GetByCode("kitchen")
	if err != nil || original == nil {
		t.Fatalf("the seeded kitchen department is missing: %v", err)
	}

	b.tap(admin.ID, admin, utils.Callback(utils.CBAdminDeptEdit, original.ID))
	b.sendText(admin.ID, admin, "🍲 Oshxona va bufet")
	b.sendText(admin.ID, admin, "Столовая и буфет")
	b.sendText(admin.ID, admin, "Canteen and buffet")

	updated, _ := b.Departments.GetByID(original.ID)

	switch {
	case updated.NameUz != "Oshxona va bufet":
		t.Fatalf("uzbek name = %q", updated.NameUz)
	case updated.NameRu != "Столовая и буфет":
		t.Fatalf("russian name = %q", updated.NameRu)
	case updated.Code != original.Code:
		t.Fatalf("code changed from %q to %q; past appeals would be orphaned", original.Code, updated.Code)
	}
}

func TestAdminRejectsADuplicateDepartmentName(t *testing.T) {
	b := newTestBot(t, adminPhone)
	registerParent(t, b, i18n.Uzbek, admin, adminPhone)

	before, _ := b.Departments.GetAll()

	b.tap(admin.ID, admin, utils.CBAdminDeptAdd)

	b.tg.Reset()
	b.sendText(admin.ID, admin, "Kutubxona (kitob)")

	assertContains(t, b.tg, "allaqachon mavjud")

	after, _ := b.Departments.GetAll()
	if len(after) != len(before) {
		t.Fatal("a duplicate name must not create a department")
	}
}

func TestAdminHidesAndDeletesADepartment(t *testing.T) {
	b := newTestBot(t, adminPhone)
	registerParent(t, b, i18n.Uzbek, admin, adminPhone)

	dept, _ := b.Departments.GetByCode("library")

	b.tap(admin.ID, admin, utils.Callback(utils.CBAdminDeptTgl, dept.ID))

	active, _ := b.Departments.GetActive()
	for _, d := range active {
		if d.Code == "library" {
			t.Fatal("a hidden department must not be offered to parents")
		}
	}

	b.tap(admin.ID, admin, utils.Callback(utils.CBAdminDeptDel, dept.ID))
	b.tap(admin.ID, admin, utils.Callback(utils.CBAdminDeptDelY, dept.ID))

	gone, _ := b.Departments.GetByID(dept.ID)
	if gone != nil {
		t.Fatal("the department was not deleted")
	}
}

// TestDeletingADepartmentKeepsPastAppealsReadable is why appeals store their
// own copy of the department name.
func TestDeletingADepartmentKeepsPastAppealsReadable(t *testing.T) {
	b := newTestBot(t, adminPhone)
	seedSchool(t, b)
	registerParent(t, b, i18n.Uzbek, admin, adminPhone)
	registerParent(t, b, i18n.Uzbek, parent, parentPhone)

	fileAppeal(t, b, "Oshxona", "Ovqat sovuq berilmoqda, iltimos tekshiring.")

	dept, _ := b.Departments.GetByCode("kitchen")
	if err := b.Departments.Delete(dept.ID); err != nil {
		t.Fatalf("failed to delete department: %v", err)
	}

	stored, _ := b.Submissions.GetRecent(10)
	if len(stored) != 1 {
		t.Fatalf("expected one appeal, got %d", len(stored))
	}

	if stored[0].DepartmentName != "Oshxona" {
		t.Fatalf("department name after deletion = %q, want the snapshot kept", stored[0].DepartmentName)
	}

	// And the parent's own list still renders.
	b.tg.Reset()
	b.sendText(parent.ID, parent, i18n.Get(i18n.BtnMyAppeals, i18n.Uzbek))
	assertContains(t, b.tg, "Oshxona")
}

func TestAdminStatsCountEachType(t *testing.T) {
	b := newTestBot(t, adminPhone)
	seedSchool(t, b)
	registerParent(t, b, i18n.Uzbek, admin, adminPhone)
	registerParent(t, b, i18n.Uzbek, parent, parentPhone)

	fileAppeal(t, b, "Oshxona", "Birinchi shikoyat matni, tekshirib ko‘ring.")
	fileAppeal(t, b, "Mentorlar", "Ikkinchi shikoyat matni, tekshirib ko‘ring.")

	b.tg.Reset()
	b.tap(admin.ID, admin, utils.CBAdminStats)

	body := b.tg.Last().Body()
	for _, want := range []string{"Shikoyatlar: <b>2</b>", "Takliflar: <b>0</b>", "Jami murojaatlar: <b>2</b>"} {
		if !strings.Contains(body, want) {
			t.Fatalf("statistics are missing %q:\n%s", want, body)
		}
	}
}

// TestAdminResendsADocument covers an admin re-opening an appeal later.
func TestAdminResendsADocument(t *testing.T) {
	b := newTestBot(t, adminPhone)
	seedSchool(t, b)
	registerParent(t, b, i18n.Uzbek, admin, adminPhone)
	registerParent(t, b, i18n.Uzbek, parent, parentPhone)

	fileAppeal(t, b, "Moliya", "To‘lov kvitansiyasi kelmadi, iltimos tekshiring.")

	stored, _ := b.Submissions.GetRecent(1)
	appealID := stored[0].ID

	b.tg.Reset()
	b.tap(admin.ID, admin, utils.CBAdminRecent)
	b.tap(admin.ID, admin, utils.Callback(utils.CBAdminDoc, appealID))

	docs := documentsTo(b, admin.ID)
	if len(docs) != 1 {
		t.Fatalf("expected one document resent, got %d:\n%s", len(docs), dump(b.tg.Sent()))
	}

	if !strings.Contains(docs[0].Caption, stored[0].Reference) {
		t.Fatalf("the resent caption is missing the reference:\n%s", docs[0].Caption)
	}
}

// TestOneAdminWithSeveralNumbersIsNotifiedOnce is the case the school asked
// about: an admin who lists more than one phone number in .env.
func TestOneAdminWithSeveralNumbersIsNotifiedOnce(t *testing.T) {
	second := "+998977778899"

	b := newTestBot(t, adminPhone, second)
	seedSchool(t, b)

	// The same person registers with their second number.
	registerParent(t, b, i18n.Uzbek, admin, second)
	registerParent(t, b, i18n.Uzbek, parent, parentPhone)

	b.tg.Reset()
	fileAppeal(t, b, "Oshxona", "Ovqat sifati pasayib ketdi, iltimos tekshiring.")

	docs := documentsTo(b, admin.ID)
	if len(docs) != 1 {
		t.Fatalf("the admin received %d copies, want exactly 1:\n%s", len(docs), dump(b.tg.Sent()))
	}
}

// TestFourAdminsAllReceiveTheAppeal covers the four-admin setup the school
// described.
func TestFourAdminsAllReceiveTheAppeal(t *testing.T) {
	phones := []string{"+998901112233", "+998935556677", "+998977778899", "+998901234567"}

	b := newTestBot(t, phones...)
	seedSchool(t, b)

	admins := []int64{301, 302, 303, 304}
	for i, id := range admins {
		registerParent(t, b, i18n.Uzbek, user(id, "admin"), phones[i])
	}

	registerParent(t, b, i18n.Uzbek, parent, "+998935556000")

	b.tg.Reset()
	fileAppeal(t, b, "Tadbirlar", "Olimpiada natijalari e’lon qilinmadi, iltimos ko‘ring.")

	for _, id := range admins {
		if got := len(documentsTo(b, id)); got != 1 {
			t.Fatalf("admin %d received %d documents, want 1", id, got)
		}
	}
}

// TestAppealSurvivesWhenNoAdminHasConnected must never silently drop what a
// parent wrote.
func TestAppealSurvivesWhenNoAdminHasConnected(t *testing.T) {
	b := newTestBot(t, adminPhone)
	seedSchool(t, b)
	registerParent(t, b, i18n.Uzbek, parent, parentPhone)

	b.tg.Reset()
	fileAppeal(t, b, "Tibbiy", "Shifokor kabineti yopiq edi, iltimos tekshiring.")

	stored, _ := b.Submissions.GetRecent(10)
	if len(stored) != 1 {
		t.Fatalf("the appeal was not stored: %d rows", len(stored))
	}

	// And the parent is told plainly that nobody is connected yet.
	assertContains(t, b.tg, "rahbariyat botga ulanmagan")
}

// TestStaleAdminButtonsAreInert covers a parent tapping buttons copied from
// an admin's chat, or left over in their own history.
func TestStaleAdminButtonsAreInert(t *testing.T) {
	b := newTestBot(t, adminPhone)
	registerParent(t, b, i18n.Uzbek, parent, parentPhone)

	before, _ := b.Departments.GetAll()

	// The generic Skip callback must not open the admin panel.
	b.tg.Reset()
	b.tap(parent.ID, parent, utils.CBSkipStep)

	if b.hasButton(i18n.Get(i18n.BtnManageDepartments, i18n.Uzbek)) {
		t.Fatal("the admin panel was opened for a parent")
	}

	// Neither may destructive admin callbacks do anything.
	dept, _ := b.Departments.GetByCode("kitchen")
	b.tap(parent.ID, parent, utils.Callback(utils.CBAdminDeptDelY, dept.ID))
	b.tap(parent.ID, parent, utils.Callback(utils.CBAdminDeptTgl, dept.ID))

	after, _ := b.Departments.GetAll()
	if len(after) != len(before) {
		t.Fatalf("a parent changed the departments: %d → %d", len(before), len(after))
	}

	still, _ := b.Departments.GetByCode("kitchen")
	if still == nil || !still.IsActive {
		t.Fatal("a parent managed to delete or hide a department")
	}
}

// TestOpeningHelpAbandonsAHalfFilledForm: whatever the parent types next
// must not be swallowed as an answer to a question they left behind.
func TestOpeningHelpAbandonsAHalfFilledForm(t *testing.T) {
	b := newTestBot(t, adminPhone)
	seedSchool(t, b)
	registerParent(t, b, i18n.Uzbek, parent, parentPhone)

	b.sendText(parent.ID, parent, i18n.Get(i18n.BtnComplaint, i18n.Uzbek))
	b.tap(parent.ID, parent, b.callbackFor(t, "Oshxona"))
	b.tap(parent.ID, parent, b.callbackFor(t, "9-B"))

	// Mid-form, the parent opens the help text instead of answering.
	b.sendText(parent.ID, parent, i18n.Get(i18n.BtnHelp, i18n.Uzbek))

	state, _, _ := b.State.Get(parent.ID)
	if state != "" {
		t.Fatalf("state after opening help = %q, want the form abandoned", state)
	}

	// The next thing they type is a message, not a child's name.
	b.tg.Reset()
	b.sendText(parent.ID, parent, "Salom")
	assertContains(t, b.tg, "Asosiy menyu")
}

// TestAnExistingParentBecomesAdminOnRestart covers adding someone to
// ADMIN_PHONES who had already registered as an ordinary parent: they must
// be connected at startup, not left waiting to be noticed.
func TestAnExistingParentBecomesAdminOnRestart(t *testing.T) {
	b := newTestBot(t, "+998900000001")

	// They register as a plain parent, with a number not yet on the list.
	registerParent(t, b, i18n.Uzbek, parent, parentPhone)

	linked, _ := b.Admins.GetLinkedTelegramIDs()
	if len(linked) != 0 {
		t.Fatal("a plain parent must not be connected as an admin")
	}

	// The school adds their number to .env and restarts.
	b.Config.Admin.PhoneNumbers = []string{parentPhone}
	if err := b.SyncAdmins(); err != nil {
		t.Fatalf("SyncAdmins failed: %v", err)
	}

	linked, _ = b.Admins.GetLinkedTelegramIDs()
	if len(linked) != 1 || linked[0] != parent.ID {
		t.Fatalf("connected admins = %v, want just %d", linked, parent.ID)
	}

	// And they can reach the panel straight away.
	b.tg.Reset()
	b.sendText(parent.ID, parent, "/admin")

	if !b.hasButton(i18n.Get(i18n.BtnManageDepartments, i18n.Uzbek)) {
		t.Fatalf("the admin panel did not open:\n%s", dump(b.tg.Sent()))
	}
}

// TestAdminCannotCreateAnAppeal enforces the rule beyond the menu: a button
// left in the chat from before someone was made an admin must not work
// either.
func TestAdminCannotCreateAnAppeal(t *testing.T) {
	b := newTestBot(t, adminPhone)
	seedSchool(t, b)
	registerParent(t, b, i18n.Uzbek, admin, adminPhone)

	for _, key := range []string{i18n.BtnComplaint, i18n.BtnProposal, i18n.BtnGratitude} {
		b.tg.Reset()
		b.sendText(admin.ID, admin, i18n.Get(key, i18n.Uzbek))

		assertContains(t, b.tg, "Ma’muriyat murojaatlarni")

		state, _, _ := b.State.Get(admin.ID)
		if state != "" {
			t.Fatalf("%q started a flow for an admin: state = %q", i18n.Get(key, i18n.Uzbek), state)
		}
	}

	stored, _ := b.Submissions.GetRecent(10)
	if len(stored) != 0 {
		t.Fatalf("an admin filed %d appeals", len(stored))
	}

	// "My appeals" is a parent feature too.
	b.tg.Reset()
	b.sendText(admin.ID, admin, i18n.Get(i18n.BtnMyAppeals, i18n.Uzbek))
	assertContains(t, b.tg, "Ma’muriyat murojaatlarni")
}

// TestEveryAdminGetsTheParentsAppealOnce checks the delivery fan-out with
// more than one admin connected.
func TestEveryAdminGetsTheParentsAppealOnce(t *testing.T) {
	second := user(333, "ikkinchi_admin")
	secondPhone := "+998977778899"

	b := newTestBot(t, adminPhone, secondPhone)
	seedSchool(t, b)
	registerParent(t, b, i18n.Uzbek, admin, adminPhone)
	registerParent(t, b, i18n.Uzbek, second, secondPhone)
	registerParent(t, b, i18n.Uzbek, parent, parentPhone)

	b.tg.Reset()
	fileAppeal(t, b, "Oshxona", "Ovqat sovuq berilmoqda, iltimos tekshiring.")

	for _, who := range []int64{admin.ID, second.ID} {
		if got := len(documentsTo(b, who)); got != 1 {
			t.Fatalf("admin %d received %d documents, want 1", who, got)
		}
	}

	// And the parent keeps their own copy.
	if got := len(documentsTo(b, parent.ID)); got != 1 {
		t.Fatalf("the parent received %d documents, want 1", got)
	}
}

// TestOnlyOneMenuIsEverOnScreen: each role has exactly one menu, and the
// two never mix.
func TestOnlyOneMenuIsEverOnScreen(t *testing.T) {
	b := newTestBot(t, adminPhone)
	registerParent(t, b, i18n.Uzbek, admin, adminPhone)
	registerParent(t, b, i18n.Uzbek, parent, parentPhone)

	complaint := i18n.Get(i18n.BtnComplaint, i18n.Uzbek)
	departments := i18n.Get(i18n.BtnManageDepartments, i18n.Uzbek)

	// The parent sees the parent menu and nothing administrative.
	if !b.lastMarkupHas(parent.ID, complaint) {
		t.Fatal("the parent menu is not showing for a parent")
	}

	if b.lastMarkupHas(parent.ID, departments) {
		t.Fatal("a parent was shown the admin menu")
	}

	// The admin sees the admin menu and nothing parental.
	if !b.lastMarkupHas(admin.ID, departments) {
		t.Fatalf("the admin menu is not showing for an admin:\n%s", b.tg.Last().Markup)
	}

	if b.lastMarkupHas(admin.ID, complaint) {
		t.Fatal("an admin was shown the parent menu")
	}

	// Navigating the panel keeps the admin menu, never stacking a second one.
	b.tg.Reset()
	b.sendText(admin.ID, admin, departments)

	if b.lastMarkupHas(admin.ID, complaint) {
		t.Fatal("the parent menu appeared inside the admin panel")
	}
}

// TestAdminMenuButtonsWork drives the panel from the bottom keyboard rather
// than from inline buttons.
func TestAdminMenuButtonsWork(t *testing.T) {
	b := newTestBot(t, adminPhone)
	seedSchool(t, b)
	registerParent(t, b, i18n.Uzbek, admin, adminPhone)

	b.sendText(admin.ID, admin, i18n.Get(i18n.BtnAdminPanel, i18n.Uzbek))

	cases := []struct {
		button string
		expect string
	}{
		{i18n.Get(i18n.BtnManageDepartments, i18n.Uzbek), "Bo‘limlar"},
		{i18n.Get(i18n.BtnManageClasses, i18n.Uzbek), "Sinflar"},
		{i18n.Get(i18n.BtnStats, i18n.Uzbek), "Statistika"},
		{i18n.Get(i18n.BtnRecent, i18n.Uzbek), "murojaat"},
	}

	for _, c := range cases {
		b.tg.Reset()
		b.sendText(admin.ID, admin, c.button)
		assertContains(t, b.tg, c.expect)
	}
}

// TestParentsCannotUseTheAdminKeyboard: its labels must do nothing for a
// parent who types them by hand.
func TestParentsCannotUseTheAdminKeyboard(t *testing.T) {
	b := newTestBot(t, adminPhone)
	registerParent(t, b, i18n.Uzbek, parent, parentPhone)

	b.tg.Reset()
	b.sendText(parent.ID, parent, i18n.Get(i18n.BtnManageDepartments, i18n.Uzbek))

	if b.lastMarkupHas(parent.ID, i18n.Get(i18n.BtnAddDept, i18n.Uzbek)) {
		t.Fatalf("a parent reached the department editor:\n%s", b.tg.Last().Markup)
	}
}
