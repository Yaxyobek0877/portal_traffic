# Portal — Release notes

[CHANGELOG.md](https://github.com/Yaxyobek0877/portal_traffic/blob/main/CHANGELOG.md)
da batafsil. To'liq o'zgarishlar ro'yxati uchun shu yerga qarang.

See [CHANGELOG.md](https://github.com/Yaxyobek0877/portal_traffic/blob/main/CHANGELOG.md)
for the full list of changes.

---

## Yuklab olish / Downloads

Pastdagi linklar har doim **eng yangi** relizga olib boradi —
yangi versiya chiqsa avtomatik yangilanadi. The links below always
point at the **latest** release.

| Platform | Yuklab olish / Download |
| --- | --- |
| **macOS** (Apple Silicon — M1/M2/M3/M4) | [Portal-darwin-arm64.zip](https://github.com/Yaxyobek0877/portal_traffic/releases/latest/download/Portal-darwin-arm64.zip) |
| **macOS** (Intel) | [Portal-darwin-amd64.zip](https://github.com/Yaxyobek0877/portal_traffic/releases/latest/download/Portal-darwin-amd64.zip) |
| **Windows** (10/11, 64-bit) | [Portal-windows-amd64.zip](https://github.com/Yaxyobek0877/portal_traffic/releases/latest/download/Portal-windows-amd64.zip) |
| **Linux** (Ubuntu 22.04+, Debian 12+) | [Portal-linux-amd64.tar.gz](https://github.com/Yaxyobek0877/portal_traffic/releases/latest/download/Portal-linux-amd64.tar.gz) |

Mobile (Android) — beta, alohida release. / Mobile is in beta, separate
release.

---

## O'rnatish / Installation

### macOS

1. `.zip` faylni yuklab oling va oching → `Portal.app` chiqadi
2. `Portal.app` ni `Applications/` ga ko'chiring
3. **Birinchi marta ochish:**
   - Macda imzolanmagan dasturlar Gatekeeper tomonidan to'siladi
   - Right-click → **Open** → ko'rsatma chiqsa **Open** ni bosing
   - Yoki: System Settings → Privacy & Security → "Open Anyway"

### Windows

1. `.zip` faylni yuklab oling va oching
2. `Portal.exe` ni `C:\Program Files\Portal\` ga ko'chiring
3. **Birinchi marta ochish:**
   - SmartScreen "Reputation Unknown" deb ogohlantiradi
   - **More info** → **Run anyway**

### Linux

```bash
tar -xzf Portal-<tag>-linux-amd64.tar.gz
chmod +x Portal
./Portal
```

WebKit/GTK runtime kerak — ko'p distrolarda mavjud:
WebKit/GTK runtime needed — present on most distros:

```bash
# Ubuntu/Debian:
sudo apt install libwebkit2gtk-4.1-0 libgtk-3-0
```

---

## Birinchi qadamlar / Quick start

1. **Yangi portal yarating:** "Portal yaratish" ni bosing →
   ID + 6-xonali kodni do'stingizga yuboring
2. **Yoki qo'shiling:** "Portalga qo'shilish" → ID va kodni kiriting
3. Ulanish 5-30 sekund davom etadi (NAT bo'yicha turlicha)
4. Mesh hosil bo'lgach: chat, fayl uzatish, port ulashish

Symmetric NAT ortida bo'lsangiz: Settings → TURN'da Cloudflare yoki
o'zingizning TURN serveringizni sozlang. Batafsil: [NAT-VA-TURN.md](https://github.com/Yaxyobek0877/portal_traffic/blob/main/NAT-VA-TURN.md).

---

## Xavfsizlik / Security

Bu reliz [PCP-1 (Portal Cipher Protocol v1)](https://github.com/Yaxyobek0877/portal_traffic/blob/main/client/crypt/pcp/SPEC.md)
ni qo'llaydi: WebRTC DTLS + PCP-1 ikki qatlam shifrlash.

Xavfsizlik zaifligi topdingizmi? Iltimos ommaviy issue ochmang —
[xususiy security advisory](https://github.com/Yaxyobek0877/portal_traffic/security/advisories/new)
oching.

Found a security vulnerability? Please don't open a public issue —
file a [private security advisory](https://github.com/Yaxyobek0877/portal_traffic/security/advisories/new).

---

## Litsenziya / License

[MIT](https://github.com/Yaxyobek0877/portal_traffic/blob/main/LICENSE)
