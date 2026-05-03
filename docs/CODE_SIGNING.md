# Code signing — qo'llanma / Setup guide

Bu hujjat Portal binarlarini imzolash uchun zarur narsalarni va GitHub
Actions release pipeline'iga qanday ulashni tushuntiradi. Imzolash
ixtiyoriy: agar secret'lar `Settings → Secrets and variables → Actions`
da bo'lmasa, pipeline imzolanmagan binarlar chiqaradi.

This document explains code signing for Portal binaries and how to wire
it into the GitHub Actions release pipeline. Signing is optional: if
the secrets aren't set in the repo, the pipeline produces unsigned
binaries (which work, but trigger OS warnings).

---

## macOS — Apple Developer ID

### Nima kerak / Requirements

- **Apple Developer Program** a'zoligi: $99/yil
  ([developer.apple.com/programs](https://developer.apple.com/programs/))
- **Developer ID Application** sertifikati (Apple Developer'da yaratiladi)
- **Apple ID** + **app-specific password** (notarization uchun)
- **Team ID** (Apple Developer Account ostida ko'rsatilgan, 10-belgili)

### Qadamlar / Steps

1. **Developer ID Application** sertifikatini Keychain Access'da
   yarating yoki yuklab oling
2. Sertifikatni `.p12` formatda eksport qiling (parol qo'ying)
3. `.p12` ni base64 ga aylantiring:
   ```bash
   base64 -i developer_id.p12 -o developer_id.p12.b64
   ```
4. App-specific password yarating: [appleid.apple.com](https://appleid.apple.com)
   → Sign-In and Security → App-Specific Passwords

### Repo secret'lari / Repo secrets

GitHub repo → Settings → Secrets and variables → Actions:

| Secret nomi / Name | Qiymat / Value |
| --- | --- |
| `APPLE_ID` | Apple ID email (notarization owner) |
| `APPLE_TEAM_ID` | 10-belgili Team ID (e.g., `ABC1234567`) |
| `APPLE_APP_PASSWORD` | app-specific password |
| `APPLE_DEVELOPER_ID_CERT` | base64'lik `.p12` content (yuqorida) |
| `APPLE_DEVELOPER_ID_CERT_PASSWORD` | `.p12` ni eksport qilishda qo'ygan parol |

Secret'lar qo'shilgandan keyin keyingi tag bosishda imzolash avtomatik
ishlaydi.

---

## Windows — Authenticode

### Variantlar / Options

**1. EV (Extended Validation) sertifikat — eng yaxshi**
- Reputatsiya darhol mavjud, SmartScreen warning yo'q
- Narx: ~$300+/yil (Sectigo, DigiCert, GlobalSign)
- Yetkazib berish: USB token yoki HSM

**2. OV (Organization Validation) sertifikat**
- Reputatsiya vaqt o'tishi bilan o'sadi (yuzlab download'lardan keyin)
- Narx: ~$80+/yil
- Boshlanishida SmartScreen warning chiqaradi

**3. Self-signed (faqat sinov uchun)**
- Foydalanuvchi qo'lda root sertifikatni qabul qilishi kerak
- Public release uchun **tavsiya etilmaydi**

### Qadamlar / Steps

1. CA'dan `.pfx` fayl oling
2. base64 ga aylantiring:
   ```bash
   base64 -w 0 cert.pfx > cert.pfx.b64    # Linux
   certutil -encode cert.pfx cert.pfx.b64 # Windows
   ```

### Repo secret'lari

| Secret | Qiymat |
| --- | --- |
| `WINDOWS_CERT` | base64'lik `.pfx` content |
| `WINDOWS_CERT_PASSWORD` | `.pfx` paroli |

EV token uchun GitHub Actions hozirda mahalliy USB/HSM'ga kira olmaydi.
Buning uchun **self-hosted Windows runner** kerak yoki **Azure Code
Signing** xizmati ishlatish kerak ([docs](https://learn.microsoft.com/azure/trusted-signing/)).

---

## Imzo'siz holatda / When unsigned

Unsigned binarlar foydalanuvchilarga qo'shimcha qadam talab qiladi:

### macOS

```
"Portal" cannot be opened because it is from an unidentified developer.
```

Yechim:
1. Right-click → **Open** → ogohlantirishda **Open** ni bosish
2. Yoki: System Settings → Privacy & Security → "Open Anyway"

Bizning Releases hujjati bunda foydalanuvchilarga tushuntiradi.

### Windows

```
Microsoft Defender SmartScreen prevented an unrecognized app from starting.
```

Yechim:
1. **More info** → **Run anyway**

---

## Reja / Plan

- **v0.4.0:** Imzolanmagan reliz. README + Releases body'da foydalanuvchilarga
  qanday ochish tushuntiriladi.
- **v0.4.1 yoki v0.5.0:** Apple Developer Program va Windows OV sertifikat
  qo'shilgach, secret'larni qo'shish va imzolanan reliz.

Apple Developer'ni hozir ochish 1-3 soat oladi va deyarli darhol tasdiqlanadi.
Tasdiqlangach yana issue ochilsin va ushbu hujjatdagi qadamlarni
bajarish.

Opening Apple Developer takes 1-3 hours and gets approved almost
immediately. Once approved, file an issue and follow the steps in this
doc.
