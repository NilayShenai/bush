package builtins

import (
	"bush/internal/color"
	"bush/internal/terminal"
	"fmt"
	"os"
	"strings"
	"time"
)

func builtinWatch(args []string, ctx *ShellContext) int {
	pal := color.ActivePalette

	if len(args) < 2 {
		fmt.Fprintf(ctx.Stderr, "bush: watch: usage: watch [interval] <command>\n")
		fmt.Fprintf(ctx.Stderr, "example: watch 2s \"docker ps\" or watch \"cat /proc/loadavg\"\n")
		return 1
	}

	interval := 2 * time.Second
	cmdToRun := ""

	if dur, err := time.ParseDuration(args[1]); err == nil {
		interval = dur
		if len(args) < 3 {
			fmt.Fprintf(ctx.Stderr, "bush: watch: missing command to execute\n")
			return 1
		}
		cmdToRun = strings.Join(args[2:], " ")
	} else {
		cmdToRun = strings.Join(args[1:], " ")
	}

	if ctx.SubshellRunner == nil {
		fmt.Fprintf(ctx.Stderr, "bush: watch: subshell runner unavailable\n")
		return 1
	}

	fd := terminal.StdinFd()
	isTerm := terminal.IsTerminal(fd)

	if isTerm {
		_, _ = terminal.MakeRaw(fd)
		defer terminal.Restore(fd)
	}

	stopChan := make(chan struct{})
	go func() {
		for {
			key, err := terminal.ReadKey(os.Stdin)
			if err != nil {
				return
			}
			if key.Type == terminal.KeyCtrlC || key.Type == terminal.KeyCtrlD || (key.Type == terminal.KeyRune && (key.Rune == 'q' || key.Rune == 'Q')) {
				close(stopChan)
				return
			}
		}
	}()

	var prevLines []string
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	runOnce := func() {
		out, err := ctx.SubshellRunner(cmdToRun)
		currentLines := strings.Split(strings.TrimRight(out, "\n"), "\n")

		// Clear screen
		fmt.Print("\033[H\033[2J")

		// Header
		header := fmt.Sprintf("Every %v: %s", interval, cmdToRun)
		nowStr := time.Now().Format("15:04:05")
		fmt.Printf("%s   %s   %s\r\n",
			color.BoldColorize(header, pal.Accent),
			color.Colorize("(press 'q' to quit)", pal.GhostText),
			color.Colorize(nowStr, pal.Flags),
		)
		fmt.Printf("%s\r\n", color.Colorize(strings.Repeat("-", 60), pal.Border))

		if err != nil {
			fmt.Printf("%s\r\n", color.Colorize(fmt.Sprintf("Error: %v", err), pal.Error))
		}

		for i, line := range currentLines {
			isDiff := true
			if i < len(prevLines) && prevLines[i] == line {
				isDiff = false
			}

			if isDiff && len(prevLines) > 0 {
				// Highlight changed line
				fmt.Printf("%s\r\n", color.BoldColorize(line, pal.Directory))
			} else {
				fmt.Printf("%s\r\n", line)
			}
		}

		prevLines = currentLines
	}

	runOnce()

	for {
		select {
		case <-stopChan:
			fmt.Print("\r\n")
			return 0
		case <-ticker.C:
			runOnce()
		}
	}
}
