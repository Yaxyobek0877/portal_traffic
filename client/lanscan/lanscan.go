// Package lanscan does a quick TCP probe across the host's local /24 on
// a small list of well-known service ports (RTSP, HTTP web UIs, IPP
// printers, etc.) and returns whatever it finds. The user gets a
// clickable list of LAN devices to expose through the mesh — much
// faster than typing 192.168.x.x:y by hand for each camera.
//
// Why TCP probe and not mDNS: cameras / NVRs / printers from cheaper
// vendors often don't publish via Bonjour, but they all answer on a
// well-known port. A 256×N scan with bounded concurrency finishes in
// a few seconds and catches everything mDNS misses.
package lanscan

import (
	"context"
	"fmt"
	"net"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// Discovery is a single live host:port found on the LAN.
type Discovery struct {
	IP       string `json:"ip"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol"` // "tcp" for now; UDP probing is unreliable
	Service  string `json:"service"`  // "rtsp" | "http" | "ipp" | … (best-guess from port)
	Hostname string `json:"hostname"` // reverse-DNS, best-effort, may be empty
}

// probedPort describes one port we probe and the human label we give
// hits on it. Order of the list controls UI sort priority too.
type probedPort struct {
	port    int
	service string
}

// Default port list — covers the things people actually want to share
// through Portal. Errs on the side of "include it" because a missed
// port means the user has to type IP:port by hand, while a false
// positive just shows up as one extra clickable row they can ignore.
//
// Roughly grouped by what each port commonly hosts:
//   • cameras / NVRs (Hikvision / Dahua / Reolink / generic ONVIF)
//   • web UIs (HTTP / HTTPS / common admin variants)
//   • NAS appliances (Synology / QNAP / TrueNAS / OMV)
//   • home automation (Home Assistant / OpenHAB / ESPHome / Zigbee2MQTT)
//   • media servers (Plex / Jellyfin / Emby)
//   • dev tools (Vite / Next / Jupyter / Selenium / debug)
//   • file / print sharing (SMB / IPP / raw print)
//   • game servers (Minecraft / CS2 / Rust / Terraria — TCP only here;
//     UDP-only games are surfaced via the lsof UDP sweep instead)
//
// 254 IPs × 30 ports = ~7.6K connects; with 256-way parallelism and
// a 400ms per-connect timeout this lands in under 10s on a healthy
// Wi-Fi. Slow networks just take a bit longer.
var defaultPorts = []probedPort{
	// Cameras / NVRs
	{554, "rtsp"},
	{8554, "rtsp-alt"},
	{37777, "dahua"},
	{34567, "dahua-cam"},
	{8000, "hikvision"},
	{8001, "hikvision-sdk"},
	{81, "cam-web"},
	// Web UIs
	{80, "http"},
	{443, "https"},
	{8080, "http-alt"},
	{8443, "https-alt"},
	// NAS
	{5000, "synology"},
	{5001, "synology-https"},
	{8200, "qnap"},
	// Home automation / media servers
	{8123, "home-assistant"},
	{1883, "mqtt"},
	{32400, "plex"},
	{8096, "jellyfin"},
	// Dev tools
	{3000, "web-dev"},
	{5173, "vite"},
	{4200, "angular"},
	{8888, "jupyter"},
	{9000, "web-dev"},
	// File / print sharing
	{631, "ipp"},
	{9100, "rawprint"},
	{139, "smb"},
	{445, "smb"},
	// Remote shell
	{22, "ssh"},
	// Common game servers (TCP)
	{25565, "minecraft"},
	{7777, "terraria"},
}

// Progress reports incremental scan state. Emitted by ScanWithProgress
// roughly every 5% of the IP queue so the UI can paint a live counter
// instead of staring at a 12-second blank panel.
type Progress struct {
	Done    int    `json:"done"`    // probes completed
	Total   int    `json:"total"`   // total probes scheduled
	Hits    int    `json:"hits"`    // discoveries found so far
	Current string `json:"current"` // last IP being probed (best-effort, may lag)
}

// Scan walks the host's primary IPv4 /24 and returns reachable
// (host:port) tuples. ctx caps the total scan time; a 5–10s budget
// is the sweet spot — past that you're just probing dead IPs.
func Scan(ctx context.Context) []Discovery {
	return ScanWithProgress(ctx, nil)
}

// ScanWithProgress is Scan with a progress callback. cb may be nil.
// Called from a worker goroutine so it should be cheap and
// non-blocking — typically just emits a Wails event.
func ScanWithProgress(ctx context.Context, cb func(Progress)) []Discovery {
	prefixes, err := allLANPrefixes()
	if err != nil || len(prefixes) == 0 {
		return nil
	}

	type target struct {
		ip   string
		port int
		svc  string
	}
	queue := make(chan target, 1024)

	var results []Discovery
	var resMu sync.Mutex

	total := 254 * len(defaultPorts) * len(prefixes)
	var done int64
	var hits int64
	var lastIP atomic.Value // string
	lastIP.Store("")

	const workers = 256
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			d := net.Dialer{Timeout: 400 * time.Millisecond}
			for t := range queue {
				select {
				case <-ctx.Done():
					return
				default:
				}
				addr := fmt.Sprintf("%s:%d", t.ip, t.port)
				lastIP.Store(t.ip)
				conn, err := d.DialContext(ctx, "tcp", addr)
				atomic.AddInt64(&done, 1)
				if err != nil {
					continue
				}
				_ = conn.Close()
				atomic.AddInt64(&hits, 1)
				resMu.Lock()
				results = append(results, Discovery{
					IP: t.ip, Port: t.port, Protocol: "tcp", Service: t.svc,
				})
				resMu.Unlock()
			}
		}()
	}

	// Progress emitter — ticks roughly every 200ms so the UI sees
	// continuous motion without us spamming Wails events.
	if cb != nil {
		var tickerCancel context.CancelFunc
		var tickerCtx context.Context
		tickerCtx, tickerCancel = context.WithCancel(ctx)
		go func() {
			t := time.NewTicker(200 * time.Millisecond)
			defer t.Stop()
			for {
				select {
				case <-tickerCtx.Done():
					return
				case <-t.C:
					ip, _ := lastIP.Load().(string)
					cb(Progress{
						Done:    int(atomic.LoadInt64(&done)),
						Total:   total,
						Hits:    int(atomic.LoadInt64(&hits)),
						Current: ip,
					})
				}
			}
		}()
		defer tickerCancel()
	}

	// Enqueue every (subnet, IP, port) combo. The host's own IP is
	// included — useful when a service is on this Mac itself but
	// bound to the LAN address rather than 127.0.0.1 (it won't show
	// up under the "lokalda topilgan" lsof sweep then).
	for _, prefix := range prefixes {
		for ip := 1; ip <= 254; ip++ {
			host := fmt.Sprintf("%s.%d", prefix, ip)
			for _, pp := range defaultPorts {
				select {
				case <-ctx.Done():
					close(queue)
					wg.Wait()
					return sortResults(results)
				case queue <- target{host, pp.port, pp.service}:
				}
			}
		}
	}
	close(queue)
	wg.Wait()

	// Best-effort reverse DNS for the hits — names are friendlier than
	// raw IPs ("eshik-camera.local" vs "192.168.1.100"). Bounded so a
	// stalled DNS doesn't block the whole scan.
	resolveHostnames(ctx, results)

	return sortResults(results)
}

// allLANPrefixes returns every "X.Y.Z" /24 prefix derived from the
// host's non-loopback IPv4 interfaces. Multi-interface laptops (Wi-Fi
// + ethernet + corporate VPN + container bridges) all get scanned.
//
// Always uses the host IP's /24, regardless of the interface mask:
//   - /24 (192.168.1.5/24)  → "192.168.1"
//   - /16 (10.5.6.7/16)     → "10.5.6" (the host's neighbourhood)
//   - /8  (10.5.6.7/8)      → "10.5.6" too — scanning all 16M of
//     a /8 would take hours and find nothing useful, but the host's
//     own /24 is exactly where its peers live.
//
// Skips public IPs entirely (refuses to port-scan the internet) —
// only RFC1918 / CGNAT ranges are eligible. A laptop on a server
// with a directly-attached public IPv4 won't accidentally scan
// other people's servers.
func allLANPrefixes() ([]string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	var out []string
	seen := map[string]bool{}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ip4 := ipnet.IP.To4()
			if ip4 == nil {
				continue
			}
			if ip4.IsLoopback() || ip4.IsLinkLocalUnicast() {
				continue
			}
			if !isPrivateIPv4(ip4) {
				// Public IP on the interface (e.g. running on a VPS).
				// Skip — port-scanning the wider internet is rude
				// and would just produce false positives anyway.
				continue
			}
			prefix := fmt.Sprintf("%d.%d.%d", ip4[0], ip4[1], ip4[2])
			if seen[prefix] {
				continue
			}
			seen[prefix] = true
			out = append(out, prefix)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("lanscan: no private IPv4 interface found")
	}
	return out, nil
}

// isPrivateIPv4 covers the standard RFC1918 ranges plus CGNAT
// (100.64.0.0/10). Anything else is treated as public and skipped
// by the scanner.
func isPrivateIPv4(ip4 net.IP) bool {
	if ip4 = ip4.To4(); ip4 == nil {
		return false
	}
	switch {
	case ip4[0] == 10:
		return true
	case ip4[0] == 192 && ip4[1] == 168:
		return true
	case ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31:
		return true
	case ip4[0] == 100 && ip4[1] >= 64 && ip4[1] <= 127:
		return true
	}
	return false
}

// resolveHostnames does a parallel reverse DNS lookup with a tight
// timeout so the UI gets a label per device when possible.
func resolveHostnames(ctx context.Context, results []Discovery) {
	rctx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
	defer cancel()

	// Dedup IPs to avoid 5x lookups for a host that has 5 open ports.
	seen := map[string]string{}
	var seenMu sync.Mutex
	var wg sync.WaitGroup
	for i := range results {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			ip := results[i].IP
			seenMu.Lock()
			if name, ok := seen[ip]; ok {
				seenMu.Unlock()
				results[i].Hostname = name
				return
			}
			seenMu.Unlock()

			r := net.Resolver{}
			names, err := r.LookupAddr(rctx, ip)
			name := ""
			if err == nil && len(names) > 0 {
				name = names[0]
				// strip trailing dot from PTR records
				if n := len(name); n > 0 && name[n-1] == '.' {
					name = name[:n-1]
				}
			}
			seenMu.Lock()
			seen[ip] = name
			seenMu.Unlock()
			results[i].Hostname = name
		}()
	}
	wg.Wait()
}

// sortResults orders by IP-numeric, then port. Stable so the UI
// doesn't reshuffle when a re-scan finds the same set.
func sortResults(in []Discovery) []Discovery {
	sort.SliceStable(in, func(i, j int) bool {
		if in[i].IP != in[j].IP {
			return ipNumeric(in[i].IP) < ipNumeric(in[j].IP)
		}
		return in[i].Port < in[j].Port
	})
	return in
}

func ipNumeric(s string) uint32 {
	ip := net.ParseIP(s).To4()
	if ip == nil {
		return 0
	}
	return uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])
}
