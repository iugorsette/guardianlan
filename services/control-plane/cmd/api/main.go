package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/sette/guardian-lan/services/control-plane/internal/app"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := app.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
