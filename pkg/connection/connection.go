package connection

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/linkedin-automation/pkg/browser"
	"github.com/linkedin-automation/pkg/config"
	"github.com/linkedin-automation/pkg/logger"
	"github.com/linkedin-automation/pkg/stealth"
	"github.com/linkedin-automation/pkg/storage"
)

type ConnectionManager struct {
	config     *config.RateLimitConfig
	msgConfig  *config.MessagingConfig
	browser    *browser.Browser
	storage    *storage.Storage
	timing     *stealth.TimingController
	log        *logger.Logger
	
	dailyCount  int
	hourlyCount int
	lastReset   time.Time
}

type ConnectionResult struct {
	ProfileURL string
	Name       string
	Success    bool
	Message    string
	Error      error
}

func New(cfg *config.RateLimitConfig, msgCfg *config.MessagingConfig, b *browser.Browser, s *storage.Storage, timing *stealth.TimingController) *ConnectionManager {
	return &ConnectionManager{
		config:    cfg,
		msgConfig: msgCfg,
		browser:   b,
		storage:   s,
		timing:    timing,
		log:       logger.WithComponent("connection"),
		lastReset: time.Now(),
	}
}

func (c *ConnectionManager) SendConnectionRequest(ctx context.Context, profileURL, note string) (*ConnectionResult, error) {
	c.log.Info("Sending connection request to: %s", profileURL)

	if !c.canSendConnection() {
		return &ConnectionResult{
			ProfileURL: profileURL,
			Success:    false,
			Message:    "Rate limit reached",
		}, nil
	}

	exists, err := c.storage.ConnectionExists(profileURL)
	if err != nil {
		c.log.Warn("Error checking existing connection: %v", err)
	}
	if exists {
		return &ConnectionResult{
			ProfileURL: profileURL,
			Success:    false,
			Message:    "Connection already exists or pending",
		}, nil
	}

	if err := c.browser.Navigate(ctx, profileURL); err != nil {
		return &ConnectionResult{
			ProfileURL: profileURL,
			Success:    false,
			Error:      err,
		}, err
	}

	if err := c.timing.SleepPageLoad(ctx); err != nil {
		return nil, err
	}

	name := c.extractProfileName(ctx)

	if err := c.clickConnectButton(ctx); err != nil {
		return &ConnectionResult{
			ProfileURL: profileURL,
			Name:       name,
			Success:    false,
			Message:    fmt.Sprintf("Failed to click connect: %v", err),
		}, nil
	}

	if err := c.timing.SleepAction(ctx); err != nil {
		return nil, err
	}

	if note != "" {
		if err := c.addConnectionNote(ctx, note); err != nil {
			c.log.Warn("Failed to add note, sending without note: %v", err)
		}
	}

	if err := c.confirmConnection(ctx); err != nil {
		return &ConnectionResult{
			ProfileURL: profileURL,
			Name:       name,
			Success:    false,
			Message:    fmt.Sprintf("Failed to confirm connection: %v", err),
		}, nil
	}

	conn := storage.Connection{
		ProfileURL:  profileURL,
		Name:        name,
		RequestedAt: time.Now(),
		Status:      "pending",
		Note:        note,
	}
	if err := c.storage.AddConnection(conn); err != nil {
		c.log.Warn("Failed to save connection: %v", err)
	}

	c.incrementCount()

	c.storage.UpdateTodayStats(func(stats *storage.DailyStats) {
		stats.ConnectionsSent++
	})

	c.log.Info("Connection request sent successfully to %s", name)
	return &ConnectionResult{
		ProfileURL: profileURL,
		Name:       name,
		Success:    true,
		Message:    "Connection request sent",
	}, nil
}

func (c *ConnectionManager) clickConnectButton(ctx context.Context) error {
	connectSelectors := []string{
		"button[aria-label*='Connect']",
		"button[aria-label*='connect']",
		".pv-s-profile-actions button:has-text('Connect')",
		"button.pvs-profile-actions__action:has-text('Connect')",
		"[data-control-name='connect']",
		"button:has-text('Connect')",
	}

	for _, selector := range connectSelectors {
		if c.browser.Exists(selector) {
			if err := c.browser.Click(ctx, selector); err == nil {
				return nil
			}
		}
	}

	moreSelectors := []string{
		"button[aria-label='More actions']",
		".pvs-overflow-actions-dropdown__trigger",
		"button:has-text('More')",
	}

	for _, moreSelector := range moreSelectors {
		if c.browser.Exists(moreSelector) {
			if err := c.browser.Click(ctx, moreSelector); err != nil {
				continue
			}

			if err := c.timing.SleepAction(ctx); err != nil {
				return err
			}

			dropdownConnectSelectors := []string{
				"div[data-control-name='connect']",
				".artdeco-dropdown__item:has-text('Connect')",
				"li button:has-text('Connect')",
			}

			for _, dropdownSelector := range dropdownConnectSelectors {
				if c.browser.Exists(dropdownSelector) {
					return c.browser.Click(ctx, dropdownSelector)
				}
			}
		}
	}

	return fmt.Errorf("connect button not found")
}

func (c *ConnectionManager) addConnectionNote(ctx context.Context, note string) error {
	addNoteSelectors := []string{
		"button[aria-label='Add a note']",
		"button:has-text('Add a note')",
		".artdeco-modal button:has-text('Add a note')",
	}

	for _, selector := range addNoteSelectors {
		if c.browser.Exists(selector) {
			if err := c.browser.Click(ctx, selector); err != nil {
				continue
			}

			if err := c.timing.SleepAction(ctx); err != nil {
				return err
			}

			break
		}
	}

	noteInputSelectors := []string{
		"textarea[name='message']",
		"textarea#custom-message",
		".connect-button-send-invite__custom-message",
		"textarea",
	}

	if len(note) > c.msgConfig.MaxMessageLength {
		note = note[:c.msgConfig.MaxMessageLength]
	}

	for _, selector := range noteInputSelectors {
		if c.browser.Exists(selector) {
			return c.browser.Type(ctx, selector, note)
		}
	}

	return fmt.Errorf("note input not found")
}

func (c *ConnectionManager) confirmConnection(ctx context.Context) error {
	sendSelectors := []string{
		"button[aria-label='Send now']",
		"button[aria-label='Send invitation']",
		"button:has-text('Send')",
		".artdeco-modal button.artdeco-button--primary",
	}

	for _, selector := range sendSelectors {
		if c.browser.Exists(selector) {
			return c.browser.Click(ctx, selector)
		}
	}

	return fmt.Errorf("send button not found")
}

func (c *ConnectionManager) extractProfileName(ctx context.Context) string {
	nameSelectors := []string{
		"h1.text-heading-xlarge",
		".pv-text-details__left-panel h1",
		"h1[data-anonymize='person-name']",
	}

	for _, selector := range nameSelectors {
		if c.browser.Exists(selector) {
			name, err := c.browser.GetText(ctx, selector)
			if err == nil && name != "" {
				return strings.TrimSpace(name)
			}
		}
	}

	return "Unknown"
}

func (c *ConnectionManager) canSendConnection() bool {
	c.checkAndResetCounts()

	if c.dailyCount >= c.config.DailyConnectionLimit {
		c.log.Warn("Daily connection limit reached (%d/%d)", c.dailyCount, c.config.DailyConnectionLimit)
		return false
	}

	if c.hourlyCount >= c.config.HourlyConnectionLimit {
		c.log.Warn("Hourly connection limit reached (%d/%d)", c.hourlyCount, c.config.HourlyConnectionLimit)
		return false
	}

	return true
}

func (c *ConnectionManager) checkAndResetCounts() {
	now := time.Now()

	if now.Day() != c.lastReset.Day() {
		c.dailyCount = 0
		c.hourlyCount = 0
		c.lastReset = now
		return
	}

	if now.Hour() != c.lastReset.Hour() {
		c.hourlyCount = 0
		c.lastReset = now
	}
}

func (c *ConnectionManager) incrementCount() {
	c.dailyCount++
	c.hourlyCount++
}

func (c *ConnectionManager) ProcessProfiles(ctx context.Context, profiles []storage.Profile, noteTemplate string) ([]ConnectionResult, error) {
	var results []ConnectionResult

	for _, profile := range profiles {
		if !c.canSendConnection() {
			c.log.Info("Rate limit reached, stopping batch processing")
			break
		}

		note := c.personalizeNote(noteTemplate, profile)

		result, err := c.SendConnectionRequest(ctx, profile.URL, note)
		if err != nil {
			c.log.Error("Error processing profile %s: %v", profile.URL, err)
		}

		if result != nil {
			results = append(results, *result)
		}

		if err := c.storage.MarkProfileProcessed(profile.URL); err != nil {
			c.log.Warn("Failed to mark profile as processed: %v", err)
		}

		if err := c.timing.SleepThink(ctx); err != nil {
			return results, err
		}
	}

	return results, nil
}

func (c *ConnectionManager) personalizeNote(template string, profile storage.Profile) string {
	note := template

	firstName := strings.Split(profile.Name, " ")[0]
	note = strings.ReplaceAll(note, "{{.FirstName}}", firstName)
	note = strings.ReplaceAll(note, "{{.Name}}", profile.Name)
	note = strings.ReplaceAll(note, "{{.Title}}", profile.Title)
	note = strings.ReplaceAll(note, "{{.Company}}", profile.Company)
	note = strings.ReplaceAll(note, "{{.Location}}", profile.Location)
	note = strings.ReplaceAll(note, "{{.Industry}}", profile.Industry)

	return note
}

func (c *ConnectionManager) GetRemainingQuota() (daily, hourly int) {
	c.checkAndResetCounts()
	daily = c.config.DailyConnectionLimit - c.dailyCount
	hourly = c.config.HourlyConnectionLimit - c.hourlyCount
	return
}

func (c *ConnectionManager) WithdrawConnection(ctx context.Context, profileURL string) error {
	if err := c.browser.Navigate(ctx, profileURL); err != nil {
		return err
	}

	if err := c.timing.SleepPageLoad(ctx); err != nil {
		return err
	}

	pendingSelectors := []string{
		"button[aria-label='Pending']",
		"button:has-text('Pending')",
	}

	for _, selector := range pendingSelectors {
		if c.browser.Exists(selector) {
			if err := c.browser.Click(ctx, selector); err != nil {
				continue
			}

			if err := c.timing.SleepAction(ctx); err != nil {
				return err
			}

			withdrawSelectors := []string{
				"button:has-text('Withdraw')",
				".artdeco-dropdown__item:has-text('Withdraw')",
			}

			for _, withdrawSelector := range withdrawSelectors {
				if c.browser.Exists(withdrawSelector) {
					return c.browser.Click(ctx, withdrawSelector)
				}
			}
		}
	}

	return fmt.Errorf("pending/withdraw button not found")
}
