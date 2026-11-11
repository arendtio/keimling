package main

import (
	"strings"
	"testing"
)

func TestScreenToolDoesNothingUntilShared(t *testing.T) {
	detachScreenControl(currentScreenControl())
	got := runScreenTool("screen_click", `{"x":0.5,"y":0.5,"button":"left"}`)
	if !strings.Contains(got, "not shared") {
		t.Fatalf("click: %s", got)
	}
	if runScreenTool("screen_type", `{"text":"hi"}`) == "Typed." {
		t.Fatal("typed without a share")
	}
}

func TestScreenClickUsesFractions(t *testing.T) {
	var gotX, gotY float64
	var gotButton string
	ctrl := &screenControl{click: func(x, y float64, button string) error {
		gotX, gotY, gotButton = x, y, button
		return nil
	}}
	attachScreenControl(ctrl)
	defer detachScreenControl(ctrl)
	if got := runScreenTool("screen_click", `{"x":1.4,"y":-3,"button":"right"}`); got != "Clicked." {
		t.Fatalf("click: %s", got)
	}
	if gotX != 1 || gotY != 0 || gotButton != "right" {
		t.Fatalf("mapped %v %v %s", gotX, gotY, gotButton)
	}
	if got := runScreenTool("screen_click", `{"x":0.2,"y":0.2,"button":"other"}`); !strings.Contains(got, "button") {
		t.Fatalf("button: %s", got)
	}
}

func TestScreenKeyAndScroll(t *testing.T) {
	var key string
	var dy int
	ctrl := &screenControl{
		key:    func(name string) error { key = name; return nil },
		scroll: func(x, y float64, steps int) error { dy = steps; return nil },
		text:   func(s string) error { key = s; return nil },
	}
	attachScreenControl(ctrl)
	defer detachScreenControl(ctrl)
	if got := runScreenTool("screen_key", `{"key":"Return"}`); got != "Pressed enter." || key != "enter" {
		t.Fatalf("key %q %s", key, got)
	}
	if got := runScreenTool("screen_key", `{"key":"nope"}`); got != "unknown key" {
		t.Fatalf("unknown: %s", got)
	}
	if got := runScreenTool("screen_scroll", `{"x":0,"y":1,"dy":40}`); got != "Scrolled." || dy != 10 {
		t.Fatalf("scroll %d %s", dy, got)
	}
	if got := runScreenTool("screen_type", `{"text":"hi"}`); got != "Typed." || key != "hi" {
		t.Fatalf("type %q %s", key, got)
	}
}

func TestScreenPixelAndWindowsAbsolute(t *testing.T) {
	if screenPixel(0, 100) != 0 || screenPixel(1, 100) != 99 || screenPixel(0.5, 11) != 5 {
		t.Fatalf("pixels %d %d %d", screenPixel(0, 100), screenPixel(1, 100), screenPixel(0.5, 11))
	}
	x, y := pointInBounds(1, 0, screenBounds{X: 10, Y: 20, W: 100, H: 50})
	if x != 109 || y != 20 {
		t.Fatalf("point %d %d", x, y)
	}
	ax, ay := windowsAbsolute(10, 20, 10, 20, 101, 51)
	if ax != 0 || ay != 0 {
		t.Fatalf("origin %d %d", ax, ay)
	}
	ax, ay = windowsAbsolute(110, 70, 10, 20, 101, 51)
	if ax != 65535 || ay != 65535 {
		t.Fatalf("far %d %d", ax, ay)
	}
}

func TestJPEGSize(t *testing.T) {
	b := []byte{0xff, 0xd8, 0xff, 0xc0, 0x00, 0x11, 0x08, 0x00, 0x02, 0x00, 0x03, 0x00}
	w, h, ok := jpegSize(b)
	if !ok || w != 3 || h != 2 {
		t.Fatalf("size %d %d %v", w, h, ok)
	}
	if _, _, ok := jpegSize([]byte{1, 2, 3}); ok {
		t.Fatal("accepted a non-jpeg")
	}
}

func TestScreenShareCaptionMentionsInputOnlyWhenShared(t *testing.T) {
	detachScreenControl(currentScreenControl())
	if strings.Contains(screenShareCaption(), "Keyboard and mouse") {
		t.Fatal("caption offers input while it is off")
	}
	ctrl := &screenControl{}
	attachScreenControl(ctrl)
	defer detachScreenControl(ctrl)
	if !strings.Contains(screenShareCaption(), "fractions") {
		t.Fatal(screenShareCaption())
	}
}

func TestGateScreenInputDoesNotAsk(t *testing.T) {
	g := gateToolCall(false, "ask", "screen_click", `{"x":0.1,"y":0.2,"button":"left"}`, false)
	if !g.Allow || g.Ask {
		t.Fatalf("gate %+v", g)
	}
}

func TestKeysymForRune(t *testing.T) {
	if s, ok := keysymForRune('A'); !ok || s != 'A' {
		t.Fatalf("A %d %v", s, ok)
	}
	if s, ok := keysymForRune('\n'); !ok || s != 0xff0d {
		t.Fatalf("enter %x", s)
	}
	if s, ok := keysymForRune('ä'); !ok || s != 0x01000000+int32('ä') {
		t.Fatalf("umlaut %x", s)
	}
}
