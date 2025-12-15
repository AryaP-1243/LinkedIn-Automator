package stealth

import (
	"math/rand"
	"time"

	"github.com/linkedin-automation/pkg/config"
	"github.com/linkedin-automation/pkg/logger"
)

type FingerprintManager struct {
	config     *config.FingerprintConfig
	browserCfg *config.BrowserConfig
	log        *logger.Logger
	rand       *rand.Rand
}

type BrowserFingerprint struct {
	UserAgent      string
	ViewportWidth  int
	ViewportHeight int
	Timezone       string
	Language       string
	Platform       string
	WebGLVendor    string
	WebGLRenderer  string
	ScreenWidth    int
	ScreenHeight   int
	ColorDepth     int
	PixelRatio     float64
}

var defaultUserAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:120.0) Gecko/20100101 Firefox/120.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Safari/605.1.15",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1 Safari/605.1.15",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64; rv:121.0) Gecko/20100101 Firefox/121.0",
}

var commonTimezones = []string{
	"America/New_York",
	"America/Chicago",
	"America/Denver",
	"America/Los_Angeles",
	"America/Phoenix",
	"America/Toronto",
	"Europe/London",
	"Europe/Paris",
	"Europe/Berlin",
	"Asia/Tokyo",
	"Asia/Shanghai",
	"Australia/Sydney",
}

var commonResolutions = []struct {
	Width  int
	Height int
}{
	{1920, 1080},
	{1366, 768},
	{1536, 864},
	{1440, 900},
	{1280, 720},
	{2560, 1440},
	{1680, 1050},
	{1600, 900},
	{1280, 1024},
	{1280, 800},
}

var webGLConfigs = []struct {
	Vendor   string
	Renderer string
}{
	{"Google Inc. (NVIDIA)", "ANGLE (NVIDIA, NVIDIA GeForce GTX 1080 Direct3D11 vs_5_0 ps_5_0, D3D11)"},
	{"Google Inc. (NVIDIA)", "ANGLE (NVIDIA, NVIDIA GeForce RTX 3060 Direct3D11 vs_5_0 ps_5_0, D3D11)"},
	{"Google Inc. (AMD)", "ANGLE (AMD, AMD Radeon RX 580 Series Direct3D11 vs_5_0 ps_5_0, D3D11)"},
	{"Google Inc. (Intel)", "ANGLE (Intel, Intel(R) UHD Graphics 630 Direct3D11 vs_5_0 ps_5_0, D3D11)"},
	{"Apple Inc.", "Apple GPU"},
	{"Intel Inc.", "Intel Iris OpenGL Engine"},
}

func NewFingerprintManager(cfg *config.FingerprintConfig, browserCfg *config.BrowserConfig) *FingerprintManager {
	return &FingerprintManager{
		config:     cfg,
		browserCfg: browserCfg,
		log:        logger.WithComponent("fingerprint"),
		rand:       rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (f *FingerprintManager) Generate() *BrowserFingerprint {
	fp := &BrowserFingerprint{
		Language:   "en-US",
		Platform:   "Win32",
		ColorDepth: 24,
		PixelRatio: 1.0,
	}

	if f.config.RotateUserAgent {
		userAgents := f.browserCfg.UserAgents
		if len(userAgents) == 0 {
			userAgents = defaultUserAgents
		}
		fp.UserAgent = userAgents[f.rand.Intn(len(userAgents))]
		fp.Platform = f.detectPlatform(fp.UserAgent)
	} else if len(f.browserCfg.UserAgents) > 0 {
		fp.UserAgent = f.browserCfg.UserAgents[0]
		fp.Platform = f.detectPlatform(fp.UserAgent)
	} else {
		fp.UserAgent = defaultUserAgents[0]
	}

	if f.config.RandomizeViewport {
		res := commonResolutions[f.rand.Intn(len(commonResolutions))]
		fp.ScreenWidth = res.Width
		fp.ScreenHeight = res.Height

		fp.ViewportWidth = res.Width - f.rand.Intn(100)
		fp.ViewportHeight = res.Height - 100 - f.rand.Intn(50)
	} else {
		fp.ViewportWidth = f.browserCfg.ViewportWidth
		fp.ViewportHeight = f.browserCfg.ViewportHeight
		fp.ScreenWidth = f.browserCfg.ViewportWidth
		fp.ScreenHeight = f.browserCfg.ViewportHeight
	}

	if f.config.SpoofTimezone {
		fp.Timezone = commonTimezones[f.rand.Intn(len(commonTimezones))]
	} else {
		fp.Timezone = "America/New_York"
	}

	if f.config.SpoofLanguage {
		languages := []string{"en-US", "en-GB", "en-CA", "en-AU"}
		fp.Language = languages[f.rand.Intn(len(languages))]
	}

	webGL := webGLConfigs[f.rand.Intn(len(webGLConfigs))]
	fp.WebGLVendor = webGL.Vendor
	fp.WebGLRenderer = webGL.Renderer

	pixelRatios := []float64{1.0, 1.25, 1.5, 2.0}
	fp.PixelRatio = pixelRatios[f.rand.Intn(len(pixelRatios))]

	return fp
}

func (f *FingerprintManager) detectPlatform(userAgent string) string {
	switch {
	case contains(userAgent, "Windows"):
		return "Win32"
	case contains(userAgent, "Macintosh"):
		return "MacIntel"
	case contains(userAgent, "Linux"):
		return "Linux x86_64"
	default:
		return "Win32"
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func (f *FingerprintManager) GetStealthScripts() []string {
	scripts := []string{}

	if f.config.DisableAutomation {
		scripts = append(scripts, `
			Object.defineProperty(navigator, 'webdriver', {
				get: () => undefined
			});
			
			delete navigator.__proto__.webdriver;
			
			Object.defineProperty(navigator, 'plugins', {
				get: () => [1, 2, 3, 4, 5]
			});
			
			Object.defineProperty(navigator, 'languages', {
				get: () => ['en-US', 'en']
			});
			
			window.chrome = {
				runtime: {}
			};
			
			Object.defineProperty(navigator, 'permissions', {
				get: () => ({
					query: (parameters) => (
						parameters.name === 'notifications' ?
							Promise.resolve({ state: Notification.permission }) :
							Promise.resolve({ state: 'prompt' })
					)
				})
			});
		`)
	}

	return scripts
}

func (f *FingerprintManager) GetBrowserArgs() []string {
	args := []string{
		"--disable-blink-features=AutomationControlled",
		"--disable-infobars",
		"--disable-dev-shm-usage",
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-background-timer-throttling",
		"--disable-backgrounding-occluded-windows",
		"--disable-renderer-backgrounding",
	}

	if f.config.DisableAutomation {
		args = append(args,
			"--disable-extensions",
			"--disable-plugins-discovery",
		)
	}

	if f.browserCfg.DisableWebRTC {
		args = append(args,
			"--disable-webrtc",
			"--disable-webrtc-hw-encoding",
			"--disable-webrtc-hw-decoding",
		)
	}

	return args
}
