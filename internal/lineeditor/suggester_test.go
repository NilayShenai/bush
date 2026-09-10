package lineeditor

import (
	"bush/internal/builtins"
	"testing"
)

func TestSmartSuggesterNoHistory(t *testing.T) {

	h := &History{}
	ctx := &builtins.ShellContext{
		Bookmarks: map[string]string{
			"work": "/home/user/work",
		},
	}

	sug := FindSmartSuggestion("ab", h, ctx)
	if sug != "out" {
		t.Fatalf("expected 'out' for 'ab', got '%s'", sug)
	}

	sug = FindSmartSuggestion("pe", h, ctx)
	if sug != "ek" {
		t.Fatalf("expected 'ek' for 'pe', got '%s'", sug)
	}

	sug = FindSmartSuggestion("ca", h, ctx)
	if sug != "lc" {
		t.Fatalf("expected 'lc' for 'ca', got '%s'", sug)
	}

	sug = FindSmartSuggestion("git st", h, ctx)
	if sug != "atus" {
		t.Fatalf("expected 'atus' for 'git st', got '%s'", sug)
	}

	sug = FindSmartSuggestion("docker p", h, ctx)
	if sug != "s" {
		t.Fatalf("expected 's' for 'docker p', got '%s'", sug)
	}

	sug = FindSmartSuggestion("jump w", h, ctx)
	if sug != "ork" {
		t.Fatalf("expected 'ork' for 'jump w', got '%s'", sug)
	}
}

func TestSmartSuggesterPrioritizeHistory(t *testing.T) {
	h := &History{
		entries: []string{"git checkout feature-branch"},
	}
	ctx := &builtins.ShellContext{}

	sug := FindSmartSuggestion("git ch", h, ctx)
	if sug != "eckout feature-branch" {
		t.Fatalf("expected 'eckout feature-branch', got '%s'", sug)
	}
}
