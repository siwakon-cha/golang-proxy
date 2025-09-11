package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"rpc-proxy/internal/config"
	"rpc-proxy/internal/proxy"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Configuration loaded successfully")
	log.Printf("Supported chains: %d", len(cfg.Chains))
	for _, chain := range cfg.Chains {
		if chain.IsEnabled {
			log.Printf("  - %s (%s) - Path: /rpc/%s - Endpoints: %d", 
				chain.DisplayName, chain.Name, chain.RPCPath, len(cfg.ChainEndpoints[chain.Name]))
		}
	}

	// Create multi-chain health checker
	multiChainHealthChecker := cfg.CreateMultiChainHealthChecker()
	if multiChainHealthChecker == nil {
		log.Fatalf("Failed to create multi-chain health checker")
	}

	// Create proxy server with multi-chain support
	proxyServer := proxy.NewServer(cfg, multiChainHealthChecker)

	// Start health checking for all chains
	multiChainHealthChecker.Start()
	defer func() {
		log.Println("Stopping multi-chain health checker...")
		multiChainHealthChecker.Stop()
	}()

	server := &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%d", cfg.Server.Port),
		Handler: proxyServer.Handler(),
	}

	go func() {
		log.Printf("Starting Multi-Chain RPC Proxy server on port %d", cfg.Server.Port)
		log.Printf("Available endpoints:")
		log.Printf("  - /health (overall health status)")
		log.Printf("  - /health/{chainName} (chain-specific health)")
		log.Printf("  - /rpc/{chainName} (chain-specific RPC)")
		log.Printf("  - /rpc (legacy, defaults to ethereum)")
		
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}