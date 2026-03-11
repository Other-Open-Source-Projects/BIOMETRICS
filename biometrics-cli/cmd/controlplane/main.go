package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"biometrics-cli/internal/controlplane"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := controlplane.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
