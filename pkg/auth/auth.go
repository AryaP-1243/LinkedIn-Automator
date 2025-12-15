package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-rod/rod/lib/proto"

	"github.com/linkedin-automation/pkg/browser"
	"github.com/linkedin-automation/pkg/config"
	"github.com/linkedin-automation/pkg/logger"
	"github.com/linkedin-automation/pkg/stealth"
	"github.com/linkedin-automation/pkg/storage"
)

const (
	linkedInLoginURL = "https://www.linkedin.com/login"
	linkedInFeedURL  = "https://www.linkedin.com/feed/"
	linkedInHomeURL  = "https://www.linkedin.com/"
)

type Authenticator struct {
	config  *config.LinkedInConfig
	browser *browser.Browser
	storage *storage.Storage
	timing  *stealth.TimingController
	log     *logger.Logger
}

type AuthResult struct {
	Success     bool
	Message     string
	SessionData *storage.Session
	Error       error
}

func New(cfg *config.LinkedInConfig, b *browser.Browser, s *storage.Storage, timing *stealth.TimingController) *Authenticator {
	return &Authenticator{
		config:  cfg,
		browser: b,
		storage: s,
		timing:  timing,
		log:     logger.WithComponent("auth"),
	}
}

func (a *Authenticator) Login(ctx context.Context) (*AuthResult, error) {
	a.log.Info("Starting LinkedIn login process...")

	session, err := a.storage.LoadSession()
	if err != nil {
		a.log.Warn("Failed to load existing session: %v", err)
	}

	if session != nil && session.IsValid {
		a.log.Info("Found existing session, attempting to restore...")
		if restored, err := a.restoreSession(ctx, session); err == nil && restored {
			a.log.Info("Session restored successfully")
			return &AuthResult{
				Success:     true,
				Message:     "Session restored",
				SessionData: session,
			}, nil
		}
		a.log.Info("Failed to restore session, proceeding with fresh login")
	}

	return a.performLogin(ctx)
}

func (a *Authenticator) restoreSession(ctx context.Context, session *storage.Session) (bool, error) {
	cookies := make([]*proto.NetworkCookie, len(session.Cookies))
	for i, c := range session.Cookies {
		cookies[i] = &proto.NetworkCookie{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			Expires:  proto.TimeSinceEpoch(c.Expires.Unix()),
			HTTPOnly: c.HTTPOnly,
			Secure:   c.Secure,
		}
	}

	if err := a.browser.SetCookies(cookies); err != nil {
		return false, fmt.Errorf("failed to set cookies: %w", err)
	}

	if err := a.browser.Navigate(ctx, linkedInFeedURL); err != nil {
		return false, fmt.Errorf("failed to navigate: %w", err)
	}

	if err := a.timing.SleepPageLoad(ctx); err != nil {
		return false, err
	}

	if a.isLoggedIn() {
		return true, nil
	}

	return false, nil
}

func (a *Authenticator) performLogin(ctx context.Context) (*AuthResult, error) {
	a.log.Info("Navigating to login page...")

	if err := a.browser.Navigate(ctx, linkedInLoginURL); err != nil {
		return &AuthResult{Success: false, Error: err}, err
	}

	if err := a.timing.SleepPageLoad(ctx); err != nil {
		return nil, err
	}

	if checkpoint := a.detectSecurityCheckpoint(); checkpoint != "" {
		a.log.Warn("Security checkpoint detected: %s", checkpoint)
		return &AuthResult{
			Success: false,
			Message: fmt.Sprintf("Security checkpoint detected: %s", checkpoint),
		}, nil
	}

	a.log.Info("Entering email...")
	if err := a.browser.Type(ctx, "#username", a.config.Email); err != nil {
		return &AuthResult{Success: false, Error: err}, err
	}

	if err := a.timing.SleepThink(ctx); err != nil {
		return nil, err
	}

	a.log.Info("Entering password...")
	if err := a.browser.Type(ctx, "#password", a.config.Password); err != nil {
		return &AuthResult{Success: false, Error: err}, err
	}

	if err := a.timing.SleepAction(ctx); err != nil {
		return nil, err
	}

	a.log.Info("Clicking sign in button...")
	if err := a.browser.Click(ctx, "button[type='submit']"); err != nil {
		return &AuthResult{Success: false, Error: err}, err
	}

	if err := a.timing.SleepPageLoad(ctx); err != nil {
		return nil, err
	}

	if err := a.waitForLoginResult(ctx); err != nil {
		return &AuthResult{Success: false, Error: err}, err
	}

	if !a.isLoggedIn() {
		errorMsg := a.getLoginError()
		if errorMsg != "" {
			return &AuthResult{
				Success: false,
				Message: fmt.Sprintf("Login failed: %s", errorMsg),
			}, nil
		}

		if checkpoint := a.detectSecurityCheckpoint(); checkpoint != "" {
			return &AuthResult{
				Success: false,
				Message: fmt.Sprintf("Security checkpoint: %s", checkpoint),
			}, nil
		}

		return &AuthResult{
			Success: false,
			Message: "Login failed for unknown reason",
		}, nil
	}

	session, err := a.saveSession()
	if err != nil {
		a.log.Warn("Failed to save session: %v", err)
	}

	a.log.Info("Login successful!")
	return &AuthResult{
		Success:     true,
		Message:     "Login successful",
		SessionData: session,
	}, nil
}

func (a *Authenticator) waitForLoginResult(ctx context.Context) error {
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("login timeout")
		case <-ticker.C:
			if a.isLoggedIn() {
				return nil
			}
			if a.getLoginError() != "" {
				return nil
			}
			if a.detectSecurityCheckpoint() != "" {
				return nil
			}
		}
	}
}

func (a *Authenticator) isLoggedIn() bool {
	currentURL := a.browser.CurrentURL()

	if strings.Contains(currentURL, "/feed") ||
		strings.Contains(currentURL, "/mynetwork") ||
		strings.Contains(currentURL, "/messaging") {
		return true
	}

	if a.browser.Exists(".global-nav__me-photo") ||
		a.browser.Exists(".feed-identity-module") ||
		a.browser.Exists("[data-control-name='identity_welcome_message']") {
		return true
	}

	return false
}

func (a *Authenticator) getLoginError() string {
	errorSelectors := []string{
		"#error-for-username",
		"#error-for-password",
		".form__label--error",
		"[data-test='form-error']",
		".alert-content",
	}

	for _, selector := range errorSelectors {
		if a.browser.Exists(selector) {
			text, err := a.browser.GetText(context.Background(), selector)
			if err == nil && text != "" {
				return strings.TrimSpace(text)
			}
		}
	}

	return ""
}

func (a *Authenticator) detectSecurityCheckpoint() string {
	checkpoints := map[string]string{
		"input#captcha":                          "captcha",
		"[data-test='challenge-form']":           "security_challenge",
		"input[name='pin']":                      "pin_verification",
		".challenge-dialog":                      "challenge_dialog",
		"[data-test='email-or-phone-challenge']": "email_phone_verification",
		"#two-step-challenge":                    "two_factor_auth",
	}

	for selector, checkpointType := range checkpoints {
		if a.browser.Exists(selector) {
			return checkpointType
		}
	}

	currentURL := a.browser.CurrentURL()
	if strings.Contains(currentURL, "checkpoint") ||
		strings.Contains(currentURL, "challenge") {
		return "unknown_checkpoint"
	}

	return ""
}

func (a *Authenticator) saveSession() (*storage.Session, error) {
	cookies, err := a.browser.GetCookies()
	if err != nil {
		return nil, fmt.Errorf("failed to get cookies: %w", err)
	}

	cookieData := make([]storage.CookieData, len(cookies))
	for i, c := range cookies {
		cookieData[i] = storage.CookieData{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			Expires:  time.Unix(int64(c.Expires), 0),
			HTTPOnly: c.HTTPOnly,
			Secure:   c.Secure,
		}
	}

	session := &storage.Session{
		Cookies:   cookieData,
		LastLogin: time.Now(),
		Email:     a.config.Email,
		IsValid:   true,
	}

	if err := a.storage.SaveSession(session); err != nil {
		return nil, err
	}

	return session, nil
}

func (a *Authenticator) Logout(ctx context.Context) error {
	a.log.Info("Logging out...")

	if err := a.browser.Navigate(ctx, "https://www.linkedin.com/m/logout/"); err != nil {
		return fmt.Errorf("failed to navigate to logout: %w", err)
	}

	session := &storage.Session{IsValid: false}
	if err := a.storage.SaveSession(session); err != nil {
		a.log.Warn("Failed to invalidate session: %v", err)
	}

	return nil
}

func (a *Authenticator) IsSessionValid(ctx context.Context) bool {
	if err := a.browser.Navigate(ctx, linkedInFeedURL); err != nil {
		return false
	}

	a.timing.SleepPageLoad(ctx)

	return a.isLoggedIn()
}
