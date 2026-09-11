package lineeditor

import (
	"bush/internal/builtins"
	"bush/internal/color"
	"bush/internal/prompt"
	"bush/internal/terminal"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"
)

type LineEditor struct {
	History *History
	Ctx     *builtins.ShellContext
}

func NewLineEditor(ctx *builtins.ShellContext) *LineEditor {
	h := NewHistory()
	if ctx != nil {
		ctx.HistoryList = h.Entries()
	}
	return &LineEditor{
		History: h,
		Ctx:     ctx,
	}
}

func (le *LineEditor) ReadLine(promptState *prompt.State) (string, error) {
	fd := terminal.StdinFd()
	if !terminal.IsTerminal(fd) {
		var line string
		_, err := fmt.Scanln(&line)
		return line, err
	}

	_, err := terminal.MakeRaw(fd)
	if err != nil {
		return "", err
	}
	defer terminal.Restore(fd)

	os.Stdout.WriteString("\033[?2004h")
	defer os.Stdout.WriteString("\033[?2004l")

	topPrompt, promptSym := prompt.Render(promptState)
	os.Stdout.WriteString(topPrompt)
	os.Stdout.WriteString("\r\n")

	var runes []rune
	cursorPos := 0
	historyIdx := -1
	var savedDraft string

	renderLine := func() {
		lineStr := string(runes)
		suggestion := ""
		if cursorPos == len(runes) && len(lineStr) > 0 {
			suggestion = FindSmartSuggestion(lineStr, le.History, le.Ctx)
		}

		os.Stdout.WriteString("\r\033[2K")
		os.Stdout.WriteString(promptSym)

		highlighted := Highlight(lineStr)
		os.Stdout.WriteString(highlighted)

		if suggestion != "" {
			ghost := color.Colorize(suggestion, color.ActivePalette.GhostText)
			os.Stdout.WriteString(ghost)
		}

		backSteps := (len(runes) - cursorPos) + len([]rune(suggestion))
		if backSteps > 0 {
			fmt.Fprintf(os.Stdout, "\033[%dD", backSteps)
		}
	}

	renderLine()

	for {
		key, err := terminal.ReadKey(os.Stdin)
		if err != nil {
			if err == io.EOF {
				return "", err
			}
			return "", err
		}

		switch key.Type {
		case terminal.KeyEnter:

			os.Stdout.WriteString("\r\033[2K")
			os.Stdout.WriteString(promptSym)
			os.Stdout.WriteString(Highlight(string(runes)))
			os.Stdout.WriteString("\r\n")

			result := string(runes)
			if strings.TrimSpace(result) != "" {
				le.History.Add(result)
				if le.Ctx != nil {
					le.Ctx.HistoryList = le.History.Entries()
				}
			}
			return result, nil

		case terminal.KeyCtrlC:
			os.Stdout.WriteString("^C\r\n")
			return "", nil

		case terminal.KeyCtrlD:
			if len(runes) == 0 {
				os.Stdout.WriteString("exit\r\n")
				return "exit", nil
			}
			if cursorPos < len(runes) {
				newRunes := make([]rune, 0, len(runes)-1)
				newRunes = append(newRunes, runes[:cursorPos]...)
				newRunes = append(newRunes, runes[cursorPos+1:]...)
				runes = newRunes
				renderLine()
			}

		case terminal.KeyCtrlL:
			os.Stdout.WriteString("\033[H\033[2J")
			os.Stdout.WriteString(topPrompt)
			os.Stdout.WriteString("\r\n")
			renderLine()

		case terminal.KeyBackspace:
			if cursorPos > 0 {
				newRunes := make([]rune, 0, len(runes)-1)
				newRunes = append(newRunes, runes[:cursorPos-1]...)
				newRunes = append(newRunes, runes[cursorPos:]...)
				runes = newRunes
				cursorPos--
				renderLine()
			}

		case terminal.KeyDelete:
			if cursorPos < len(runes) {
				newRunes := make([]rune, 0, len(runes)-1)
				newRunes = append(newRunes, runes[:cursorPos]...)
				newRunes = append(newRunes, runes[cursorPos+1:]...)
				runes = newRunes
				renderLine()
			}

		case terminal.KeyArrowLeft:
			if cursorPos > 0 {
				cursorPos--
				renderLine()
			}

		case terminal.KeyArrowRight:
			lineStr := string(runes)
			suggestion := FindSmartSuggestion(lineStr, le.History, le.Ctx)
			if cursorPos == len(runes) && suggestion != "" {

				runes = append(runes, []rune(suggestion)...)
				cursorPos = len(runes)
			} else if cursorPos < len(runes) {
				cursorPos++
			}
			renderLine()

		case terminal.KeyAltRight:
			lineStr := string(runes)
			suggestion := FindSmartSuggestion(lineStr, le.History, le.Ctx)
			if cursorPos == len(runes) && suggestion != "" {
				words := strings.Fields(suggestion)
				if len(words) > 0 {
					nextWord := words[0] + " "
					runes = append(runes, []rune(nextWord)...)
					cursorPos = len(runes)
				}
			} else if cursorPos < len(runes) {
				for cursorPos < len(runes) && runes[cursorPos] == ' ' {
					cursorPos++
				}
				for cursorPos < len(runes) && runes[cursorPos] != ' ' {
					cursorPos++
				}
			}
			renderLine()

		case terminal.KeyAltLeft:
			for cursorPos > 0 && runes[cursorPos-1] == ' ' {
				cursorPos--
			}
			for cursorPos > 0 && runes[cursorPos-1] != ' ' {
				cursorPos--
			}
			renderLine()

		case terminal.KeyHome, terminal.KeyCtrlA:
			cursorPos = 0
			renderLine()

		case terminal.KeyEnd, terminal.KeyCtrlE:
			cursorPos = len(runes)
			renderLine()

		case terminal.KeyCtrlU:
			newRunes := make([]rune, len(runes[cursorPos:]))
			copy(newRunes, runes[cursorPos:])
			runes = newRunes
			cursorPos = 0
			renderLine()

		case terminal.KeyCtrlK:
			newRunes := make([]rune, cursorPos)
			copy(newRunes, runes[:cursorPos])
			runes = newRunes
			renderLine()

		case terminal.KeyCtrlW:
			if cursorPos > 0 {
				oldPos := cursorPos
				for cursorPos > 0 && runes[cursorPos-1] == ' ' {
					cursorPos--
				}
				for cursorPos > 0 && runes[cursorPos-1] != ' ' {
					cursorPos--
				}
				newRunes := make([]rune, 0, len(runes)-(oldPos-cursorPos))
				newRunes = append(newRunes, runes[:cursorPos]...)
				newRunes = append(newRunes, runes[oldPos:]...)
				runes = newRunes
				renderLine()
			}

		case terminal.KeyArrowUp:
			entries := le.History.Entries()
			if len(entries) > 0 {
				if historyIdx == -1 {
					savedDraft = string(runes)
					historyIdx = len(entries) - 1
				} else if historyIdx > 0 {
					historyIdx--
				}
				if historyIdx >= 0 && historyIdx < len(entries) {
					runes = []rune(entries[historyIdx])
					cursorPos = len(runes)
					renderLine()
				}
			}

		case terminal.KeyArrowDown:
			entries := le.History.Entries()
			if historyIdx != -1 {
				if historyIdx < len(entries)-1 {
					historyIdx++
					runes = []rune(entries[historyIdx])
					cursorPos = len(runes)
				} else {
					historyIdx = -1
					runes = []rune(savedDraft)
					cursorPos = len(runes)
				}
				renderLine()
			}

		case terminal.KeyTab:
			lineStr := string(runes)
			suggestion := FindSmartSuggestion(lineStr, le.History, le.Ctx)
			if cursorPos == len(runes) && suggestion != "" {
				runes = append(runes, []rune(suggestion)...)
				cursorPos = len(runes)
				renderLine()
				continue
			}

			candidates := Complete(lineStr, le.Ctx)
			if len(candidates) == 1 {
				match := candidates[0]
				tokens := strings.Fields(lineStr)
				if len(tokens) > 0 && !strings.HasSuffix(lineStr, " ") {
					lastTok := tokens[len(tokens)-1]
					prefixLen := utf8.RuneCountInString(lineStr) - utf8.RuneCountInString(lastTok)
					runes = append([]rune(lineStr[:prefixLen]), []rune(match)...)
				} else {
					runes = append(runes, []rune(match)...)
				}
				if !strings.HasSuffix(match, "/") {
					runes = append(runes, ' ')
				}
				cursorPos = len(runes)
				renderLine()
			} else if len(candidates) > 1 {
				common := LongestCommonPrefix(candidates)
				tokens := strings.Fields(lineStr)
				if len(tokens) > 0 && !strings.HasSuffix(lineStr, " ") {
					lastTok := tokens[len(tokens)-1]
					if len(common) > len(lastTok) {
						prefixLen := utf8.RuneCountInString(lineStr) - utf8.RuneCountInString(lastTok)
						runes = append([]rune(lineStr[:prefixLen]), []rune(common)...)
						cursorPos = len(runes)
					}
				}

				os.Stdout.WriteString("\r\n")
				limit := min(len(candidates), 20)
				for i := 0; i < limit; i++ {
					cColor := color.PastelMint
					if strings.HasSuffix(candidates[i], "/") {
						cColor = color.Lavender
					}
					fmt.Fprintf(os.Stdout, "  %s", color.Colorize(candidates[i], cColor))
					if (i+1)%4 == 0 {
						os.Stdout.WriteString("\r\n")
					}
				}
				if len(candidates) > limit {
					fmt.Fprintf(os.Stdout, "  ... (+%d more)", len(candidates)-limit)
				}
				os.Stdout.WriteString("\r\n")
				os.Stdout.WriteString(promptSym)
				renderLine()
			}

		case terminal.KeyCtrlR, terminal.KeyCtrlP:
			selected := le.runFuzzyPalette()
			if selected != "" {
				runes = []rune(selected)
				cursorPos = len(runes)
			}
			os.Stdout.WriteString("\r\n")
			os.Stdout.WriteString(promptSym)
			renderLine()

		case terminal.KeyPaste:
			pasteText := key.Text
			if cursorPos == 0 && len(runes) == 0 {
				pasteText = strings.TrimLeft(pasteText, " \t")
			}
			pasteText = strings.TrimRight(pasteText, "\r\n")
			pasteText = strings.ReplaceAll(pasteText, "\r\n", " ")
			pasteText = strings.ReplaceAll(pasteText, "\n", " ")
			pasteText = strings.ReplaceAll(pasteText, "\r", " ")

			pasteRunes := []rune(pasteText)
			if len(pasteRunes) > 0 {
				newRunes := make([]rune, 0, len(runes)+len(pasteRunes))
				newRunes = append(newRunes, runes[:cursorPos]...)
				newRunes = append(newRunes, pasteRunes...)
				newRunes = append(newRunes, runes[cursorPos:]...)
				runes = newRunes
				cursorPos += len(pasteRunes)
			}
			renderLine()

		case terminal.KeyRune:
			if cursorPos == 0 && len(runes) == 0 && (key.Rune == ' ' || key.Rune == '\t') {
				continue
			}
			newRunes := make([]rune, 0, len(runes)+1)
			newRunes = append(newRunes, runes[:cursorPos]...)
			newRunes = append(newRunes, key.Rune)
			newRunes = append(newRunes, runes[cursorPos:]...)
			runes = newRunes
			cursorPos++
			renderLine()
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (le *LineEditor) runFuzzyPalette() string {
	entries := le.History.Entries()
	if len(entries) == 0 {
		return ""
	}

	var query []rune
	selectedIdx := 0

	drawPalette := func() {
		os.Stdout.WriteString("\r\n")
		os.Stdout.WriteString(color.BoldColorize("+--- Bush Command Palette & Fuzzy History ----------------+\r\n", color.Lavender))
		qStr := string(query)
		fmt.Fprintf(os.Stdout, "| %s: %s\033[K\r\n", color.Colorize("Search", color.PastelPeach), color.BoldColorize(qStr, color.PastelMint))
		os.Stdout.WriteString(color.BoldColorize("+---------------------------------------------------------+\r\n", color.Mauve))

		var filtered []string
		for i := len(entries) - 1; i >= 0; i-- {
			if strings.Contains(strings.ToLower(entries[i]), strings.ToLower(qStr)) {
				filtered = append(filtered, entries[i])
			}
			if len(filtered) >= 5 {
				break
			}
		}

		if len(filtered) == 0 {
			os.Stdout.WriteString("|   " + color.Colorize("No matches found", color.PastelGray) + "\033[K\r\n")
		} else {
			if selectedIdx >= len(filtered) {
				selectedIdx = 0
			}
			for i, item := range filtered {
				if i == selectedIdx {
					fmt.Fprintf(os.Stdout, "| -> %s\033[K\r\n", color.ColorizeBg(" "+item+" ", color.SurfaceDark, color.Lavender))
				} else {
					fmt.Fprintf(os.Stdout, "|    %s\033[K\r\n", color.Colorize(item, color.TextWhite))
				}
			}
		}
		os.Stdout.WriteString(color.BoldColorize("+---------------------------------------------------------+\r\n", color.Lavender))
	}

	drawPalette()

	for {
		key, err := terminal.ReadKey(os.Stdin)
		if err != nil {
			return ""
		}

		switch key.Type {
		case terminal.KeyEscape, terminal.KeyCtrlC:
			return ""
		case terminal.KeyEnter:
			qStr := string(query)
			var filtered []string
			for i := len(entries) - 1; i >= 0; i-- {
				if strings.Contains(strings.ToLower(entries[i]), strings.ToLower(qStr)) {
					filtered = append(filtered, entries[i])
				}
			}
			if len(filtered) > 0 && selectedIdx < len(filtered) {
				return filtered[selectedIdx]
			}
			return qStr
		case terminal.KeyArrowUp:
			if selectedIdx > 0 {
				selectedIdx--
				drawPalette()
			}
		case terminal.KeyArrowDown:
			selectedIdx++
			drawPalette()
		case terminal.KeyBackspace:
			if len(query) > 0 {
				query = query[:len(query)-1]
				selectedIdx = 0
				drawPalette()
			}
		case terminal.KeyRune:
			query = append(query, key.Rune)
			selectedIdx = 0
			drawPalette()
		}
	}
}
