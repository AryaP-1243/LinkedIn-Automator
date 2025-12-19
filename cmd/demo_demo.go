package main

import (
	"context"
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
	log := logger.WithComponent("demo-demo")
	log.Info("Starting full LinkedIn bot demo sequence...")

	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatal("Failed to load config: %v", err)
	}

	// Override for demo visibility
	cfg.Browser.Headless = false
	cfg.Stealth.MouseMovement.Enabled = true

	// Initialize stealth controllers
	timing := stealth.NewTimingController(&cfg.Stealth.Timing)
	mouse := stealth.NewMouseController(&cfg.Stealth.MouseMovement)
	scroll := stealth.NewScrollController(&cfg.Stealth.Scrolling, timing)
	// Slower, more reliable typing for demo search box
	cfg.Stealth.Typing.MinKeyDelay = 100 * time.Millisecond
	cfg.Stealth.Typing.MaxKeyDelay = 250 * time.Millisecond
	cfg.Stealth.Typing.ThinkPauseChance = 0.02
	typing := stealth.NewTypingController(&cfg.Stealth.Typing)
	fingerprint := stealth.NewFingerprintManager(&cfg.Stealth.Fingerprinting, &cfg.Browser)

	// Initialize browser
	br := browser.New(browser.Options{
		Config:      cfg,
		Fingerprint: fingerprint,
		Mouse:       mouse,
		Timing:      timing,
		Scroll:      scroll,
		Typing:      typing,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()

	if err := br.Launch(ctx); err != nil {
		log.Fatal("Failed to launch browser: %v", err)
	}
	defer br.Close()

	// Handle interrupts
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Info("Interrupt received, shutting down...")
		cancel()
	}()

	log.Info("Navigating to LinkedIn...")
	if err := br.Navigate(ctx, "https://www.linkedin.com"); err != nil {
		log.Fatal("Failed to navigate: %v", err)
	}

	log.Info("Checking login status...")
	time.Sleep(3 * time.Second)

	loggedIn := false
	if br.Exists("#global-nav") || br.Exists(".feed-identity-module") {
		log.Info("✓ Already logged in!")
		loggedIn = true
	}

	if !loggedIn {
		log.Info("Not logged in, performing login...")

		// Handle landing page "Sign in" button if present
		if br.Exists("a.nav__button-secondary") {
			log.Info("Found 'Sign in' link, clicking...")
			_ = br.Click(ctx, "a.nav__button-secondary")
			time.Sleep(2 * time.Second)
		} else if br.Exists("a[data-tracking-control-name='guest_homepage-basic_nav-header-signin']") {
			log.Info("Found guest sign-in link, clicking...")
			_ = br.Click(ctx, "a[data-tracking-control-name='guest_homepage-basic_nav-header-signin']")
			time.Sleep(2 * time.Second)
		}

		// Wait for either #username or #session_key
		selector := ""
		if br.Exists("#username") {
			selector = "#username"
		} else if br.Exists("#session_key") {
			selector = "#session_key"
		}

		if selector != "" {
			log.Info("Found login field: %s", selector)

			// CALM LOGIN: Slow down typing and disable typos JUST for login
			cfg.Stealth.Typing.MinKeyDelay = 200 * time.Millisecond
			cfg.Stealth.Typing.MaxKeyDelay = 400 * time.Millisecond
			cfg.Stealth.Typing.TypoChance = 0
			cfg.Stealth.Typing.ThinkPauseChance = 0
			br.SetTypingController(stealth.NewTypingController(&cfg.Stealth.Typing))

			log.Info("Typing username...")
			if err := br.Type(ctx, selector, cfg.LinkedIn.Email); err != nil {
				log.Fatal("Failed to type username: %v", err)
			}
			time.Sleep(2 * time.Second)

			passSelector := "#password"
			if !br.Exists(passSelector) && br.Exists("#session_password") {
				passSelector = "#session_password"
			}

			log.Info("Typing password...")
			if err := br.Type(ctx, passSelector, cfg.LinkedIn.Password); err != nil {
				log.Fatal("Failed to type password: %v", err)
			}
			time.Sleep(2 * time.Second)

			submitSelector := "button[type='submit']"
			if err := br.Click(ctx, submitSelector); err != nil {
				log.Fatal("Failed to click sign-in button: %v", err)
			}

			// Restore demo-tuned typing for the search phase
			cfg.Stealth.Typing.MinKeyDelay = 100 * time.Millisecond
			cfg.Stealth.Typing.MaxKeyDelay = 250 * time.Millisecond
			cfg.Stealth.Typing.ThinkPauseChance = 0.02
			br.SetTypingController(stealth.NewTypingController(&cfg.Stealth.Typing))

			log.Info("Waiting for login to complete and feed to load...")
			time.Sleep(10 * time.Second)
		} else {
			log.Warn("Could not find login fields, attempting to proceed anyway...")
		}
	}

	// Initialize visual searcher
	visualSearcher := search.NewVisualSearcher(br, timing, scroll, mouse)

	// TARGET SEQUENCE
	targets := []struct {
		Search   string
		Keywords []string
	}{
		{
			Search:   "Jensen Huang NVIDIA CEO",
			Keywords: []string{"Jensen", "Huang", "NVIDIA", "CEO"},
		},
		{
			Search:   "Amarnath PES Apple",
			Keywords: []string{"Amarnath", "PES", "Apple", "India"},
		},
		{
			Search:   "Santosh Mokashi PES",
			Keywords: []string{"Santhosh", "Santosh", "Mokashi", "PES", "University"},
		},
		{
			Search:   "Mahesh Awati PES",
			Keywords: []string{"Mahesh", "Awati", "PES", "University"},
		},
		{
			Search:   "Shreya Lingamallu PES",
			Keywords: []string{"Shreya", "Lingamallu", "PES", "University"},
		},
	}

	for i, t := range targets {
		log.Info("--- STEP %d: Target %s ---", i+1, t.Search)

		// Perform search
		if err := visualSearcher.SearchWithUI(ctx, t.Search); err != nil {
			log.Error("Search failed for %s: %v", t.Search, err)
			continue
		}
		time.Sleep(2 * time.Second)

		// Click profile
		log.Info("Clicking profile for %s...", t.Search)
		if err := visualSearcher.ClickProfile(ctx, t.Keywords); err != nil {
			log.Error("Failed to click profile for %s: %v", t.Search, err)
		}

		time.Sleep(3 * time.Second)
		log.Info("Target search %d/5 complete.", i+1)

		// Navigate back home for next iteration
		if i < len(targets)-1 {
			log.Info("Preparing for next target...")
			_ = br.Navigate(ctx, "https://www.linkedin.com/feed/")
			time.Sleep(1 * time.Second)
		}
	}

	log.Info("FULL DEMO COMPLETED! Browser will remain open for inspection.")

	// Keep open until interrupted
	select {
	case <-ctx.Done():
	case <-sigChan:
	}
}
