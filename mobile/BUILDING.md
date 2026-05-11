# BUILDING — Portal Android client

Bu fayl manbadan qurish, toolchain pinlari va keng tarqalgan xatolar
haqida. APK ni qurilmaga yuklash uchun [INSTALL.md](INSTALL.md).

This file covers building from source, toolchain pins, and common
errors. For loading the APK onto a device see [INSTALL.md](INSTALL.md).

---

## TL;DR

```bash
cd mobile
export JAVA_HOME="/Applications/Android Studio.app/Contents/jbr/Contents/Home"
./gradlew assembleDebug
```

APK joyi: `app/build/outputs/apk/debug/app-debug.apk`.

---

## Toolchain

| Komponent | Versiya | Sabab |
| --- | --- | --- |
| **JDK launcher** | 17 yoki 21 | `gradlew` boot uchun. SDK ni topish; daemon JVM ni provision qilish. |
| **Gradle daemon JVM** | 21 (foojay) | `gradle/gradle-daemon-jvm.properties` da pinlangan. AGP 9 21'ni talab qiladi. |
| **Android Gradle Plugin (AGP)** | 9.0.1 (canary) | `compileSdk { release(36) { minorApiLevel = 1 } }` DSL faqat 9+ tarkibida. |
| **Kotlin** | 2.0.21 | Compose plugin K2 talab. |
| **Compose BOM** | 2024.09.00 | Material3, Foundation, UI versiya guruhi. |
| **compileSdk** | 36 (minorApiLevel 1) | Android 16's "minor API levels". |
| **minSdk** | 26 (Android 8.0) | DataStore + foreground service + adaptive icon shu yerdan. |
| **targetSdk** | 36 | Play Store talabi. |
| **WebRTC** | `io.getstream:stream-webrtc-android:1.3.8` | Google'ning library'ning faol fork'i. |
| **lazysodium-android** | 5.1.0 (`@aar`) | NaCl secretbox. JNA ham `@aar` formatida. |

`gradle/libs.versions.toml` — version catalog. Yangi dependency qo'shilsa,
shu yerga yozish kerak (`build.gradle.kts` ichidagi inline string'lar
emas).

> **Istisno:** `lazysodium-android` va `jna` `app/build.gradle.kts` ichida
> to'g'ridan-to'g'ri stringlar bilan yoziladi, chunki ular
> `@aar` packaging classifier bilan kelishi kerak. Catalog bu ifoda
> turini qo'llab-quvvatlamaydi. Sabab: JNA jar variant Android'dagi AAR
> variant bilan to'qnashadi.

---

## JDK ni o'rnatish

Bu tizimda PATH'da JDK yo'q deb taxmin qilingan. Gradle bootstrap
launcher'iga JDK kerak — keyin u o'zi daemon JVM'ni `gradle-daemon-jvm.properties`
asosida provision qiladi.

**Eng oson:** Android Studio bilan kelgan JBR (JetBrains Runtime).

```bash
# Mac:
export JAVA_HOME="/Applications/Android Studio.app/Contents/jbr/Contents/Home"

# Linux (snap):
export JAVA_HOME="/snap/android-studio/current/android-studio/jbr"

# Linux (deb):
export JAVA_HOME="/opt/android-studio/jbr"
```

Bu shell session'da ishlaydi. `~/.zshrc` yoki `~/.bashrc` ga qo'shsangiz
har safar yozish kerak emas.

---

## SDK ni o'rnatish

Android Studio orqali eng oson:

1. Android Studio ni oching → **More Actions → SDK Manager**
2. **Android 14 (API 34)** va **Android 16 (API 36)** larni belgilang
3. **SDK Tools** tab → **Android SDK Build-Tools 36**, **Android SDK Platform-Tools** belgilangan
4. Apply → Yuklab oling

`local.properties` faylida SDK joylashuvi `sdk.dir=/Users/.../Library/Android/sdk`
sifatida turishi kerak. Bu fayl gitignored — qo'lda yozish yoki
Android Studio yarataadi.

```properties
sdk.dir=/Users/coder/Library/Android/sdk
```

---

## Buyruqlar

```bash
# barcha unit testlar
./gradlew test

# bitta test
./gradlew :app:testDebugUnitTest --tests "uz.aihealth.portal_mobile.ExampleUnitTest.addition_isCorrect"

# instrumented testlar (qurilmada — crypt moduli uchun shart)
./gradlew connectedAndroidTest

# lint
./gradlew lint

# debug APK
./gradlew assembleDebug

# release APK (signing kerak — keyingi bo'limga qarang)
./gradlew assembleRelease

# install
./gradlew installDebug

# clean
./gradlew clean
```

Birinchi build ~6-10 daqiqa oladi (dependencies yuklab olinadi). Keyingi
build'lar 30 sek-2 daq.

---

## Release imzolash (keyingi reliz uchun)

Release APK ni Play Store yoki rasmiy distributsiya uchun imzolash kerak:

```bash
# 1. keystore yaratish (bir marta)
keytool -genkey -v -keystore portal-release.jks -keyalg RSA \
        -keysize 2048 -validity 10000 -alias portal-release

# 2. local.properties (yoki ~/.gradle/gradle.properties) ga qo'shish:
PORTAL_RELEASE_STORE_FILE=/path/to/portal-release.jks
PORTAL_RELEASE_STORE_PASSWORD=...
PORTAL_RELEASE_KEY_ALIAS=portal-release
PORTAL_RELEASE_KEY_PASSWORD=...

# 3. app/build.gradle.kts ga signingConfigs blokini qo'shish (keyingi
#    reliz uchun)
```

Hozir bu joyda qilingani yo'q — debug APK ni topshiramiz.

---

## Ikonkalar

Launcher ikonkalari `scripts/gen_icons.py` orqali yaratiladi. Logoni
o'zgartirsangiz:

```bash
python3 scripts/gen_icons.py
```

Bu:

- `app/src/main/res/mipmap-{mdpi,hdpi,xhdpi,xxhdpi,xxxhdpi}/` ichida
  `ic_launcher.png`, `ic_launcher_round.png`, `ic_launcher_foreground.png`,
  `ic_launcher_monochrome.png` ni qayta yaratadi
- `build/playstore_icon.png` (512×512) ni Play Store entrysi uchun yaratadi

Agar PIL/Pillow o'rnatilmagan bo'lsa: `pip3 install Pillow`.

---

## Keng tarqalgan xatolar

### `JAVA_HOME is not set and no 'java' command could be found`

JDK yo'q. Yuqoridagi "JDK ni o'rnatish" bo'limiga qarang.

### `error: package R does not exist` yoki `Compose plugin not found`

Gradle cache buzilgan:

```bash
./gradlew clean
rm -rf .gradle ~/.gradle/caches/build-cache-1
./gradlew assembleDebug
```

### `INSTALL_FAILED_OLDER_SDK`

Telefonning Android versiyasi `minSdk = 26` (Android 8.0) dan eski.
Yangi telefon kerak (Android 8 oldi mobile foydalanuvchi sonining
~3% ni tashkil qiladi).

### `Gradle daemon disappeared unexpectedly`

Daemon JVM versiyasi PATH'da turgan JDK bilan to'qnashdi. Tozalash:

```bash
./gradlew --stop
rm -rf ~/.gradle/daemon
./gradlew assembleDebug
```

### `Could not resolve com.goterl:lazysodium-android:5.1.0@aar`

Maven Central + JCenter yetib bo'lmadi yoki `@aar` classifier'ini
o'qiydigan plugin yangilanmagan. `settings.gradle.kts` da
`mavenCentral()` repositorysi borligini tekshiring.

### `WebRTCFactory.initialize` lib `.so` yo'qligida ishlamaydi (instrumented test)

`lazysodium-android` ABI'ga bog'liq `.so` fayl. Host JVM testida ishlamaydi.
`crypt/` paketini test qilmoqchi bo'lsangiz `connectedAndroidTest`
ishlatish kerak — fizik qurilma yoki emulyator.

### `Compilation error: Unresolved reference: BuildConfig`

`buildFeatures { buildConfig = true }` ni `app/build.gradle.kts` ga
qo'shish kerak (bu reposda allaqachon qo'shilgan).

### Build muvaffaqiyatli, lekin telefonda eski versiya ko'rinmoqda

```bash
adb uninstall uz.aihealth.portal_mobile
./gradlew installDebug
```

Eski APK kesh'da qolgan bo'lishi mumkin. Force-stop ham yordam beradi:

```bash
adb shell am force-stop uz.aihealth.portal_mobile
```

---

## CI/CD

Hozir mobile uchun CI ishlamaydi. Reja:

- GitHub Actions: `ubuntu-latest` + Android SDK + JDK 21
- `./gradlew test lint assembleDebug` — har push'da
- Release tag'da: `assembleRelease` + signed APK ni Releases'ga upload

Buni `release-mobile.yml` workflow sifatida `.github/workflows/` ga
qo'shamiz keyingi reliz bilan.
