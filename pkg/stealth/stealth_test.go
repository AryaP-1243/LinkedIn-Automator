package stealth

import (
	"testing"
	"time"

	"github.com/linkedin-automation/pkg/config"
)

func TestMousePathGeneration(t *testing.T) {
	cfg := &config.MouseMovementConfig{
		Enabled:          true,
		MinSpeed:         0.5,
		MaxSpeed:         2.0,
		OvershootEnabled: true,
		MicroMovements:   true,
		BezierComplexity: 3,
	}

	mouse := NewMouseController(cfg)

	tests := []struct {
		name   string
		startX float64
		startY float64
		endX   float64
		endY   float64
	}{
		{"Short distance", 0, 0, 100, 100},
		{"Long distance", 0, 0, 1000, 800},
		{"Horizontal", 0, 0, 500, 0},
		{"Vertical", 0, 0, 0, 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := Point{X: tt.startX, Y: tt.startY}
			end := Point{X: tt.endX, Y: tt.endY}

			path := mouse.GeneratePath(start, end)

			if len(path) == 0 {
				t.Error("Expected non-empty path")
			}

			// First point should be near start (allowing for micro-movements jitter)
			tolerance := 5.0
			dx := path[0].X - tt.startX
			dy := path[0].Y - tt.startY
			if dx*dx+dy*dy > tolerance*tolerance {
				t.Errorf("First point should be near (%f, %f), got (%f, %f)",
					tt.startX, tt.startY, path[0].X, path[0].Y)
			}

			// Path should have reasonable number of points
			if len(path) < 2 {
				t.Error("Path should have at least 2 points")
			}
		})
	}
}

func TestTimingDelays(t *testing.T) {
	cfg := &config.TimingConfig{
		MinActionDelay: 100 * time.Millisecond,
		MaxActionDelay: 500 * time.Millisecond,
		MinThinkTime:   200 * time.Millisecond,
		MaxThinkTime:   1000 * time.Millisecond,
		PageLoadWait:   1000 * time.Millisecond,
		HumanVariation: 0.3,
	}

	timing := NewTimingController(cfg)

	t.Run("Action delay within range", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			delay := timing.ActionDelay()
			if delay < cfg.MinActionDelay || delay > cfg.MaxActionDelay*2 {
				t.Errorf("Action delay %v out of reasonable range [%v, %v]",
					delay, cfg.MinActionDelay, cfg.MaxActionDelay*2)
			}
		}
	})

	t.Run("Think time within range", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			delay := timing.ThinkDelay()
			if delay < cfg.MinThinkTime || delay > cfg.MaxThinkTime*2 {
				t.Errorf("Think time %v out of reasonable range [%v, %v]",
					delay, cfg.MinThinkTime, cfg.MaxThinkTime*2)
			}
		}
	})

	t.Run("Random delay", func(t *testing.T) {
		min := 100 * time.Millisecond
		max := 500 * time.Millisecond

		for i := 0; i < 100; i++ {
			delay := timing.RandomDelay(min, max)
			if delay < min || delay > max*2 {
				t.Errorf("Random delay %v out of reasonable range [%v, %v]",
					delay, min, max*2)
			}
		}
	})
}

func TestTypingKeystrokes(t *testing.T) {
	cfg := &config.TypingConfig{
		Enabled:          true,
		MinKeyDelay:      50 * time.Millisecond,
		MaxKeyDelay:      150 * time.Millisecond,
		TypoChance:       0.1,
		CorrectionDelay:  300 * time.Millisecond,
		ThinkPauseChance: 0.05,
	}

	typing := NewTypingController(cfg)

	t.Run("Generate keystrokes", func(t *testing.T) {
		text := "Hello, World!"
		keystrokes := typing.GenerateKeystrokes(text)

		if len(keystrokes) == 0 {
			t.Error("Expected non-empty keystrokes array")
		}

		// Should have at least as many keystrokes as characters
		if len(keystrokes) < len(text) {
			t.Errorf("Expected at least %d keystrokes, got %d", len(text), len(keystrokes))
		}

		// Check that delays are reasonable
		for i, ks := range keystrokes {
			if ks.Delay < 0 {
				t.Errorf("Keystroke %d has negative delay: %v", i, ks.Delay)
			}
		}
	})

	t.Run("Typing duration", func(t *testing.T) {
		text := "Test"
		duration := typing.TypingDuration(text)

		if duration <= 0 {
			t.Error("Typing duration should be positive")
		}

		// Should be at least min delay per character
		minExpected := cfg.MinKeyDelay * time.Duration(len(text))
		if duration < minExpected {
			t.Errorf("Duration %v should be at least %v", duration, minExpected)
		}
	})
}

func TestScrollController(t *testing.T) {
	timingCfg := &config.TimingConfig{
		MinActionDelay: 50 * time.Millisecond,
		MaxActionDelay: 100 * time.Millisecond,
	}
	timing := NewTimingController(timingCfg)

	cfg := &config.ScrollingConfig{
		Enabled:          true,
		MinScrollSpeed:   50,
		MaxScrollSpeed:   200,
		ScrollBackChance: 0.1,
		PauseChance:      0.15,
		SmoothScrolling:  true,
	}

	scroll := NewScrollController(cfg, timing)

	if scroll == nil {
		t.Fatal("ScrollController should not be nil")
	}

	// Test that controller was created successfully
	if scroll.config != cfg {
		t.Error("ScrollController config not set correctly")
	}
}

func TestFingerprintManager(t *testing.T) {
	browserCfg := &config.BrowserConfig{
		UserAgents: []string{
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120.0.0.0",
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) Chrome/120.0.0.0",
		},
		ViewportWidth:  1280,
		ViewportHeight: 800,
	}

	cfg := &config.FingerprintConfig{
		RotateUserAgent:   true,
		RandomizeViewport: true,
		DisableAutomation: true,
		SpoofTimezone:     true,
		SpoofLanguage:     false,
	}

	fingerprint := NewFingerprintManager(cfg, browserCfg)

	t.Run("Generate fingerprint", func(t *testing.T) {
		fp := fingerprint.Generate()

		if fp == nil {
			t.Fatal("Fingerprint should not be nil")
		}

		if fp.UserAgent == "" {
			t.Error("User agent should not be empty")
		}

		if fp.ViewportWidth <= 0 || fp.ViewportHeight <= 0 {
			t.Error("Viewport dimensions should be positive")
		}

		if fp.Timezone == "" {
			t.Error("Timezone should not be empty")
		}
	})

	t.Run("Browser args", func(t *testing.T) {
		args := fingerprint.GetBrowserArgs()

		if len(args) == 0 {
			t.Error("Browser args should not be empty")
		}

		// Should contain automation disabling flag
		found := false
		for _, arg := range args {
			if arg == "--disable-blink-features=AutomationControlled" {
				found = true
				break
			}
		}

		if !found {
			t.Error("Should contain automation disabling flag")
		}
	})
}

func TestMouseMovementDuration(t *testing.T) {
	cfg := &config.MouseMovementConfig{
		Enabled:          true,
		MinSpeed:         0.5,
		MaxSpeed:         2.0,
		OvershootEnabled: false,
		MicroMovements:   false,
		BezierComplexity: 2,
	}

	mouse := NewMouseController(cfg)

	start := Point{X: 0, Y: 0}
	end := Point{X: 1000, Y: 800}

	path := mouse.GeneratePath(start, end)
	duration := mouse.GetMovementDuration(path)

	if duration <= 0 {
		t.Error("Movement duration should be positive")
	}

	// Duration should be reasonable (not too fast or slow)
	if duration < 100*time.Millisecond || duration > 10*time.Second {
		t.Errorf("Movement duration %v seems unreasonable", duration)
	}
}

func TestHoverPath(t *testing.T) {
	cfg := &config.MouseMovementConfig{
		Enabled:          true,
		MinSpeed:         0.5,
		MaxSpeed:         2.0,
		OvershootEnabled: false,
		MicroMovements:   false,
		BezierComplexity: 2,
	}

	mouse := NewMouseController(cfg)

	center := Point{X: 500, Y: 500}
	duration := 2 * time.Second

	path := mouse.GenerateHoverPath(center, duration)

	if len(path) == 0 {
		t.Error("Hover path should not be empty")
	}

	// All points should be near the center
	for i, p := range path {
		dx := p.X - center.X
		dy := p.Y - center.Y
		distance := dx*dx + dy*dy

		if distance > 100*100 { // Within 100 pixels
			t.Errorf("Point %d is too far from center: (%f, %f)", i, p.X, p.Y)
		}
	}
}
