package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/linkedin-automation/pkg/auth"
	"github.com/linkedin-automation/pkg/browser"
	"github.com/linkedin-automation/pkg/config"
	"github.com/linkedin-automation/pkg/logger"
	"github.com/linkedin-automation/pkg/search"
	"github.com/linkedin-automation/pkg/stealth"
	"github.com/linkedin-automation/pkg/storage"
)

func main() {
	// Load configuration
	cfg, err := config.Load("./config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	mainLog := logger.WithComponent("visual-demo")
	mainLog.Info("Starting LinkedIn visual demo...")
	mainLog.Info("This demo will show visible mouse movements, typing, and scrolling")

	// Create stealth components
	fingerprint := stealth.NewFingerprintManager(&cfg.Stealth.Fingerprinting, &cfg.Browser)
	mouse := stealth.NewMouseController(&cfg.Stealth.MouseMovement)
	timing := stealth.NewTimingController(&cfg.Stealth.Timing)
	scroll := stealth.NewScrollController(&cfg.Stealth.Scrolling, timing)
	typing := stealth.NewTypingController(&cfg.Stealth.Typing)

	// Create storage for session management
	store, err := storage.New(&cfg.Storage)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}

	// Create browser
	br := browser.New(browser.Options{
		Config:      cfg,
		Fingerprint: fingerprint,
		Mouse:       mouse,
		Timing:      timing,
		Scroll:      scroll,
		Typing:      typing,
	})

	// NO TIMEOUT - runs indefinitely until Ctrl+C
	ctx := context.Background()

	// Handle graceful shutdown (Ctrl+C only)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		mainLog.Info("Received shutdown signal, closing browser...")
		br.Close()
		os.Exit(0)
	}()

	// Launch browser
	mainLog.Info("Launching Google Chrome...")
	if err := br.Launch(ctx); err != nil {
		log.Fatalf("Failed to launch browser: %v", err)
	}

	// Create authenticator - this uses visible mouse movements and typing!
	authenticator := auth.New(&cfg.LinkedIn, br, store, timing)

	// Perform login with visible interactions
	mainLog.Info("Starting authentication (watch the mouse and keyboard!)...")
	result, err := authenticator.Login(ctx)
	if err != nil {
		mainLog.Error("Login failed: %v", err)
		mainLog.Info("Browser will stay open - you can manually interact")
		// Keep browser open for debugging
		select {}
	}

	if !result.Success {
		mainLog.Warn("Login was not successful: %s", result.Message)
		mainLog.Info("Browser will stay open - you can manually interact")
		// Keep browser open for debugging
		select {}
	}

	mainLog.Info("Login successful! Now performing visual search...")

	// Create visual searcher
	visualSearcher := search.NewVisualSearcher(br, timing, scroll)

	// Get search term from config
	searchTerm := "Bill Gates"
	if len(cfg.Search.Keywords) > 0 {
		searchTerm = cfg.Search.Keywords[0]
	}

	// Perform visual search with mouse movements
	mainLog.Info("Starting visual search for: %s", searchTerm)
	mainLog.Info("Watch the mouse pointer move to the search box and type...")

	if err := visualSearcher.SearchWithUI(ctx, searchTerm); err != nil {
		mainLog.Warn("Search encountered an issue: %v", err)
	} else {
		mainLog.Info("Search completed successfully!")
	}

	// Click on the first profile in results
	mainLog.Info("Clicking on first profile in search results...")
	if err := visualSearcher.ClickProfile(ctx, 0); err != nil {
		mainLog.Warn("Failed to click profile: %v", err)
	} else {
		mainLog.Info("Profile opened successfully!")

		// Scroll through the profile (Screenshot 3 feature)
		mainLog.Info("Scrolling through profile...")
		if err := visualSearcher.ScrollProfile(ctx); err != nil {
			mainLog.Warn("Failed to scroll profile: %v", err)
		}

		// Try to send connection request (Screenshot 4 feature)
		mainLog.Info("Attempting to send connection request...")
		if err := visualSearcher.SendConnectionRequest(ctx); err != nil {
			mainLog.Warn("Could not send connection request: %v", err)

			// Try sending a message instead (Screenshot 5 feature)
			mainLog.Info("Trying to send a message instead...")
			msg := "Hi! I'd love to connect and learn more about your work."
			if err := visualSearcher.SendMessage(ctx, msg); err != nil {
				mainLog.Warn("Could not send message: %v", err)
			}
		}
	}

	mainLog.Info("Demo complete! Browser will stay open.")
	mainLog.Info("Press Ctrl+C to exit...")

	// Keep running forever until user presses Ctrl+C
	select {}
}
