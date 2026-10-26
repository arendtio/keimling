package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
)

func runRemoteCLI(connectURL, secret string) error {
	c, err := newRemoteClient(connectURL, secret)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if _, err := c.Health(ctx); err != nil {
		return err
	}
	lines, model, mode, busy, err := c.Session(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("keimling remote | %s | mode=%s | %s\n", model, mode, c.base)
	for _, line := range lines {
		fmt.Println(line)
	}
	var state atomic.Value
	state.Store("idle")
	if busy {
		state.Store("running")
	}

	go func() {
		err := c.StreamEvents(ctx, sseHandler{
			OnLine: func(text string) { fmt.Print(text) },
			OnStatus: func(s string) {
				state.Store(s)
				if s == "awaiting_approval" {
					fmt.Println("\nApproval required. Reply y/n or /approve /deny")
				}
			},
		})
		if err != nil && ctx.Err() == nil {
			fmt.Fprintln(os.Stderr, "events:", err)
			cancel()
		}
	}()

	in := bufio.NewReader(os.Stdin)
	fmt.Println("Connected. Type a message, /approve, /deny, or /exit.")
	for {
		line, err := in.ReadString('\n')
		if err != nil {
			cancel()
			return nil
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if line == "/exit" || line == "/quit" {
			cancel()
			return nil
		}
		st, _ := state.Load().(string)
		lower := strings.ToLower(line)
		if st == "awaiting_approval" || lower == "/approve" || lower == "/deny" {
			allow := lower == "y" || lower == "yes" || lower == "j" || lower == "ja" || lower == "/approve"
			deny := lower == "n" || lower == "no" || lower == "/deny"
			if allow || deny {
				if err := c.Approve(ctx, allow); err != nil {
					fmt.Println("approve:", err)
				}
				continue
			}
		}
		if err := c.Message(ctx, line); err != nil {
			fmt.Println("send:", err)
		}
	}
}
