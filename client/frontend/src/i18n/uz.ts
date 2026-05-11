import type { Dict } from "./strings";

export const uz: Dict = {
  // Welcome
  "welcome.greeting": "Xush kelibsiz",
  "welcome.subtitle": "Yangi portal oching yoki do'stingiznikiga qo'shiling.",
  "welcome.tagline": "To'g'ridan-to'g'ri ulanish.\nOrada hech qanday server yo'q.",
  "welcome.nickname.label": "Taxallus",
  "welcome.nickname.placeholder": "alice",
  "welcome.portalId.label": "Portal ID",
  "welcome.portalId.placeholder": "123456",
  "welcome.code.label": "Kod",
  "welcome.code.placeholder": "654321",
  "welcome.create": "Portal yaratish",
  "welcome.create.card_title": "Yangi portal",
  "welcome.create.card_hint": "Bir bosishda yarating va do'stingizga ID + kod yuboring.",
  "welcome.join": "Qo'shilish",
  "welcome.join.card_title": "Qo'shilish",
  "welcome.join.card_hint": "Sizga berilgan ID va 6 xonali kodni kiriting.",
  "welcome.creating": "Yaratish",
  "welcome.joining": "Qo'shilish",
  "welcome.cancel": "Bekor qilish",
  "welcome.back": "Orqaga",
  "welcome.recent": "Yaqindagi portallar",
  "welcome.recent.empty": "Hozircha hech qanday portalga kirmagansiz.",
  "welcome.recent.remove": "Tarixdan o'chirish",
  "welcome.recent.background": "Fonda ulash (xonaga kirmasdan)",
  "welcome.rename.title": "Nomlash",
  "welcome.rename.placeholder": "masalan: Oilaviy portal",
  "welcome.rename.save": "Saqlash",
  "welcome.rename.cancel": "Bekor qilish",
  "welcome.signout": "Chiqish",
  "welcome.active": "Faol ulanishlar",
  "welcome.active.foreground": "ko'rinmoqda",
  "welcome.active.peers": "ulanish",
  "welcome.active.leave": "Bu portaldan chiqish",
  "welcome.create.background": "Fonda yaratish (xonaga kirmasdan)",
  "welcome.join.background": "Fonda ulash — xonaga kirmasdan",
  "welcome.join.background_short": "Fonda",
  "welcome.error.empty_nickname": "Avval taxallus yozing",
  "welcome.error.empty_id_or_code": "ID va kod ikkalasi kerak",
  "welcome.error.no_such_portal":
    "Bu portal allaqachon yopilgan. Egasi chiqib ketgan bo'lsa, ID + KOD avtomatik bekor qilinadi.",
  "welcome.error.wrong_code": "Kod noto'g'ri. Qaytadan tekshiring yoki egasidan so'rang.",
  "welcome.error.full": "Portal to'lib qolgan (16 ta peer max).",
  "welcome.error.locked": "Egasi portalni qulflagan. Ochilishini kuting.",
  "welcome.error.nickname_taken": "Bu taxallus portalda allaqachon ishlatilmoqda. Boshqa nom tanlang.",
  "welcome.create_new_link": "Yangi portal yaratish →",
  "welcome.symmetric_nat_warning":
    "Simmetrik NAT aniqlandi. To'g'ridan-to'g'ri ulanish ishlamasligi mumkin. Bepul TURN serveri orqali avtomatik o'tib ulanadi — hech narsa qilish kerak emas. Ulanish sekinroq bo'lishi mumkin.",

  // Banner messages
  "banner.portal_closed": "Portal yopildi.",
  "banner.signaling_disconnected": "Signal serveri uzildi — qayta ulanmoqda...",
  "banner.signaling_reconnected": "Qayta ulandi ✓",
  "banner.reconnect_failed": "Qayta ulanish muvaffaqiyatsiz. Qaytadan portal yarating.",

  // Settings
  "settings.title": "Sozlamalar",
  "settings.tab.profile": "Profil",
  "settings.tab.network": "Tarmoq",
  "settings.tab.identity": "Identitet",
  "settings.tab.diagnostics": "Diagnostika",
  "settings.tab.files": "Fayllar",
  "settings.tab.activity": "Faollik",
  "settings.tab.history": "Tarix",
  "settings.tab.about": "Haqida",
  "settings.tab.logs": "Loglar",
  "settings.back": "Orqaga",
  "settings.language": "Til",
  "settings.language.uz": "O'zbek",
  "settings.language.en": "English",

  // Profile
  "settings.profile.title": "Profil",
  "settings.profile.username": "Foydalanuvchi nomi",
  "settings.profile.account_status": "Akkaunt holati",
  "settings.profile.account_local": "Mahalliy akkaunt",
  "settings.profile.account_local_hint":
    "Akkaunt ma'lumotlari faqat shu kompyuterda saqlanadi. Hech qanday server bilan sinxronlash yo'q.",
  "settings.profile.signout": "Akkauntdan chiqish",
  "settings.profile.reset": "Akkauntni tiklash",
  "settings.profile.reset_warn":
    "Akkauntni tiklash foydalanuvchi nomi, parolni va portal tarixini o'chiradi. Tarmoq sozlamalari saqlanadi. Bu amalni qaytarib bo'lmaydi.",
  "settings.profile.reset_confirm": "Ha, tiklash",
  "settings.profile.reset_cancel": "Bekor qilish",
  "settings.profile.member_since": "Akkaunt yaratildi",
  "settings.profile.device_name": "Qurilma nomi",
  "settings.profile.device_name_placeholder": "uy, ish, mac, win 64…",
  "settings.profile.device_name_hint":
    "Boshqa qurilmalardan ham shu akkaunt bilan kirsangiz, peer'lar qaysi qurilmadan ekanligingizni ko'radi:",
  "settings.profile.autorun": "Tizim yuklanganda avtomatik ishga tushish",
  "settings.profile.autorun_on": "Yoqilgan — Portal tizim yonganda avtomatik ochiladi",
  "settings.profile.autorun_off": "O'chirilgan — qo'lda ochish kerak",
  "settings.profile.autorun_hint":
    "Yoqilgan bo'lsa, OS startup hook yoziladi (macOS LaunchAgent / Windows registry / Linux .desktop). Saqlangan portallar avtomatik qayta ulanadi.",

  // Network
  "settings.network.title": "Tarmoq",
  "settings.network.signaling_label": "Signal serveri URL",
  "settings.network.signaling_hint": "Standart:",
  "settings.network.save": "Saqlash",
  "settings.network.saved": "Saqlandi ✓",
  "settings.network.turn_title": "TURN relay",
  "settings.network.turn_managed":
    "TURN serveri avtomatik server tomonidan ta'minlanadi.",
  "settings.network.turn_managed_hint":
    "Simmetrik NAT yoki CGNAT ortida bo'lsangiz, signal serveri qisqa muddatli kredensiallarni o'zi yuboradi — siz hech narsa sozlashingiz shart emas.",
  "settings.network.turn_status_label": "Holat",
  "settings.network.turn_status_active": "Faol — server tomonidan boshqarilmoqda",
  "settings.network.turn_status_idle": "Tayyor — kerak bo'lganda yoqiladi",

  // Diagnostics
  "settings.diag.title": "Diagnostika",
  "settings.diag.nat_label": "NAT turi",
  "settings.diag.detecting": "Aniqlanmoqda...",
  "settings.diag.local": "lokal",
  "settings.diag.public": "tashqi",
  "settings.diag.refresh": "Yangilash",

  // Files
  "settings.files.title": "Fayllar",
  "settings.files.save_dir_label": "Qabul qilingan fayllar joylashuvi",
  "settings.files.open": "Ochish",

  // Activity
  "settings.activity.title": "Peer ulanishlari",
  "settings.activity.empty":
    "Hech kim hali sizning servislaringizga ulanmagan. Mehmon \"Ulash\" bosganida shu yerda yozuv paydo bo'ladi.",
  "settings.activity.loading": "Yuklanmoqda…",

  // Logs
  "settings.logs.title": "Loglar",
  "settings.logs.hint":
    "Dastur ichidagi voqealar oxirgi 500 qatorda saqlanadi va bir kunlik fayl sifatida ham diskka yoziladi.",
  "settings.logs.show": "Oxirgi 200 qatorni ko'rish",
  "settings.logs.open_folder": "Fayl papkasini ochish",
  "settings.logs.copy": "Nusxa",
  "settings.logs.clear": "Tozalash",
  "settings.logs.close": "Yopish",
  "settings.logs.empty": "(bo'sh)",

  // History
  "settings.history.title": "Yaqindagi portallar",
  "settings.history.empty": "Hozircha tarix bo'sh.",
  "settings.history.clear": "Tarixni tozalash",

  // About
  "settings.about.title": "Haqida",
  "settings.about.version": "Portal versiyasi",
  "settings.about.check_now": "Hozir tekshirish",
  "settings.about.checking": "Tekshirilmoqda...",
  "settings.about.up_to_date": "✓ Eng so'nggi versiyada",
  "settings.about.new_available": "Yangi versiya mavjud:",
  "settings.about.open_download": "Yuklab olish sahifasini ochish",
  "settings.about.crashes_label": "Halokat hisobotlari",
  "settings.about.crashes_hint":
    "Hisobotlar faqat lokal saqlanadi. Ularda stack trace bor (uy papkasi yashirilgan), portal ID, peer ID, IP yoki xabar tarkibi yo'q.",
  "settings.about.crashes_empty": "Halokat hisobotlari yo'q.",
  "settings.about.crashes_more": "+ yana {0} ta",
  "settings.about.refresh": "Yangilash",
  "settings.about.clear_all": "Hammasini o'chirish",
  "settings.about.docs": "Hujjatlar",

  // Common
  "common.owner": "egasi",
  "common.connecting": "Ulanmoqda...",
  "common.connected": "Ulangan",
  "common.failed": "Failed",
  "common.tooltip.settings": "Sozlamalar",

  // Updater
  "update.available": "Yangi versiya mavjud:",
  "update.download": "Sahifa",
  "update.install": "Yangilash",
  "update.installing": "Yangilanmoqda…",
  "update.install_failed": "Yangilash muvaffaqiyatsiz tugadi: ",
  "update.dismiss": "Yopish",

  // Portal header
  "header.copy_both_tooltip": "ID + KOD ni birga nusxa olish",
  "header.copied": "Nusxalandi",
  "header.leave": "Chiqish",
  "header.dashboard": "Boshqaruv paneliga qaytish (portal ulangan holda qoladi)",
  "header.dashboard_short": "Asosiy",
  "header.close": "Bu portalni yopish (uzilish)",
  "header.switcher": "Portallarni o'zgartirish",
  "header.switcher.title": "Faol portallar",
  "header.qr.close": "yopish",
  "header.qr.help": "QR ni do'stingizning kamerasiga tutsangiz,\nportalga to'g'ridan-to'g'ri kiradi.",
  "header.copy_tooltip": "Nusxa olish",
  "header.reveal_first": "Avval ko'rsatish kerak",
  "header.hide": "Yashirish",
  "header.show": "Ko'rsatish",

  // Chat
  "chat.placeholder": "Xabar yozing...",
  "chat.empty": "Hozircha xabarlar yo'q.",
  "chat.empty_hint": "Birinchi bo'lib salom yozing yoki faylni shu yerga tashlang.",
  "chat.send": "Yuborish",
  "chat.send_file": "Fayl yuborish",
  "chat.drop_to_send": "Tashlang — meshda yuborish boshlanadi",

  // Lock screen
  "lock.signup.title": "Ro'yxatdan o'tish",
  "lock.signup.subtitle":
    "Akkaunt yaratib qo'ying — keyingi safar parol bilan kirasiz.",
  "lock.signup.username.label": "Foydalanuvchi nomi",
  "lock.signup.username.placeholder": "alice",
  "lock.signup.password.label": "Parol",
  "lock.signup.password.placeholder": "Kamida 8 ta belgi, kuchli parol",
  "lock.signup.confirm.label": "Parolni takrorlang",
  "lock.signup.confirm.placeholder": "Yana bir marta",
  "lock.signup.submit": "Davom etish",
  "lock.signup.submitting": "Saqlanmoqda...",
  "lock.signup.replace_warning":
    "Bu kompyuterda allaqachon akkaunt bor. Yangisini yaratsangiz, eski akkaunt o'chiriladi (portal tarixi saqlanadi).",
  "lock.signup.error.username_empty": "Foydalanuvchi nomini kiriting.",
  "lock.signup.error.username_too_long": "Foydalanuvchi nomi 24 ta belgidan oshmasligi kerak.",
  "lock.signup.error.password_too_short": "Parol kamida 8 ta belgidan iborat bo'lishi kerak.",
  "lock.signup.error.password_weak":
    "Parol kuchsiz. Quyidagi mezonlarning hammasini bajaring.",
  "lock.signup.error.mismatch": "Parollar bir-biriga mos kelmadi.",
  "lock.signup.error.failed": "Saqlash imkonsiz bo'ldi. Qayta urinib ko'ring.",
  "lock.signup.hint":
    "Eslab qoling: parol unutsangiz, tiklash mumkin emas — faqat tarix tozalanadi va qaytadan boshlaysiz.",
  "lock.signup.strength.title": "Parol mezonlari:",
  "lock.signup.strength.length": "Kamida 8 ta belgi",
  "lock.signup.strength.lower": "Kichik harf (a-z)",
  "lock.signup.strength.upper": "Katta harf (A-Z)",
  "lock.signup.strength.digit": "Raqam (0-9)",
  "lock.signup.strength.special": "Maxsus belgi (!@#$...)",
  "lock.signin.title": "Kirish",
  "lock.signin.subtitle": "Akkauntingizga kirish uchun ma'lumotlarni kiriting.",
  "lock.signin.username.label": "Foydalanuvchi nomi",
  "lock.signin.username.placeholder": "alice",
  "lock.signin.password.label": "Parol",
  "lock.signin.password.placeholder": "••••",
  "lock.signin.submit": "Kirish",
  "lock.signin.checking": "Tekshirilmoqda...",
  "lock.signin.error.wrong": "Foydalanuvchi nomi yoki parol noto'g'ri.",
  "lock.signin.error.locked":
    "Juda ko'p urinish bo'ldi. {0} soniyadan so'ng qayta urinib ko'ring.",
  "lock.signin.forgot": "Parolni unutdingizmi?",
  "lock.remember": "Bu qurilmada eslab qol",
  "lock.remember.hint":
    "Yoqilgan bo'lsa, ilovani qaytadan ochganda parol so'ralmaydi. Faqat o'zingizning kompyuteringizda yoqing.",
  "lock.reset.title": "Akkauntni tiklash?",
  "lock.reset.body":
    "Parol esga tushmasa, faqatgina mahalliy ma'lumotlarni tozalab qaytadan boshlash mumkin: portal tarixi, foydalanuvchi nomi va parol o'chiriladi. Tarmoq sozlamalari (signal serveri, TURN) saqlanadi.",
  "lock.reset.confirm": "Ha, tozalash",
  "lock.reset.cancel": "Bekor qilish",
};
