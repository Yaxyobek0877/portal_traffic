// portal-cli is the headless test harness for Phase 2. It connects to
// a signaling server, creates or joins a portal, and drives the full
// WebRTC mesh: handshake, data channels, and heartbeat. It prints
// every observable event so two instances on the same (or different)
// machines can verify the mesh end-to-end.
//
// Usage:
//
//	# Lokal dev (lokal signal serverga)
//	go run ./cmd/portal-cli -url ws://localhost:18080/ws -mode create -nick alice
//	go run ./cmd/portal-cli -url ws://localhost:18080/ws -mode join \
//	    -nick bob -portal 123456 -code 654321
//
//	# Produksiya (signaling.1pro.uz orqali)
//	go run ./cmd/portal-cli -mode create -nick alice
//	go run ./cmd/portal-cli -mode join -nick bob -portal 123456 -code 654321
//
// You should see a "PEER READY" line for each side when the WebRTC
// connection is fully up, followed by periodic RTT updates.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"portal_traffic_client/mesh"
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

	<-ctx.Done()
	fmt.Println("\nshutting down")
	_ = m.Leave()
	m.Close()
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
				fmt.Printf("[✓] PEER READY    %s (nick=%s, vip=%s) — all data channels open\n",
					short(p.ID), p.Nickname, p.VirtualIP)

			case mesh.EventPeerRTT:
				p := ev.Peer
				fmt.Printf("    RTT %-8s  %s (nick=%s)\n",
					p.RTT().Round(time.Microsecond*100), short(p.ID), p.Nickname)

			case mesh.EventPeerLeft:
				fmt.Printf("[-] PEER LEFT     %s\n", short(ev.Peer.ID))

			case mesh.EventPortalClosed:
				fmt.Println("[!] PORTAL CLOSED")

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
