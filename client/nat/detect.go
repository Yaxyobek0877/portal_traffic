// Package nat does a quick STUN-based NAT classification on app
// startup. The result is shown in the status bar so the user knows
// whether direct WebRTC connections are likely to succeed; if we
// detect a Symmetric NAT or CGNAT we surface a banner asking them
// to configure a TURN server.
//
// We don't attempt full RFC 5780 NAT-behaviour discovery. The
// algorithm here is intentionally simpler:
//
//  1. Send a STUN binding request to two distinct servers.
//  2. Compare each response's mapped address to our local socket.
//     - If they match, we're not behind any NAT (or the NAT is "open").
//     - If they differ from local but match each other, we're behind a
//       reasonable cone NAT — direct WebRTC will probably work.
//     - If the two reflexive addresses are different ports, the NAT is
//       address+port-mapping (i.e. Symmetric or some CGNATs) — direct
//       WebRTC is unlikely without TURN.
//
// This is fast (~300ms typical), cheap (two UDP exchanges), and
// accurate enough to give a useful UI hint.
package nat

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/pion/stun/v2"
)

// Type describes the result of NAT classification.
type Type int

const (
	TypeUnknown Type = iota
	TypeDirect              // not behind a NAT (or fully open)
	TypeCone                // address-mapped but port consistent across servers
	TypeSymmetric           // each destination produces a different external port
	TypeUnreachable         // both STUN servers timed out
)

func (t Type) String() string {
	return [...]string{
		"unknown", "direct", "cone", "symmetric", "unreachable",
	}[t]
}

// HumanLabel returns a short Uzbek-language label suitable for UI.
func (t Type) HumanLabel() string {
	switch t {
	case TypeDirect:
		return "to'g'ridan-to'g'ri"
	case TypeCone:
		return "NAT (yaxshi)"
	case TypeSymmetric:
		return "Simmetrik NAT"
	case TypeUnreachable:
		return "STUN yetib bo'lmaydi"
	default:
		return "noma'lum"
	}
}

// Result is what Detect returns.
type Result struct {
	Type            Type      `json:"type"`
	Label           string    `json:"label"`
	LocalAddr       string    `json:"localAddr"`
	ReflexiveAddrs  []string  `json:"reflexiveAddrs"`
	NeedsTURN       bool      `json:"needsTurn"`
	DetectedAt      time.Time `json:"detectedAt"`
	Servers         []string  `json:"servers"`
}

// DefaultServers is the canonical pair we test against. Both should be
// on the open internet and reachable from anywhere; we deliberately
// pick *different* operators so a single failure doesn't tank the
// classification.
var DefaultServers = []string{
	"stun.l.google.com:19302",
	"stun.cloudflare.com:3478",
}

// Detect runs the classification. Use ctx to bound how long we'll wait;
// 4 seconds is usually plenty.
func Detect(ctx context.Context, servers []string) (Result, error) {
	if len(servers) < 2 {
		return Result{}, errors.New("nat: need at least two STUN servers for classification")
	}

	type probe struct {
		server   string
		local    *net.UDPAddr
		mapped   *net.UDPAddr
		err      error
	}

	probes := make([]probe, len(servers))
	var wg sync.WaitGroup
	for i, s := range servers {
		i, s := i, s
		wg.Add(1)
		go func() {
			defer wg.Done()
			local, mapped, err := stunMap(ctx, s)
			probes[i] = probe{server: s, local: local, mapped: mapped, err: err}
		}()
	}
	wg.Wait()

	res := Result{
		DetectedAt: time.Now(),
		Servers:    servers,
	}

	successes := 0
	for _, p := range probes {
		if p.err != nil {
			continue
		}
		successes++
		res.LocalAddr = p.local.String()
		res.ReflexiveAddrs = append(res.ReflexiveAddrs, p.mapped.String())
	}

	switch successes {
	case 0:
		res.Type = TypeUnreachable
		res.NeedsTURN = false // we don't actually know
	case 1:
		// Only one probe succeeded. We can tell if there's a NAT (local
		// vs mapped) but not whether it's symmetric.
		var p probe
		for _, x := range probes {
			if x.err == nil {
				p = x
				break
			}
		}
		if addrEqual(p.local, p.mapped) {
			res.Type = TypeDirect
		} else {
			res.Type = TypeCone
		}
	default:
		// Two or more successful probes — we can compare ports.
		var first, second *net.UDPAddr
		for _, p := range probes {
			if p.err != nil {
				continue
			}
			if first == nil {
				first = p.mapped
			} else {
				second = p.mapped
				break
			}
		}
		switch {
		case addrEqual(first, second):
			// Same external port for two destinations → cone-like.
			// If it also matches local addr → direct.
			if addrEqual(probes[0].local, first) {
				res.Type = TypeDirect
			} else {
				res.Type = TypeCone
			}
		default:
			res.Type = TypeSymmetric
			res.NeedsTURN = true
		}
	}
	res.Label = res.Type.HumanLabel()
	return res, nil
}

// stunMap performs a single STUN binding request and returns the local
// socket address plus the server-reflexive (mapped) address.
func stunMap(ctx context.Context, server string) (*net.UDPAddr, *net.UDPAddr, error) {
	dialer := &net.Dialer{Timeout: 3 * time.Second}
	conn, err := dialer.DialContext(ctx, "udp4", server)
	if err != nil {
		return nil, nil, fmt.Errorf("dial %s: %w", server, err)
	}
	defer conn.Close()

	udpConn, ok := conn.(*net.UDPConn)
	if !ok {
		return nil, nil, errors.New("not a UDP conn")
	}

	deadline := time.Now().Add(2 * time.Second)
	if d, hasDL := ctx.Deadline(); hasDL && d.Before(deadline) {
		deadline = d
	}
	_ = udpConn.SetDeadline(deadline)

	msg := stun.MustBuild(stun.TransactionID, stun.BindingRequest)
	if _, err := udpConn.Write(msg.Raw); err != nil {
		return nil, nil, fmt.Errorf("write stun: %w", err)
	}

	buf := make([]byte, 1500)
	n, _, err := udpConn.ReadFromUDP(buf)
	if err != nil {
		return nil, nil, fmt.Errorf("read stun: %w", err)
	}
	resp := &stun.Message{Raw: buf[:n]}
	if err := resp.Decode(); err != nil {
		return nil, nil, fmt.Errorf("decode stun: %w", err)
	}

	// Prefer XOR-MAPPED-ADDRESS; fall back to MAPPED-ADDRESS.
	var xor stun.XORMappedAddress
	if err := xor.GetFrom(resp); err == nil {
		mapped := &net.UDPAddr{IP: xor.IP, Port: xor.Port}
		local := udpConn.LocalAddr().(*net.UDPAddr)
		return local, mapped, nil
	}
	var plain stun.MappedAddress
	if err := plain.GetFrom(resp); err == nil {
		mapped := &net.UDPAddr{IP: plain.IP, Port: plain.Port}
		local := udpConn.LocalAddr().(*net.UDPAddr)
		return local, mapped, nil
	}
	return nil, nil, errors.New("no mapped address attribute in STUN response")
}

func addrEqual(a, b *net.UDPAddr) bool {
	if a == nil || b == nil {
		return false
	}
	return a.IP.Equal(b.IP) && a.Port == b.Port
}
