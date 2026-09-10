package expander

import (
	"os"
	"testing"
)

func TestExpandVariablesAndTilde(t *testing.T) {
	os.Setenv("TEST_VAR", "goosh_rocks")
	home, _ := os.UserHomeDir()

	ctx := &Context{
		LastExitCode: 42,
	}

	res := ExpandToken("$TEST_VAR", ctx)
	if len(res) != 1 || res[0] != "goosh_rocks" {
		t.Fatalf("expected 'goosh_rocks', got %v", res)
	}

	res = ExpandToken("$?", ctx)
	if len(res) != 1 || res[0] != "42" {
		t.Fatalf("expected '42', got %v", res)
	}

	res = ExpandToken("~/test", ctx)
	expected := home + "/test"
	if len(res) != 1 || res[0] != expected {
		t.Fatalf("expected '%s', got %v", expected, res)
	}
}

func TestExpandSubshell(t *testing.T) {
	ctx := &Context{
		SubshellRunner: func(cmd string) (string, error) {
			if cmd == "echo hi" {
				return "hi\n", nil
			}
			return "", nil
		},
	}

	res := ExpandToken("Result: $(echo hi)", ctx)
	if len(res) != 1 || res[0] != "Result: hi" {
		t.Fatalf("expected 'Result: hi', got %v", res)
	}
}
