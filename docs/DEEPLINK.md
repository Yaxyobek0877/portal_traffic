# Portal deeplink scheme — `portal://`

Portal'ning `portal://` URL scheme'i: do'stingizning telefoni QR ni
skan qilsa yoki messaging ilovada link bossa, Portal ochilib join
flow'iga to'g'ridan-to'g'ri kiradi.

Portal's `portal://` URL scheme: when a friend scans a QR or taps a
link in a messenger, Portal opens directly to the join flow.

---

## Format

```
portal://join/<portal_id>
portal://join/<portal_id>?code=<code>
```

Misollar / Examples:

```
portal://join/428591
portal://join/428591?code=739204
portal://join/428591?code=X3K9-A2F1   (PCP-1)
```

Eski versiyalar uchun fallback (eskirib bormoqda) / Legacy fallback:

```
portal://<portal_id>:<code>
```

---

## Hozirgi qo'llab-quvvatlash / Current support

**v0.4.0:**
- ✅ QR kod `portal://` URL'ni encode qiladi (PortalHeader)
- ✅ Welcome ekrani'da paste handler — clipboard'dan portal:// URL
  yopishtirilsa, ID + kod avtomatik to'ldiriladi (`client/frontend/src/lib/deeplink.ts`)
- ⏳ OS-level scheme handler — keyingi reliz'da

**v0.4.1+ (rejada):**
- macOS: `Info.plist` ga `CFBundleURLTypes` qo'shish
- Windows: registry'da scheme ro'yxati
- Linux: `.desktop` faylda `MimeType` qo'shish
- Wails event handler — OS Portal'ni `portal://` URL bilan ochsa,
  Welcome ekrani avtomatik join modega o'tib ID + kodni to'ldiradi

---

## OS-level setup (v0.4.1 uchun reja)

### macOS — `Info.plist`

`build/darwin/Info.plist` ga:

```xml
<key>CFBundleURLTypes</key>
<array>
  <dict>
    <key>CFBundleURLName</key>
    <string>uz.1pro.portal.deeplink</string>
    <key>CFBundleURLSchemes</key>
    <array>
      <string>portal</string>
    </array>
  </dict>
</array>
```

Wails 2.x'da `wails.json` orqali avtomatik kiritish:

```json
{
  "info": {
    "infoPlistData": {
      "CFBundleURLTypes": [...]
    }
  }
}
```

### Windows — Registry

NSIS yoki MSI installer'ga:

```nsis
WriteRegStr HKCR "portal" "" "URL:Portal Protocol"
WriteRegStr HKCR "portal" "URL Protocol" ""
WriteRegStr HKCR "portal\shell\open\command" "" '"$INSTDIR\Portal.exe" "%1"'
```

### Linux — `.desktop`

`/usr/share/applications/portal.desktop`:

```ini
[Desktop Entry]
Name=Portal
Exec=portal %u
MimeType=x-scheme-handler/portal;
```

Keyin: `xdg-mime default portal.desktop x-scheme-handler/portal`

### Wails handler

`main.go` da yoki yangi `client/deeplink.go`:

```go
// Wails 2'da OS event hook orqali "OnSecondInstanceLaunch" yoki
// platform-specific event listener:
//
// macOS — application:openURL: NSAppleEventManager
// Windows — argv[1] da URL keladi
// Linux — argv[1] da URL keladi
//
// URL'ni parse qilib runtime.EventsEmit(ctx, "deeplink:join", parsed)
// yuboriladi va frontend Welcome'da tinglagan handler ishga tushadi.
```

---

## Xavfsizlik / Security

- Portal kodi URL'da yashirin emas (querystring'da). Bu o'qilsa,
  oddiy text kabi — yon kanal orqali (chat, voice) ulashish bilan bir
  xil tahdid darajasi.
- Browser'larda click bilan ochiladigan portal:// linklarda CSRF
  o'xshash xavf yo'q, chunki Portal joining uchun foydalanuvchi
  `Yaratish` yoki `Qo'shilish` tugmasini bosishi kerak. Avtomatik
  join qilmaymiz.
- The portal code is in the querystring, not hidden. Same threat
  model as sharing via chat/voice.
- No CSRF-like risk because the user must click "Create" or "Join" —
  we never auto-join from a deeplink.
