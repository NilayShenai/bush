package guard

import (
	"bufio"
	"bush/internal/color"
	"bush/internal/config"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type BlastRadiusAlert struct {
	Triggered bool
	Reason    string
	Files     int
	Target    string
}

func CheckBlastRadius(cmd string, args []string) *BlastRadiusAlert {
	cfg := config.Get()
	if !cfg.Guard.Enabled {
		return nil
	}

	thresh := cfg.Guard.FileThreshold
	if thresh <= 0 {
		thresh = 10
	}

	cmdBase := filepath.Base(cmd)

	if cmdBase == "rm" {
		hasRecursive := false
		hasForce := false
		var targets []string

		for _, arg := range args {
			if strings.HasPrefix(arg, "-") {
				if strings.Contains(arg, "r") || strings.Contains(arg, "R") {
					hasRecursive = true
				}
				if strings.Contains(arg, "f") {
					hasForce = true
				}
			} else {
				targets = append(targets, arg)
			}
		}

		if hasRecursive || hasForce {
			for _, t := range targets {
				clean := filepath.Clean(t)
				if clean == "/" || clean == "/*" || clean == "*" || clean == "." || clean == ".." || clean == os.Getenv("HOME") {
					return &BlastRadiusAlert{
						Triggered: true,
						Reason:    "Target is root, wildcard, or home directory",
						Files:     countFilesInDir(t),
						Target:    t,
					}
				}

				count := countFilesInDir(t)
				if count >= thresh {
					return &BlastRadiusAlert{
						Triggered: true,
						Reason:    fmt.Sprintf("Directory contains %d files (threshold: %d)", count, thresh),
						Files:     count,
						Target:    t,
					}
				}
			}
		}
	}

	if cmdBase == "git" && len(args) >= 2 {
		if args[0] == "reset" && len(args) >= 2 && args[1] == "--hard" {
			return &BlastRadiusAlert{
				Triggered: true,
				Reason:    "Hard reset will discard all uncommitted working tree and staged changes",
				Target:    "current repository",
			}
		}

		if args[0] == "push" {
			for _, a := range args[1:] {
				if a == "--force" || a == "-f" || strings.HasPrefix(a, "+") {
					return &BlastRadiusAlert{
						Triggered: true,
						Reason:    "Force-push may overwrite remote history for collaborators",
						Target:    "remote repository",
					}
				}
			}
		}
	}

	if strings.HasPrefix(cmdBase, "mkfs") {
		return &BlastRadiusAlert{
			Triggered: true,
			Reason:    "Formatting filesystem will erase all existing partition data",
			Target:    strings.Join(args, " "),
		}
	}

	if cmdBase == "dd" {
		for _, a := range args {
			if strings.HasPrefix(a, "of=/dev/sd") || strings.HasPrefix(a, "of=/dev/nvme") || strings.HasPrefix(a, "of=/dev/vd") {
				return &BlastRadiusAlert{
					Triggered: true,
					Reason:    "Direct raw block write targeting storage device",
					Target:    a,
				}
			}
		}
	}

	if cmdBase == "chmod" {
		hasRecursive := false
		for _, a := range args {
			if strings.Contains(a, "R") {
				hasRecursive = true
			}
			if hasRecursive && (a == "/" || a == "/*" || a == "/etc" || a == "/usr") {
				return &BlastRadiusAlert{
					Triggered: true,
					Reason:    "Recursive permission change on core system directory",
					Target:    a,
				}
			}
		}
	}

	return nil
}

func ConfirmExecution(alert *BlastRadiusAlert, fullCmd string) bool {
	pal := color.ActivePalette

	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, color.BoldColorize("+--- [!] Blast Radius Guard -------------------------------+", pal.Error))
	fmt.Fprintf(os.Stderr, "| %s: %s\n", color.Colorize("Command", pal.Flags), color.BoldColorize(fullCmd, pal.Error))
	fmt.Fprintf(os.Stderr, "| %s: %s\n", color.Colorize("Reason", pal.Flags), alert.Reason)
	if alert.Files > 0 {
		fmt.Fprintf(os.Stderr, "| %s: %d files\n", color.Colorize("Affected", pal.Flags), alert.Files)
	}
	fmt.Fprintf(os.Stderr, "| %s: %s\n", color.Colorize("Target", pal.Flags), alert.Target)
	fmt.Fprintln(os.Stderr, color.BoldColorize("+----------------------------------------------------------+", pal.Error))
	fmt.Fprint(os.Stderr, color.BoldColorize("Type 'yes' or press Enter to proceed, or 'no'/Esc to cancel: ", pal.PromptSymbol))

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	trimmed := strings.TrimSpace(strings.ToLower(response))
	if trimmed == "" || trimmed == "y" || trimmed == "yes" {
		return true
	}

	fmt.Fprintln(os.Stderr, color.Colorize("Command aborted by Blast Radius Guard.", pal.Success))
	return false
}

func countFilesInDir(dir string) int {
	fi, err := os.Stat(dir)
	if err != nil {
		return 0
	}
	if !fi.IsDir() {
		return 1
	}

	count := 0
	_ = filepath.Walk(dir, func(_ string, info os.FileInfo, err error) error {
		if err == nil && info != nil && !info.IsDir() {
			count++
		}
		if count > 1000 {
			return filepath.SkipDir
		}
		return nil
	})
	return count
}
