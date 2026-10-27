//go:build cgo && !nogui && linux && webkitgtk_41

package main

/*
#cgo pkg-config: webkit2gtk-4.1
#include <webkit2/webkit2.h>
#include <stdlib.h>

extern void keimlingGoOpenExternal(char* uri);

static gboolean keimling_on_decide_policy(WebKitWebView *web_view,
                                          WebKitPolicyDecision *decision,
                                          WebKitPolicyDecisionType type,
                                          gpointer user_data) {
	(void)web_view;
	(void)user_data;
	if (type != WEBKIT_POLICY_DECISION_TYPE_NAVIGATION_ACTION &&
	    type != WEBKIT_POLICY_DECISION_TYPE_NEW_WINDOW_ACTION) {
		return FALSE;
	}
	WebKitNavigationPolicyDecision *nav = WEBKIT_NAVIGATION_POLICY_DECISION(decision);
	WebKitNavigationAction *action = webkit_navigation_policy_decision_get_navigation_action(nav);
	WebKitURIRequest *req = webkit_navigation_action_get_request(action);
	const gchar *uri = webkit_uri_request_get_uri(req);
	if (uri == NULL) {
		return FALSE;
	}
	WebKitNavigationType ntype = webkit_navigation_action_get_navigation_type(action);
	gboolean open_out = (type == WEBKIT_POLICY_DECISION_TYPE_NEW_WINDOW_ACTION) ||
	                    (ntype == WEBKIT_NAVIGATION_TYPE_LINK_CLICKED);
	if (!open_out) {
		return FALSE;
	}
	if (g_str_has_prefix(uri, "http://") || g_str_has_prefix(uri, "https://")) {
		keimlingGoOpenExternal((char *)uri);
		webkit_policy_decision_ignore(decision);
		return TRUE;
	}
	return FALSE;
}

static void keimling_intercept_external_links(void *webview) {
	if (webview == NULL || !WEBKIT_IS_WEB_VIEW(webview)) {
		return;
	}
	g_signal_connect(WEBKIT_WEB_VIEW(webview), "decide-policy",
	                 G_CALLBACK(keimling_on_decide_policy), NULL);
}
*/
import "C"
import (
	"unsafe"

	webview "github.com/samcharles93/webview_go"
)

//export keimlingGoOpenExternal
func keimlingGoOpenExternal(uri *C.char) {
	_ = openExternalURL(C.GoString(uri))
}

func interceptExternalLinks(w webview.WebView) {
	view := w.NativeHandle(webview.NativeHandleKindBrowserController)
	if view == nil {
		return
	}
	C.keimling_intercept_external_links(unsafe.Pointer(view))
}
