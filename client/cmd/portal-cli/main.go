// portal-cli is the headless test harness for Phase 2+5. It connects
// to a signaling server, creates or joins a portal, drives the full
// WebRTC mesh, and (optionally) exposes/dials TCP services through
// the mesh's proxy data channel.
//
// Usage:
//
//	# Lokal dev — host
//	go run ./cmd/portal-cli -url ws://localhost:18080/ws -mode create -nick alice
//
//	# Lokal dev — joiner
//	go run ./cmd/portal-cli -url ws://localhost:18080/ws -mode join -nick bob \
//	    -portal 123456 -code 654321
//
//	# Lokal dev — host + expose a local TCP service into the mesh
//	go run ./cmd/portal-cli -mode create -nick alice -expose tcp:8000
//
//	# Lokal dev — joiner + dial host's exposed port through the mesh
//	go run ./cmd/portal-cli -mode join -nick bob -portal ... -code ... \
//	    -dial tcp:<owner-peer-id>:8000:9001
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"portal_traffic_client/mesh"
	"portal_traffic_client/proxy"
	"portal_traffic_client/signaling"
)

func main() {
	defaultURL := signaling.DefaultURL
	if v := os.Getenv("PORTAL_SIGNALING_URL"); v != "" {
		defaultURL = v
	}
	var (
		url    = flag.String("url", defaultURL, "signaling WS URL (env PORTAL_SIGNALING_URL)")
		mode   = flag.String("mode", "create", "create | join")
		nick   = flag.String("nick", "tester", "nickname")
		portal = flag.String("portal", "", "portal ID (for join)")
		code   = flag.String("code", "", "portal code (for join)")
		hbz    = flag.Duration("hb", 2*time.Second, "heartbeat interval")
		debug  = flag.Bool("debug", false, "verbose logging")
		expose = flag.String("expose", "", "expose a local service to the mesh, e.g. tcp:8000")
		dial   = flag.String("dial", "", "open a local listener that pumps to a peer's exposed port. Format: tcp:<peer-id>:<remote-port>:<local-port>")
	)
	flag.Parse()

	level := slog.LevelInfo
	if *debug {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))

	m := mesh.New(mesh.Config{
		SignalingURL:      *url,
		Nickname:          *nick,
		HeartbeatInterval: *hbz,
		Logger:            logger,
	})

	// Wire the proxy. It's cheap to set up even if we never use it;
	// peers we never speak to in the proxy channel cost nothing.
	fwd := proxy.New(m, logger)
	m.SetProxyHandler(fwd)
	defer fwd.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	switch *mode {
	case "create":
		if err := m.CreatePortal(ctx); err != nil {
			log.Fatalf("CreatePortal: %v", err)
		}
	case "join":
		if *portal == "" || *code == "" {
			log.Fatal("join needs -portal and -code")
		}
		if err := m.JoinPortal(ctx, *portal, *code); err != nil {
			log.Fatalf("JoinPortal: %v", err)
		}
	default:
		log.Fatalf("unknown mode %q", *mode)
	}

	go printEvents(m)

	// Defer expose/dial until after the portal is up — these need the
	// mesh to have finished its initial handshake to be useful.
	if *expose != "" {
		go func() {
			waitPortalReady(m)
			if err := applyExpose(m, fwd, *expose); err != nil {
				fmt.Printf("[!] expose error: %v\n", err)
			}
		}()
	}
	if *dial != "" {
		go func() {
			// Slightly more wait since dial needs the peer to be ready
			// (their proxy channel open).
			waitPortalReady(m)
			time.Sleep(2 * time.Second)
			if err := applyDial(ctx, fwd, *dial); err != nil {
				fmt.Printf("[!] dial error: %v\n", err)
			}
		}()
	}

	<-ctx.Done()
	fmt.Println("\nshutting down")
	_ = m.Leave()
	m.Close()
}

func waitPortalReady(m *mesh.Manager) {
	for m.Portal() == nil {
		time.Sleep(100 * time.Millisecond)
	}
}

func applyExpose(m *mesh.Manager, fwd *proxy.Forwarder, spec string) error {
	// "tcp:8000" or "tcp:8000:my-name"
	parts := strings.SplitN(spec, ":", 3)
	if len(parts) < 2 || parts[0] != "tcp" {
		return fmt.Errorf("expose format: tcp:<port>[:<name>]")
	}
	port, err := strconv.Atoi(parts[1])
	if err != nil {
		return fmt.Errorf("port: %w", err)
	}
	name := "tcp:" + parts[1]
	if len(parts) == 3 {
		name = parts[2]
	}
	fwd.Expose(port)
	if err := m.AnnounceService(name, "tcp", port); err != nil {
		return err
	}
	fmt.Printf("[!] EXPOSED  tcp:%d (%s) — peers can dial via mesh\n", port, name)
	return nil
}

func applyDial(ctx context.Context, fwd *proxy.Forwarder, spec string) error {
	// "tcp:<peer-id>:<remote-port>:<local-port>"
	parts := strings.SplitN(spec, ":", 4)
	if len(parts) != 4 || parts[0] != "tcp" {
		return fmt.Errorf("dial format: tcp:<peer-id>:<remote-port>:<local-port>")
	}
	remotePort, err := strconv.Atoi(parts[2])
	if err != nil {
		return err
	}
	localPort := parts[3]
	var ln net.Listener
	if localPort == "0" || localPort == "" {
		// Match the GUI behaviour: prefer 127.0.0.1:<remotePort>, fall
		// back to OS-pick if it's busy.
		ln, err = fwd.DialPreferringPort(ctx, parts[1], remotePort)
	} else {
		if !strings.Contains(localPort, ":") {
			localPort = "127.0.0.1:" + localPort
		}
		ln, err = fwd.Dial(ctx, parts[1], remotePort, localPort)
	}
	if err != nil {
		return err
	}
	fmt.Printf("[!] DIALING  local %s → peer %s :%d\n", ln.Addr(), short(parts[1]), remotePort)
	return nil
}

func printEvents(m *mesh.Manager) {
	for {
		select {
		case <-m.Done():
			return
		case ev := <-m.Events():
			switch ev.Type {
			case mesh.EventPortalReady:
				p := ev.Portal
				owner := "owner"
				if !p.IsOwner {
					owner = "joiner"
				}
				fmt.Printf("\n=== PORTAL READY (%s) ===\n", owner)
				fmt.Printf("  ID:    %s\n", p.PortalID)
				if p.Code != "" {
					fmt.Printf("  CODE:  %s\n", p.Code)
				}
				fmt.Printf("  ME:    %s @ %s\n\n", p.OwnPeerID, p.OwnVIP)

			case mesh.EventPeerJoining:
				p := ev.Peer
				fmt.Printf("[+] PEER JOINING  %s (nick=%s, vip=%s) — handshake started\n",
					short(p.ID), p.Nickname, p.VirtualIP)

			case mesh.EventPeerReady:
				p := ev.Peer
				fmt.Printf("[✓] PEER READY    %s (nick=%s, vip=%s) — channels open\n",
					short(p.ID), p.Nickname, p.VirtualIP)

			case mesh.EventPeerRTT:
				p := ev.Peer
				fmt.Printf("    RTT %-8s  %s (nick=%s)\n",
					p.RTT().Round(time.Microsecond*100), short(p.ID), p.Nickname)

			case mesh.EventPeerLeft:
				fmt.Printf("[-] PEER LEFT     %s\n", short(ev.Peer.ID))

			case mesh.EventPortalClosed:
				fmt.Println("[!] PORTAL CLOSED")

			case mesh.EventChat:
				fmt.Printf("    CHAT      %s: %s\n", short(ev.Peer.ID), ev.ChatText)

			case mesh.EventServiceAnnounce:
				p := ev.Peer
				services := p.Services()
				if len(services) == 0 {
					continue
				}
				fmt.Printf("    SERVICES  %s exposes:", short(p.ID))
				for _, s := range services {
					fmt.Printf(" %s:%d (%s)", s.Protocol, s.Port, s.Name)
				}
				fmt.Println()

			case mesh.EventError:
				fmt.Printf("[!] ERROR: %v\n", ev.Err)
			}
		}
	}
}

func short(s string) string {
	if len(s) <= 8 {
		return s
	}
	return s[:8]
}
