package builtins

import (
	"bush/internal/color"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	githubRepo = "NilayShenai/bush"
	apiURL     = "https://api.github.com/repos/" + githubRepo + "/releases/latest"
	installURL = "https://raw.githubusercontent.com/" + githubRepo + "/main/install.sh"
)

func init() {
	Register("update", builtinUpdate)
}

type githubRelease struct {
	TagName string `json:"tag_name"`
	HtmlURL string `json:"html_url"`
}

type updateCache struct {
	LastChecked int64  `json:"last_checked"`
	LatestTag   string `json:"latest_tag"`
}

func fetchLatestRelease() (string, string, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("User-Agent", "bush-shell-updater")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("GitHub API returned HTTP %d", resp.StatusCode)
	}

	var rel githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", "", err
	}

	return rel.TagName, rel.HtmlURL, nil
}

func parseSemver(v string) [3]int {
	v = strings.TrimPrefix(v, "v")
	parts := strings.Split(v, ".")
	var res [3]int
	for i := 0; i < len(parts) && i < 3; i++ {
		clean := strings.Split(parts[i], "-")[0]
		n, _ := strconv.Atoi(clean)
		res[i] = n
	}
	return res
}

func isNewerVersion(current, latest string) bool {
	c := parseSemver(current)
	l := parseSemver(latest)
	for i := 0; i < 3; i++ {
		if l[i] > c[i] {
			return true
		}
		if l[i] < c[i] {
			return false
		}
	}
	return false
}

func builtinUpdate(args []string, ctx *ShellContext) int {
	pal := color.ActivePalette
	current := ctx.Version
	if current == "" {
		current = "2.8.3"
	}

	isCheckOnly := len(args) > 1 && args[1] == "check"
	isForce := len(args) > 1 && args[1] == "force"

	fmt.Fprintln(ctx.Stdout, color.BoldColorize("+--- Bush Auto-Updater -----------------------------------+", pal.Accent))
	fmt.Fprintf(ctx.Stdout, "| Checking latest release on GitHub (%s)...\n", githubRepo)

	latestTag, htmlURL, err := fetchLatestRelease()
	if err != nil {
		fmt.Fprintf(ctx.Stderr, "| Error: %v\n", err)
		fmt.Fprintln(ctx.Stderr, color.BoldColorize("+---------------------------------------------------------+", pal.Accent))
		return 1
	}

	latestClean := strings.TrimPrefix(latestTag, "v")
	currentClean := strings.TrimPrefix(current, "v")

	fmt.Fprintf(ctx.Stdout, "| %-18s: %s\n", color.Colorize("Current Version", pal.Flags), color.Colorize(currentClean, pal.Directory))
	fmt.Fprintf(ctx.Stdout, "| %-18s: %s\n", color.Colorize("Latest Version", pal.Flags), color.BoldColorize(latestClean, pal.Success))
	if htmlURL != "" {
		fmt.Fprintf(ctx.Stdout, "| %-18s: %s\n", color.Colorize("Release Notes", pal.Flags), color.Colorize(htmlURL, pal.PromptSymbol))
	}
	fmt.Fprintln(ctx.Stdout, color.BoldColorize("+---------------------------------------------------------+", pal.Accent))

	hasUpdate := isNewerVersion(currentClean, latestClean)
	if !hasUpdate && !isForce {
		fmt.Fprintf(ctx.Stdout, "%s Bush is already up to date (%s).\n", color.BoldColorize("ok", pal.Success), currentClean)
		return 0
	}

	if isCheckOnly {
		fmt.Fprintf(ctx.Stdout, "%s A new version of Bush is available (%s -> %s).\nRun 'update' to install it.\n",
			color.BoldColorize("notice", pal.PromptSymbol), currentClean, latestClean)
		return 0
	}

	fmt.Fprintf(ctx.Stdout, "\n==> Upgrading Bush from %s to %s...\n", currentClean, latestTag)

	// Execute install script via bash
	cmd := exec.Command("bash", "-c", "curl -fsSL "+installURL+" | bash")
	cmd.Stdin = ctx.Stdin
	cmd.Stdout = ctx.Stdout
	cmd.Stderr = ctx.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(ctx.Stderr, "bush: update failed: %v\n", err)
		return 1
	}

	fmt.Fprintf(ctx.Stdout, "\n%s Successfully updated Bush to %s! Restart your shell to use the new version.\n",
		color.BoldColorize("ok", pal.Success), latestTag)
	return 0
}

func updateCacheFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".bush_update_cache"
	}
	return filepath.Join(home, ".bush_update_cache")
}

func CheckStartupUpdateNotice(currentVersion string) string {
	cacheFile := updateCacheFilePath()
	now := time.Now().Unix()

	var cache updateCache
	if data, err := os.ReadFile(cacheFile); err == nil {
		_ = json.Unmarshal(data, &cache)
	}

	// Check at most once every 24 hours
	if now-cache.LastChecked > 86400 {
		latestTag, _, err := fetchLatestRelease()
		if err == nil && latestTag != "" {
			cache.LastChecked = now
			cache.LatestTag = latestTag
			if data, err := json.Marshal(cache); err == nil {
				_ = os.WriteFile(cacheFile, data, 0644)
			}
		}
	}

	if cache.LatestTag != "" && isNewerVersion(currentVersion, cache.LatestTag) {
		pal := color.ActivePalette
		return fmt.Sprintf("  %s %s (%s -> %s). Run '%s' to upgrade.",
			color.BoldColorize("Tip:", pal.PromptSymbol),
			color.Colorize("A new release of Bush is available", pal.GhostText),
			color.Colorize(currentVersion, pal.Flags),
			color.BoldColorize(cache.LatestTag, pal.Success),
			color.BoldColorize("update", pal.Accent),
		)
	}

	return ""
}
