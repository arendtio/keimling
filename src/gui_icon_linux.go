//go:build cgo && !nogui && linux

package main

/*
#cgo pkg-config: gtk+-3.0 gdk-pixbuf-2.0
#include <gtk/gtk.h>
#include <gdk-pixbuf/gdk-pixbuf.h>
#include <stdlib.h>

static void keimling_set_prgname(const char *name) {
	g_set_prgname(name);
	gdk_set_program_class(name);
}

static int keimling_set_window_icon(void *window, const unsigned char *data, int len) {
	GError *err = NULL;
	GdkPixbufLoader *loader = gdk_pixbuf_loader_new();
	if (!gdk_pixbuf_loader_write(loader, data, (gsize)len, &err)) {
		if (err) g_error_free(err);
		g_object_unref(loader);
		return 0;
	}
	if (!gdk_pixbuf_loader_close(loader, &err)) {
		if (err) g_error_free(err);
		g_object_unref(loader);
		return 0;
	}
	GdkPixbuf *pixbuf = gdk_pixbuf_loader_get_pixbuf(loader);
	if (pixbuf == NULL) {
		g_object_unref(loader);
		return 0;
	}
	g_object_ref(pixbuf);
	gtk_window_set_icon(GTK_WINDOW(window), pixbuf);
	gtk_window_set_default_icon(pixbuf);
	g_object_unref(pixbuf);
	g_object_unref(loader);
	return 1;
}
*/
import "C"
import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unsafe"

	webview "github.com/samcharles93/webview_go"
)

func setProgramName(name string) {
	c := C.CString(name)
	defer C.free(unsafe.Pointer(c))
	C.keimling_set_prgname(c)
}

func applyWindowIcon(w webview.WebView) {
	win := w.NativeHandle(webview.NativeHandleKindUIWindow)
	if win == nil || len(keimlingIconPNG) == 0 {
		return
	}
	C.keimling_set_window_icon(win, (*C.uchar)(unsafe.Pointer(&keimlingIconPNG[0])), C.int(len(keimlingIconPNG)))
}

func installDesktopIcon() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	hicolor := filepath.Join(home, ".local", "share", "icons", "hicolor")
	scalableDir := filepath.Join(hicolor, "scalable", "apps")
	if err := os.MkdirAll(scalableDir, 0o755); err != nil {
		return err
	}
	svgPath := filepath.Join(scalableDir, desktopAppID+".svg")
	if len(keimlingIconSVG) > 0 {
		if err := os.WriteFile(svgPath, keimlingIconSVG, 0o644); err != nil {
			return err
		}
	}
	// Raster fallback when a theme ignores scalable/.
	if len(keimlingIconPNG) > 0 {
		pngDir := filepath.Join(hicolor, "128x128", "apps")
		if err := os.MkdirAll(pngDir, 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(pngDir, desktopAppID+".png"), keimlingIconPNG, 0o644); err != nil {
			return err
		}
	}

	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}

	appDir := filepath.Join(home, ".local", "share", "applications")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		return err
	}
	// Absolute SVG path avoids stale Plasma theme-name resolution.
	iconRef := desktopAppID
	if len(keimlingIconSVG) > 0 {
		iconRef = svgPath
	}
	desktop := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=Keimling
Comment=Keimling agent
Exec=%s --ui
Icon=%s
Terminal=false
Categories=Development;
StartupWMClass=%s
StartupNotify=true
`, shellQuote(exe), iconRef, desktopAppID)
	if err := os.WriteFile(filepath.Join(appDir, desktopAppID+".desktop"), []byte(desktop), 0o644); err != nil {
		return err
	}
	_ = os.Remove(filepath.Join(home, ".cache", "icon-cache.kcache"))
	_ = exec.Command("update-desktop-database", appDir).Run()
	_ = exec.Command("gtk-update-icon-cache", "-f", hicolor).Run()
	_ = exec.Command("kbuildsycoca6", "--noincremental").Run()
	_ = exec.Command("kbuildsycoca5", "--noincremental").Run()
	return nil
}

func shellQuote(path string) string {
	if path == "" {
		return `""`
	}
	if !strings.ContainsAny(path, " \t\"'\\$`") {
		return path
	}
	return `"` + strings.ReplaceAll(path, `"`, `\"`) + `"`
}
