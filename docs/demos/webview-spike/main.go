// WebView + embedded HTTP spike for Keimling client/server exploration.
// Builds against system WebKitGTK (Linux) / WKWebView / WebView2 via
// github.com/samcharles93/webview_go.
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	webview "github.com/samcharles93/webview_go"
)

func main() {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()
	url := "http://" + addr + "/"

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>Keimling WebView Spike</title>
<style>
 body{font-family:system-ui,sans-serif;margin:2rem;background:#1a1a1a;color:#e8e8e8}
 .ok{color:#7dcea0} code{background:#333;padding:.1rem .3rem;border-radius:3px}
</style></head><body>
<h1 class="ok">WebView ↔ HTTP ok</h1>
<p>Served from embedded Go HTTP on <code>%s</code>.</p>
<p>Same page a browser would see.</p>
</body></html>`, addr)
	})

	srv := &http.Server{Handler: mux}
	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Printf("http: %v", err)
		}
	}()

	if os.Getenv("SPIKE_HTTP_ONLY") == "1" {
		log.Printf("SPIKE_HTTP_ONLY=1 — listening on %s (no WebView)", url)
		select {}
	}

	w, err := webview.NewWithOptions(webview.Options{
		Title:  "Keimling WebView Spike",
		Width:  640,
		Height: 420,
		Hint:   webview.HintNone,
		URL:    url,
		Debug:  os.Getenv("SPIKE_DEBUG") == "1",
	})
	if err != nil {
		log.Fatalf("webview: %v", err)
	}
	defer w.Destroy()

	log.Printf("webview → %s", url)
	if err := w.Run(); err != nil {
		log.Fatalf("run: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
