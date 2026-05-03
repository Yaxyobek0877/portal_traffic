# O'rnatish

Eng oson yo'l — [Releases sahifasidan](https://github.com/Yaxyobek0877/portal_traffic/releases)
o'z OS'ingizga tayyor binarni yuklab oling (macOS, Windows, Linux).
Bu sahifa **manbadan build** qilishni hohlovchi developer'lar uchun.

> ⚠️ v0.4.0 binarlari imzolanmagan. macOS'da right-click → Open;
> Windows'da SmartScreen → "Run anyway" kerak bo'ladi. Code signing
> [keyingi reliz'da](docs/CODE_SIGNING.md).

## Manbadan build qilish

Tanlang:

- [Ubuntu / Debian (Linux)](#ubuntu--debian) — eng ko'p so'raladigan platforma
- [macOS](#macos)
- [Windows](#windows)

> **Eslatma:** quyidagi qadamlarni xat-by-xat bajarsangiz hech narsa
> qoldirmaydi. Bir buyruq xato berib qolsa, oldingi qadamga qayting va
> chiqgan xatoni nusxa olib menga yuboring.

---

## Ubuntu / Debian

Ubuntu 22.04+ yoki Debian 12+ uchun. Boshidan oxirigacha bajaring:

### 1) Tizim paketlarini o'rnating

```bash
sudo apt update
sudo apt install -y \
    build-essential pkg-config git curl wget \
    libgtk-3-dev libwebkit2gtk-4.1-dev \
    nodejs npm
```

> Eski Ubuntu (20.04) da `libwebkit2gtk-4.0-dev` ishlatilgan; agar
> `libwebkit2gtk-4.1-dev` topilmasa, uni sinab ko'ring.

Tekshirish:

```bash
node --version    # v18+ bo'lishi kerak
npm --version     # 8+
gcc --version     # mavjud
```

### 2) Go ni o'rnating (1.22+)

Ubuntu repolaridagi Go versiyasi ko'pincha eski. Rasmiy tarball
afzal:

```bash
cd /tmp
wget https://go.dev/dl/go1.22.5.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.22.5.linux-amd64.tar.gz

# PATH ga qo'shish (bashrc uchun; zsh ishlatsangiz ~/.zshrc da bajaring)
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
echo 'export PATH=$PATH:$HOME/go/bin' >> ~/.bashrc
source ~/.bashrc

go version    # "go version go1.22.5 linux/amd64" chiqishi kerak
```

### 3) Wails CLI ni o'rnating

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
wails doctor
```

`wails doctor` chiqishida hamma satrlar yashil ✓ bo'lishi kerak.
Agar bir qaysi paket "Not installed" deb chiqsa, o'sha paketni `apt
install` orqali qo'shing va qaytadan tekshiring.

### 4) Repo ni klonlang va build qiling

```bash
cd ~
git clone https://github.com/Yaxyobek0877/portal_traffic.git
cd portal_traffic/client

# Frontend bog'liqliklari (~30s)
cd frontend
npm install
cd ..

# To'liq build (Wails frontend ni avtomatik build qiladi va
# bitta self-contained binary hosil qiladi)
wails build
```

Build muvaffaqiyatli tugagandan keyin binary `build/bin/Portal`
yo'lida bo'ladi.

### 5) Ishga tushirish

```bash
./build/bin/Portal
```

Birinchi ishga tushganda welcome ekrani chiqishi kerak: wormhole
logo, taxallus inputi, Yaratish/Qo'shilish tugmalari. Ulanish uchun
default endpoint `wss://signaling.1pro.uz/ws` ishlatiladi.

### Ubuntu da uchragan tipik xatolar

| Xato | Yechim |
| --- | --- |
| `Package webkit2gtk-4.1 not found` | `sudo apt install libwebkit2gtk-4.0-dev` ni sinab ko'ring (eski Ubuntu) |
| `wails: command not found` | `~/go/bin` `PATH` da emas — yuqoridagi 2-qadamning oxirgi qatorini bajaring |
| `go: go.mod requires go >= 1.22` | Go versiyasi eski. 2-qadamni qaytadan, lekin ko'rsatilgan tarball ni ishlating |
| Welcome ekrani chiqmaydi (oyna qora) | `~/portal_traffic/client/frontend && npm run build` ni qayta urinib, keyin `wails build` ni qayta bajaring |
| `Could not find webkit2gtk` runtime'da | Build vaqtidagi va run vaqtidagi versiyalar mos kelmagan; `apt install libwebkit2gtk-4.1-0` bilan runtime kutubxonasini ham qo'shing |

---

## macOS

macOS 12+ uchun:

### 1) Xcode Command Line Tools

```bash
xcode-select --install
```

### 2) Go (1.22+)

```bash
brew install go    # Homebrew o'rnatilgan bo'lsa
# yoki rasmiy installer: https://go.dev/dl/

go version
```

### 3) Node.js (LTS)

```bash
brew install node
node --version
```

### 4) Wails CLI

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
wails doctor
```

### 5) Build

```bash
git clone https://github.com/Yaxyobek0877/portal_traffic.git
cd portal_traffic/client
wails build
open build/bin/Portal.app
```

`Portal.app` — odatdagi macOS dasturi. `Applications/` papkasiga
ko'chirib qo'ysangiz Spotlight orqali ham ochiladi.

---

## Windows

Windows 10 21H2+ yoki Windows 11 uchun:

### 1) Talab qilinadigan dasturlar

- **Go 1.22+**: https://go.dev/dl/ — MSI installer
- **Node.js LTS**: https://nodejs.org/ — MSI installer
- **Git for Windows**: https://git-scm.com/download/win
- **WebView2 Runtime**: Windows 11 da kelgan; Win 10 uchun
  https://developer.microsoft.com/microsoft-edge/webview2/

### 2) PowerShell ochib

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@latest
wails doctor

git clone https://github.com/Yaxyobek0877/portal_traffic.git
cd portal_traffic\client
wails build
.\build\bin\Portal.exe
```

---

## Konfiguratsiya

Build dan keyin:

- **Standart signal endpoint:** `wss://signaling.1pro.uz/ws` — siz hech narsa
  qilmasangiz dasturning ichida shu sozlangan.
- **O'zingizniki ishlatish uchun** dastur ichida sozlamalar (⚙) →
  Tarmoq → Signal serveri URL ni o'zgartiring va Saqlash.
- Yoki environment variable orqali:
  ```bash
  PORTAL_SIGNALING_URL=wss://your-host/ws ./build/bin/Portal
  ```

Sizning shaxsiy serveringizni qanday ishga tushirish — ARCHITECTURE.md
va PROTOCOL.md ga qarang. Mos signal server bir necha yuz qator Go
kodi.

---

## Build qilingan dasturni boshqa qurilmalarga ko'chirish

`build/bin/Portal` (Linux/macOS) yoki `Portal.exe` (Windows) — bu
**self-contained binary**. U bilan birga frontend ham, backend ham
yagona faylga paketlangan.

**Linux (AppImage shaklida emas, oddiy executable):**

```bash
scp build/bin/Portal user@server:/opt/portal/
ssh user@server
chmod +x /opt/portal/Portal
/opt/portal/Portal
```

Lekin desktop ilova GUI talab qiladi. Linux serveringizda GUI yo'q
bo'lsa, faqat CLI test harness ishlatiladi:

```bash
go build ./cmd/portal-cli -o /tmp/portal-cli
/tmp/portal-cli -mode create -nick alice
```

**macOS (boshqa Mac ga):**

`Portal.app` papkasini AirDrop yoki USB orqali bering. Birinchi
ochilganda Gatekeeper "tasdiqlanmagan dastur" deb to'sishi mumkin —
Sozlamalar → Privacy & Security → Open Anyway.

---

## Muammolarni hal qilish (umumiy)

| Belgi | Sabab | Yechim |
| --- | --- | --- |
| Welcome ekrani chiqmaydi (qora oyna) | Frontend assetlari yuklanmagan | `cd frontend && npm install && npm run build`, keyin `wails build -clean` |
| "Connecting…" da qotib qoladi | Signal URL noto'g'ri yoki yetib bo'lmaydigan | Sozlamalardan URL ni tekshiring; `wss://` (`ws://` emas) bo'lishi kerak |
| Peer >30s davomida 🔴 (failed) | NAT ni o'tib bo'lmadi; TURN sozlanmagan | `Settings → Diagnostika`, agar Simmetrik NAT chiqsa TURN serveringiz kerak (coturn) |
| `wails build` da "no such file" | Repo to'liq klon qilinmagan | `git status` qilib branch'ni `main` ga sozlang |
| Eski Ubuntu da paket topilmaydi | webkit2gtk-4.1 hali yo'q | `libwebkit2gtk-4.0-dev` ga o'ting (4.1 wails 2.10+ uchun, 4.0 oldingi versiyalar uchun) |

Boshqa muammoga duch kelsangiz: GitHub'da **Issue** oching va xato
matnini, OS versiyasini, Go va Wails versiyalarini birga jo'nating.
