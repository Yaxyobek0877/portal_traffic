# Manbadan build qilish

Desktop dastur foydalanuvchilar uchun hali paketlanmagan
([ROADMAP.md](ROADMAP.md) dagi 2-bosqich). Loyihani bugun build qilmoqchi
bo'lsangiz, sizga Go toolchain va o'z OS ingizda
[Wails](https://wails.io) uchun kerakli platforma talablari kerak
bo'ladi.

> 2-bosqich tugagandan keyin macOS, Windows va Linux uchun tayyor
> installer lar Releases sahifasida paydo bo'ladi va bu fayl
> "kontributorlar uchun" qo'llanmaga aylanadi.

---

## 1. Talab qilinadigan narsalar

### Hamma platformalar

- **Go 1.22+** — `https://go.dev/dl/`
- **Git**

### macOS

```sh
xcode-select --install
brew install node          # React frontend uchun (2-bosqich)
```

### Linux (Debian / Ubuntu)

```sh
sudo apt update
sudo apt install -y \
    build-essential pkg-config \
    libgtk-3-dev libwebkit2gtk-4.1-dev \
    nodejs npm
```

Fedora da:

```sh
sudo dnf install -y gcc-c++ pkg-config gtk3-devel webkit2gtk4.1-devel nodejs npm
```

### Windows

- Rasmiy MSI installerdan **Go** ni o'rnating.
- [WebView2 Runtime](https://developer.microsoft.com/en-us/microsoft-edge/webview2/)
  ni o'rnating (Windows 11 da o'rnatilgan).
- [Node.js LTS](https://nodejs.org/) ni o'rnating.
- [Git for Windows](https://git-scm.com/download/win) ni o'rnating.

---

## 2. Klonlash

```sh
git clone https://github.com/<sizning-username>/portal_traffic.git
cd portal_traffic
```

---

## 3. Bugun mavjud bo'lgan qismlarni build qilish

### Shared protocol paketi

```sh
cd shared
go test ./...
```

Hammasi shu — `shared/` da binary yo'q. U client tomonidan import qilinadi.

### Desktop dastur (2-bosqich — yo'lda)

`client/` paydo bo'lganda build shunday ko'rinishda bo'ladi. Repo
ildizidan:

```sh
# Wails CLI ni bir marta o'rnatish
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Development rejimi (live reload)
cd client
wails dev

# Joriy OS uchun production build
wails build -clean
# → client/build/bin/Portal[.app|.exe|] hosil qiladi
```

Cross-platform build:

```sh
wails build -platform darwin/universal     # macOS .app
wails build -platform windows/amd64        # Windows .exe
wails build -platform linux/amd64          # Linux binary
```

Hosil bo'lgan paketlar mustaqil — foydalanuvchining mashinasida Go yoki
Node talab qilinmaydi.

---

## 4. Signal URL ni sozlash

Portal client lar siz sozlagan signal serveriga ulanadi. Variantlar:

1. **Loyihaning hosting qilingan signal endpoint i** — yetkazib
   beriladigan binary da standart. Sozlash kerak emas.
2. **O'zingizniki ishga tushirish.** Simli protokol [PROTOCOL.md](PROTOCOL.md)
   da hujjatlashtirilgan; mos server bir necha yuz qator Go kodi.
   Ishga tushgandan keyin client ni Settings → Network → Signaling URL
   orqali yo'naltiring yoki env var ni sozlang:
   ```sh
   export PORTAL_SIGNALING_URL=wss://sizning-host/ws
   ```

---

## 5. Build ni tekshirish

Client ni build qilgandan keyin (2-bosqich+):

```sh
./build/bin/Portal      # macOS da Portal.app ni oching
```

Wormhole logo, taxallus inputi va **Portal yaratish / Portalga
qo'shilish** tugmalari bilan welcome ekrani ko'rinishi kerak.
Loyihaning signal URL i sozlangan bo'lsa, **Portal yaratish** ni bosing
— bir-ikki soniyada 6 xonali ID va kod chiqishi kerak.

ID + kodni Portal ishlayotgan ikkinchi qurilmaga bering, **Portalga
qo'shilish** ni bosing va mesh diagrammasi yorishishini kuzating.

---

## Muammolarni hal qilish

| Belgi | Ehtimoliy sabab | Yechim |
| --- | --- | --- |
| `wails: command not found` | Wails CLI o'rnatilmagan | `go install github.com/wailsapp/wails/v2/cmd/wails@latest`; `$GOPATH/bin` ni `PATH` da bo'lishini ta'minlang |
| "Connecting…" da qotib qoladi | Signal URL noto'g'ri yoki yetib bo'lmaydigan | `wss://` ekanini, sertifikat eskirmaganini tekshiring |
| Peer >30s davomida 🔴 (failed) | NAT ni o'tib bo'lmadi; TURN sozlanmagan | [ARCHITECTURE.md § NAT va TURN zaxiraga o'tish](ARCHITECTURE.md#nat-va-turn-zaxiraga-otish) ga qarang |
| Linux: `Package webkit2gtk-4.1 not found` | Eskiroq distroda `webkit2gtk-4.0` bor | `libwebkit2gtk-4.0-dev` ni sinab ko'ring |
