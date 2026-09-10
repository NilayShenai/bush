package builtins

import (
	"bush/internal/color"
	"fmt"
	"strings"
	"time"
)

var (
	memoizedOutput string
	memoizedCmd    string
	memoizedTime   time.Time
)

func builtinMemo(args []string, ctx *ShellContext) int {
	pal := color.ActivePalette

	if len(args) == 1 {
		if memoizedOutput == "" {
			fmt.Fprintln(ctx.Stderr, "bush: memo: no command output is currently memoized")
			fmt.Fprintln(ctx.Stderr, "usage: memo <command> (e.g. memo curl api.com) then replay with 'memo' or 'memo | grep ...'")
			return 1
		}

		// Replay memoized output directly to stdout
		fmt.Fprint(ctx.Stdout, memoizedOutput)
		return 0
	}

	sub := args[1]
	if sub == "clear" {
		memoizedOutput = ""
		memoizedCmd = ""
		memoizedTime = time.Time{}
		fmt.Fprintln(ctx.Stdout, color.Colorize("Memoized output cache cleared.", pal.Success))
		return 0
	}

	if sub == "info" {
		if memoizedOutput == "" {
			fmt.Fprintln(ctx.Stdout, color.Colorize("Memo cache is empty.", pal.GhostText))
			return 0
		}
		lines := len(strings.Split(memoizedOutput, "\n"))
		bytes := len(memoizedOutput)
		elapsed := time.Since(memoizedTime).Round(time.Second)

		fmt.Fprintln(ctx.Stdout, color.BoldColorize("+--- Memoized Cache Info ----------------------------------+", pal.Accent))
		fmt.Fprintf(ctx.Stdout, "| %-12s: %s\n", color.Colorize("Command", pal.Flags), memoizedCmd)
		fmt.Fprintf(ctx.Stdout, "| %-12s: %d bytes (%d lines)\n", color.Colorize("Size", pal.Flags), bytes, lines)
		fmt.Fprintf(ctx.Stdout, "| %-12s: %v ago (%s)\n", color.Colorize("Cached", pal.Flags), elapsed, memoizedTime.Format("15:04:05"))
		fmt.Fprintln(ctx.Stdout, color.BoldColorize("+----------------------------------------------------------+", pal.Accent))
		return 0
	}

	// Run command and memoize output
	cmdToRun := strings.Join(args[1:], " ")
	if ctx.SubshellRunner == nil {
		fmt.Fprintf(ctx.Stderr, "bush: memo: subshell runner unavailable\n")
		return 1
	}

	out, err := ctx.SubshellRunner(cmdToRun)
	if err != nil {
		fmt.Fprintf(ctx.Stderr, "bush: memo: %v\n", err)
		return 1
	}

	memoizedOutput = out
	memoizedCmd = cmdToRun
	memoizedTime = time.Now()

	// Print output to terminal
	fmt.Fprint(ctx.Stdout, out)
	return 0
}
