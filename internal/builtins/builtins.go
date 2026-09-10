package builtins

import (
	"bush/internal/color"
	"bush/internal/jobcontrol"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

var lastDir string

func init() {
	Register("cd", builtinCd)
	Register("pwd", builtinPwd)
	Register("exit", builtinExit)
	Register("quit", builtinExit)
	Register("export", builtinExport)
	Register("unset", builtinUnset)
	Register("env", builtinEnv)
	Register("alias", builtinAlias)
	Register("unalias", builtinUnalias)
	Register("history", builtinHistory)
	Register("jobs", builtinJobs)
	Register("fg", builtinFg)
	Register("bg", builtinBg)
	Register("which", builtinWhich)
	Register("type", builtinType)
	Register("echo", builtinEcho)
	Register("source", builtinSource)
	Register(".", builtinSource)
	Register("time", builtinTime)
	Register("clear", builtinClear)
	Register("help", builtinHelp)
	Register("about", builtinAbout)
	Register("z", builtinZ)
	Register("port", builtinPort)
	Register("killport", builtinKillport)
	Register("watch", builtinWatch)
	Register("envload", builtinEnvload)
	Register("memo", builtinMemo)
	Register("g", builtinG)
}

func builtinCd(args []string, ctx *ShellContext) int {
	var targetDir string
	if len(args) <= 1 {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(ctx.Stderr, "bush: cd: %v\n", err)
			return 1
		}
		targetDir = home
	} else if args[1] == "-" {
		if lastDir == "" {
			fmt.Fprintf(ctx.Stderr, "bush: cd: OLDPWD not set\n")
			return 1
		}
		targetDir = lastDir
		fmt.Fprintln(ctx.Stdout, targetDir)
	} else {
		targetDir = args[1]
	}

	currentDir, _ := os.Getwd()
	err := os.Chdir(targetDir)
	if err != nil {
		fmt.Fprintf(ctx.Stderr, "bush: cd: %s: %v\n", targetDir, err)
		return 1
	}

	lastDir = currentDir
	newDir, _ := os.Getwd()
	os.Setenv("OLDPWD", lastDir)
	os.Setenv("PWD", newDir)
	RecordDirectoryVisit(newDir)
	return 0
}

func builtinPwd(args []string, ctx *ShellContext) int {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(ctx.Stderr, "bush: pwd: %v\n", err)
		return 1
	}
	fmt.Fprintln(ctx.Stdout, dir)
	return 0
}

func builtinExit(args []string, ctx *ShellContext) int {
	code := 0
	if len(args) > 1 {
		var err error
		code, err = strconv.Atoi(args[1])
		if err != nil {
			fmt.Fprintf(ctx.Stderr, "bush: exit: numeric argument required\n")
			code = 2
		}
	}
	if ctx.ExitShell != nil {
		ctx.ExitShell(code)
	} else {
		os.Exit(code)
	}
	return code
}

func builtinExport(args []string, ctx *ShellContext) int {
	if len(args) == 1 {
		for _, e := range os.Environ() {
			fmt.Fprintln(ctx.Stdout, e)
		}
		return 0
	}

	for _, arg := range args[1:] {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) == 2 {
			os.Setenv(parts[0], parts[1])
		} else {
			if _, exists := os.LookupEnv(parts[0]); !exists {
				os.Setenv(parts[0], "")
			}
		}
	}
	return 0
}

func builtinUnset(args []string, ctx *ShellContext) int {
	for _, arg := range args[1:] {
		os.Unsetenv(arg)
	}
	return 0
}

func builtinEnv(args []string, ctx *ShellContext) int {
	for _, e := range os.Environ() {
		fmt.Fprintln(ctx.Stdout, e)
	}
	return 0
}

func builtinAlias(args []string, ctx *ShellContext) int {
	if ctx.Aliases == nil {
		ctx.Aliases = make(map[string]string)
	}

	if len(args) == 1 {
		var keys []string
		for k := range ctx.Aliases {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(ctx.Stdout, "alias %s='%s'\n", k, ctx.Aliases[k])
		}
		return 0
	}

	for _, arg := range args[1:] {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) == 2 {
			ctx.Aliases[parts[0]] = strings.Trim(parts[1], "'\"")
		} else {
			if val, ok := ctx.Aliases[parts[0]]; ok {
				fmt.Fprintf(ctx.Stdout, "alias %s='%s'\n", parts[0], val)
			} else {
				fmt.Fprintf(ctx.Stderr, "bush: alias: %s: not found\n", parts[0])
				return 1
			}
		}
	}
	return 0
}

func builtinUnalias(args []string, ctx *ShellContext) int {
	if len(args) < 2 {
		fmt.Fprintf(ctx.Stderr, "bush: unalias: usage: unalias name...\n")
		return 1
	}
	for _, name := range args[1:] {
		delete(ctx.Aliases, name)
	}
	return 0
}

func builtinHistory(args []string, ctx *ShellContext) int {
	for i, cmd := range ctx.HistoryList {
		numStr := color.Colorize(fmt.Sprintf("%4d", i+1), color.PastelGray)
		fmt.Fprintf(ctx.Stdout, "  %s  %s\n", numStr, cmd)
	}
	return 0
}

func builtinJobs(args []string, ctx *ShellContext) int {
	jobcontrol.DefaultManager.ListJobs()
	return 0
}

func builtinFg(args []string, ctx *ShellContext) int {
	id := 1
	if len(args) > 1 {
		clean := strings.TrimPrefix(args[1], "%")
		var err error
		id, err = strconv.Atoi(clean)
		if err != nil {
			fmt.Fprintf(ctx.Stderr, "bush: fg: invalid job id\n")
			return 1
		}
	}
	err := jobcontrol.DefaultManager.Foreground(id)
	if err != nil {
		fmt.Fprintf(ctx.Stderr, "bush: fg: %v\n", err)
		return 1
	}
	return 0
}

func builtinBg(args []string, ctx *ShellContext) int {
	id := 1
	if len(args) > 1 {
		clean := strings.TrimPrefix(args[1], "%")
		var err error
		id, err = strconv.Atoi(clean)
		if err != nil {
			fmt.Fprintf(ctx.Stderr, "bush: bg: invalid job id\n")
			return 1
		}
	}
	err := jobcontrol.DefaultManager.Background(id)
	if err != nil {
		fmt.Fprintf(ctx.Stderr, "bush: bg: %v\n", err)
		return 1
	}
	return 0
}

func builtinWhich(args []string, ctx *ShellContext) int {
	if len(args) < 2 {
		fmt.Fprintf(ctx.Stderr, "bush: which: missing operand\n")
		return 1
	}
	exitCode := 0
	for _, name := range args[1:] {
		if IsBuiltin(name) {
			fmt.Fprintf(ctx.Stdout, "%s: shell built-in command\n", name)
			continue
		}
		if val, ok := ctx.Aliases[name]; ok {
			fmt.Fprintf(ctx.Stdout, "%s: aliased to %s\n", name, val)
			continue
		}
		path, err := exec.LookPath(name)
		if err != nil {
			fmt.Fprintf(ctx.Stderr, "%s not found\n", name)
			exitCode = 1
		} else {
			fmt.Fprintln(ctx.Stdout, path)
		}
	}
	return exitCode
}

func builtinType(args []string, ctx *ShellContext) int {
	return builtinWhich(args, ctx)
}

func builtinEcho(args []string, ctx *ShellContext) int {
	noNewline := false
	interpretEscapes := false
	startIdx := 1

	for startIdx < len(args) && strings.HasPrefix(args[startIdx], "-") && len(args[startIdx]) > 1 {
		arg := args[startIdx]
		matched := true
		for _, ch := range arg[1:] {
			if ch == 'n' {
				noNewline = true
			} else if ch == 'e' {
				interpretEscapes = true
			} else {
				matched = false
				break
			}
		}
		if !matched {
			break
		}
		startIdx++
	}

	text := strings.Join(args[startIdx:], " ")

	if interpretEscapes {
		text = strings.ReplaceAll(text, `\n`, "\n")
		text = strings.ReplaceAll(text, `\t`, "\t")
		text = strings.ReplaceAll(text, `\\`, "\\")
		text = strings.ReplaceAll(text, `\033`, "\033")
		text = strings.ReplaceAll(text, `\e`, "\033")
	}

	if noNewline {
		fmt.Fprint(ctx.Stdout, text)
	} else {
		fmt.Fprintln(ctx.Stdout, text)
	}
	return 0
}

func builtinSource(args []string, ctx *ShellContext) int {
	if len(args) < 2 {
		fmt.Fprintf(ctx.Stderr, "bush: source: filename argument required\n")
		return 1
	}
	file := args[1]
	data, err := os.ReadFile(file)
	if err != nil {
		fmt.Fprintf(ctx.Stderr, "bush: source: %v\n", err)
		return 1
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if ctx.SubshellRunner != nil {
			_, _ = ctx.SubshellRunner(trimmed)
		}
	}
	return 0
}

func builtinTime(args []string, ctx *ShellContext) int {
	if len(args) < 2 {
		fmt.Fprintf(ctx.Stderr, "bush: time: command required\n")
		return 1
	}
	cmdStr := strings.Join(args[1:], " ")
	start := time.Now()
	var code int
	if ctx.SubshellRunner != nil {
		_, err := ctx.SubshellRunner(cmdStr)
		if err != nil {
			code = 1
		}
	}
	duration := time.Since(start)

	timeHeader := color.BoldColorize("[time] Execution Time:", color.Lavender)
	timeVal := color.Colorize(fmt.Sprintf("%v", duration), color.PastelMint)
	fmt.Fprintf(ctx.Stderr, "\n%s %s\n", timeHeader, timeVal)
	return code
}

func builtinClear(args []string, ctx *ShellContext) int {
	fmt.Fprint(ctx.Stdout, "\033[H\033[2J")
	return 0
}

func builtinHelp(args []string, ctx *ShellContext) int {
	fmt.Fprintln(ctx.Stdout, color.BoldColorize("+-----------------------------------------------------------+", color.Lavender))
	fmt.Fprintln(ctx.Stdout, color.BoldColorize("|                    BUSH SHELL (bush)                      |", color.Lavender))
	fmt.Fprintln(ctx.Stdout, color.BoldColorize("|              FOR BUSH LOVERS, BY BUSH LOVERS              |", color.Mauve))
	fmt.Fprintln(ctx.Stdout, color.BoldColorize("+-----------------------------------------------------------+", color.Lavender))
	fmt.Fprintln(ctx.Stdout, "  "+color.Colorize("\"A shell for people of refined taste.\"", color.PastelPink))
	fmt.Fprintln(ctx.Stdout)

	fmt.Fprintln(ctx.Stdout, color.BoldColorize("STANDARD BUILT-IN COMMANDS:", color.PastelPeach))
	fmt.Fprintln(ctx.Stdout, "  "+color.Colorize("cd [dir|-]", color.PastelMint)+"        Change directory (supports '-' for previous dir)")
	fmt.Fprintln(ctx.Stdout, "  "+color.Colorize("pwd", color.PastelMint)+"               Print working directory")
	fmt.Fprintln(ctx.Stdout, "  "+color.Colorize("export [k=v]", color.PastelMint)+"       Set environment variables")
	fmt.Fprintln(ctx.Stdout, "  "+color.Colorize("alias [k=v]", color.PastelMint)+"        Create command shortcut aliases")
	fmt.Fprintln(ctx.Stdout, "  "+color.Colorize("history", color.PastelMint)+"           Show command history")
	fmt.Fprintln(ctx.Stdout, "  "+color.Colorize("jobs, fg, bg", color.PastelMint)+"      Manage background jobs")
	fmt.Fprintln(ctx.Stdout, "  "+color.Colorize("which, type", color.PastelMint)+"       Display information about command type")
	fmt.Fprintln(ctx.Stdout, "  "+color.Colorize("time <cmd>", color.PastelMint)+"        Measure command execution time with high precision")
	fmt.Fprintln(ctx.Stdout, "  "+color.Colorize("clear", color.PastelMint)+"             Clear terminal screen")
	fmt.Fprintln(ctx.Stdout, "  "+color.Colorize("config", color.PastelMint)+"            Configure Bush (config path|edit|reload)")
	fmt.Fprintln(ctx.Stdout, "  "+color.Colorize("about", color.PastelMint)+"             About Bush Shell")
	fmt.Fprintln(ctx.Stdout, "  "+color.Colorize("exit, quit", color.PastelMint)+"        Exit Bush Shell")
	fmt.Fprintln(ctx.Stdout)

	fmt.Fprintln(ctx.Stdout, color.BoldColorize("EXTRA FEATURES:", color.Mauve))
	fmt.Fprintln(ctx.Stdout, "  "+color.Colorize("z [query]", color.PastelPink)+"           Native Frecency Jump: Teleport to frequent/recent directories")
	fmt.Fprintln(ctx.Stdout, "  "+color.Colorize("port [num]", color.PastelPink)+"          Port Doctor: Inspect process & memory holding any local port")
	fmt.Fprintln(ctx.Stdout, "  "+color.Colorize("killport <num>", color.PastelPink)+"      Instantly terminate whatever process is hogging a port")
	fmt.Fprintln(ctx.Stdout, "  "+color.Colorize("watch <interval> <cmd>", color.PastelPink)+" Differential live watcher with real-time pastel diff highlighting")
	fmt.Fprintln(ctx.Stdout, "  "+color.Colorize("envload [file]", color.PastelPink)+"      Safely load .env files with automatic secret masking")
	fmt.Fprintln(ctx.Stdout, "  "+color.Colorize("memo <cmd>", color.PastelPink)+"         Memoize command stdout and replay instantly without re-running")
	fmt.Fprintln(ctx.Stdout, "  "+color.Colorize("g [args...]", color.PastelPink)+"         Compact Git HUD (when run alone) or transparent git alias")
	fmt.Fprintln(ctx.Stdout, "  "+color.Colorize("peek [file]", color.PastelPink)+"         Inspect streaming pipeline data & stats live without breaking stdout")
	fmt.Fprintln(ctx.Stdout, "  "+color.Colorize("mark <tag>", color.PastelPink)+"          Save current directory as a persistent named bookmark")
	fmt.Fprintln(ctx.Stdout, "  "+color.Colorize("jump <tag>", color.PastelPink)+"          Instantly teleport to a bookmarked directory")
	fmt.Fprintln(ctx.Stdout, "  "+color.Colorize("marks", color.PastelPink)+"              List all saved directory bookmarks in pastel table")
	fmt.Fprintln(ctx.Stdout, "  "+color.Colorize("why", color.PastelPink)+"                Command Doctor: Diagnoses previous error with typo suggestions")
	fmt.Fprintln(ctx.Stdout, "  "+color.Colorize("explain <cmd>", color.PastelPink)+"       Explains command syntax and flags")
	fmt.Fprintln(ctx.Stdout, "  "+color.Colorize("calc <expr>", color.PastelPink)+"         Fast inline math expression evaluator e.g. calc (10+2)*5")
	fmt.Fprintln(ctx.Stdout, "  "+color.Colorize("dashboard", color.PastelPink)+"           Pastel session HUD with system & shell telemetry")
	fmt.Fprintln(ctx.Stdout)

	fmt.Fprintln(ctx.Stdout, color.BoldColorize("KEYBOARD SHORTCUTS:", color.PastelYellow))
	fmt.Fprintln(ctx.Stdout, "  "+color.Colorize("Tab", color.Lavender)+"                 Smart auto-completion (paths, commands, env vars, bookmarks)")
	fmt.Fprintln(ctx.Stdout, "  "+color.Colorize("Right Arrow", color.Lavender)+"         Accept fish-style inline history autosuggestion")
	fmt.Fprintln(ctx.Stdout, "  "+color.Colorize("Up / Down", color.Lavender)+"           Browse command history with prefix filtering")
	fmt.Fprintln(ctx.Stdout, "  "+color.Colorize("Ctrl + R / Ctrl + P", color.Lavender)+" Interactive fuzzy history search & command palette")
	fmt.Fprintln(ctx.Stdout, "  "+color.Colorize("Ctrl + A / Ctrl + E", color.Lavender)+" Move cursor to beginning / end of line")
	fmt.Fprintln(ctx.Stdout, "  "+color.Colorize("Ctrl + U / Ctrl + K", color.Lavender)+" Kill text to start / end of line")
	fmt.Fprintln(ctx.Stdout, "  "+color.Colorize("Ctrl + C", color.Lavender)+"            Cancel current command line")
	fmt.Fprintln(ctx.Stdout, "  "+color.Colorize("Ctrl + D", color.Lavender)+"            Exit shell (on empty line)")
	fmt.Fprintln(ctx.Stdout)
	return 0
}

func builtinAbout(args []string, ctx *ShellContext) int {
	fmt.Fprintln(ctx.Stdout, color.BoldColorize("+-----------------------------------------------------------+", color.Lavender))
	fmt.Fprintln(ctx.Stdout, color.BoldColorize("|                    BUSH SHELL (bush)                      |", color.Lavender))
	fmt.Fprintln(ctx.Stdout, color.BoldColorize("|              FOR BUSH LOVERS, BY BUSH LOVERS              |", color.Mauve))
	fmt.Fprintln(ctx.Stdout, color.BoldColorize("+-----------------------------------------------------------+", color.Lavender))
	fmt.Fprintln(ctx.Stdout, "  "+color.BoldColorize("\"A shell for people of refined taste.\"", color.PastelPink))
	fmt.Fprintln(ctx.Stdout)
	fmt.Fprintln(ctx.Stdout, "  "+color.Colorize("Version:      2.1.1 release", color.PastelMint))
	fmt.Fprintln(ctx.Stdout, "  "+color.Colorize("Engine:       Go POSIX", color.TextWhite))
	fmt.Fprintln(ctx.Stdout, "  "+color.Colorize("Theme:        default", color.Lavender))
	fmt.Fprintln(ctx.Stdout, "  "+color.Colorize("Source:       https://github.com/NilayShenai/bush", color.PastelPeach))
	fmt.Fprintln(ctx.Stdout)
	fmt.Fprintln(ctx.Stdout, "  "+color.Colorize("Crafted for anyone who appreciates high performance, endless customizability,", color.TextWhite))
	fmt.Fprintln(ctx.Stdout, "  "+color.Colorize("peak Unix shellcraft, and ", color.TextWhite)+
		color.BoldColorize("the bush", color.PastelPink)+
		color.Colorize(". Bring it back.", color.TextWhite))
	fmt.Fprintln(ctx.Stdout)
	return 0
}
