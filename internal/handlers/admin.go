package handlers

import (
	"fmt"
	"strconv"

	"aist-bot/internal/i18n"
	"aist-bot/internal/models"
	"aist-bot/internal/utils"
	"aist-bot/internal/validator"
)

// handleAdminCallback routes every inline action of the admin panel.
func handleAdminCallback(c *ctx, action, value string) error {
	id, _ := strconv.ParseInt(value, 10, 64)

	switch action {
	case utils.CBAdminPanel:
		c.Ack("")
		return showAdminPanel(c, true)

	// ── Departments ──────────────────────────────────────────────────
	case utils.CBAdminDepts:
		c.Ack("")
		return showDepartmentList(c)

	case utils.CBAdminDeptAdd:
		c.Ack("")
		return startDepartmentEditor(c, 0)

	case utils.CBAdminDeptOne:
		c.Ack("")
		return showDepartmentMenu(c, id)

	case utils.CBAdminDeptEdit:
		c.Ack("")
		return startDepartmentEditor(c, id)

	case utils.CBAdminDeptTgl:
		return toggleDepartment(c, id)

	case utils.CBAdminDeptDel:
		c.Ack("")
		return confirmDeleteDepartment(c, id)

	case utils.CBAdminDeptDelY:
		return deleteDepartment(c, id)

	// ── Classes ──────────────────────────────────────────────────────
	case utils.CBAdminClasses:
		c.Ack("")
		return showClassList(c)

	case utils.CBAdminClassAdd:
		c.Ack("")
		return startClassEditor(c)

	case utils.CBAdminClassOne:
		c.Ack("")
		return showClassMenu(c, id)

	case utils.CBAdminClassTgl:
		return toggleClass(c, id)

	case utils.CBAdminClassDel:
		c.Ack("")
		return confirmDeleteClass(c, id)

	case utils.CBAdminClassDelY:
		return deleteClass(c, id)

	// ── Reports ──────────────────────────────────────────────────────
	case utils.CBAdminStats:
		c.Ack("")
		return showStats(c)

	case utils.CBAdminRecent:
		c.Ack("")
		return showRecentAppeals(c)

	case utils.CBAdminDoc:
		return resendDocument(c, id)
	}

	c.Ack("")
	return nil
}

// showAdminPanel opens the admin home screen, swapping the bottom keyboard
// from the parent menu to the admin one.
//
// inPlace is used by the Back button of a sub-screen: the admin keyboard is
// already showing, so the inline list is simply cleared away rather than a
// new message being sent. A reply keyboard cannot be changed by editing a
// message, which is exactly why entering the panel always sends.
func showAdminPanel(c *ctx, inPlace bool) error {
	if err := c.ClearState(); err != nil {
		return c.Fail("showAdminPanel/clear", err)
	}

	if inPlace && c.MessageID != 0 {
		return c.Reply(c.T(i18n.MsgAdminPanel), nil)
	}

	return c.Send(c.T(i18n.MsgAdminPanel), utils.AdminMenu(c.Lang))
}

// handleAdminMenuButton reacts to the admin reply keyboard. It reports
// whether the text was one of its buttons.
func handleAdminMenuButton(c *ctx, text string) (bool, error) {
	if !c.IsAdmin {
		return false, nil
	}

	switch {
	case matchesButton(text, i18n.BtnManageDepartments):
		return true, leaveFlow(c, showDepartmentListFresh)

	case matchesButton(text, i18n.BtnManageClasses):
		return true, leaveFlow(c, showClassListFresh)

	case matchesButton(text, i18n.BtnRecent):
		return true, leaveFlow(c, showRecentAppeals)

	case matchesButton(text, i18n.BtnStats):
		return true, leaveFlow(c, showStats)
	}

	return false, nil
}

// showClassListFresh sends the class list as a new message, for when the
// admin arrives from the bottom keyboard rather than an inline button.
func showClassListFresh(c *ctx) error {
	classes, err := c.bot.Classes.GetAll()
	if err != nil {
		return c.Fail("showClassListFresh", err)
	}

	text := c.Tf(i18n.MsgClassList, len(classes))
	if len(classes) == 0 {
		text = c.T(i18n.MsgClassListEmpty)
	}

	return c.Send(text, utils.AdminClassListKeyboard(classes, c.Lang))
}

// ── Departments ──────────────────────────────────────────────────────

func showDepartmentList(c *ctx) error {
	departments, err := c.bot.Departments.GetAll()
	if err != nil {
		return c.Fail("showDepartmentList", err)
	}

	keyboard := utils.AdminDeptListKeyboard(departments, c.Lang)

	text := c.Tf(i18n.MsgDeptList, len(departments))
	if len(departments) == 0 {
		text = c.T(i18n.MsgDeptListEmpty)
	}

	return c.Reply(text, &keyboard)
}

func showDepartmentMenu(c *ctx, id int64) error {
	department, err := c.bot.Departments.GetByID(id)
	if err != nil {
		return c.Fail("showDepartmentMenu", err)
	}

	if department == nil {
		return showDepartmentList(c)
	}

	status := "👁"
	if !department.IsActive {
		status = "🚫"
	}

	keyboard := utils.AdminDeptMenuKeyboard(department, c.Lang)

	return c.Reply(c.Tf(i18n.MsgDeptMenu,
		utils.EscapeHTML(department.LabelFor(c.Lang.String())),
		utils.EscapeHTML(orDash(department.NameUz)),
		utils.EscapeHTML(orDash(department.NameRu)),
		utils.EscapeHTML(orDash(department.NameEn)),
		status,
	), &keyboard)
}

// startDepartmentEditor begins the three-step name editor. An id of zero
// creates a new department; any other id rewrites that one in place, which
// keeps its code and so keeps past appeals pointing at it.
func startDepartmentEditor(c *ctx, id int64) error {
	c.Data = &models.StateData{
		Language:         c.Lang.String(),
		EditDepartmentID: id,
	}

	if err := c.SetState(models.StateAwaitingDeptNameUz); err != nil {
		return c.Fail("startDepartmentEditor/state", err)
	}

	keyboard := utils.NavKeyboard(c.Lang, "")
	return c.Reply(c.T(i18n.MsgDeptPromptUz), &keyboard)
}

// handleDeptNameInput collects one language of a department name.
func handleDeptNameInput(c *ctx, text string) error {
	if !c.IsAdmin {
		return showMainMenu(c, c.T(i18n.MsgMainMenu))
	}

	name, err := validator.ValidateDepartmentName(text)
	if err != nil {
		return c.Send(c.T(i18n.MsgDeptInvalid), utils.NavKeyboard(c.Lang, ""))
	}

	switch c.State {
	case models.StateAwaitingDeptNameUz:
		// An emoji typed at the front of the Uzbek name becomes the icon
		// shown on the button, so admins get one without an extra step.
		emoji, plain := models.SplitLeadingEmoji(name)
		if plain == "" {
			plain = name
			emoji = ""
		}

		taken, err := c.bot.Departments.ExistsByNameUz(plain, c.Data.EditDepartmentID)
		if err != nil {
			return c.Fail("handleDeptNameInput/exists", err)
		}

		if taken {
			return c.Send(c.Tf(i18n.MsgDeptExists, utils.EscapeHTML(plain)), utils.NavKeyboard(c.Lang, ""))
		}

		c.Data.DraftEmoji = emoji
		c.Data.DraftNameUz = plain

		if err := c.SetState(models.StateAwaitingDeptNameRu); err != nil {
			return c.Fail("handleDeptNameInput/state", err)
		}

		return c.Send(c.T(i18n.MsgDeptPromptRu), utils.SkipNavKeyboard(c.Lang, ""))

	case models.StateAwaitingDeptNameRu:
		c.Data.DraftNameRu = name

		if err := c.SetState(models.StateAwaitingDeptNameEn); err != nil {
			return c.Fail("handleDeptNameInput/state", err)
		}

		return c.Send(c.T(i18n.MsgDeptPromptEn), utils.SkipNavKeyboard(c.Lang, ""))

	case models.StateAwaitingDeptNameEn:
		return saveDepartment(c, name)
	}

	return showAdminPanel(c, false)
}

// handleDeptNameSkip leaves a translation empty, which falls back to the
// Uzbek name wherever it is displayed.
func handleDeptNameSkip(c *ctx) error {
	// Skip is a generic callback, so it is checked against both the panel
	// and the step rather than trusted on its own: a stale button from an
	// old message must not open anything.
	if !c.IsAdmin {
		return nil
	}

	switch c.State {
	case models.StateAwaitingDeptNameRu:
		if err := c.SetState(models.StateAwaitingDeptNameEn); err != nil {
			return c.Fail("handleDeptNameSkip/state", err)
		}

		keyboard := utils.SkipNavKeyboard(c.Lang, "")
		return c.Reply(c.T(i18n.MsgDeptPromptEn), &keyboard)

	case models.StateAwaitingDeptNameEn:
		return saveDepartment(c, "")
	}

	return nil
}

// saveDepartment writes the collected names, creating or updating.
func saveDepartment(c *ctx, nameEn string) error {
	emoji := c.Data.DraftEmoji
	if emoji == "" {
		emoji = models.DefaultDepartmentEmoji
	}

	nameUz := c.Data.DraftNameUz
	nameRu := c.Data.DraftNameRu

	var (
		label string
		err   error
	)

	if c.Data.EditDepartmentID == 0 {
		var created *models.Department
		created, err = c.bot.Departments.Create(emoji, nameUz, nameRu, nameEn)
		if created != nil {
			label = created.LabelFor(c.Lang.String())
		}
	} else {
		err = c.bot.Departments.Update(c.Data.EditDepartmentID, emoji, nameUz, nameRu, nameEn)
		label = emoji + " " + nameUz
	}

	if err != nil {
		return c.Fail("saveDepartment", err)
	}

	key := i18n.MsgDeptAdded
	if c.Data.EditDepartmentID != 0 {
		key = i18n.MsgDeptUpdated
	}

	if err := c.ClearState(); err != nil {
		return c.Fail("saveDepartment/clear", err)
	}

	if err := c.Send(c.Tf(key, utils.EscapeHTML(label)), nil); err != nil {
		return err
	}

	c.MessageID = 0
	return showDepartmentListFresh(c)
}

func toggleDepartment(c *ctx, id int64) error {
	department, err := c.bot.Departments.GetByID(id)
	if err != nil {
		return c.Fail("toggleDepartment", err)
	}

	if department == nil {
		c.Ack("")
		return showDepartmentList(c)
	}

	if err := c.bot.Departments.ToggleActive(id); err != nil {
		return c.Fail("toggleDepartment/toggle", err)
	}

	c.Ack(c.Tf(i18n.MsgDeptToggled, department.NameFor(c.Lang.String())))

	return showDepartmentMenu(c, id)
}

func confirmDeleteDepartment(c *ctx, id int64) error {
	department, err := c.bot.Departments.GetByID(id)
	if err != nil {
		return c.Fail("confirmDeleteDepartment", err)
	}

	if department == nil {
		return showDepartmentList(c)
	}

	keyboard := utils.ConfirmDeleteKeyboard(utils.CBAdminDeptDelY, utils.CBAdminDeptOne, id, c.Lang)

	return c.Reply(c.Tf(i18n.MsgDeptDeleteConfirm,
		utils.EscapeHTML(department.LabelFor(c.Lang.String()))), &keyboard)
}

func deleteDepartment(c *ctx, id int64) error {
	department, err := c.bot.Departments.GetByID(id)
	if err != nil {
		return c.Fail("deleteDepartment", err)
	}

	if department == nil {
		c.Ack("")
		return showDepartmentList(c)
	}

	if err := c.bot.Departments.Delete(id); err != nil {
		return c.Fail("deleteDepartment/delete", err)
	}

	c.Ack(c.Tf(i18n.MsgDeptDeleted, department.NameFor(c.Lang.String())))

	return showDepartmentList(c)
}

// showDepartmentListFresh sends the list as a new message, used after a
// text step where there is no inline message left to edit.
func showDepartmentListFresh(c *ctx) error {
	departments, err := c.bot.Departments.GetAll()
	if err != nil {
		return c.Fail("showDepartmentListFresh", err)
	}

	text := c.Tf(i18n.MsgDeptList, len(departments))
	if len(departments) == 0 {
		text = c.T(i18n.MsgDeptListEmpty)
	}

	return c.Send(text, utils.AdminDeptListKeyboard(departments, c.Lang))
}

// ── Classes ──────────────────────────────────────────────────────────

func showClassList(c *ctx) error {
	classes, err := c.bot.Classes.GetAll()
	if err != nil {
		return c.Fail("showClassList", err)
	}

	keyboard := utils.AdminClassListKeyboard(classes, c.Lang)

	text := c.Tf(i18n.MsgClassList, len(classes))
	if len(classes) == 0 {
		text = c.T(i18n.MsgClassListEmpty)
	}

	return c.Reply(text, &keyboard)
}

func showClassMenu(c *ctx, id int64) error {
	class, err := c.bot.Classes.GetByID(id)
	if err != nil {
		return c.Fail("showClassMenu", err)
	}

	if class == nil {
		return showClassList(c)
	}

	status := "👁"
	if !class.IsActive {
		status = "🚫"
	}

	keyboard := utils.AdminClassMenuKeyboard(class, c.Lang)

	return c.Reply(fmt.Sprintf("🏫 <b>%s</b>\n\n%s", utils.EscapeHTML(class.Name), status), &keyboard)
}

func startClassEditor(c *ctx) error {
	c.Data = &models.StateData{Language: c.Lang.String()}

	if err := c.SetState(models.StateAwaitingClassName); err != nil {
		return c.Fail("startClassEditor/state", err)
	}

	keyboard := utils.NavKeyboard(c.Lang, "")
	return c.Reply(c.T(i18n.MsgClassPrompt), &keyboard)
}

func handleClassNameInput(c *ctx, text string) error {
	if !c.IsAdmin {
		return showMainMenu(c, c.T(i18n.MsgMainMenu))
	}

	name, err := validator.ValidateClassName(text)
	if err != nil {
		return c.Send(c.T(i18n.MsgClassInvalid), utils.NavKeyboard(c.Lang, ""))
	}

	created, err := c.bot.Classes.Create(name)
	if err != nil {
		return c.Fail("handleClassNameInput", err)
	}

	key := i18n.MsgClassAdded
	if !created {
		key = i18n.MsgClassExists
	}

	if err := c.ClearState(); err != nil {
		return c.Fail("handleClassNameInput/clear", err)
	}

	if err := c.Send(c.Tf(key, utils.EscapeHTML(name)), nil); err != nil {
		return err
	}

	classes, err := c.bot.Classes.GetAll()
	if err != nil {
		return c.Fail("handleClassNameInput/list", err)
	}

	return c.Send(c.Tf(i18n.MsgClassList, len(classes)), utils.AdminClassListKeyboard(classes, c.Lang))
}

func toggleClass(c *ctx, id int64) error {
	class, err := c.bot.Classes.GetByID(id)
	if err != nil {
		return c.Fail("toggleClass", err)
	}

	if class == nil {
		c.Ack("")
		return showClassList(c)
	}

	if err := c.bot.Classes.ToggleActive(id); err != nil {
		return c.Fail("toggleClass/toggle", err)
	}

	c.Ack(c.Tf(i18n.MsgClassToggled, class.Name))

	return showClassMenu(c, id)
}

func confirmDeleteClass(c *ctx, id int64) error {
	class, err := c.bot.Classes.GetByID(id)
	if err != nil {
		return c.Fail("confirmDeleteClass", err)
	}

	if class == nil {
		return showClassList(c)
	}

	keyboard := utils.ConfirmDeleteKeyboard(utils.CBAdminClassDelY, utils.CBAdminClassOne, id, c.Lang)

	return c.Reply(c.Tf(i18n.MsgDeptDeleteConfirm, utils.EscapeHTML(class.Name)), &keyboard)
}

func deleteClass(c *ctx, id int64) error {
	class, err := c.bot.Classes.GetByID(id)
	if err != nil {
		return c.Fail("deleteClass", err)
	}

	if class == nil {
		c.Ack("")
		return showClassList(c)
	}

	if err := c.bot.Classes.Delete(id); err != nil {
		return c.Fail("deleteClass/delete", err)
	}

	c.Ack(c.Tf(i18n.MsgClassDeleted, class.Name))

	return showClassList(c)
}

// ── Reports ──────────────────────────────────────────────────────────

func showStats(c *ctx) error {
	users, err := c.bot.Users.Count()
	if err != nil {
		return c.Fail("showStats/users", err)
	}

	classes, err := c.bot.Classes.Count()
	if err != nil {
		return c.Fail("showStats/classes", err)
	}

	departments, err := c.bot.Departments.Count()
	if err != nil {
		return c.Fail("showStats/departments", err)
	}

	counts, err := c.bot.Submissions.CountByType()
	if err != nil {
		return c.Fail("showStats/submissions", err)
	}

	complaints := counts[string(models.TypeComplaint)]
	proposals := counts[string(models.TypeProposal)]
	gratitude := counts[string(models.TypeGratitude)]

	keyboard := utils.BackToAdminPanelKeyboard(c.Lang)

	return c.Reply(c.Tf(i18n.MsgAdminStats,
		users, classes, departments,
		complaints, proposals, gratitude,
		complaints+proposals+gratitude,
	), &keyboard)
}

func showRecentAppeals(c *ctx) error {
	const limit = 10

	submissions, err := c.bot.Submissions.GetRecent(limit)
	if err != nil {
		return c.Fail("showRecentAppeals", err)
	}

	if len(submissions) == 0 {
		keyboard := utils.BackToAdminPanelKeyboard(c.Lang)
		return c.Reply(c.T(i18n.MsgAdminNoRecent), &keyboard)
	}

	keyboard := utils.AdminRecentKeyboard(submissions, c.Lang)
	return c.Reply(c.T(i18n.MsgAdminRecent), &keyboard)
}

// resendDocument sends a stored appeal's PDF to the admin again.
func resendDocument(c *ctx, id int64) error {
	submission, err := c.bot.Submissions.GetWithUserByID(id)
	if err != nil {
		return c.Fail("resendDocument", err)
	}

	c.Ack("")

	if submission == nil {
		return showRecentAppeals(c)
	}

	if err := c.bot.Appeals.ResendDocument(c.ChatID, submission, c.Lang); err != nil {
		return c.Fail("resendDocument/send", err)
	}

	return nil
}

func orDash(text string) string {
	if text == "" {
		return "—"
	}
	return text
}
