package builtins

import (
	"bush/internal/color"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

func builtinG(args []string, ctx *ShellContext) int {
	pal := color.ActivePalette

	if len(args) > 1 {
		// Passthrough to git
		cmd := exec.Command("git", args[1:]...)
		cmd.Stdin = ctx.Stdin
		cmd.Stdout = ctx.Stdout
		cmd.Stderr = ctx.Stderr
		err := cmd.Run()
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				return exitErr.ExitCode()
			}
			return 1
		}
		return 0
	}

	// Compact Git HUD
	out, err := exec.Command("git", "rev-parse", "--is-inside-work-tree").Output()
	if err != nil || strings.TrimSpace(string(out)) != "true" {
		fmt.Fprintln(ctx.Stderr, "bush: g: not a git repository (or any of the parent directories)")
		return 1
	}

	branchOut, _ := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	branch := strings.TrimSpace(string(branchOut))

	// Ahead/Behind count
	aheadBehind := ""
	abOut, err := exec.Command("git", "rev-list", "--left-right", "--count", "HEAD...@{u}").Output()
	if err == nil {
		parts := strings.Fields(string(abOut))
		if len(parts) == 2 {
			ahead, _ := strconv.Atoi(parts[0])
			behind, _ := strconv.Atoi(parts[1])
			if ahead > 0 && behind > 0 {
				aheadBehind = fmt.Sprintf(" [ahead %d, behind %d]", ahead, behind)
			} else if ahead > 0 {
				aheadBehind = fmt.Sprintf(" [ahead %d]", ahead)
			} else if behind > 0 {
				aheadBehind = fmt.Sprintf(" [behind %d]", behind)
			}
		}
	}

	// Status counts
	statusOut, _ := exec.Command("git", "status", "--porcelain").Output()
	stagedCount := 0
	modifiedCount := 0
	untrackedCount := 0

	for _, line := range strings.Split(string(statusOut), "\n") {
		if len(line) < 2 {
			continue
		}
		x := line[0]
		y := line[1]

		if x == '?' && y == '?' {
			untrackedCount++
		} else {
			if x != ' ' && x != '?' {
				stagedCount++
			}
			if y != ' ' && y != '?' {
				modifiedCount++
			}
		}
	}

	// Latest commit
	logOut, _ := exec.Command("git", "log", "-1", "--pretty=format:%h|%an|%cr|%s").Output()
	logParts := strings.Split(string(logOut), "|")

	fmt.Fprintln(ctx.Stdout, color.BoldColorize("+--- Git Status HUD ---------------------------------------+", pal.Accent))
	fmt.Fprintf(ctx.Stdout, "| %s: %s%s\n",
		color.Colorize("Branch", pal.Flags),
		color.BoldColorize(branch, pal.PromptSymbol),
		color.Colorize(aheadBehind, pal.Directory),
	)

	fmt.Fprintf(ctx.Stdout, "| %s: %s | %s: %s | %s: %s\n",
		color.Colorize("Staged", pal.Flags), color.BoldColorize(strconv.Itoa(stagedCount), pal.Success),
		color.Colorize("Modified", pal.Flags), color.BoldColorize(strconv.Itoa(modifiedCount), pal.Duration),
		color.Colorize("Untracked", pal.Flags), color.Colorize(strconv.Itoa(untrackedCount), pal.GhostText),
	)

	if len(logParts) == 4 {
		fmt.Fprintf(ctx.Stdout, "| %s: %s by %s (%s)\n",
			color.Colorize("Latest", pal.Flags),
			color.Colorize(logParts[0], pal.Directory),
			color.Colorize(logParts[1], pal.PromptSymbol),
			color.Colorize(logParts[2], pal.GhostText),
		)
		fmt.Fprintf(ctx.Stdout, "|         \"%s\"\n", logParts[3])
	}

	fmt.Fprintln(ctx.Stdout, color.BoldColorize("+----------------------------------------------------------+", pal.Accent))
	return 0
}
