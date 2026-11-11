//go:build !linux && !windows && !darwin

package main

import (
	"context"
	"fmt"
)

func screenShareChooser() (string, []screenTarget) {
	return "unavailable", nil
}

func openScreenCast(ctx context.Context, input bool, target string) (*screenCast, error) {
	_ = input
	_ = target
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("screen share is not available on this system")
}
