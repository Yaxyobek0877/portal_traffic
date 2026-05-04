# Server-side: log upload endpoint

Mijoz tomon (`client/logsink/`) **tayyor** — har 30 sekundda mahalliy
log faylga qo'shilgan yangi qatorlarni gzip qilib serveringizga POST
qilib turadi. Mijoz default'da yoqilgan; foydalanuvchi `Settings →
Diagnostika` → "Logni serverga yuborish" toggle bilan o'chirishi mumkin
(`storage.KeyLogUpload="0"`).

Qo'shish kerak: signaling serveringizda (`signaling.1pro.uz`) bitta
HTTP handler.

## Wire format

Mijoz har upload tsikldida shu HTTP so'rovni jo'natadi:

```
POST /logs/upload HTTP/1.1
Host: signaling.1pro.uz
Content-Type: text/plain
Content-Encoding: gzip
X-Client-ID: 8f4a1b2c3d4e5f607182930a4b5c6d7e
X-Portal-Version: 0.4.0

<gzipped log lines, one per newline>
```

`X-Client-ID` — har Portal o'rnatishi uchun **bir martalik tasodifiy
32-hex UUID** (mijoz `~/.portal/client-id`'da saqlaydi). Anonim;
foydalanuvchining ismi/IP'si emas. Server shu kalit bilan log'larni
guruhlaydi — bir foydalanuvchining sessiyalari bir papkaga tushadi.

## Go handler

`server/loghandler.go` (yoki shu papkada) qo'shing:

```go
package main

import (
	"compress/gzip"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"
)

// LogUploadDir is where /logs/upload writes incoming bundles. One
// subdir per X-Client-ID, one file per UTC-day. Default is overridable
// via PORTAL_LOG_DIR env var.
var LogUploadDir = func() string {
	if v := os.Getenv("PORTAL_LOG_DIR"); v != "" {
		return v
	}
	return "/var/log/portal/clients"
}()

// X-Client-ID must be 32 hex chars to be accepted (matches the format
// the desktop client mints). Filters out drive-by garbage requests.
var clientIDRE = regexp.MustCompile(`^[a-f0-9]{32}$`)

// 5 MiB per request — plenty for a 30s window. Reject anything bigger
// to prevent abuse.
const maxLogPayload = 5 << 20

// Per-client serialisation: writes from the same client land in
// the same file, and we want them in order. Using a sharded mutex
// keyed on client ID keeps the lock collision domain tiny.
var clientLocks sync.Map // map[string]*sync.Mutex

func clientLock(id string) *sync.Mutex {
	if v, ok := clientLocks.Load(id); ok {
		return v.(*sync.Mutex)
	}
	l := &sync.Mutex{}
	actual, _ := clientLocks.LoadOrStore(id, l)
	return actual.(*sync.Mutex)
}

func handleLogUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	clientID := r.Header.Get("X-Client-ID")
	if !clientIDRE.MatchString(clientID) {
		http.Error(w, "invalid X-Client-ID", http.StatusBadRequest)
		return
	}
	version := r.Header.Get("X-Portal-Version")

	r.Body = http.MaxBytesReader(w, r.Body, maxLogPayload)
	defer r.Body.Close()

	var src io.Reader = r.Body
	if r.Header.Get("Content-Encoding") == "gzip" {
		gr, err := gzip.NewReader(r.Body)
		if err != nil {
			http.Error(w, "bad gzip", http.StatusBadRequest)
			return
		}
		defer gr.Close()
		src = gr
	}

	dir := filepath.Join(LogUploadDir, clientID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Printf("logupload: mkdir: %v", err)
		http.Error(w, "internal", http.StatusInternalServerError)
		return
	}
	stamp := time.Now().UTC().Format("2006-01-02")
	path := filepath.Join(dir, "portal-"+stamp+".log")

	mu := clientLock(clientID)
	mu.Lock()
	defer mu.Unlock()

	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		log.Printf("logupload: open %s: %v", path, err)
		http.Error(w, "internal", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	// Write a one-line header so multiple uploads in the same file
	// stay distinguishable.
	header := "# upload at=" + time.Now().UTC().Format(time.RFC3339) +
		" version=" + version + " from=" + clientHost(r) + "\n"
	if _, err := f.WriteString(header); err != nil {
		log.Printf("logupload: write header: %v", err)
		http.Error(w, "internal", http.StatusInternalServerError)
		return
	}
	n, err := io.Copy(f, src)
	if err != nil {
		log.Printf("logupload: copy: %v (wrote %d bytes)", err, n)
		// Don't 500 — the client retries next tick anyway.
	}

	w.WriteHeader(http.StatusNoContent)
}

func clientHost(r *http.Request) string {
	// Trust X-Forwarded-For if you're behind Cloudflare; otherwise
	// RemoteAddr is fine. Strip port for tidiness.
	if v := r.Header.Get("CF-Connecting-IP"); v != "" {
		return v
	}
	return r.RemoteAddr
}
```

## Wiring it into your HTTP mux

```go
// In your existing server bootstrap:
http.HandleFunc("/logs/upload", handleLogUpload)

// If you already have a single Hub-style handler, wrap it:
mux := http.NewServeMux()
mux.HandleFunc("/ws", hub.Serve)             // existing signaling
mux.HandleFunc("/logs/upload", handleLogUpload) // new
log.Fatal(http.ListenAndServe(":443", mux))
```

## systemd service va deploy

```bash
# /etc/systemd/system/portal-signaling.service
[Service]
Environment="PORTAL_LOG_DIR=/var/log/portal/clients"
ExecStart=/usr/local/bin/portal-signaling
...

# Init the directory:
sudo mkdir -p /var/log/portal/clients
sudo chown portal:portal /var/log/portal/clients

sudo systemctl restart portal-signaling
```

## Logni o'qish

```bash
# Bir foydalanuvchining bugungi logi:
sudo cat /var/log/portal/clients/<client-id>/portal-2026-05-04.log

# Bugungi barcha xatolar:
sudo grep -h "level=WARN\|level=ERROR" /var/log/portal/clients/*/portal-2026-05-04.log

# Symmetric NAT tashxisi qaysi foydalanuvchilarda:
sudo grep -l "TURN serveri zarur" /var/log/portal/clients/*/portal-2026-05-04.log
```

## Maxfiylik haqida

Server logiga tushadigan ma'lumotlar:
- Peer ID'lar (UUID — anonim, kim ekanini bilmaysiz)
- LAN IP (`192.168.x.x`) — sezgir emas
- Public IP (`srflx`) — STUN serverlari ham ko'radi, sezgir emas
- Foydalanuvchi taxallusi (sezgir bo'lishi mumkin — masalan haqiqiy ism)
- Portal ID va kod (kodga ega bo'lgan kishi portal'ga kira oladi)

Productionda buni e'longa qo'shish tavsiya etiladi yoki default'ni
"opt-in"ga o'zgartirish (`storage.KeyLogUpload` default'ini `"0"`'ga
o'zgartirish — `client/app.go` `startLogSink` ichida).
