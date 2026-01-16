package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/autoslides/video-proxy/internal/config"
	"github.com/autoslides/video-proxy/internal/crypto"
	"github.com/autoslides/video-proxy/internal/handler"
	"github.com/autoslides/video-proxy/internal/mapping"
	"github.com/autoslides/video-proxy/internal/proxy"
	"github.com/autoslides/video-proxy/internal/token"
)

func main() {
	// Load configuration
	cfg := config.Load()

	log.Printf("Starting video proxy server on port %s", cfg.Port)
	log.Printf("Upstream API: %s", cfg.UpstreamAPI)
	log.Printf("Mappings file: %s", cfg.MappingsFile)

	// Initialize intranet mapper
	mapper, err := mapping.New(cfg.MappingsFile)
	if err != nil {
		log.Fatalf("Failed to load intranet mappings: %v", err)
	}

	// Initialize components
	cryptoService := crypto.New(cfg.MagicKey)
	tokenCache := token.NewCache(cfg.UpstreamAPI, cfg.MagicKey)
	proxyClient := proxy.NewClient(cfg.RequestTimeout, cfg.IntranetTimeout, mapper)

	// Initialize handlers
	healthHandler := handler.NewHealthHandler()
	streamHandler := handler.NewStreamHandler(cryptoService, tokenCache, proxyClient, cfg.VideoHost)
	segmentHandler := handler.NewSegmentHandler(cryptoService, tokenCache, proxyClient, cfg.VideoHost)
	configHandler := handler.NewConfigHandler(mapper)

	// Set up SIGHUP handler for config reload
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGHUP)
	go func() {
		for range sigChan {
			log.Println("Received SIGHUP, reloading mappings...")
			if err := mapper.Reload(); err != nil {
				log.Printf("Failed to reload mappings: %v", err)
			} else {
				log.Println("Mappings reloaded successfully")
			}
		}
	}()

	// Set up HTTP routes
	mux := http.NewServeMux()

	// Health check
	mux.Handle("/health", healthHandler)

	// Stream endpoints (path-based routing for network mode)
	mux.HandleFunc("/external/stream", streamHandler.ServeHTTP)
	mux.HandleFunc("/intranet/stream", streamHandler.ServeHTTP)

	// TS segment endpoints
	mux.HandleFunc("/external/ts/", segmentHandler.ServeHTTP)
	mux.HandleFunc("/intranet/ts/", segmentHandler.ServeHTTP)

	// Config API
	mux.HandleFunc("/api/v1/config/", configHandler.ServeHTTP)

	// CORS middleware wrapper
	corsHandler := corsMiddleware(mux)

	// Start server
	addr := ":" + cfg.Port
	log.Printf("Server listening on %s", addr)
	if err := http.ListenAndServe(addr, corsHandler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// corsMiddleware adds CORS headers to all responses
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Log request
		log.Printf("%s %s", r.Method, r.URL.Path)

		next.ServeHTTP(w, r)
	})
}

func init() {
	// Suppress unused import error for strings package
	_ = strings.TrimSpace
}
