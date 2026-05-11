# INSTALL — Portal Android client

Bu fayl APK ni Android qurilmaga yuklash bo'yicha. Manbadan qurish
ko'rsatmalari [BUILDING.md](BUILDING.md) da.

This file covers loading the APK onto an Android device. To build from
source see [BUILDING.md](BUILDING.md).

---

## 1. APK ni olish

Ikkita variant:

### A. Tayyor APK (release)

GitHub Releases sahifasidan: <https://github.com/Yaxyobek0877/portal_traffic/releases>

> Hozircha mobile relizlari GitHub'da rasmiy emas — desktop bilan
> birga taqsimlanmagan. Tayyor APK paydo bo'lguncha **B** variantdan
> foydalaning.

### B. Manbadan qurish (debug APK)

```bash
cd mobile
export JAVA_HOME="/Applications/Android Studio.app/Contents/jbr/Contents/Home"
./gradlew assembleDebug
```

APK joyi: `app/build/outputs/apk/debug/app-debug.apk`.

---

## 2. Yuklash usullari

### Variant 1: ADB (eng oson, hisobxona kerak emas)

ADB sizning kompyuteringiz bilan telefonni USB orqali bog'laydi —
Play Store kerak emas, Google hisobxona kerak emas.

**1‑qadam: Developer mode ni yoqish.**

Telefoningizda:

1. Settings → About phone
2. **Build number** ustiga 7 marta tegish (yoki tegmaslik kerak bo'lsa
   "Software version → Build number")
3. "You are now a developer" xabari chiqadi

**2‑qadam: USB debugging ni yoqish.**

Settings → Developer options → **USB debugging** = ON

Samsung'da yana: **USB debugging (Security settings)** ni ham yoqing.

**3‑qadam: Telefoni USB bilan ulang.** Telefonda dialog chiqadi:
**"Allow USB debugging?"** → **Allow**.

**4‑qadam: O'rnatish.**

```bash
# adb yo'lini ko'rsatish (Mac, Android Studio'dan):
ADB="$HOME/Library/Android/sdk/platform-tools/adb"

# qurilmalarni ko'rish:
$ADB devices
# kutilgan output:
# List of devices attached
# R5CY81RZ87B    device

# o'rnatish:
$ADB install -r app/build/outputs/apk/debug/app-debug.apk
```

`-r` (replace) — eski versiya o'rnatilgan bo'lsa, qayta yozadi.
Imzo bir xil bo'lishi kerak (debug imzosi keyhash o'zgarmaydi).

---

### Variant 2: Sideload (USB yo'q, faqat APK fayli)

APK ni telefonga ko'chiring (Telegram, Bluetooth, Google Drive — har
qanday yo'l bilan), keyin telefonda o'sha faylga teging.

**Birinchi marta:** Android "Install unknown apps" ruxsatini so'raydi.
Ruxsat berib qaytadan teging — APK Manager (yoki Files) orqali
o'rnatasiz.

> Xavfsizlik eslatmasi: faqat ishonchli manbalardan APK o'rnating.
> Portal APK'ning hash'ini taqqoslash uchun:
>
> ```bash
> shasum -a 256 app-debug.apk
> ```

---

### Variant 3: Wireless ADB (Android 11+)

USB kabel ishlatmaslik uchun:

```bash
# 1. telefonda Wi-Fi ulanmasin va Wi-Fi yoqilsin
# 2. Developer options → Wireless debugging → ON
# 3. "Pair device with pairing code" tegish, paydo bo'lgan IP+port+kodni
#    quyidagiga kiritish:
$ADB pair 192.168.1.123:43521
# pairing code: 123456

# 4. yana Wireless debugging ekranida "IP address & Port" ni ko'ring:
$ADB connect 192.168.1.123:38205

# 5. o'rnatish (USB kerak emas):
$ADB install -r app/build/outputs/apk/debug/app-debug.apk
```

---

## 3. Birinchi ishga tushirish

1. Launcher'da **Portal** ikonkasini bosing (qora fonda violet/cyan
   wormhole — bu sizning APK).
2. Welcome ekran:
   - O'ng yuqorida **UZ/EN** tugmasi bilan til tanlang
   - Taxallus kiriting (32 belgigacha)
   - **Yangi portal yaratish** yoki **Mavjud portalga qo'shilish**
3. Birinchi ulanishda Android quyidagi ruxsatlarni so'raydi:
   - **POST_NOTIFICATIONS** — foreground service uchun (Android 13+)
   - **Camera** — QR skanerlanganida
4. Foreground service ishga tushgach, telefon notifikatsiyasida "Portal
   ulangan" yozuvi ko'rinadi — buni o'chirmang, aks holda telefon
   ulanishni uzib qo'yishi mumkin.

---

## 4. Yangilanish

### Debug APK (B variant)

Yangi build:

```bash
./gradlew assembleDebug
$ADB install -r app/build/outputs/apk/debug/app-debug.apk
```

`-r` flagi sozlamalarni saqlaydi (DataStore, taxallus, oxirgi portallar).

### Release APK (A variant, mavjud bo'lganda)

Auto-update hozircha **yo'q**. GitHub Releases'dan qo'lda yangilang.

---

## 5. O'chirish

```bash
$ADB uninstall uz.aihealth.portal_mobile
```

Yoki telefondan: Settings → Apps → Portal → Uninstall.

> O'chirish DataStore'ni ham o'chiradi (taxallus, signal URL, TURN, oxirgi
> portallar). Saqlash kerak bo'lsa Android settings → Apps → Portal →
> Storage → "Backup" qiling (manuel).

---

## 6. Muammolar

### "INSTALL_FAILED_UPDATE_INCOMPATIBLE"

Eski APK boshqa imzo bilan o'rnatilgan. Avval o'chiring:

```bash
$ADB uninstall uz.aihealth.portal_mobile
$ADB install app/build/outputs/apk/debug/app-debug.apk
```

### "INSTALL_FAILED_VERIFICATION_FAILURE"

Play Protect APK'ni xavfsizlik tomondan tekshirgan va rad etgan. Sabab —
debug imzo. Vaqtinchalik yo'l: Play Protect ni o'chirish (Play Store →
Profile → Play Protect → Settings → Scan apps with Play Protect = OFF).
Tegishli reliz APK paydo bo'lguncha shu yaxshi.

### "App not installed" (sideload)

Telefonda allaqachon `uz.aihealth.portal_mobile` o'rnatilgan, lekin
imzo boshqa. Variant 1 dagi `adb uninstall` qiling, keyin qayta
yuklang.

### Telefon `adb devices` da ko'rinmaydi

- USB kabelni almashtirib ko'ring (charge-only kabel — debugging
  ishlamaydi)
- Telefonda paydo bo'lgan "Allow USB debugging?" dialogini qabul
  qilganmisiz?
- Mac'da: `$ADB kill-server && $ADB start-server` qayta urinib ko'ring
- Linux'da `udev` qoidalari kerak bo'lishi mumkin
  (`50-android.rules`)

### "wss:// connection failed"

Settings → Tarmoq da signal serveri URL noto'g'ri yoki yetib bo'lmadi.
Standart: `wss://signaling.1pro.uz/ws` ni qaytaring.

### Mobile internet'da peer'larga ulana olmayapti

CGNAT (Carrier-Grade NAT). Settings → **Cloudflare TURN** qismiga
Token ID + API Token kiriting yoki **Bepul TURN ni yoqing** (Open
Relay Project — sekin lekin sinashga yetadi).
