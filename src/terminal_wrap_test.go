package main

import (
	"strings"
	"testing"
)

func TestLayoutPromptWrapAndCursor(t *testing.T) {
	text := []rune(strings.Repeat("d", 70) + "aaaa")
	rendered, row, col := layoutPrompt(text, len(text), 24)
	if row < 2 {
		t.Fatalf("expected multi-row prompt, got %d: %q", row, rendered)
	}
	if !strings.Contains(rendered, "\r\n") {
		t.Fatalf("expected explicit wrapping: %q", rendered)
	}
	_, earlyRow, earlyCol := layoutPrompt(text, 1, 24)
	if earlyRow != 0 || earlyCol != 3 {
		t.Fatalf("unexpected early cursor %d:%d", earlyRow, earlyCol)
	}
	if col >= 24 {
		t.Fatalf("invalid end column %d", col)
	}
}
func TestLayoutPromptUnicodeAndPaste(t *testing.T) {
	text := []rune("äöü日本\nabcdef")
	rendered, row, col := layoutPrompt(text, len(text), 16)
	if !strings.Contains(rendered, "\r\n") || row < 1 || col != 6 {
		t.Fatalf("unexpected layout %d:%d: %q", row, col, rendered)
	}
}
