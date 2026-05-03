# Portal Cipher Protocol v1 (PCP-1) — Spetsifikatsiya / Specification

**Status:** Draft v1, 2026-05-01
**Authors:** Portal contributors
**License:** MIT (open for any project)
**Implements:** `client/crypt/pcp/`

---

## 1. Maqsad / Goal

PCP-1 — Portal'ning ilova qatlam shifrlash protokoli. WebRTC DTLS ostida
qo'shimcha himoya qatlami sifatida ishlaydi va quyidagilarni ta'minlaydi:

PCP-1 is Portal's application-layer cipher protocol. It runs as an
additional security layer beneath WebRTC DTLS and provides:

1. **Long-term identity** — har bir qurilma o'z Ed25519 keypair'iga ega.
   Foydalanuvchilar bir-birini sessiyalar bo'ylab tasdiqlay oladi.
   Each device has its own Ed25519 keypair. Users can verify each other
   across sessions.
2. **Per-portal forward secrecy** — har portal o'z ephemeral X25519
   kalitlariga ega. Kelajakda asosiy kalit qo'lga olinsa ham, o'tgan
   portal sessiyalari xavfsiz qoladi.
   Each portal has its own ephemeral X25519 keys. Even if long-term
   keys are later compromised, past portal sessions remain safe.
3. **Identity-bound portal codes** — portal kodi session pubkey hash
   bilan bog'langan. Brute-force attacker server javobisiz hech narsa
   hisoblay olmaydi.
   Portal codes embed a tag derived from the session pubkey. A
   brute-force attacker can't precompute valid codes without the
   server's response.
4. **Per-pair AEAD with fresh nonces** — har juftlik o'z yo'nalishli
   kalitlariga va monoton counter'iga ega. Nonce-reuse mumkin emas.
   Each pair has directional keys and a monotonic counter. Nonce reuse
   is impossible.

---

## 2. Cryptographic primitives

PCP-1 standart, audit qilingan primitivlardan foydalanadi. **Hech qachon
o'z kriptografiyasini yozmaymiz.**

PCP-1 uses standard, audited primitives. **We never roll our own crypto.**

| Primitive | Purpose | Library (Go) |
| --- | --- | --- |
| **Ed25519** | Long-term identity signatures | `crypto/ed25519` |
| **X25519** | Ephemeral Diffie-Hellman | `golang.org/x/crypto/curve25519` |
| **HKDF-SHA256** | Key derivation | `golang.org/x/crypto/hkdf` |
| **XChaCha20-Poly1305** | AEAD (24-byte nonce) | `golang.org/x/crypto/chacha20poly1305` |
| **SHA-256** | Hashing, fingerprints | `crypto/sha256` |
| **Argon2id** | Password-based KDF (legacy code derivation) | `golang.org/x/crypto/argon2` |

---

## 3. Identity layer

### 3.1 Keypair generation

Ilk ishga tushganda har bir qurilma quyidagini yaratadi:

On first launch, each device generates:

```
IdSK ← random 32 bytes (ed25519 seed)
IdPK ← Ed25519 PublicKey from IdSK
```

`IdSK` OS keychain'da saqlanadi (macOS: Keychain, Windows: DPAPI,
Linux: libsecret). Boshqa joyda saqlanmaydi.

`IdSK` is stored in the OS keychain (macOS: Keychain, Windows: DPAPI,
Linux: libsecret). Stored nowhere else.

### 3.2 Fingerprint

Foydalanuvchilar bir-birini tasdiqlash uchun:

For users to verify each other:

```
Fingerprint ← base32(SHA-256(IdPK))[:10]   # 10 chars, e.g. "K3F9-A2B1-X7"
```

Settings → Identity bo'limida ko'rsatiladi. Foydalanuvchilar yon
kanal orqali (chat, ovoz) almashishadi.

Shown in Settings → Identity. Users exchange via side channel (chat,
voice) for verification.

### 3.3 Signature

Hujjatlash uchun:

For documentation:

```
Sig ← Ed25519.Sign(IdSK, msg)
verify ← Ed25519.Verify(IdPK, msg, Sig)
```

---

## 4. Portal session layer

### 4.1 Owner — portal yaratish / creates a portal

```
1. eSK_o, ePK_o ← X25519 keypair (ephemeral, fresh per portal)

2. random_part ← 4 random alphanumeric chars (≈ 24 bits, 16M space)
   tag       ← base32(HKDF(ePK_o, "pcp1:code-tag")[:3])  → 4 chars
   Code      ← random_part + "-" + tag                   → 9 chars total

3. AuthSig ← Ed25519.Sign(IdSK_o,
                         "pcp1:portal:" + portal_id + ":" + ePK_o + ":" + ts)

4. Server ga yuboring / Send to server:
   {
     portal_id,
     code_hash:  SHA-256(Code),
     ePK_o,
     IdPK_o,
     AuthSig,
     ts
   }
```

Server kodning to'g'riligini saqlaydi (hash bilan), lekin pubkey'lar va
sig matnda saqlanadi (joiner ko'radi).

The server stores the code's correctness (as hash) but pubkeys and sig
are stored in plaintext (joiner sees them).

### 4.2 Joiner — portalga qo'shilish / joins a portal

```
1. User types: portal_id + Code

2. Joiner → Server: {portal_id, Code}

3. Server:
     - SHA-256(Code) bilan portal'ning code_hash'ni solishtiradi
       Compares SHA-256(Code) with stored code_hash
     - Mos kelsa qaytaradi: {ePK_o, IdPK_o, AuthSig, ts}
       If match, returns: {ePK_o, IdPK_o, AuthSig, ts}

4. Joiner verifies:
     a. AuthSig valid for "pcp1:portal:" + portal_id + ":" + ePK_o + ":" + ts
        signed by IdPK_o
     b. Code's tag part matches base32(HKDF(ePK_o, "pcp1:code-tag")[:3])
        → ensures the Code's structure is bound to the actual ePK_o
     c. ts within ±5 minutes of now (replay protection)

5. eSK_j, ePK_j ← X25519 keypair (ephemeral)

6. JoinSig ← Ed25519.Sign(IdSK_j,
                         "pcp1:join:" + portal_id + ":" + ePK_j + ":" + ts_j)

7. Joiner → Server → Owner (and other peers via signaling relay):
   {
     ePK_j,
     IdPK_j,
     JoinSig,
     ts_j
   }
```

### 4.3 Brute-force resistance

Eski 6-xonali kod: 1M space, 200k PBKDF2 iter ≈ 22 soat single-CPU.
Old 6-digit code: 1M space, 200k PBKDF2 iter ≈ 22 hours single-CPU.

Yangi PCP-1 kodi:
- Random part 24 bit ≈ 16M
- Tag part — server javobisiz tekshirib bo'lmaydi
- Server per-IP rate limit + per-portal failed-attempt counter

PCP-1 code:
- Random part 24 bits ≈ 16M
- Tag part — uncheckable without server response
- Server applies per-IP rate limit + per-portal failed-attempt counter

Brute-forcer har sinov uchun server'ga so'rov yuboradi — server-side
rate limit chegarasiga tushadi.

A brute-forcer must query the server for each guess — runs into
server-side rate limits.

---

## 5. Pair session layer

Ikkita peer ePK_remote va IdPK_remote ni bilgandan keyin (signaling
orqali):

After two peers know each other's ePK_remote and IdPK_remote (via
signaling):

```
shared    ← X25519(eSK_local, ePK_remote)
master    ← HKDF-Extract(salt=portal_id, ikm=shared)

# Lexicographic order on identity pubkeys for deterministic role
lo_pk, hi_pk ← lex-sorted (IdPK_local, IdPK_remote)

k_lo_to_hi ← HKDF-Expand(master, info="pcp1:pair:" + lo_pk + "->" + hi_pk, L=32)
k_hi_to_lo ← HKDF-Expand(master, info="pcp1:pair:" + hi_pk + "->" + lo_pk, L=32)

# Determine local direction
if IdPK_local == lo_pk:
  k_send ← k_lo_to_hi
  k_recv ← k_hi_to_lo
else:
  k_send ← k_hi_to_lo
  k_recv ← k_lo_to_hi

# Nonce prefix (16 bytes, deterministic per direction)
nonce_prefix_send ← HKDF-Expand(master, info="pcp1:nonce:" + lo_pk + "->" + hi_pk, L=16)
nonce_prefix_recv ← HKDF-Expand(master, info="pcp1:nonce:" + hi_pk + "->" + lo_pk, L=16)
```

Ikkala peer ham bir xil hisoblaydi (faqat send/recv yo'nalishi farqi).
Both peers compute the same (only direction differs).

---

## 6. Frame format

```
| counter (8 bytes BE) | tag (16 bytes Poly1305) | ciphertext... |

XChaCha20-Poly1305:
  Key   = k_send (or k_recv on receive)
  Nonce = nonce_prefix_send || counter (16 + 8 = 24 bytes)
  AD    = "pcp1:frame:v1"
```

Counter:
- Send tomoni: 0 dan boshlanib har frame'da +1
- Sender: starts at 0, +1 per frame
- Recv tomoni: kelgan counter > oxirgi qabul qilingan bo'lishi shart
  (replay protection)
- Receiver: incoming counter must be > last accepted (replay protection)

Counter overflow (2^64 frame'dan keyin) — amaliy emas; xavfsizlikni
kuchaytirish uchun **rekey 2^48 frame'dan keyin** majburiy.

Counter overflow (after 2^64 frames) — not practical; for safety,
**rekey is mandatory after 2^48 frames** (~280 trillion).

### 6.1 Rekey

```
master_new ← HKDF-Expand(master, info="pcp1:rekey:N", L=32)
# All directional keys recomputed from master_new
# Counter resets to 0
```

`N` = rekey hisoblagichi (1, 2, 3, ...) sessiya uchun.
`N` = rekey count (1, 2, 3, ...) for the session.

---

## 7. Sealed box (no DH)

DH'siz yopiq quti — ko'p qabul qiluvchili xabarlar, e'lonlar uchun.

Sealed box without DH — for multi-recipient messages, announcements.

```
ephSK, ephPK ← X25519 keypair (per-message ephemeral)
shared       ← X25519(ephSK, recipient_ePK)
key          ← HKDF-Expand(shared, info="pcp1:sealedbox", L=32)

nonce ← random 24 bytes
ct    ← XChaCha20-Poly1305.Seal(key, nonce, plaintext, AD=ephPK)

output = ephPK (32) || nonce (24) || ct
```

Recipient:
```
ephPK || nonce || ct = output
shared       ← X25519(recipient_eSK, ephPK)
key          ← HKDF-Expand(shared, info="pcp1:sealedbox", L=32)
plaintext    ← XChaCha20-Poly1305.Open(key, nonce, ct, AD=ephPK)
```

---

## 8. Wire format constants

| Field | Bytes | Encoding |
| --- | --- | --- |
| Ed25519 pubkey (`IdPK`) | 32 | raw |
| Ed25519 signature (`Sig`) | 64 | raw |
| X25519 pubkey (`ePK`) | 32 | raw |
| Counter | 8 | big-endian uint64 |
| Frame tag (Poly1305) | 16 | raw |
| Nonce | 24 | raw |
| Fingerprint | 10 | base32 chars |
| Code | 9 | `XXXX-YYYY` (alphanum) |

JSON serialization: ikkilik maydonlar base64 (RawStdEncoding, padding'siz).
JSON serialization: binary fields are base64 (RawStdEncoding, no padding).

---

## 9. Backwards compatibility

PCP-1 v0.4.0+ da default. Eski mijozlar uchun:

PCP-1 is default in v0.4.0+. For older clients:

- Server `protocol_version` field qaytaradi (va kutadi). Server returns
  (and expects) a `protocol_version` field.
- Joiner `pcp1` qo'llab-quvvatlamasa, server eski PBKDF2-secretbox
  rejimida ishlaydi (audit qilingan, ishlaydi). Sekin keyingi major
  versiya'da olib tashlanadi.
  If joiner doesn't support `pcp1`, server falls back to legacy
  PBKDF2-secretbox mode (audited, works). To be removed in next major
  version.

---

## 10. Hujjat versiyasi / Document version

| Version | Date | Changes |
| --- | --- | --- |
| 1.0-draft | 2026-05-01 | Initial draft |

Spetsifikatsiya o'zgarganda, "Document version" yangilanadi. Wire
o'zgarishlari major version bump'i (PCP-2).

When the spec changes, Document version is updated. Wire changes mean
a major version bump (PCP-2).

---

## 11. Threat model

Quyidagilardan himoya qiladi / Protects against:

- **Tarmoq kuzatuvchisi** / Network observer — DTLS+PCP-1 ikki qatlam
- **Qo'lga olingan signal serveri** / Compromised signaling — joiner verifies AuthSig before trust
- **Brute-force code** / Brute-force code — code bound to ePK; server rate limit
- **Replay** / Replay — monotonic counter + ts
- **Identity spoof** / Identity spoof — Ed25519 sign on every join

Quyidagilardan himoya QILMAYDI / Does NOT protect against:

- **Portalga qo'shilgan halol-ko'rinishli yomonniyatli peer** — ularda
  legitimate session keys bor.
  **Malicious-but-legitimate peer in portal** — they have legitimate
  session keys. (Trust model: don't share codes with strangers.)
- **Qurilmaning to'liq qo'lga olinishi** — `IdSK` keychain'da, lekin
  to'liq disk shifri yo'q.
  **Full device compromise** — `IdSK` is in keychain, but no full
  disk encryption guarantee.
- **Side-channel timing hujumlar PBKDF2/Argon2 ga** — qamrovga
  kirmaydi; mavjud rate limit bilan kompensatsiya.
  **Side-channel timing on PBKDF2/Argon2** — out of scope; mitigated
  by existing rate limit.

---

## 12. Audit history

| Date | Auditor | Result |
| --- | --- | --- |
| 2026-XX-XX | (nazariy review) | TBD |

PCP-1 spetsifikatsiyasini ochiq audit uchun e'lon qilamiz. Topilgan
har qanday muammo issue tracker'da `security` label'i bilan ochilsin
yoki `security/advisories/new` orqali xususiy bildirilsin.

PCP-1 spec is open for public audit. Any finding should be opened with
`security` label or reported privately via `security/advisories/new`.

---

## Ilovalar / References

- RFC 8439 — ChaCha20-Poly1305
- RFC 7748 — X25519
- RFC 8032 — Ed25519
- RFC 5869 — HKDF
- RFC 9106 — Argon2
- Curve25519: New Diffie-Hellman Speed Records (Bernstein, 2006)
- ChaCha, a variant of Salsa20 (Bernstein, 2008)
