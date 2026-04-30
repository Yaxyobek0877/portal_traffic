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

## 2-bosqich — Client yadrosi (jarayonda)

Go asosidagi mesh dvigateli, hali UI yo'q. Bu bosqich oxirida maqsad —
turli tarmoqlardagi ikki CLI test client quyidagilarni qila oladigan
bo'lishi:

- [ ] Wails loyiha skeletini ochish (`client/`)
- [ ] Xavfsiz WebSocket orqali signal URL ga ulanish
- [ ] `pion/webrtc/v4` yordamida WebRTC handshake ni o'tkazish
- [ ] Har bir peer o'rtasida to'liq mesh hosil qilish
- [ ] To'rtta data kanalni multipleks qilish: `control`, `chat`,
      `transfer`, `proxy`
- [ ] RTT kuzatuvi bilan heartbeat ping/pong
- [ ] Eksponensial backoff bilan avtomatik qayta ulanish
- [ ] CLI smoke test: ikkita client `control` kanali orqali `ping`
      almashadi va o'lchangan RTT ni hisobot qiladi

## 3-bosqich — Desktop UI

Oddiy foydalanuvchi ko'radigan qismlar.

- [ ] Welcome ekrani (animatsiyali wormhole logo, taxallus inputi,
      Yaratish / Qo'shilish tugmalari)
- [ ] Portal ko'rinishi (sarlavha — ID + kod + QR; peer sidebar;
      animatsiyali mesh diagrammasi; chat paneli; status bar)
- [ ] Jonli mesh vizualizatsiyasi (SVG, Framer Motion, ma'lumot oqimida
      yorishish)
- [ ] Go backend va React frontend o'rtasida Wails bog'lanishlari
- [ ] Standart qorong'i mode, parametrlarda yorug' mode tugmasi

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

- [ ] Virtual IP larni WebRTC `proxy` kanali bilan bog'laydigan lokal
      TCP/UDP proksi (`portal expose tcp 25565`)
- [ ] Mesh bo'ylab fosh qilingan portlarni ko'rsatadigan Services paneli
- [ ] Push-to-talk bilan ovoz kanali
- [ ] Demo sifatida o'rnatilgan mini-o'yinlar (data kanali ustida
      tic-tac-toe)

---

5-bosqichdan keyin loyiha asl spetsifikatsiyaga nisbatan funksional
jihatdan to'liq bo'ladi. Undan keyingi mumkin bo'lgan yo'nalishlar:

- Mobil hamroh ilova (iOS / Android, dastlab faqat ko'rish rejimida)
- Federated discovery (portal ID larni ixtiyoriy DHT da chop etish, shunda
  do'stining portalini 6 xonali kodni ulashmasdan topish mumkin)
- Plugin API — uchinchi tomon ilovalari mesh orqali xizmat ko'rsatishi uchun
