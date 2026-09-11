package terminal

import (
	"bytes"
	"testing"
)

func TestReadKeyStandard(t *testing.T) {
	tests := []struct {
		input    []byte
		expected KeyType
		expRune  rune
	}{
		{[]byte{13}, KeyEnter, 0},
		{[]byte{10}, KeyEnter, 0},
		{[]byte{9}, KeyTab, 0},
		{[]byte{127}, KeyBackspace, 0},
		{[]byte{3}, KeyCtrlC, 0},
		{[]byte{4}, KeyCtrlD, 0},
		{[]byte{'a'}, KeyRune, 'a'},
		{[]byte{'Z'}, KeyRune, 'Z'},
		{[]byte{27, '[', 'A'}, KeyArrowUp, 0},
		{[]byte{27, '[', 'B'}, KeyArrowDown, 0},
		{[]byte{27, '[', 'C'}, KeyArrowRight, 0},
		{[]byte{27, '[', 'D'}, KeyArrowLeft, 0},
		{[]byte{27, '[', 'H'}, KeyHome, 0},
		{[]byte{27, '[', 'F'}, KeyEnd, 0},
	}

	for _, tc := range tests {
		buf := bytes.NewReader(tc.input)
		ev, err := ReadKey(buf)
		if err != nil {
			t.Fatalf("unexpected error for input %v: %v", tc.input, err)
		}
		if ev.Type != tc.expected {
			t.Errorf("input %v: expected type %v, got %v", tc.input, tc.expected, ev.Type)
		}
		if tc.expRune != 0 && ev.Rune != tc.expRune {
			t.Errorf("input %v: expected rune %c, got %c", tc.input, tc.expRune, ev.Rune)
		}
	}
}

func TestReadBracketedPaste(t *testing.T) {
	pastedCmd := "chsh -s \"/home/MTECH/.local/bin/bush\""
	seq := "\033[200~" + pastedCmd + "\033[201~"

	buf := bytes.NewReader([]byte(seq))
	ev, err := ReadKey(buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Type != KeyPaste {
		t.Fatalf("expected KeyPaste, got %v", ev.Type)
	}
	if ev.Text != pastedCmd {
		t.Fatalf("expected text %q, got %q", pastedCmd, ev.Text)
	}
}

func TestReadUTF8Rune(t *testing.T) {
	buf := bytes.NewReader([]byte("❯"))
	ev, err := ReadKey(buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Type != KeyRune || ev.Rune != '❯' {
		t.Fatalf("expected KeyRune '❯', got %v rune %c", ev.Type, ev.Rune)
	}
}
