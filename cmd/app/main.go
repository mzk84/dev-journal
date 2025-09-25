package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"dev-journal/internal/config"
	"dev-journal/internal/content"
	"dev-journal/internal/database"
	"dev-journal/internal/server"
)

func main() {
	// Load configuration from environment variables
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	// Initialize SQLite database
	db, err := database.New(cfg.DBPath)
	if err != nil {
		log.Fatalf("Error initializing database: %v", err)
	}
	defer db.Close()
	log.Println("Database initialized.")

	// Check for git command
	if _, err := exec.LookPath("git"); err != nil {
		log.Fatalf("git command not found, please install git")
	}

	// Initial clone of the repository
	if err := content.CloneRepo(cfg); err != nil {
		log.Fatalf("Failed to clone repo: %v", err)
	}
	log.Println("Content repository cloned.")

	// Initial content sync
	if err := content.Sync(cfg.ContentPath, db); err != nil {
		log.Fatalf("Failed to sync content: %v", err)
	}
	log.Println("Initial content sync complete.")

	// Set up the server
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)
	r.Use(middleware.Timeout(60 * time.Second))

	// Create server handler with dependencies
	s := server.New(db, cfg)
	s.RegisterRoutes(r)

	// Start the server
	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	go func() {
		log.Println("Server starting on port 8080...")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exiting")
}
