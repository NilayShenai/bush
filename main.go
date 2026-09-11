package main

import (
	"bush/internal/builtins"
	"bush/internal/color"
	"bush/internal/config"
	"bush/internal/executor"
	"bush/internal/jobcontrol"
	"bush/internal/lineeditor"
	"bush/internal/prompt"
	"bush/internal/terminal"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

var Version = "2.8.1"

func main() {
	cmdFlag := flag.String("c", "", "Execute command string and exit")
	versionFlag := flag.Bool("v", false, "Print Bush version")
	flag.BoolVar(versionFlag, "version", false, "Print Bush version")
	updateFlag := flag.Bool("u", false, "Check for updates and upgrade Bush")
	flag.BoolVar(updateFlag, "update", false, "Check for updates and upgrade Bush")
	loginFlag := flag.Bool("l", false, "Start as a login shell")
	flag.BoolVar(loginFlag, "login", false, "Start as a login shell")
	interactiveFlag := flag.Bool("i", false, "Force interactive shell")
	stdinFlag := flag.Bool("s", false, "Read commands from standard input")
	helpFlag := flag.Bool("h", false, "Show help")
	flag.BoolVar(helpFlag, "help", false, "Show help")
	flag.Parse()

	_ = interactiveFlag

	if *helpFlag {
		fmt.Printf("Usage: bush [-l|--login] [-i] [-s] [-c command] [-u|--update] [-v|--version] [script.sh]\n")
		os.Exit(0)
	}

	if *versionFlag {
		printVersion()
		os.Exit(0)
	}

	ctx := &builtins.ShellContext{
		Stdin:       os.Stdin,
		Stdout:      os.Stdout,
		Stderr:      os.Stderr,
		Version:     Version,
		StartTime:   time.Now().Unix(),
		Aliases:     make(map[string]string),
		Bookmarks:   builtins.LoadBookmarks(),
		CommandFreq: make(map[string]int),
		ExitShell: func(code int) {
			os.Exit(code)
		},
	}

	exec := executor.NewExecutor(ctx)

	if exePath, err := os.Executable(); err == nil {
		os.Setenv("SHELL", exePath)
	}

	isLogin := *loginFlag || strings.HasPrefix(filepath.Base(os.Args[0]), "-")
	if isLogin {
		os.Setenv("LOGIN_SHELL", "1")
		os.Setenv("SHLVL", "1")
	}

	config.LoadConfig(ctx)

	if *updateFlag || (len(flag.Args()) > 0 && flag.Args()[0] == "update") {
		updateArgs := []string{"update"}
		if len(flag.Args()) > 1 && flag.Args()[0] == "update" {
			updateArgs = append(updateArgs, flag.Args()[1:]...)
		}
		if fn, ok := builtins.GetBuiltin("update"); ok {
			code := fn(updateArgs, ctx)
			os.Exit(code)
		}
	}

	if *cmdFlag != "" {
		code := exec.RunString(*cmdFlag)
		os.Exit(code)
	}

	if *stdinFlag && !terminal.IsTerminal(terminal.StdinFd()) {
		var line string
		exitCode := 0
		for {
			_, err := fmt.Scanln(&line)
			if err != nil {
				break
			}
			exitCode = exec.RunString(line)
		}
		os.Exit(exitCode)
	}

	args := flag.Args()
	if len(args) > 0 {
		scriptFile := args[0]
		data, err := os.ReadFile(scriptFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "bush: %s: %v\n", scriptFile, err)
			os.Exit(1)
		}
		lines := strings.Split(string(data), "\n")
		exitCode := 0
		for _, line := range lines {
			exitCode = exec.RunString(line)
		}
		os.Exit(exitCode)
	}

	runInteractiveREPL(exec, ctx)
}

func printVersion() {
	fmt.Println(color.BoldColorize("Bush Shell", color.ActivePalette.Accent) + " version " + color.Colorize(Version+" release", color.ActivePalette.Directory) + " (For Bush Lovers, By Bush Lovers)")
}

func printWelcomeBanner() {
	cfg := config.Get()
	if !cfg.Shell.GreetingBanner {
		return
	}

	motto := cfg.Shell.Motto
	if motto == "" {
		motto = "A shell for people of refined taste."
	}

	pal := color.ActivePalette
	fmt.Println(color.BoldColorize("+-------------------------------------------------------------+", pal.Accent))
	fmt.Println(color.BoldColorize("|                    BUSH SHELL (bush)                        |", pal.Accent))
	fmt.Println(color.BoldColorize("|              FOR BUSH LOVERS, BY BUSH LOVERS                |", pal.PromptSymbol))
	fmt.Println(color.BoldColorize("+-------------------------------------------------------------+", pal.Accent))
	fmt.Println("  " + color.BoldColorize(fmt.Sprintf("\"%s\"", motto), pal.GitBranch))
	if cfg.Shell.CheckUpdates {
		if notice := builtins.CheckStartupUpdateNotice(Version); notice != "" {
			fmt.Println(notice)
			fmt.Println()
		}
	}

	fmt.Println("  " + color.Colorize("Type", pal.GhostText) + " " +
		color.BoldColorize("help", pal.Directory) + " " +
		color.Colorize("for commands,", pal.GhostText) + " " +
		color.BoldColorize("config", pal.Flags) + " " +
		color.Colorize("to customize,", pal.GhostText) + " " +
		color.BoldColorize("update", pal.Success) + " " +
		color.Colorize("to upgrade,", pal.GhostText) + " " +
		color.BoldColorize("dashboard", pal.GitBranch) + " " +
		color.Colorize("for session stats,", pal.GhostText) + " " +
		color.BoldColorize("why", pal.Flags) + " " +
		color.Colorize("for command doctor.", pal.GhostText))
	fmt.Println()
}

func runInteractiveREPL(exec *executor.Executor, ctx *builtins.ShellContext) {
	printWelcomeBanner()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTSTP)
	go func() {
		for range sigChan {

		}
	}()

	editor := lineeditor.NewLineEditor(ctx)
	promptState := &prompt.State{
		LastExitCode: 0,
		LastDuration: 0,
	}

	for {

		jobcontrol.DefaultManager.CheckJobs()

		line, err := editor.ReadLine(promptState)
		if err != nil {
			if err == io.EOF {
				fmt.Println(color.Colorize("exit", color.PastelGray))
				break
			}
			continue
		}

		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		start := time.Now()
		code := exec.RunString(trimmed)
		duration := time.Since(start)

		promptState.LastExitCode = code
		promptState.LastDuration = duration
		ctx.LastExitCode = code
	}
}
