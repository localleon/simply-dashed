package main

import (
	"context"
	"flag"
	"log"

	"github.com/localleon/simply-dashed/internal/config"
	"github.com/localleon/simply-dashed/internal/icons"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to YAML configuration")
	iconDir := flag.String("icon-dir", "data/icons", "Directory for downloaded icons")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	cache, err := icons.NewCache(*iconDir)
	if err != nil {
		log.Fatalf("create cache: %v", err)
	}

	if err := cache.Prime(context.Background(), cfg, true); err != nil {
		log.Fatalf("fetch icons: %v", err)
	}
}
