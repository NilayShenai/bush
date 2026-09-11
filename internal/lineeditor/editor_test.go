package lineeditor

import (
	"strings"
	"testing"
)

func TestSliceEditingOperations(t *testing.T) {

	runes := []rune{}
	cursorPos := 0

	for _, ch := range "hello" {
		newRunes := make([]rune, 0, len(runes)+1)
		newRunes = append(newRunes, runes[:cursorPos]...)
		newRunes = append(newRunes, ch)
		newRunes = append(newRunes, runes[cursorPos:]...)
		runes = newRunes
		cursorPos++
	}

	if string(runes) != "hello" || cursorPos != 5 {
		t.Fatalf("expected 'hello' with cursor 5, got '%s' at %d", string(runes), cursorPos)
	}

	for i := 0; i < 3; i++ {
		if cursorPos > 0 {
			newRunes := make([]rune, 0, len(runes)-1)
			newRunes = append(newRunes, runes[:cursorPos-1]...)
			newRunes = append(newRunes, runes[cursorPos:]...)
			runes = newRunes
			cursorPos--
		}
	}

	if string(runes) != "he" || cursorPos != 2 {
		t.Fatalf("expected 'he' with cursor 2 after 3 backspaces, got '%s' at %d", string(runes), cursorPos)
	}

	cursorPos = 1
	insCh := 'i'
	newRunes := make([]rune, 0, len(runes)+1)
	newRunes = append(newRunes, runes[:cursorPos]...)
	newRunes = append(newRunes, insCh)
	newRunes = append(newRunes, runes[cursorPos:]...)
	runes = newRunes
	cursorPos++

	if string(runes) != "hie" || cursorPos != 2 {
		t.Fatalf("expected 'hie' with cursor 2, got '%s' at %d", string(runes), cursorPos)
	}

	cursorPos = 1
	if cursorPos < len(runes) {
		delRunes := make([]rune, 0, len(runes)-1)
		delRunes = append(delRunes, runes[:cursorPos]...)
		delRunes = append(delRunes, runes[cursorPos+1:]...)
		runes = delRunes
	}

	if string(runes) != "he" || cursorPos != 1 {
		t.Fatalf("expected 'he' with cursor 1 after delete, got '%s' at %d", string(runes), cursorPos)
	}

	cursorPos = len(runes)
	for i := 0; i < 10; i++ {
		if cursorPos > 0 {
			emptyRunes := make([]rune, 0, len(runes)-1)
			emptyRunes = append(emptyRunes, runes[:cursorPos-1]...)
			emptyRunes = append(emptyRunes, runes[cursorPos:]...)
			runes = emptyRunes
			cursorPos--
		}
	}

	if string(runes) != "" || cursorPos != 0 {
		t.Fatalf("expected '' with cursor 0, got '%s' at %d", string(runes), cursorPos)
	}
}

func TestPasteSanitization(t *testing.T) {
	pasteText := "     chsh -s \"/home/MTECH/.local/bin/bush\"\n"
	cursorPos := 0
	runes := []rune{}

	if cursorPos == 0 && len(runes) == 0 {
		pasteText = strings.TrimLeft(pasteText, " \t")
	}
	pasteText = strings.TrimRight(pasteText, "\r\n")
	pasteText = strings.ReplaceAll(pasteText, "\r\n", " ")
	pasteText = strings.ReplaceAll(pasteText, "\n", " ")
	pasteText = strings.ReplaceAll(pasteText, "\r", " ")

	expected := "chsh -s \"/home/MTECH/.local/bin/bush\""
	if pasteText != expected {
		t.Fatalf("expected %q, got %q", expected, pasteText)
	}

	multi := "echo step 1\necho step 2\r\n"
	multi = strings.TrimRight(multi, "\r\n")
	multi = strings.ReplaceAll(multi, "\r\n", " ")
	multi = strings.ReplaceAll(multi, "\n", " ")
	if multi != "echo step 1 echo step 2" {
		t.Fatalf("expected flattened multiline, got %q", multi)
	}
}
