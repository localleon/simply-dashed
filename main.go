package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/localleon/simply-dashed/internal/config"
	"github.com/localleon/simply-dashed/internal/icons"
	"github.com/localleon/simply-dashed/internal/server"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to YAML configuration")
	iconDir := flag.String("icon-dir", "data/icons", "Directory for downloaded icons")
	refreshIcons := flag.Bool("refresh-icons", true, "Refresh icons from remote URLs on startup")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	iconCache, err := icons.NewCache(*iconDir)
	if err != nil {
		log.Fatalf("create icon cache: %v", err)
	}

	if err := iconCache.Prime(context.Background(), cfg, *refreshIcons); err != nil {
		log.Printf("icon prefetch completed with errors: %v", err)
	}

	app, err := server.New(cfg, iconCache)
	if err != nil {
		log.Fatalf("create server: %v", err)
	}

	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           app.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("serving simply-dashed on %s", cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("serve: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}
