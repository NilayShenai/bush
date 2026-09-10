package builtins

import (
	"bufio"
	"bush/internal/color"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type FrecencyEntry struct {
	Path        string
	Count       int
	LastVisited int64
}

func (e *FrecencyEntry) Score() float64 {
	age := time.Since(time.Unix(e.LastVisited, 0))
	weight := 0.5
	if age < time.Hour {
		weight = 4.0
	} else if age < 24*time.Hour {
		weight = 2.0
	} else if age < 7*24*time.Hour {
		weight = 1.0
	}
	return float64(e.Count) * weight
}

func frecencyFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".bush_frecency"
	}
	return filepath.Join(home, ".bush_frecency")
}

func loadFrecency() []*FrecencyEntry {
	path := frecencyFilePath()
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var entries []*FrecencyEntry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) >= 3 {
			cnt, _ := strconv.Atoi(parts[1])
			ts, _ := strconv.ParseInt(parts[2], 10, 64)
			entries = append(entries, &FrecencyEntry{
				Path:        parts[0],
				Count:       cnt,
				LastVisited: ts,
			})
		}
	}
	return entries
}

func saveFrecency(entries []*FrecencyEntry) {
	path := frecencyFilePath()
	var sb strings.Builder
	for _, e := range entries {
		sb.WriteString(fmt.Sprintf("%s\t%d\t%d\n", e.Path, e.Count, e.LastVisited))
	}
	_ = os.WriteFile(path, []byte(sb.String()), 0644)
}

func RecordDirectoryVisit(dir string) {
	cleanDir, err := filepath.Abs(dir)
	if err != nil {
		return
	}

	entries := loadFrecency()
	found := false
	for _, e := range entries {
		if e.Path == cleanDir {
			e.Count++
			e.LastVisited = time.Now().Unix()
			found = true
			break
		}
	}

	if !found {
		entries = append(entries, &FrecencyEntry{
			Path:        cleanDir,
			Count:       1,
			LastVisited: time.Now().Unix(),
		})
	}

	saveFrecency(entries)
}

func builtinZ(args []string, ctx *ShellContext) int {
	pal := color.ActivePalette
	entries := loadFrecency()

	if len(args) == 1 {
		if len(entries) == 0 {
			fmt.Fprintln(ctx.Stdout, color.Colorize("No frecent directories recorded yet.", pal.GhostText))
			return 0
		}

		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Score() > entries[j].Score()
		})

		fmt.Fprintln(ctx.Stdout, color.BoldColorize("+--- Frecency Jump (z) ------------------------------------+", pal.Accent))
		limit := len(entries)
		if limit > 10 {
			limit = 10
		}
		for i := 0; i < limit; i++ {
			e := entries[i]
			fmt.Fprintf(ctx.Stdout, "| %2d. %-42s (score: %.1f, visits: %d)\n",
				i+1,
				color.Colorize(shortenPath(e.Path), pal.Directory),
				e.Score(),
				e.Count,
			)
		}
		fmt.Fprintln(ctx.Stdout, color.BoldColorize("+----------------------------------------------------------+", pal.Accent))
		fmt.Fprintln(ctx.Stdout, color.Colorize("Usage: z <query> to teleport to directory", pal.GhostText))
		return 0
	}

	if args[1] == "--clear" || args[1] == "-c" {
		saveFrecency(nil)
		fmt.Fprintln(ctx.Stdout, color.Colorize("Frecency database cleared.", pal.Success))
		return 0
	}

	queries := args[1:]
	var matches []*FrecencyEntry

	for _, e := range entries {
		pathLower := strings.ToLower(e.Path)
		matchAll := true
		for _, q := range queries {
			if !strings.Contains(pathLower, strings.ToLower(q)) {
				matchAll = false
				break
			}
		}
		if matchAll {
			if _, err := os.Stat(e.Path); err == nil {
				matches = append(matches, e)
			}
		}
	}

	if len(matches) == 0 {
		fmt.Fprintf(ctx.Stderr, "bush: z: no matching directory found for '%s'\n", strings.Join(queries, " "))
		return 1
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score() > matches[j].Score()
	})

	target := matches[0].Path
	err := os.Chdir(target)
	if err != nil {
		fmt.Fprintf(ctx.Stderr, "bush: z: %v\n", err)
		return 1
	}

	RecordDirectoryVisit(target)
	fmt.Fprintln(ctx.Stdout, color.Colorize(shortenPath(target), pal.Directory))
	return 0
}

func shortenPath(path string) string {
	home, err := os.UserHomeDir()
	if err == nil && strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}
