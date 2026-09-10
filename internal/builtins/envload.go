package builtins

import (
	"bufio"
	"bush/internal/color"
	"fmt"
	"os"
	"strings"
)

func builtinEnvload(args []string, ctx *ShellContext) int {
	pal := color.ActivePalette
	targetFile := ".env"

	if len(args) > 1 {
		targetFile = args[1]
	} else {
		if _, err := os.Stat(targetFile); os.IsNotExist(err) {
			for _, candidate := range []string{".env.local", ".env.development", ".env.prod"} {
				if _, err := os.Stat(candidate); err == nil {
					targetFile = candidate
					break
				}
			}
		}
	}

	f, err := os.Open(targetFile)
	if err != nil {
		fmt.Fprintf(ctx.Stderr, "bush: envload: %v\n", err)
		return 1
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	loadedCount := 0
	sensitiveCount := 0

	fmt.Fprintln(ctx.Stdout, color.BoldColorize(fmt.Sprintf("+--- Environment Loader (%s) -------------------+", targetFile), pal.Accent))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		line = strings.TrimPrefix(line, "export ")
		idx := strings.Index(line, "=")
		if idx == -1 {
			continue
		}

		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])

		if strings.HasPrefix(val, "\"") {
			endQuote := strings.Index(val[1:], "\"")
			if endQuote != -1 {
				val = val[1 : endQuote+1]
			}
		} else if strings.HasPrefix(val, "'") {
			endQuote := strings.Index(val[1:], "'")
			if endQuote != -1 {
				val = val[1 : endQuote+1]
			}
		} else {
			if hashIdx := strings.Index(val, " #"); hashIdx != -1 {
				val = strings.TrimSpace(val[:hashIdx])
			}
		}

		os.Setenv(key, val)
		loadedCount++

		isSensitive := isSensitiveKey(key)
		displayVal := val
		if isSensitive {
			sensitiveCount++
			displayVal = "********"
		} else if len(displayVal) > 30 {
			displayVal = displayVal[:27] + "..."
		}

		fmt.Fprintf(ctx.Stdout, "| %-22s = %s\n",
			color.Colorize(key, pal.Directory),
			color.Colorize(displayVal, pal.Flags),
		)
	}

	fmt.Fprintln(ctx.Stdout, color.BoldColorize("+----------------------------------------------------------+", pal.Accent))
	fmt.Fprintf(ctx.Stdout, "%s Loaded %d environment variables from %s (%d sensitive keys masked)\n",
		color.BoldColorize("ok", pal.Success),
		loadedCount,
		targetFile,
		sensitiveCount,
	)

	return 0
}

func isSensitiveKey(key string) bool {
	upper := strings.ToUpper(key)
	return strings.Contains(upper, "SECRET") ||
		strings.Contains(upper, "KEY") ||
		strings.Contains(upper, "PASSWORD") ||
		strings.Contains(upper, "TOKEN") ||
		strings.Contains(upper, "AUTH") ||
		strings.Contains(upper, "PRIVATE")
}
