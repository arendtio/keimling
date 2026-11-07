//go:build cgo && !nogui && linux && webkitgtk_41

package main

/*
#cgo pkg-config: webkit2gtk-4.1
#include <webkit2/webkit2.h>
#include <stdlib.h>

extern void keimlingGoOpenExternal(char* uri);
extern void keimlingGoAcceptDrop(char* uri);

static void keimling_on_drag_data(GtkWidget *widget,
                                  GdkDragContext *context,
                                  gint x,
                                  gint y,
                                  GtkSelectionData *selection,
                                  guint info,
                                  guint time,
                                  gpointer user_data) {
	(void)widget;
	(void)context;
	(void)x;
	(void)y;
	(void)info;
	(void)time;
	(void)user_data;
	gchar **uris = gtk_selection_data_get_uris(selection);
	if (uris == NULL) {
		return;
	}
	for (guint i = 0; uris[i] != NULL; i++) {
		keimlingGoAcceptDrop(uris[i]);
	}
	g_strfreev(uris);
}

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
	// A file dropped from Dolphin is a navigation to file://. Ignore it or the UI is replaced by the picture.
	if (g_str_has_prefix(uri, "file:")) {
		keimlingGoAcceptDrop((char *)uri);
		webkit_policy_decision_ignore(decision);
		return TRUE;
	}
	WebKitNavigationType ntype = webkit_navigation_action_get_navigation_type(action);
	gboolean open_out = (type == WEBKIT_POLICY_DECISION_TYPE_NEW_WINDOW_ACTION) ||
	                    (ntype == WEBKIT_NAVIGATION_TYPE_LINK_CLICKED);
	if (!open_out) {
		// The default policy plays a moment, then marks the audio session
		// interrupted. A call needs the whole spoken reply.
		WebKitWebsitePolicies *policies = webkit_website_policies_new_with_policies(
			"autoplay", WEBKIT_AUTOPLAY_ALLOW, NULL);
		webkit_policy_decision_use_with_policies(decision, policies);
		g_object_unref(policies);
		return TRUE;
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
	g_signal_connect(WEBKIT_WEB_VIEW(webview), "drag-data-received",
	                 G_CALLBACK(keimling_on_drag_data), NULL);
}
*/
import "C"
import (
	"unsafe"

	webview "github.com/samcharles93/webview_go"
)

var guiDropView webview.WebView

//export keimlingGoAcceptDrop
func keimlingGoAcceptDrop(uri *C.char) {
	js, ok := fileDropEval(C.GoString(uri))
	if !ok || guiDropView == nil {
		return
	}
	w := guiDropView
	_ = w.Dispatch(func() { _ = w.Eval(js) })
}

//export keimlingGoOpenExternal
func keimlingGoOpenExternal(uri *C.char) {
	_ = openExternalURL(C.GoString(uri))
}

func interceptExternalLinks(w webview.WebView) {
	guiDropView = w
	view := w.NativeHandle(webview.NativeHandleKindBrowserController)
	if view == nil {
		return
	}
	C.keimling_intercept_external_links(unsafe.Pointer(view))
}
