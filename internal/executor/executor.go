package executor

import (
	"bush/internal/ast"
	"bush/internal/builtins"
	"bush/internal/expander"
	"bush/internal/guard"
	"bush/internal/jobcontrol"
	"bush/internal/parser"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

type Executor struct {
	Ctx *builtins.ShellContext
}

func NewExecutor(ctx *builtins.ShellContext) *Executor {
	execInstance := &Executor{Ctx: ctx}
	ctx.SubshellRunner = execInstance.RunSubshell
	return execInstance
}

func (e *Executor) RunString(input string) int {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return 0
	}

	cmdList, err := parser.Parse(input)
	if err != nil {
		fmt.Fprintf(e.Ctx.Stderr, "bush: syntax error: %v\n", err)
		e.Ctx.LastExitCode = 2
		return 2
	}

	return e.ExecuteCommandList(cmdList)
}

func (e *Executor) RunSubshell(cmd string) (string, error) {
	var outBuf bytes.Buffer
	subCtx := &builtins.ShellContext{
		Stdin:        e.Ctx.Stdin,
		Stdout:       &outBuf,
		Stderr:       e.Ctx.Stderr,
		LastExitCode: e.Ctx.LastExitCode,
		Aliases:      e.Ctx.Aliases,
		Bookmarks:    e.Ctx.Bookmarks,
	}
	subExec := NewExecutor(subCtx)
	code := subExec.RunString(cmd)
	if code != 0 {
		return outBuf.String(), fmt.Errorf("exit code %d", code)
	}
	return outBuf.String(), nil
}

func (e *Executor) ExecuteCommandList(cmdList *ast.CommandList) int {
	lastCode := 0

	for i := 0; i < len(cmdList.Pipelines); i++ {
		chained := cmdList.Pipelines[i]
		code := e.ExecutePipeline(chained.Pipeline)
		lastCode = code
		isWhy := len(chained.Pipeline.Commands) == 1 && len(chained.Pipeline.Commands[0].Args) > 0 && chained.Pipeline.Commands[0].Args[0] == "why"
		if !isWhy {
			e.Ctx.LastExitCode = code
		}

		e.Ctx.CommandCount++

		if chained.Operator == ast.OpAnd && code != 0 {

			break
		}
		if chained.Operator == ast.OpOr && code == 0 {

			break
		}
	}

	return lastCode
}

func (e *Executor) ExecutePipeline(pipe *ast.Pipeline) int {
	if len(pipe.Commands) == 0 {
		return 0
	}

	if len(pipe.Commands) == 1 {
		cmd := pipe.Commands[0]
		cmd.Background = pipe.Background
		return e.ExecuteSingleCommand(cmd, e.Ctx.Stdin, e.Ctx.Stdout, e.Ctx.Stderr)
	}

	numCmds := len(pipe.Commands)
	type pipeEnds struct {
		r *os.File
		w *os.File
	}
	pipes := make([]pipeEnds, numCmds-1)
	for i := 0; i < numCmds-1; i++ {
		r, w, err := os.Pipe()
		if err != nil {
			fmt.Fprintf(e.Ctx.Stderr, "bush: pipe error: %v\n", err)
			return 1
		}
		pipes[i] = pipeEnds{r: r, w: w}
	}

	var exitCodes []int
	var cmds []*exec.Cmd

	for i, cmd := range pipe.Commands {
		var in io.Reader = e.Ctx.Stdin
		var out io.Writer = e.Ctx.Stdout

		if i > 0 {
			in = pipes[i-1].r
		}
		if i < numCmds-1 {
			out = pipes[i].w
		}

		expCtx := &expander.Context{
			LastExitCode:   e.Ctx.LastExitCode,
			SubshellRunner: e.Ctx.SubshellRunner,
		}
		expandedArgs := expander.ExpandArgs(cmd.Args, expCtx)
		if len(expandedArgs) == 0 {
			continue
		}

		if aliasVal, ok := e.Ctx.Aliases[expandedArgs[0]]; ok {
			aliasParts := strings.Fields(aliasVal)
			expandedArgs = append(aliasParts, expandedArgs[1:]...)
		}

		if builtins.IsBuiltin(expandedArgs[0]) {

			fn, _ := builtins.GetBuiltin(expandedArgs[0])
			go func(args []string, stdin io.Reader, stdout io.Writer, closeW *os.File) {
				subCtx := &builtins.ShellContext{
					Stdin:          stdin,
					Stdout:         stdout,
					Stderr:         e.Ctx.Stderr,
					LastExitCode:   e.Ctx.LastExitCode,
					Aliases:        e.Ctx.Aliases,
					Bookmarks:      e.Ctx.Bookmarks,
					SubshellRunner: e.Ctx.SubshellRunner,
				}
				_ = fn(args, subCtx)
				if closeW != nil {
					_ = closeW.Close()
				}
			}(expandedArgs, in, out, func() *os.File {
				if i < numCmds-1 {
					return pipes[i].w
				}
				return nil
			}())
			continue
		}

		ec := exec.Command(expandedArgs[0], expandedArgs[1:]...)
		ec.Stdin = in
		ec.Stdout = out
		ec.Stderr = e.Ctx.Stderr

		for k, v := range cmd.Env {
			ec.Env = append(os.Environ(), fmt.Sprintf("%s=%s", k, v))
		}

		err := ec.Start()
		if err != nil {
			fmt.Fprintf(e.Ctx.Stderr, "bush: %v\n", err)
			exitCodes = append(exitCodes, 127)
			continue
		}
		cmds = append(cmds, ec)

		if i < numCmds-1 {
			pipes[i].w.Close()
		}
		if i > 0 {
			pipes[i-1].r.Close()
		}
	}

	lastExit := 0
	for _, c := range cmds {
		err := c.Wait()
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				lastExit = exitErr.ExitCode()
			} else {
				lastExit = 1
			}
		} else {
			lastExit = 0
		}
	}

	return lastExit
}

func (e *Executor) ExecuteSingleCommand(cmd *ast.Command, in io.Reader, out io.Writer, errW io.Writer) int {
	expCtx := &expander.Context{
		LastExitCode:   e.Ctx.LastExitCode,
		SubshellRunner: e.Ctx.SubshellRunner,
	}

	if len(cmd.Args) == 0 && len(cmd.Env) > 0 {
		for k, v := range cmd.Env {
			expV := expander.ExpandToken(v, expCtx)
			if len(expV) > 0 {
				os.Setenv(k, expV[0])
			} else {
				os.Setenv(k, "")
			}
		}
		return 0
	}

	expandedArgs := expander.ExpandArgs(cmd.Args, expCtx)
	if len(expandedArgs) == 0 {
		return 0
	}

	cmdName := expandedArgs[0]
	if cmdName != "why" {
		e.Ctx.LastCommand = strings.Join(expandedArgs, " ")
	}
	if e.Ctx.CommandFreq == nil {
		e.Ctx.CommandFreq = make(map[string]int)
	}
	e.Ctx.CommandFreq[cmdName]++

	if aliasVal, ok := e.Ctx.Aliases[cmdName]; ok {
		aliasParts := strings.Fields(aliasVal)
		expandedArgs = append(aliasParts, expandedArgs[1:]...)
		cmdName = expandedArgs[0]
	}

	actualIn := in
	actualOut := out
	actualErr := errW

	var openFiles []*os.File
	defer func() {
		for _, f := range openFiles {
			_ = f.Close()
		}
	}()

	for _, redir := range cmd.Redirects {
		targetPath := expander.ExpandRedirectionTarget(redir.Target, expCtx)

		switch redir.Type {
		case ast.RedirIn:
			f, err := os.Open(targetPath)
			if err != nil {
				fmt.Fprintf(e.Ctx.Stderr, "bush: %v\n", err)
				return 1
			}
			openFiles = append(openFiles, f)
			actualIn = f

		case ast.RedirOut:
			f, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
			if err != nil {
				fmt.Fprintf(e.Ctx.Stderr, "bush: %v\n", err)
				return 1
			}
			openFiles = append(openFiles, f)
			actualOut = f

		case ast.RedirAppend:
			f, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err != nil {
				fmt.Fprintf(e.Ctx.Stderr, "bush: %v\n", err)
				return 1
			}
			openFiles = append(openFiles, f)
			actualOut = f

		case ast.RedirErr:
			f, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
			if err != nil {
				fmt.Fprintf(e.Ctx.Stderr, "bush: %v\n", err)
				return 1
			}
			openFiles = append(openFiles, f)
			actualErr = f

		case ast.RedirErrApp:
			f, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err != nil {
				fmt.Fprintf(e.Ctx.Stderr, "bush: %v\n", err)
				return 1
			}
			openFiles = append(openFiles, f)
			actualErr = f

		case ast.RedirErrToOut:
			actualErr = actualOut

		case ast.RedirAll:
			f, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
			if err != nil {
				fmt.Fprintf(e.Ctx.Stderr, "bush: %v\n", err)
				return 1
			}
			openFiles = append(openFiles, f)
			actualOut = f
			actualErr = f
		}
	}

	if builtins.IsBuiltin(cmdName) {
		fn, _ := builtins.GetBuiltin(cmdName)
		callCtx := &builtins.ShellContext{
			Stdin:          actualIn,
			Stdout:         actualOut,
			Stderr:         actualErr,
			LastExitCode:   e.Ctx.LastExitCode,
			LastCommand:    e.Ctx.LastCommand,
			StartTime:      e.Ctx.StartTime,
			CommandCount:   e.Ctx.CommandCount,
			CommandFreq:    e.Ctx.CommandFreq,
			Aliases:        e.Ctx.Aliases,
			Bookmarks:      e.Ctx.Bookmarks,
			HistoryList:    e.Ctx.HistoryList,
			SubshellRunner: e.Ctx.SubshellRunner,
			ExitShell:      e.Ctx.ExitShell,
		}
		return fn(expandedArgs, callCtx)
	}

	path, err := exec.LookPath(cmdName)
	if err != nil {
		fmt.Fprintf(e.Ctx.Stderr, "bush: %s: command not found\n", cmdName)
		return 127
	}

	execArgs := expandedArgs[1:]
	if cmdName == "su" {
		hasShell := false
		for _, arg := range execArgs {
			if arg == "-s" || strings.HasPrefix(arg, "--shell") {
				hasShell = true
				break
			}
		}
		if !hasShell {
			if selfExe, err := os.Executable(); err == nil {
				execArgs = append([]string{"-s", selfExe}, execArgs...)
			}
		}
	}

	ec := exec.Command(path, execArgs...)
	ec.Stdin = actualIn
	ec.Stdout = actualOut
	ec.Stderr = actualErr

	if len(cmd.Env) > 0 {
		env := os.Environ()
		for k, v := range cmd.Env {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		ec.Env = env
	}

	if alert := guard.CheckBlastRadius(cmdName, execArgs); alert != nil && alert.Triggered {
		if !guard.ConfirmExecution(alert, strings.Join(expandedArgs, " ")) {
			return 130
		}
	}

	if cmd.Background {
		err := ec.Start()
		if err != nil {
			fmt.Fprintf(e.Ctx.Stderr, "bush: %v\n", err)
			return 1
		}
		jobcontrol.DefaultManager.AddJob(ec, strings.Join(expandedArgs, " "))
		return 0
	}

	err = ec.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				if status.Signaled() {
					sig := status.Signal()
					return 128 + int(sig)
				}
				return status.ExitStatus()
			}
			return exitErr.ExitCode()
		}
		return 1
	}

	return 0
}

func LookExecutable(name string) bool {
	if builtins.IsBuiltin(name) {
		return true
	}
	if strings.Contains(name, "/") {
		info, err := os.Stat(name)
		return err == nil && !info.IsDir() && (info.Mode()&0111 != 0)
	}
	_, err := exec.LookPath(name)
	return err == nil
}

func FindPathCompletions(prefix string) []string {
	var matches []string

	for _, b := range builtins.GetBuiltinNames() {
		if strings.HasPrefix(b, prefix) {
			matches = append(matches, b)
		}
	}

	if strings.Contains(prefix, "/") || strings.HasPrefix(prefix, ".") || strings.HasPrefix(prefix, "~") {
		return FindFileCompletions(prefix)
	}

	pathEnv := os.Getenv("PATH")
	dirs := filepath.SplitList(pathEnv)
	seen := make(map[string]bool)

	for _, dir := range dirs {
		files, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, f := range files {
			name := f.Name()
			if strings.HasPrefix(name, prefix) && !seen[name] {
				seen[name] = true
				matches = append(matches, name)
			}
		}
	}

	return matches
}

func FindFileCompletions(prefix string) []string {
	var matches []string
	dir := "."
	base := prefix

	if strings.HasPrefix(prefix, "~/") {
		home, _ := os.UserHomeDir()
		prefix = filepath.Join(home, prefix[2:])
	}

	if idx := strings.LastIndex(prefix, "/"); idx >= 0 {
		dir = prefix[:idx+1]
		base = prefix[idx+1:]
		if dir == "" {
			dir = "/"
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return matches
	}

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, base) {
			full := filepath.Join(dir, name)
			if entry.IsDir() {
				full += "/"
			}
			matches = append(matches, full)
		}
	}

	return matches
}
