package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func runAppServer(listen, secret string, agent *Agent) error {
	if err := validateServerListen(listen, secret); err != nil {
		return err
	}
	srv := &http.Server{
		Addr:              listen,
		Handler:           newAppServer(appServerOpts{AccessSecret: secret, Agent: agent}).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		fmt.Fprintf(os.Stderr, "keimling server listening on http://%s/\n", listen)
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
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		return <-errCh
	}
}
