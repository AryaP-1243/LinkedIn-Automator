package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/linkedin-automation/pkg/browser"
	"github.com/linkedin-automation/pkg/config"
	"github.com/linkedin-automation/pkg/logger"
	"github.com/linkedin-automation/pkg/search"
	"github.com/linkedin-automation/pkg/stealth"
)

func main() {
	// Load configuration
	cfg, err := config.Load("./config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	mainLog := logger.WithComponent("visual-demo")

	mainLog.Info("Starting visual LinkedIn demo...")

	// Create stealth components
	fingerprint := stealth.NewFingerprintManager(&cfg.Stealth.Fingerprinting, &cfg.Browser)
	mouse := stealth.NewMouseController(&cfg.Stealth.MouseMovement)
	timing := stealth.NewTimingController(&cfg.Stealth.Timing)
	scroll := stealth.NewScrollController(&cfg.Stealth.Scrolling, timing)
	typing := stealth.NewTypingController(&cfg.Stealth.Typing)

	// Create browser
	br := browser.New(browser.Options{
		Config:      cfg,
		Fingerprint: fingerprint,
		Mouse:       mouse,
		Timing:      timing,
		Scroll:      scroll,
		Typing:      typing,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		mainLog.Info("Received shutdown signal, cleaning up...")
		cancel()
	}()

	// Launch browser
	mainLog.Info("Launching browser...")
	if err := br.Launch(ctx); err != nil {
		log.Fatalf("Failed to launch browser: %v", err)
	}
	defer br.Close()

	// Authenticate (simplified - no storage needed for demo)
	mainLog.Info("Authenticating...")
	if err := br.Navigate(ctx, "https://www.linkedin.com/login"); err != nil {
		log.Fatalf("Failed to navigate to login: %v", err)
	}

	// Type credentials
	if err := br.Type(ctx, "input[name='session_key']", cfg.LinkedIn.Email); err != nil {
		mainLog.Warn("Failed to type email: %v", err)
	}
	if err := br.Type(ctx, "input[name='session_password']", cfg.LinkedIn.Password); err != nil {
		mainLog.Warn("Failed to type password: %v", err)
	}
	if err := br.Click(ctx, "button[type='submit']"); err != nil {
		mainLog.Warn("Failed to click login: %v", err)
	}

	time.Sleep(5 * time.Second) // Wait for login

	// Create visual searcher
	visualSearcher := search.NewVisualSearcher(br, timing, scroll)

	// Perform visual search
	searchTerm := "Bill Gates"
	if len(cfg.Search.Keywords) > 0 {
		searchTerm = cfg.Search.Keywords[0]
	}

	mainLog.Info("Starting visual search for: %s", searchTerm)
	if err := visualSearcher.SearchWithUI(ctx, searchTerm); err != nil {
		log.Fatalf("Visual search failed: %v", err)
	}

	mainLog.Info("Visual search completed successfully!")
	mainLog.Info("Demo will stay open for 30 seconds so you can see the results...")

	// Keep browser open for a bit to see the results
	time.Sleep(30 * time.Second)

	mainLog.Info("Demo completed, closing...")
}
