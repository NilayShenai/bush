package config

import (
	"bush/internal/builtins"
	"bush/internal/color"
	"bytes"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
)

func TestTOMLDecoding(t *testing.T) {
	tomlData := `
[shell]
greeting_banner = false
motto = "Bush lovers forever."
history_limit = 1000

[prompt]
multiline = false
symbol = "$ "
badges = ["dir", "status"]
max_dir_depth = 2

[theme]
name = "nord"
[theme.colors]
accent = "#5E81AC"
error = "#FF0000"

[aliases]
foo = "bar"
`

	var cfg Config
	_, err := toml.Decode(tomlData, &cfg)
	if err != nil {
		t.Fatalf("failed to decode TOML: %v", err)
	}

	if cfg.Shell.GreetingBanner != false {
		t.Errorf("expected greeting_banner false, got %v", cfg.Shell.GreetingBanner)
	}
	if cfg.Shell.Motto != "Bush lovers forever." {
		t.Errorf("expected custom motto, got %s", cfg.Shell.Motto)
	}
	if cfg.Prompt.Multiline != false {
		t.Errorf("expected multiline false, got %v", cfg.Prompt.Multiline)
	}
	if cfg.Prompt.Symbol != "$ " {
		t.Errorf("expected symbol '$ ', got %s", cfg.Prompt.Symbol)
	}
	if len(cfg.Prompt.Badges) != 2 || cfg.Prompt.Badges[0] != "dir" {
		t.Errorf("unexpected badges: %v", cfg.Prompt.Badges)
	}
	if cfg.Aliases["foo"] != "bar" {
		t.Errorf("expected alias foo='bar', got %s", cfg.Aliases["foo"])
	}

	ApplyTheme(cfg.Theme)
	if color.ActivePalette.Accent != HexToRGB("#5E81AC") {
		t.Errorf("expected accent #5E81AC, got %v", color.ActivePalette.Accent)
	}
	if color.ActivePalette.Error != HexToRGB("#FF0000") {
		t.Errorf("expected error #FF0000, got %v", color.ActivePalette.Error)
	}
}

func TestHexToRGB(t *testing.T) {
	c := HexToRGB("#FFFFFF")
	if c.R != 255 || c.G != 255 || c.B != 255 {
		t.Errorf("expected 255,255,255 for #FFFFFF, got %v", c)
	}

	c = HexToRGB("#000000")
	if c.R != 0 || c.G != 0 || c.B != 0 {
		t.Errorf("expected 0,0,0 for #000000, got %v", c)
	}

	c = HexToRGB("#B4BEFE")
	if c.R != 180 || c.G != 190 || c.B != 254 {
		t.Errorf("expected 180,190,254 for #B4BEFE, got %v", c)
	}
}

func TestBuiltinConfigCommand(t *testing.T) {
	var out bytes.Buffer
	ctx := &builtins.ShellContext{
		Stdout: &out,
	}
	code := builtinConfig([]string{"config", "path"}, ctx)
	if code != 0 {
		t.Errorf("expected 0, got %d", code)
	}
	if !strings.Contains(out.String(), "config.toml") && !strings.Contains(out.String(), ".bush.toml") {
		t.Errorf("expected config path, got %s", out.String())
	}
}
