package i18n

var uzbek = map[string]string{
	// ── Registration ────────────────────────────────────────────────
	MsgChooseLanguage: "Tilni tanlang:",
	MsgWelcome: "👋 <b>Assalomu alaykum!</b>\n\n" +
		"Bu bot orqali siz maktabga <b>shikoyat</b>, <b>taklif</b> yoki <b>minnatdorchilik</b> yo‘llashingiz mumkin.\n\n" +
		"Murojaatingiz chiroyli hujjat (PDF) ko‘rinishida rasmiylashtiriladi va rahbariyatga yetkaziladi.",
	MsgRequestPhone: "📱 <b>1-qadam</b>\n\nTelefon raqamingizni yuboring.\n\n" +
		"Pastdagi <b>«📱 Raqamni yuborish»</b> tugmasini bosing — raqam avtomatik yuboriladi.",
	MsgPhoneHint:    "💡 Tugma ko‘rinmasa, raqamingizni qo‘lda yozing.\nNamuna: <code>+998901234567</code>",
	MsgPhoneInvalid: "❌ Raqam noto‘g‘ri.\n\nIltimos, <b>«📱 Raqamni yuborish»</b> tugmasini bosing yoki raqamni shu ko‘rinishda yozing:\n<code>+998901234567</code>",
	MsgRegistered:   "✅ <b>Rahmat, ro‘yxatdan o‘tdingiz!</b>\n\nEndi quyidagi tugmalardan birini tanlang 👇",

	// ── Main menu ───────────────────────────────────────────────────
	MsgMainMenu: "🏠 <b>Asosiy menyu</b>\n\nNima qilmoqchisiz?",
	MsgHelp: "ℹ️ <b>Bot qanday ishlaydi?</b>\n\n" +
		"1️⃣ Murojaat turini tanlaysiz — shikoyat, taklif yoki minnatdorchilik.\n" +
		"2️⃣ Qaysi bo‘limga tegishli ekanini tanlaysiz.\n" +
		"3️⃣ Sinfni tanlaysiz.\n" +
		"4️⃣ Farzandingiz ism-familiyasini yozasiz.\n" +
		"5️⃣ Murojaat matnini yozasiz.\n" +
		"6️⃣ Xohlasangiz rasm qo‘shasiz (majburiy emas).\n" +
		"7️⃣ Tekshirib, «Yuborish» tugmasini bosasiz.\n\n" +
		"Hujjat PDF ko‘rinishida tayyorlanadi va rahbariyatga yuboriladi.\n\n" +
		"Har qanday paytda <b>/start</b> buyrug‘i bilan boshidan boshlashingiz mumkin.",

	// ── Appeal flow ─────────────────────────────────────────────────
	MsgStepDepartment: "📂 <b>1-qadam: Bo‘lim</b>\n\nMurojaatingiz qaysi bo‘limga tegishli?",
	MsgStepClass:      "🏫 <b>2-qadam: Sinf</b>\n\nFarzandingiz o‘qiydigan sinfni tanlang:",
	MsgNoClasses: "⚠️ Hozircha sinflar ro‘yxati kiritilmagan.\n\n" +
		"Iltimos, maktab ma’muriyatiga murojaat qiling yoki keyinroq urinib ko‘ring.",
	MsgStepChildName: "👶 <b>3-qadam: Farzandingiz</b>\n\nFarzandingizning ism-familiyasini yozing.\n\n" +
		"Namuna: <code>Aliyev Sardor</code>",
	MsgChildNameInvalid: "❌ Ism-familiya juda qisqa yoki noto‘g‘ri.\n\nIltimos, to‘liq yozing. Namuna: <code>Aliyev Sardor</code>",
	MsgStepTextComplaint: "📝 <b>4-qadam: Shikoyat matni</b>\n\nShikoyatingizni batafsil yozing.\n\n" +
		"💡 Nima bo‘lgani, qachon va kim bilan bog‘liqligini yozsangiz, masala tezroq hal bo‘ladi.",
	MsgStepTextProposal: "💡 <b>4-qadam: Taklif matni</b>\n\nTaklifingizni batafsil yozing.\n\n" +
		"💡 Nimani yaxshilash kerakligini va qanday qilib yaxshilash mumkinligini yozing.",
	MsgStepTextGratitude: "🙏 <b>4-qadam: Minnatdorchilik matni</b>\n\nMinnatdorchiligingizni yozing.\n\n" +
		"💡 Kimga va nima uchun minnatdorchilik bildirayotganingizni yozing.",
	MsgTextTooShort: "❌ Matn juda qisqa.\n\nIltimos, kamida %d ta harf yozing.",
	MsgTextTooLong:  "❌ Matn juda uzun.\n\nIltimos, %d ta belgidan oshmasin. Hozir: %d ta.",
	MsgStepPhotoAsk: "📷 <b>5-qadam: Rasm</b>\n\nMurojaatingizga rasm qo‘shmoqchimisiz?\n\n" +
		"Rasm qo‘shish <b>majburiy emas</b>.",
	MsgPhotoHint: "📎 Rasmni yuboring.\n\n" +
		"💡 Buning uchun pastdagi <b>📎 (qisqich)</b> belgisini bosing va rasmni tanlang.\n\n" +
		"Jami %d tagacha rasm yuborishingiz mumkin.\nTugagach — <b>«✅ Tayyor»</b> tugmasini bosing.",
	MsgPhotoAdded:      "✅ Rasm qabul qilindi (%d/%d).\n\nYana rasm yuborishingiz yoki <b>«✅ Tayyor»</b> tugmasini bosishingiz mumkin.",
	MsgPhotoMax:        "ℹ️ Siz maksimal %d ta rasm yubordingiz.\n\n<b>«✅ Tayyor»</b> tugmasini bosing.",
	MsgPhotoWrongInput: "📷 Iltimos, <b>rasm</b> yuboring yoki <b>«✅ Tayyor»</b> tugmasini bosing.",
	MsgConfirmTitle:    "🔎 <b>Tekshirib chiqing</b>\n\nHammasi to‘g‘ri bo‘lsa, «Yuborish» tugmasini bosing.",
	MsgSending:         "⏳ Hujjat tayyorlanmoqda, biroz kuting...",
	MsgSent: "✅ <b>Murojaatingiz yuborildi!</b>\n\n" +
		"Hujjat raqami: <b>%s</b>\n\n" +
		"Hujjat nusxasi sizga ham yuborildi. Rahbariyat ko‘rib chiqadi.\n\nMurojaatingiz uchun rahmat! 🙏",
	MsgSentNoAdmins: "✅ <b>Murojaatingiz qabul qilindi!</b>\n\n" +
		"Hujjat raqami: <b>%s</b>\n\n" +
		"⚠️ Hozircha rahbariyat botga ulanmagan, murojaat saqlab qo‘yildi va ular ulanishi bilan yetkaziladi.",
	MsgSentNoDocument: "✅ <b>Murojaatingiz qabul qilindi!</b>\n\n" +
		"Hujjat raqami: <b>%s</b>\n\n" +
		"⚠️ Hujjatni tayyorlashda muammo bo‘ldi, ammo murojaatingiz saqlandi va rahbariyatga matn ko‘rinishida yuborildi.",
	MsgCancelled: "❌ Bekor qilindi.\n\nAsosiy menyuga qaytdingiz.",

	// ── My appeals ──────────────────────────────────────────────────
	MsgMyAppealsTitle: "📂 <b>Mening murojaatlarim</b>\n\nOxirgi %d ta murojaat:",
	MsgNoAppeals:      "📭 Sizda hali murojaat yo‘q.\n\nAsosiy menyudan birinchi murojaatingizni yuborishingiz mumkin.",

	// ── Settings ────────────────────────────────────────────────────
	MsgSettingsTitle:   "⚙️ <b>Sozlamalar</b>\n\nNimani o‘zgartirmoqchisiz?",
	MsgLanguageChanged: "✅ Til o‘zgartirildi.",

	// ── Labels ──────────────────────────────────────────────────────
	LblType:       "Murojaat turi",
	LblDepartment: "Bo‘lim",
	LblClass:      "Sinf",
	LblChild:      "Farzand",
	LblPhone:      "Telefon",
	LblUsername:   "Telegram",
	LblDate:       "Sana va vaqt",
	LblText:       "Matn",
	LblPhotos:     "Rasmlar",
	LblReference:  "Hujjat raqami",
	LblNone:       "yo‘q",

	TypeComplaint: "Shikoyat",
	TypeProposal:  "Taklif",
	TypeGratitude: "Minnatdorchilik",

	// ── Buttons ─────────────────────────────────────────────────────
	BtnSharePhone: "📱 Raqamni yuborish",
	BtnComplaint:  "📝 Shikoyat",
	BtnProposal:   "💡 Taklif",
	BtnGratitude:  "🙏 Minnatdorchilik",
	BtnMyAppeals:  "📂 Mening murojaatlarim",
	BtnSettings:   "⚙️ Sozlamalar",
	BtnHelp:       "ℹ️ Yordam",
	BtnAdminPanel: "🛠 Admin panel",
	BtnChangeLang: "🌐 Tilni o‘zgartirish",
	BtnAddPhoto:   "📷 Ha, rasm qo‘shaman",
	BtnNoPhoto:    "➡️ Yo‘q, davom etish",
	BtnPhotoDone:  "✅ Tayyor",
	BtnSend:       "✅ Yuborish",
	BtnCancel:     "❌ Bekor qilish",
	BtnBack:       "⬅️ Orqaga",
	BtnMainMenu:   "🏠 Asosiy menyu",

	// ── Admin ───────────────────────────────────────────────────────
	MsgAdminPanel: "🛠 <b>Admin panel</b>\n\nKerakli bo‘limni tanlang:",
	MsgRoleNowAdmin: "🛠 <b>Sizga ma’muriyat huquqlari berildi.</b>\n\n" +
		"Endi ota-onalarning murojaatlari to‘g‘ridan-to‘g‘ri sizga keladi.\n" +
		"Quyidagi menyu orqali bo‘limlar va sinflarni boshqarishingiz mumkin 👇",
	MsgRoleNowParent: "ℹ️ <b>Ma’muriyat huquqlaringiz olib tashlandi.</b>\n\n" +
		"Endi siz ota-ona sifatida shikoyat, taklif yoki minnatdorchilik yuborishingiz mumkin 👇",
	MsgAdminCannotSubmit: "ℹ️ Siz maktab ma’muriyati vakilisiz.\n\n" +
		"Ma’muriyat murojaatlarni <b>qabul qiladi</b> — yangi murojaat yuborish ota-onalar uchun.",
	MsgAdminOnly: "⛔️ Bu bo‘lim faqat ma’muriyat uchun.",
	MsgClassPrompt: "➕ <b>Yangi sinf</b>\n\nSinf nomini yozing.\n\n" +
		"Namuna: <code>5-A</code>, <code>9-B</code>, <code>11-V</code>",
	MsgClassAdded:     "✅ <b>%s</b> sinfi qo‘shildi.",
	MsgClassExists:    "⚠️ <b>%s</b> sinfi allaqachon mavjud.",
	MsgClassInvalid:   "❌ Sinf nomi noto‘g‘ri. 1 tadan 20 tagacha belgi bo‘lishi kerak.",
	MsgClassList:      "🏫 <b>Sinflar</b>\n\nJami: %d ta\n\nO‘zgartirish uchun sinf ustiga bosing:",
	MsgClassListEmpty: "📭 Hali sinf qo‘shilmagan.\n\n«➕ Sinf qo‘shish» tugmasi orqali qo‘shing.",
	MsgClassDeleted:   "🗑 <b>%s</b> sinfi o‘chirildi.",
	MsgClassToggled:   "✅ <b>%s</b> holati o‘zgartirildi.",
	MsgAdminStats: "📊 <b>Statistika</b>\n\n" +
		"👥 Foydalanuvchilar: <b>%d</b>\n" +
		"🏫 Sinflar: <b>%d</b>\n" +
		"📂 Bo‘limlar: <b>%d</b>\n\n" +
		"📝 Shikoyatlar: <b>%d</b>\n" +
		"💡 Takliflar: <b>%d</b>\n" +
		"🙏 Minnatdorchiliklar: <b>%d</b>\n" +
		"━━━━━━━━━━━━━━\n" +
		"📨 Jami murojaatlar: <b>%d</b>",
	MsgAdminRecent:     "📬 <b>Oxirgi murojaatlar</b>\n\nHujjatni qayta olish uchun ustiga bosing:",
	MsgAdminNoRecent:   "📭 Hali murojaat yo‘q.",
	MsgNewAppealHeader: "🔔 <b>YANGI MUROJAAT</b>",

	BtnAddClass:      "➕ Sinf qo‘shish",
	BtnManageClasses: "🏫 Sinflar",
	BtnStats:         "📊 Statistika",
	BtnRecent:        "📬 Oxirgi murojaatlar",

	// ── Errors ──────────────────────────────────────────────────────
	ErrGeneric:       "❌ Xatolik yuz berdi. Iltimos, qaytadan urinib ko‘ring yoki /start buyrug‘ini yuboring.",
	ErrNotRegistered: "👋 Avval ro‘yxatdan o‘ting.\n\n/start buyrug‘ini yuboring.",
	ErrSessionLost:   "⏱ Sessiya eskirdi.\n\nIltimos, boshidan boshlang — /start",

	// ── PDF ─────────────────────────────────────────────────────────
	PDFTitleComplaint: "SHIKOYAT",
	PDFTitleProposal:  "TAKLIF",
	PDFTitleGratitude: "MINNATDORCHILIK",
	PDFSectionDetails: "MUROJAAT MA'LUMOTLARI",
	PDFSectionText:    "MUROJAAT MATNI",
	PDFSectionPhotos:  "ILOVA QILINGAN RASMLAR",
	PDFPhotoNo:        "Rasm %d / %d",
	PDFFooter:         "Ushbu hujjat Telegram bot orqali avtomatik tayyorlandi",
	PDFPage:           "%d-bet",

	// ── Admin: departments ──────────────────────────────────────────
	MsgDeptList:      "📂 <b>Bo‘limlar</b>\n\nJami: %d ta\n\nO‘zgartirish uchun bo‘lim ustiga bosing:",
	MsgDeptListEmpty: "📭 Hali bo‘lim qo‘shilmagan.\n\n«➕ Bo‘lim qo‘shish» tugmasi orqali qo‘shing.",
	MsgDeptMenu:      "📂 <b>%s</b>\n\n🇺🇿 %s\n🇷🇺 %s\n🇬🇧 %s\n\nHolati: %s\n\nNima qilmoqchisiz?",
	MsgDeptPromptUz: "🇺🇿 <b>1/3 — O‘zbekcha nomi</b>\n\nBo‘lim nomini o‘zbekcha yozing.\n\n" +
		"💡 Boshiga belgi (emoji) qo‘yishingiz mumkin.\nNamuna: <code>🍽 Oshxona</code>",
	MsgDeptPromptRu: "🇷🇺 <b>2/3 — Ruscha nomi</b>\n\nShu bo‘limning ruscha nomini yozing.\n\n" +
		"Namuna: <code>Столовая</code>\n\nBilmasangiz — «⏭ O‘tkazib yuborish» tugmasini bosing.",
	MsgDeptPromptEn: "🇬🇧 <b>3/3 — Inglizcha nomi</b>\n\nShu bo‘limning inglizcha nomini yozing.\n\n" +
		"Namuna: <code>Canteen</code>\n\nBilmasangiz — «⏭ O‘tkazib yuborish» tugmasini bosing.",
	MsgDeptAdded:         "✅ <b>%s</b> bo‘limi qo‘shildi.",
	MsgDeptUpdated:       "✅ <b>%s</b> bo‘limi yangilandi.",
	MsgDeptDeleted:       "🗑 <b>%s</b> bo‘limi o‘chirildi.",
	MsgDeptToggled:       "✅ <b>%s</b> holati o‘zgartirildi.",
	MsgDeptExists:        "⚠️ <b>%s</b> nomli bo‘lim allaqachon mavjud.",
	MsgDeptInvalid:       "❌ Bo‘lim nomi noto‘g‘ri. 2 tadan 60 tagacha belgi bo‘lishi kerak.",
	MsgDeptDeleteConfirm: "🗑 <b>%s</b> bo‘limini o‘chirmoqchimisiz?\n\nEski murojaatlar saqlanib qoladi.",
	MsgNoDepartments: "⚠️ Hozircha bo‘limlar ro‘yxati kiritilmagan.\n\n" +
		"Iltimos, maktab ma’muriyatiga murojaat qiling yoki keyinroq urinib ko‘ring.",

	BtnManageDepartments: "📂 Bo‘limlar",
	BtnAddDept:           "➕ Bo‘lim qo‘shish",
	BtnEditDept:          "✏️ Nomini o‘zgartirish",
	BtnDelete:            "🗑 O‘chirish",
	BtnToggleOn:          "👁 Ko‘rsatish",
	BtnToggleOff:         "🚫 Yashirish",
	BtnSkip:              "⏭ O‘tkazib yuborish",
	BtnYesDelete:         "🗑 Ha, o‘chirilsin",
}
