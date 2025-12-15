// Package messaging handles LinkedIn messaging functionality.
// It detects newly accepted connections and sends follow-up messages.
package messaging

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/go-rod/rod"
	"github.com/nikshitha/linkedin-automation-poc/config"
	"github.com/nikshitha/linkedin-automation-poc/logger"
	"github.com/nikshitha/linkedin-automation-poc/stealth"
	"github.com/nikshitha/linkedin-automation-poc/storage"
)

// LinkedIn messaging URLs
const (
	LinkedInMessagingURL    = "https://www.linkedin.com/messaging/"
	LinkedInConnectionsURL  = "https://www.linkedin.com/mynetwork/invite-connect/connections/"
	LinkedInInvitationsURL  = "https://www.linkedin.com/mynetwork/invitation-manager/"
)

// MessagingManager handles messaging operations
type MessagingManager struct {
	config      *config.Config
	logger      *logger.Logger
	stealth     *stealth.StealthManager
	rateLimiter *stealth.RateLimiter
	db          *storage.Database
	page        *rod.Page
}

// NewMessagingManager creates a new messaging manager
func NewMessagingManager(cfg *config.Config, log *logger.Logger, s *stealth.StealthManager, rl *stealth.RateLimiter, db *storage.Database) *MessagingManager {
	return &MessagingManager{
		config:      cfg,
		logger:      log.WithModule("messaging"),
		stealth:     s,
		rateLimiter: rl,
		db:          db,
	}
}

// SetPage sets the page instance
func (m *MessagingManager) SetPage(page *rod.Page) {
	m.page = page
}

// MessageTemplateData holds data for message personalization
type MessageTemplateData struct {
	FirstName  string
	LastName   string
	FullName   string
	Company    string
	Headline   string
	Location   string
	DaysSince  int
}

// AcceptedConnection represents a newly accepted connection
type AcceptedConnection struct {
	ProfileURL string
	Name       string
	FirstName  string
	LastName   string
	Headline   string
	Company    string
	AcceptedAt time.Time
}

// CheckNewlyAcceptedConnections checks for newly accepted connection requests
func (m *MessagingManager) CheckNewlyAcceptedConnections() ([]*AcceptedConnection, error) {
	m.logger.Info("Checking for newly accepted connections")

	// Get pending requests from database
	pendingRequests, err := m.db.GetPendingConnectionRequests()
	if err != nil {
		return nil, fmt.Errorf("failed to get pending requests: %w", err)
	}

	if len(pendingRequests) == 0 {
		m.logger.Debug("No pending connection requests to check")
		return nil, nil
	}

	m.logger.Infof("Checking %d pending requests", len(pendingRequests))

	var newlyAccepted []*AcceptedConnection

	// Navigate to connections page
	err = m.page.Navigate(LinkedInConnectionsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to navigate to connections: %w", err)
	}

	m.stealth.PageLoadDelay()
	m.stealth.ApplyFingerprintMasking(m.page)

	// Get list of current 1st-degree connections
	connections, err := m.getRecentConnections()
	if err != nil {
		m.logger.WithError(err).Warn("Failed to get recent connections")
	}

	// Check each pending request
	for _, request := range pendingRequests {
		// Check if this profile is now in connections
		if m.isInConnections(request.ProfileURL, connections) {
			m.logger.WithField("profile_url", request.ProfileURL).Info("Connection accepted!")

			// Update status in database
			m.db.UpdateConnectionStatus(request.ProfileURL, "accepted")

			// Get profile details
			profile, _ := m.db.GetProfile(request.ProfileURL)

			accepted := &AcceptedConnection{
				ProfileURL: request.ProfileURL,
				AcceptedAt: time.Now(),
			}

			if profile != nil {
				accepted.Name = profile.Name
				accepted.FirstName = profile.FirstName
				accepted.LastName = profile.LastName
				accepted.Headline = profile.Headline
				accepted.Company = profile.Company
			}

			newlyAccepted = append(newlyAccepted, accepted)
		}
	}

	m.logger.Infof("Found %d newly accepted connections", len(newlyAccepted))
	return newlyAccepted, nil
}

// getRecentConnections retrieves recent connections from the connections page
func (m *MessagingManager) getRecentConnections() ([]string, error) {
	var connections []string

	// Wait for connections list
	_, err := m.page.Timeout(10 * time.Second).Element(".mn-connection-card, .mn-connections, .scaffold-finite-scroll__content")
	if err != nil {
		return nil, err
	}

	// Scroll to load more connections
	for i := 0; i < 3; i++ {
		m.stealth.HumanScroll(m.page, "down", 400)
		time.Sleep(500 * time.Millisecond)
	}

	// Get connection profile URLs
	links, err := m.page.Elements("a[href*='/in/'].mn-connection-card__link, a.ember-view[href*='/in/']")
	if err != nil {
		return nil, err
	}

	for _, link := range links {
		href, err := link.Attribute("href")
		if err == nil && href != nil {
			connections = append(connections, m.cleanProfileURL(*href))
		}
	}

	return connections, nil
}

// isInConnections checks if a profile URL is in the connections list
func (m *MessagingManager) isInConnections(profileURL string, connections []string) bool {
	cleanURL := m.cleanProfileURL(profileURL)
	for _, conn := range connections {
		if m.cleanProfileURL(conn) == cleanURL {
			return true
		}
	}
	return false
}

// cleanProfileURL cleans a LinkedIn profile URL for comparison
func (m *MessagingManager) cleanProfileURL(url string) string {
	// Remove query parameters and trailing slashes
	url = strings.Split(url, "?")[0]
	url = strings.TrimSuffix(url, "/")
	// Ensure lowercase
	url = strings.ToLower(url)
	return url
}

// SendFollowUpMessage sends a follow-up message to an accepted connection
func (m *MessagingManager) SendFollowUpMessage(connection *AcceptedConnection, customMessage string) error {
	m.logger.WithFields(map[string]interface{}{
		"profile_url": connection.ProfileURL,
		"name":        connection.Name,
	}).Info("Sending follow-up message")

	// Check rate limits
	if !m.rateLimiter.CanPerformAction("message") {
		return fmt.Errorf("message rate limit reached")
	}

	// Check if already sent follow-up
	hasSent, err := m.db.HasSentFollowUpMessage(connection.ProfileURL)
	if err != nil {
		m.logger.WithError(err).Warn("Failed to check existing follow-up")
	}
	if hasSent {
		return fmt.Errorf("follow-up message already sent to %s", connection.ProfileURL)
	}

	// Generate message if not provided
	message := customMessage
	if message == "" {
		message, err = m.generateFollowUpMessage(connection)
		if err != nil {
			return fmt.Errorf("failed to generate message: %w", err)
		}
	}

	// Navigate to profile
	err = m.navigateToProfile(connection.ProfileURL)
	if err != nil {
		return fmt.Errorf("failed to navigate to profile: %w", err)
	}

	m.stealth.ThinkingDelay()

	// Click Message button
	err = m.clickMessageButton()
	if err != nil {
		return fmt.Errorf("failed to click message button: %w", err)
	}

	// Type and send message
	err = m.typeAndSendMessage(message)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	// Record action
	m.rateLimiter.RecordAction("message")

	// Save to database
	m.saveMessage(connection.ProfileURL, message, "follow_up")

	m.logger.Message(connection.ProfileURL, "sent", "follow_up")

	return nil
}

// SendDirectMessage sends a direct message to a connection
func (m *MessagingManager) SendDirectMessage(profileURL string, message string) error {
	m.logger.WithField("profile_url", profileURL).Info("Sending direct message")

	// Check rate limits
	if !m.rateLimiter.CanPerformAction("message") {
		return fmt.Errorf("message rate limit reached")
	}

	// Navigate to profile
	err := m.navigateToProfile(profileURL)
	if err != nil {
		return fmt.Errorf("failed to navigate to profile: %w", err)
	}

	m.stealth.ThinkingDelay()

	// Click Message button
	err = m.clickMessageButton()
	if err != nil {
		return fmt.Errorf("failed to click message button: %w", err)
	}

	// Type and send message
	err = m.typeAndSendMessage(message)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	// Record action
	m.rateLimiter.RecordAction("message")

	// Save to database
	m.saveMessage(profileURL, message, "direct")

	m.logger.Message(profileURL, "sent", "direct")

	return nil
}

// navigateToProfile navigates to a profile page
func (m *MessagingManager) navigateToProfile(profileURL string) error {
	err := m.page.Navigate(profileURL)
	if err != nil {
		return err
	}

	m.stealth.PageLoadDelay()
	m.stealth.ApplyFingerprintMasking(m.page)

	// Wait for profile to load
	_, err = m.page.Timeout(10 * time.Second).Element(".pv-top-card, .scaffold-layout__main")
	if err != nil {
		return fmt.Errorf("profile not loaded: %w", err)
	}

	return nil
}

// clickMessageButton finds and clicks the Message button
func (m *MessagingManager) clickMessageButton() error {
	m.logger.Debug("Looking for Message button")

	messageSelectors := []string{
		"button[aria-label*='Message']",
		"button.pvs-profile-actions__action:has-text('Message')",
		"a.message-anywhere-button",
		"button:has-text('Message')",
	}

	var messageButton *rod.Element
	var err error

	for _, selector := range messageSelectors {
		messageButton, err = m.page.Timeout(3 * time.Second).Element(selector)
		if err == nil && messageButton != nil {
			visible, _ := messageButton.Visible()
			if visible {
				break
			}
		}
		messageButton = nil
	}

	if messageButton == nil {
		return fmt.Errorf("message button not found - may not be connected")
	}

	err = m.stealth.ClickElement(m.page, messageButton)
	if err != nil {
		return err
	}

	// Wait for message composer to open
	time.Sleep(time.Second)

	return nil
}

// typeAndSendMessage types a message and sends it
func (m *MessagingManager) typeAndSendMessage(message string) error {
	// Wait for message input
	messageInput, err := m.page.Timeout(10 * time.Second).Element(".msg-form__contenteditable, .msg-form__msg-content-container--scrollable div[contenteditable='true'], textarea.msg-form__textarea")
	if err != nil {
		return fmt.Errorf("message input not found: %w", err)
	}

	// Click on input to focus
	err = m.stealth.ClickElement(m.page, messageInput)
	if err != nil {
		return err
	}

	m.stealth.ActionDelay()

	// Truncate message if too long
	maxLength := m.config.Messaging.MaxMessageLength
	if len(message) > maxLength {
		message = message[:maxLength-3] + "..."
		m.logger.Warnf("Message truncated to %d characters", maxLength)
	}

	// Type the message with human-like behavior
	err = m.stealth.HumanType(m.page, messageInput, message)
	if err != nil {
		return fmt.Errorf("failed to type message: %w", err)
	}

	m.stealth.ThinkingDelay()

	// Find and click send button
	sendButton, err := m.page.Timeout(5 * time.Second).Element("button.msg-form__send-button:not([disabled]), button[type='submit'].msg-form__send-button")
	if err != nil {
		// Try alternative selector
		sendButton, err = m.page.Element("button.msg-form__send-button, button[aria-label='Send']")
		if err != nil {
			return fmt.Errorf("send button not found: %w", err)
		}
	}

	// Wait for button to be enabled
	time.Sleep(500 * time.Millisecond)

	err = m.stealth.ClickElement(m.page, sendButton)
	if err != nil {
		return fmt.Errorf("failed to click send: %w", err)
	}

	// Wait for message to be sent
	time.Sleep(time.Second)

	// Close message window
	m.closeMessageWindow()

	return nil
}

// closeMessageWindow closes the messaging window/popup
func (m *MessagingManager) closeMessageWindow() {
	closeButton, err := m.page.Timeout(2 * time.Second).Element("button.msg-overlay-bubble-header__control--close, button[aria-label='Close your conversation']")
	if err == nil && closeButton != nil {
		closeButton.MustClick()
	}
}

// generateFollowUpMessage generates a personalized follow-up message
func (m *MessagingManager) generateFollowUpMessage(connection *AcceptedConnection) (string, error) {
	templateStr := m.config.Messaging.FollowUpMessageTemplate
	if templateStr == "" {
		templateStr = "Thanks for connecting, {{.FirstName}}! I look forward to staying in touch."
	}

	data := MessageTemplateData{
		FirstName:  connection.FirstName,
		LastName:   connection.LastName,
		FullName:   connection.Name,
		Company:    connection.Company,
		Headline:   connection.Headline,
		DaysSince:  int(time.Since(connection.AcceptedAt).Hours() / 24),
	}

	// Handle empty first name
	if data.FirstName == "" {
		if data.FullName != "" {
			data.FirstName = strings.Split(data.FullName, " ")[0]
		} else {
			data.FirstName = "there"
		}
	}

	tmpl, err := template.New("message").Parse(templateStr)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

// saveMessage saves a sent message to the database
func (m *MessagingManager) saveMessage(profileURL, content, messageType string) {
	// Get profile ID
	profile, _ := m.db.GetProfile(profileURL)
	var profileID int64
	if profile != nil {
		profileID = profile.ID
	}

	message := &storage.Message{
		ProfileID:   profileID,
		ProfileURL:  profileURL,
		Content:     content,
		Template:    m.config.Messaging.FollowUpMessageTemplate,
		MessageType: messageType,
	}

	m.db.SaveMessage(message)
}

// SendBulkFollowUpMessages sends follow-up messages to multiple connections
func (m *MessagingManager) SendBulkFollowUpMessages(connections []*AcceptedConnection, customMessage string) (int, int, error) {
	sent := 0
	failed := 0

	for _, conn := range connections {
		// Check rate limits
		if !m.rateLimiter.CanPerformAction("message") {
			m.logger.Warn("Rate limit reached, stopping bulk messages")
			break
		}

		err := m.SendFollowUpMessage(conn, customMessage)
		if err != nil {
			m.logger.WithError(err).WithField("profile", conn.ProfileURL).Warn("Failed to send follow-up")
			failed++
		} else {
			sent++
		}

		// Natural delay between messages
		m.stealth.ThinkingDelay()
		m.rateLimiter.WaitForNextAction()
	}

	m.logger.Infof("Bulk follow-up messages: %d sent, %d failed", sent, failed)
	return sent, failed, nil
}

// GetRemainingMessages returns how many more messages can be sent today
func (m *MessagingManager) GetRemainingMessages() int {
	return m.rateLimiter.GetRemainingActions("message")
}

// ProcessNewConnectionsWorkflow checks for new connections and sends follow-ups
func (m *MessagingManager) ProcessNewConnectionsWorkflow() error {
	m.logger.Info("Processing new connections workflow")

	// Check for newly accepted connections
	newConnections, err := m.CheckNewlyAcceptedConnections()
	if err != nil {
		return fmt.Errorf("failed to check new connections: %w", err)
	}

	if len(newConnections) == 0 {
		m.logger.Info("No new connections to process")
		return nil
	}

	m.logger.Infof("Found %d new connections to follow up with", len(newConnections))

	// Send follow-up messages
	sent, failed, err := m.SendBulkFollowUpMessages(newConnections, "")
	if err != nil {
		return err
	}

	m.logger.Infof("Workflow complete: %d messages sent, %d failed", sent, failed)
	return nil
}
