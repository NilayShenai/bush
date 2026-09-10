package builtins

import (
	"bytes"
	"strings"
	"testing"
)

func TestBuiltinEcho(t *testing.T) {
	var out bytes.Buffer
	ctx := &ShellContext{
		Stdout: &out,
		Stderr: &out,
	}

	code := builtinEcho([]string{"echo", "hello", "world"}, ctx)
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	if strings.TrimSpace(out.String()) != "hello world" {
		t.Fatalf("expected 'hello world', got '%s'", out.String())
	}
}

func TestBuiltinCalc(t *testing.T) {
	var out bytes.Buffer
	ctx := &ShellContext{
		Stdout: &out,
		Stderr: &out,
	}

	code := builtinCalc([]string{"calc", "(10 + 20) * 3 / 2"}, ctx)
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	if !strings.Contains(out.String(), "45") {
		t.Fatalf("expected output to contain '45', got '%s'", out.String())
	}
}

func TestBuiltinMarksAndJump(t *testing.T) {
	var out bytes.Buffer
	ctx := &ShellContext{
		Stdout:    &out,
		Stderr:    &out,
		Bookmarks: make(map[string]string),
	}

	code := builtinMark([]string{"mark", "testdir"}, ctx)
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}

	if _, exists := ctx.Bookmarks["testdir"]; !exists {
		t.Fatalf("expected bookmark 'testdir' to be saved")
	}
}
