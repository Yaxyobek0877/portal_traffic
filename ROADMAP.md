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

## 4-bosqich — Sayqal

- [ ] QR kod yaratish + skanerlash modal
- [ ] Ishga tushganda STUN asosidagi NAT turi aniqlash, simmetrik / CGNAT
      uchun banner
- [ ] Drag-and-drop fayl uzatish va progress bar
- [ ] Sozlamalar sahifasi (tarmoq, maxfiylik, ko'rinish, diagnostika)
- [ ] UI da avtomatik qayta ulanish ko'rsatkichi
- [ ] Sozlamalar, portal tarixi, saqlangan kontaktlar uchun SQLite
- [ ] Sozlamalarda bandwidth grafigi

## 5-bosqich — Kuchli imkoniyatlar

- [x] **Lokal TCP proksi** — `portal expose tcp 25565`, `portal dial`
      orqali boshqa peer ning portiga lokal listener qo'yish
      (2026-04-30 audit paytida ko'chirildi, ishlaydi)
- [x] **App-layer secretbox shifrlash** — portal kodidan PBKDF2 bilan
      olingan kalit, har frame uchun yangi nonce
- [ ] UDP proksi (ko'p o'yinlar UDP ham ishlatadi)
- [ ] Mesh bo'ylab fosh qilingan portlarni ko'rsatadigan Services paneli (UI)
- [ ] Push-to-talk bilan ovoz kanali
- [ ] Demo sifatida o'rnatilgan mini-o'yinlar

---

5-bosqichdan keyin loyiha asl spetsifikatsiyaga nisbatan funksional
jihatdan to'liq bo'ladi. Undan keyingi mumkin bo'lgan yo'nalishlar:

- Mobil hamroh ilova (iOS / Android, dastlab faqat ko'rish rejimida)
- Federated discovery (portal ID larni ixtiyoriy DHT da chop etish, shunda
  do'stining portalini 6 xonali kodni ulashmasdan topish mumkin)
- Plugin API — uchinchi tomon ilovalari mesh orqali xizmat ko'rsatishi uchun
