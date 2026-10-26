package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"
)

func randomAccessSecret() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func uiURL(base, secret string) string {
	u, err := url.Parse(base)
	if err != nil {
		return base
	}
	q := u.Query()
	if secret != "" {
		q.Set("access_secret", secret)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// runGUIRemote opens the system webview against an existing agent URL.
func runGUIRemote(connectURL, secret string) error {
	if _, err := newRemoteClient(connectURL, secret); err != nil {
		return err
	}
	return openWebView(uiURL(connectURL, secret), "Keimling")
}

// runGUILocal serves the agent on an ephemeral localhost port and opens WebView.
func runGUILocal(agent *Agent, secret string) error {
	if secret == "" {
		var err error
		secret, err = randomAccessSecret()
		if err != nil {
			return err
		}
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	addr := ln.Addr().String()
	srv := &http.Server{
		Handler:           newAppServer(appServerOpts{AccessSecret: secret, Agent: agent}).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		fmt.Fprintf(os.Stderr, "keimling local UI http://%s/\n", addr)
		err := srv.Serve(ln)
		if err == http.ErrServerClosed {
			errCh <- nil
			return
		}
		errCh <- err
	}()

	page := uiURL("http://"+addr+"/", secret)
	guiErr := openWebView(page, "Keimling")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	serveErr := <-errCh
	if guiErr != nil {
		return guiErr
	}
	return serveErr
}
