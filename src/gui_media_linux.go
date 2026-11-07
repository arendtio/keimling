//go:build cgo && !nogui && linux && webkitgtk_41

package main

/*
#cgo pkg-config: webkit2gtk-4.1
#include <webkit2/webkit2.h>

static gboolean keimling_on_permission_request(WebKitWebView *web_view,
                                               WebKitPermissionRequest *request,
                                               gpointer user_data) {
	(void)web_view;
	(void)user_data;
	// Local desktop app: auto-allow mic (and device enumeration). No browser prompt.
	if (WEBKIT_IS_USER_MEDIA_PERMISSION_REQUEST(request) ||
	    WEBKIT_IS_DEVICE_INFO_PERMISSION_REQUEST(request)) {
		webkit_permission_request_allow(request);
		return TRUE;
	}
	return FALSE;
}

static void keimling_allow_media(void *webview) {
	if (webview == NULL || !WEBKIT_IS_WEB_VIEW(webview)) {
		return;
	}
	WebKitWebView *view = WEBKIT_WEB_VIEW(webview);
	WebKitSettings *settings = webkit_web_view_get_settings(view);
	// WebRTC defaults to off. The microphone capture uses WebAudio.
	// Playback starts again when you interrupt, outside the original click.
	webkit_settings_set_enable_webrtc(settings, TRUE);
	webkit_settings_set_enable_webaudio(settings, TRUE);
	webkit_settings_set_enable_media(settings, TRUE);
	webkit_settings_set_enable_media_stream(settings, TRUE);
	webkit_settings_set_media_playback_requires_user_gesture(settings, FALSE);
	g_signal_connect(view, "permission-request",
	                 G_CALLBACK(keimling_on_permission_request), NULL);
}
*/
import "C"
import (
	"unsafe"

	webview "github.com/samcharles93/webview_go"
)

// allowMediaPermissions auto-grants mic/camera permission requests inside the
// embedded WebKit view so getUserMedia does not fail with NotAllowedError.
func allowMediaPermissions(w webview.WebView) {
	view := w.NativeHandle(webview.NativeHandleKindBrowserController)
	if view == nil {
		return
	}
	C.keimling_allow_media(unsafe.Pointer(view))
}
