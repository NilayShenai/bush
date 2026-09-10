package expander

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Context struct {
	LastExitCode   int
	SubshellRunner func(cmd string) (string, error)
}

func ExpandToken(token string, ctx *Context) []string {
	if len(token) == 0 {
		return []string{""}
	}

	if strings.HasPrefix(token, "~/") || token == "~" {
		home, err := os.UserHomeDir()
		if err == nil {
			token = home + token[1:]
		}
	}

	if ctx != nil && ctx.SubshellRunner != nil {
		token = expandSubshells(token, ctx.SubshellRunner)
	}

	expanded, hasWildcards := expandVariablesAndQuotes(token, ctx)

	if hasWildcards {
		matches, err := filepath.Glob(expanded)
		if err == nil && len(matches) > 0 {
			return matches
		}
	}

	return []string{expanded}
}

func expandSubshells(s string, runner func(string) (string, error)) string {
	var sb strings.Builder
	runes := []rune(s)
	n := len(runes)
	i := 0

	for i < n {
		if runes[i] == '$' && i+1 < n && runes[i+1] == '(' {

			i += 2
			parenCount := 1
			start := i
			for i < n && parenCount > 0 {
				if runes[i] == '(' {
					parenCount++
				} else if runes[i] == ')' {
					parenCount--
				}
				if parenCount > 0 {
					i++
				}
			}
			cmd := string(runes[start:i])
			if i < n && runes[i] == ')' {
				i++
			}
			out, err := runner(cmd)
			if err == nil {

				sb.WriteString(strings.TrimRight(out, "\r\n"))
			}
			continue
		}

		if runes[i] == '`' {
			i++
			start := i
			for i < n && runes[i] != '`' {
				i++
			}
			cmd := string(runes[start:i])
			if i < n && runes[i] == '`' {
				i++
			}
			out, err := runner(cmd)
			if err == nil {
				sb.WriteString(strings.TrimRight(out, "\r\n"))
			}
			continue
		}

		sb.WriteRune(runes[i])
		i++
	}

	return sb.String()
}

func expandVariablesAndQuotes(s string, ctx *Context) (string, bool) {
	var sb strings.Builder
	runes := []rune(s)
	n := len(runes)
	i := 0
	hasWildcards := false

	for i < n {
		r := runes[i]

		if r == '\'' {
			i++
			for i < n && runes[i] != '\'' {
				sb.WriteRune(runes[i])
				i++
			}
			if i < n && runes[i] == '\'' {
				i++
			}
			continue
		}

		if r == '"' {
			i++
			for i < n && runes[i] != '"' {
				if runes[i] == '\\' && i+1 < n && (runes[i+1] == '"' || runes[i+1] == '\\' || runes[i+1] == '$') {
					i++
					sb.WriteRune(runes[i])
					i++
					continue
				}
				if runes[i] == '$' {
					varVal, adv := parseVar(runes, i, ctx)
					sb.WriteString(varVal)
					i += adv
					continue
				}
				sb.WriteRune(runes[i])
				i++
			}
			if i < n && runes[i] == '"' {
				i++
			}
			continue
		}

		if r == '$' {
			varVal, adv := parseVar(runes, i, ctx)
			sb.WriteString(varVal)
			i += adv
			continue
		}

		if r == '*' || r == '?' {
			hasWildcards = true
		}

		sb.WriteRune(r)
		i++
	}

	return sb.String(), hasWildcards
}

func parseVar(runes []rune, start int, ctx *Context) (string, int) {
	n := len(runes)
	if start+1 >= n {
		return "$", 1
	}

	next := runes[start+1]

	if next == '?' {
		code := 0
		if ctx != nil {
			code = ctx.LastExitCode
		}
		return strconv.Itoa(code), 2
	}

	if next == '$' {
		return strconv.Itoa(os.Getpid()), 2
	}

	if next == '{' {
		end := start + 2
		for end < n && runes[end] != '}' {
			end++
		}
		if end < n && runes[end] == '}' {
			varName := string(runes[start+2 : end])
			val := os.Getenv(varName)
			return val, end - start + 1
		}
	}

	i := start + 1
	for i < n && (isAlphaNumeric(runes[i]) || runes[i] == '_') {
		i++
	}
	if i == start+1 {
		return "$", 1
	}

	varName := string(runes[start+1 : i])
	return os.Getenv(varName), i - start
}

func isAlphaNumeric(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}

func ExpandArgs(args []string, ctx *Context) []string {
	var result []string
	for _, arg := range args {
		expanded := ExpandToken(arg, ctx)
		result = append(result, expanded...)
	}
	return result
}

func ExpandRedirectionTarget(target string, ctx *Context) string {
	res := ExpandToken(target, ctx)
	if len(res) > 0 {
		return res[0]
	}
	return target
}

func Sprintf(format string, a ...interface{}) string {
	return fmt.Sprintf(format, a...)
}
