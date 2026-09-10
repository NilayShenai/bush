package lineeditor

import (
	"bush/internal/builtins"
	"bush/internal/executor"
	"os"
	"sort"
	"strings"
)

func Complete(line string, ctx *builtins.ShellContext) []string {
	tokens := strings.Fields(line)

	if len(tokens) >= 1 && tokens[0] == "jump" {
		prefix := ""
		if len(tokens) >= 2 && !strings.HasSuffix(line, " ") {
			prefix = tokens[1]
		}
		var matches []string
		if ctx.Bookmarks != nil {
			for tag := range ctx.Bookmarks {
				if strings.HasPrefix(tag, prefix) {
					matches = append(matches, tag)
				}
			}
		}
		sort.Strings(matches)
		return matches
	}

	if len(tokens) == 0 {
		return executor.FindPathCompletions("")
	}

	lastWord := tokens[len(tokens)-1]
	if strings.HasSuffix(line, " ") {
		lastWord = ""
	}

	if strings.HasPrefix(lastWord, "$") {
		varName := strings.TrimPrefix(lastWord, "$")
		var matches []string
		for _, env := range os.Environ() {
			k := strings.SplitN(env, "=", 2)[0]
			if strings.HasPrefix(k, varName) {
				matches = append(matches, "$"+k)
			}
		}
		sort.Strings(matches)
		return matches
	}

	if len(tokens) == 1 && !strings.HasSuffix(line, " ") {
		matches := executor.FindPathCompletions(lastWord)

		if ctx.Aliases != nil {
			for a := range ctx.Aliases {
				if strings.HasPrefix(a, lastWord) {
					matches = append(matches, a)
				}
			}
		}
		sort.Strings(matches)
		return matches
	}

	matches := executor.FindFileCompletions(lastWord)
	sort.Strings(matches)
	return matches
}

func LongestCommonPrefix(strs []string) string {
	if len(strs) == 0 {
		return ""
	}
	prefix := strs[0]
	for _, s := range strs[1:] {
		for !strings.HasPrefix(s, prefix) {
			if len(prefix) == 0 {
				return ""
			}
			prefix = prefix[:len(prefix)-1]
		}
	}
	return prefix
}
