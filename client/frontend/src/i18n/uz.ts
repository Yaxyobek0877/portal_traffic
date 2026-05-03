import type { Dict } from "./strings";

export const uz: Dict = {
  // Welcome
  "welcome.tagline": "To'g'ridan-to'g'ri ulanish.\nOrada hech qanday server yo'q.",
  "welcome.nickname.label": "Taxallus",
  "welcome.nickname.placeholder": "alice",
  "welcome.portalId.label": "Portal ID",
  "welcome.portalId.placeholder": "123456",
  "welcome.code.label": "Kod",
  "welcome.code.placeholder": "654321",
  "welcome.create": "Portal yaratish",
  "welcome.join": "Qo'shilish",
  "welcome.creating": "Yaratish",
  "welcome.joining": "Qo'shilish",
  "welcome.cancel": "Bekor qilish",
  "welcome.back": "Orqaga",
  "welcome.recent": "Yaqindagilar",
  "welcome.error.empty_nickname": "Avval taxallus yozing",
  "welcome.error.empty_id_or_code": "ID va kod ikkalasi kerak",
  "welcome.error.no_such_portal":
    "Bu portal allaqachon yopilgan. Egasi chiqib ketgan bo'lsa, ID + KOD avtomatik bekor qilinadi.",
  "welcome.error.wrong_code": "Kod noto'g'ri. Qaytadan tekshiring yoki egasidan so'rang.",
  "welcome.error.full": "Portal to'lib qolgan (16 ta peer max).",
  "welcome.error.locked": "Egasi portalni qulflagan. Ochilishini kuting.",
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
  "settings.tab.network": "Tarmoq",
  "settings.tab.identity": "Identitet",
  "settings.tab.diagnostics": "Diagnostika",
  "settings.tab.files": "Fayllar",
  "settings.tab.history": "Tarix",
  "settings.tab.about": "Haqida",
  "settings.back": "Orqaga",
  "settings.language": "Til",
  "settings.language.uz": "O'zbek",
  "settings.language.en": "English",

  // Common
  "common.owner": "egasi",
  "common.connecting": "Ulanmoqda...",
  "common.connected": "Ulangan",
  "common.failed": "Failed",
  "common.tooltip.settings": "Sozlamalar",

  // Updater
  "update.available": "Yangi versiya mavjud:",
  "update.download": "Yuklab olish",
  "update.dismiss": "Yopish",

  // Portal header
  "header.copy_both_tooltip": "ID + KOD ni birga nusxa olish",
  "header.copied": "Nusxalandi",
  "header.leave": "Chiqish",
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
};
