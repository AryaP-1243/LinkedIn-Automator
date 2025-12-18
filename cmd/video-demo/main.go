package main

import (
	"context"
	"fmt"
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
	log := logger.WithComponent("video-demo")
	log.Info("Starting LinkedIn automation with VIDEO RECORDING...")

	// Load configuration
	cfg, err := config.Load("./config.yaml")
	if err != nil {
		log.Error("Failed to load config: %v", err)
		os.Exit(1)
	}

	// Ensure browser is NOT headless for video recording
	cfg.Browser.Headless = false

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

	// Setup context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Info("Received shutdown signal, cleaning up...")
		cancel()
	}()

	// Launch browser
	log.Info("Launching browser...")
	if err := br.Launch(ctx); err != nil {
		log.Error("Failed to launch browser: %v", err)
		os.Exit(1)
	}
	defer br.Close()

	// START VIDEO RECORDING
	log.Info("🎥 STARTING VIDEO RECORDING...")
	videoFile := fmt.Sprintf("linkedin_automation_%d.webm", time.Now().Unix())

	// Start recording using Rod's built-in recorder
	page := br.Page()
	recorder, err := page.RecordVideo(videoFile)
	if err != nil {
		log.Error("Failed to start video recording: %v", err)
		os.Exit(1)
	}
	defer func() {
		log.Info("🎥 STOPPING VIDEO RECORDING...")
		if err := recorder.Stop(); err != nil {
			log.Error("Failed to stop recording: %v", err)
		} else {
			log.Info("✅ Video saved to: %s", videoFile)
		}
	}()

	log.Info("🎥 Recording started! File: %s", videoFile)

	// Navigate to LinkedIn
	log.Info("Navigating to LinkedIn...")
	if err := br.Navigate(ctx, "https://www.linkedin.com"); err != nil {
		log.Error("Failed to navigate: %v", err)
		return
	}

	// Wait a moment for page to settle
	time.Sleep(2 * time.Second)

	// STEP 1: Search for Jensen Huang
	log.Info("STEP 1: Searching for 'Jensen Huang'...")

	searcher := search.NewVisualSearcher(&cfg.Search, br, timing, scroll)

	results, err := searcher.PerformSearch(ctx, "Jensen Huang")
	if err != nil {
		log.Error("Search failed: %v", err)
	} else {
		log.Info("Search completed! Found %d results", len(results))
	}

	// Wait to show results
	time.Sleep(3 * time.Second)

	// STEP 2: Try to apply NVIDIA filter (optional - may fail if UI changed)
	log.Info("STEP 2: Attempting to apply Company filter 'NVIDIA'...")
	if err := searcher.ApplyFilter(ctx, "company", "NVIDIA"); err != nil {
		log.Warn("Failed to apply company filter: %v", err)
	} else {
		log.Info("Company filter applied!")
		time.Sleep(2 * time.Second)
	}

	// STEP 3: Scroll through results
	log.Info("STEP 3: Scrolling through results...")
	for i := 0; i < 2; i++ {
		if err := br.Scroll(ctx, 300); err != nil {
			log.Error("Scroll down failed: %v", err)
			break
		}
		time.Sleep(1 * time.Second)
	}

	// STEP 4: Click the first profile
	log.Info("STEP 4: Clicking the first profile...")
	if err := searcher.ClickProfile(ctx, 0); err != nil {
		log.Error("Failed to click profile: %v", err)
	} else {
		log.Info("Profile clicked successfully!")
	}

	// Wait to show the profile
	log.Info("Displaying profile for 5 seconds...")
	time.Sleep(5 * time.Second)

	log.Info("Demo completed! Video recording will be saved...")
	// Video will be saved when recorder.Stop() is called in defer
}
