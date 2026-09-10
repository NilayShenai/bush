package builtins

import (
	"bytes"
	"strings"
	"testing"
)

func TestMemoCaptureAndReplay(t *testing.T) {
	var out bytes.Buffer
	ctx := &ShellContext{
		Stdout: &out,
		Stderr: &out,
		SubshellRunner: func(cmd string) (string, error) {
			return "computed output: 42\n", nil
		},
	}

	// 1. Memoize execution
	code := builtinMemo([]string{"memo", "compute", "something"}, ctx)
	if code != 0 {
		t.Fatalf("expected memo exit 0, got %d", code)
	}
	if !strings.Contains(out.String(), "computed output: 42") {
		t.Fatalf("expected memo output, got '%s'", out.String())
	}

	// 2. Replay memoized output
	out.Reset()
	code = builtinMemo([]string{"memo"}, ctx)
	if code != 0 {
		t.Fatalf("expected memo replay exit 0, got %d", code)
	}
	if !strings.Contains(out.String(), "computed output: 42") {
		t.Fatalf("expected replayed memo output, got '%s'", out.String())
	}

	// 3. Clear cache
	out.Reset()
	code = builtinMemo([]string{"memo", "clear"}, ctx)
	if code != 0 {
		t.Fatalf("expected memo clear exit 0, got %d", code)
	}
}
