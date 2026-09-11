package config

import (
	"bush/internal/builtins"
	"bush/internal/color"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

func init() {
	builtins.Register("config", builtinConfig)
}

type Config struct {
	Shell       ShellConfig        `toml:"shell"`
	Prompt      PromptConfig       `toml:"prompt"`
	Theme       ThemeConfig        `toml:"theme"`
	Autosuggest AutosuggestConfig  `toml:"autosuggest"`
	Guard       GuardConfig        `toml:"guard"`
	Aliases     map[string]string  `toml:"aliases"`
	Env         map[string]string  `toml:"env"`
}

type GuardConfig struct {
	Enabled       bool `toml:"enabled"`
	FileThreshold int  `toml:"file_threshold"`
}

type ShellConfig struct {
	GreetingBanner bool   `toml:"greeting_banner"`
	Motto          string `toml:"motto"`
	HistoryLimit   int    `toml:"history_limit"`
	CheckUpdates   bool   `toml:"check_updates"`
}

type PromptConfig struct {
	Multiline           bool     `toml:"multiline"`
	Symbol              string   `toml:"symbol"`
	Badges              []string `toml:"badges"`
	MaxDirDepth         int      `toml:"max_dir_depth"`
	DurationThresholdMs int      `toml:"duration_threshold_ms"`
	ShowGit             bool     `toml:"show_git"`
	ShowGitDirty        bool     `toml:"show_git_dirty"`
}

type ThemeConfig struct {
	Name   string      `toml:"name"`
	Colors ThemeColors `toml:"colors"`
}

type ThemeColors struct {
	Accent       string `toml:"accent"`
	User         string `toml:"user"`
	PromptSymbol string `toml:"prompt_symbol"`
	Directory    string `toml:"directory"`
	GitBranch    string `toml:"git_branch"`
	Duration     string `toml:"duration"`
	Success      string `toml:"success"`
	Error        string `toml:"error"`
	GhostText    string `toml:"ghost_text"`
	Command      string `toml:"command"`
	InvalidCmd   string `toml:"invalid_cmd"`
	Flags        string `toml:"flags"`
	Strings      string `toml:"strings"`
	Border       string `toml:"border"`
}

type AutosuggestConfig struct {
	Enabled            bool `toml:"enabled"`
	SuggestHistory     bool `toml:"suggest_history"`
	SuggestSubcommands bool `toml:"suggest_subcommands"`
	SuggestFiles       bool `toml:"suggest_files"`
	SuggestBinaries    bool `toml:"suggest_binaries"`
}

var currentConfig *Config

func DefaultConfig() *Config {
	return &Config{
		Shell: ShellConfig{
			GreetingBanner: true,
			Motto:          "A shell for people of refined taste.",
			HistoryLimit:   5000,
			CheckUpdates:   true,
		},
		Guard: GuardConfig{
			Enabled:       true,
			FileThreshold: 10,
		},
		Prompt: PromptConfig{
			Multiline:           true,
			Symbol:              "❯ ",
			Badges:              []string{"badge", "user", "dir", "git", "duration", "status"},
			MaxDirDepth:         4,
			DurationThresholdMs: 5,
			ShowGit:             true,
			ShowGitDirty:        true,
		},
		Theme: ThemeConfig{
			Name: "lavender",
			Colors: ThemeColors{
				Accent:       "#B4BEFE",
				User:         "#FAB387",
				PromptSymbol: "#CBA6F7",
				Directory:    "#94E2D5",
				GitBranch:    "#F5C2E7",
				Duration:     "#F9E2AF",
				Success:      "#A6E3A1",
				Error:        "#F38BA8",
				GhostText:    "#6C7086",
				Command:      "#B4BEFE",
				InvalidCmd:   "#F38BA8",
				Flags:        "#FAB387",
				Strings:      "#F9E2AF",
				Border:       "#6C7086",
			},
		},
		Autosuggest: AutosuggestConfig{
			Enabled:            true,
			SuggestHistory:     true,
			SuggestSubcommands: true,
			SuggestFiles:       true,
			SuggestBinaries:    true,
		},
		Aliases: map[string]string{
			"ll":    "ls -la --color=auto",
			"la":    "ls -A --color=auto",
			"l":     "ls -CF --color=auto",
			"gs":    "git status",
			"gp":    "git pull",
			"gpush": "git push",
			"gd":    "git diff",
			"..":    "cd ..",
			"...":   "cd ../..",
		},
		Env: map[string]string{
			"BUSH": "1",
		},
	}
}

func Get() *Config {
	if currentConfig == nil {
		currentConfig = DefaultConfig()
	}
	return currentConfig
}

func ConfigFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "config.toml"
	}

	dotFile := filepath.Join(home, ".bush.toml")
	if _, err := os.Stat(dotFile); err == nil {
		return dotFile
	}

	return filepath.Join(home, ".config", "bush", "config.toml")
}

func LoadConfig(ctx *builtins.ShellContext) *Config {
	cfg := DefaultConfig()
	path := ConfigFilePath()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		_ = os.MkdirAll(filepath.Dir(path), 0755)
		SaveConfig(path, cfg)
	} else {
		_, err := toml.DecodeFile(path, cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "bush: error reading %s: %v\n", path, err)
		}
	}

	currentConfig = cfg

	ApplyTheme(cfg.Theme)

	if ctx != nil {
		if ctx.Aliases == nil {
			ctx.Aliases = make(map[string]string)
		}
		for k, v := range cfg.Aliases {
			ctx.Aliases[k] = v
		}
	}

	for k, v := range cfg.Env {
		os.Setenv(k, v)
	}

	return cfg
}

func SaveConfig(path string, cfg *Config) error {
	var buf bytes.Buffer
	buf.WriteString("# ==============================================================================\n")
	buf.WriteString("# BUSH SHELL CONFIGURATION (config.toml)\n")
	buf.WriteString("# FOR BUSH LOVERS, BY BUSH LOVERS.\n")
	buf.WriteString("# \"A shell for people of refined taste.\"\n")
	buf.WriteString("# ==============================================================================\n\n")

	encoder := toml.NewEncoder(&buf)
	err := encoder.Encode(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0644)
}

func ApplyTheme(t ThemeConfig) {

	switch strings.ToLower(t.Name) {
	case "tokyo_night":
		color.ActivePalette = color.Palette{
			Accent:       HexToRGB("#7AA2F7"),
			User:         HexToRGB("#FF9E64"),
			PromptSymbol: HexToRGB("#BB9AF7"),
			Directory:    HexToRGB("#7DCFFF"),
			GitBranch:    HexToRGB("#F7768E"),
			Duration:     HexToRGB("#E0AF68"),
			Success:      HexToRGB("#9ECE6A"),
			Error:        HexToRGB("#F7768E"),
			GhostText:    HexToRGB("#565F89"),
			Command:      HexToRGB("#7AA2F7"),
			InvalidCmd:   HexToRGB("#F7768E"),
			Flags:        HexToRGB("#FF9E64"),
			Strings:      HexToRGB("#9ECE6A"),
			Border:       HexToRGB("#414868"),
		}
	case "nord":
		color.ActivePalette = color.Palette{
			Accent:       HexToRGB("#88C0D0"),
			User:         HexToRGB("#D08770"),
			PromptSymbol: HexToRGB("#B48EAD"),
			Directory:    HexToRGB("#81A1C1"),
			GitBranch:    HexToRGB("#D08770"),
			Duration:     HexToRGB("#EBCB8B"),
			Success:      HexToRGB("#A3BE8C"),
			Error:        HexToRGB("#BF616A"),
			GhostText:    HexToRGB("#4C566A"),
			Command:      HexToRGB("#88C0D0"),
			InvalidCmd:   HexToRGB("#BF616A"),
			Flags:        HexToRGB("#D08770"),
			Strings:      HexToRGB("#A3BE8C"),
			Border:       HexToRGB("#4C566A"),
		}
	case "rose_pine":
		color.ActivePalette = color.Palette{
			Accent:       HexToRGB("#C4A7E7"),
			User:         HexToRGB("#F6C177"),
			PromptSymbol: HexToRGB("#EB6F92"),
			Directory:    HexToRGB("#9CCFD8"),
			GitBranch:    HexToRGB("#F6C177"),
			Duration:     HexToRGB("#EA9A97"),
			Success:      HexToRGB("#31748F"),
			Error:        HexToRGB("#EB6F92"),
			GhostText:    HexToRGB("#6E6A86"),
			Command:      HexToRGB("#C4A7E7"),
			InvalidCmd:   HexToRGB("#EB6F92"),
			Flags:        HexToRGB("#F6C177"),
			Strings:      HexToRGB("#EA9A97"),
			Border:       HexToRGB("#524F67"),
		}
	default:
		color.ActivePalette = color.Palette{
			Accent:       HexToRGB("#B4BEFE"),
			User:         HexToRGB("#FAB387"),
			PromptSymbol: HexToRGB("#CBA6F7"),
			Directory:    HexToRGB("#94E2D5"),
			GitBranch:    HexToRGB("#F5C2E7"),
			Duration:     HexToRGB("#F9E2AF"),
			Success:      HexToRGB("#A6E3A1"),
			Error:        HexToRGB("#F38BA8"),
			GhostText:    HexToRGB("#6C7086"),
			Command:      HexToRGB("#B4BEFE"),
			InvalidCmd:   HexToRGB("#F38BA8"),
			Flags:        HexToRGB("#FAB387"),
			Strings:      HexToRGB("#F9E2AF"),
			Border:       HexToRGB("#45475A"),
		}
	}

	overrideColor(&color.ActivePalette.Accent, t.Colors.Accent)
	overrideColor(&color.ActivePalette.User, t.Colors.User)
	overrideColor(&color.ActivePalette.PromptSymbol, t.Colors.PromptSymbol)
	overrideColor(&color.ActivePalette.Directory, t.Colors.Directory)
	overrideColor(&color.ActivePalette.GitBranch, t.Colors.GitBranch)
	overrideColor(&color.ActivePalette.Duration, t.Colors.Duration)
	overrideColor(&color.ActivePalette.Success, t.Colors.Success)
	overrideColor(&color.ActivePalette.Error, t.Colors.Error)
	overrideColor(&color.ActivePalette.GhostText, t.Colors.GhostText)
	overrideColor(&color.ActivePalette.Command, t.Colors.Command)
	overrideColor(&color.ActivePalette.InvalidCmd, t.Colors.InvalidCmd)
	overrideColor(&color.ActivePalette.Flags, t.Colors.Flags)
	overrideColor(&color.ActivePalette.Strings, t.Colors.Strings)
	overrideColor(&color.ActivePalette.Border, t.Colors.Border)
}

func overrideColor(dst *color.RGB, hex string) {
	if hex != "" {
		*dst = HexToRGB(hex)
	}
}

func HexToRGB(hex string) color.RGB {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) == 3 {
		hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
	}
	if len(hex) != 6 {
		return color.RGB{180, 190, 254}
	}

	r, _ := strconv.ParseUint(hex[0:2], 16, 8)
	g, _ := strconv.ParseUint(hex[2:4], 16, 8)
	b, _ := strconv.ParseUint(hex[4:6], 16, 8)

	return color.RGB{R: uint8(r), G: uint8(g), B: uint8(b)}
}

func builtinConfig(args []string, ctx *builtins.ShellContext) int {
	var out io.Writer = os.Stdout
	if ctx != nil && ctx.Stdout != nil {
		out = ctx.Stdout
	}
	path := ConfigFilePath()
	if len(args) > 1 {
		switch args[1] {
		case "path":
			fmt.Fprintln(out, path)
			return 0
		case "reload":
			LoadConfig(ctx)
			fmt.Fprintln(out, color.Colorize("Configuration reloaded from "+path, color.ActivePalette.Success))
			return 0
		case "edit":
			editor := os.Getenv("EDITOR")
			if editor == "" {
				editor = "nano"
			}
			cmd := exec.Command(editor, path)
			if ctx != nil {
				cmd.Stdin = ctx.Stdin
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
			}
			_ = cmd.Run()
			LoadConfig(ctx)
			return 0
		}
	}

	cfg := Get()
	fmt.Fprintln(out, color.BoldColorize("+--- Bush Configuration -----------------------------------+", color.ActivePalette.Accent))
	fmt.Fprintf(out, "| File:         %s\n", path)
	fmt.Fprintf(out, "| Theme:        %s\n", cfg.Theme.Name)
	fmt.Fprintf(out, "| Prompt:       multiline=%v, symbol=\"%s\"\n", cfg.Prompt.Multiline, cfg.Prompt.Symbol)
	fmt.Fprintf(out, "| Badges:       %s\n", strings.Join(cfg.Prompt.Badges, ", "))
	fmt.Fprintf(out, "| Autosuggest:  enabled=%v (history=%v, subcommands=%v)\n",
		cfg.Autosuggest.Enabled, cfg.Autosuggest.SuggestHistory, cfg.Autosuggest.SuggestSubcommands)
	fmt.Fprintf(out, "| Aliases:      %d configured\n", len(cfg.Aliases))
	fmt.Fprintln(out, color.BoldColorize("+----------------------------------------------------------+", color.ActivePalette.Accent))
	fmt.Fprintln(out, color.Colorize("Usage: config [path | edit | reload]", color.ActivePalette.GhostText))
	return 0
}
