# Yo'l xaritasi (Roadmap)

Portal beshta bosqichda quriladi. Har bir bosqich ishlaydigan to'xtash
nuqtasi — oldingisi tasdiqlangan tarzda ishlamasdan turib keyingisiga
o'tilmaydi.

## 1-bosqich — Signal magistrali ✅

WebRTC handshake larni bog'laydigan ingichka server. Ushbu bosqichdan
keyin ikki client faqat 6 xonali ID + kod orqali bir-birini internetda
topa oladi; undan keyingi har bir narsa to'g'ridan-to'g'ri.

- [x] Per-IP rate limiting bilan WebSocket transport
- [x] Portal hayot tsikli: yaratish / qo'shilish / chetlatish (kick) /
      qulflash / chiqib ketish
- [x] Public / private / friends-only ko'rinish bilan taxallus katalogi
- [x] WebRTC SDP/ICE relay, qat'iy portal ichida cheklangan
- [x] Har bir uzatilgan freym da server tomonidan qo'yilgan `from`
      maydoni (spoof ga qarshi)
- [x] TLS qo'llab-quvvatlash (Cloudflare Origin Certificate yoki
      istalgan PEM/key juftligi)
- [x] systemd hardening profili + deploy qo'llanma
- [x] Integration test paketi + skript orqali end-to-end demo

Signal server kodi xususiy (bizning boshqaradigan VPS da ishlaydi). U
amalga oshiradigan simli protokol [PROTOCOL.md](PROTOCOL.md) da to'liq
hujjatlashtirilgan, shuning uchun istalgan kishi o'zinikini ishga
tushira oladi.

## 2-bosqich — Client yadrosi ✅

Go asosidagi mesh dvigateli, hali UI yo'q. CLI test harness ichida
ikki yoki uchta client to'liq peer-to-peer mesh hosil qiladi va
o'lchangan RTT ni hisobotlaydi.

- [x] `client/` da Go modul (Wails 3-bosqichda qo'shiladi)
- [x] WebSocket orqali signal URL ga ulanish (`signaling.Client`)
- [x] `pion/webrtc/v4` yordamida WebRTC handshake (`peer.Connection`)
- [x] Har bir peer o'rtasida to'liq mesh hosil qilish (`mesh.Manager`)
- [x] To'rtta data kanal multipleks: `control`, `chat`, `transfer`,
      `proxy` (negotiated channel ID lari bilan)
- [x] RTT kuzatuvi bilan heartbeat ping/pong (`mesh/heartbeat.go`)
- [x] CLI test harness (`cmd/portal-cli`): 2-3 ta peer to'liq mesh
      hosil qiladi va sub-ms RTT ko'rsatadi
- [ ] Eksponensial backoff bilan avtomatik qayta ulanish (4-bosqichga
      ko'chirildi)

## 3-bosqich — Desktop UI ✅

Oddiy foydalanuvchi ko'radigan qismlar — Wails + React + TS + Tailwind.

- [x] Welcome ekrani (animatsiyali wormhole logo, taxallus inputi,
      Yaratish / Qo'shilish tugmalari)
- [x] Portal ko'rinishi (sarlavha — ID + kod + QR + copy; peer sidebar;
      animatsiyali mesh diagrammasi; chat paneli; servislar paneli;
      status bar)
- [x] Jonli mesh vizualizatsiyasi (SVG, Framer Motion, edge ustida pulses)
- [x] Go backend va React frontend o'rtasida Wails bog'lanishlari
- [x] Standart qorong'i mode (light mode keyingi versiyada)
- [x] Servislar paneli — fosh qilish + boshqalarniki bilan ulash UI dan
- [x] QR kod modal (offline-tarzda generatsiya)

## 4-bosqich — Sayqal ✅

- [x] QR kod yaratish (3-bosqichda kelgan)
- [x] **Ishga tushganda STUN asosidagi NAT turi aniqlash** — Status bar
      badge'i, simmetrik NAT uchun banner sozlamalarda
- [x] **Drag-and-drop fayl uzatish va progress bar** — chat panelga
      fayl tashlang, har juftlik uchun chunk-chunk uzatiladi
- [x] **Sozlamalar sahifasi** — tarmoq (signal URL), diagnostika (NAT,
      tashqi IP), fayllar (saqlanadigan papka), tarix
- [x] **SQLite persistence** — `~/.portal/portal.db` da sozlamalar,
      portal tarixi (oxirgi 50), kontaktlar
- [x] Welcome ekranida yaqindagi portallar — bir click bilan qayta kirish
- [x] **Avtomatik qayta ulanish ko'rsatkichi** — status bar'da transport
      badge (host / srflx / relay) + peer count; mesh sahifasidagi
      PeerTable har peer uchun ICE selected-pair tipini ko'rsatadi
- [x] **Bandwidth metrikasi** — har peer uchun atomic counter + status
      bar'da yuqi/pastga o'qlar; "Tezligi" tugmasi 32 KB chunk bilan
      3 sekundlik probe qilib Mbps ko'rsatadi (real-time chart hozircha
      kechikadi)

## 5-bosqich — Kuchli imkoniyatlar

- [x] **Lokal TCP proksi** — `portal expose tcp 25565`, `portal dial`
      orqali boshqa peer ning portiga lokal listener qo'yish
      (2026-04-30 audit paytida ko'chirildi, ishlaydi)
- [x] **App-layer secretbox shifrlash** — portal kodidan PBKDF2 bilan
      olingan kalit, har frame uchun yangi nonce
- [x] **UDP proksi** — `f.DialUDP(...)` har source-addr uchun stream
      hosil qiladi; o'yinlar va boshqa UDP servislar uchun
- [x] Servislar paneli (UI) — Phase 3 da kelgan
- [x] **Auto-reconnect** — signaling uzilsa eksponensial backoff bilan
      qayta ulanadi; joiner avtomatik portalga qaytadi
- [x] **Per-peer bandwidth** — atomic counterlar har peer da, status barda
      yuqi/pastga o'qlar bilan ko'rsatiladi
- [x] **Cross-platform build script** — `client/build-all.sh`
- [ ] Push-to-talk bilan ovoz kanali (kelajakda)
- [ ] Demo sifatida o'rnatilgan mini-o'yinlar (kelajakda)

## 6-bosqich — Hisob va Bulut ✅

v0.5.x serial yangilanishlari. Portal endi bitta qurilmaga bog'lanmaydi —
bir marta hisob ochasiz, va o'sha hisob bilan har joydan o'zingizning
portallaringizni boshqarasiz.

- [x] **Hisob tizimi** — `signaling.1pro.uz/api/auth/{signup,signin,signout}` +
      `/api/me`; parol Argon2id bilan hashlanadi, `users.json` atomic
      write bilan saqlanadi; per-IP lockout brute-force'dan himoyalaydi.
      Sessiya cookie `__Host-portal_session` (HttpOnly, Secure).
- [x] **Multi-portal (hisob-vault)** — bir vaqtda bir nechta portalga
      ulanishingiz mumkin; barchasi sidebar'da turadi va orasida
      tab bilan o'tasiz.
- [x] **In-app auto-update** — update banner'dagi "Yangilash" tugmasi
      yangi releasе'ni yuklab oladi, swap-script bilan binary'ni almashtiradi
      va dasturni qayta ishga tushiradi. Foydalanuvchi qo'lda DMG / .exe
      yuklab olib o'rnatishi kerak emas.
- [x] **Cross-device portallar** — desktop endi sign-in qilingan paytda
      o'z aktiv portallarini `/api/portals` ga push qiladi. `portal.1pro.uz/admin/dashboard`
      brauzerdan turib har bir signed-in qurilmangizdagi portallarni ko'rsatadi.
- [x] **Web sign-in/up** — bosh sahifaning forma'si `/api/auth` ga POST
      qiladi; allaqachon kirgan tashrif buyuruvchi formani umuman ko'rmaydi.
      Sign-up'da parol takror + visibility toggle.
- [x] **Web admin shell (skeleton)** — `/admin/login.html`,
      `/admin/dashboard.html` static (Cloudflare Pages auto-deploy);
      real backend wiring (`/api/portals/:id/services`, `service.announce`)
      keyingi sprintlarda.
- [x] **Android client** — desktop bilan teng huquqli: sign-in/up,
      Cloudflare Calls TURN, UZ+EN i18n, brendlangan launcher icon.
      v0.5.5 da Play Store'dan tashqari APK sifatida tarqatiladi.
- [ ] iOS client (kelajakda) — Android'dan keyin

---

6-bosqichdan keyin loyiha tarmoqlardan tashqari hisob va platforma
qatlami bilan ham yopilgan bo'ladi. Undan keyingi mumkin bo'lgan
yo'nalishlar:

- Federated discovery (portal ID larni ixtiyoriy DHT da chop etish,
  shunda do'stining portalini 6 xonali kodni ulashmasdan topish mumkin)
- Plugin API — uchinchi tomon ilovalari mesh orqali xizmat ko'rsatishi uchun
- Push-to-talk ovoz / video kanali
- Tashqi tarmoqlarga gateway (Tailscale-style subnet routing)
