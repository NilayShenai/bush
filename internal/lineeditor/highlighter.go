package lineeditor

import (
	"bush/internal/color"
	"bush/internal/executor"
	"strings"
	"unicode"
)

func Highlight(input string) string {
	if input == "" {
		return ""
	}

	pal := color.ActivePalette
	var sb strings.Builder
	runes := []rune(input)
	n := len(runes)
	i := 0
	isCommandPos := true

	for i < n {

		if unicode.IsSpace(runes[i]) {
			sb.WriteRune(runes[i])
			i++
			continue
		}

		if runes[i] == '|' || runes[i] == ';' || runes[i] == '&' || runes[i] == '>' || runes[i] == '<' {
			opStart := i
			for i < n && (runes[i] == '|' || runes[i] == ';' || runes[i] == '&' || runes[i] == '>' || runes[i] == '<') {
				i++
			}
			op := string(runes[opStart:i])
			sb.WriteString(color.BoldColorize(op, pal.PromptSymbol))
			if op == "|" || op == "&&" || op == "||" || op == ";" {
				isCommandPos = true
			}
			continue
		}

		if runes[i] == '\'' {
			start := i
			i++
			for i < n && runes[i] != '\'' {
				i++
			}
			if i < n && runes[i] == '\'' {
				i++
			}
			strVal := string(runes[start:i])
			sb.WriteString(color.Colorize(strVal, pal.Strings))
			isCommandPos = false
			continue
		}

		if runes[i] == '"' {
			start := i
			i++
			for i < n && runes[i] != '"' {
				if runes[i] == '\\' && i+1 < n {
					i += 2
					continue
				}
				i++
			}
			if i < n && runes[i] == '"' {
				i++
			}
			strVal := string(runes[start:i])
			sb.WriteString(color.Colorize(strVal, pal.Strings))
			isCommandPos = false
			continue
		}

		start := i
		for i < n && !unicode.IsSpace(runes[i]) && runes[i] != '|' && runes[i] != ';' && runes[i] != '&' && runes[i] != '>' && runes[i] != '<' {
			i++
		}
		word := string(runes[start:i])

		if isCommandPos {

			if executor.LookExecutable(word) {
				sb.WriteString(color.BoldColorize(word, pal.Command))
			} else {
				sb.WriteString(color.Colorize(word, pal.InvalidCmd))
			}
			isCommandPos = false
		} else if strings.HasPrefix(word, "-") {

			sb.WriteString(color.Colorize(word, pal.Flags))
		} else if strings.HasPrefix(word, "$") {

			sb.WriteString(color.Colorize(word, pal.GitBranch))
		} else if isNumber(word) {

			sb.WriteString(color.Colorize(word, pal.Directory))
		} else {

			sb.WriteString(color.Colorize(word, color.TextWhite))
		}
	}

	return sb.String()
}

func isNumber(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, r := range s {
		if (r < '0' || r > '9') && r != '.' {
			return false
		}
	}
	return true
}
