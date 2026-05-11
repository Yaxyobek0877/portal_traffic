# Changelog

Bu hujjat Portal'ning har bir public chiqarilishidagi o'zgarishlarni qayd
etadi. Format [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) ga
asoslangan; loyiha [Semantic Versioning](https://semver.org/spec/v2.0.0.html)
ga rioya qiladi.

## [Unreleased]

v0.5.5 dan keyingi ishlanma bu yerga yoziladi.

## [0.5.5] — 2026-05-11

Sign-in tajribasini brauzergacha kengaytirish va mobile platformani live
qilish. Foydalanuvchi endi `portal.1pro.uz` web'da ham, Android'da ham,
desktop'da ham bir xil hisob bilan kirib, o'zining portallarini har uch
joydan ko'ra oladi. Bundan tashqari mobile birinchi marta to'liq simlangan
— signaling.1pro.uz `/api/auth/*` ga ulanish, Cloudflare Calls TURN,
UZ+EN i18n va brendlangan launcher icon bilan.

### Added — Yangi imkoniyatlar

- **Web tomondan jonli sign-in/up** — bosh sahifaning chap ustunidagi
  forma endi haqiqiy `/api/auth/{signup,signin}` ga POST qiladi:
  - `web/login.js` validator client-side (3-32 belgi username, 8+ belgi
    parol) bilan, server xatolari `USERNAME_TAKEN`, `INVALID_CREDENTIALS`,
    `LOCKED_OUT` field error sifatida chiqadi.
  - Muvaffaqiyatli kirishdan keyin `/admin/dashboard` ga redirect; allaqachon
    signed-in tashrif buyuruvchi (`/api/me` 200) formani umuman ko'rmaydi.
  - Sign-up / Sign-in tab'lari bitta formani rejimga qarab moslaydi;
    Sign-up'da parol takror maydoni + visibility toggle (`Eye`/`EyeOff`),
    Sign-in'da yo'q.
  - `/download/<asset>` brendlangan URL'lar — signaling server
    `--downloads-base` orqali GitHub Releases'ga 302 qiladi, foydalanuvchi
    `portal.1pro.uz/download/Portal-darwin-arm64.dmg` ko'rinishdagi
    havolani ulashishi mumkin.
- **Persistent userstore** — `server/userstore.go` parollarni Argon2id
  bilan saqlaydi (`~/.portal-server/users.json` atomic write); per-IP
  lockout (`golang.org/x/time/rate`), typed error code'lar. Userstore
  state restartlardan keyin saqlanadi.
- **Cross-device portallar (account-linked)** — desktop endi cloud
  hisobiga sign-in qilingan paytda `signaling.1pro.uz/api/portals` ga
  o'z aktiv portallarini push qiladi; web admin shell (`/admin/dashboard`)
  brauzerdan turib ularning ro'yxatini ko'rsata oladi. Avvalgi "account
  linkage isn't wired yet" eslatmasi olib tashlandi.
- **Parol reveal toggle** — web va desktop sign-in formalarida `Eye` /
  `EyeOff` ikonali tugma; default `type="password"`.
- **`Connected` chrome tushirildi** — desktop sarlavhasidagi har vaqtli
  "Connected" badge realda kam ma'lumotli edi va estetik shovqin keltirar
  edi. Status barda peer count + transport indicator bor — ular yetarli.

### Mobile — Android ilovasi birinchi marta to'liq simlangan

- **Sign-in / sign-up** — `mobile/.../auth/AuthApi.kt` deployed
  `/api/auth/{signup,signin,signout,me}` ga gaplashadi. Sessiya cookie
  `__Host-portal_session` DataStore'da turadi va har so'rovga qayta
  qo'shiladi.
- **Forma UI (`AuthScreen.kt`)** — web sign-in/up formasining mirror'i:
  field-level error mapping (`USERNAME_TAKEN`, `PASSWORD_WEAK`,
  `LOCKED_OUT` va h.k.), Sign-up'da parol takror, visibility toggle.
- **Cloudflare Calls TURN** — `mobile/.../turn/` paketi
  `client/cloudflareturn.go` ning Kotlin nusxasi:
  `rtc.live.cloudflare.com` ga bitta POST → qisqa muddatli ICE
  credential (30 daqiqaga cache); Settings'da Cloudflare `TOKEN_ID` +
  `API_TOKEN` kiritiladi. `MeshManager` resolved server'larni
  `PeerConnectionFactory` ga uzatadi.
- **i18n (UZ + EN)** — `i18n/Strings.kt` ikki tilli string jadvali;
  har ekran `t(key)` orqali o'qiydi; Settings'da locale toggle.
  Default — Uzbek (desktop bilan bir xil).
- **Brendlangan launcher icon** — `scripts/gen_icons.py` ishlab chiqaradigan
  foreground + monochrome PNG mipmap'lar; flat zinc-950 background;
  `app_name = "Portal"`. Eski material webp + vector drawable'lar olib
  tashlandi.
- **Animatsiyali Logo** — `ui/Logo.kt`, frontend Logo'sining Compose
  porti (besh konsentrik aylanuvchi halqa). Welcome va Auth ekranlarida.
- **About card** — Settings'da `versionName` ko'rsatadi
  (`buildConfig = true` orqali generatsiya qilingan).
- **Hujjatlar** — `mobile/{ARCHITECTURE,BUILDING,INSTALL}.md` +
  yangilangan README.

### Changed — O'zgartirilgan

- **Update banner i18n** — `update.install`, `update.installing`,
  `update.install_failed` key'lari (v0.5.4'dan).
- **Release pipeline** — `.github/workflows/release.yml` endi
  `draft=false` bilan publish qiladi (avval qo'lda promote kerak edi);
  web/ assetlar uchun cache-busting query string qo'shiladi.

### Infrastructure

- **Cloudflare Pages auto-deploy** — `web/` papkasining har push'i
  Cloudflare Pages'da avtomatik build bo'ladi; `/admin/*` URL'lar
  trailing slash siz ham ishlaydi.
- **Server'ning private repoga ko'chirishi** — backend
  (`server/auth_handlers.go`, `userstore.go`, va h.k.) endi alohida
  `Yaxyobek0877/portal_server` private GitHub repo'sida saqlanadi.
  Public `portal_traffic` repo'sida `server/` allaqachon `.gitignore`'da
  edi — endi qachondir kerak bo'lganda fetch qilish mumkin.

## [0.5.4] — 2026-05-08

In-app auto-update va web admin paneli uchun server-side roadmap.
v0.5.x serial yangilanishlarining yopilishi: foydalanuvchi endi
relizlarni qo'lda yuklab olib, ikki marta bosib chiqarib qayta
o'rnatishi shart emas — banner'dagi "Yangilash" tugmasi bir bosishda
yangilab beradi.

### Added — Yangi imkoniyatlar

- **In-app auto-update (Yangilash tugmasi)** — Update banner'ga
  ikkita yangi tugma qo'shildi:
  - **Yangilash** — `App.InstallUpdate()` chaqiradi: GitHub Releases'dan
    OS-specific asset'ni yuklab oladi (`client/updater/install.go`),
    OS temp dir'ga staging qiladi (`portal-update-<unixns>`), swap
    script (mac/linux: `/bin/bash`, windows: `.bat`) yozadi va detached
    spawn qiladi (Unix `Setsid: true`, Windows `cmd /c start /min`).
    500 ms keyin `runtime.Quit(ctx)` chaqirib o'zini o'chiradi; script
    eski binary'ni `rm -rf` qilib, yangisini ko'chirib qayta ishga
    tushiradi (macOS'da `open`, Linux'da bevosita exec, Windows'da
    `start ""`).
  - **↗** — release sahifasini brauzerda ochadi (avval bo'lgan oqim,
    qoldirildi: foydalanuvchi notlarni o'qishi yoki qo'lda yuklab
    olishi mumkin).
  - Tugma `installing` paytida `Yangilanmoqda…` deb yoziladi va
    `disabled` bo'ladi (Dismiss tugmasi ham bloklanadi). Xato bo'lsa
    banner ostida `update.install_failed` matni ko'rinadi.
  - Zip va tar.gz arxivlari qo'llab-quvvatlanadi (`unzip` + `untargz`),
    zip-slip / tar-slip himoyasi bor.
- **Web admin paneli — server-side roadmap** —
  `docs/ROOM-CONTROLS-GAPS.md` da yangi bo'lim
  **"Web admin paneli (online qurilma boshqaruvi)"**: foydalanuvchi
  brauzerdan turib o'z portal/peer/servislarini ko'rish va boshqarish
  uchun nima kerak — `/api/portals`, `/api/portals/:id/services`,
  `service.announce` / `service.pause_request` xabarlari, opt-in
  toggle'lari. Hozirgi `web/admin/login.html` + `dashboard.html`
  static mockup, real backend deploy qilinishi kerak. Hech qaysi
  client tomon o'zgarish ishlamaguncha avval server tomon qayta
  yozilishi kerak — bu hujjatlash, kelajakda bosqichma-bosqich
  amalga oshirish uchun.

### Changed — O'zgartirilgan

- **Update banner kengligi** — `max-w-[280px]` → `w-[320px]` (yangi
  Yangilash tugmasini joylab olish uchun).
- **Update banner tugmasi yorlig'i** — eski "Yuklab olish" tugmasi
  endi kompakt "↗" (release sahifa link), `Yangilash` esa primary
  CTA. i18n key'lari: `update.install`, `update.installing`,
  `update.install_failed`.

### Documentation

- `docs/ROOM-CONTROLS-GAPS.md` — yangi bo'lim **"Auto-update
  (v0.5.4 da yopildi)"**: backend / detach / frontend dizayni
  qisqacha yozilgan; web admin uchun **uchta server endpoint'i**
  va **ikkita yangi WebSocket xabari**ning shartlari.

## [0.5.3] — 2026-05-08

Patch reliz: device-name separator'ni server tomonidan ruxsat
etilgan belgiga to'g'rilash va resume paytida nicknamasi
ikki marta stamp bo'lishidan saqlash.

### Fixed — Tuzatilgan xatolar

- **NICKNAME_INVALID hali ham qaytayotgan edi** — v0.5.2'dagi `'@'`
  separator ham server validator regex'iga (`server/nickname.go
  validNickname`: `[a-zA-Z0-9_\-.]`) tushmadi. Test qildim:
  ```
  send portal.create  nick=texuz@mac → recv NICKNAME_INVALID
  ```
  Endi separator `'.'` (period) — server qabul qiladigan uchta
  belgidan biri (`_`, `-`, `.`); o'qilishi eng tabiiy: `texuz.uy`,
  `texuz.win64`. Device label sanitize qilinadi: faqat
  `[a-zA-Z0-9_\-]` qoladi.
- **Raw nickname'ni storage'ga saqlash** — avval `persistEnter`
  combined nicknamesini (`texuz.mac`) saqlardi. Resume paytida
  `BackgroundCreatePortal('texuz.mac')` chaqirilardi va
  `nicknameWithDevice` yana stamp qilardi → `texuz.mac.mac`. Fix:
  `createPortal` / `joinPortal` raw nickname'ni (`texuz`) saqlaydi,
  mesh layer'ga combined formni yuboradi. Device label har create'da
  joriy qiymatdan olinadi — agar foydalanuvchi `mac` → `uy` ga
  o'zgartirsa keyingi sessiyalar `texuz.uy` bilan announce qilinadi.

### Changed — O'zgartirilgan

- **Frontend `splitNicknameAndDevice`** — endi `'.'` da (avval `'@'`)
  bo'linadi va oxirgi `'.'` ni topadi, shuning uchun
  `'john.doe.mac'` to'g'ri parsed: nickname=`'john.doe'`,
  device=`'mac'`.

## [0.5.2] — 2026-05-08

Patch reliz: device-name bilan auto-resume bog'liq blokerni tuzatish
va landing page'ni yangilash.

### Fixed — Tuzatilgan xatolar

- **Auto-reconnect server tomonidan bloklanardi** — v0.5.1'da
  qo'shilgan device-name `'·'` (middle-dot) ni mesh nicknamesiga
  qo'shardi (`'texuz · mac'`). Deployed signaling server'ning
  nickname validator'i Unicode glyph'larni va whitespace'ni rad
  qiladi → `NICKNAME_INVALID` xatosi → `ResumeActiveSessions` har
  saqlangan owner sessiya uchun fail bo'lardi va yangi portal_id
  ham olinmasdan rad etilardi. Endi separator `'@'` (server qabul
  qiladigan ASCII char) va whitespace strip qilinadi: `'texuz@win64'`
  → server qabul qiladi → auto-resume ishlaydi.
- Frontend `splitNicknameAndDevice` helper'i wire-format'ni
  vizual ko'rinishga aylantiradi: `'texuz@uy'` → "texuz · uy"
  (PeerCard'da username + device'ni alohida pretty-print qiladi).

### Documentation

- `docs/ROOM-CONTROLS-GAPS.md` — yangi bo'lim **Owner-offline portal
  survival**: foydalanuvchi shikoyatining server-side root cause'i
  (server owner disconnect bo'lishi bilan portal'ni o'chiradi → yangi
  portal_id → friends'ning eski code'i ishlamaydi). Server-side
  yechim sxemasi yozildi: 5-10 daqiqa grace period, ownership
  transfer fallback.

### Changed — Landing page

- `web/index.html`: eski "6 xonali kod" ga oid matnlar yangilandi.
- Yangi bo'lim **"v0.5.x da yangi"** — 8 ta feature card (mahalliy
  hisob, multi-portal, auto-reconnect, approval, device name,
  LAN qurilmalar, system startup, bandwidth probe).
- Roadmap to'liq qayta yozildi: 8-bosqich (v0.5.0 + v0.5.1) tugadi
  belgilandi, 9-bosqich (code signing) jarayonda, 10-bosqich
  (cloud auth + xona boshqaruvi) keyingi.
- "Qadamlar" bo'limiga "Hisob yarating" qadami qo'shildi (v0.5.0
  vault'i sababli).

## [0.5.1] — 2026-05-08

Polish reliz: app icon barcha platformalarda, qurilma nomi (multi-
device disambiguation), tizim startup'ida avtomatik ishga tushish, va
landing/release jadvalining stabil URL'lari.

### Added — Yangi imkoniyatlar

- **Windows .ico va Linux ikonkalari** — `build/windows/icon.ico` (6
  o'lcham: 16/32/48/64/128/256), `build/linux/icon.png`. .exe va Linux
  desktop'da Portal logosi to'g'ri ko'rinadi (avval default Wails
  ikonkasi edi).
- **Qurilma nomi (Device name)** — `Settings → Profil` da yangi maydon.
  Default platform-derived (`mac` / `win 64` / `linux`); foydalanuvchi
  `uy` / `ish` / `serverim` ga o'zgartirishi mumkin. Mesh nicknamesi
  `texuz · uy` ko'rinishida announce qilinadi → bir akkauntdan turli
  qurilmalar room'da farqlanadi.
- **Auto-run on system startup** — `Settings → Profil` da toggle.
  - macOS: `~/Library/LaunchAgents/uz.1pro.portal.plist` LaunchAgent
  - Windows: `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`
    registry value
  - Linux: `~/.config/autostart/portal.desktop` XDG autostart
  Toggle yoqilsa OS-darajasidagi entry yoziladi; o'chirilsa olib
  tashlanadi. Saqlangan portallar `ResumeActiveSessions` orqali
  reboot keyin avtomatik qayta ulanadi.

### Changed — O'zgartirilgan

- **Stabil yuklab olish URL'lari** — Release CI endi assetlarni tag'siz
  nom bilan paketlaydi: `Portal-darwin-arm64.zip`,
  `Portal-windows-amd64.zip`, va h.k. README, web/index.html va
  `release_body.md` jadvallari `releases/latest/download/Portal-<triplet>.<ext>`
  formatida — har versiya chiqishida URL o'zgarmaydi.
- **README va landing page tablitsasi** — placeholder `vX.Y.Z`
  o'rniga to'g'ridan-to'g'ri klikli download linklari, har bir
  platforma uchun.

### Documentation — Hujjatlar

- `docs/ROOM-CONTROLS-GAPS.md` — multi-device login bo'limi qo'shildi
  (4-feature). Cloud-auth (HTTP `/api/auth`) signaling server'da
  deploy qilingach client tomon avtomatik ishlaydi degan reja
  yozilgan.

## [0.5.0] — 2026-05-08

Katta UX va arxitektura yangilanishi: hisob-vault tizimi, bir nechta
portalga bir vaqtda ulanish, avtomatik qayta ulanish, foydalanuvchi
profili, va xavfsizlik gate'lari. Asosiy yo'nalishlar — meeting
rejimiga yaqin tajriba va session lifecycle ustidan to'liq nazorat.

### Added — Yangi imkoniyatlar

- **Mahalliy hisob (vault) + Lock screen** — har ishga tushishda
  username + parol gate. bcrypt cost 10, 8-char strong-password
  policy (lower + upper + digit + special), 5 ta urinishdan keyin
  30s rate-limit lockout. "Eslab qol" bayrog'i remembered=true
  bo'lsa keyingi cold start lock'ni o'tkazib yuboradi.
- **Account-app dashboard** — Welcome ekrani signed-in tajribaga
  aylantirildi: avatar/nickname pill, Lock screen tabbed Sign-Up /
  Sign-In, sign-out flow vault'ni qaytadan qulflaydi.
- **Multi-portal sessions** — bir App bir vaqtda bir nechta portalga
  ulanishi mumkin. `portalSession` per-portal mesh + forwarder +
  transfer engine'ni egallaydi; foreground / background sessiyalar
  Welcome dashboard'ning "Faol ulanishlar" stripida ko'rinadi;
  PortalHeader'da Layers chip + dropdown bilan portal switcher.
- **Background-connect** — Welcome'dagi PlugZap tugmalari xonaga
  kirmasdan portal yaratish/ulash imkonini beradi.
- **Avtomatik qayta ulanish** — `active_sessions` jadvalga yozilgan
  sessiyalar app restart paytida unlock keyin avtomatik tiklanadi.
  Owner sessiyalar yangi portal_id bilan qayta yaratiladi (eski
  o'lgan); joiner sessiyalar saqlangan id+code bilan urinadi.
  Exposed servislar har yangi sessionga `restoreExposedServicesFor`
  orqali qayta announce bo'ladi.
- **Per-port "tasdiqlab yoqish" (approval gate)** — har servis
  qatorida shield ikoni bilan toggle. Yoqilgan bo'lsa har peer dial
  paytida egasiga modal popup chiqadi (kim, qaysi servis, TCP/UDP),
  Allow/Deny tugmalari bilan. Decision per (peer, port, protocol)
  sticky cache. 5s TCP / 3s UDP timeout — javobsiz qoldirilsa
  avtomatik rad etiladi.
- **Persistent peers** — peer chiqib ketsa A'zolar ro'yxatida offline
  ko'rinishida qoladi (kulrang dot, opacity-60, "offline" yorlig'i).
  Hover'da X tugmasi forget uchun. Online qaytsa avtomatik tirilib
  qaytadi.
- **Yangi Portal layout** — markaz = "Servislar" grid (siz +
  peer'lar birga, prominent Ulash tugmalari), o'ng tomon yarmi
  "Mening servislarim" formasi + LAN scan, yarmi Chat. Chap tomon
  kengaytirilgan PeerCard — har peer uchun state, RTT, P2P/TURN
  badge, traffic hisoblari, ICE pair manzillari, "Tezligi" probe.
- **Settings'ni real ilova qilish** — sidebar nav (Profile / Network
  / Diagnostics / Files / Activity / History / Logs / About).
  Profile tab egasi avatar, parolni tiklash, sign-out; Network tab
  faqat signaling URL + read-only TURN status (Cloudflare/qo'lda
  TURN UI olib tashlandi — server-side avtomatik beriladi).
- **Portal nomlash + Recent dedup** — har owner sessiyaga `label`
  tayinlash mumkin (pencil ikoni). Owner sessiyalari nickname
  bo'yicha dedup qilinadi. Faol ulanishdagi portallar Recent
  ro'yxatidan yashiriladi (bir portal ikki joyda ko'rinmaydi).
- **NVR / kamera workflow** — ServicesPanel formasi Nom / LAN IP /
  Port / Protocol qatorlariga bo'lindi. Hikvision NVR (SDK :8000)
  preset, smart-paste IP:port, mesh-port remap (kamera 554 → mesh
  8554) qo'llab-quvvatlanadi. Inline target editor (pencil) — health
  uchun aniq maslahat: localhost target bo'lsa "✏️ bilan LAN IP'ga
  o'zgartiring".
- **Portal app ikonkasi** — Logo komponenti motivi (binafsha→cyan
  halqalar + yorug' markaz) static rasterize qilindi. Wails build
  uni iconfile.icns / appicon.ico ga aylantiradi.

### Changed — O'zgartirilgan

- **TURN sozlamalari UI olib tashlandi** — Cloudflare TURN credentials
  va manual TURN URL formalari Settings'dan chiqarildi. signaling
  server `PortalCreated/Joined` event'larida qisqa muddatli ICE
  servers yuboradi → mesh.applyServerICE avtomatik qo'llaydi.
  Foydalanuvchi hech narsa sozlashi shart emas. Backward-compat:
  Go-side `Get/Set/TestCloudflareTurn` metodlari mavjud.
- **History dedup mantiqi** — Owner qatorlari nickname bo'yicha
  dedup. Pre-existing duplikatlar keyingi `AddHistory` chaqiruvida
  bitta qatorga consolidate bo'ladi (label saqlanadi).
- **Health probe protocol-aware** — TCP service'lar TCP-dial bilan
  probe qilinadi, UDP service'lar `unknown` deb belgilanadi (UDP'ni
  protokolsiz tekshirib bo'lmaydi). Friendly xato xabarlari:
  "qurilma yoqilgan, lekin shu portda servis yo'q", "qurilma o'chiq
  yoki tarmoqda yo'q", va h.k.
- **PortalHeader navigation** — "Asosiy" tugmasi (label + arrow-left)
  prominent, portal'ni uzilmasdan dashboardga qaytaradi. Kichik
  qizil LogOut ikoni alohida — bu portalni yopish (uzilish). Ilgari
  ikkalasi bir tugmada edi, foydalanuvchilar tasodifan disconnect
  qilib qo'yardi.
- **Landing page** — yuklab olish kartochkalari hero CTA tugmalari
  ustiga ko'tarildi, har platforma uchun .zip / .tar.gz / .exe
  yorlig'i bilan ko'rinadi. Versiya pill'i `v0.5.0 · jonli reliz`.

### Fixed — Tuzatilgan xatolar

- **Portal ekrani bo'sh chiqishi** — multi-portal handover'da
  PortalView'da `SessionID` yo'q bo'lgani sababli `setActiveSession`
  top-level `portal`'ni `null` qilib qo'yardi. SessionID Go-side
  struct'ga qo'shildi va store reducer'i mavjud projection'ni
  saqlaydi.
- **useShallow on session selectors** — zustand v5 + React 18'da
  `Object.values(...).map(...)` selector har render'da yangi array
  yaratardi → "Maximum update depth exceeded" infinite loop.
  `useShallow` bilan elementwise compare.
- **History list stale snapshot** — Create/Join keyin RecentPortals
  qayta yuklanadi (`refreshLists` helper).

### Hujjatlar — Documentation

- `docs/ROOM-CONTROLS-GAPS.md` — server-side talab qiladigan uch
  feature'ni hujjatlashtirdi: 8-char password format, code
  regenerate, "ask admin" join mode. Har biri uchun kerak bo'lgan
  protocol qo'shimchalari yozilgan.

## [0.4.1] — 2026-05-04

v0.4.0'dan keyingi katta UX va xavfsizlik ishlari. Asosiy yo'nalishlar:
peer ulanish ishonchliligi, UDP / o'yin server qo'llab-quvvatlashi, LAN
qurilmalarni mesh orqali ulashish (kamera / NVR / printer), va remote
debug uchun avtomatik log yuklash.

### Added — Yangi imkoniyatlar
- **PeerTable + ⚡P2P / ☁️TURN badge** — qimirlovchi mesh diagrammasi
  o'rniga 6 ustunli jadval. Har peer uchun ICE selected pair tipi
  (`host` / `srflx` / `relay`) va manzillari ko'rinadi; LAN bo'lsa
  "bir tarmoq ichida", srflx pair bo'lsa "Internet — turli tarmoq",
  TURN bo'lsa "TURN relay" deb belgilanadi.
- **3 sekundlik bandwidth probe** — har peer kartochkasidagi "Tezligi"
  tugmasi 32 KB binary chunk'larini control channel'da yuborib qabul
  qiluvchi tomondan o'lchangan Mbps ni qaytaradi.
- **UDP servislar** — listeners.go endi `lsof -iUDP` / `ss -lnup` ham
  yuradi, GUI'dagi expose va dial path UDP'ni qabul qiladi. CS2,
  Valorant, Minecraft Bedrock kabi UDP o'yin serverlari mesh orqali
  ulashish mumkin. Single port'ga TCP+UDP "Ikkalasi" rejimi ham bor
  (Steam dedicated server'lar uchun).
- **LAN qurilmalarni forward qilish** — Expose'da yangi "+ LAN qurilma"
  maydoni: `192.168.1.100:554` kabi target kiritsangiz, mehmon
  ulanganda traffic shu qurilmaga (kamera / NVR / printer) yo'naltiriladi.
- **TCP-probe LAN scanner** — bir bosishda local /24'ni 11 ta well-known
  portga skanerlaydi (~10s), topilgan kameralar/printerlarni one-click
  expose. "Hammasini och" — barcha topilganlarni avtomatik ekspoz
  qiladi (xavfli portlarni avtomatik o'tkazib yuboradi).
- **Servis health check** — har 30 sekundda exposed target'ni TCP
  probe qiladi; UI'da yashil/sariq/qizil dot ko'rinadi (kamera
  uzilsa darhol bilib turasiz).
- **Risk warning** — DB / router admin / RDP kabi xavfli portlarni
  expose qilishda confirm dialog (false positives bilan, "danger"
  va "warn" darajalari).
- **Activity log** — Settings → Faollik: kim qachon qaysi servisga
  ulandi, success/error bilan. 200 ta yozuvli ring buffer.
- **Service presets** — ✨ Tezkor tugmasi: Minecraft (Java/Bedrock),
  CS2, Rust, Factorio, Terraria, RTSP kamera, Web UI, SSH, Vite —
  bir bosishda nom + port + protokol to'ldiriladi.
- **Servis persistence** — exposed servislar SQLite'da saqlanadi va
  har portal yaratish/qo'shilishda avtomatik qayta ekspoz qilinadi.
  ⏸ pause toggle: vaqtincha to'xtatish (LAN target IP'ni yo'qotmasdan).
- **Server-issued TURN credentials** — signaling server clientga
  qisqa muddatli TURN creds yuborib bersa, foydalanuvchi qo'lda
  Cloudflare account ochmaydi (deploy qilingach ishlaydi).
- **Avtomatik log upload** — har 30 sekundda mahalliy log fayli
  yangi qatorlari `https://<signaling>/logs/upload`'ga yuklanadi.
  Default-on, opt-out toggle Settings'da. Anonim 32-hex client ID.
  Server-side handler `docs/SERVER_LOG_UPLOAD.md` da.
- **Live LAN scan progress** — 12s skanerlash davomida bar va
  "192.168.1.123… · 754/2794 · 1 topildi" qatori ko'rsatiladi.

### Changed — O'zgartirilgan
- ServicesPanel butun qayta dizayn: TCP/UDP/Ikkalasi toggle, LAN target
  collapsible advanced bo'limi, dialed pill "lokal" tag bilan.
- Local proxy listener endi remote port bilan teng port'ga bog'lanishni
  afzal ko'radi (`127.0.0.1:5000` ↔ `peer:5000` mos keladi); band
  bo'lsa OS-pick'ga tushadi va UI sariq "lokal*" badge ko'rsatadi.
- Signaling event'larning hammasi INFO darajada loglanadi
  (`recv portal.joined peer_count=1`, `send portal.create nick=…`).
- macOS system processes (ControlCenter, mDNSResponder, sharingd va
  hk.) "Lokalda topilgan portlar"da ko'rsatilmaydi — tasodifan tizim
  servisini ulashmaslik uchun.

### Fixed — Tuzatildi
- **Critical: deadlock in onPeerJoinedWithRoster** — `m.mu.Lock()`
  ushlab turib `iceServersForPeer()` chaqirilardi (ichida RLock).
  Go'ning sync.RWMutex recursive lock'ni qo'llab-quvvatlamaydi →
  goroutine mangu bloklangan, peer kartochkasi UI'da paydo bo'lmagan.
  `iceServersForPeerLocked` helper'i bilan tuzatildi.
- **Critical: portal:ready event payload shape** — `*ev.Portal` (raw
  `mesh.PortalInfo`) yuborilardi; Go field nomlari bosh harf bilan
  marshal qilingan, frontend `portalId`/`code` undefined ko'rgan,
  Windows'da ID/KOD blank bo'lib qolardi. `portalToView()` orqali
  o'tkazildi.
- "Tarmoqdagi qurilmalar" ro'yxatida bir port bir necha marta
  ko'rinardi; (port, protocol) bo'yicha dedup.

### Documentation
- `docs/SERVER_TURN.md` — signaling serverga Cloudflare TURN integration
  Go snippet.
- `docs/SERVER_LOG_UPLOAD.md` — log upload endpoint Go handler.
- README'ga "Game servers and dev URLs" + "Where the traffic flows"
  bo'limlari qo'shildi.

## [0.4.0] — 2026-05-03

Birinchi rasmiy ommaviy chiqarilish.

### Added — Yangi imkoniyatlar
- **MIT litsenziya** — kod ochiq, fork va commercial foydalanish ruxsat etilgan.
- **PCP-1 (Portal Cipher Protocol v1)** — yangi shifrlash qatlami: Ed25519
  identity + X25519 ephemeral session + XChaCha20-Poly1305 AEAD. Forward
  secrecy va identity-bound portal kodlari.
- **Internationalization (i18n)** — English + O'zbek UI; til Settings'dan
  tanlanadi.
- **CI/CD pipeline** — GitHub Actions: har push'da test, har tag'da
  cross-platform binarlar Releases'ga yuklanadi.
- **CHANGELOG va versiyalash** — endi har relizning aniq qaydi bor.

### Changed — O'zgartirilgan
- README endi screenshot va demo GIF bilan keladi.
- Web landing (portal.1pro.uz) qayta dizayn qilindi: feature kartalari,
  yuklab olish tugmalari har platforma uchun.

### Security — Xavfsizlik
- PBKDF2-SHA256 + NaCl secretbox o'rniga PCP-1 (yuqorida).
- Brute-force kuzatuvi: per-portal failed-join hisoblagichi (sozlashda).

### Documentation
- ENGLISH README qo'shildi (README.md asosiy, README-uz.md o'zbekcha).
- PCP-1 spetsifikatsiyasi: `client/crypt/pcp/SPEC.md`.
- PRIVACY.md va TERMS.md qo'shildi.
- Issue/PR template'lari `.github/` ga qo'shildi.

### Known limitations
- Mobile (Android) — eksperimental, bu relizga kirmaydi. Kelajakda alohida
  reliz.
- Code signing yo'q: macOS va Windows binarlari rasmiy imzolanmagan;
  foydalanuvchi qo'lda ruxsat berishi kerak. Keyingi reliz'da imzolanadi.

## [0.3.0] — 2026-04-30

Repodagi oxirgi commit holatigacha — desktop UI va proxy tugagan ichki
versiya. Public reliz emas.

### Added
- Wails + React desktop UI (welcome ekrani, mesh diagrammasi, chat,
  servislar paneli, Settings).
- TCP va UDP proxy (`portal expose`, `portal dial`).
- App-layer NaCl secretbox shifrlash (PBKDF2 portal kodidan).
- SQLite persistence (`~/.portal/portal.db`).
- NAT detection + Cloudflare TURN integratsiyasi.
- Drag-and-drop fayl uzatish.

### Fixed
- Heartbeat outstanding map data race.
- 3+ peer mesh glare (joiner-always-offerer qoidasi).
- Channel close race teardown'da.

Batafsil: [SECURITY-AUDIT.md](SECURITY-AUDIT.md).

## [0.2.0] — 2026-04-30 (ichki)

Client mesh dvigateli — Wails'siz CLI test harness.

## [0.1.0] — 2026-04-30 (ichki)

Signal serveri (xususiy repo'da) ishga tushdi: `signaling.1pro.uz/ws`.

[Unreleased]: https://github.com/Yaxyobek0877/portal_traffic/compare/v0.5.4...HEAD
[0.5.4]: https://github.com/Yaxyobek0877/portal_traffic/releases/tag/v0.5.4
[0.5.3]: https://github.com/Yaxyobek0877/portal_traffic/releases/tag/v0.5.3
[0.5.2]: https://github.com/Yaxyobek0877/portal_traffic/releases/tag/v0.5.2
[0.5.1]: https://github.com/Yaxyobek0877/portal_traffic/releases/tag/v0.5.1
[0.5.0]: https://github.com/Yaxyobek0877/portal_traffic/releases/tag/v0.5.0
[0.4.1]: https://github.com/Yaxyobek0877/portal_traffic/releases/tag/v0.4.1
[0.4.0]: https://github.com/Yaxyobek0877/portal_traffic/releases/tag/v0.4.0
[0.3.0]: https://github.com/Yaxyobek0877/portal_traffic/releases/tag/v0.3.0
[0.2.0]: https://github.com/Yaxyobek0877/portal_traffic/releases/tag/v0.2.0
[0.1.0]: https://github.com/Yaxyobek0877/portal_traffic/releases/tag/v0.1.0
