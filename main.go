package main

import (
	"context"
	"flag"
	"lightning-splitter/config"
	"lightning-splitter/lnd"
	"time"
)

var (
	configPath *string
	lndClient  *lnd.Client
)

func init() {
	configPath = flag.String("config", "config/config.yaml", "configuration file")
}

func main() {
	flag.Parse()

	// Load the configuration file
	config.LoadConfig(*configPath)

	lndClient = lnd.New()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	lndClient.Connect(ctx)

	lndClient.PrintInfo(ctx)
}
