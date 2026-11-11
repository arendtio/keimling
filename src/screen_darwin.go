//go:build darwin && cgo

package main

/*
#cgo LDFLAGS: -framework CoreGraphics -framework CoreFoundation -framework ApplicationServices -framework ScreenCaptureKit -framework Foundation

#include <CoreGraphics/CoreGraphics.h>
#include <CoreFoundation/CoreFoundation.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

extern Boolean AXIsProcessTrusted(void);

static int screenTrusted(void) {
	return AXIsProcessTrusted() ? 1 : 0;
}

typedef struct {
	uint32_t id;
	double x, y, w, h;
	char name[256];
} ScreenItem;

static void copyCFName(char *dst, int n, CFStringRef s) {
	if (n < 1) return;
	dst[0] = 0;
	if (s) CFStringGetCString(s, dst, n, kCFStringEncodingUTF8);
}

static int screenListDisplays(ScreenItem *out, int max) {
	uint32_t ids[16];
	uint32_t count = 0;
	if (CGGetActiveDisplayList(16, ids, &count) != kCGErrorSuccess) return 0;
	int n = 0;
	for (uint32_t i = 0; i < count && n < max; i++) {
		CGRect r = CGDisplayBounds(ids[i]);
		out[n].id = ids[i];
		out[n].x = r.origin.x;
		out[n].y = r.origin.y;
		out[n].w = r.size.width;
		out[n].h = r.size.height;
		snprintf(out[n].name, sizeof out[n].name, "Screen %u", (unsigned)(i + 1));
		n++;
	}
	return n;
}

static int screenListWindows(ScreenItem *out, int max) {
	CFArrayRef list = CGWindowListCopyWindowInfo(kCGWindowListOptionOnScreenOnly | kCGWindowListExcludeDesktopElements, kCGNullWindowID);
	if (!list) return 0;
	CFIndex count = CFArrayGetCount(list);
	int n = 0;
	for (CFIndex i = 0; i < count && n < max; i++) {
		CFDictionaryRef info = (CFDictionaryRef)CFArrayGetValueAtIndex(list, i);
		int layer = 0;
		CFNumberRef layerRef = CFDictionaryGetValue(info, kCGWindowLayer);
		if (layerRef) CFNumberGetValue(layerRef, kCFNumberIntType, &layer);
		if (layer != 0) continue;
		int wid = 0;
		CFNumberRef num = CFDictionaryGetValue(info, kCGWindowNumber);
		if (!num || !CFNumberGetValue(num, kCFNumberIntType, &wid) || wid <= 0) continue;
		CGRect rect = CGRectZero;
		CFDictionaryRef bounds = CFDictionaryGetValue(info, kCGWindowBounds);
		if (!bounds || !CGRectMakeWithDictionaryRepresentation(bounds, &rect)) continue;
		if (rect.size.width < 2 || rect.size.height < 2) continue;
		char owner[160];
		char title[160];
		copyCFName(owner, (int)sizeof owner, CFDictionaryGetValue(info, kCGWindowOwnerName));
		copyCFName(title, (int)sizeof title, CFDictionaryGetValue(info, kCGWindowName));
		if (owner[0] == 0 && title[0] == 0) continue;
		out[n].id = (uint32_t)wid;
		out[n].x = rect.origin.x;
		out[n].y = rect.origin.y;
		out[n].w = rect.size.width;
		out[n].h = rect.size.height;
		if (title[0] && owner[0]) snprintf(out[n].name, sizeof out[n].name, "%s — %s", owner, title);
		else if (title[0]) snprintf(out[n].name, sizeof out[n].name, "%s", title);
		else snprintf(out[n].name, sizeof out[n].name, "%s", owner);
		n++;
	}
	CFRelease(list);
	return n;
}

static int screenBoundsOf(int display, uint32_t id, double *x, double *y, double *w, double *h) {
	if (display) {
		CGRect r = CGDisplayBounds(id);
		*x = r.origin.x; *y = r.origin.y; *w = r.size.width; *h = r.size.height;
		return r.size.width >= 1 && r.size.height >= 1;
	}
	CFArrayRef list = CGWindowListCopyWindowInfo(kCGWindowListOptionIncludingWindow, id);
	if (!list) return 0;
	int ok = 0;
	if (CFArrayGetCount(list) > 0) {
		CFDictionaryRef info = (CFDictionaryRef)CFArrayGetValueAtIndex(list, 0);
		CGRect rect = CGRectZero;
		CFDictionaryRef bounds = CFDictionaryGetValue(info, kCGWindowBounds);
		if (bounds && CGRectMakeWithDictionaryRepresentation(bounds, &rect) && rect.size.width >= 1 && rect.size.height >= 1) {
			*x = rect.origin.x; *y = rect.origin.y; *w = rect.size.width; *h = rect.size.height;
			ok = 1;
		}
	}
	CFRelease(list);
	return ok;
}

CGImageRef screenCaptureImage(int display, uint32_t targetID);

static int screenGrab(int display, uint32_t id, uint8_t **rgba, int *w, int *h) {
	CGImageRef img = screenCaptureImage(display, id);
	if (!img) return 0;
	size_t width = CGImageGetWidth(img);
	size_t height = CGImageGetHeight(img);
	if (width == 0 || height == 0 || width > 16000 || height > 16000) {
		CFRelease(img);
		return 0;
	}
	uint8_t *buf = (uint8_t *)malloc(width * height * 4);
	if (!buf) { CFRelease(img); return 0; }
	CGColorSpaceRef cs = CGColorSpaceCreateDeviceRGB();
	CGContextRef ctx = CGBitmapContextCreate(buf, width, height, 8, width * 4, cs, kCGImageAlphaPremultipliedLast | kCGBitmapByteOrder32Big);
	CGColorSpaceRelease(cs);
	if (!ctx) { free(buf); CFRelease(img); return 0; }
	CGContextDrawImage(ctx, CGRectMake(0, 0, (CGFloat)width, (CGFloat)height), img);
	CGContextRelease(ctx);
	CFRelease(img);
	*rgba = buf;
	*w = (int)width;
	*h = (int)height;
	return 1;
}

static void screenMove(double x, double y) {
	CGEventRef ev = CGEventCreateMouseEvent(NULL, kCGEventMouseMoved, CGPointMake(x, y), kCGMouseButtonLeft);
	if (ev) { CGEventPost(kCGHIDEventTap, ev); CFRelease(ev); }
}

static void screenButton(double x, double y, int button, int down) {
	CGEventType type;
	CGMouseButton btn;
	if (button == 1) { btn = kCGMouseButtonRight; type = down ? kCGEventRightMouseDown : kCGEventRightMouseUp; }
	else if (button == 2) { btn = kCGMouseButtonCenter; type = down ? kCGEventOtherMouseDown : kCGEventOtherMouseUp; }
	else { btn = kCGMouseButtonLeft; type = down ? kCGEventLeftMouseDown : kCGEventLeftMouseUp; }
	CGEventRef ev = CGEventCreateMouseEvent(NULL, type, CGPointMake(x, y), btn);
	if (ev) { CGEventPost(kCGHIDEventTap, ev); CFRelease(ev); }
}

static void screenWheel(int dy) {
	CGEventRef ev = CGEventCreateScrollWheelEvent(NULL, kCGScrollEventUnitLine, 1, (int32_t)(-dy));
	if (ev) { CGEventPost(kCGHIDEventTap, ev); CFRelease(ev); }
}

static void screenKeyCode(uint16_t code, int down) {
	CGEventRef ev = CGEventCreateKeyboardEvent(NULL, (CGKeyCode)code, down ? true : false);
	if (ev) { CGEventPost(kCGHIDEventTap, ev); CFRelease(ev); }
}

static void screenUnicode(uint16_t ch) {
	UniChar u = (UniChar)ch;
	CGEventRef down = CGEventCreateKeyboardEvent(NULL, 0, true);
	CGEventRef up = CGEventCreateKeyboardEvent(NULL, 0, false);
	if (down) { CGEventKeyboardSetUnicodeString(down, 1, &u); CGEventPost(kCGHIDEventTap, down); CFRelease(down); }
	if (up) { CGEventKeyboardSetUnicodeString(up, 1, &u); CGEventPost(kCGHIDEventTap, up); CFRelease(up); }
}
*/
import "C"

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

func screenShareChooser() (string, []screenTarget) {
	return "list", darwinTargets()
}

func openScreenCast(ctx context.Context, input bool, target string) (*screenCast, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	display, id, err := resolveDarwinTarget(target)
	if err != nil {
		return nil, err
	}
	if input && C.screenTrusted() == 0 {
		return nil, fmt.Errorf("allow Keimling under Accessibility in System Settings")
	}
	ctrl := (*screenControl)(nil)
	if input {
		ctrl = &screenControl{}
		ctrl.click = func(x, y float64, button string) error {
			px, py, err := darwinPoint(display, id, x, y)
			if err != nil {
				return err
			}
			return darwinClick(px, py, button)
		}
		ctrl.scroll = func(x, y float64, dy int) error {
			px, py, err := darwinPoint(display, id, x, y)
			if err != nil {
				return err
			}
			C.screenMove(C.double(px), C.double(py))
			C.screenWheel(C.int(dy))
			return nil
		}
		ctrl.key = darwinKey
		ctrl.text = darwinType
	}
	cast := &screenCast{fd: -1, control: ctrl}
	cast.run = func(ctx context.Context, write func([]byte) error) error {
		send := func() error {
			frame, err := darwinGrab(display, id)
			if err != nil {
				return err
			}
			return write(frame)
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

func resolveDarwinTarget(target string) (bool, uint32, error) {
	kind, raw, ok := strings.Cut(strings.TrimSpace(target), ":")
	if !ok || (kind != "screen" && kind != "window") {
		return false, 0, fmt.Errorf("choose a screen or window")
	}
	n, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		return false, 0, fmt.Errorf("choose a screen or window")
	}
	return kind == "screen", uint32(n), nil
}

func darwinTargets() []screenTarget {
	var out []screenTarget
	items := make([]C.ScreenItem, 16)
	n := int(C.screenListDisplays(&items[0], C.int(len(items))))
	for i := 0; i < n && i < len(items); i++ {
		out = append(out, screenTarget{
			ID:   "screen:" + strconv.FormatUint(uint64(items[i].id), 10),
			Kind: "screen",
			Name: fmt.Sprintf("%s (%d×%d)", darwinName(items[i]), int(items[i].w), int(items[i].h)),
		})
	}
	wins := make([]C.ScreenItem, 40)
	n = int(C.screenListWindows(&wins[0], C.int(len(wins))))
	for i := 0; i < n && i < len(wins); i++ {
		out = append(out, screenTarget{
			ID:   "window:" + strconv.FormatUint(uint64(wins[i].id), 10),
			Kind: "window",
			Name: darwinName(wins[i]),
		})
	}
	return out
}

func darwinName(item C.ScreenItem) string {
	return C.GoString((*C.char)(unsafe.Pointer(&item.name[0])))
}

func darwinGrab(display bool, id uint32) ([]byte, error) {
	var rgba *C.uint8_t
	var w, h C.int
	flag := C.int(0)
	if display {
		flag = 1
	}
	if C.screenGrab(flag, C.uint32_t(id), &rgba, &w, &h) == 0 || rgba == nil {
		return nil, fmt.Errorf("allow Keimling under Screen Recording in System Settings, then share again")
	}
	defer C.free(unsafe.Pointer(rgba))
	width, height := int(w), int(h)
	if width < 1 || height < 1 {
		return nil, fmt.Errorf("screen share produced no picture")
	}
	raw := C.GoBytes(unsafe.Pointer(rgba), C.int(width*height*4))
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	copy(img.Pix, raw)
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 70}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func darwinPoint(display bool, id uint32, fx, fy float64) (float64, float64, error) {
	var x, y, w, h C.double
	flag := C.int(0)
	if display {
		flag = 1
	}
	if C.screenBoundsOf(flag, C.uint32_t(id), &x, &y, &w, &h) == 0 || w < 1 || h < 1 {
		return 0, 0, fmt.Errorf("that screen or window is no longer there")
	}
	return float64(x) + clampFrac(fx)*float64(w), float64(y) + clampFrac(fy)*float64(h), nil
}

func darwinClick(x, y float64, button string) error {
	code := C.int(0)
	switch button {
	case "right":
		code = 1
	case "middle":
		code = 2
	}
	C.screenMove(C.double(x), C.double(y))
	C.screenButton(C.double(x), C.double(y), code, 1)
	C.screenButton(C.double(x), C.double(y), code, 0)
	return nil
}

func darwinKey(name string) error {
	code, ok := darwinKeyCode(name)
	if !ok {
		return fmt.Errorf("unknown key")
	}
	C.screenKeyCode(C.uint16_t(code), 1)
	C.screenKeyCode(C.uint16_t(code), 0)
	return nil
}

func darwinKeyCode(name string) (uint16, bool) {
	switch name {
	case "enter":
		return 0x24, true
	case "escape":
		return 0x35, true
	case "tab":
		return 0x30, true
	case "backspace":
		return 0x33, true
	case "delete":
		return 0x75, true
	case "left":
		return 0x7B, true
	case "right":
		return 0x7C, true
	case "down":
		return 0x7D, true
	case "up":
		return 0x7E, true
	case "home":
		return 0x73, true
	case "end":
		return 0x77, true
	case "pageup":
		return 0x74, true
	case "pagedown":
		return 0x79, true
	case "space":
		return 0x31, true
	default:
		return 0, false
	}
}

func darwinType(s string) error {
	for _, r := range s {
		if r == '\n' || r == '\r' {
			if err := darwinKey("enter"); err != nil {
				return err
			}
			continue
		}
		if r > 0xFFFF {
			continue
		}
		C.screenUnicode(C.uint16_t(r))
	}
	return nil
}
