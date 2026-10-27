package main

import (
	"strings"
	"testing"
)

func TestMarkdownLinks(t *testing.T) {
	got := formatTerminal("[Arch wiki](https://wiki.archlinux.org/)")
	if got != "Arch wiki (https://wiki.archlinux.org/)" {
		t.Fatal(got)
	}
}

func TestMarkdownInline(t *testing.T) {
	got := renderMarkdown("use **bold** and *italic* with `code`")
	if !strings.Contains(got, "\033[1mbold\033[0m") {
		t.Fatalf("bold: %q", got)
	}
	if !strings.Contains(got, "\033[3mitalic\033[0m") {
		t.Fatalf("italic: %q", got)
	}
	if !strings.Contains(got, "\033[36mcode\033[0m") {
		t.Fatalf("code: %q", got)
	}
}

func TestMarkdownHeadingListFence(t *testing.T) {
	in := "# Title\n\n- one\n- two\n\n```go\nfmt.Println(1)\n```\n"
	got := renderMarkdown(in)
	if !strings.Contains(got, "\033[1mTitle\033[0m") {
		t.Fatalf("heading: %q", got)
	}
	if !strings.Contains(got, "\033[2m•\033[0m one") {
		t.Fatalf("list: %q", got)
	}
	if !strings.Contains(got, "\033[2mgo\033[0m") {
		t.Fatalf("lang: %q", got)
	}
	if !strings.Contains(got, "\033[36mfmt.Println(1)\033[0m") {
		t.Fatalf("fence: %q", got)
	}
}

func TestMarkdownOrderedList(t *testing.T) {
	got := renderMarkdown("1. first\n2. second")
	if !strings.Contains(got, "\033[2m•\033[0m first") || !strings.Contains(got, "second") {
		t.Fatal(got)
	}
}

func TestMarkdownPassthrough(t *testing.T) {
	in := "plain text without markup"
	if got := renderMarkdown(in); got != in {
		t.Fatalf("%q", got)
	}
}
