// Package auth provides LinkedIn authentication functionality.
// It handles login, session persistence, and security checkpoint detection.
package auth

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/nikshitha/linkedin-automation-poc/config"
	"github.com/nikshitha/linkedin-automation-poc/logger"
	"github.com/nikshitha/linkedin-automation-poc/stealth"
	"github.com/nikshitha/linkedin-automation-poc/storage"
)

// Common LinkedIn URLs
const (
	LinkedInBaseURL     = "https://www.linkedin.com"
	LinkedInLoginURL    = "https://www.linkedin.com/login"
	LinkedInFeedURL     = "https://www.linkedin.com/feed/"
	LinkedInCheckpoint  = "https://www.linkedin.com/checkpoint"
)

// Error types for authentication
var (
	ErrLoginFailed       = errors.New("login failed: invalid credentials or unknown error")
	ErrTwoFactorRequired = errors.New("two-factor authentication required")
	ErrCaptchaRequired   = errors.New("captcha verification required")
	ErrSecurityCheck     = errors.New("security checkpoint detected")
	ErrSessionExpired    = errors.New("session has expired")
	ErrAccountRestricted = errors.New("account access restricted")
)

// Authenticator handles LinkedIn authentication
type Authenticator struct {
	config    *config.Config
	logger    *logger.Logger
	stealth   *stealth.StealthManager
	db        *storage.Database
	page      *rod.Page
	browser   *rod.Browser
	isLoggedIn bool
}

// NewAuthenticator creates a new authenticator
func NewAuthenticator(cfg *config.Config, log *logger.Logger, s *stealth.StealthManager, db *storage.Database) *Authenticator {
	return &Authenticator{
		config:  cfg,
		logger:  log.WithModule("auth"),
		stealth: s,
		db:      db,
	}
}

// SetBrowser sets the browser instance
func (a *Authenticator) SetBrowser(browser *rod.Browser) {
	a.browser = browser
}

// SetPage sets the page instance
func (a *Authenticator) SetPage(page *rod.Page) {
	a.page = page
}

// Login performs LinkedIn login with human-like behavior
func (a *Authenticator) Login() error {
	a.logger.Info("Starting login process")

	// First, try to use existing session
	if a.tryExistingSession() {
		a.logger.Info("Successfully restored existing session")
		a.isLoggedIn = true
		return nil
	}

	// Navigate to login page
	a.logger.Info("Navigating to login page")
	err := a.page.Navigate(LinkedInLoginURL)
	if err != nil {
		return fmt.Errorf("failed to navigate to login page: %w", err)
	}

	// Wait for page load
	a.stealth.PageLoadDelay()
	err = a.page.WaitLoad()
	if err != nil {
		return fmt.Errorf("failed to load login page: %w", err)
	}

	// Apply fingerprint masking
	a.stealth.ApplyFingerprintMasking(a.page)

	// Random mouse movement before interaction
	a.stealth.RandomMouseWander(a.page)
	a.stealth.ThinkingDelay()

	// Enter email
	a.logger.Debug("Entering email")
	emailField, err := a.page.Element("#username")
	if err != nil {
		return fmt.Errorf("failed to find email field: %w", err)
	}

	// Click on email field first
	err = a.stealth.ClickElement(a.page, emailField)
	if err != nil {
		return fmt.Errorf("failed to click email field: %w", err)
	}

	// Type email with human-like behavior
	err = a.stealth.HumanType(a.page, emailField, a.config.LinkedIn.Email)
	if err != nil {
		return fmt.Errorf("failed to enter email: %w", err)
	}

	// Small delay before moving to password
	a.stealth.ActionDelay()

	// Enter password
	a.logger.Debug("Entering password")
	passwordField, err := a.page.Element("#password")
	if err != nil {
		return fmt.Errorf("failed to find password field: %w", err)
	}

	err = a.stealth.ClickElement(a.page, passwordField)
	if err != nil {
		return fmt.Errorf("failed to click password field: %w", err)
	}

	err = a.stealth.HumanType(a.page, passwordField, a.config.LinkedIn.Password)
	if err != nil {
		return fmt.Errorf("failed to enter password: %w", err)
	}

	// Thinking delay before submitting
	a.stealth.ThinkingDelay()

	// Click login button
	a.logger.Debug("Clicking login button")
	loginButton, err := a.page.Element(`button[type="submit"]`)
	if err != nil {
		return fmt.Errorf("failed to find login button: %w", err)
	}

	err = a.stealth.ClickElement(a.page, loginButton)
	if err != nil {
		return fmt.Errorf("failed to click login button: %w", err)
	}

	// Wait for navigation
	a.stealth.PageLoadDelay()
	time.Sleep(3 * time.Second) // Extra wait for login processing

	// Check login result
	return a.checkLoginResult()
}

// checkLoginResult verifies if login was successful and handles errors
func (a *Authenticator) checkLoginResult() error {
	currentURL := a.page.MustInfo().URL

	a.logger.WithField("url", currentURL).Debug("Checking login result")

	// Check for successful login (redirected to feed)
	if strings.Contains(currentURL, "/feed") {
		a.logger.Info("Login successful - redirected to feed")
		a.isLoggedIn = true
		a.saveCookies()
		return nil
	}

	// Check for security checkpoint
	if strings.Contains(currentURL, "/checkpoint") {
		// Detect specific checkpoint type
		return a.handleSecurityCheckpoint()
	}

	// Check for 2FA
	if a.detect2FA() {
		a.logger.SecurityEvent("2FA_REQUIRED", "Two-factor authentication is required")
		return ErrTwoFactorRequired
	}

	// Check for captcha
	if a.detectCaptcha() {
		a.logger.SecurityEvent("CAPTCHA_REQUIRED", "Captcha verification is required")
		return ErrCaptchaRequired
	}

	// Check for login errors on the page
	if a.detectLoginError() {
		return ErrLoginFailed
	}

	// Check for account restrictions
	if a.detectAccountRestriction() {
		a.logger.SecurityEvent("ACCOUNT_RESTRICTED", "Account access has been restricted")
		return ErrAccountRestricted
	}

	// If we're still on login page, login failed
	if strings.Contains(currentURL, "/login") {
		return ErrLoginFailed
	}

	// Unknown state - might be logged in
	a.logger.Warn("Login state unclear, attempting to verify")
	if a.IsLoggedIn() {
		a.isLoggedIn = true
		a.saveCookies()
		return nil
	}

	return ErrLoginFailed
}

// handleSecurityCheckpoint handles various security checkpoints
func (a *Authenticator) handleSecurityCheckpoint() error {
	currentURL := a.page.MustInfo().URL
	pageHTML, _ := a.page.HTML()

	// Phone verification
	if strings.Contains(pageHTML, "phone") || strings.Contains(currentURL, "phone-challenge") {
		a.logger.SecurityEvent("PHONE_VERIFICATION", "Phone verification required")
		return fmt.Errorf("%w: phone verification required", ErrSecurityCheck)
	}

	// Email verification
	if strings.Contains(pageHTML, "email") || strings.Contains(currentURL, "email-challenge") {
		a.logger.SecurityEvent("EMAIL_VERIFICATION", "Email verification required")
		return fmt.Errorf("%w: email verification required", ErrSecurityCheck)
	}

	// Identity verification
	if strings.Contains(pageHTML, "identity") || strings.Contains(pageHTML, "verify") {
		a.logger.SecurityEvent("IDENTITY_VERIFICATION", "Identity verification required")
		return fmt.Errorf("%w: identity verification required", ErrSecurityCheck)
	}

	// Generic checkpoint
	a.logger.SecurityEvent("SECURITY_CHECKPOINT", "Unknown security checkpoint detected")
	return ErrSecurityCheck
}

// detect2FA checks if two-factor authentication is required
func (a *Authenticator) detect2FA() bool {
	// Look for 2FA indicators
	indicators := []string{
		"#input__phone_verification_pin",
		"#input__email_verification_pin",
		"two-step-challenge",
		"verification-code",
	}

	for _, selector := range indicators {
		el, err := a.page.Timeout(2 * time.Second).Element(selector)
		if err == nil && el != nil {
			return true
		}
	}

	// Check page content
	pageHTML, _ := a.page.HTML()
	twoFAKeywords := []string{
		"Enter the code",
		"verification code",
		"two-step verification",
		"We sent a code",
	}

	for _, keyword := range twoFAKeywords {
		if strings.Contains(pageHTML, keyword) {
			return true
		}
	}

	return false
}

// detectCaptcha checks if captcha verification is required
func (a *Authenticator) detectCaptcha() bool {
	// Look for captcha indicators
	captchaSelectors := []string{
		"#captcha",
		".captcha-container",
		"iframe[src*='captcha']",
		"iframe[src*='recaptcha']",
		"#arkose-challenge",
	}

	for _, selector := range captchaSelectors {
		el, err := a.page.Timeout(2 * time.Second).Element(selector)
		if err == nil && el != nil {
			return true
		}
	}

	// Check page content
	pageHTML, _ := a.page.HTML()
	return strings.Contains(pageHTML, "captcha") || strings.Contains(pageHTML, "robot")
}

// detectLoginError checks for login error messages
func (a *Authenticator) detectLoginError() bool {
	errorSelectors := []string{
		".form__label--error",
		"#error-for-username",
		"#error-for-password",
		".alert-content",
	}

	for _, selector := range errorSelectors {
		el, err := a.page.Timeout(2 * time.Second).Element(selector)
		if err == nil && el != nil {
			text, _ := el.Text()
			if text != "" {
				a.logger.WithField("error_message", text).Error("Login error detected")
				return true
			}
		}
	}

	return false
}

// detectAccountRestriction checks if the account is restricted
func (a *Authenticator) detectAccountRestriction() bool {
	pageHTML, _ := a.page.HTML()
	restrictionKeywords := []string{
		"account has been restricted",
		"unusual activity",
		"temporarily restricted",
		"account suspended",
	}

	for _, keyword := range restrictionKeywords {
		if strings.Contains(strings.ToLower(pageHTML), strings.ToLower(keyword)) {
			return true
		}
	}

	return false
}

// IsLoggedIn checks if the user is currently logged in
func (a *Authenticator) IsLoggedIn() bool {
	if a.isLoggedIn {
		return true
	}

	// Navigate to feed to check
	a.page.Navigate(LinkedInFeedURL)
	a.stealth.PageLoadDelay()
	time.Sleep(2 * time.Second)

	currentURL := a.page.MustInfo().URL

	// Check if redirected to login
	if strings.Contains(currentURL, "/login") || strings.Contains(currentURL, "/authwall") {
		return false
	}

	// Check for feed elements
	feedElement, err := a.page.Timeout(5 * time.Second).Element(".feed-shared-update-v2, .feed-follows-module")
	if err == nil && feedElement != nil {
		a.isLoggedIn = true
		return true
	}

	// Check for navigation elements (logged-in user has these)
	navElement, err := a.page.Timeout(3 * time.Second).Element("#global-nav")
	if err == nil && navElement != nil {
		a.isLoggedIn = true
		return true
	}

	return false
}

// tryExistingSession attempts to use an existing session from cookies
func (a *Authenticator) tryExistingSession() bool {
	a.logger.Debug("Attempting to restore existing session")

	// Load cookies from file
	cookies, err := a.db.LoadCookiesFromFile(a.config.Storage.CookiesPath)
	if err != nil {
		a.logger.WithError(err).Debug("Failed to load cookies from file")
		return false
	}

	if len(cookies) == 0 {
		a.logger.Debug("No existing cookies found")
		return false
	}

	// Check if cookies are expired
	validCookies := make([]*storage.SessionCookie, 0)
	now := time.Now().Unix()
	for _, cookie := range cookies {
		if cookie.Expires == 0 || cookie.Expires > now {
			validCookies = append(validCookies, cookie)
		}
	}

	if len(validCookies) == 0 {
		a.logger.Debug("All cookies have expired")
		return false
	}

	// Navigate to LinkedIn first (needed to set cookies for the domain)
	a.page.Navigate(LinkedInBaseURL)
	a.stealth.PageLoadDelay()

	// Set cookies
	for _, cookie := range validCookies {
		err := a.page.SetCookies([]*proto.NetworkCookieParam{{
			Name:     cookie.Name,
			Value:    cookie.Value,
			Domain:   cookie.Domain,
			Path:     cookie.Path,
			Expires:  proto.TimeSinceEpoch(cookie.Expires),
			HTTPOnly: cookie.HTTPOnly,
			Secure:   cookie.Secure,
		}})
		if err != nil {
			a.logger.WithError(err).Debug("Failed to set cookie")
		}
	}

	// Navigate to feed to test session
	a.page.Navigate(LinkedInFeedURL)
	a.stealth.PageLoadDelay()
	time.Sleep(2 * time.Second)

	return a.IsLoggedIn()
}

// saveCookies saves the current session cookies
func (a *Authenticator) saveCookies() error {
	cookies, err := a.page.Cookies([]string{LinkedInBaseURL})
	if err != nil {
		return fmt.Errorf("failed to get cookies: %w", err)
	}

	// Convert to storage format
	storageCookies := make([]*storage.SessionCookie, len(cookies))
	for i, cookie := range cookies {
		storageCookies[i] = &storage.SessionCookie{
			Name:     cookie.Name,
			Value:    cookie.Value,
			Domain:   cookie.Domain,
			Path:     cookie.Path,
			Expires:  int64(cookie.Expires),
			HTTPOnly: cookie.HTTPOnly,
			Secure:   cookie.Secure,
		}
	}

	// Save to database
	if err := a.db.SaveCookies(storageCookies); err != nil {
		a.logger.WithError(err).Warn("Failed to save cookies to database")
	}

	// Save to file
	if err := a.db.SaveCookiesToFile(storageCookies, a.config.Storage.CookiesPath); err != nil {
		a.logger.WithError(err).Warn("Failed to save cookies to file")
		return err
	}

	a.logger.Info("Session cookies saved successfully")
	return nil
}

// Logout performs logout from LinkedIn
func (a *Authenticator) Logout() error {
	a.logger.Info("Logging out")

	// Navigate to logout
	err := a.page.Navigate("https://www.linkedin.com/m/logout/")
	if err != nil {
		return fmt.Errorf("failed to navigate to logout: %w", err)
	}

	a.stealth.PageLoadDelay()
	a.isLoggedIn = false

	// Clear stored cookies
	a.page.SetCookies(nil)

	a.logger.Info("Logged out successfully")
	return nil
}

// RefreshSession refreshes the session by re-authenticating
func (a *Authenticator) RefreshSession() error {
	a.logger.Info("Refreshing session")
	a.isLoggedIn = false
	return a.Login()
}

// GetCurrentUser returns information about the currently logged-in user
func (a *Authenticator) GetCurrentUser() (map[string]string, error) {
	if !a.IsLoggedIn() {
		return nil, ErrSessionExpired
	}

	// Navigate to profile
	a.page.Navigate("https://www.linkedin.com/in/me/")
	a.stealth.PageLoadDelay()
	time.Sleep(2 * time.Second)

	user := make(map[string]string)

	// Get name
	nameEl, err := a.page.Timeout(5 * time.Second).Element("h1.text-heading-xlarge")
	if err == nil && nameEl != nil {
		user["name"], _ = nameEl.Text()
	}

	// Get headline
	headlineEl, err := a.page.Timeout(3 * time.Second).Element(".text-body-medium.break-words")
	if err == nil && headlineEl != nil {
		user["headline"], _ = headlineEl.Text()
	}

	// Get profile URL
	user["profile_url"] = a.page.MustInfo().URL

	return user, nil
}
