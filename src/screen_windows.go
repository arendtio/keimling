//go:build windows

package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"image"
	"image/jpeg"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	winOnce    sync.Once
	user32     = windows.NewLazySystemDLL("user32.dll")
	gdi32      = windows.NewLazySystemDLL("gdi32.dll")
	dwmapi     = windows.NewLazySystemDLL("dwmapi.dll")
	procGetDC  = user32.NewProc("GetDC")
	procRelDC  = user32.NewProc("ReleaseDC")
	procGetWin = user32.NewProc("GetWindowRect")
	procPrint  = user32.NewProc("PrintWindow")
	procText   = user32.NewProc("GetWindowTextW")
	procVis    = user32.NewProc("IsWindowVisible")
	procStyle  = user32.NewProc("GetWindowLongPtrW")
	procEnumW  = user32.NewProc("EnumWindows")
	procEnumM  = user32.NewProc("EnumDisplayMonitors")
	procMetric = user32.NewProc("GetSystemMetrics")
	procSend   = user32.NewProc("SendInput")
	procDPI    = user32.NewProc("SetProcessDpiAwarenessContext")
	procAware  = user32.NewProc("SetProcessDPIAware")
	procDC     = gdi32.NewProc("CreateCompatibleDC")
	procBitmap = gdi32.NewProc("CreateCompatibleBitmap")
	procSelect = gdi32.NewProc("SelectObject")
	procDelete = gdi32.NewProc("DeleteObject")
	procDelDC  = gdi32.NewProc("DeleteDC")
	procBits   = gdi32.NewProc("GetDIBits")
	procBlt    = gdi32.NewProc("BitBlt")
	procCloak  = dwmapi.NewProc("DwmGetWindowAttribute")
)

func winInit() {
	winOnce.Do(func() {
		// PER_MONITOR_AWARE_V2 is -4. Clicks then use the same pixels as the picture.
		r, _, _ := procDPI.Call(^uintptr(3))
		if r == 0 {
			procAware.Call()
		}
	})
}

func screenShareChooser() (string, []screenTarget) {
	winInit()
	return "list", listWindowsTargets()
}

func openScreenCast(ctx context.Context, input bool, target string) (*screenCast, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	winInit()
	kind, id, bounds, err := resolveWindowsTarget(target)
	if err != nil {
		return nil, err
	}
	var boxMu sync.Mutex
	box := bounds
	current := func() screenBounds {
		boxMu.Lock()
		defer boxMu.Unlock()
		if kind == "window" {
			if next, ok := windowRect(id); ok {
				box = next
			}
		}
		return box
	}
	ctrl := (*screenControl)(nil)
	if input {
		ctrl = &screenControl{}
		ctrl.click = func(x, y float64, button string) error {
			px, py := pointInBounds(x, y, current())
			return windowsClick(px, py, button)
		}
		ctrl.scroll = func(x, y float64, dy int) error {
			px, py := pointInBounds(x, y, current())
			return windowsScroll(px, py, dy)
		}
		ctrl.key = func(name string) error { return windowsKey(name) }
		ctrl.text = func(s string) error { return windowsType(s) }
	}
	cast := &screenCast{fd: -1, control: ctrl}
	cast.run = func(ctx context.Context, write func([]byte) error) error {
		send := func() error {
			jpegBytes, err := captureWindows(kind, id, current())
			if err != nil {
				return err
			}
			return write(jpegBytes)
		}
		if err := send(); err != nil {
			return err
		}
		tick := time.NewTicker(200 * time.Millisecond)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-tick.C:
				if err := send(); err != nil {
					return err
				}
			}
		}
	}
	cast.stop = func() {}
	return cast, nil
}

func resolveWindowsTarget(target string) (kind string, id uintptr, bounds screenBounds, err error) {
	target = strings.TrimSpace(target)
	kind, raw, ok := strings.Cut(target, ":")
	if !ok || (kind != "screen" && kind != "window") {
		return "", 0, screenBounds{}, fmt.Errorf("choose a screen or window")
	}
	n, conv := strconv.ParseUint(raw, 10, 64)
	if conv != nil {
		return "", 0, screenBounds{}, fmt.Errorf("choose a screen or window")
	}
	id = uintptr(n)
	bounds = windowsTargetBounds(kind, id, screenBounds{})
	if bounds.W < 1 || bounds.H < 1 {
		return "", 0, screenBounds{}, fmt.Errorf("that screen or window is no longer there")
	}
	return kind, id, bounds, nil
}

func windowRect(id uintptr) (screenBounds, bool) {
	var r winRect
	ok, _, _ := procGetWin.Call(id, uintptr(unsafe.Pointer(&r)))
	if ok == 0 {
		return screenBounds{}, false
	}
	b := screenBounds{X: int(r.Left), Y: int(r.Top), W: int(r.Right - r.Left), H: int(r.Bottom - r.Top)}
	if b.W < 1 || b.H < 1 {
		return screenBounds{}, false
	}
	return b, true
}

func windowsTargetBounds(kind string, id uintptr, fallback screenBounds) screenBounds {
	if kind == "window" {
		if b, ok := windowRect(id); ok {
			return b
		}
		return fallback
	}
	var found screenBounds
	cb := windows.NewCallback(func(monitor, _, lprc, _ uintptr) uintptr {
		if monitor != id || lprc == 0 {
			return 1
		}
		rect := *(*winRect)(unsafe.Pointer(lprc))
		found = screenBounds{X: int(rect.Left), Y: int(rect.Top), W: int(rect.Right - rect.Left), H: int(rect.Bottom - rect.Top)}
		return 1
	})
	procEnumM.Call(0, 0, cb, 0)
	if found.W < 1 {
		return fallback
	}
	return found
}

func listWindowsTargets() []screenTarget {
	var out []screenTarget
	var n int
	cb := windows.NewCallback(func(monitor, _, lprc, _ uintptr) uintptr {
		n++
		name := "Screen " + strconv.Itoa(n)
		if lprc != 0 {
			rect := *(*winRect)(unsafe.Pointer(lprc))
			name = fmt.Sprintf("Screen %d (%d×%d)", n, rect.Right-rect.Left, rect.Bottom-rect.Top)
		}
		out = append(out, screenTarget{ID: "screen:" + strconv.FormatUint(uint64(monitor), 10), Kind: "screen", Name: name})
		return 1
	})
	procEnumM.Call(0, 0, cb, 0)
	var wins []screenTarget
	wcb := windows.NewCallback(func(hwnd, _ uintptr) uintptr {
		vis, _, _ := procVis.Call(hwnd)
		if vis == 0 {
			return 1
		}
		ex, _, _ := procStyle.Call(hwnd, ^uintptr(19)) // GWL_EXSTYLE = -20
		if ex&0x80 != 0 {                              // WS_EX_TOOLWINDOW
			return 1
		}
		var cloaked uint32
		if r, _, _ := procCloak.Call(hwnd, 14, uintptr(unsafe.Pointer(&cloaked)), 4); r == 0 && cloaked != 0 {
			return 1
		}
		buf := make([]uint16, 256)
		tn, _, _ := procText.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
		if tn == 0 {
			return 1
		}
		title := windows.UTF16ToString(buf[:tn])
		if strings.TrimSpace(title) == "" {
			return 1
		}
		wins = append(wins, screenTarget{ID: "window:" + strconv.FormatUint(uint64(hwnd), 10), Kind: "window", Name: title})
		return 1
	})
	procEnumW.Call(wcb, 0)
	if len(wins) > 40 {
		wins = wins[:40]
	}
	return append(out, wins...)
}

type winRect struct {
	Left, Top, Right, Bottom int32
}

func captureWindows(kind string, id uintptr, b screenBounds) ([]byte, error) {
	if b.W < 1 || b.H < 1 || b.W > 16000 || b.H > 16000 {
		return nil, fmt.Errorf("screen share produced no picture")
	}
	hdc, _, _ := procGetDC.Call(0)
	if hdc == 0 {
		return nil, fmt.Errorf("screen share produced no picture")
	}
	defer procRelDC.Call(0, hdc)
	mem, _, _ := procDC.Call(hdc)
	if mem == 0 {
		return nil, fmt.Errorf("screen share produced no picture")
	}
	defer procDelDC.Call(mem)
	bmp, _, _ := procBitmap.Call(hdc, uintptr(b.W), uintptr(b.H))
	if bmp == 0 {
		return nil, fmt.Errorf("screen share produced no picture")
	}
	defer procDelete.Call(bmp)
	procSelect.Call(mem, bmp)
	if kind == "window" {
		procPrint.Call(id, mem, 2)
	} else {
		procBlt.Call(mem, 0, 0, uintptr(b.W), uintptr(b.H), hdc, uintptr(b.X), uintptr(b.Y), 0x00CC0020)
	}
	pix := make([]byte, b.W*b.H*4)
	hdr := make([]byte, 40)
	binary.LittleEndian.PutUint32(hdr[0:4], 40)
	binary.LittleEndian.PutUint32(hdr[4:8], uint32(int32(b.W)))
	binary.LittleEndian.PutUint32(hdr[8:12], uint32(-int32(b.H)))
	binary.LittleEndian.PutUint16(hdr[12:14], 1)
	binary.LittleEndian.PutUint16(hdr[14:16], 32)
	r, _, _ := procBits.Call(mem, bmp, 0, uintptr(b.H), uintptr(unsafe.Pointer(&pix[0])), uintptr(unsafe.Pointer(&hdr[0])), 0)
	if r == 0 {
		return nil, fmt.Errorf("screen share produced no picture")
	}
	img := image.NewRGBA(image.Rect(0, 0, b.W, b.H))
	for i := 0; i < len(pix); i += 4 {
		img.Pix[i] = pix[i+2]
		img.Pix[i+1] = pix[i+1]
		img.Pix[i+2] = pix[i]
		img.Pix[i+3] = 255
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 70}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func windowsClick(x, y int, button string) error {
	if err := windowsMove(x, y); err != nil {
		return err
	}
	down, up := uint32(0x0002), uint32(0x0004)
	switch button {
	case "right":
		down, up = 0x0008, 0x0010
	case "middle":
		down, up = 0x0020, 0x0040
	}
	if err := windowsMouse(down, 0, 0, 0); err != nil {
		return err
	}
	return windowsMouse(up, 0, 0, 0)
}

func windowsScroll(x, y, dy int) error {
	if err := windowsMove(x, y); err != nil {
		return err
	}
	return windowsMouse(0x0800, 0, 0, uint32(int32(-dy*120)))
}

func windowsMove(x, y int) error {
	vx, _, _ := procMetric.Call(76) // SM_XVIRTUALSCREEN
	vy, _, _ := procMetric.Call(77)
	vw, _, _ := procMetric.Call(78)
	vh, _, _ := procMetric.Call(79)
	ax, ay := windowsAbsolute(x, y, int(vx), int(vy), int(vw), int(vh))
	return windowsMouse(0x8001|0x4000, ax, ay, 0) // MOVE|ABSOLUTE|VIRTUALDESK
}

func windowsMouse(flags uint32, x, y int32, data uint32) error {
	var b [40]byte
	binary.LittleEndian.PutUint32(b[8:12], uint32(x))
	binary.LittleEndian.PutUint32(b[12:16], uint32(y))
	binary.LittleEndian.PutUint32(b[16:20], data)
	binary.LittleEndian.PutUint32(b[20:24], flags)
	r, _, err := procSend.Call(1, uintptr(unsafe.Pointer(&b[0])), 40)
	if r == 0 {
		return fmt.Errorf("pointer input failed: %v", err)
	}
	return nil
}

func windowsKey(name string) error {
	vk, ok := windowsVK(name)
	if !ok {
		return fmt.Errorf("unknown key")
	}
	if err := windowsKeyTap(vk, 0); err != nil {
		return err
	}
	return nil
}

func windowsVK(name string) (uint16, bool) {
	switch name {
	case "enter":
		return 0x0D, true
	case "escape":
		return 0x1B, true
	case "tab":
		return 0x09, true
	case "backspace":
		return 0x08, true
	case "delete":
		return 0x2E, true
	case "left":
		return 0x25, true
	case "up":
		return 0x26, true
	case "right":
		return 0x27, true
	case "down":
		return 0x28, true
	case "home":
		return 0x24, true
	case "end":
		return 0x23, true
	case "pageup":
		return 0x21, true
	case "pagedown":
		return 0x22, true
	case "space":
		return 0x20, true
	default:
		return 0, false
	}
}

func windowsType(s string) error {
	for _, r := range s {
		if r == '\n' || r == '\r' {
			if err := windowsKey("enter"); err != nil {
				return err
			}
			continue
		}
		if r > 0xFFFF {
			continue
		}
		if err := windowsUnicode(uint16(r)); err != nil {
			return err
		}
	}
	return nil
}

func windowsKeyTap(vk uint16, flags uint32) error {
	if err := windowsKeyEvent(vk, 0, flags); err != nil {
		return err
	}
	return windowsKeyEvent(vk, 0, flags|0x0002)
}

func windowsUnicode(ch uint16) error {
	if err := windowsKeyEvent(0, ch, 0x0004); err != nil {
		return err
	}
	return windowsKeyEvent(0, ch, 0x0004|0x0002)
}

func windowsKeyEvent(vk, scan uint16, flags uint32) error {
	var b [40]byte
	binary.LittleEndian.PutUint32(b[0:4], 1)
	binary.LittleEndian.PutUint16(b[8:10], vk)
	binary.LittleEndian.PutUint16(b[10:12], scan)
	binary.LittleEndian.PutUint32(b[12:16], flags)
	r, _, err := procSend.Call(1, uintptr(unsafe.Pointer(&b[0])), 40)
	if r == 0 {
		return fmt.Errorf("keyboard input failed: %v", err)
	}
	return nil
}
