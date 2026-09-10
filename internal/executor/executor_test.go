package executor

import (
	"bush/internal/builtins"
	"bytes"
	"strings"
	"testing"
)

func TestExecutorRunString(t *testing.T) {
	var outBuf bytes.Buffer
	var errBuf bytes.Buffer

	ctx := &builtins.ShellContext{
		Stdout:  &outBuf,
		Stderr:  &errBuf,
		Aliases: make(map[string]string),
	}

	exec := NewExecutor(ctx)
	code := exec.RunString("echo testing-executor")
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	if !strings.Contains(outBuf.String(), "testing-executor") {
		t.Fatalf("expected output 'testing-executor', got '%s'", outBuf.String())
	}
}

func TestExecutorPipeline(t *testing.T) {
	var outBuf bytes.Buffer
	var errBuf bytes.Buffer

	ctx := &builtins.ShellContext{
		Stdout:  &outBuf,
		Stderr:  &errBuf,
		Aliases: make(map[string]string),
	}

	exec := NewExecutor(ctx)
	code := exec.RunString("echo -e 'foo\nbar\nbaz' | grep bar")
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	if strings.TrimSpace(outBuf.String()) != "bar" {
		t.Fatalf("expected 'bar', got '%s'", outBuf.String())
	}
}

func TestExecutorChaining(t *testing.T) {
	var outBuf bytes.Buffer
	var errBuf bytes.Buffer

	ctx := &builtins.ShellContext{
		Stdout:  &outBuf,
		Stderr:  &errBuf,
		Aliases: make(map[string]string),
	}

	exec := NewExecutor(ctx)

	code := exec.RunString("true && echo yes || echo no")
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	if !strings.Contains(outBuf.String(), "yes") || strings.Contains(outBuf.String(), "no") {
		t.Fatalf("expected only 'yes', got '%s'", outBuf.String())
	}
}
