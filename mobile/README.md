# Portal Mobile (Android) — Beta

Bu papka Portal'ning Android client'i. **Beta holatida** — desktop versiya
bilan to'liq simli protokol mosligiga ega, lekin v0.4.0 reliz tarkibiga
hali kirmaydi.

This directory contains Portal's Android client. It is **in beta** —
fully wire-protocol compatible with the desktop version, but not part
of the v0.4.0 release.

---

## Hozirgi imkoniyatlar / Current features

- Portal yaratish va qo'shilish (6 xonali ID + kod) / Create and join portal (6-digit ID + code)
- To'liq mesh WebRTC handshake (desktop bilan birga ishlaydi)
- Shifrlangan broadcast chat (PBKDF2-SHA256 + NaCl secretbox)
- RTT heartbeat va peer ro'yxati / RTT heartbeat and peer roster
- Bo'lakli fayl uzatish / Chunked file transfer
- QR kod ko'rsatish va skanerlash / QR code display + scan
- DataStore sozlamalar (taxallus, signal URL, oxirgi 10 portal)
- Joiner uchun avtomatik qayta ulanish (8 marta, exponential backoff)
- Foreground service — backgroundga ketganda ulanishni saqlaydi

---

## Hali yo'q / Not yet implemented

- TCP/UDP proxy (`portal expose` / `portal dial`)
- Servislar paneli (UI)
- NAT turi aniqlash / NAT type detection
- Join-by-nick (taxallus orqali so'rov)
- PCP-1 (Portal Cipher Protocol v1) — desktop'da v0.4.0'da bor;
  mobile keyingi reliz'da (v0.5.0) PCP-1 ga o'tadi.

Bu rejada / These are planned. Mobile version Portal'ning desktop
versiyasidan biroz orqada qoladi — bu odatiy hol.

---

## v0.4.0 reliz uchun status

Mobile **v0.4.0 GitHub Releases'iga kirmaydi** chunki:
1. PCP-1 hali mobile'da yo'q (faqat desktop'da). Wire mosligi xaqiqatda
   bor lekin yangi crypto layer mobile'da ham bo'lishi kerak.
2. Play Store distributsiyasi alohida pipeline talab qiladi (signing,
   App Bundle, listing).
3. Foydalanuvchi tajribasi hali sayqallangani yo'q (icon, splash,
   permissions ekranlari).

**Reja:** v0.4.1 yoki v0.5.0 bilan birga — Play Store internal testing
track, keyin closed alpha.

For v0.4.0 release: mobile is **NOT bundled in GitHub Releases**.
Reasons listed above. Plan is to ship via Play Store internal testing
track in v0.4.1 or v0.5.0.

---

## Build

JDK kerak (Android Studio'ning JBR'i tavsiya etiladi):
JDK required (Android Studio's JBR is recommended):

```bash
export JAVA_HOME="/Applications/Android Studio.app/Contents/jbr/Contents/Home"
cd mobile
./gradlew assembleDebug
./gradlew installDebug   # connected device/emulator'ga o'rnatish
```

Test'lar:

```bash
./gradlew test                       # host JVM unit tests
./gradlew connectedAndroidTest       # device required (crypt module needs lazysodium-android)
```

Toolchain pinlari [CLAUDE.md](CLAUDE.md) da batafsil.
Toolchain pins detailed in [CLAUDE.md](CLAUDE.md).

---

## Sinab ko'rish istagan jasurlar uchun / For brave testers

Hozirgi APK shaxsiy build sifatida ishlatish uchun yetarli darajada
barqaror. Lekin:

- **Sign qilingan reliz APK yo'q** — debug APK manbadan build
  qilishingiz kerak.
- **Auto-update yo'q** — yangi versiya chiqsa qo'lda yangilash kerak.
- **Crash reporting yo'q** — xato topganingizda iltimos GitHub'da issue
  oching ([Issue Templates](../.github/ISSUE_TEMPLATE/)).

The APK is stable enough for personal use as a self-built debug build.
But: no signed release APK, no auto-update, no crash reporting yet.
Please file issues if you find bugs.

---

## Hissa qo'shish / Contributing

Mobile development uchun [CONTRIBUTING.md](../CONTRIBUTING.md) va bu
papkadagi [CLAUDE.md](CLAUDE.md) ni o'qing. Asosiy qoidalar:

- Wire protokol o'zgarishi taqiqlanadi (faqat ixtiyoriy maydon qo'shish).
  Desktop bilan moslik buzilsa, foydalanuvchilar zarar ko'radi.
- `app/src/main/java/uz/aihealth/portal_mobile/` Go reference'ga oyna
  oydek mos kelishi kerak (mirror struktura — `protocol/`, `crypt/`,
  `signaling/`, `peer/`, `mesh/`, `transfer/`).

Read [CONTRIBUTING.md](../CONTRIBUTING.md) and [CLAUDE.md](CLAUDE.md) in
this folder for mobile development. Key rules: no breaking wire
changes; mirror Go reference structure.
