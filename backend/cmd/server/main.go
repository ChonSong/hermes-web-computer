package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"hermes-web-computer/backend/audio"
	"hermes-web-computer/backend/ws"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
	}

	audioURL := os.Getenv("FUN_AUDIO_WS")
	if audioURL == "" {
		audioURL = "ws://localhost:11235/api/chat"
	}

	mux := ws.NewMultiplexer()
	mux.SetAudioBridge(audio.NewBridge(audioURL))

	// Start telemetry sync if endpoint configured
	if endpoint := os.Getenv("TELEMETRY_ENDPOINT"); endpoint != "" {
		if syncer := mux.GetTelemetrySyncer(); syncer != nil {
			go syncer.Start()
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux.Router(),
	}

	go func() {
		<-ctx.Done()
		log.Println("shutting down...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()

	log.Printf("agent-os backend starting on :%s", port)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
	log.Println("server stopped")
}
