package builtins

import (
	"bush/internal/color"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type PortInfo struct {
	Port     int
	Proto    string
	PID      int
	Process  string
	Cmdline  string
	User     string
	State    string
}

func findProcessOnPort(port int) (*PortInfo, error) {
	out, err := exec.Command("lsof", "-i", fmt.Sprintf(":%d", port), "-sTCP:LISTEN", "-t").Output()
	if err == nil && len(out) > 0 {
		pidStr := strings.TrimSpace(strings.Split(string(out), "\n")[0])
		if pid, err := strconv.Atoi(pidStr); err == nil && pid > 0 {
			return getPortInfoForPID(port, "TCP", pid), nil
		}
	}

	ssOut, err := exec.Command("ss", "-tulpn").Output()
	if err == nil {
		lines := strings.Split(string(ssOut), "\n")
		portPattern := fmt.Sprintf(":%d ", port)
		for _, line := range lines {
			if strings.Contains(line, portPattern) || strings.Contains(line, fmt.Sprintf(":%d\t", port)) {
				proto := "TCP"
				if strings.HasPrefix(line, "udp") {
					proto = "UDP"
				}
				pid := extractPIDFromSS(line)
				if pid > 0 {
					return getPortInfoForPID(port, proto, pid), nil
				}
				return &PortInfo{
					Port:    port,
					Proto:   proto,
					State:   "LISTEN",
					Process: "system/unknown",
				}, nil
			}
		}
	}

	return nil, fmt.Errorf("no process found listening on port %d", port)
}

func extractPIDFromSS(line string) int {
	idx := strings.Index(line, "pid=")
	if idx == -1 {
		return 0
	}
	sub := line[idx+4:]
	end := strings.IndexAny(sub, ",)")
	if end != -1 {
		sub = sub[:end]
	}
	pid, _ := strconv.Atoi(sub)
	return pid
}

func getPortInfoForPID(port int, proto string, pid int) *PortInfo {
	info := &PortInfo{
		Port:    port,
		Proto:   proto,
		PID:     pid,
		State:   "LISTEN",
		Process: "unknown",
	}

	cmdBytes, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err == nil {
		parts := strings.Split(string(cmdBytes), "\x00")
		if len(parts) > 0 && parts[0] != "" {
			info.Cmdline = strings.Join(parts, " ")
			tokens := strings.Split(parts[0], "/")
			info.Process = tokens[len(tokens)-1]
		}
	}

	if commBytes, err := os.ReadFile(fmt.Sprintf("/proc/%d/comm", pid)); err == nil {
		name := strings.TrimSpace(string(commBytes))
		if name != "" {
			info.Process = name
		}
	}

	return info
}

func builtinPort(args []string, ctx *ShellContext) int {
	pal := color.ActivePalette

	if len(args) == 1 {
		out, err := exec.Command("ss", "-tulpn", "-H").Output()
		if err != nil {
			fmt.Fprintf(ctx.Stderr, "bush: port: %v\n", err)
			return 1
		}

		fmt.Fprintln(ctx.Stdout, color.BoldColorize("+--- Active Listening Ports -------------------------------+", pal.Accent))
		fmt.Fprintf(ctx.Stdout, "| %-6s %-20s %-12s %s\n", "PROTO", "LOCAL ADDRESS", "PID", "PROCESS")
		fmt.Fprintln(ctx.Stdout, color.BoldColorize("+----------------------------------------------------------+", pal.Accent))

		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		count := 0
		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) >= 5 {
				proto := fields[0]
				addr := fields[4]
				proc := ""
				if len(fields) >= 7 {
					proc = fields[6]
				}
				fmt.Fprintf(ctx.Stdout, "| %-6s %-20s %s\n",
					color.Colorize(proto, pal.PromptSymbol),
					color.Colorize(addr, pal.Directory),
					color.Colorize(proc, pal.Flags),
				)
				count++
			}
		}
		if count == 0 {
			fmt.Fprintln(ctx.Stdout, "| No active listening ports detected.")
		}
		fmt.Fprintln(ctx.Stdout, color.BoldColorize("+----------------------------------------------------------+", pal.Accent))
		fmt.Fprintln(ctx.Stdout, color.Colorize("Usage: port <number> to inspect, killport <number> to kill", pal.GhostText))
		return 0
	}

	portNum, err := strconv.Atoi(args[1])
	if err != nil || portNum <= 0 || portNum > 65535 {
		fmt.Fprintf(ctx.Stderr, "bush: port: invalid port '%s'\n", args[1])
		return 1
	}

	info, err := findProcessOnPort(portNum)
	if err != nil {
		fmt.Fprintf(ctx.Stderr, "bush: port: %v\n", err)
		return 1
	}

	fmt.Fprintln(ctx.Stdout, color.BoldColorize(fmt.Sprintf("+--- Port Doctor (%d) ------------------------------------+", portNum), pal.Accent))
	fmt.Fprintf(ctx.Stdout, "| %-12s: %s\n", color.Colorize("Port", pal.Flags), color.BoldColorize(strconv.Itoa(info.Port), pal.Directory))
	fmt.Fprintf(ctx.Stdout, "| %-12s: %s\n", color.Colorize("Protocol", pal.Flags), info.Proto)
	fmt.Fprintf(ctx.Stdout, "| %-12s: %s\n", color.Colorize("Status", pal.Flags), color.Colorize(info.State, pal.Success))
	if info.PID > 0 {
		fmt.Fprintf(ctx.Stdout, "| %-12s: %s (PID: %d)\n", color.Colorize("Process", pal.Flags), color.BoldColorize(info.Process, pal.PromptSymbol), info.PID)
		if info.Cmdline != "" {
			fmt.Fprintf(ctx.Stdout, "| %-12s: %s\n", color.Colorize("Command", pal.Flags), info.Cmdline)
		}
	} else {
		fmt.Fprintf(ctx.Stdout, "| %-12s: %s\n", color.Colorize("Process", pal.Flags), info.Process)
	}
	fmt.Fprintln(ctx.Stdout, color.BoldColorize("+----------------------------------------------------------+", pal.Accent))
	return 0
}

func builtinKillport(args []string, ctx *ShellContext) int {
	pal := color.ActivePalette

	if len(args) < 2 {
		fmt.Fprintf(ctx.Stderr, "bush: killport: usage: killport <port>\n")
		return 1
	}

	portNum, err := strconv.Atoi(args[1])
	if err != nil || portNum <= 0 || portNum > 65535 {
		fmt.Fprintf(ctx.Stderr, "bush: killport: invalid port '%s'\n", args[1])
		return 1
	}

	info, err := findProcessOnPort(portNum)
	if err != nil {
		fmt.Fprintf(ctx.Stderr, "bush: killport: %v\n", err)
		return 1
	}

	if info.PID <= 0 {
		fmt.Fprintf(ctx.Stderr, "bush: killport: unable to determine PID for port %d (try running with sudo)\n", portNum)
		return 1
	}

	// Try graceful SIGTERM first
	err = syscall.Kill(info.PID, syscall.SIGTERM)
	if err != nil {
		fmt.Fprintf(ctx.Stderr, "bush: killport: failed to signal PID %d: %v\n", info.PID, err)
		return 1
	}

	// Wait up to 1 second for termination
	freed := false
	for i := 0; i < 10; i++ {
		time.Sleep(100 * time.Millisecond)
		if _, checkErr := findProcessOnPort(portNum); checkErr != nil {
			freed = true
			break
		}
	}

	// Force SIGKILL if still holding port
	if !freed {
		_ = syscall.Kill(info.PID, syscall.SIGKILL)
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Fprintf(ctx.Stdout, "%s Released port %s (terminated %s PID %d)\n",
		color.BoldColorize("ok", pal.Success),
		color.BoldColorize(strconv.Itoa(portNum), pal.Directory),
		color.Colorize(info.Process, pal.Flags),
		info.PID,
	)
	return 0
}
