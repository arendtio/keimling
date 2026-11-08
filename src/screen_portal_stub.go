//go:build !linux

package main

import (
	"context"
	"fmt"
)

func openScreenCast(ctx context.Context) (*screenCast, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("screen share is only available on the Linux desktop")
}
