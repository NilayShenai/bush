package prompt

import (
	"bush/internal/color"
	"bush/internal/config"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"time"
)

type State struct {
	LastExitCode int
	LastDuration time.Duration
}

func Render(s *State) (string, string) {
	cfg := config.Get()
	pal := color.ActivePalette

	var sb strings.Builder

	isRoot := os.Geteuid() == 0

	for _, badge := range cfg.Prompt.Badges {
		switch badge {
		case "badge":
			badgeText := " bush "
			badgeColor := pal.Accent
			if isRoot {
				badgeText = " root "
				badgeColor = pal.Error
			}
			sb.WriteString(color.ColorizeBg(badgeText, color.SurfaceDark, badgeColor))
		case "user":
			if !isRoot {
				u := getCurrentUser()
				sb.WriteString(color.ColorizeBg(" "+u+" ", color.SurfaceDark, pal.User))
			}
		case "user_host":
			if !isRoot {
				u := getCurrentUser()
				h, _ := os.Hostname()
				if h != "" {
					h = strings.Split(h, ".")[0]
					u = u + "@" + h
				}
				sb.WriteString(color.ColorizeBg(" "+u+" ", color.SurfaceDark, pal.User))
			}
		case "dir":
			cwd, err := os.Getwd()
			if err != nil {
				cwd = "?"
			}
			formattedDir := formatCwd(cwd, cfg.Prompt.MaxDirDepth)
			dirColor := pal.Directory
			if isRoot {
				dirColor = pal.Error
			}
			sb.WriteString(color.ColorizeBg(" "+formattedDir+" ", color.SurfaceDark, dirColor))
		case "git":
			if cfg.Prompt.ShowGit {
				gitInfo := getGitStatus(cfg.Prompt.ShowGitDirty)
				if gitInfo != "" {
					sb.WriteString(color.ColorizeBg(" "+gitInfo+" ", color.SurfaceDark, pal.GitBranch))
				}
			}
		case "duration":
			thresh := time.Duration(cfg.Prompt.DurationThresholdMs) * time.Millisecond
			if thresh <= 0 {
				thresh = 5 * time.Millisecond
			}
			if s != nil && s.LastDuration >= thresh {
				durStr := fmt.Sprintf(" [%v] ", s.LastDuration.Round(time.Millisecond))
				sb.WriteString(color.ColorizeBg(durStr, color.SurfaceDark, pal.Duration))
			}
		case "status":
			if s != nil {
				if s.LastExitCode == 0 {
					statusColor := pal.Success
					if isRoot {
						statusColor = pal.Error
					}
					sb.WriteString(color.ColorizeBg(" ok ", color.SurfaceDark, statusColor))
				} else {
					sb.WriteString(color.ColorizeBg(fmt.Sprintf(" err:%d ", s.LastExitCode), color.SurfaceDark, pal.Error))
				}
			}
		}
	}

	sym := cfg.Prompt.Symbol
	if sym == "" {
		sym = "❯ "
	}
	symColor := pal.PromptSymbol
	if isRoot {
		symColor = pal.Error
	}
	promptSymbol := color.BoldColorize(sym, symColor)

	if !cfg.Prompt.Multiline {

		if sb.Len() > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(promptSymbol)
		return "", sb.String()
	}

	return sb.String(), promptSymbol
}

func formatCwd(dir string, maxDepth int) string {
	home, err := os.UserHomeDir()
	if err == nil && strings.HasPrefix(dir, home) {
		dir = "~" + dir[len(home):]
	}
	if maxDepth <= 0 {
		maxDepth = 4
	}
	parts := strings.Split(dir, string(filepath.Separator))
	if len(parts) > maxDepth {
		return parts[0] + "/.../" + strings.Join(parts[len(parts)-2:], "/")
	}
	return dir
}

func getGitStatus(showDirty bool) string {
	out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return ""
	}
	branch := strings.TrimSpace(string(out))
	if branch == "" {
		return ""
	}

	dirty := ""
	if showDirty {
		diffOut, err := exec.Command("git", "status", "--porcelain").Output()
		if err == nil && len(diffOut) > 0 {
			dirty = "*"
		}
	}

	return fmt.Sprintf("git:%s%s", branch, dirty)
}

func getCurrentUser() string {
	if u := os.Getenv("USER"); u != "" {
		return u
	}
	if u := os.Getenv("LOGNAME"); u != "" {
		return u
	}
	if usr, err := user.Current(); err == nil && usr.Username != "" {
		return usr.Username
	}
	return "user"
}
