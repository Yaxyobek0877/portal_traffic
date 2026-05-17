// app.js — landing page interactions and animations
// All effects respect prefers-reduced-motion.

(() => {
  const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches;

  // ---------------------------------------------------------------
  // 0. i18n — UZ + EN dictionary
  // ---------------------------------------------------------------
  // Two-language toggle for the landing page. Persists to the same
  // localStorage key as /admin/* (portal:lang) so a single decision
  // sticks across both surfaces. Text nodes are tagged with
  // `data-i18n="key"` (textContent) or `data-i18n-html="key"`
  // (innerHTML, for nodes that contain inline tags like <br>, <span>,
  // <code>, <strong>). Attributes can be translated via
  // `data-i18n-attr="attr1:key1,attr2:key2"` (e.g. placeholder, title).
  const DICT = {
    uz: {
      // nav
      "nav.features":  "Imkoniyatlar",
      "nav.whatsnew":  "v0.5.x da yangi",
      "nav.mesh":      "Mesh",
      "nav.how":       "Qanday ishlaydi",
      "nav.roadmap":   "Roadmap",
      "nav.signin":    "Kirish",
      "nav.download":  "Yuklab olish",

      // pills
      "pill.live":     "signaling.1pro.uz · jonli",
      "pill.release":  "v0.5.0 · jonli reliz",
      "pill.pcp":      "PCP-1 shifrlash",

      // hero
      "hero.h1":       "To'g'ridan-to'g'ri ulanish.<br/>Orada hech qanday <span class=\"gradient-text\">server yo'q</span>.",
      "hero.tag":      "Portal — qurilmalar o'rtasida xususiy <strong>peer-to-peer mesh</strong> tarmoq quradigan desktop dastur. Mahalliy hisob, bir nechta xona bir vaqtda, avtomatik qayta ulanish. Chat, fayl, kamera/NVR/o'yin serveri uchun port tunneli — hammasi P2P.",
      "hero.dlhead":   "Yuklab olish · Download",
      "hero.releases": "Barcha relizlar:",
      "hero.androidsrc": "Android (manbadan)",
      "hero.btn.release":  "Reliz sahifasi",
      "hero.btn.github":   "GitHub'da ko'rish",

      // download tiles
      "dl.mac.arm.arch": "Apple Silicon (M1/M2/M3/M4)",
      "dl.mac.intel.arch": "Intel (x86_64)",
      "dl.win.arch":   "10 / 11 · 64-bit",
      "dl.linux.arch": "Ubuntu 22.04+ · 64-bit",

      // login section
      "login.badge":   "Mavjud · v0.5.5",
      "login.h2":      "Brauzerdan boshqaring",
      "login.p":       "Hisobingizga kirib portal'laringizni, ulangan qurilmalarni va ochilgan portlarni <strong>brauzer orqali</strong> nazorat qiling. Desktop dasturni ochmasdan ham — telefondan, ish kompyuteringizdan, istalgan joydan.",
      "login.li1":     "Faol portallar va ularning ko'rinarli kodlari",
      "login.li2":     "Online/offline qurilmalar, RTT, ulanish turi (P2P / TURN)",
      "login.li3":     "Servislarni vaqtinchalik to'xtatish, qayta yoqish",
      "login.li4":     "Activity log — kim qachon qaysi servisga ulangan",
      "login.cta.h":   "Admin paneliga kirish",
      "login.cta.p":   "Mahalliy hisob — bcrypt parol, sessiya cookie'si <code>__Host-</code> prefiksi bilan.",
      "login.cta.in":  "Kirish",
      "login.cta.up":  "Ro'yxatdan o'tish",
      "login.cta.foot":"Akkaunt mahalliy — email kerak emas, parolingiz hech qachon serverga ochiq holda yuborilmaydi.",

      // features section
      "f.eyebrow":     "Imkoniyatlar",
      "f.h2":          "Bir portal ichida — istalgan narsa.",
      "f.lede":        "Bir marta kod ulashing va do'stlaringiz bilan birga ishlash uchun zarur bo'lgan barcha narsa qo'lingizda. Hech qanday markaziy server, hisob qaydnomasi yoki port forwarding kerak emas.",
      "f1.h":          "2 dan 16 qurilmagacha",
      "f1.p":          "Portal ID + kod ulashing — do'stlaringiz internetning istalgan joyidan to'g'ridan-to'g'ri qo'shiladi. Router sozlash yo'q, VPN yo'q, port forwarding yo'q.",
      "f2.h":          "Chat va fayl",
      "f2.p":          "Guruh chat, shaxsiy xabar, drag-and-drop fayl uzatish. Ma'lumot portal kodidan olingan kalit bilan uchidan-uchiga shifrlanadi.",
      "f3.h":          "TCP/UDP tunnel",
      "f3.p":          "Minecraft serverni <code style=\"font-family:'JetBrains Mono',monospace;color:#22d3ee\">localhost:25565</code> da oching, do'stlaringiz <code style=\"font-family:'JetBrains Mono',monospace;color:#22d3ee\">10.42.0.3:25565</code> orqali ulanadi.",
      "f4.h":          "Jonli mesh ko'rinish",
      "f4.p":          "Kim kim bilan ulangani jonli vizual ko'rinishda. Haqiqiy vaqt RTT, NAT-traversal indikatorlari (to'g'ridan-to'g'ri yoki TURN orqali).",
      "f5.h":          "Ikki qatlam shifrlash",
      "f5.p":          "WebRTC DTLS ostida <a href=\"https://github.com/Yaxyobek0877/portal_traffic/blob/main/client/crypt/pcp/SPEC.md\" style=\"color:var(--accent-3)\">PCP-1</a> — Ed25519 identity + X25519 ephemeral + XChaCha20-Poly1305. Per-pair forward secrecy.",
      "f6.h":          "QR + portal kod",
      "f6.p":          "Yonidagi do'stga QR ni ko'rsating yoki masofadagi do'stga ID + kodni yuboring. Email yoki tashqi ro'yxatdan o'tish yo'q — faqat mahalliy hisob.",

      // v0.5.x section
      "n.eyebrow":     "v0.5.x da yangi",
      "n.h2":          "Bir desktop, <span class=\"gradient-text\">to'liq xona boshqaruvi</span>.",
      "n.lede":        "Oxirgi relizda hisob, multi-portal, avtomatik qayta ulanish va xavfsizlik gate'lari qo'shildi. Hammasi mahalliy — signaling server faqat handshake'da ishtirok etadi.",
      "n1.h":          "Mahalliy hisob",
      "n1.p":          "Username + bcrypt parol (8 ta belgi, kuchli). Akkaunt va portal tarixingiz hech qachon serverga jo'natilmaydi.",
      "n2.h":          "Bir nechta xona bir vaqtda",
      "n2.p":          "\"Fonda ulash\" bilan oilaviy + ishxona + o'yin xonalarini parallel oching. Header'dagi switcher bilan o'tib turing.",
      "n3.h":          "Avtomatik qayta ulanish",
      "n3.p":          "Tizim qayta yuklansa, oldingi sessiyalar va exposed servislar lock'dan chiqishi bilan avtomatik tiklanadi.",
      "n4.h":          "Tasdiqlab yoqish",
      "n4.p":          "Har servisga \"approval mode\" toggle'i. Peer ulanmoqchi bo'lganda sizga modal chiqadi: <em>kim, qaysi servis</em>, ruxsat berasizmi?",
      "n5.h":          "Qurilma nomi",
      "n5.p":          "Bir akkauntdan turli qurilmalardan kirsangiz, room'da <code style=\"font-family:'JetBrains Mono',monospace;color:#22d3ee\">texuz · uy</code> ko'rinishida farqlanasiz.",
      "n6.h":          "LAN qurilmalar (NVR/kamera)",
      "n6.p":          "Hikvision NVR'ni LAN'dagi <code style=\"font-family:'JetBrains Mono',monospace;color:#22d3ee\">192.168.x.x:554</code> dan mesh orqali do'stga uzating. Mesh-port remap, smart-paste IP:port, health probe.",
      "n7.h":          "Tizim startup'i",
      "n7.p":          "macOS LaunchAgent / Windows registry / Linux .desktop bilan boot keyin avtomatik ishga tushish — settings'dan toggle.",
      "n8.h":          "Bandwidth probe + path tagi",
      "n8.p":          "Har peer kartasida 3-soniyalik bandwidth o'lchov, ICE selected-pair manzillari, LAN/Internet/TURN tagi.",

      // mesh section
      "m.eyebrow":     "Mesh ichida",
      "m.h2":          "Har bir peer — boshqa har biriga.",
      "m.lede":        "Tarmoq topologiyasi to'liq mesh: 5 ta qurilmada <strong style=\"color:var(--text)\">10 ta</strong> to'g'ridan-to'g'ri ulanish, 16 ta qurilmada <strong style=\"color:var(--text)\">120 ta</strong>. Server faqat handshake da ishtirok etadi, undan keyin hech qanday rolda emas.",
      "m.stat1":       "faol ulanish",
      "m.stat2":       "o'rtacha RTT (lokal)",
      "m.stat3":       "data kanal har juftlikka",

      // steps section
      "s.eyebrow":     "Qanday ishlaydi",
      "s.h2":          "To'rt qadamda <span class=\"gradient-text\">jonli mesh</span>.",
      "s.lede":        "Akkaunt mahalliy — bcrypt-himoyalangan parol bilan, qurilmangizdan tashqariga chiqmaydi. Email yoki tashqi serverda ro'yxatdan o'tish yo'q.",
      "s1.h":          "Hisob yarating",
      "s1.p":          "Birinchi ochishda <strong>username + parol</strong> tanlang (8 ta belgi, kuchli). Keyingi ochishlarda lock screen'dan kirasiz.",
      "s2.h":          "Portalni oching",
      "s2.p":          "<strong>\"Portal yaratish\"</strong> bossangiz yangi xona ID va kod bilan tug'iladi.",
      "s3.h":          "Kodni ulashing",
      "s3.p":          "Do'stlaringizga raqamlarni yuboring yoki QR kodni ko'rsating.",
      "s4.h":          "Mesh tushadi",
      "s4.p":          "Har bir qurilma boshqa har biriga to'g'ridan-to'g'ri ulanadi. Server orqali emas.",

      // install section
      "i.eyebrow":     "Boshlash",
      "i.h2":          "Uchta buyruq, mesh tayyor.",
      "i.lede":        "Desktop binary tayyor (Wails + React). CLI test harness ham to'liq mesh ni quradi va RTT ni ko'rsatadi — har ikkala yo'l ham ishlaydi.",
      "i.head":        "terminal · clone va build",
      "i.foot":        "Standart endpoint: <code style=\"font-family:'JetBrains Mono',monospace;color:var(--accent-3)\">wss://signaling.1pro.uz/ws</code>. O'zingizniki ishlatish uchun <a href=\"https://github.com/Yaxyobek0877/portal_traffic/blob/main/PROTOCOL.md\" style=\"color:var(--accent-3)\">PROTOCOL.md</a> ga qarang.",

      // roadmap
      "r.eyebrow":     "Joriy holat",
      "r.h2":          "Bosqichma-bosqich quriladi.",
      "r.lede":        "Birinchi sakkiz bosqich tugadi — Portal funksional jihatdan to'liq. Har bir bosqich ishlaydigan to'xtash nuqtasi: oldingisi tasdiqlangan tarzda ishlamasdan turib keyingisiga o'tilmaydi.",
      "r.status.done": "tugadi",
      "r.status.wip":  "jarayonda",
      "r.status.next": "keyin",
      "r1.n":  "Signal magistrali",
      "r1.d":  "Portal lifecycle, WebRTC SDP/ICE relay, TLS, rate-limiting",
      "r2.n":  "Client yadrosi",
      "r2.d":  "pion/webrtc, multipleks data channel, full mesh, heartbeat",
      "r3.n":  "Desktop UI",
      "r3.d":  "Wails + React, welcome ekrani, animatsiyali mesh diagrammasi, chat, QR",
      "r4.n":  "Sayqal",
      "r4.d":  "NAT detection, fayl uzatish, sozlamalar, SQLite, yaqindagi portallar",
      "r5.n":  "Kuchli imkoniyatlar",
      "r5.d":  "TCP/UDP proxy, secretbox shifrlash, bandwidth metrikasi, peer table",
      "r6.n":  "v0.4.0 launch",
      "r6.d":  "PCP-1 shifrlash, MIT litsenziya, GitHub Actions CI/CD, Privacy/Terms, mobile (Android) beta",
      "r7.n":  "v0.5.0 — meeting tajribasi",
      "r7.d":  "Mahalliy hisob (vault), multi-portal sessiyalar, background-connect, avtomatik qayta ulanish, per-port approval gate, persistent peers",
      "r8.n":  "v0.5.1 — desktop polish",
      "r8.d":  "App icons (mac/win/linux), qurilma nomi, system startup auto-run, stabil download URL'lari",
      "r9.n":  "Code signing",
      "r9.d":  "Apple Developer ID + Authenticode — Gatekeeper / SmartScreen ogohlantirishlarini olib tashlash",
      "r10.n": "Cloud auth + xona boshqaruvi",
      "r10.d": "Server-side /api/auth (multi-device login), kod regenerate, \"ask admin\" join mode, owner-offline portal-survival",
      "r11.n": "Mobile (Android) reliz",
      "r11.d": "Play Store internal track, PCP-1 portativ, signed APK",
      "r12.n": "Kelajak",
      "r12.d": "Ovoz/video kanali (WebRTC media), iOS, plugin API",

      // big cta + footer
      "bigcta.h":  "Loyihaga qarshing.",
      "bigcta.p":  "Kod ochiq. Issue oching, patch yuboring, yoki o'zingizning fork ni quring.",
      "bigcta.repo": "GitHub Repo",
      "bigcta.contrib": "Hissa qo'shish",
      "ft.roadmap":  "Roadmap",
      "ft.arch":     "Arxitektura",
      "ft.proto":    "Protokol",
      "ft.security": "Xavfsizlik",
      "ft.privacy":  "Maxfiylik",
      "ft.terms":    "Shartlar",
    },
    en: {
      "nav.features":  "Features",
      "nav.whatsnew":  "v0.5.x highlights",
      "nav.mesh":      "Mesh",
      "nav.how":       "How it works",
      "nav.roadmap":   "Roadmap",
      "nav.signin":    "Sign in",
      "nav.download":  "Download",

      "pill.live":     "signaling.1pro.uz · live",
      "pill.release":  "v0.5.0 · live release",
      "pill.pcp":      "PCP-1 encryption",

      "hero.h1":       "Direct connection.<br/>No <span class=\"gradient-text\">server in the middle</span>.",
      "hero.tag":      "Portal is a desktop app that builds a private <strong>peer-to-peer mesh</strong> between your devices. Local account, multiple rooms at once, auto-reconnect. Chat, file transfer, port tunnels for cameras/NVRs/game servers — all P2P.",
      "hero.dlhead":   "Download",
      "hero.releases": "All releases:",
      "hero.androidsrc": "Android (from source)",
      "hero.btn.release":  "Release page",
      "hero.btn.github":   "View on GitHub",

      "dl.mac.arm.arch": "Apple Silicon (M1/M2/M3/M4)",
      "dl.mac.intel.arch": "Intel (x86_64)",
      "dl.win.arch":   "10 / 11 · 64-bit",
      "dl.linux.arch": "Ubuntu 22.04+ · 64-bit",

      "login.badge":   "Live · v0.5.5",
      "login.h2":      "Manage from your browser",
      "login.p":       "Sign in to control your portals, connected devices, and exposed ports right <strong>from your browser</strong> — without opening the desktop app. From your phone, work laptop, anywhere.",
      "login.li1":     "Active portals and their codes (owner-only)",
      "login.li2":     "Online/offline devices, RTT, connection type (P2P / TURN)",
      "login.li3":     "Pause and resume exposed services",
      "login.li4":     "Activity log — who connected to what, when",
      "login.cta.h":   "Sign in to admin",
      "login.cta.p":   "Local account — bcrypt password, session cookie with the <code>__Host-</code> prefix.",
      "login.cta.in":  "Sign in",
      "login.cta.up":  "Create account",
      "login.cta.foot":"Local account — no email needed, your password never leaves your device in plaintext.",

      "f.eyebrow":     "Features",
      "f.h2":          "One portal — anything you need.",
      "f.lede":        "Share a code once and have everything you need to collaborate with friends. No central server, account database, or port forwarding required.",
      "f1.h":          "2 to 16 devices",
      "f1.p":          "Share a portal ID + code — friends join directly from anywhere on the internet. No router setup, no VPN, no port forwarding.",
      "f2.h":          "Chat and files",
      "f2.p":          "Group chat, direct messages, drag-and-drop file transfer. End-to-end encrypted with a key derived from the portal code.",
      "f3.h":          "TCP/UDP tunnels",
      "f3.p":          "Run a Minecraft server on <code style=\"font-family:'JetBrains Mono',monospace;color:#22d3ee\">localhost:25565</code> and friends connect via <code style=\"font-family:'JetBrains Mono',monospace;color:#22d3ee\">10.42.0.3:25565</code>.",
      "f4.h":          "Live mesh view",
      "f4.p":          "See who's connected to whom in real time. Live RTT, NAT-traversal indicators (direct or via TURN).",
      "f5.h":          "Double-layer encryption",
      "f5.p":          "WebRTC DTLS plus <a href=\"https://github.com/Yaxyobek0877/portal_traffic/blob/main/client/crypt/pcp/SPEC.md\" style=\"color:var(--accent-3)\">PCP-1</a> — Ed25519 identity + X25519 ephemeral + XChaCha20-Poly1305. Per-pair forward secrecy.",
      "f6.h":          "QR + portal code",
      "f6.p":          "Show the QR to a friend next to you or send the ID + code to a remote friend. No email or external signup — just local accounts.",

      "n.eyebrow":     "What's new in v0.5.x",
      "n.h2":          "One desktop, <span class=\"gradient-text\">full room control</span>.",
      "n.lede":        "The latest releases added accounts, multi-portal, auto-reconnect, and security gates. Everything stays local — the signaling server only handles the handshake.",
      "n1.h":          "Local account",
      "n1.p":          "Username + bcrypt password (8 chars, strong). Your account and portal history never leave your device.",
      "n2.h":          "Multiple rooms at once",
      "n2.p":          "Background-connect to family + work + gaming rooms in parallel. Hop between them with the header switcher.",
      "n3.h":          "Auto-reconnect",
      "n3.p":          "After a reboot, prior sessions and exposed services restore automatically once you unlock the vault.",
      "n4.h":          "Approval mode",
      "n4.p":          "Per-service \"approval\" toggle. When a peer tries to connect, a modal asks: <em>who, which service</em>, allow?",
      "n5.h":          "Device name",
      "n5.p":          "Sign in from multiple devices on one account and rooms show you as <code style=\"font-family:'JetBrains Mono',monospace;color:#22d3ee\">texuz · home</code> to keep them distinguishable.",
      "n6.h":          "LAN devices (NVR/camera)",
      "n6.p":          "Stream a Hikvision NVR from LAN <code style=\"font-family:'JetBrains Mono',monospace;color:#22d3ee\">192.168.x.x:554</code> over the mesh to a friend. Mesh-port remap, smart-paste IP:port, health probes.",
      "n7.h":          "System startup",
      "n7.p":          "Auto-start at boot via macOS LaunchAgent / Windows registry / Linux .desktop — toggle in settings.",
      "n8.h":          "Bandwidth probe + path tag",
      "n8.p":          "Per-peer 3-second bandwidth measurement, ICE selected-pair addresses, LAN/Internet/TURN tag.",

      "m.eyebrow":     "Inside the mesh",
      "m.h2":          "Every peer — to every other.",
      "m.lede":        "Topology is full mesh: 5 devices = <strong style=\"color:var(--text)\">10</strong> direct connections, 16 devices = <strong style=\"color:var(--text)\">120</strong>. The server only participates in the handshake; never after.",
      "m.stat1":       "active links",
      "m.stat2":       "average RTT (local)",
      "m.stat3":       "data channels per pair",

      "s.eyebrow":     "How it works",
      "s.h2":          "Live mesh in <span class=\"gradient-text\">four steps</span>.",
      "s.lede":        "Local account — bcrypt-protected password, never leaves your device. No email or external signup.",
      "s1.h":          "Create an account",
      "s1.p":          "On first launch, choose a <strong>username + password</strong> (8 chars, strong). Subsequent launches go through the lock screen.",
      "s2.h":          "Open a portal",
      "s2.p":          "Click <strong>\"Create portal\"</strong> and a new room is born with an ID and code.",
      "s3.h":          "Share the code",
      "s3.p":          "Send the numbers to friends or show the QR code.",
      "s4.h":          "Mesh forms",
      "s4.p":          "Every device connects directly to every other. Not through the server.",

      "i.eyebrow":     "Get started",
      "i.h2":          "Three commands, mesh ready.",
      "i.lede":        "Desktop binary is ready (Wails + React). The CLI test harness builds a full mesh and shows RTT — either path works.",
      "i.head":        "terminal · clone and build",
      "i.foot":        "Default endpoint: <code style=\"font-family:'JetBrains Mono',monospace;color:var(--accent-3)\">wss://signaling.1pro.uz/ws</code>. To run your own see <a href=\"https://github.com/Yaxyobek0877/portal_traffic/blob/main/PROTOCOL.md\" style=\"color:var(--accent-3)\">PROTOCOL.md</a>.",

      "r.eyebrow":     "Current status",
      "r.h2":          "Built step by step.",
      "r.lede":        "The first eight phases are done — Portal is functionally complete. Each phase is a working checkpoint; we don't move on until the previous one is verified.",
      "r.status.done": "done",
      "r.status.wip":  "in progress",
      "r.status.next": "next",
      "r1.n":  "Signal backbone",
      "r1.d":  "Portal lifecycle, WebRTC SDP/ICE relay, TLS, rate-limiting",
      "r2.n":  "Client core",
      "r2.d":  "pion/webrtc, multiplexed data channels, full mesh, heartbeat",
      "r3.n":  "Desktop UI",
      "r3.d":  "Wails + React, welcome screen, animated mesh diagram, chat, QR",
      "r4.n":  "Polish",
      "r4.d":  "NAT detection, file transfer, settings, SQLite, recent portals",
      "r5.n":  "Power features",
      "r5.d":  "TCP/UDP proxy, secretbox encryption, bandwidth metrics, peer table",
      "r6.n":  "v0.4.0 launch",
      "r6.d":  "PCP-1 encryption, MIT license, GitHub Actions CI/CD, Privacy/Terms, mobile (Android) beta",
      "r7.n":  "v0.5.0 — meeting experience",
      "r7.d":  "Local account (vault), multi-portal sessions, background-connect, auto-reconnect, per-port approval gate, persistent peers",
      "r8.n":  "v0.5.1 — desktop polish",
      "r8.d":  "App icons (mac/win/linux), device name, system startup auto-run, stable download URLs",
      "r9.n":  "Code signing",
      "r9.d":  "Apple Developer ID + Authenticode — remove Gatekeeper / SmartScreen warnings",
      "r10.n": "Cloud auth + room control",
      "r10.d": "Server-side /api/auth (multi-device login), code regenerate, \"ask admin\" join mode, owner-offline portal survival",
      "r11.n": "Mobile (Android) release",
      "r11.d": "Play Store internal track, PCP-1 portable, signed APK",
      "r12.n": "Future",
      "r12.d": "Voice/video channel (WebRTC media), iOS, plugin API",

      "bigcta.h":  "Join the project.",
      "bigcta.p":  "Source is open. Open an issue, send a patch, or build your own fork.",
      "bigcta.repo": "GitHub repo",
      "bigcta.contrib": "Contribute",
      "ft.roadmap":  "Roadmap",
      "ft.arch":     "Architecture",
      "ft.proto":    "Protocol",
      "ft.security": "Security",
      "ft.privacy":  "Privacy",
      "ft.terms":    "Terms",
    },
  };

  const LANG_KEY = "portal:lang";
  function readLang() {
    try {
      const v = localStorage.getItem(LANG_KEY);
      if (v === "uz" || v === "en") return v;
    } catch (_) {}
    return (navigator.language || "").toLowerCase().startsWith("en") ? "en" : "uz";
  }
  function writeLang(l) {
    try { localStorage.setItem(LANG_KEY, l); } catch (_) {}
    applyLang(l);
  }
  function applyLang(l) {
    const dict = DICT[l] || DICT.uz;
    document.documentElement.lang = l;
    document.querySelectorAll("[data-i18n]").forEach((el) => {
      const k = el.getAttribute("data-i18n");
      if (dict[k] != null) el.textContent = dict[k];
    });
    document.querySelectorAll("[data-i18n-html]").forEach((el) => {
      const k = el.getAttribute("data-i18n-html");
      if (dict[k] != null) el.innerHTML = dict[k];
    });
    document.querySelectorAll("[data-i18n-attr]").forEach((el) => {
      el.getAttribute("data-i18n-attr").split(",").forEach((pair) => {
        const [attr, k] = pair.split(":").map((s) => s.trim());
        if (attr && k && dict[k] != null) el.setAttribute(attr, dict[k]);
      });
    });
    // Update the toggle's active state.
    document.querySelectorAll(".lang-toggle button[data-lang]").forEach((b) => {
      b.classList.toggle("active", b.dataset.lang === l);
      b.setAttribute("aria-pressed", b.dataset.lang === l ? "true" : "false");
    });
  }
  // Bind clicks on the toggle (rendered statically in the nav).
  document.addEventListener("click", (ev) => {
    const b = ev.target.closest(".lang-toggle button[data-lang]");
    if (!b) return;
    writeLang(b.dataset.lang);
  });
  // Initial paint.
  applyLang(readLang());

  // ---------------------------------------------------------------
  // 1. Reveal-on-scroll with stagger
  // ---------------------------------------------------------------
  const io = new IntersectionObserver((entries) => {
    for (const e of entries) {
      if (e.isIntersecting) {
        e.target.classList.add('in');
        io.unobserve(e.target);
      }
    }
  }, { threshold: 0.12, rootMargin: '0px 0px -40px 0px' });

  document.querySelectorAll('.reveal').forEach((el, i) => {
    el.style.transitionDelay = (i % 6) * 60 + 'ms';
    io.observe(el);
  });

  // ---------------------------------------------------------------
  // 2. Copy-to-clipboard buttons
  // ---------------------------------------------------------------
  document.querySelectorAll('.copy[data-copy]').forEach((btn) => {
    btn.addEventListener('click', async () => {
      try {
        await navigator.clipboard.writeText(btn.dataset.copy);
        const old = btn.textContent;
        btn.textContent = 'copied ✓';
        btn.classList.add('flash');
        setTimeout(() => { btn.textContent = old; btn.classList.remove('flash'); }, 1400);
      } catch (e) { /* ignore */ }
    });
  });

  if (reduced) {
    // Render a static mesh so the section is still meaningful for
    // motion-sensitive users — no animation, just nodes and edges.
    const svg = document.getElementById('live-mesh');
    if (svg) {
      const NS = 'http://www.w3.org/2000/svg';
      const N = 5, cx = 250, cy = 200, R = 130;
      const peers = Array.from({ length: N }, (_, i) => {
        const a = (i / N) * Math.PI * 2 - Math.PI / 2;
        return { x: cx + Math.cos(a) * R, y: cy + Math.sin(a) * R };
      });
      for (let i = 0; i < N; i++) {
        for (let j = i + 1; j < N; j++) {
          const ln = document.createElementNS(NS, 'line');
          ln.setAttribute('x1', peers[i].x); ln.setAttribute('y1', peers[i].y);
          ln.setAttribute('x2', peers[j].x); ln.setAttribute('y2', peers[j].y);
          ln.setAttribute('stroke', '#6366f1');
          ln.setAttribute('stroke-width', '1');
          ln.setAttribute('opacity', '0.6');
          svg.appendChild(ln);
        }
      }
      peers.forEach((p, i) => {
        const dot = document.createElementNS(NS, 'circle');
        dot.setAttribute('cx', p.x); dot.setAttribute('cy', p.y);
        dot.setAttribute('r', 8);
        dot.setAttribute('fill', 'url(#nodeFill)');
        svg.appendChild(dot);
        const lbl = document.createElementNS(NS, 'text');
        const ang = Math.atan2(p.y - cy, p.x - cx);
        lbl.setAttribute('x', p.x + Math.cos(ang) * 28);
        lbl.setAttribute('y', p.y + Math.sin(ang) * 28 + 4);
        lbl.setAttribute('text-anchor', 'middle');
        lbl.setAttribute('fill', '#98a2b3');
        lbl.setAttribute('font-family', 'JetBrains Mono, monospace');
        lbl.setAttribute('font-size', '11');
        lbl.textContent = `peer-${i + 1}`;
        svg.appendChild(lbl);
      });
    }
    return;
  }

  // ---------------------------------------------------------------
  // 3. Particle network canvas
  //    Subtle drifting points; lines connect close pairs. Mouse
  //    repels gently. Renders behind everything via CSS z-index.
  // ---------------------------------------------------------------
  const canvas = document.getElementById('particles');
  if (canvas) {
    const ctx = canvas.getContext('2d');
    let w, h, dpr, particles, mouse = { x: -9999, y: -9999 };

    function resize() {
      dpr = Math.min(window.devicePixelRatio || 1, 2);
      w = canvas.clientWidth = window.innerWidth;
      h = canvas.clientHeight = window.innerHeight;
      canvas.width  = w * dpr;
      canvas.height = h * dpr;
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
      // Density scales with viewport area; clamp so phones don't melt.
      const target = Math.min(120, Math.max(40, Math.floor((w * h) / 18000)));
      particles = Array.from({ length: target }, () => spawn());
    }
    function spawn() {
      return {
        x: Math.random() * w,
        y: Math.random() * h,
        vx: (Math.random() - 0.5) * 0.18,
        vy: (Math.random() - 0.5) * 0.18,
        r: Math.random() * 1.4 + 0.4,
      };
    }

    function tick() {
      ctx.clearRect(0, 0, w, h);
      const linkDist = 130;
      const linkDist2 = linkDist * linkDist;
      const repelDist = 110;
      const repelDist2 = repelDist * repelDist;

      // Update + draw nodes
      for (const p of particles) {
        // Mouse repulsion
        const dx = p.x - mouse.x, dy = p.y - mouse.y;
        const d2 = dx * dx + dy * dy;
        if (d2 < repelDist2) {
          const f = (1 - d2 / repelDist2) * 0.4;
          p.vx += (dx / Math.sqrt(d2 + 0.001)) * f * 0.05;
          p.vy += (dy / Math.sqrt(d2 + 0.001)) * f * 0.05;
        }
        p.x += p.vx; p.y += p.vy;
        p.vx *= 0.99; p.vy *= 0.99;
        if (p.x < 0) { p.x = 0; p.vx *= -1; }
        if (p.y < 0) { p.y = 0; p.vy *= -1; }
        if (p.x > w) { p.x = w; p.vx *= -1; }
        if (p.y > h) { p.y = h; p.vy *= -1; }

        ctx.beginPath();
        ctx.arc(p.x, p.y, p.r, 0, Math.PI * 2);
        ctx.fillStyle = 'rgba(180, 195, 230, 0.55)';
        ctx.fill();
      }

      // Connect lines under threshold
      for (let i = 0; i < particles.length; i++) {
        const a = particles[i];
        for (let j = i + 1; j < particles.length; j++) {
          const b = particles[j];
          const dx = a.x - b.x, dy = a.y - b.y;
          const d2 = dx * dx + dy * dy;
          if (d2 < linkDist2) {
            const t = 1 - d2 / linkDist2;
            ctx.strokeStyle = `rgba(139,92,246,${t * 0.18})`;
            ctx.lineWidth = 0.7;
            ctx.beginPath();
            ctx.moveTo(a.x, a.y); ctx.lineTo(b.x, b.y);
            ctx.stroke();
          }
        }
      }
      raf = requestAnimationFrame(tick);
    }

    let raf;
    window.addEventListener('mousemove', (e) => { mouse.x = e.clientX; mouse.y = e.clientY; });
    window.addEventListener('mouseleave', () => { mouse.x = mouse.y = -9999; });
    window.addEventListener('resize', resize);
    document.addEventListener('visibilitychange', () => {
      if (document.hidden) { cancelAnimationFrame(raf); }
      else { raf = requestAnimationFrame(tick); }
    });
    resize();
    raf = requestAnimationFrame(tick);
  }

  // ---------------------------------------------------------------
  // 4. Wormhole hero — mouse parallax tilt
  // ---------------------------------------------------------------
  const wh = document.querySelector('.wormhole-wrap');
  const hero = document.querySelector('section.hero');
  if (wh && hero) {
    let tx = 0, ty = 0, cx = 0, cy = 0;
    hero.addEventListener('mousemove', (e) => {
      const r = hero.getBoundingClientRect();
      const px = (e.clientX - r.left) / r.width  - 0.5;
      const py = (e.clientY - r.top)  / r.height - 0.5;
      tx = px * 16;
      ty = py * 16;
    });
    hero.addEventListener('mouseleave', () => { tx = 0; ty = 0; });
    function frame() {
      cx += (tx - cx) * 0.08;
      cy += (ty - cy) * 0.08;
      wh.style.transform = `translate3d(${cx * 0.6}px, ${cy * 0.6}px, 0) rotateY(${cx}deg) rotateX(${-cy}deg)`;
      requestAnimationFrame(frame);
    }
    frame();
  }

  // ---------------------------------------------------------------
  // 5. 3D tilt on feature cards
  // ---------------------------------------------------------------
  document.querySelectorAll('.card.tilt').forEach((card) => {
    let raf, tx = 0, ty = 0, cx = 0, cy = 0, gx = 50, gy = 50;
    card.addEventListener('mousemove', (e) => {
      const r = card.getBoundingClientRect();
      const px = (e.clientX - r.left) / r.width  - 0.5;
      const py = (e.clientY - r.top)  / r.height - 0.5;
      tx = -py * 8;
      ty = px * 8;
      gx = ((e.clientX - r.left) / r.width)  * 100;
      gy = ((e.clientY - r.top)  / r.height) * 100;
      if (!raf) raf = requestAnimationFrame(apply);
    });
    card.addEventListener('mouseleave', () => {
      tx = 0; ty = 0; gx = 50; gy = 50;
      if (!raf) raf = requestAnimationFrame(apply);
    });
    function apply() {
      cx += (tx - cx) * 0.16;
      cy += (ty - cy) * 0.16;
      card.style.transform = `perspective(900px) rotateX(${cx}deg) rotateY(${cy}deg) translateY(${Math.abs(cx) + Math.abs(cy) > 0.05 ? -3 : 0}px)`;
      card.style.setProperty('--mx', gx + '%');
      card.style.setProperty('--my', gy + '%');
      raf = (Math.abs(tx - cx) + Math.abs(ty - cy) > 0.01) ? requestAnimationFrame(apply) : null;
    }
  });

  // ---------------------------------------------------------------
  // 6. Number counters (data-count attribute)
  // ---------------------------------------------------------------
  const counterIO = new IntersectionObserver((entries) => {
    for (const e of entries) {
      if (!e.isIntersecting) continue;
      counterIO.unobserve(e.target);
      const el = e.target;
      const target = parseFloat(el.dataset.count);
      const decimals = (el.dataset.count.split('.')[1] || '').length;
      const start = performance.now();
      const dur = 1400;
      function step(t) {
        const k = Math.min(1, (t - start) / dur);
        const eased = 1 - Math.pow(1 - k, 3);
        el.textContent = (target * eased).toFixed(decimals);
        if (k < 1) requestAnimationFrame(step);
        else el.textContent = el.dataset.suffix
          ? (target.toFixed(decimals) + el.dataset.suffix)
          : target.toFixed(decimals);
      }
      requestAnimationFrame(step);
    }
  }, { threshold: 0.5 });
  document.querySelectorAll('[data-count]').forEach((el) => counterIO.observe(el));

  // ---------------------------------------------------------------
  // 7. Live mesh visualization (animated SVG)
  //    5 peers in a circle, full mesh (10 edges). Every edge has a
  //    constant dashed flow so it always reads as "live". On top of
  //    that, multiple concurrent comet-style pulses travel along
  //    edges in round-robin order, so every link gets activity.
  // ---------------------------------------------------------------
  const meshSvg = document.getElementById('live-mesh');
  if (meshSvg) {
    const N = 5;
    const cx = 250, cy = 200, R = 130;
    const peers = Array.from({ length: N }, (_, i) => {
      const a = (i / N) * Math.PI * 2 - Math.PI / 2;
      return { id: i, x: cx + Math.cos(a) * R, y: cy + Math.sin(a) * R };
    });

    const NS = 'http://www.w3.org/2000/svg';
    const layer = (cls) => {
      const g = document.createElementNS(NS, 'g');
      g.setAttribute('class', cls);
      meshSvg.appendChild(g);
      return g;
    };
    const lineLayer = layer('mesh-lines');
    const pulseLayer = layer('mesh-pulses');
    const nodeLayer = layer('mesh-nodes');

    // Edges — each gets its own gradient aligned with the line so
    // every link reads identically regardless of orientation, plus
    // a constant dashed flow animation on top of the static base.
    const defs = meshSvg.querySelector('defs') || (() => {
      const d = document.createElementNS(NS, 'defs');
      meshSvg.insertBefore(d, meshSvg.firstChild);
      return d;
    })();

    const edges = [];
    let edgeIdx = 0;
    for (let i = 0; i < N; i++) {
      for (let j = i + 1; j < N; j++) {
        const a = peers[i], b = peers[j];

        // Per-edge gradient in user space so direction is consistent.
        const gid = `edgeGrad-${edgeIdx++}`;
        const g = document.createElementNS(NS, 'linearGradient');
        g.setAttribute('id', gid);
        g.setAttribute('gradientUnits', 'userSpaceOnUse');
        g.setAttribute('x1', a.x); g.setAttribute('y1', a.y);
        g.setAttribute('x2', b.x); g.setAttribute('y2', b.y);
        g.innerHTML =
          '<stop offset="0%" stop-color="#8b5cf6"/>' +
          '<stop offset="50%" stop-color="#6366f1"/>' +
          '<stop offset="100%" stop-color="#22d3ee"/>';
        defs.appendChild(g);

        // Static base line — always visible so the mesh reads as
        // "all connected" even between pulses.
        const base = document.createElementNS(NS, 'line');
        base.setAttribute('x1', a.x); base.setAttribute('y1', a.y);
        base.setAttribute('x2', b.x); base.setAttribute('y2', b.y);
        base.setAttribute('stroke', `url(#${gid})`);
        base.setAttribute('stroke-width', '1.1');
        base.setAttribute('opacity', '0.55');
        base.setAttribute('class', 'mesh-edge-base');
        lineLayer.appendChild(base);

        // Flow line — same path with dashed stroke that animates,
        // giving every edge a subtle constant data-flow look.
        const flow = document.createElementNS(NS, 'line');
        flow.setAttribute('x1', a.x); flow.setAttribute('y1', a.y);
        flow.setAttribute('x2', b.x); flow.setAttribute('y2', b.y);
        flow.setAttribute('stroke', '#22d3ee');
        flow.setAttribute('stroke-width', '1');
        flow.setAttribute('stroke-dasharray', '3 7');
        flow.setAttribute('opacity', '0.45');
        flow.setAttribute('class', 'mesh-edge-flow');
        flow.style.animationDelay = (edgeIdx * 0.18) + 's';
        // Half the edges flow in the opposite direction so traffic
        // looks bidirectional across the mesh.
        if ((i + j) % 2 === 0) flow.classList.add('reverse');
        lineLayer.appendChild(flow);

        edges.push({ a, b, base, flow });
      }
    }

    // Nodes
    peers.forEach((p, i) => {
      const halo = document.createElementNS(NS, 'circle');
      halo.setAttribute('cx', p.x); halo.setAttribute('cy', p.y);
      halo.setAttribute('r', 14);
      halo.setAttribute('fill', 'rgba(139,92,246,0.25)');
      halo.setAttribute('class', 'mesh-halo');
      halo.style.transformOrigin = `${p.x}px ${p.y}px`;
      halo.style.animation = `haloPulse 2.6s ease-in-out infinite ${i * 0.35}s`;
      nodeLayer.appendChild(halo);

      const ring = document.createElementNS(NS, 'circle');
      ring.setAttribute('cx', p.x); ring.setAttribute('cy', p.y);
      ring.setAttribute('r', 18);
      ring.setAttribute('fill', 'none');
      ring.setAttribute('stroke', 'url(#nodeRing)');
      ring.setAttribute('stroke-width', '1');
      ring.setAttribute('opacity', '0.6');
      nodeLayer.appendChild(ring);

      const dot = document.createElementNS(NS, 'circle');
      dot.setAttribute('cx', p.x); dot.setAttribute('cy', p.y);
      dot.setAttribute('r', 8);
      dot.setAttribute('fill', 'url(#nodeFill)');
      dot.setAttribute('class', 'mesh-node');
      dot.style.transformOrigin = `${p.x}px ${p.y}px`;
      dot.style.animation = `nodePulse 3s ease-in-out infinite ${i * 0.4}s`;
      nodeLayer.appendChild(dot);

      const lbl = document.createElementNS(NS, 'text');
      const off = 28;
      const ang = Math.atan2(p.y - cy, p.x - cx);
      lbl.setAttribute('x', p.x + Math.cos(ang) * off);
      lbl.setAttribute('y', p.y + Math.sin(ang) * off + 4);
      lbl.setAttribute('text-anchor', 'middle');
      lbl.setAttribute('fill', '#98a2b3');
      lbl.setAttribute('font-family', 'JetBrains Mono, monospace');
      lbl.setAttribute('font-size', '11');
      lbl.textContent = `peer-${i + 1}`;
      nodeLayer.appendChild(lbl);
    });

    // Round-robin pulse scheduler: shuffle, walk through every edge
    // once, then reshuffle. Guarantees every link is exercised on
    // each cycle instead of relying on chance.
    const order = edges.map((_, i) => i);
    let cursor = 0;
    function nextEdge() {
      if (cursor === 0) {
        for (let k = order.length - 1; k > 0; k--) {
          const r = Math.floor(Math.random() * (k + 1));
          [order[k], order[r]] = [order[r], order[k]];
        }
      }
      const e = edges[order[cursor]];
      cursor = (cursor + 1) % order.length;
      return e;
    }

    // Comet-style pulse: a head plus a fading trail of dots.
    function spawnPulse() {
      const e = nextEdge();
      const reverse = Math.random() > 0.5;
      const a = reverse ? e.b : e.a;
      const b = reverse ? e.a : e.b;

      const TRAIL = 5;
      const trail = [];
      for (let i = 0; i < TRAIL; i++) {
        const c = document.createElementNS(NS, 'circle');
        c.setAttribute('r', String(3 - i * 0.4));
        c.setAttribute('fill', '#22d3ee');
        c.setAttribute('opacity', String(0.95 * (1 - i / TRAIL)));
        c.style.filter = i === 0 ? 'drop-shadow(0 0 8px #22d3ee)' : 'none';
        pulseLayer.appendChild(c);
        trail.push(c);
      }

      const dur = 700 + Math.random() * 400;
      const start = performance.now();
      const dx = b.x - a.x, dy = b.y - a.y;
      const len = Math.hypot(dx, dy);
      const segLen = Math.min(22, len * 0.16);

      function step(t) {
        const k = Math.min(1, (t - start) / dur);
        for (let i = 0; i < TRAIL; i++) {
          const back = i * (segLen / len);
          const kk = Math.max(0, k - back * 0.9);
          trail[i].setAttribute('cx', a.x + dx * kk);
          trail[i].setAttribute('cy', a.y + dy * kk);
          trail[i].setAttribute('opacity', String(0.95 * (1 - i / TRAIL) * (1 - k * 0.3)));
        }
        if (k < 1) requestAnimationFrame(step);
        else trail.forEach((c) => c.remove());
      }
      requestAnimationFrame(step);

      // Boost the base line briefly so the active edge stands out.
      e.base.setAttribute('opacity', '1');
      e.base.setAttribute('stroke-width', '1.9');
      setTimeout(() => {
        e.base.setAttribute('opacity', '0.55');
        e.base.setAttribute('stroke-width', '1.1');
      }, dur);
    }

    // Multiple concurrent emitters so several edges are active at
    // once — keeps the mesh feeling alive across all 10 links.
    function emitter(period, jitter) {
      function tick() {
        if (!document.hidden) spawnPulse();
        setTimeout(tick, period + Math.random() * jitter);
      }
      setTimeout(tick, Math.random() * period);
    }
    emitter(420, 260);
    emitter(520, 320);
    emitter(680, 360);
  }

  // ---------------------------------------------------------------
  // 8. Tagline typewriter effect (h1 sub-line)
  // ---------------------------------------------------------------
  const phrases = [
    'WebRTC mesh.',
    "Server o'rtada yo'q.",
    "DTLS + secretbox.",
    "16 ta peer, to'liq mesh.",
    "Chat, fayl, port tunnel.",
  ];
  const tw = document.getElementById('typewriter');
  if (tw) {
    let pi = 0, ci = 0, deleting = false;
    function step() {
      const word = phrases[pi];
      if (!deleting) {
        ci++;
        tw.textContent = word.slice(0, ci);
        if (ci === word.length) { deleting = true; setTimeout(step, 1700); return; }
        setTimeout(step, 55 + Math.random() * 40);
      } else {
        ci--;
        tw.textContent = word.slice(0, ci);
        if (ci === 0) { deleting = false; pi = (pi + 1) % phrases.length; setTimeout(step, 250); return; }
        setTimeout(step, 25);
      }
    }
    setTimeout(step, 1100);
  }

  // ---------------------------------------------------------------
  // 9. Smooth-scroll for in-page anchors
  // ---------------------------------------------------------------
  document.querySelectorAll('a[href^="#"]').forEach((a) => {
    a.addEventListener('click', (e) => {
      const id = a.getAttribute('href').slice(1);
      if (!id) return;
      const target = document.getElementById(id);
      if (!target) return;
      e.preventDefault();
      target.scrollIntoView({ behavior: 'smooth', block: 'start' });
    });
  });

  // ---------------------------------------------------------------
  // 10. Header shadow on scroll
  // ---------------------------------------------------------------
  const nav = document.querySelector('nav.top');
  if (nav) {
    const onScroll = () => {
      if (window.scrollY > 4) nav.classList.add('scrolled');
      else nav.classList.remove('scrolled');
    };
    onScroll();
    window.addEventListener('scroll', onScroll, { passive: true });
  }

})();
