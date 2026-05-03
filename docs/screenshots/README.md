# Screenshots — qo'llanma / Guide

Bu papka Portal'ning marketing va hujjatlash skrinshotlarini saqlaydi.
Asosiy README.md va web landing'da ishlatiladi.

This folder holds Portal's marketing and documentation screenshots.
Used by the main README.md and the web landing.

---

## Kerak bo'lgan rasmlar / Required images

Hammasi **PNG**, retina sifati (2x DPI). Wide format (16:10) afzal —
README jadvalda yaxshi ko'rinadi.

All as **PNG**, retina (2x DPI). Wide format (16:10) preferred — looks
better in the README's table.

| Fayl / File | Ko'rsatadi / Shows | Tavsiya / Recommendation |
| --- | --- | --- |
| `welcome.png` | Welcome ekrani | Logo + nickname inputi + "Portal yaratish" / "Qo'shilish" tugmalari ko'rinadi. NAT ogohlantirishi yo'q (toza interfeys). |
| `portal-mesh.png` | Faol portal | 3-4 peer mesh diagrammasi animatsiya pulse'da, peer sidebar nick'lar bilan, header ID + KOD ko'rsatilgan |
| `chat-services.png` | Chat + servislar paneli | Bir nechta xabar (turli peer'lardan), pastga drag-drop fayl progress bar, o'ng tomondan Services panelda Minecraft expose qilingan |
| `qr-modal.png` | QR kod modal | QR kod, ID + KOD katta shrift, "yopish" tugmasi |
| `settings-network.png` | Settings → Network | Signal URL, TURN config, Cloudflare TURN, "Test TURN" tugmasi |
| `settings-about.png` | Settings → About | Versiya, "Yangi versiya mavjud" panel (mock 0.4.1), crash reports bo'sh, hujjatlar tugmalari |
| `nat-warning.png` | Welcome'da NAT bannerli | Symmetric NAT ogohlantirishli holat |

---

## Demo GIF / Screencast

Asosiy: **`demo.gif`** — 8-15 soniyalik animatsiya:

1. Welcome ekrani → "Portal yaratish" bosish
2. ID + KOD chiqadi
3. Boshqa oynada (split screen) "Qo'shilish" → ID + KOD kiritish
4. Mesh hosil bo'ladi, animatsiyali edge'lar
5. Chat'ga "salom" yozish

Tavsiyalar / Recommendations:
- 1280×800, 30 FPS, max 8 MB (GitHub README cap)
- `gifski`, `peek`, yoki `kap` (macOS) ishlating
- Ovoz yo'q
- Cursor ko'rinmaydi yoki kichik

```bash
# macOS — Kap'da ekran yozib oling, keyin GIF'ga aylantirish:
brew install gifski
ffmpeg -i screen-recording.mov -r 24 -vf "scale=1280:-1" frame_%03d.png
gifski --quality 85 --width 1280 -o demo.gif frame_*.png
```

---

## Hozirgi holat / Current status

- [ ] welcome.png
- [ ] portal-mesh.png
- [ ] chat-services.png
- [ ] qr-modal.png
- [ ] settings-network.png
- [ ] settings-about.png
- [ ] nat-warning.png
- [ ] demo.gif

Rasmlar qo'shilganda, README.md'da "_coming soon_" jadvalini yangilang
([README.md#screenshots](../../README.md#screenshots)).

When images are added, update the "_coming soon_" table in README.md.

---

## Marketing materiallar / Marketing assets (kelajak)

`docs/screenshots/og/` — Open Graph (Twitter, social share) uchun:
- `og-card.png` — 1200×630, brand colors gradient + logo + tagline
- `app-icon.png` — 1024×1024, app icon (Wails build'da ishlatiladi ham)

For social previews and app store listings.
