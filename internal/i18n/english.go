package i18n

var english = map[string]string{
	// ── Registration ────────────────────────────────────────────────
	MsgChooseLanguage: "Choose your language:",
	MsgWelcome: "👋 <b>Hello!</b>\n\n" +
		"With this bot you can send the school a <b>complaint</b>, a <b>proposal</b> or a <b>thank-you note</b>.\n\n" +
		"Your message is turned into a neat PDF document and delivered to the administration.",
	MsgRequestPhone: "📱 <b>Step 1</b>\n\nPlease send your phone number.\n\n" +
		"Tap the <b>«📱 Share my number»</b> button below — the number is sent automatically.",
	MsgPhoneHint:    "💡 If you do not see the button, type your number instead.\nExample: <code>+998901234567</code>",
	MsgPhoneInvalid: "❌ That number is not valid.\n\nPlease tap <b>«📱 Share my number»</b> or type it like this:\n<code>+998901234567</code>",
	MsgRegistered:   "✅ <b>Thank you, you are registered!</b>\n\nNow choose one of the buttons below 👇",

	// ── Main menu ───────────────────────────────────────────────────
	MsgWelcomeAdmin: "👋 <b>Hello!</b>\n\n" +
		"You are a member of the school administration.\n\n" +
		"Parents' messages arrive here as a PDF document. " +
		"Use the menu below to manage departments and classes 👇",
	MsgRegisteredAdmin: "✅ <b>Thank you, you are registered!</b>\n\n" +
		"You have been recognised as a member of the administration — all messages will now come to you 👇",
	MsgHelpAdmin: "ℹ️ <b>Admin guide</b>\n\n" +
		"When a parent sends a message it arrives here as a <b>PDF document</b>, showing the department, " +
		"class, child's name, phone number and time.\n\n" +
		"<b>Your menu:</b>\n" +
		"📂 <b>Departments</b> — add, rename, hide or delete the topics parents write about.\n" +
		"🏫 <b>Classes</b> — add and manage classes. At least one is required, or parents cannot send anything.\n" +
		"📬 <b>Latest messages</b> — the most recent messages; tap one to receive its document again.\n" +
		"📊 <b>Statistics</b> — overall totals.\n\n" +
		"💡 The administration <b>receives</b> messages — sending new ones is for parents.\n\n" +
		"Who is an admin is set in the <code>.env</code> file on the server (<code>ADMIN_PHONES</code>).",
	MsgMainMenu: "🏠 <b>Main menu</b>\n\nWhat would you like to do?",
	MsgHelp: "ℹ️ <b>How does the bot work?</b>\n\n" +
		"1️⃣ Choose the kind of message — complaint, proposal or thank-you.\n" +
		"2️⃣ Choose which department it concerns.\n" +
		"3️⃣ Choose the class.\n" +
		"4️⃣ Type your child's full name.\n" +
		"5️⃣ Type your message.\n" +
		"6️⃣ Add a photo if you wish (optional).\n" +
		"7️⃣ Check everything and tap «Send».\n\n" +
		"A PDF document is created and sent to the administration.\n\n" +
		"You can start over at any time with <b>/start</b>.",

	// ── Appeal flow ─────────────────────────────────────────────────
	MsgStepDepartment: "📂 <b>Step 1: Department</b>\n\nWhich department does your message concern?",
	MsgStepClass:      "🏫 <b>Step 2: Class</b>\n\nChoose the class your child attends:",
	MsgNoClasses: "⚠️ The list of classes has not been set up yet.\n\n" +
		"Please contact the school administration or try again later.",
	MsgStepChildName: "👶 <b>Step 3: Your child</b>\n\nType your child's first and last name.\n\n" +
		"Example: <code>Sardor Aliyev</code>",
	MsgChildNameInvalid: "❌ That name is too short or not valid.\n\nPlease type the full name. Example: <code>Sardor Aliyev</code>",
	MsgStepTextComplaint: "📝 <b>Step 4: Your complaint</b>\n\nDescribe your complaint in detail.\n\n" +
		"💡 Telling us what happened, when, and who was involved helps resolve it faster.",
	MsgStepTextProposal: "💡 <b>Step 4: Your proposal</b>\n\nDescribe your proposal in detail.\n\n" +
		"💡 Write what should be improved and how it could be done.",
	MsgStepTextGratitude: "🙏 <b>Step 4: Your thank-you note</b>\n\nWrite your thank-you message.\n\n" +
		"💡 Mention who you are thanking and what for.",
	MsgTextTooShort: "❌ The text is too short.\n\nPlease write at least %d characters.",
	MsgTextTooLong:  "❌ The text is too long.\n\nMaximum %d characters. Right now: %d.",
	MsgStepPhotoAsk: "📷 <b>Step 5: Photo</b>\n\nWould you like to attach a photo?\n\n" +
		"Attaching a photo is <b>optional</b>.",
	MsgPhotoHint: "📎 Please send the photo.\n\n" +
		"💡 Tap the <b>📎 (paperclip)</b> icon below and pick a photo.\n\n" +
		"You can send up to %d photos.\nWhen you are done, tap <b>«✅ Done»</b>.",
	MsgPhotoAdded:      "✅ Photo received (%d/%d).\n\nSend another one, or tap <b>«✅ Done»</b>.",
	MsgPhotoMax:        "ℹ️ You have sent the maximum of %d photos.\n\nPlease tap <b>«✅ Done»</b>.",
	MsgPhotoWrongInput: "📷 Please send a <b>photo</b>, or tap <b>«✅ Done»</b>.",
	MsgConfirmTitle:    "🔎 <b>Please check</b>\n\nIf everything is correct, tap «Send».",
	MsgSending:         "⏳ Preparing your document, one moment...",
	MsgSent: "✅ <b>Your message has been sent!</b>\n\n" +
		"Document number: <b>%s</b>\n\n" +
		"A copy has been sent to you as well. The administration will review it.\n\nThank you for reaching out! 🙏",
	MsgSentNoAdmins: "✅ <b>Your message has been received!</b>\n\n" +
		"Document number: <b>%s</b>\n\n" +
		"⚠️ The administration has not connected to the bot yet. Your message is saved and will be delivered as soon as they do.",
	MsgSentNoDocument: "✅ <b>Your message has been received!</b>\n\n" +
		"Document number: <b>%s</b>\n\n" +
		"⚠️ The document could not be produced, but your message is saved and has been passed to the administration as text.",
	MsgCancelled: "❌ Cancelled.\n\nYou are back at the main menu.",

	// ── My appeals ──────────────────────────────────────────────────
	MsgMyAppealsTitle: "📂 <b>My messages</b>\n\nYour last %d messages:",
	MsgNoAppeals:      "📭 You have not sent anything yet.\n\nYou can send your first message from the main menu.",

	// ── Settings ────────────────────────────────────────────────────
	MsgSettingsTitle:   "⚙️ <b>Settings</b>\n\nWhat would you like to change?",
	MsgLanguageChanged: "✅ Language changed.",

	// ── Labels ──────────────────────────────────────────────────────
	LblType:       "Type",
	LblDepartment: "Department",
	LblClass:      "Class",
	LblChild:      "Child",
	LblPhone:      "Phone",
	LblUsername:   "Telegram",
	LblDate:       "Date and time",
	LblText:       "Message",
	LblPhotos:     "Photos",
	LblReference:  "Document number",
	LblNone:       "none",

	TypeComplaint: "Complaint",
	TypeProposal:  "Proposal",
	TypeGratitude: "Gratitude",

	// ── Buttons ─────────────────────────────────────────────────────
	BtnSharePhone: "📱 Share my number",
	BtnComplaint:  "📝 Complaint",
	BtnProposal:   "💡 Proposal",
	BtnGratitude:  "🙏 Thank you",
	BtnMyAppeals:  "📂 My messages",
	BtnSettings:   "⚙️ Settings",
	BtnHelp:       "ℹ️ Help",
	BtnAdminPanel: "🛠 Admin panel",
	BtnChangeLang: "🌐 Change language",
	BtnAddPhoto:   "📷 Yes, add a photo",
	BtnNoPhoto:    "➡️ No, continue",
	BtnPhotoDone:  "✅ Done",
	BtnSend:       "✅ Send",
	BtnCancel:     "❌ Cancel",
	BtnBack:       "⬅️ Back",
	BtnMainMenu:   "🏠 Main menu",

	// ── Admin ───────────────────────────────────────────────────────
	MsgAdminPanel: "🛠 <b>Admin panel</b>\n\nChoose a section:",
	MsgRoleNowAdmin: "🛠 <b>You have been given administration access.</b>\n\n" +
		"Parents' messages will now come straight to you.\n" +
		"Use the menu below to manage departments and classes 👇",
	MsgRoleNowParent: "ℹ️ <b>Your administration access has been removed.</b>\n\n" +
		"You can now send complaints, proposals and thank-you notes as a parent 👇",
	MsgAdminCannotSubmit: "ℹ️ You are a member of the school administration.\n\n" +
		"The administration <b>receives</b> appeals — sending new ones is for parents.",
	MsgAdminOnly: "⛔️ This section is for the administration only.",
	MsgClassPrompt: "➕ <b>New class</b>\n\nType the class name.\n\n" +
		"Example: <code>5-A</code>, <code>9-B</code>, <code>11-V</code>",
	MsgClassAdded:     "✅ Class <b>%s</b> has been added.",
	MsgClassExists:    "⚠️ Class <b>%s</b> already exists.",
	MsgClassInvalid:   "❌ Invalid class name. It must be 1 to 20 characters.",
	MsgClassList:      "🏫 <b>Classes</b>\n\nTotal: %d\n\nTap a class to change it:",
	MsgClassListEmpty: "📭 No classes yet.\n\nAdd one with the «➕ Add class» button.",
	MsgClassDeleted:   "🗑 Class <b>%s</b> has been deleted.",
	MsgClassToggled:   "✅ <b>%s</b> has been updated.",
	MsgAdminStats: "📊 <b>Statistics</b>\n\n" +
		"👥 Users: <b>%d</b>\n" +
		"🏫 Classes: <b>%d</b>\n" +
		"📂 Departments: <b>%d</b>\n\n" +
		"📝 Complaints: <b>%d</b>\n" +
		"💡 Proposals: <b>%d</b>\n" +
		"🙏 Thank-you notes: <b>%d</b>\n" +
		"━━━━━━━━━━━━━━\n" +
		"📨 Total messages: <b>%d</b>",
	MsgAdminRecent:     "📬 <b>Latest messages</b>\n\nTap one to receive its document again:",
	MsgAdminNoRecent:   "📭 No messages yet.",
	MsgNewAppealHeader: "🔔 <b>NEW MESSAGE</b>",

	BtnAddClass:      "➕ Add class",
	BtnManageClasses: "🏫 Classes",
	BtnStats:         "📊 Statistics",
	BtnRecent:        "📬 Latest messages",

	// ── Errors ──────────────────────────────────────────────────────
	ErrGeneric:       "❌ Something went wrong. Please try again, or send /start.",
	ErrNotRegistered: "👋 Please register first.\n\nSend the /start command.",
	ErrSessionLost:   "⏱ Your session has expired.\n\nPlease start again — /start",

	// ── PDF ─────────────────────────────────────────────────────────
	PDFTitleComplaint: "COMPLAINT",
	PDFTitleProposal:  "PROPOSAL",
	PDFTitleGratitude: "GRATITUDE",
	PDFSectionDetails: "MESSAGE DETAILS",
	PDFSectionText:    "MESSAGE",
	PDFSectionPhotos:  "ATTACHED PHOTOS",
	PDFPhotoNo:        "Photo %d of %d",
	PDFFooter:         "This document was generated automatically by a Telegram bot",
	PDFPage:           "page %d",

	// ── Admin: departments ──────────────────────────────────────────
	MsgDeptList:      "📂 <b>Departments</b>\n\nTotal: %d\n\nTap a department to change it:",
	MsgDeptListEmpty: "📭 No departments yet.\n\nAdd one with the «➕ Add department» button.",
	MsgDeptMenu:      "📂 <b>%s</b>\n\n🇺🇿 %s\n🇷🇺 %s\n🇬🇧 %s\n\nStatus: %s\n\nWhat would you like to do?",
	MsgDeptPromptUz: "🇺🇿 <b>1/3 — Uzbek name</b>\n\nType the department name in Uzbek.\n\n" +
		"💡 You may start with an emoji.\nExample: <code>🍽 Oshxona</code>",
	MsgDeptPromptRu: "🇷🇺 <b>2/3 — Russian name</b>\n\nType the same department name in Russian.\n\n" +
		"Example: <code>Столовая</code>\n\nNot sure? Tap «⏭ Skip».",
	MsgDeptPromptEn: "🇬🇧 <b>3/3 — English name</b>\n\nType the same department name in English.\n\n" +
		"Example: <code>Canteen</code>\n\nNot sure? Tap «⏭ Skip».",
	MsgDeptAdded:         "✅ Department <b>%s</b> has been added.",
	MsgDeptUpdated:       "✅ Department <b>%s</b> has been updated.",
	MsgDeptDeleted:       "🗑 Department <b>%s</b> has been deleted.",
	MsgDeptToggled:       "✅ <b>%s</b> has been updated.",
	MsgDeptExists:        "⚠️ A department named <b>%s</b> already exists.",
	MsgDeptInvalid:       "❌ Invalid department name. It must be 2 to 60 characters.",
	MsgDeptDeleteConfirm: "🗑 Delete the department <b>%s</b>?\n\nPast messages will be kept.",
	MsgNoDepartments: "⚠️ The list of departments has not been set up yet.\n\n" +
		"Please contact the school administration or try again later.",

	BtnManageDepartments: "📂 Departments",
	BtnAddDept:           "➕ Add department",
	BtnEditDept:          "✏️ Rename",
	BtnDelete:            "🗑 Delete",
	BtnToggleOn:          "👁 Show",
	BtnToggleOff:         "🚫 Hide",
	BtnSkip:              "⏭ Skip",
	BtnYesDelete:         "🗑 Yes, delete",
}
