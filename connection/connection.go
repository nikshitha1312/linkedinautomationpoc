// Package connection handles LinkedIn connection requests.
// It navigates to profiles, sends personalized connection notes, and tracks requests.
package connection

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/go-rod/rod"
	"github.com/nikshitha/linkedin-automation-poc/config"
	"github.com/nikshitha/linkedin-automation-poc/logger"
	"github.com/nikshitha/linkedin-automation-poc/search"
	"github.com/nikshitha/linkedin-automation-poc/stealth"
	"github.com/nikshitha/linkedin-automation-poc/storage"
)

// ConnectionManager handles connection request operations
type ConnectionManager struct {
	config      *config.Config
	logger      *logger.Logger
	stealth     *stealth.StealthManager
	rateLimiter *stealth.RateLimiter
	db          *storage.Database
	page        *rod.Page
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager(cfg *config.Config, log *logger.Logger, s *stealth.StealthManager, rl *stealth.RateLimiter, db *storage.Database) *ConnectionManager {
	return &ConnectionManager{
		config:      cfg,
		logger:      log.WithModule("connection"),
		stealth:     s,
		rateLimiter: rl,
		db:          db,
	}
}

// SetPage sets the page instance
func (c *ConnectionManager) SetPage(page *rod.Page) {
	c.page = page
}

// TemplateData holds data for personalizing connection notes
type TemplateData struct {
	FirstName  string
	LastName   string
	FullName   string
	Company    string
	Headline   string
	Location   string
	Connection string
}

// SendConnectionRequest sends a connection request to a profile
func (c *ConnectionManager) SendConnectionRequest(profile *search.SearchResult, customNote string) error {
	c.logger.WithFields(map[string]interface{}{
		"profile_url": profile.ProfileURL,
		"name":        profile.Name,
	}).Info("Sending connection request")

	// Check rate limits
	if !c.rateLimiter.CanPerformAction("connection") {
		remaining := c.rateLimiter.GetRemainingActions("connection")
		return fmt.Errorf("connection rate limit reached (remaining: %d)", remaining)
	}

	// Check if already sent
	hasSent, err := c.db.HasSentConnectionRequest(profile.ProfileURL)
	if err != nil {
		c.logger.WithError(err).Warn("Failed to check existing connection request")
	}
	if hasSent {
		c.logger.Warn("Connection request already sent to this profile")
		return fmt.Errorf("connection request already sent to %s", profile.ProfileURL)
	}

	// Navigate to profile
	err = c.navigateToProfile(profile.ProfileURL)
	if err != nil {
		return fmt.Errorf("failed to navigate to profile: %w", err)
	}

	// Record profile view
	c.rateLimiter.RecordAction("profile_view")
	c.db.IncrementProfileViews()

	// Random behavior on profile page
	c.stealth.RandomMouseWander(c.page)
	c.stealth.ThinkingDelay()

	// Scroll down to simulate reading profile
	c.stealth.HumanScroll(c.page, "down", 300)
	c.stealth.ActionDelay()

	// Find and click Connect button
	err = c.clickConnectButton()
	if err != nil {
		return fmt.Errorf("failed to click connect button: %w", err)
	}

	// Generate personalized note if not provided
	note := customNote
	if note == "" {
		note, err = c.generatePersonalizedNote(profile)
		if err != nil {
			c.logger.WithError(err).Warn("Failed to generate personalized note, sending without note")
			note = ""
		}
	}

	// Send with note if applicable
	if note != "" {
		err = c.addConnectionNote(note)
		if err != nil {
			c.logger.WithError(err).Warn("Failed to add note, sending without note")
		}
	}

	// Click Send button
	err = c.clickSendButton()
	if err != nil {
		return fmt.Errorf("failed to send connection request: %w", err)
	}

	// Record the connection request
	c.rateLimiter.RecordAction("connection")

	// Save to database
	c.saveConnectionRequest(profile, note)

	c.logger.ConnectionRequest(profile.ProfileURL, "sent", note)

	// Wait before next action
	c.rateLimiter.WaitForNextAction()

	return nil
}

// navigateToProfile navigates to a LinkedIn profile page
func (c *ConnectionManager) navigateToProfile(profileURL string) error {
	c.logger.WithField("url", profileURL).Debug("Navigating to profile")

	err := c.page.Navigate(profileURL)
	if err != nil {
		return err
	}

	c.stealth.PageLoadDelay()
	err = c.page.WaitLoad()
	if err != nil {
		return err
	}

	// Apply fingerprint masking
	c.stealth.ApplyFingerprintMasking(c.page)

	// Wait for profile content
	_, err = c.page.Timeout(10 * time.Second).Element(".pv-top-card, .profile-background-image, .scaffold-layout__main")
	if err != nil {
		return fmt.Errorf("profile content not loaded: %w", err)
	}

	return nil
}

// clickConnectButton finds and clicks the Connect button
func (c *ConnectionManager) clickConnectButton() error {
	c.logger.Debug("Looking for Connect button")

	// Various selectors for Connect button (LinkedIn changes these frequently)
	connectSelectors := []string{
		// Primary Connect button on profile
		"button.pvs-profile-actions__action[aria-label*='connect' i]",
		"button[aria-label*='Invite'][aria-label*='connect' i]",
		"button.artdeco-button--primary:has-text('Connect')",
		// More button dropdown
		"div.pvs-profile-actions button:has-text('Connect')",
		// Generic connect buttons
		"button:has-text('Connect'):not(:has-text('Message'))",
		"button[aria-label*='Connect']",
		// Fallback
		"button.pv-s-profile-actions__overflow-toggle", // More button
	}

	var connectButton *rod.Element
	var err error

	for _, selector := range connectSelectors {
		connectButton, err = c.page.Timeout(3 * time.Second).Element(selector)
		if err == nil && connectButton != nil {
			// Verify it's visible and clickable
			visible, _ := connectButton.Visible()
			if visible {
				break
			}
		}
		connectButton = nil
	}

	// If not found, try the More button dropdown
	if connectButton == nil {
		err = c.tryMoreButtonDropdown()
		if err != nil {
			return fmt.Errorf("connect button not found: %w", err)
		}
		return nil
	}

	// Check button text to ensure it's Connect (not Connected or Pending)
	buttonText, _ := connectButton.Text()
	buttonText = strings.ToLower(strings.TrimSpace(buttonText))
	
	if strings.Contains(buttonText, "pending") {
		return fmt.Errorf("connection request already pending")
	}
	if strings.Contains(buttonText, "connected") || strings.Contains(buttonText, "message") {
		return fmt.Errorf("already connected to this profile")
	}

	// Human-like click
	err = c.stealth.ClickElement(c.page, connectButton)
	if err != nil {
		return fmt.Errorf("failed to click connect button: %w", err)
	}

	c.stealth.ActionDelay()
	return nil
}

// tryMoreButtonDropdown tries to find Connect in the More dropdown
func (c *ConnectionManager) tryMoreButtonDropdown() error {
	c.logger.Debug("Trying More button dropdown")

	// Find and click More button
	moreButton, err := c.page.Timeout(3 * time.Second).Element("button[aria-label='More actions'], button.artdeco-dropdown__trigger")
	if err != nil {
		return fmt.Errorf("more button not found: %w", err)
	}

	err = c.stealth.ClickElement(c.page, moreButton)
	if err != nil {
		return err
	}

	c.stealth.ActionDelay()

	// Find Connect in dropdown
	dropdownConnect, err := c.page.Timeout(3 * time.Second).Element("div.artdeco-dropdown__content button:has-text('Connect'), li.artdeco-dropdown__item:has-text('Connect')")
	if err != nil {
		return fmt.Errorf("connect option not found in dropdown: %w", err)
	}

	err = c.stealth.ClickElement(c.page, dropdownConnect)
	if err != nil {
		return err
	}

	return nil
}

// addConnectionNote adds a personalized note to the connection request
func (c *ConnectionManager) addConnectionNote(note string) error {
	c.logger.Debug("Adding connection note")

	// Wait for modal to appear
	time.Sleep(500 * time.Millisecond)

	// Look for "Add a note" button
	addNoteButton, err := c.page.Timeout(5 * time.Second).Element("button[aria-label*='Add a note'], button:has-text('Add a note')")
	if err != nil {
		// Note might not be available for this connection type
		c.logger.Debug("Add note button not found, may not be available")
		return nil
	}

	err = c.stealth.ClickElement(c.page, addNoteButton)
	if err != nil {
		return fmt.Errorf("failed to click add note button: %w", err)
	}

	c.stealth.ActionDelay()

	// Find note textarea
	noteTextarea, err := c.page.Timeout(5 * time.Second).Element("textarea[name='message'], textarea#custom-message, textarea.connect-button-send-invite__custom-message")
	if err != nil {
		// Try alternative selectors
		noteTextarea, err = c.page.Element("textarea")
		if err != nil {
			return fmt.Errorf("note textarea not found: %w", err)
		}
	}

	// Truncate note if too long
	maxLength := c.config.Messaging.MaxNoteLength
	if len(note) > maxLength {
		note = note[:maxLength-3] + "..."
		c.logger.Warnf("Note truncated to %d characters", maxLength)
	}

	// Clear any existing text
	noteTextarea.SelectAllText()
	
	// Type the note with human-like behavior
	err = c.stealth.HumanType(c.page, noteTextarea, note)
	if err != nil {
		return fmt.Errorf("failed to type note: %w", err)
	}

	c.stealth.ActionDelay()
	return nil
}

// clickSendButton clicks the Send button to submit the connection request
func (c *ConnectionManager) clickSendButton() error {
	c.logger.Debug("Clicking send button")

	// Various selectors for Send button
	sendSelectors := []string{
		"button[aria-label*='Send']:not([disabled])",
		"button.artdeco-button--primary:has-text('Send'):not([disabled])",
		"button[aria-label*='invitation']:not([disabled])",
		"button.ml1:has-text('Send')",
		"button:has-text('Send invitation')",
		"button:has-text('Send now')",
	}

	var sendButton *rod.Element
	var err error

	for _, selector := range sendSelectors {
		sendButton, err = c.page.Timeout(3 * time.Second).Element(selector)
		if err == nil && sendButton != nil {
			visible, _ := sendButton.Visible()
			if visible {
				break
			}
		}
		sendButton = nil
	}

	if sendButton == nil {
		return fmt.Errorf("send button not found")
	}

	// Human-like click
	err = c.stealth.ClickElement(c.page, sendButton)
	if err != nil {
		return fmt.Errorf("failed to click send button: %w", err)
	}

	// Wait for confirmation
	time.Sleep(time.Second)

	// Check for success (modal closes)
	_, modalErr := c.page.Timeout(3 * time.Second).Element(".send-invite, .artdeco-modal--layer-default")
	if modalErr != nil {
		// Modal closed, likely success
		return nil
	}

	// Check for error messages
	errorEl, err := c.page.Timeout(2 * time.Second).Element(".artdeco-inline-feedback--error, .form-error")
	if err == nil && errorEl != nil {
		errorText, _ := errorEl.Text()
		return fmt.Errorf("connection request failed: %s", errorText)
	}

	return nil
}

// generatePersonalizedNote generates a personalized connection note using templates
func (c *ConnectionManager) generatePersonalizedNote(profile *search.SearchResult) (string, error) {
	templateStr := c.config.Messaging.ConnectionNoteTemplate
	if templateStr == "" {
		return "", nil
	}

	// Prepare template data
	data := TemplateData{
		FirstName:  profile.FirstName,
		LastName:   profile.LastName,
		FullName:   profile.Name,
		Company:    profile.Company,
		Headline:   profile.Headline,
		Location:   profile.Location,
		Connection: profile.Connection,
	}

	// Handle empty first name
	if data.FirstName == "" {
		data.FirstName = "there"
	}

	// Parse and execute template
	tmpl, err := template.New("note").Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	note := buf.String()

	// Ensure note doesn't exceed max length
	if len(note) > c.config.Messaging.MaxNoteLength {
		note = note[:c.config.Messaging.MaxNoteLength]
	}

	return note, nil
}

// saveConnectionRequest saves the connection request to the database
func (c *ConnectionManager) saveConnectionRequest(profile *search.SearchResult, note string) error {
	// First save the profile
	profileID, err := c.db.SaveProfile(&storage.Profile{
		ProfileURL:       profile.ProfileURL,
		Name:             profile.Name,
		FirstName:        profile.FirstName,
		LastName:         profile.LastName,
		Headline:         profile.Headline,
		Company:          profile.Company,
		Location:         profile.Location,
		ConnectionDegree: profile.Connection,
	})
	if err != nil {
		c.logger.WithError(err).Warn("Failed to save profile")
	}

	// Save connection request
	request := &storage.ConnectionRequest{
		ProfileID:  profileID,
		ProfileURL: profile.ProfileURL,
		Note:       note,
		Status:     "pending",
	}

	_, err = c.db.SaveConnectionRequest(request)
	return err
}

// SendBulkConnectionRequests sends connection requests to multiple profiles
func (c *ConnectionManager) SendBulkConnectionRequests(profiles []*search.SearchResult, customNote string) (int, int, error) {
	sent := 0
	failed := 0

	for _, profile := range profiles {
		// Check rate limits before each request
		if !c.rateLimiter.CanPerformAction("connection") {
			c.logger.Warn("Rate limit reached, stopping bulk connection requests")
			break
		}

		err := c.SendConnectionRequest(profile, customNote)
		if err != nil {
			c.logger.WithError(err).WithField("profile", profile.ProfileURL).Warn("Failed to send connection request")
			failed++
		} else {
			sent++
		}

		// Natural delay between requests
		c.stealth.ThinkingDelay()
		c.rateLimiter.WaitForNextAction()
	}

	c.logger.Infof("Bulk connection requests: %d sent, %d failed", sent, failed)
	return sent, failed, nil
}

// GetRemainingConnections returns how many more connections can be sent today
func (c *ConnectionManager) GetRemainingConnections() int {
	return c.rateLimiter.GetRemainingActions("connection")
}

// WithdrawConnectionRequest withdraws a pending connection request
func (c *ConnectionManager) WithdrawConnectionRequest(profileURL string) error {
	c.logger.WithField("profile_url", profileURL).Info("Withdrawing connection request")

	// Navigate to profile
	err := c.navigateToProfile(profileURL)
	if err != nil {
		return err
	}

	c.stealth.ThinkingDelay()

	// Find Pending button
	pendingButton, err := c.page.Timeout(5 * time.Second).Element("button:has-text('Pending'), button[aria-label*='Pending']")
	if err != nil {
		return fmt.Errorf("pending button not found - may not have a pending request")
	}

	err = c.stealth.ClickElement(c.page, pendingButton)
	if err != nil {
		return err
	}

	c.stealth.ActionDelay()

	// Click Withdraw
	withdrawButton, err := c.page.Timeout(3 * time.Second).Element("button:has-text('Withdraw'), button[aria-label*='Withdraw']")
	if err != nil {
		return fmt.Errorf("withdraw button not found")
	}

	err = c.stealth.ClickElement(c.page, withdrawButton)
	if err != nil {
		return err
	}

	// Update database
	c.db.UpdateConnectionStatus(profileURL, "withdrawn")

	c.logger.Info("Connection request withdrawn")
	return nil
}
