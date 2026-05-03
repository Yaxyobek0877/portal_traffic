# Server-side o'zgarishlar — v0.4.0 dan keyin

Signal serveri xususiy repo'da. Quyidagi o'zgarishlar **server tomonida**
bajarilishi kerak (client tomon allaqachon tayyor).

The signaling server is in a private repo. The following changes need
to be made **server-side** (the client is already prepared for them).

---

## 1. 8-xonali kod opsiyasi (Task #7)

**Maqsad:** 6-xonali kod (1M space) + brute-force kuchaytirish.

**Muammo:** Audit hujjatida (`SECURITY-AUDIT.md` issue #8) yozilgan:
6-xonali kod 1M = botnet bilan soatlar ichida brute-force qilinadi.

**Yechim:** Portal yaratishda foydalanuvchi tanlaydi:
- **Standart (qisqa):** 6 xonali raqam (mavjud, eski client'lar bilan moslik)
- **Kuchli:** 8 xonali raqam (100M space) yoki PCP-1 format (`XXXX-YYYY`)

### Wire protokol o'zgarishlari

`portal.create` so'roviga yangi maydon (`shared/protocol/messages.go`):

```go
type PortalCreate struct {
    Type        string `json:"type"`        // "portal.create"
    Nickname    string `json:"nickname"`
    PublicNick  bool   `json:"public_nick"`
    Capacity    int    `json:"capacity"`
    CodeStrength string `json:"code_strength,omitempty"` // YANGI: "short" (default) | "strong"
}
```

`portal.created` javobida ham:

```go
type PortalCreated struct {
    Type       string `json:"type"`
    PortalID   string `json:"portal_id"`
    Code       string `json:"code"`         // 6 yoki 8 yoki "XXXX-YYYY"
    PeerID     string `json:"peer_id"`
    VirtualIP  string `json:"virtual_ip"`
    Capacity   int    `json:"capacity"`
    CodeFormat string `json:"code_format"`  // YANGI: "digit6" | "digit8" | "pcp1"
}
```

Eski client'lar `code_format` ni e'tiborsiz qoldiradilar. Kod uzunligi
har xil bo'lsa, eski client UI 6 belgi bilan kesib qo'yadi — bu
nomatlub xulq bo'lsa-da, boshqa bilan moslik xavfsiz.

Old clients ignore `code_format`. UI cap of 6 chars truncates a longer
code — undesirable but safe.

### Server o'zgarishlari (`server/portal.go`)

```go
// generateCode produces a 6-digit code by default; 8-digit if
// strength == "strong". PCP-1 mode (Phase 2) returns "XXXX-YYYY"
// derived from the portal's session pubkey.
func generateCode(strength string) string {
    switch strength {
    case "strong":
        return generateDigitCode(8)  // 100M space
    case "pcp1":
        return generatePCP1Code()    // see PCP-1 SPEC §4.1
    default:
        return generateDigitCode(6)  // legacy
    }
}
```

### Client tomon — TAYYOR

Client tomonda hech narsa qilish kerak emas: `welcome.code.placeholder`
hozirda 6-xonali, lekin input maxLength va validation 6+ ni qabul qiladi
(faqat raqamlar bilan cheklangan; 8 xonali ham ishlaydi).

PCP-1 alphanumeric kod uchun esa Welcome'da regex'ni `[0-9]` dan
`[A-Z0-9-]` ga o'zgartirish kerak — bu server tayyor bo'lsa kichik
o'zgarish.

---

## 2. Failed-join attack alert (Task #8)

**Maqsad:** Per-portal failed-attempt counter + owner'ga real-time
alert.

**Muammo:** Audit issue #9: noto'g'ri kodlar umumiy rate limit'da
hisoblanadi. Botnet bo'lib portal'ga hujum qilsa, owner hech narsani
bilmaydi. Owner'ni real-time xabardor qilish kerak.

### Wire protokol o'zgarishlari

Yangi server → owner xabari (`shared/protocol/messages.go`):

```go
// PortalAttackAlert — sent to a portal owner when failed-join attempts
// exceed a threshold. Lets the owner choose to lock the portal or
// rotate the code.
type PortalAttackAlert struct {
    Type           string `json:"type"`            // "portal.attack_alert"
    PortalID       string `json:"portal_id"`
    AttemptsTotal  int    `json:"attempts_total"`  // since portal creation
    AttemptsRecent int    `json:"attempts_recent"` // last 60 seconds
    DistinctIPs    int    `json:"distinct_ips"`    // distinct source IPs
    Suggestion     string `json:"suggestion"`      // "lock" | "rotate_code"
}
```

### Server o'zgarishlari (`server/portal.go`)

```go
type Portal struct {
    // ... existing fields ...

    failedAttempts struct {
        mu       sync.Mutex
        total    int
        recent   *ringBuffer  // timestamps of last 100 attempts
        ipsSeen  map[string]struct{}
        lastAlert time.Time
    }
}

func (p *Portal) recordFailedJoin(ip string) {
    p.failedAttempts.mu.Lock()
    defer p.failedAttempts.mu.Unlock()

    p.failedAttempts.total++
    p.failedAttempts.recent.push(time.Now())
    p.failedAttempts.ipsSeen[ip] = struct{}{}

    // Threshold: 5+ recent (60s window) attempts from 3+ IPs.
    if p.recentCount(60*time.Second) >= 5 && len(p.failedAttempts.ipsSeen) >= 3 {
        // Throttle alerts to one per 5 minutes.
        if time.Since(p.failedAttempts.lastAlert) > 5*time.Minute {
            p.failedAttempts.lastAlert = time.Now()
            p.notifyOwnerOfAttack()
        }
    }
}
```

### Client tomon

Yangi `subscribe` event va banner. UI:

```typescript
// src/App.tsx
offs.push(
  subscribe<PortalAttackAlert>("portal.attack_alert", (a) => {
    setBanner(
      `⚠️ Hujum aniqlandi: oxirgi 60s'da ${a.attemptsRecent} ta urinish ` +
      `(${a.distinctIPs} ta IP'dan). Portal'ni qulflashni o'ylab ko'ring.`
    );
  })
);
```

Translation kalitlari `i18n/strings.ts`'ga qo'shilishi kerak (hozir
yo'q — server tayyor bo'lganda qo'shaman).

---

## 3. PCP-1 protokol versiyasi (kelgusi v0.5.0)

PCP-1 hozir client tomonda paket sifatida tayyor (`client/crypt/pcp/`).
Lekin mesh.Manager hali ham eski PBKDF2-secretbox ishlatadi. PCP-1'ga
o'tish:

1. Server'da `protocol_version` field handshake'da bildirish
2. Mijoz `pcp1` qo'llab-quvvatlasa, X25519 ephemeral pubkey'ni
   `portal.create` ga qo'shadi
3. Joiner'lar bunga mos (server orqali) o'tib AuthSig + JoinSig
   tekshiradi
4. PairSession (`crypt/pcp/session.go`) chat / proxy / transfer
   data channel'larida ishga tushadi

Bu **breaking change emas**: eski client'lar eski mode'da ishlay
oladilar.

---

## Vaqt rejimi / Timeline

| Task | Bosqich | Qachon |
| --- | --- | --- |
| #7 8-xonali kod | Server change kerak | v0.4.1 |
| #8 Attack alert | Server change kerak | v0.4.1 |
| PCP-1 mesh integration | Server + client coord | v0.5.0 |
| Mobile PCP-1 | Mobile change | v0.5.0 |

---

## Mavjud tayyor narsalar (client tomon)

✅ Client side:
- `client/crypt/pcp/` — to'liq paket, 33 ta test
- Updater: `client/updater/` — GitHub Releases polling, banner
- Crash reporting: `client/crashreport/` — local file capture
- i18n: `client/frontend/src/i18n/` — uz/en
- Deeplink: `client/frontend/src/lib/deeplink.ts` — `portal://` URL
