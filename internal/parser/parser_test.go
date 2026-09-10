package parser

import (
	"bush/internal/ast"
	"testing"
)

func TestParseSimpleCommand(t *testing.T) {
	cmdList, err := Parse("echo hello world")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cmdList.Pipelines) != 1 {
		t.Fatalf("expected 1 pipeline, got %d", len(cmdList.Pipelines))
	}
	pipe := cmdList.Pipelines[0].Pipeline
	if len(pipe.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(pipe.Commands))
	}
	cmd := pipe.Commands[0]
	if len(cmd.Args) != 3 || cmd.Args[0] != "echo" || cmd.Args[1] != "hello" || cmd.Args[2] != "world" {
		t.Fatalf("unexpected args: %v", cmd.Args)
	}
}

func TestParsePipelineAndRedirection(t *testing.T) {
	cmdList, err := Parse("cat < input.txt | grep foo > output.txt 2>&1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cmdList.Pipelines) != 1 {
		t.Fatalf("expected 1 pipeline, got %d", len(cmdList.Pipelines))
	}
	pipe := cmdList.Pipelines[0].Pipeline
	if len(pipe.Commands) != 2 {
		t.Fatalf("expected 2 commands in pipeline, got %d", len(pipe.Commands))
	}

	cmd1 := pipe.Commands[0]
	if len(cmd1.Redirects) != 1 || cmd1.Redirects[0].Type != ast.RedirIn || cmd1.Redirects[0].Target != "input.txt" {
		t.Fatalf("unexpected cmd1 redirects: %v", cmd1.Redirects)
	}

	cmd2 := pipe.Commands[1]
	if len(cmd2.Redirects) != 2 {
		t.Fatalf("expected 2 redirects on cmd2, got %d", len(cmd2.Redirects))
	}
}

func TestParseChainedCommands(t *testing.T) {
	cmdList, err := Parse("make && ./run || echo fail ; exit")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cmdList.Pipelines) != 4 {
		t.Fatalf("expected 4 pipelines, got %d", len(cmdList.Pipelines))
	}
	if cmdList.Pipelines[0].Operator != ast.OpAnd {
		t.Errorf("expected OpAnd for first pipeline, got %v", cmdList.Pipelines[0].Operator)
	}
	if cmdList.Pipelines[1].Operator != ast.OpOr {
		t.Errorf("expected OpOr for second pipeline, got %v", cmdList.Pipelines[1].Operator)
	}
	if cmdList.Pipelines[2].Operator != ast.OpSemi {
		t.Errorf("expected OpSemi for third pipeline, got %v", cmdList.Pipelines[2].Operator)
	}
}
