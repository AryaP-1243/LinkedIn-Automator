package browser

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"

	"github.com/linkedin-automation/pkg/config"
	"github.com/linkedin-automation/pkg/logger"
	"github.com/linkedin-automation/pkg/stealth"
)

type Browser struct {
	config      *config.BrowserConfig
	rod         *rod.Browser
	page        *rod.Page
	log         *logger.Logger
	fingerprint *stealth.FingerprintManager
	mouse       *stealth.MouseController
	timing      *stealth.TimingController
	scroll      *stealth.ScrollController
	typing      *stealth.TypingController
}

type Options struct {
	Config      *config.Config
	Fingerprint *stealth.FingerprintManager
	Mouse       *stealth.MouseController
	Timing      *stealth.TimingController
	Scroll      *stealth.ScrollController
	Typing      *stealth.TypingController
}

func New(opts Options) *Browser {
	return &Browser{
		config:      &opts.Config.Browser,
		log:         logger.WithComponent("browser"),
		fingerprint: opts.Fingerprint,
		mouse:       opts.Mouse,
		timing:      opts.Timing,
		scroll:      opts.Scroll,
		typing:      opts.Typing,
	}
}

func (b *Browser) SetTypingController(tc *stealth.TypingController) {
	b.typing = tc
}

func (b *Browser) Launch(ctx context.Context) error {
	b.log.Info("Launching browser...")

	if err := os.MkdirAll(b.config.UserDataDir, 0755); err != nil {
		return fmt.Errorf("failed to create user data directory: %w", err)
	}

	// Skip fingerprint viewport/user-agent to prevent blur on Retina displays
	_ = b.fingerprint.Generate() // Keep fingerprint for stealth scripts only
	args := b.fingerprint.GetBrowserArgs()

	l := launcher.New().
		Bin("/Applications/Google Chrome.app/Contents/MacOS/Google Chrome").
		Headless(false). // FORCE non-headless to ensure visible rendering
		UserDataDir(b.config.UserDataDir).
		Set("no-first-run").
		Set("no-default-browser-check").
		Set("window-position", "0,0")
		// REMOVED: about:blank - let Chrome open normally

	// Apply fullscreen if enabled
	if b.config.Fullscreen {
		l = l.Set("start-fullscreen")
		b.log.Debug("Applied fullscreen flag")
	}

	// REMOVED: Zoom level setting - causes blur on Retina displays
	// Chrome will use native resolution for crisp rendering

	// REMOVED: Browser args loop was causing blur on Retina displays
	// All fingerprint flags disabled for clean rendering
	_ = args // suppress unused warning

	url, err := l.Launch()
	if err != nil {
		return fmt.Errorf("failed to launch browser: %w", err)
	}

	browser := rod.New().ControlURL(url)
	if err := browser.Connect(); err != nil {
		return fmt.Errorf("failed to connect to browser: %w", err)
	}

	b.rod = browser

	// Wait for Chrome to initialize its default page using a ticker
	// We retry for up to 15 seconds to find the existing visible tab
	var page *rod.Page
	deadline := time.Now().Add(15 * time.Second)

	for time.Now().Before(deadline) {
		pages, err := browser.Pages()
		if err != nil {
			b.log.Warn("Failed to get pages: %v", err)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		if len(pages) > 0 {
			b.log.Info("Found %d existing pages, selecting the first one", len(pages))
			page = pages[0]

			// Log page info for debugging
			info, _ := page.Info()
			b.log.Info("Attached to page: URL=%s, Title=%s", info.URL, info.Title)
			break
		}

		b.log.Debug("No pages found yet, waiting...")
		time.Sleep(500 * time.Millisecond)
	}

	if page == nil {
		b.log.Info("No existing pages found after timeout, creating new page")
		// Create a page with a real URL instead of about:blank
		page, err = browser.Page(proto.TargetCreateTarget{URL: "https://www.google.com"})
		if err != nil {
			return fmt.Errorf("failed to create page: %w", err)
		}
	}

	// Wait for the page to be ready
	if err := page.WaitLoad(); err != nil {
		b.log.Warn("Page load wait failed: %v", err)
	}

	// Skip SetViewport - it causes blur on Retina/HiDPI displays
	// Chrome will use native resolution instead

	for _, script := range b.fingerprint.GetStealthScripts() {
		if _, err := page.Evaluate(rod.Eval(script)); err != nil {
			b.log.Warn("Failed to inject stealth script: %v", err)
		}
	}

	// NOTE: Cursor overlay is injected after navigation via injectVisibleCursor()
	// because document.body doesn't exist on about:blank

	b.page = page
	b.log.Info("Browser launched successfully with fingerprint applied")

	return nil
}

// injectVisibleCursor injects the visible red cursor overlay into the current page
func (b *Browser) injectVisibleCursor() {
	cursorScript := `
		(function() {
			if (document.getElementById('visible-cursor')) return; // Already exists
			var cursor = document.createElement('div');
			cursor.id = 'visible-cursor';
			cursor.style.cssText = 'position: fixed; width: 20px; height: 20px; background: rgba(255, 0, 0, 0.8); border-radius: 50%; pointer-events: none; z-index: 999999; transform: translate(-50%, -50%); box-shadow: 0 0 10px rgba(255, 0, 0, 0.5); transition: all 0.05s ease-out;';
			cursor.style.left = '0px';
			cursor.style.top = '0px';
			document.body.appendChild(cursor);
			window.__visibleCursor = cursor;
			window.moveCursor = function(x, y) {
				if (window.__visibleCursor) {
					window.__visibleCursor.style.left = x + 'px';
					window.__visibleCursor.style.top = y + 'px';
				}
			};
			window.clickEffect = function() {
				if (window.__visibleCursor) {
					window.__visibleCursor.style.transform = 'translate(-50%, -50%) scale(0.7)';
					window.__visibleCursor.style.background = 'rgba(255, 100, 100, 1)';
					setTimeout(function() {
						window.__visibleCursor.style.transform = 'translate(-50%, -50%) scale(1)';
						window.__visibleCursor.style.background = 'rgba(255, 0, 0, 0.8)';
					}, 150);
				}
			};
		})();
	`
	if _, err := b.page.Evaluate(rod.Eval(cursorScript)); err != nil {
		b.log.Debug("Failed to re-inject cursor: %v", err)
	}
}

func (b *Browser) Navigate(ctx context.Context, url string) error {
	b.log.Debug("Navigating to %s", url)

	if err := b.page.Navigate(url); err != nil {
		return fmt.Errorf("failed to navigate to %s: %w", url, err)
	}

	if err := b.page.WaitLoad(); err != nil {
		return fmt.Errorf("page load timeout for %s: %w", url, err)
	}

	// Re-inject visible cursor after page load
	b.injectVisibleCursor()

	if err := b.timing.SleepPageLoad(ctx); err != nil {
		return err
	}

	return nil
}

func (b *Browser) WaitForElement(ctx context.Context, selector string, timeout time.Duration) (*rod.Element, error) {
	b.log.Debug("Waiting for element: %s", selector)

	element, err := b.page.Timeout(timeout).Element(selector)
	if err != nil {
		return nil, fmt.Errorf("element not found: %s: %w", selector, err)
	}

	return element, nil
}

func (b *Browser) Click(ctx context.Context, selector string) error {
	element, err := b.WaitForElement(ctx, selector, 30*time.Second)
	if err != nil {
		return err
	}
	element = element.Context(ctx)

	box, err := element.Shape()
	if err != nil {
		return fmt.Errorf("failed to get element shape: %w", err)
	}

	if box == nil || len(box.Quads) == 0 {
		return fmt.Errorf("element has no visible shape")
	}

	quad := box.Quads[0]
	centerX := (quad[0] + quad[2] + quad[4] + quad[6]) / 4
	centerY := (quad[1] + quad[3] + quad[5] + quad[7]) / 4

	startX, startY := 0.0, 0.0

	path := b.mouse.GeneratePath(
		stealth.Point{X: startX, Y: startY},
		stealth.Point{X: centerX, Y: centerY},
	)

	// Move both DevTools mouse AND visible cursor along the path
	for _, point := range path {
		if err := b.page.Mouse.MoveTo(proto.Point{X: point.X, Y: point.Y}); err != nil {
			return fmt.Errorf("mouse move failed: %w", err)
		}
		// Move the visible cursor overlay
		_, _ = b.page.Eval(`(x, y) => { if(window.moveCursor) window.moveCursor(x, y); }`, point.X, point.Y)

		// Faster but still smooth movement
		time.Sleep(5 * time.Millisecond)
	}

	// Brief hover before clicking
	time.Sleep(100 * time.Millisecond)

	if err := b.timing.SleepAction(ctx); err != nil {
		return err
	}

	// Show click effect on visible cursor
	_, _ = b.page.Eval(`() => { if(window.clickEffect) window.clickEffect(); }`)

	if err := b.page.Mouse.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return fmt.Errorf("click failed: %w", err)
	}

	b.log.Debug("Clicked element: %s", selector)
	return nil
}

func (b *Browser) Type(ctx context.Context, selector, text string) error {
	element, err := b.WaitForElement(ctx, selector, 30*time.Second)
	if err != nil {
		return err
	}
	element = element.Context(ctx)

	if err := b.Click(ctx, selector); err != nil {
		return err
	}

	if err := b.timing.SleepAction(ctx); err != nil {
		return err
	}

	typeFn := func(char rune) error {
		return element.Input(string(char))
	}

	backspaceFn := func() error {
		return element.Type(input.Backspace)
	}

	if err := b.typing.ExecuteTyping(ctx, typeFn, backspaceFn, text); err != nil {
		return fmt.Errorf("typing failed: %w", err)
	}

	b.log.Debug("Typed text into element: %s", selector)
	return nil
}
func (b *Browser) Scroll(ctx context.Context, deltaY int) error {
	actions := b.scroll.GenerateScrollSequence(deltaY)

	// Move mouse to center of page to ensure we're over scrollable content
	_, _ = b.page.Eval(`() => { }`) // Ensure page is ready
	// Use slightly offset center to avoid some center-of-screen popups
	if err := b.page.Mouse.MoveTo(proto.Point{X: 700, Y: 450}); err != nil {
		return fmt.Errorf("mouse move for scroll failed: %w", err)
	}

	// Wrapped scroll function to also move visible cursor
	wrappedScrollFn := func(delta int) error {
		// Perform aggressive brute-force scroll
		scrollScript := `
			(delta) => {
				// 1. Scroll window/body
				window.scrollBy(0, delta);
				if (document.documentElement) document.documentElement.scrollBy(0, delta);
				if (document.body) document.body.scrollBy(0, delta);

				// 2. Brute-force find all scrollable elements and scroll them
				// This handles LinkedIn's nested container layouts
				var allElements = document.querySelectorAll('*');
				for (var i = 0; i < allElements.length; i++) {
					var el = allElements[i];
					var style = window.getComputedStyle(el);
					var overflowY = style.getPropertyValue('overflow-y');
					if ((overflowY === 'auto' || overflowY === 'scroll') && el.scrollHeight > el.clientHeight) {
						el.scrollBy(0, delta);
					}
				}
				return true;
			}
		`
		_, err := b.page.Evaluate(rod.Eval(scrollScript, delta).ByUser())
		if err != nil {
			b.log.Debug("JS scroll failed, using fallback mouse scroll: %v", err)
			b.page.Mouse.Scroll(0, float64(delta), 1)
		}

		// Move the visible cursor overlay slightly - keep it near center
		// Instead of proportional movement which can drift, keep it within a small jitter range
		// to show it's "active" but not walking off screen
		_, errEval := b.page.Evaluate(rod.Eval(`(d) => { 
			if(window.moveCursor) {
				var jitter = Math.sin(Date.now() / 100) * 2;
				window.moveCursor(window.innerWidth/2 + jitter, window.innerHeight/2 + jitter);
			}
		}`, delta).ByUser())
		if errEval != nil {
			b.log.Debug("Failed to move visible cursor during scroll: %v", errEval)
		}

		// Slight pause for smooth visual effect
		time.Sleep(10 * time.Millisecond)
		return nil
	}

	return b.scroll.ExecuteScroll(ctx, wrappedScrollFn, actions)
}

func (b *Browser) GetText(ctx context.Context, selector string) (string, error) {
	element, err := b.WaitForElement(ctx, selector, 30*time.Second)
	if err != nil {
		return "", err
	}

	text, err := element.Text()
	if err != nil {
		return "", fmt.Errorf("failed to get text: %w", err)
	}

	return text, nil
}

func (b *Browser) GetAttribute(ctx context.Context, selector, attr string) (string, error) {
	element, err := b.WaitForElement(ctx, selector, 30*time.Second)
	if err != nil {
		return "", err
	}

	value, err := element.Attribute(attr)
	if err != nil {
		return "", fmt.Errorf("failed to get attribute: %w", err)
	}

	if value == nil {
		return "", nil
	}

	return *value, nil
}

func (b *Browser) Elements(ctx context.Context, selector string) ([]*rod.Element, error) {
	elements, err := b.page.Elements(selector)
	if err != nil {
		return nil, fmt.Errorf("failed to find elements: %w", err)
	}

	return elements, nil
}

func (b *Browser) Exists(selector string) bool {
	_, err := b.page.Timeout(2 * time.Second).Element(selector)
	return err == nil
}

func (b *Browser) Screenshot(filename string) error {
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return err
	}

	data, err := b.page.Screenshot(true, nil)
	if err != nil {
		return fmt.Errorf("screenshot failed: %w", err)
	}

	return os.WriteFile(filename, data, 0644)
}

func (b *Browser) GetCookies() ([]*proto.NetworkCookie, error) {
	cookies, err := b.page.Cookies(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get cookies: %w", err)
	}

	return cookies, nil
}

func (b *Browser) SetCookies(cookies []*proto.NetworkCookie) error {
	for _, cookie := range cookies {
		if err := b.page.SetCookies([]*proto.NetworkCookieParam{{
			Name:     cookie.Name,
			Value:    cookie.Value,
			Domain:   cookie.Domain,
			Path:     cookie.Path,
			Expires:  proto.TimeSinceEpoch(cookie.Expires),
			HTTPOnly: cookie.HTTPOnly,
			Secure:   cookie.Secure,
		}}); err != nil {
			return fmt.Errorf("failed to set cookie %s: %w", cookie.Name, err)
		}
	}
	return nil
}

func (b *Browser) CurrentURL() string {
	info, err := b.page.Info()
	if err != nil {
		return ""
	}
	return info.URL
}

func (b *Browser) Page() *rod.Page {
	return b.page
}

func (b *Browser) Close() error {
	if b.rod != nil {
		return b.rod.Close()
	}
	return nil
}
