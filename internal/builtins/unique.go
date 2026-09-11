package builtins

import (
	"bufio"
	"bush/internal/color"
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

func init() {
	Register("peek", builtinPeek)
	Register("mark", builtinMark)
	Register("jump", builtinJump)
	Register("marks", builtinMarks)
	Register("why", builtinWhy)
	Register("explain", builtinExplain)
	Register("calc", builtinCalc)
	Register("=", builtinCalc)
	Register("dashboard", builtinDashboard)
	Register("stats", builtinDashboard)
}

func builtinPeek(args []string, ctx *ShellContext) int {

	if len(args) > 1 && !strings.HasPrefix(args[1], "-") {
		filePath := args[1]
		info, err := os.Stat(filePath)
		if err != nil {
			fmt.Fprintf(ctx.Stderr, "bush: peek: %v\n", err)
			return 1
		}

		fmt.Fprintln(ctx.Stderr, color.BoldColorize("+--- Bush File Inspector ---------------------------------+", color.Lavender))
		fmt.Fprintf(ctx.Stderr, "| %s: %s\n", color.Colorize("Target File", color.PastelPeach), info.Name())
		fmt.Fprintf(ctx.Stderr, "| %s: %s\n", color.Colorize("Size", color.PastelPeach), formatBytes(info.Size()))
		fmt.Fprintf(ctx.Stderr, "| %s: %s\n", color.Colorize("Mode", color.PastelPeach), info.Mode().String())
		fmt.Fprintf(ctx.Stderr, "| %s: %s\n", color.Colorize("Modified", color.PastelPeach), info.ModTime().Format("2006-01-02 15:04:05"))

		f, err := os.Open(filePath)
		if err == nil {
			defer f.Close()
			scanner := bufio.NewScanner(f)
			fmt.Fprintf(ctx.Stderr, "| %s:\n", color.Colorize("Content Preview (first 5 lines)", color.PastelMint))
			lines := 0
			for scanner.Scan() && lines < 5 {
				fmt.Fprintf(ctx.Stderr, "|   | %s\n", scanner.Text())
				lines++
			}
		}
		fmt.Fprintln(ctx.Stderr, color.BoldColorize("+---------------------------------------------------------+", color.Lavender))
		return 0
	}

	var totalBytes int64
	lineCount := 0
	startTime := time.Now()
	buf := make([]byte, 32*1024)
	isJSON := true
	var sample []byte

	for {
		n, err := ctx.Stdin.Read(buf)
		if n > 0 {
			totalBytes += int64(n)
			lineCount += bytes.Count(buf[:n], []byte{'\n'})
			if len(sample) < 1024 {
				sample = append(sample, buf[:min(n, 1024-len(sample))]...)
			}
			_, _ = ctx.Stdout.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}

	trimmedSample := strings.TrimSpace(string(sample))
	if !strings.HasPrefix(trimmedSample, "{") && !strings.HasPrefix(trimmedSample, "[") {
		isJSON = false
	}

	formatDetected := "Plaintext"
	if isJSON && len(sample) > 0 {
		var js interface{}
		if json.Unmarshal(sample, &js) == nil {
			formatDetected = "JSON"
		}
	}

	elapsed := time.Since(startTime)
	var speedStr string
	if elapsed.Seconds() > 0 {
		mbSec := float64(totalBytes) / (1024 * 1024) / elapsed.Seconds()
		speedStr = fmt.Sprintf("%.2f MB/s", mbSec)
	} else {
		speedStr = "instant"
	}

	badge := fmt.Sprintf("[peek] %s | %d lines | Format: %s | Speed: %s",
		formatBytes(totalBytes), lineCount, formatDetected, speedStr)
	fmt.Fprintln(ctx.Stderr, color.ColorizeBg(" "+badge+" ", color.SurfaceDark, color.Lavender))
	return 0
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func builtinMark(args []string, ctx *ShellContext) int {
	if len(args) < 2 {
		fmt.Fprintf(ctx.Stderr, "bush: mark: usage: mark <tag-name>\n")
		return 1
	}
	tag := args[1]
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(ctx.Stderr, "bush: mark: %v\n", err)
		return 1
	}

	if ctx.Bookmarks == nil {
		ctx.Bookmarks = LoadBookmarks()
	}
	ctx.Bookmarks[tag] = cwd
	_ = SaveBookmarks(ctx.Bookmarks)

	fmt.Fprintf(ctx.Stdout, "[mark] Saved %s -> %s\n",
		color.BoldColorize(tag, color.PastelPink),
		color.Colorize(cwd, color.PastelMint),
	)
	return 0
}

func builtinJump(args []string, ctx *ShellContext) int {
	if len(args) < 2 {
		fmt.Fprintf(ctx.Stderr, "bush: jump: usage: jump <tag-name>\n")
		return 1
	}
	tag := args[1]
	if ctx.Bookmarks == nil {
		ctx.Bookmarks = LoadBookmarks()
	}

	targetDir, ok := ctx.Bookmarks[tag]
	if !ok {
		fmt.Fprintf(ctx.Stderr, "bush: jump: mark '%s' does not exist. Run 'marks' to view all.\n", tag)
		return 1
	}

	return builtinCd([]string{"cd", targetDir}, ctx)
}

func builtinMarks(args []string, ctx *ShellContext) int {
	if ctx.Bookmarks == nil {
		ctx.Bookmarks = LoadBookmarks()
	}

	if len(ctx.Bookmarks) == 0 {
		fmt.Fprintln(ctx.Stdout, color.Colorize("No bookmarks saved yet. Use 'mark <tag>' to bookmark current directory.", color.PastelGray))
		return 0
	}

	fmt.Fprintln(ctx.Stdout, color.BoldColorize("+--- Saved Directory Bookmarks ----------------------------+", color.Lavender))
	var tags []string
	for tag := range ctx.Bookmarks {
		tags = append(tags, tag)
	}
	sort.Strings(tags)

	for _, tag := range tags {
		fmt.Fprintf(ctx.Stdout, "|  %-14s ->  %s\n",
			color.BoldColorize(tag, color.PastelPink),
			color.Colorize(ctx.Bookmarks[tag], color.PastelMint),
		)
	}
	fmt.Fprintln(ctx.Stdout, color.BoldColorize("+----------------------------------------------------------+", color.Lavender))
	return 0
}

func builtinWhy(args []string, ctx *ShellContext) int {
	lastCode := ctx.LastExitCode
	lastCmd := ctx.LastCommand

	fmt.Fprintln(ctx.Stdout, color.BoldColorize("+--- Bush Command Doctor ----------------------------------+", color.Lavender))

	if lastCmd == "" {
		fmt.Fprintln(ctx.Stdout, "| No previous command has been executed in this session.")
		fmt.Fprintln(ctx.Stdout, color.BoldColorize("+----------------------------------------------------------+", color.Lavender))
		return 0
	}

	cmdParts := strings.Fields(lastCmd)
	cmdBase := ""
	if len(cmdParts) > 0 {
		cmdBase = cmdParts[0]
	}

	fmt.Fprintf(ctx.Stdout, "| %s: %s\n", color.Colorize("Last Command", color.PastelPeach), lastCmd)
	statusText := color.Colorize(fmt.Sprintf("%d (Success)", lastCode), color.PastelMint)
	if lastCode != 0 {
		statusText = color.Colorize(fmt.Sprintf("%d (Error)", lastCode), color.PastelRed)
	}
	fmt.Fprintf(ctx.Stdout, "| %s: %s\n", color.Colorize("Exit Code", color.PastelPeach), statusText)
	fmt.Fprintln(ctx.Stdout, "|")

	switch lastCode {
	case 0:
		fmt.Fprintln(ctx.Stdout, "| "+color.Colorize("The previous command completed successfully with no errors.", color.PastelMint))
	case 127:
		if path, err := exec.LookPath(cmdBase); err == nil && !IsBuiltin(cmdBase) {
			fmt.Fprintln(ctx.Stdout, "| "+color.BoldColorize("Diagnostic: Child Command or Shell Not Found", color.PastelRed))
			fmt.Fprintf(ctx.Stdout, "| Executable '%s' was found at %s,\n", cmdBase, path)
			fmt.Fprintln(ctx.Stdout, "| but a child command, script interpreter, or login shell invoked by it was not found.")
			if cmdBase == "su" || cmdBase == "sudo" {
				fmt.Fprintln(ctx.Stdout, "| Tip: The target user's login shell in /etc/passwd points to a missing file.")
			}
		} else {
			fmt.Fprintln(ctx.Stdout, "| "+color.BoldColorize("Diagnostic: Command Not Found", color.PastelRed))
			fmt.Fprintf(ctx.Stdout, "| The system could not find executable '%s' in $PATH or built-ins.\n", cmdBase)
			suggestion := findClosestMatch(cmdBase)
			if suggestion != "" {
				fmt.Fprintf(ctx.Stdout, "| Suggestion: Did you mean '%s'?\n",
					color.BoldColorize(suggestion, color.PastelMint),
				)
			} else {
				fmt.Fprintln(ctx.Stdout, "| Suggestion: Check spelling, install the package, or check your $PATH.")
			}
		}
	case 126:
		fmt.Fprintln(ctx.Stdout, "| "+color.BoldColorize("Diagnostic: Permission Denied / Not Executable", color.PastelRed))
		fmt.Fprintln(ctx.Stdout, "| Suggestion: Run 'chmod +x <file>' or ensure you have read/execute privileges.")
	case 130:
		fmt.Fprintln(ctx.Stdout, "| "+color.BoldColorize("Diagnostic: User Interrupt (SIGINT)", color.PastelPeach))
		fmt.Fprintln(ctx.Stdout, "| The command was cancelled via keyboard interrupt (Ctrl+C).")
	case 137:
		fmt.Fprintln(ctx.Stdout, "| "+color.BoldColorize("Diagnostic: Fatal Process Kill (SIGKILL / OOM)", color.PastelRed))
		fmt.Fprintln(ctx.Stdout, "| The OS killed the process immediately (likely Out-Of-Memory killer or 'kill -9').")
	case 143:
		fmt.Fprintln(ctx.Stdout, "| "+color.BoldColorize("Diagnostic: Terminated (SIGTERM)", color.PastelPeach))
		fmt.Fprintln(ctx.Stdout, "| The process was sent a termination signal.")
	default:
		fmt.Fprintln(ctx.Stdout, "| "+color.BoldColorize("Diagnostic: General Command Failure", color.PastelPeach))
		fmt.Fprintln(ctx.Stdout, "| The command exited with a non-zero status code.")
		for _, part := range cmdParts[1:] {
			if !strings.HasPrefix(part, "-") && strings.Contains(part, "/") {
				if _, err := os.Stat(part); os.IsNotExist(err) {
					fmt.Fprintf(ctx.Stdout, "| Warning: Target file/path '%s' was not found.\n", part)
				}
			}
		}
	}

	fmt.Fprintln(ctx.Stdout, color.BoldColorize("+----------------------------------------------------------+", color.Lavender))
	return 0
}

func builtinExplain(args []string, ctx *ShellContext) int {
	if len(args) < 2 {
		fmt.Fprintf(ctx.Stderr, "bush: explain: usage: explain <command>\n")
		return 1
	}
	cmd := args[1]
	fmt.Fprintln(ctx.Stdout, color.BoldColorize("+--- Command Explainer ------------------------------------+", color.Lavender))
	fmt.Fprintf(ctx.Stdout, "| Explaining: %s\n", color.BoldColorize(cmd, color.PastelMint))
	fmt.Fprintln(ctx.Stdout, "|")

	if IsBuiltin(cmd) {
		fmt.Fprintf(ctx.Stdout, "| '%s' is an ultra-fast built-in command in Bush Shell.\n", cmd)
		fmt.Fprintln(ctx.Stdout, "| Type 'help' to read the full built-in manual.")
	} else {
		path, err := exec.LookPath(cmd)
		if err != nil {
			fmt.Fprintf(ctx.Stdout, "| '%s' is not currently found in your $PATH.\n", cmd)
		} else {
			fmt.Fprintf(ctx.Stdout, "| Located at: %s\n", path)
			fmt.Fprintln(ctx.Stdout, "| To view manual pages, run: man "+cmd)
		}
	}
	fmt.Fprintln(ctx.Stdout, color.BoldColorize("+----------------------------------------------------------+", color.Lavender))
	return 0
}

func findClosestMatch(target string) string {
	candidates := GetBuiltinNames()
	candidates = append(candidates, "git", "grep", "docker", "kubectl", "ls", "cat", "vim", "nano", "curl", "python", "node", "cargo")

	bestMatch := ""
	minDist := 999

	for _, cand := range candidates {
		d := levenshtein(target, cand)
		if d < minDist && d <= 3 {
			minDist = d
			bestMatch = cand
		}
	}
	return bestMatch
}

func levenshtein(s1, s2 string) int {
	r1, r2 := []rune(s1), []rune(s2)
	l1, l2 := len(r1), len(r2)
	dp := make([][]int, l1+1)
	for i := range dp {
		dp[i] = make([]int, l2+1)
		dp[i][0] = i
	}
	for j := 0; j <= l2; j++ {
		dp[0][j] = j
	}
	for i := 1; i <= l1; i++ {
		for j := 1; j <= l2; j++ {
			cost := 0
			if r1[i-1] != r2[j-1] {
				cost = 1
			}
			dp[i][j] = min3(
				dp[i-1][j]+1,
				dp[i][j-1]+1,
				dp[i-1][j-1]+cost,
			)
		}
	}
	return dp[l1][l2]
}

func min3(a, b, c int) int {
	if a < b && a < c {
		return a
	}
	if b < c {
		return b
	}
	return c
}

func builtinCalc(args []string, ctx *ShellContext) int {
	if len(args) < 2 {
		fmt.Fprintf(ctx.Stderr, "bush: calc: usage: calc <expression> (e.g. calc 2 * (10 + 4) / 3)\n")
		return 1
	}
	expr := strings.Join(args[1:], " ")
	val, err := evaluateMath(expr)
	if err != nil {
		fmt.Fprintf(ctx.Stderr, "bush: calc: %v\n", err)
		return 1
	}

	resultStr := fmt.Sprintf("%v", val)
	fmt.Fprintf(ctx.Stdout, "[calc] %s = %s\n",
		color.Colorize(expr, color.PastelPeach),
		color.BoldColorize(resultStr, color.PastelMint),
	)
	return 0
}

func evaluateMath(expr string) (float64, error) {
	p := &mathParser{input: expr}
	res, err := p.parseExpr()
	if err != nil {
		return 0, err
	}
	return res, nil
}

type mathParser struct {
	input string
	pos   int
}

func (p *mathParser) parseExpr() (float64, error) {
	val, err := p.parseTerm()
	if err != nil {
		return 0, err
	}
	for {
		p.skipSpace()
		if p.pos >= len(p.input) {
			break
		}
		ch := p.input[p.pos]
		if ch == '+' {
			p.pos++
			term, err := p.parseTerm()
			if err != nil {
				return 0, err
			}
			val += term
		} else if ch == '-' {
			p.pos++
			term, err := p.parseTerm()
			if err != nil {
				return 0, err
			}
			val -= term
		} else {
			break
		}
	}
	return val, nil
}

func (p *mathParser) parseTerm() (float64, error) {
	val, err := p.parseFactor()
	if err != nil {
		return 0, err
	}
	for {
		p.skipSpace()
		if p.pos >= len(p.input) {
			break
		}
		ch := p.input[p.pos]
		if ch == '*' {
			p.pos++
			factor, err := p.parseFactor()
			if err != nil {
				return 0, err
			}
			val *= factor
		} else if ch == '/' {
			p.pos++
			factor, err := p.parseFactor()
			if err != nil {
				return 0, err
			}
			if factor == 0 {
				return 0, fmt.Errorf("division by zero")
			}
			val /= factor
		} else if ch == '%' {
			p.pos++
			factor, err := p.parseFactor()
			if err != nil {
				return 0, err
			}
			val = math.Mod(val, factor)
		} else if ch == '^' {
			p.pos++
			factor, err := p.parseFactor()
			if err != nil {
				return 0, err
			}
			val = math.Pow(val, factor)
		} else {
			break
		}
	}
	return val, nil
}

func (p *mathParser) parseFactor() (float64, error) {
	p.skipSpace()
	if p.pos >= len(p.input) {
		return 0, fmt.Errorf("unexpected end of expression")
	}

	if p.input[p.pos] == '(' {
		p.pos++
		val, err := p.parseExpr()
		if err != nil {
			return 0, err
		}
		p.skipSpace()
		if p.pos < len(p.input) && p.input[p.pos] == ')' {
			p.pos++
			return val, nil
		}
		return 0, fmt.Errorf("missing closing parenthesis ')'")
	}

	if p.input[p.pos] == '-' {
		p.pos++
		val, err := p.parseFactor()
		return -val, err
	}

	start := p.pos
	for p.pos < len(p.input) && unicode.IsLetter(rune(p.input[p.pos])) {
		p.pos++
	}
	if p.pos > start {
		fnName := strings.ToLower(p.input[start:p.pos])
		if fnName == "pi" {
			return math.Pi, nil
		} else if fnName == "e" {
			return math.E, nil
		}
		p.skipSpace()
		if p.pos < len(p.input) && p.input[p.pos] == '(' {
			p.pos++
			arg, err := p.parseExpr()
			if err != nil {
				return 0, err
			}
			p.skipSpace()
			if p.pos < len(p.input) && p.input[p.pos] == ')' {
				p.pos++
			}
			switch fnName {
			case "sqrt":
				return math.Sqrt(arg), nil
			case "abs":
				return math.Abs(arg), nil
			case "round":
				return math.Round(arg), nil
			case "floor":
				return math.Floor(arg), nil
			case "ceil":
				return math.Ceil(arg), nil
			case "sin":
				return math.Sin(arg), nil
			case "cos":
				return math.Cos(arg), nil
			default:
				return 0, fmt.Errorf("unknown math function '%s'", fnName)
			}
		}
		return 0, fmt.Errorf("unknown constant '%s'", fnName)
	}

	numStart := p.pos
	for p.pos < len(p.input) && (unicode.IsDigit(rune(p.input[p.pos])) || p.input[p.pos] == '.') {
		p.pos++
	}
	if numStart == p.pos {
		return 0, fmt.Errorf("invalid token at '%s'", p.input[p.pos:])
	}
	numStr := p.input[numStart:p.pos]
	return strconv.ParseFloat(numStr, 64)
}

func (p *mathParser) skipSpace() {
	for p.pos < len(p.input) && unicode.IsSpace(rune(p.input[p.pos])) {
		p.pos++
	}
}

func builtinDashboard(args []string, ctx *ShellContext) int {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	cwd, _ := os.Getwd()
	gitBranch := getGitBranch()
	uptime := time.Duration(0)
	if ctx.StartTime > 0 {
		uptime = time.Since(time.Unix(ctx.StartTime, 0)).Round(time.Second)
	}

	fmt.Fprintln(ctx.Stdout, color.BoldColorize("+-------------------------------------------------------------+", color.Lavender))
	fmt.Fprintln(ctx.Stdout, color.BoldColorize("|                       BUSH DASHBOARD                        |", color.Lavender))
	fmt.Fprintln(ctx.Stdout, color.BoldColorize("|              FOR BUSH LOVERS, BY BUSH LOVERS                |", color.Mauve))
	ver := ctx.Version
	if ver == "" {
		ver = "2.8.2"
	}
	fmt.Fprintf(ctx.Stdout, "| %-18s: %s\n", color.Colorize("Shell Version", color.PastelPeach), color.Colorize("Bush v"+ver+" release (Go "+runtime.Version()+")", color.TextWhite))
	fmt.Fprintf(ctx.Stdout, "| %-18s: %s\n", color.Colorize("Uptime", color.PastelPeach), color.Colorize(uptime.String(), color.PastelMint))
	fmt.Fprintf(ctx.Stdout, "| %-18s: %s\n", color.Colorize("Commands Executed", color.PastelPeach), color.Colorize(strconv.Itoa(ctx.CommandCount), color.PastelYellow))
	fmt.Fprintf(ctx.Stdout, "| %-18s: %s\n", color.Colorize("Memory Allocated", color.PastelPeach), color.Colorize(formatBytes(int64(m.Alloc)), color.PastelMint))
	fmt.Fprintf(ctx.Stdout, "| %-18s: %s\n", color.Colorize("Working Directory", color.PastelPeach), color.Colorize(cwd, color.TextWhite))
	if gitBranch != "" {
		fmt.Fprintf(ctx.Stdout, "| %-18s: %s\n", color.Colorize("Git Branch", color.PastelPeach), color.Colorize("git:"+gitBranch, color.PastelPink))
	}

	if len(ctx.CommandFreq) > 0 {
		type kv struct {
			Key   string
			Value int
		}
		var ss []kv
		for k, v := range ctx.CommandFreq {
			ss = append(ss, kv{k, v})
		}
		sort.Slice(ss, func(i, j int) bool {
			return ss[i].Value > ss[j].Value
		})

		fmt.Fprintln(ctx.Stdout, "|")
		fmt.Fprintf(ctx.Stdout, "| %s:\n", color.BoldColorize("Top Commands Used", color.PastelPink))
		limit := min(len(ss), 5)
		for i := 0; i < limit; i++ {
			fmt.Fprintf(ctx.Stdout, "|   %d. %-15s %s times\n",
				i+1,
				color.Colorize(ss[i].Key, color.PastelMint),
				color.Colorize(strconv.Itoa(ss[i].Value), color.PastelYellow),
			)
		}
	}

	fmt.Fprintln(ctx.Stdout, color.BoldColorize("+-------------------------------------------------------------+", color.Lavender))
	return 0
}

func getGitBranch() string {
	out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
