package messaging

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

type Messenger struct {
	config     *config.MessagingConfig
	rateConfig *config.RateLimitConfig
	browser    *browser.Browser
	storage    *storage.Storage
	timing     *stealth.TimingController
	log        *logger.Logger
	
	dailyCount  int
	hourlyCount int
	lastReset   time.Time
}

type MessageResult struct {
	ProfileURL string
	Name       string
	Success    bool
	Message    string
	Error      error
}

func New(cfg *config.MessagingConfig, rateCfg *config.RateLimitConfig, b *browser.Browser, s *storage.Storage, timing *stealth.TimingController) *Messenger {
	return &Messenger{
		config:     cfg,
		rateConfig: rateCfg,
		browser:    b,
		storage:    s,
		timing:     timing,
		log:        logger.WithComponent("messaging"),
		lastReset:  time.Now(),
	}
}

func (m *Messenger) SendMessage(ctx context.Context, profileURL, message string) (*MessageResult, error) {
	m.log.Info("Sending message to: %s", profileURL)

	if !m.canSendMessage() {
		return &MessageResult{
			ProfileURL: profileURL,
			Success:    false,
			Message:    "Rate limit reached",
		}, nil
	}

	if err := m.browser.Navigate(ctx, profileURL); err != nil {
		return &MessageResult{
			ProfileURL: profileURL,
			Success:    false,
			Error:      err,
		}, err
	}

	if err := m.timing.SleepPageLoad(ctx); err != nil {
		return nil, err
	}

	name := m.extractProfileName(ctx)

	if err := m.clickMessageButton(ctx); err != nil {
		return &MessageResult{
			ProfileURL: profileURL,
			Name:       name,
			Success:    false,
			Message:    fmt.Sprintf("Failed to click message button: %v", err),
		}, nil
	}

	if err := m.timing.SleepAction(ctx); err != nil {
		return nil, err
	}

	if err := m.typeMessage(ctx, message); err != nil {
		return &MessageResult{
			ProfileURL: profileURL,
			Name:       name,
			Success:    false,
			Message:    fmt.Sprintf("Failed to type message: %v", err),
		}, nil
	}

	if err := m.timing.SleepThink(ctx); err != nil {
		return nil, err
	}

	if err := m.sendMessage(ctx); err != nil {
		return &MessageResult{
			ProfileURL: profileURL,
			Name:       name,
			Success:    false,
			Message:    fmt.Sprintf("Failed to send message: %v", err),
		}, nil
	}

	msg := storage.Message{
		ProfileURL:    profileURL,
		RecipientName: name,
		Content:       message,
		SentAt:        time.Now(),
		Type:          "direct",
		Status:        "sent",
	}
	if err := m.storage.AddMessage(msg); err != nil {
		m.log.Warn("Failed to save message: %v", err)
	}

	m.incrementCount()

	m.storage.UpdateTodayStats(func(stats *storage.DailyStats) {
		stats.MessagesSent++
	})

	m.log.Info("Message sent successfully to %s", name)
	return &MessageResult{
		ProfileURL: profileURL,
		Name:       name,
		Success:    true,
		Message:    "Message sent",
	}, nil
}

func (m *Messenger) clickMessageButton(ctx context.Context) error {
	messageSelectors := []string{
		"button[aria-label*='Message']",
		"button[aria-label*='message']",
		"button:has-text('Message')",
		".pv-s-profile-actions button:has-text('Message')",
		"[data-control-name='message']",
	}

	for _, selector := range messageSelectors {
		if m.browser.Exists(selector) {
			return m.browser.Click(ctx, selector)
		}
	}

	moreSelectors := []string{
		"button[aria-label='More actions']",
		".pvs-overflow-actions-dropdown__trigger",
	}

	for _, moreSelector := range moreSelectors {
		if m.browser.Exists(moreSelector) {
			if err := m.browser.Click(ctx, moreSelector); err != nil {
				continue
			}

			if err := m.timing.SleepAction(ctx); err != nil {
				return err
			}

			dropdownMessageSelectors := []string{
				"div[data-control-name='message']",
				".artdeco-dropdown__item:has-text('Message')",
			}

			for _, dropdownSelector := range dropdownMessageSelectors {
				if m.browser.Exists(dropdownSelector) {
					return m.browser.Click(ctx, dropdownSelector)
				}
			}
		}
	}

	return fmt.Errorf("message button not found")
}

func (m *Messenger) typeMessage(ctx context.Context, message string) error {
	messageInputSelectors := []string{
		".msg-form__contenteditable",
		"div[data-placeholder='Write a message…']",
		".msg-form__message-texteditor",
		"div[role='textbox']",
	}

	for _, selector := range messageInputSelectors {
		if m.browser.Exists(selector) {
			return m.browser.Type(ctx, selector, message)
		}
	}

	return fmt.Errorf("message input not found")
}

func (m *Messenger) sendMessage(ctx context.Context) error {
	sendSelectors := []string{
		"button.msg-form__send-button",
		"button[type='submit']",
		"button:has-text('Send')",
		".msg-form__send-button",
	}

	for _, selector := range sendSelectors {
		if m.browser.Exists(selector) {
			return m.browser.Click(ctx, selector)
		}
	}

	return fmt.Errorf("send button not found")
}

func (m *Messenger) extractProfileName(ctx context.Context) string {
	nameSelectors := []string{
		"h1.text-heading-xlarge",
		".pv-text-details__left-panel h1",
	}

	for _, selector := range nameSelectors {
		if m.browser.Exists(selector) {
			name, err := m.browser.GetText(ctx, selector)
			if err == nil && name != "" {
				return strings.TrimSpace(name)
			}
		}
	}

	return "Unknown"
}

func (m *Messenger) canSendMessage() bool {
	m.checkAndResetCounts()

	if m.dailyCount >= m.rateConfig.DailyMessageLimit {
		m.log.Warn("Daily message limit reached (%d/%d)", m.dailyCount, m.rateConfig.DailyMessageLimit)
		return false
	}

	if m.hourlyCount >= m.rateConfig.HourlyMessageLimit {
		m.log.Warn("Hourly message limit reached (%d/%d)", m.hourlyCount, m.rateConfig.HourlyMessageLimit)
		return false
	}

	return true
}

func (m *Messenger) checkAndResetCounts() {
	now := time.Now()

	if now.Day() != m.lastReset.Day() {
		m.dailyCount = 0
		m.hourlyCount = 0
		m.lastReset = now
		return
	}

	if now.Hour() != m.lastReset.Hour() {
		m.hourlyCount = 0
		m.lastReset = now
	}
}

func (m *Messenger) incrementCount() {
	m.dailyCount++
	m.hourlyCount++
}

func (m *Messenger) GetTemplate(name string) *config.MessageTemplate {
	for _, tmpl := range m.config.Templates {
		if tmpl.Name == name {
			return &tmpl
		}
	}
	return nil
}

func (m *Messenger) PersonalizeMessage(template string, data map[string]string) string {
	message := template

	for key, value := range data {
		placeholder := fmt.Sprintf("{{.%s}}", key)
		message = strings.ReplaceAll(message, placeholder, value)
	}

	return message
}

func (m *Messenger) SendFollowUpToNewConnections(ctx context.Context) ([]MessageResult, error) {
	connections, err := m.storage.LoadConnections()
	if err != nil {
		return nil, err
	}

	var results []MessageResult
	followUpTemplate := m.GetTemplate("follow_up")
	if followUpTemplate == nil {
		return nil, fmt.Errorf("follow_up template not found")
	}

	for _, conn := range connections {
		if conn.Status != "accepted" || conn.MessageSent {
			continue
		}

		if time.Since(conn.AcceptedAt) < m.config.FollowUpDelay {
			continue
		}

		if !m.canSendMessage() {
			m.log.Info("Rate limit reached, stopping follow-up processing")
			break
		}

		data := map[string]string{
			"FirstName": strings.Split(conn.Name, " ")[0],
			"Name":      conn.Name,
			"Company":   conn.Company,
			"Title":     conn.Title,
			"Industry":  "",
		}
		message := m.PersonalizeMessage(followUpTemplate.Body, data)

		result, err := m.SendMessage(ctx, conn.ProfileURL, message)
		if err != nil {
			m.log.Error("Error sending follow-up to %s: %v", conn.Name, err)
		}

		if result != nil && result.Success {
			conn.MessageSent = true
			conn.MessageSentAt = time.Now()
			if err := m.storage.AddConnection(conn); err != nil {
				m.log.Warn("Failed to update connection: %v", err)
			}
		}

		if result != nil {
			results = append(results, *result)
		}

		if err := m.timing.SleepThink(ctx); err != nil {
			return results, err
		}
	}

	return results, nil
}

func (m *Messenger) DetectNewConnections(ctx context.Context) ([]storage.Connection, error) {
	if err := m.browser.Navigate(ctx, "https://www.linkedin.com/mynetwork/invite-connect/connections/"); err != nil {
		return nil, err
	}

	if err := m.timing.SleepPageLoad(ctx); err != nil {
		return nil, err
	}

	existingConnections, err := m.storage.LoadConnections()
	if err != nil {
		return nil, err
	}

	pendingURLs := make(map[string]storage.Connection)
	for _, conn := range existingConnections {
		if conn.Status == "pending" {
			pendingURLs[conn.ProfileURL] = conn
		}
	}

	var newlyAccepted []storage.Connection

	connectionElements, err := m.browser.Elements(ctx, ".mn-connection-card")
	if err != nil {
		return nil, err
	}

	for _, elem := range connectionElements {
		linkAttr, err := elem.Attribute("href")
		if err != nil || linkAttr == nil {
			continue
		}

		profileURL := *linkAttr
		if conn, exists := pendingURLs[profileURL]; exists {
			conn.Status = "accepted"
			conn.AcceptedAt = time.Now()

			if err := m.storage.AddConnection(conn); err != nil {
				m.log.Warn("Failed to update connection: %v", err)
			}

			m.storage.UpdateTodayStats(func(stats *storage.DailyStats) {
				stats.ConnectionsAccepted++
			})

			newlyAccepted = append(newlyAccepted, conn)
		}
	}

	m.log.Info("Detected %d newly accepted connections", len(newlyAccepted))
	return newlyAccepted, nil
}

func (m *Messenger) GetRemainingQuota() (daily, hourly int) {
	m.checkAndResetCounts()
	daily = m.rateConfig.DailyMessageLimit - m.dailyCount
	hourly = m.rateConfig.HourlyMessageLimit - m.hourlyCount
	return
}
