# Hissa qo'shish

Yordam berishni xohlaganingiz uchun rahmat. Bu hozircha kichik loyiha —
foydali bo'lishning eng tezkor yo'li aniq issue ochish yoki maqsadli
patch yuborish.

## Issue ochish

Foydali issue tarkibida quyidagilar bo'lishi kerak:

- **Nima qildingiz** (buyruqlar, screenshot, tugmalar matni)
- **Nima yuz berdi** (xato xabari, log qatori, buzilgan xatti-harakat)
- **Nimani kutgandingiz** (bir gap yetarli)
- **Atrof-muhit** (OS, Portal versiyasi, standart bo'lmasa signal URL)

Agar issue da log fayli bo'lsa, joylashtirishdan oldin `peer_id` UUID
larini va o'zingizning `virtual_ip` ingizni o'chirib qo'ying — ular
o'z-o'zicha zararsiz, lekin boshqa odamlarning loglari bilan o'zaro
bog'lanishi mumkin.

---

## Patch yuborish

1. Fork qiling va `main` dan branch oching.
2. O'zgartirish kiriting; commit tarixi maqsadli bo'lsin. Biz squash-merge
   qilamiz, shuning uchun yakuniy commit xabari muhim; oraliq commit lar
   tartibsiz bo'lishi mumkin.
3. O'zgartirgan modul uchun testlarni ishga tushiring:
   ```sh
   cd shared && go test ./...
   ```
4. `go fmt ./...` va `go vet ./...` ni ishga tushiring. CI tozaligini
   talab qiladi.
5. Pull request oching. Tavsifda *nega* o'zgartirish kerak ekanini
   tushuntiring, faqat *nima* qilganingizni emas.

---

## Stil bo'yicha eslatmalar

### Go

- Standart `gofmt` stili. `go vet` dan boshqa custom linter yo'q.
- I/O bilan ishlaydigan har qanday funksiyada birinchi parametr sifatida
  `context.Context` qabul qilishni afzal ko'ring.
- Xatolar yuqoriga oqadi. Kutubxona funksiyasidan `log.Fatal` qilmang.
- `init()` va `main()` dan tashqarida panic dan saqlaning. Xato
  qaytaring.
- Izohlar *nima* emas, *nega* ni tushuntiradi. Kod o'zi nima qilayotganini
  ko'rsatadi.

### TypeScript / React (`client/` paydo bo'lganda)

- Stil uchun Tailwind. CSS-in-JS yoki styled-components yo'q.
- State uchun Zustand, Redux emas.
- Hook lar bilan funksional komponent. Class komponent yo'q.
- `any` ni ishlatishdan saqlaning. Agar haqiqatan kerak bo'lsa, sababini
  bir qator izohda qoldiring.

---

## Orqaga moslashuvchanlik

[`shared/protocol/messages.go`](shared/protocol/messages.go) dagi simli
protokol — bu ommaviy majburiyat. Ixtiyoriy maydonlar va yangi xabar
turlarini **qo'shish** har doim xavfsiz; maydonlarni **o'zgartirish**
yoki **olib tashlash** ishlatilayotgan har bir client va serverni
buzadi. Agar shunday qilish kerak deb o'ylasangiz, avval issue oching va
migratsiya rejasini muhokama qiling.

---

## Hozir nima eng foydali

[ROADMAP.md](ROADMAP.md) ga qarang. 2-bosqich faol ish maydoni —
xususan `client/peer/` paketi (WebRTC mesh ulashlar) va
`client/signaling/` client (WebSocket → handshake orkestratsiya).
3+ bosqichda React tomonida dizayn yordami kerak bo'ladi.

---

## Qamrovga kirmaydigan narsalar

- Kriptovalyuta yoki token asosidagi discovery
- Majburiy hisob qaydnomasi tizimi / markazlashtirilgan identifikatsiya
- Yopiq chat platformalari bilan integratsiya
- Signal serverining xabarlar tarkibini ko'rishini talab qiladigan har
  qanday narsa

Portal ning butun maqsadi — operator sizning trafigingizni o'qiy
olmasligi. Bunga zid funksiyalar qo'shilmaydi.

---

## Aloqa

Hozircha: faqat GitHub issue lar. Loyiha o'sib borsa, matrix xonasi
yoki shunga o'xshash narsa o'rnatamiz.
