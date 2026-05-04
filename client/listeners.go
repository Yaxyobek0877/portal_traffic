// Local TCP-listener enumeration. macOS + Linux both ship reasonable
// CLI tools we can shell out to and parse. Windows has netstat with a
// different format we don't bother with right now.
package main

import (
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// macosSystemProcesses are macOS daemons that hold standard ports the
// user almost never wants to expose to a portal mesh — ControlCenter
// owns :5000/:7000 for AirPlay Receiver, sharingd handles iCloud/AirDrop,
// rapportd is Continuity, etc. We hide these from the "detected ports"
// list so a one-click expose doesn't accidentally share macOS internals.
var macosSystemProcesses = map[string]bool{
	"ControlCenter":     true, // AirPlay receiver — :5000/:7000
	"mDNSResponder":     true, // Bonjour — :5353
	"rapportd":          true, // Continuity / Handoff
	"sharingd":          true, // AirDrop / file sharing
	"identityservicesd": true, // iMessage / FaceTime
	"AppleAccountSer":   true, // iCloud (truncated by macOS to 15 chars in some tools)
	"AppleAccountServer": true,
	"remoted":           true, // remote-management daemon
	"airportd":          true, // Wi-Fi management
	"nehelper":          true, // Network Extension framework
	"launchd":           true, // PID 1
	"cloudd":            true, // CloudKit daemon
	"distnoted":         true,
	"trustd":            true, // certificate trust
	"corespeechd":       true, // Siri
	"avconferenced":     true, // FaceTime audio/video
	"WiFiAgent":         true,
	"replicatord":       true,
	"mdworker":          true, // Spotlight
	"launchservicesd":   true,
	"locationd":         true,
	"netbiosd":          true,
	"adprivacyd":        true,
	"akd":               true, // AppleID daemon
	"bluetoothd":        true,
	"coreaudiod":        true,
	"airplayhelperd":    true,
	"appstoreagent":     true,
}

func isSystemProcess(cmd string) bool {
	if runtime.GOOS != "darwin" {
		return false
	}
	if macosSystemProcesses[cmd] {
		return true
	}
	// Anything starting with "com.apple." is an Apple LaunchAgent/Daemon.
	if strings.HasPrefix(cmd, "com.apple.") {
		return true
	}
	return false
}

// localListenerCommands returns the platform-specific commands to
// enumerate TCP and UDP listeners separately, each tagged with the
// protocol the parser should set on emitted LocalListener rows. We
// enumerate the two protocols separately because game servers (CS2,
// Valorant, Minecraft Bedrock) bind UDP and a TCP-only sweep silently
// misses them — that was the bug behind "Portal can't see my CS2
// server".
//
// Returns nil when the platform isn't supported (Windows for now).
type listenerProbe struct {
	cmd      *exec.Cmd
	protocol string
	parser   func(out string, protocol string) []LocalListener
}

func localListenerProbes() []listenerProbe {
	switch runtime.GOOS {
	case "darwin":
		// `lsof -F pcn` emits one record per line with fields prefixed:
		//   p<pid> c<command> n<host:port>
		// Output is parseable without quoting tricks. We run two passes:
		// one for TCP listeners, one for UDP. UDP doesn't have a
		// listening "state" the way TCP does (no -sTCP:LISTEN
		// equivalent) so for UDP we just enumerate every UDP socket and
		// rely on the upper-layer filter to drop noise.
		return []listenerProbe{
			{cmd: exec.Command("lsof", "-nP", "-iTCP", "-sTCP:LISTEN", "-F", "pcn"), protocol: "tcp", parser: parseLsofF},
			{cmd: exec.Command("lsof", "-nP", "-iUDP", "-F", "pcn"), protocol: "udp", parser: parseLsofF},
		}
	case "linux":
		// `ss -H -lntp` for TCP listeners; `ss -H -lnup` for UDP.
		return []listenerProbe{
			{cmd: exec.Command("ss", "-H", "-lntp"), protocol: "tcp", parser: parseSS},
			{cmd: exec.Command("ss", "-H", "-lnup"), protocol: "udp", parser: parseSS},
		}
	default:
		return nil
	}
}

// parseLsofF parses `lsof -F pcn` output. Records are sequences of
// lines starting with field-marker bytes; entries are grouped by
// process (`p`) followed by command (`c`) followed by one or more
// `n<host:port>` lines.
func parseLsofF(out string, protocol string) []LocalListener {
	var rows []LocalListener
	var pid int
	var cmd string
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		switch line[0] {
		case 'p':
			n, err := strconv.Atoi(line[1:])
			if err == nil {
				pid = n
			}
		case 'c':
			cmd = line[1:]
		case 'n':
			addr := line[1:]
			// addr looks like: *:8000  127.0.0.1:6379  [::1]:5432
			// UDP lsof can also emit "*:*" for unbound — skip those.
			port := portFromAddr(addr)
			if port == 0 {
				continue
			}
			if isSystemProcess(cmd) {
				continue
			}
			rows = append(rows, LocalListener{
				Port:     port,
				Protocol: protocol,
				Process:  cmd,
				PID:      pid,
				Local:    addr,
			})
		}
	}
	return rows
}

// parseSS parses `ss -H -lntp` / `ss -H -lnup` output.
func parseSS(out string, protocol string) []LocalListener {
	var rows []LocalListener
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		// Local Address:Port is field 3 in `ss -lntp`.
		local := fields[3]
		port := portFromAddr(local)
		if port == 0 {
			continue
		}
		// Process column looks like: users:(("nginx",pid=1234,fd=6))
		var pid int
		var cmd string
		users := strings.Join(fields[5:], " ")
		if i := strings.Index(users, `"`); i >= 0 {
			rest := users[i+1:]
			if j := strings.Index(rest, `"`); j > 0 {
				cmd = rest[:j]
			}
		}
		if i := strings.Index(users, "pid="); i >= 0 {
			rest := users[i+4:]
			j := strings.IndexAny(rest, ",) ")
			if j < 0 {
				j = len(rest)
			}
			pid, _ = strconv.Atoi(rest[:j])
		}
		rows = append(rows, LocalListener{
			Port:     port,
			Protocol: protocol,
			Process:  cmd,
			PID:      pid,
			Local:    local,
		})
	}
	return rows
}

// portFromAddr extracts the trailing port from "host:port" / "*:port"
// / "[::1]:port" / "[fe80::...]:port".
func portFromAddr(addr string) int {
	i := strings.LastIndex(addr, ":")
	if i < 0 {
		return 0
	}
	p, err := strconv.Atoi(addr[i+1:])
	if err != nil {
		return 0
	}
	return p
}
