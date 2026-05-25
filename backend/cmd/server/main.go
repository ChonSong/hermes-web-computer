package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"hermes-web-computer/backend/audio"
	"hermes-web-computer/backend/config"
	"hermes-web-computer/backend/docker"
	"hermes-web-computer/backend/session"
	"hermes-web-computer/backend/ws"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3005"
	}

	audioURL := os.Getenv("FUN_AUDIO_WS")
	if audioURL == "" {
		audioURL = "ws://localhost:11235/api/chat"
	}

	mux := ws.NewMultiplexer()

	// Docker manager — enables container lifecycle UI
	dockerMgr, err := docker.NewManager()
	if err != nil {
		log.Printf("docker manager init failed: %v (continuing without docker)", err)
	} else {
		mux.SetDockerManager(dockerMgr)
		log.Println("docker manager attached")
	}

	// Session store — ~/.hermes/hermes-web-computer/sessions/
	homeDir, _ := os.UserHomeDir()
	stateDir := os.Getenv("HWC_STATE_DIR")
	if stateDir == "" {
		stateDir = filepath.Join(homeDir, ".hermes", "hermes-web-computer")
	}
	store, err := session.NewStore(stateDir)
	if err != nil {
		log.Printf("session store init failed: %v (continuing without sessions)", err)
	} else {
		mux.SetSessionStore(store)
		log.Printf("session store initialized at %s", stateDir)
	}

	// Config manager
	configMgr := config.NewManager(filepath.Join(stateDir, "config.yaml"))
	mux.SetConfigManager(configMgr)
	log.Printf("config manager initialized at %s", stateDir)

	mux.SetAudioBridge(audio.NewBridge(audioURL))

	// Initialize Xpra manager on a fixed display
	if err := mux.InitializeXpra(10); err != nil {
		log.Printf("xpra initialization skipped (xpra may not be installed): %v", err)
	}

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

	log.Printf("hermes-web-computer backend starting on :%s", port)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
	log.Println("server stopped")
}
