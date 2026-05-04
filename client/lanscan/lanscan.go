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

// Default port list — small enough to scan a /24 in a few seconds
// (256 × 11 = ~2.8K connects, 64-way parallel ≈ 4–8s on a typical
// home Wi-Fi). Covers the obvious stuff most users want to share.
var defaultPorts = []probedPort{
	{554, "rtsp"},     // IP cameras
	{80, "http"},      // routers, NVRs, NAS web UIs
	{443, "https"},    // same
	{8080, "http-alt"},
	{8000, "http-alt"},
	{8123, "home-assistant"},
	{5000, "http-alt"}, // Synology DSM, Flask dev servers
	{32400, "plex"},
	{631, "ipp"},      // network printers
	{9100, "rawprint"},
	{22, "ssh"},
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
	prefix, err := primaryLANPrefix()
	if err != nil {
		return nil
	}

	type target struct {
		ip   string
		port int
		svc  string
	}
	queue := make(chan target, 256)

	var results []Discovery
	var resMu sync.Mutex

	total := 254 * len(defaultPorts)
	var done int64
	var hits int64
	var lastIP atomic.Value // string
	lastIP.Store("")

	const workers = 64
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			d := net.Dialer{Timeout: 600 * time.Millisecond}
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

	// Enqueue every (IP, port) combo. The host's own IP is included —
	// useful when a service is on this Mac itself but bound to the
	// LAN address rather than 127.0.0.1 (it won't show up under the
	// "lokalda topilgan" lsof sweep then).
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
	close(queue)
	wg.Wait()

	// Best-effort reverse DNS for the hits — names are friendlier than
	// raw IPs ("eshik-camera.local" vs "192.168.1.100"). Bounded so a
	// stalled DNS doesn't block the whole scan.
	resolveHostnames(ctx, results)

	return sortResults(results)
}

// primaryLANPrefix returns "192.168.1" / "10.0.0" / "172.16.42" — the
// /24 prefix of the host's primary non-loopback IPv4 interface.
// Skips link-local (169.254.x) and loopback. Picks the first match
// because most home setups have one obvious LAN interface.
func primaryLANPrefix() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
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
			// Only scan if /16 or smaller — refusing to enumerate
			// 192.0.0.0/8-style mask if some misconfiguration sets
			// one up.
			ones, _ := ipnet.Mask.Size()
			if ones < 16 {
				continue
			}
			return fmt.Sprintf("%d.%d.%d", ip4[0], ip4[1], ip4[2]), nil
		}
	}
	return "", fmt.Errorf("lanscan: no primary IPv4 interface found")
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
