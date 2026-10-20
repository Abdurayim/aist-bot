package handlers

import (
	"strings"
	"testing"

	"aist-bot/internal/i18n"
	"aist-bot/internal/models"
	"aist-bot/internal/utils"
)

// promote adds a phone number to ADMIN_PHONES and re-runs the startup sync,
// which is exactly what editing .env and restarting the bot does.
func promote(t *testing.T, b *testBot, phones ...string) {
	t.Helper()

	b.Config.Admin.PhoneNumbers = phones

	if err := b.SyncAdmins(); err != nil {
		t.Fatalf("SyncAdmins failed: %v", err)
	}
}

// TestPromotionSwapsTheMenuOnTheNextMessage. A promotion happens while the
// person is looking at a parent keyboard, so the first thing they do after
// it must hand them the admin one.
func TestPromotionSwapsTheMenuOnTheNextMessage(t *testing.T) {
	b := newTestBot(t, adminPhone)
	seedSchool(t, b)
	registerParent(t, b, i18n.Uzbek, parent, parentPhone)

	// Before: a parent, with the parent menu.
	if !b.lastMarkupHas(parent.ID, i18n.Get(i18n.BtnComplaint, i18n.Uzbek)) {
		t.Fatal("the parent menu is not showing before promotion")
	}

	promote(t, b, adminPhone, parentPhone)

	// They tap a button from the keyboard they can still see.
	b.tg.Reset()
	b.sendText(parent.ID, parent, i18n.Get(i18n.BtnComplaint, i18n.Uzbek))

	assertContains(t, b.tg, "ma’muriyat huquqlari berildi")

	if !b.lastMarkupHas(parent.ID, i18n.Get(i18n.BtnManageDepartments, i18n.Uzbek)) {
		t.Fatalf("the admin menu was not handed over:\n%s", b.tg.Last().Markup)
	}

	if b.lastMarkupHas(parent.ID, i18n.Get(i18n.BtnComplaint, i18n.Uzbek)) {
		t.Fatal("the parent menu is still on screen after promotion")
	}

	// The stored role is updated, so they are not told again.
	b.tg.Reset()
	b.sendText(parent.ID, parent, i18n.Get(i18n.BtnStats, i18n.Uzbek))

	for _, m := range b.tg.Sent() {
		if strings.Contains(m.Body(), "ma’muriyat huquqlari berildi") {
			t.Fatal("the promotion was announced twice")
		}
	}

	assertContains(t, b.tg, "Statistika")
}

// TestPromotionMidFormDiscardsTheForm is the nasty case: a parent is halfway
// through a complaint when they are promoted. The half-filled form belongs
// to a role that is not allowed to send it.
func TestPromotionMidFormDiscardsTheForm(t *testing.T) {
	b := newTestBot(t, adminPhone)
	seedSchool(t, b)
	registerParent(t, b, i18n.Uzbek, parent, parentPhone)

	b.sendText(parent.ID, parent, i18n.Get(i18n.BtnComplaint, i18n.Uzbek))
	b.tap(parent.ID, parent, b.callbackFor(t, "Oshxona"))
	b.tap(parent.ID, parent, b.callbackFor(t, "9-B"))
	b.sendText(parent.ID, parent, "Aliyev Sardor")

	state, _, _ := b.State.Get(parent.ID)
	if state != models.StateAwaitingText {
		t.Fatalf("setup failed: state = %q", state)
	}

	promote(t, b, adminPhone, parentPhone)

	// They carry on typing the complaint they had started.
	b.tg.Reset()
	b.sendText(parent.ID, parent, "Oshxonadagi ovqat sovuq berilmoqda, tekshiring.")

	assertContains(t, b.tg, "ma’muriyat huquqlari berildi")

	state, _, _ = b.State.Get(parent.ID)
	if state != "" {
		t.Fatalf("the half-filled form survived the promotion: state = %q", state)
	}

	// And nothing was filed under the old role.
	stored, _ := b.Submissions.GetRecent(10)
	if len(stored) != 0 {
		t.Fatalf("a promoted user filed %d appeals", len(stored))
	}
}

// TestPromotionIsNoticedOnACallbackToo covers the person tapping an inline
// button rather than a keyboard one.
func TestPromotionIsNoticedOnACallbackToo(t *testing.T) {
	b := newTestBot(t, adminPhone)
	seedSchool(t, b)
	registerParent(t, b, i18n.Uzbek, parent, parentPhone)

	b.sendText(parent.ID, parent, i18n.Get(i18n.BtnComplaint, i18n.Uzbek))
	kitchen := b.callbackFor(t, "Oshxona")

	promote(t, b, adminPhone, parentPhone)

	b.tg.Reset()
	b.tap(parent.ID, parent, kitchen)

	assertContains(t, b.tg, "ma’muriyat huquqlari berildi")

	if !b.lastMarkupHas(parent.ID, i18n.Get(i18n.BtnManageDepartments, i18n.Uzbek)) {
		t.Fatal("the admin menu was not handed over after an inline tap")
	}
}

// TestAPromotedUserCanRunTheAdminPanel: access must actually work, not just
// the menu appear.
func TestAPromotedUserCanRunTheAdminPanel(t *testing.T) {
	b := newTestBot(t, adminPhone)
	registerParent(t, b, i18n.Uzbek, parent, parentPhone)

	promote(t, b, adminPhone, parentPhone)

	// Absorb the hand-over.
	b.sendText(parent.ID, parent, "salom")

	b.tg.Reset()
	b.sendText(parent.ID, parent, i18n.Get(i18n.BtnManageClasses, i18n.Uzbek))
	b.tap(parent.ID, parent, b.callbackFor(t, i18n.Get(i18n.BtnAddClass, i18n.Uzbek)))
	b.sendText(parent.ID, parent, "7-V")

	classes, _ := b.Classes.GetActive()
	if len(classes) != 1 || classes[0].Name != "7-V" {
		t.Fatalf("a promoted user could not add a class: %+v", classes)
	}

	// They are also delivered appeals now.
	linked, _ := b.Admins.GetLinkedTelegramIDs()
	var found bool
	for _, id := range linked {
		if id == parent.ID {
			found = true
		}
	}

	if !found {
		t.Fatal("a promoted user is not connected to receive appeals")
	}
}

// TestDemotionGivesBackTheParentMenu is the reverse trip.
func TestDemotionGivesBackTheParentMenu(t *testing.T) {
	b := newTestBot(t, adminPhone)
	seedSchool(t, b)
	registerParent(t, b, i18n.Uzbek, admin, adminPhone)

	if !b.lastMarkupHas(admin.ID, i18n.Get(i18n.BtnManageDepartments, i18n.Uzbek)) {
		t.Fatal("setup failed: the admin menu is not showing")
	}

	// Removed from ADMIN_PHONES and the bot restarted.
	promote(t, b, "+998900000009")

	b.tg.Reset()
	b.sendText(admin.ID, admin, i18n.Get(i18n.BtnStats, i18n.Uzbek))

	assertContains(t, b.tg, "huquqlaringiz olib tashlandi")

	if !b.lastMarkupHas(admin.ID, i18n.Get(i18n.BtnComplaint, i18n.Uzbek)) {
		t.Fatalf("the parent menu was not given back:\n%s", b.tg.Last().Markup)
	}

	if b.lastMarkupHas(admin.ID, i18n.Get(i18n.BtnManageDepartments, i18n.Uzbek)) {
		t.Fatal("the admin menu survived demotion")
	}
}

// TestADemotedUserLosesAdminPowers checks the access itself is gone, not
// just the buttons.
func TestADemotedUserLosesAdminPowers(t *testing.T) {
	b := newTestBot(t, adminPhone)
	seedSchool(t, b)
	registerParent(t, b, i18n.Uzbek, admin, adminPhone)

	promote(t, b, "+998900000009")

	// Absorb the hand-over.
	b.sendText(admin.ID, admin, "salom")

	before, _ := b.Departments.GetAll()
	dept, _ := b.Departments.GetByCode("kitchen")

	b.tg.Reset()
	b.sendText(admin.ID, admin, i18n.Get(i18n.BtnManageDepartments, i18n.Uzbek))

	if b.lastMarkupHas(admin.ID, i18n.Get(i18n.BtnAddDept, i18n.Uzbek)) {
		t.Fatal("a demoted user still reached the department editor")
	}

	// Nor can they use an inline button left in their chat.
	b.tap(admin.ID, admin, utils.Callback(utils.CBAdminDeptDelY, dept.ID))

	after, _ := b.Departments.GetAll()
	if len(after) != len(before) {
		t.Fatal("a demoted user deleted a department")
	}

	// They are no longer sent appeals.
	linked, _ := b.Admins.GetLinkedTelegramIDs()
	for _, id := range linked {
		if id == admin.ID {
			t.Fatal("a demoted user is still connected to receive appeals")
		}
	}
}

// TestADemotedUserCanFileAppealsAgain closes the loop: they are a parent in
// full, not a half-disabled admin.
func TestADemotedUserCanFileAppealsAgain(t *testing.T) {
	b := newTestBot(t, adminPhone)
	seedSchool(t, b)
	registerParent(t, b, i18n.Uzbek, admin, adminPhone)

	promote(t, b, "+998900000009")
	b.sendText(admin.ID, admin, "salom")

	b.tg.Reset()
	b.sendText(admin.ID, admin, i18n.Get(i18n.BtnComplaint, i18n.Uzbek))
	b.tap(admin.ID, admin, b.callbackFor(t, "Oshxona"))
	b.tap(admin.ID, admin, b.callbackFor(t, "9-B"))
	b.sendText(admin.ID, admin, "Xojiboy Tojiboyev")
	b.sendText(admin.ID, admin, "Ovqat sovuq berilmoqda, iltimos tekshiring.")
	b.tap(admin.ID, admin, b.callbackFor(t, "davom etish"))
	b.tap(admin.ID, admin, "send")

	stored, _ := b.Submissions.GetRecent(10)
	if len(stored) != 1 {
		t.Fatalf("a demoted user filed %d appeals, want 1", len(stored))
	}
}

// TestRoleIsRecordedAtRegistration guards the bug where a brand new admin
// was told on their second message that they had just been promoted.
func TestRoleIsRecordedAtRegistration(t *testing.T) {
	b := newTestBot(t, adminPhone)
	registerParent(t, b, i18n.Uzbek, admin, adminPhone)

	stored, _ := b.Users.GetByTelegramID(admin.ID)
	if stored.Role != models.RoleAdmin {
		t.Fatalf("role after registering as an admin = %q, want %q", stored.Role, models.RoleAdmin)
	}

	b.tg.Reset()
	b.sendText(admin.ID, admin, i18n.Get(i18n.BtnStats, i18n.Uzbek))

	for _, m := range b.tg.Sent() {
		if strings.Contains(m.Body(), "huquqlari berildi") {
			t.Fatalf("a newly registered admin was told their role changed:\n%s", m.Body())
		}
	}

	// And a parent is recorded as a parent.
	registerParent(t, b, i18n.Uzbek, parent, parentPhone)

	storedParent, _ := b.Users.GetByTelegramID(parent.ID)
	if storedParent.Role != models.RoleParent {
		t.Fatalf("role after registering as a parent = %q, want %q", storedParent.Role, models.RoleParent)
	}
}

// parentGreeting is a fragment of the parent welcome that carries no HTML
// tags, so it matches the message as actually sent.
const parentGreeting = "Bu bot orqali siz maktabga"

// TestAdminIsNeverGreetedAsAParent. The admin menu was right, but /start
// still told admins "you can send a complaint through this bot" — which
// describes something their role is not allowed to do.
func TestAdminIsNeverGreetedAsAParent(t *testing.T) {
	b := newTestBot(t, adminPhone)
	registerParent(t, b, i18n.Uzbek, admin, adminPhone)

	// The text shown the moment registration completes.
	assertContains(t, b.tg, "ma’muriyat vakili sifatida tanildingiz")

	// /start for a returning admin.
	b.tg.Reset()
	b.sendText(admin.ID, admin, "/start")

	assertContains(t, b.tg, "Siz maktab ma’muriyati vakilisiz")

	// /help for an admin.
	b.tg.Reset()
	b.sendText(admin.ID, admin, "/help")

	assertContains(t, b.tg, "Admin uchun qo‘llanma")

	// None of those may ever offer to take a complaint from them.
	for _, m := range b.tg.ToChat(admin.ID) {
		if strings.Contains(m.Body(), parentGreeting) {
			t.Fatalf("an admin was told they can send appeals:\n%s", m.Body())
		}
	}
}

// TestParentGreetingIsUnchanged guards against fixing the admin text by
// breaking the parent text.
func TestParentGreetingIsUnchanged(t *testing.T) {
	b := newTestBot(t, adminPhone)
	registerParent(t, b, i18n.Uzbek, parent, parentPhone)

	b.tg.Reset()
	b.sendText(parent.ID, parent, "/start")
	assertContains(t, b.tg, parentGreeting)

	b.tg.Reset()
	b.sendText(parent.ID, parent, "/help")
	assertContains(t, b.tg, "Bot qanday ishlaydi?")

	for _, m := range b.tg.ToChat(parent.ID) {
		if strings.Contains(m.Body(), "ma’muriyati vakilisiz") {
			t.Fatalf("a parent was greeted as an admin:\n%s", m.Body())
		}
	}
}

// TestGreetingFollowsAPromotion: the wording must track the role change,
// not the role someone had when they first registered.
func TestGreetingFollowsAPromotion(t *testing.T) {
	b := newTestBot(t, adminPhone)
	registerParent(t, b, i18n.Uzbek, parent, parentPhone)

	promote(t, b, adminPhone, parentPhone)
	b.sendText(parent.ID, parent, "salom") // absorb the hand-over

	b.tg.Reset()
	b.sendText(parent.ID, parent, "/start")

	assertContains(t, b.tg, "Siz maktab ma’muriyati vakilisiz")
}

// TestAdminGreetingExistsInEveryLanguage.
func TestAdminGreetingExistsInEveryLanguage(t *testing.T) {
	cases := []struct {
		lang   i18n.Language
		phone  string
		id     int64
		expect string
	}{
		{i18n.Uzbek, "+998901110001", 801, "ma’muriyati vakilisiz"},
		{i18n.Russian, "+998901110002", 802, "представитель администрации"},
		{i18n.English, "+998901110003", 803, "member of the school administration"},
	}

	for _, c := range cases {
		b := newTestBot(t, c.phone)
		who := user(c.id, "admin")

		registerParent(t, b, c.lang, who, c.phone)

		b.tg.Reset()
		b.sendText(who.ID, who, "/start")

		assertContains(t, b.tg, c.expect)
	}
}
