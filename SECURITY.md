# Xavfsizlik siyosati

## Zaiflik haqida xabar berish

Agar Portal da xavfsizlik muammosini topgan bo'lsangiz, ommaviy
oshkor qilishdan oldin shaxsan xabar bering.

- **Email:** *(loyiha muallifi email — TBD; hozircha `git log` dagi
  manzildan foydalaning)*
- **Mavzu qatori:** triage qilish oson bo'lishi uchun `[security]` bilan
  boshlang.

Iltimos, quyidagilarni qo'shing:

1. Muammo va uning ta'siri tavsifi.
2. Qayta ishlash bosqichlari yoki proof-of-concept.
3. Ma'lumotlarni boshqa joyda allaqachon ulashganmisiz.

Hisobotlar 72 soat ichida tasdiqlanadi. Tasdiqdan keyin 14 kun ichida
yamoq chiqarish maqsadimiz va xabar beruvchini release notes da
ko'rsatamiz (agar nomi ko'rinmasligini so'ramasa).

---

## Qamrov

Qamrovda:

- Desktop client (`client/`)
- Shared simli protokol (`shared/protocol/`)
- Signal serveri dizayni (deploy qilingan instance — operatorning
  mas'uliyati, lekin dizayn darajasidagi muammolar qamrovda)

Qamrovdan tashqarida:

- Foydalanuvchining yomonniyatli uchinchi tomon signal serverini
  o'rnatishini talab qiladigan zaifliklar. O'zingizniki ishga tushirish
  qo'llab-quvvatlanadi, lekin tahdid modeli bu endpoint ga ishonchliligini
  taxmin qiladi.
- Per-IP rate limit ni hisobga olmaydigan 6 xonali portal kodiga
  nazariy brute-force hujumlar (qarang [ARCHITECTURE.md](ARCHITECTURE.md)).
- Bog'liqliklardagi xatolar (Go stdlib, Wails, pion/webrtc,
  gorilla/websocket). Iltimos, ularni upstream ga xabar bering.

---

## Biz nima va'da qilamiz

- Signal serveri ilova trafigini o'qiy olmaydi. Agar buni qila olishning
  yo'lini topsangiz — bu kritik issue.
- A portalidagi peer Portal orqali B portalidagi peer ga yeta olmaydi.
  Agar yo'lini topsangiz — bu kritik issue.
- Uzatilgan `webrtc.*` xabarlardagi `from` maydoni server tomonidan
  shtamplangan va soxtalashtirib bo'lmaydi. Agar yo'lini topsangiz — bu
  yuqori darajadagi issue.

---

## Self-host qiluvchilar uchun hardening maslahatlari

Agar o'z signal serveringizni boshqarsangiz:

- TLS ishlating. Cloudflare Origin Certificate lari bepul; deploy
  qo'llanmaga qarang.
- Cloudflare ortida (yoki WebSocket qo'llab-quvvatlovchi DDoS-mitigating
  CDN ortida) ishga tushiring.
- Og'ir foydalanish kutilsa, standart bo'lmagan rate limit larni
  o'rnating; standart qiymatlar konservativ.
- Birga keladigan systemd hardening profili bilan root bo'lmagan
  foydalanuvchi sifatida ishga tushiring.
- Origin Certificate lar muddati tugashidan oldin yangilang
  (Cloudflare standart 15 yil, lekin baribir — kalendarga eslatma
  qo'ying).
