package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"biometrics-cli/internal/controlplane"
)

func main() {
	fmt.Fprintln(os.Stderr, "[DEPRECATION] `biometrics` is a temporary CLI shim and now forwards to BIOMETRICS V3 controlplane runtime.")
	fmt.Fprintln(os.Stderr, "[DEPRECATION] Build/run `cmd/controlplane` directly for canonical behavior.")

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := controlplane.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
