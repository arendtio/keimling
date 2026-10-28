package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// runHostServer listens as the workspace supervisor (no nested workers of workers).
func runHostServer(listen, secret, configPath, initialWorkspace string) error {
	if err := validateServerListen(listen, secret); err != nil {
		return err
	}
	h, err := newHost(secret, configPath)
	if err != nil {
		return err
	}
	defer h.StopAll()

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	if initialWorkspace == "" {
		cwd, _ := os.Getwd()
		ws, _, _, e := resolveStartupWorkspace(cwd)
		if e != nil {
			return e
		}
		initialWorkspace = ws.ID
	}
	if err := h.Attach(ctx, initialWorkspace); err != nil {
		return err
	}

	srv := &http.Server{
		Addr:              listen,
		Handler:           h.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		fmt.Fprintf(os.Stderr, "keimling host listening on http://%s/ (workspace %s)\n", listen, initialWorkspace)
		err := srv.ListenAndServe()
		if err == http.ErrServerClosed {
			errCh <- nil
			return
		}
		errCh <- err
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	select {
	case err := <-errCh:
		return err
	case <-sigCh:
		shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutCancel()
		_ = srv.Shutdown(shutCtx)
		h.StopAll()
		return <-errCh
	}
}

// runHostGUI starts an ephemeral host and opens the WebView against it.
func runHostGUI(secret, configPath, initialWorkspace string) error {
	h, err := newHost(secret, configPath)
	if err != nil {
		return err
	}
	defer h.StopAll()

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	if initialWorkspace == "" {
		cwd, _ := os.Getwd()
		ws, _, _, e := resolveStartupWorkspace(cwd)
		if e != nil {
			return e
		}
		initialWorkspace = ws.ID
	} else if _, e := readWorkspaceMeta(initialWorkspace); e != nil {
		cwd, _ := os.Getwd()
		ws, _, _, e2 := resolveStartupWorkspace(cwd)
		if e2 != nil {
			return e2
		}
		initialWorkspace = ws.ID
	}
	if err := h.Attach(ctx, initialWorkspace); err != nil {
		return err
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	addr := ln.Addr().String()
	srv := &http.Server{
		Handler:           h.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		fmt.Fprintf(os.Stderr, "keimling host UI http://%s/ (workspace %s)\n", addr, initialWorkspace)
		err := srv.Serve(ln)
		if err == http.ErrServerClosed {
			errCh <- nil
			return
		}
		errCh <- err
	}()

	page := uiURL("http://"+addr+"/", h.Secret())
	guiErr := openWebViewLocal(page, "Keimling", configPath, page)

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutCancel()
	_ = srv.Shutdown(shutCtx)
	h.StopAll()
	serveErr := <-errCh
	if guiErr != nil {
		return guiErr
	}
	return serveErr
}
