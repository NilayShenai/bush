package color

import (
	"fmt"
	"os"
	"strings"
)

var trueColorSupported = func() bool {
	colorTerm := os.Getenv("COLORTERM")
	term := os.Getenv("TERM")
	return colorTerm == "truecolor" || colorTerm == "24bit" || strings.Contains(term, "256color") || strings.Contains(term, "xterm")
}()

type RGB struct {
	R, G, B uint8
}

var (
	Lavender      = RGB{180, 190, 254}
	Mauve         = RGB{203, 166, 247}
	PastelMint    = RGB{148, 226, 213}
	PastelPink    = RGB{245, 194, 231}
	PastelPeach   = RGB{250, 179, 135}
	PastelYellow  = RGB{249, 226, 175}
	PastelGreen   = RGB{166, 227, 161}
	PastelRed     = RGB{243, 139, 168}
	PastelBlue    = RGB{137, 180, 250}
	PastelGray    = RGB{108, 112, 134}
	TextWhite     = RGB{205, 214, 244}
	SurfaceDark   = RGB{49, 50, 68}
	SurfaceBorder = RGB{69, 71, 90}
)

type Palette struct {
	Accent       RGB
	User         RGB
	PromptSymbol RGB
	Directory    RGB
	GitBranch    RGB
	Duration     RGB
	Success      RGB
	Error        RGB
	GhostText    RGB
	Command      RGB
	InvalidCmd   RGB
	Flags        RGB
	Strings      RGB
	Border       RGB
}

var ActivePalette = Palette{
	Accent:       Lavender,
	User:         PastelPeach,
	PromptSymbol: Mauve,
	Directory:    PastelMint,
	GitBranch:    PastelPink,
	Duration:     PastelYellow,
	Success:      PastelGreen,
	Error:        PastelRed,
	GhostText:    PastelGray,
	Command:      Lavender,
	InvalidCmd:   PastelRed,
	Flags:        PastelPeach,
	Strings:      PastelYellow,
	Border:       SurfaceBorder,
}

const (
	Reset     = "\033[0m"
	Bold      = "\033[1m"
	Dim       = "\033[2m"
	Italic    = "\033[3m"
	Underline = "\033[4m"
)

func FgRGB(c RGB) string {
	return fmt.Sprintf("\033[38;2;%d;%d;%dm", c.R, c.G, c.B)
}

func BgRGB(c RGB) string {
	return fmt.Sprintf("\033[48;2;%d;%d;%dm", c.R, c.G, c.B)
}

func Colorize(text string, c RGB) string {
	return FgRGB(c) + text + Reset
}

func ColorizeBg(text string, fg RGB, bg RGB) string {
	return FgRGB(fg) + BgRGB(bg) + text + Reset
}

func BoldColorize(text string, c RGB) string {
	return Bold + FgRGB(c) + text + Reset
}
