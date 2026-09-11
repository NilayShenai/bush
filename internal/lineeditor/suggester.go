package lineeditor

import (
	"bush/internal/builtins"
	"bush/internal/config"
	"bush/internal/executor"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var subcommands = map[string][]string{
	"git":       {"status", "commit -m \"\"", "push", "pull", "add .", "checkout", "branch", "diff", "log", "clone", "merge", "rebase", "stash"},
	"docker":    {"ps", "run -it", "build -t", "images", "compose up", "compose down", "stop", "logs -f", "exec -it"},
	"go":        {"run main.go", "build", "test ./...", "mod tidy", "mod init", "get", "vet ./..."},
	"cargo":     {"build", "run", "test", "check", "add", "update"},
	"npm":       {"run dev", "run build", "install", "start", "test"},
	"yarn":      {"dev", "build", "add", "install", "test"},
	"pnpm":      {"dev", "build", "add", "install"},
	"python":    {"-m venv .venv", "-m http.server", "main.py", "app.py"},
	"python3":   {"-m venv .venv", "-m http.server", "main.py", "app.py"},
	"systemctl": {"status", "restart", "start", "stop", "enable", "disable"},
	"service":   {"status", "restart", "start", "stop"},
	"apt":       {"install", "update", "upgrade", "remove", "search"},
	"kill":      {"-9"},
	"ls":        {"-la", "-lh"},
	"grep":      {"-rnI"},
	"find":      {". -name"},
	"curl":      {"-sSL", "-I", "-X POST"},
	"bush":      {"-c \"about\"", "--version", "config"},
	"g":         {"status", "commit -m \"\"", "push", "pull", "add .", "checkout", "branch", "diff", "log"},
	"memo":      {"clear", "info"},
	"envload":   {".env", ".env.local", ".env.prod"},
	"killport":  {"3000", "8080", "5000", "8000"},
	"update":    {"check", "force"},
}

var fileCommands = map[string]bool{
	"cat": true, "less": true, "more": true, "vim": true, "vi": true, "nano": true,
	"head": true, "tail": true, "peek": true, "rm": true, "cp": true, "mv": true,
	"source": true, ".": true, "source-file": true,
}

var (
	cachedExecutables []string
	cacheInitialized  bool
)

func getCachedExecutables() []string {
	if cacheInitialized {
		return cachedExecutables
	}

	seen := make(map[string]bool)
	var list []string

	for _, b := range builtins.GetBuiltinNames() {
		seen[b] = true
		list = append(list, b)
	}

	pathEnv := os.Getenv("PATH")
	for _, dir := range filepath.SplitList(pathEnv) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			name := e.Name()
			if !seen[name] {
				seen[name] = true
				list = append(list, name)
			}
		}
	}

	sort.Strings(list)
	cachedExecutables = list
	cacheInitialized = true
	return cachedExecutables
}

func FindSmartSuggestion(line string, history *History, ctx *builtins.ShellContext) string {
	if line == "" {
		return ""
	}

	cfg := config.Get()
	if !cfg.Autosuggest.Enabled {
		return ""
	}

	if cfg.Autosuggest.SuggestHistory && history != nil {
		if histSug := history.FindSuggestion(line); histSug != "" {
			return histSug
		}
	}

	tokens := strings.Fields(line)

	if len(tokens) == 1 && !strings.HasSuffix(line, " ") {
		word := tokens[0]

		if ctx != nil && ctx.Aliases != nil {
			for a := range ctx.Aliases {
				if strings.HasPrefix(a, word) && len(a) > len(word) {
					return a[len(word):]
				}
			}
		}

		if cfg.Autosuggest.SuggestBinaries {

			for _, b := range builtins.GetBuiltinNames() {
				if strings.HasPrefix(b, word) && len(b) > len(word) {
					return b[len(word):]
				}
			}

			for _, exe := range getCachedExecutables() {
				if strings.HasPrefix(exe, word) && len(exe) > len(word) {
					return exe[len(word):]
				}
			}
		}

		return ""
	}

	if len(tokens) == 0 {
		return ""
	}

	cmd := tokens[0]

	if cfg.Autosuggest.SuggestSubcommands {
		if subList, ok := subcommands[cmd]; ok {
			if len(tokens) == 1 && strings.HasSuffix(line, " ") {
				return subList[0]
			}
			if len(tokens) == 2 && !strings.HasSuffix(line, " ") {
				subPrefix := tokens[1]
				for _, sub := range subList {
					if strings.HasPrefix(sub, subPrefix) && len(sub) > len(subPrefix) {
						return sub[len(subPrefix):]
					}
				}
			}
		}
	}

	if cmd == "jump" {
		if ctx != nil && ctx.Bookmarks != nil {
			prefix := ""
			if len(tokens) == 2 && !strings.HasSuffix(line, " ") {
				prefix = tokens[1]
			}
			for tag := range ctx.Bookmarks {
				if strings.HasPrefix(tag, prefix) && len(tag) > len(prefix) {
					return tag[len(prefix):]
				}
			}
		}
	}

	if cmd == "cd" {
		prefix := ""
		if len(tokens) >= 2 && !strings.HasSuffix(line, " ") {
			prefix = tokens[len(tokens)-1]
		}
		matches := executor.FindFileCompletions(prefix)
		for _, m := range matches {
			if strings.HasSuffix(m, "/") && strings.HasPrefix(m, prefix) && len(m) > len(prefix) {
				return m[len(prefix):]
			}
		}
	}

	if cfg.Autosuggest.SuggestFiles {
		if fileCommands[cmd] || strings.Contains(line, "/") {
			lastWord := tokens[len(tokens)-1]
			if strings.HasSuffix(line, " ") {
				lastWord = ""
			}
			matches := executor.FindFileCompletions(lastWord)
			if len(matches) > 0 {
				m := matches[0]
				if strings.HasPrefix(m, lastWord) && len(m) > len(lastWord) {
					return m[len(lastWord):]
				}
			}
		}
	}

	if strings.HasSuffix(line, "$") || (len(tokens) > 0 && strings.HasPrefix(tokens[len(tokens)-1], "$")) {
		lastWord := tokens[len(tokens)-1]
		varPrefix := strings.TrimPrefix(lastWord, "$")
		for _, env := range os.Environ() {
			k := strings.SplitN(env, "=", 2)[0]
			if strings.HasPrefix(k, varPrefix) && len(k) > len(varPrefix) {
				return k[len(varPrefix):]
			}
		}
	}

	return ""
}
