package main

import (
	"math"
	"strings"
	"sync"
)

// screenControl is the keyboard and mouse of the share that is open now.
// Fractions run from 0 at the left or top of the picture to 1 at the opposite edge.
type screenControl struct {
	mu     sync.Mutex
	w, h   int
	click  func(x, y float64, button string) error
	scroll func(x, y float64, dy int) error
	key    func(name string) error
	text   func(s string) error
}

func (c *screenControl) setSize(w, h int) {
	if c == nil || w < 1 || h < 1 {
		return
	}
	c.mu.Lock()
	c.w, c.h = w, h
	c.mu.Unlock()
}

func (c *screenControl) size() (int, int) {
	if c == nil {
		return 0, 0
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.w, c.h
}

var screenDesk struct {
	mu   sync.Mutex
	ctrl *screenControl
}

func attachScreenControl(c *screenControl) {
	screenDesk.mu.Lock()
	screenDesk.ctrl = c
	screenDesk.mu.Unlock()
}

func detachScreenControl(c *screenControl) {
	screenDesk.mu.Lock()
	if screenDesk.ctrl == c {
		screenDesk.ctrl = nil
	}
	screenDesk.mu.Unlock()
}

func currentScreenControl() *screenControl {
	screenDesk.mu.Lock()
	defer screenDesk.mu.Unlock()
	return screenDesk.ctrl
}

func screenInputShared() bool {
	return currentScreenControl() != nil
}

func screenShareCaption() string {
	text := "This is the live screen share. The current picture is included in this message and replaces any earlier frame."
	if screenInputShared() {
		text += " Keyboard and mouse are shared. screen_click, screen_scroll, screen_key, and screen_type act on this picture. x and y are fractions of the picture, from 0 at the left or top to 1 at the right or bottom."
	}
	return text
}

func screenToolDefs() []map[string]any {
	num := func(desc string) map[string]any { return map[string]any{"type": "number", "description": desc} }
	str := func(desc string) map[string]any { return map[string]any{"type": "string", "description": desc} }
	frac := "Fraction of the shared picture, from 0 at the left or top to 1 at the right or bottom. Not a pixel."
	return []map[string]any{
		toolDef("screen_click", "Click in the shared picture. Keyboard and mouse must be shared. x and y are fractions of that picture, from 0 to 1.", map[string]any{
			"x":      num(frac),
			"y":      num(frac),
			"button": str("left, right, or middle"),
		}, "x", "y", "button"),
		toolDef("screen_scroll", "Scroll in the shared picture. Positive dy moves the page down. x and y are fractions from 0 to 1.", map[string]any{
			"x":  num(frac),
			"y":  num(frac),
			"dy": map[string]any{"type": "integer", "description": "Wheel steps. Positive moves the page down. Negative moves it up."},
		}, "x", "y", "dy"),
		toolDef("screen_key", "Press one key in the shared picture. Use enter, escape, tab, backspace, delete, left, right, up, down, home, end, pageup, pagedown, or space.", map[string]any{
			"key": str("Key name"),
		}, "key"),
		toolDef("screen_type", "Type text into the shared picture, as if the keys were pressed. Use screen_key for Enter and the arrow keys.", map[string]any{
			"text": str("Characters to type"),
		}, "text"),
	}
}

func screenTool(name string) bool {
	switch name {
	case "screen_click", "screen_scroll", "screen_key", "screen_type":
		return true
	default:
		return false
	}
}

var screenAct sync.Mutex

func runScreenTool(name, args string) string {
	screenAct.Lock()
	defer screenAct.Unlock()
	ctrl := currentScreenControl()
	if ctrl == nil {
		return "Keyboard and mouse are not shared. Turn them on while a screen is shared."
	}
	switch name {
	case "screen_click":
		var a struct {
			X      float64 `json:"x"`
			Y      float64 `json:"y"`
			Button string  `json:"button"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return err.Error()
		}
		button := strings.ToLower(strings.TrimSpace(a.Button))
		if button == "" {
			button = "left"
		}
		if button != "left" && button != "right" && button != "middle" {
			return "button must be left, right, or middle"
		}
		if ctrl.click == nil {
			return "Keyboard and mouse are not shared."
		}
		if err := ctrl.click(clampFrac(a.X), clampFrac(a.Y), button); err != nil {
			return "screen click: " + err.Error()
		}
		return "Clicked."
	case "screen_scroll":
		var a struct {
			X  float64 `json:"x"`
			Y  float64 `json:"y"`
			Dy int     `json:"dy"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return err.Error()
		}
		if a.Dy > 10 {
			a.Dy = 10
		}
		if a.Dy < -10 {
			a.Dy = -10
		}
		if ctrl.scroll == nil {
			return "Keyboard and mouse are not shared."
		}
		if err := ctrl.scroll(clampFrac(a.X), clampFrac(a.Y), a.Dy); err != nil {
			return "screen scroll: " + err.Error()
		}
		return "Scrolled."
	case "screen_key":
		var a struct {
			Key string `json:"key"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return err.Error()
		}
		key := canonicalScreenKey(a.Key)
		if key == "" {
			return "unknown key"
		}
		if ctrl.key == nil {
			return "Keyboard and mouse are not shared."
		}
		if err := ctrl.key(key); err != nil {
			return "screen key: " + err.Error()
		}
		return "Pressed " + key + "."
	case "screen_type":
		var a struct {
			Text string `json:"text"`
		}
		if err := decodeArgs(args, &a); err != nil {
			return err.Error()
		}
		if strings.TrimSpace(a.Text) == "" {
			return "empty text"
		}
		if len([]rune(a.Text)) > 2000 {
			return "text is too long"
		}
		if ctrl.text == nil {
			return "Keyboard and mouse are not shared."
		}
		if err := ctrl.text(a.Text); err != nil {
			return "screen type: " + err.Error()
		}
		return "Typed."
	default:
		return "unknown tool"
	}
}

func clampFrac(v float64) float64 {
	if math.IsNaN(v) || v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func screenPixel(frac float64, size int) int {
	if size <= 1 {
		return 0
	}
	frac = clampFrac(frac)
	return int(math.Round(frac * float64(size-1)))
}

type screenBounds struct {
	X, Y, W, H int
}

func pointInBounds(fx, fy float64, b screenBounds) (int, int) {
	if b.W < 1 {
		b.W = 1
	}
	if b.H < 1 {
		b.H = 1
	}
	return b.X + screenPixel(fx, b.W), b.Y + screenPixel(fy, b.H)
}

// windowsAbsolute maps a point on the virtual desktop into the 0..65535 range SendInput expects.
func windowsAbsolute(x, y, originX, originY, spanW, spanH int) (int32, int32) {
	ax, ay := 0, 0
	if spanW > 1 {
		ax = (x - originX) * 65535 / (spanW - 1)
	}
	if spanH > 1 {
		ay = (y - originY) * 65535 / (spanH - 1)
	}
	if ax < 0 {
		ax = 0
	}
	if ay < 0 {
		ay = 0
	}
	if ax > 65535 {
		ax = 65535
	}
	if ay > 65535 {
		ay = 65535
	}
	return int32(ax), int32(ay)
}

func canonicalScreenKey(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "_", "")
	s = strings.ReplaceAll(s, "-", "")
	switch s {
	case "return", "enter":
		return "enter"
	case "esc", "escape":
		return "escape"
	case "tab":
		return "tab"
	case "backspace", "back":
		return "backspace"
	case "delete", "del":
		return "delete"
	case "left", "arrowleft":
		return "left"
	case "right", "arrowright":
		return "right"
	case "up", "arrowup":
		return "up"
	case "down", "arrowdown":
		return "down"
	case "home":
		return "home"
	case "end":
		return "end"
	case "pageup", "pgup":
		return "pageup"
	case "pagedown", "pgdn":
		return "pagedown"
	case "space":
		return "space"
	default:
		return ""
	}
}

// jpegSize reads the frame size from a JPEG SOF marker.
func jpegSize(b []byte) (int, int, bool) {
	if len(b) < 4 || b[0] != 0xff || b[1] != 0xd8 {
		return 0, 0, false
	}
	i := 2
	for i+8 < len(b) {
		if b[i] != 0xff {
			i++
			continue
		}
		for i < len(b) && b[i] == 0xff {
			i++
		}
		if i >= len(b) {
			break
		}
		marker := b[i]
		i++
		if marker == 0xd8 || marker == 0x01 || (marker >= 0xd0 && marker <= 0xd9) {
			continue
		}
		if i+1 >= len(b) {
			break
		}
		if marker >= 0xc0 && marker <= 0xc3 && i+6 < len(b) {
			h := int(b[i+3])<<8 | int(b[i+4])
			w := int(b[i+5])<<8 | int(b[i+6])
			if w > 0 && h > 0 {
				return w, h, true
			}
		}
		n := int(b[i])<<8 | int(b[i+1])
		if n < 2 || i+n > len(b) {
			break
		}
		i += n
	}
	return 0, 0, false
}

// keysymForRune maps one typed character to an XKB keysym.
func keysymForRune(r rune) (int32, bool) {
	switch r {
	case '\n', '\r':
		return 0xff0d, true
	case '\t':
		return 0xff09, true
	}
	if r >= 0x20 && r <= 0x7e {
		return int32(r), true
	}
	if r > 0x7e && r <= 0x10ffff {
		return 0x01000000 + int32(r), true
	}
	return 0, false
}

func linuxKeysym(name string) (int32, bool) {
	switch name {
	case "enter":
		return 0xff0d, true
	case "escape":
		return 0xff1b, true
	case "tab":
		return 0xff09, true
	case "backspace":
		return 0xff08, true
	case "delete":
		return 0xffff, true
	case "left":
		return 0xff51, true
	case "up":
		return 0xff52, true
	case "right":
		return 0xff53, true
	case "down":
		return 0xff54, true
	case "home":
		return 0xff50, true
	case "end":
		return 0xff57, true
	case "pageup":
		return 0xff55, true
	case "pagedown":
		return 0xff56, true
	case "space":
		return 0x20, true
	default:
		return 0, false
	}
}
