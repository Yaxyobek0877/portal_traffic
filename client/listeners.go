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

// localListenersCommand returns an os/exec command and a parser
// function appropriate for the current platform. Returns (nil, nil)
// if the platform isn't supported.
func localListenersCommand() (*exec.Cmd, func(string) []LocalListener) {
	switch runtime.GOOS {
	case "darwin":
		// `lsof -F pcn` emits one record per line with fields prefixed:
		//   p<pid> c<command> n<host:port>
		// Output is parseable without quoting tricks.
		return exec.Command("lsof", "-nP", "-iTCP", "-sTCP:LISTEN", "-F", "pcn"), parseLsofF
	case "linux":
		// `ss -H -lntp` ("no header, listening, numeric, tcp, processes")
		// State Recv-Q Send-Q Local Address:Port  Peer Address:Port  Process
		return exec.Command("ss", "-H", "-lntp"), parseSS
	default:
		return nil, nil
	}
}

// parseLsofF parses `lsof -F pcn` output. Records are sequences of
// lines starting with field-marker bytes; entries are grouped by
// process (`p`) followed by command (`c`) followed by one or more
// `n<host:port>` lines.
func parseLsofF(out string) []LocalListener {
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
			port := portFromAddr(addr)
			if port == 0 {
				continue
			}
			rows = append(rows, LocalListener{
				Port:    port,
				Process: cmd,
				PID:     pid,
				Local:   addr,
			})
		}
	}
	return rows
}

// parseSS parses `ss -H -lntp` output.
func parseSS(out string) []LocalListener {
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
			Port:    port,
			Process: cmd,
			PID:     pid,
			Local:   local,
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
