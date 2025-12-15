// Package connection - navigate.go handles navigation to connections and profile search
package connection

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
)

// ConnectionsPageURL is the URL for the connections page
const ConnectionsPageURL = "https://www.linkedin.com/mynetwork/invite-connect/connections/"

// NavigateAndOpenProfile navigates to My Network → Connections, applies filters,
// searches for a specific person, and opens their profile
func (c *ConnectionManager) NavigateAndOpenProfile(institution string, personName string) error {
	c.logger.Info("Starting connections navigation workflow")
	c.logger.WithFields(map[string]interface{}{
		"institution": institution,
		"person":      personName,
	}).Info("Search parameters")

	// Step 1: Navigate to Connections page
	if err := c.navigateToConnectionsPage(); err != nil {
		return fmt.Errorf("failed to navigate to connections page: %w", err)
	}

	// Step 2: Wait for page to load and perform human-like actions
	c.stealth.PageLoadDelay()
	c.stealth.RandomMouseWander(c.page)
	c.stealth.ThinkingDelay()

	// Step 3: Search for the person with institution context
	if err := c.searchConnection(personName); err != nil {
		return fmt.Errorf("failed to search for connection: %w", err)
	}

	// Step 4: Wait for results with human-like behavior
	c.stealth.ThinkingDelay()
	c.stealth.HumanScroll(c.page, "down", 100)

	// Step 5: Find and click on the profile
	if err := c.findAndClickProfile(personName, institution); err != nil {
		return fmt.Errorf("failed to find and open profile: %w", err)
	}

	c.logger.Info("Successfully opened profile!")
	return nil
}

// navigateToConnectionsPage navigates to LinkedIn My Network → Connections page
func (c *ConnectionManager) navigateToConnectionsPage() error {
	c.logger.Info("Navigating to Connections page")

	// First try direct URL navigation
	err := c.page.Navigate(ConnectionsPageURL)
	if err != nil {
		return fmt.Errorf("failed to navigate: %w", err)
	}

	// Wait for page to load
	c.stealth.PageLoadDelay()
	err = c.page.WaitLoad()
	if err != nil {
		c.logger.WithError(err).Warn("Page load wait failed, continuing anyway")
	}

	// Additional wait for dynamic content
	time.Sleep(2 * time.Second)

	// Verify we're on the connections page
	currentURL := c.page.MustInfo().URL
	c.logger.WithField("url", currentURL).Debug("Current URL")

	if !strings.Contains(currentURL, "connections") && !strings.Contains(currentURL, "mynetwork") {
		// Try clicking My Network from navbar
		c.logger.Debug("Not on connections page, trying to navigate via My Network")
		return c.navigateViaMyNetwork()
	}

	c.logger.Info("Successfully navigated to Connections page")
	return nil
}

// navigateViaMyNetwork navigates to connections through My Network menu
func (c *ConnectionManager) navigateViaMyNetwork() error {
	c.logger.Debug("Navigating via My Network menu")

	// Click on My Network in navbar
	myNetworkSelectors := []string{
		`a[href*="/mynetwork/"]`,
		`span:has-text("My Network")`,
		`a[data-link-to="mynetwork"]`,
		`.global-nav__primary-link[href*="mynetwork"]`,
	}

	var myNetworkLink *rod.Element
	var err error
	for _, selector := range myNetworkSelectors {
		myNetworkLink, err = c.page.Timeout(5 * time.Second).Element(selector)
		if err == nil && myNetworkLink != nil {
			break
		}
	}

	if myNetworkLink == nil {
		return fmt.Errorf("could not find My Network link")
	}

	// Human-like hover and click
	c.stealth.HoverElement(c.page, myNetworkLink)
	c.stealth.ActionDelay()
	myNetworkLink.MustClick()

	c.stealth.PageLoadDelay()
	time.Sleep(2 * time.Second)

	// Now look for Connections link
	connectionsSelectors := []string{
		`a[href*="/connections/"]`,
		`a:has-text("Connections")`,
		`.mn-community-summary__link`,
	}

	var connectionsLink *rod.Element
	for _, selector := range connectionsSelectors {
		connectionsLink, err = c.page.Timeout(5 * time.Second).Element(selector)
		if err == nil && connectionsLink != nil {
			break
		}
	}

	if connectionsLink != nil {
		c.stealth.HoverElement(c.page, connectionsLink)
		c.stealth.ActionDelay()
		connectionsLink.MustClick()
		c.stealth.PageLoadDelay()
	}

	return nil
}

// searchConnection searches for a connection by name using global search
func (c *ConnectionManager) searchConnection(personName string) error {
	c.logger.WithField("name", personName).Info("Searching for connection")

	// Use global search which is more reliable
	return c.searchViaGlobalSearch(personName)
}

// searchViaGlobalSearch uses LinkedIn's global search to find the connection
func (c *ConnectionManager) searchViaGlobalSearch(personName string) error {
	c.logger.Info("Using global search to find connection")

	// Wait for page to stabilize
	time.Sleep(2 * time.Second)

	// Find global search input - try multiple selectors
	globalSearchSelectors := []string{
		`input.search-global-typeahead__input`,
		`input[placeholder="Search"]`,
		`input[aria-label="Search"]`,
		`input[role="combobox"]`,
		`.search-global-typeahead input`,
		`#global-nav-typeahead input`,
	}

	var searchInput *rod.Element
	var err error
	for _, selector := range globalSearchSelectors {
		c.logger.WithField("selector", selector).Debug("Trying global search selector")
		searchInput, err = c.page.Timeout(10 * time.Second).Element(selector)
		if err == nil && searchInput != nil {
			visible, _ := searchInput.Visible()
			if visible {
				c.logger.WithField("selector", selector).Info("Found global search input")
				break
			}
		}
		searchInput = nil
	}

	if searchInput == nil {
		return fmt.Errorf("could not find global search input after trying all selectors")
	}

	// Click and focus the search input
	c.logger.Debug("Clicking search input")
	searchInput.MustClick()
	time.Sleep(500 * time.Millisecond)
	c.stealth.ActionDelay()
	
	// Type the person's name
	c.logger.WithField("query", personName).Debug("Typing search query")
	err = c.stealth.HumanType(c.page, searchInput, personName)
	if err != nil {
		c.logger.Debug("Human typing failed, using direct input")
		searchInput.MustInput(personName)
	}

	c.stealth.ThinkingDelay()
	
	// Press Enter to search
	c.logger.Debug("Pressing Enter to search")
	c.page.Keyboard.Press(input.Enter)

	// Wait for search results to load
	c.logger.Debug("Waiting for search results")
	c.stealth.PageLoadDelay()
	time.Sleep(3 * time.Second)

	// Click on People filter to show only people
	c.logger.Debug("Looking for People filter")
	peopleFilterSelectors := []string{
		`button:has-text("People")`,
		`button[aria-label*="People"]`,
		`.search-reusables__filter-trigger:has-text("People")`,
		`a[href*="&network="][href*="search"]`,
		`.artdeco-pill:has-text("People")`,
		`[data-test-filter-button="People"]`,
	}

	for _, selector := range peopleFilterSelectors {
		filter, err := c.page.Timeout(3 * time.Second).Element(selector)
		if err == nil && filter != nil {
			c.logger.WithField("selector", selector).Debug("Found People filter, clicking")
			c.stealth.HoverElement(c.page, filter)
			filter.MustClick()
			c.stealth.PageLoadDelay()
			break
		}
	}

	return nil
}

// findAndClickProfile finds the matching profile card and clicks to open it
func (c *ConnectionManager) findAndClickProfile(personName string, institution string) error {
	c.logger.WithFields(map[string]interface{}{
		"name":        personName,
		"institution": institution,
	}).Info("Finding and clicking profile")

	// Wait for search results to load
	time.Sleep(2 * time.Second)

	// Scroll to load results
	c.stealth.HumanScroll(c.page, "down", 200)
	c.stealth.ActionDelay()

	// Look for profile cards with the person's name
	profileCardSelectors := []string{
		// Connections page result cards
		`.mn-connection-card`,
		`.entity-result`,
		`.search-result__info`,
		// General profile links
		`a[href*="/in/"]`,
		`.app-aware-link[href*="/in/"]`,
	}

	var profileCards []*rod.Element
	for _, selector := range profileCardSelectors {
		cards, err := c.page.Elements(selector)
		if err == nil && len(cards) > 0 {
			profileCards = cards
			c.logger.WithField("count", len(cards)).Debug("Found profile cards")
			break
		}
	}

	if len(profileCards) == 0 {
		return fmt.Errorf("no profile cards found")
	}

	// Find the matching profile
	nameLower := strings.ToLower(personName)
	nameParts := strings.Fields(nameLower)

	for _, card := range profileCards {
		// Get the card text content
		text, err := card.Text()
		if err != nil {
			continue
		}
		textLower := strings.ToLower(text)

		// Check if all name parts are present
		allPartsMatch := true
		for _, part := range nameParts {
			if !strings.Contains(textLower, part) {
				allPartsMatch = false
				break
			}
		}

		// Also check for institution if provided
		institutionMatch := institution == "" || strings.Contains(textLower, strings.ToLower(institution))

		if allPartsMatch && institutionMatch {
			c.logger.WithField("card_text", text[:min(100, len(text))]).Info("Found matching profile")

			// Find the clickable link within the card
			link, err := card.Element(`a[href*="/in/"]`)
			if err != nil {
				// Card itself might be the link
				link = card
			}

			// Human-like hover before click
			c.stealth.HoverElement(c.page, link)
			c.stealth.ThinkingDelay()

			// Click to open profile
			link.MustClick()

			// Wait for profile page to load
			c.stealth.PageLoadDelay()
			time.Sleep(2 * time.Second)

			// Verify we're on a profile page
			currentURL := c.page.MustInfo().URL
			c.logger.WithField("url", currentURL).Info("Navigated to profile")

			if strings.Contains(currentURL, "/in/") {
				c.logger.Info("Successfully opened profile page")
				return nil
			}
		}
	}

	// If exact match not found, try clicking the first result
	c.logger.Warn("Exact match not found, trying first result")
	if len(profileCards) > 0 {
		firstCard := profileCards[0]
		link, err := firstCard.Element(`a[href*="/in/"]`)
		if err != nil || link == nil {
			link = firstCard
		}

		if link != nil {
			c.stealth.HoverElement(c.page, link)
			c.stealth.ThinkingDelay()
			link.MustClick()
			c.stealth.PageLoadDelay()
			return nil
		}
	}

	// Try alternative approach - find any profile link on the page
	c.logger.Warn("No profile cards found, trying alternative approach")
	profileLinks, err := c.page.Elements(`a[href*="/in/"]`)
	if err == nil && len(profileLinks) > 0 {
		for _, link := range profileLinks {
			text, _ := link.Text()
			if text != "" && strings.Contains(strings.ToLower(text), strings.ToLower(personName)) {
				c.logger.WithField("link_text", text).Info("Found profile link")
				c.stealth.HoverElement(c.page, link)
				c.stealth.ThinkingDelay()
				link.MustClick()
				c.stealth.PageLoadDelay()
				return nil
			}
		}
		
		// Just click the first profile link if nothing matches
		c.logger.Warn("Clicking first available profile link")
		firstLink := profileLinks[0]
		c.stealth.HoverElement(c.page, firstLink)
		c.stealth.ThinkingDelay()
		firstLink.MustClick()
		c.stealth.PageLoadDelay()
		return nil
	}

	return fmt.Errorf("profile not found for: %s", personName)
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
