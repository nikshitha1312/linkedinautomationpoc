// Package search provides LinkedIn user search and profile collection functionality.
// It handles search queries, pagination, and duplicate detection.
package search

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/nikshitha/linkedin-automation-poc/config"
	"github.com/nikshitha/linkedin-automation-poc/logger"
	"github.com/nikshitha/linkedin-automation-poc/stealth"
	"github.com/nikshitha/linkedin-automation-poc/storage"
)

// LinkedIn search URLs
const (
	LinkedInPeopleSearchURL = "https://www.linkedin.com/search/results/people/"
)

// SearchParams holds search parameters
type SearchParams struct {
	JobTitle  string   `json:"job_title"`
	Company   string   `json:"company"`
	Location  string   `json:"location"`
	Keywords  []string `json:"keywords"`
	Network   []string `json:"network"` // 1st, 2nd, 3rd+
	MaxResults int     `json:"max_results"`
}

// SearchResult represents a single search result
type SearchResult struct {
	ProfileURL   string `json:"profile_url"`
	Name         string `json:"name"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Headline     string `json:"headline"`
	Company      string `json:"company"`
	Location     string `json:"location"`
	Connection   string `json:"connection"` // 1st, 2nd, 3rd+
	MutualConns  int    `json:"mutual_connections"`
}

// Searcher handles LinkedIn search operations
type Searcher struct {
	config      *config.Config
	logger      *logger.Logger
	stealth     *stealth.StealthManager
	rateLimiter *stealth.RateLimiter
	db          *storage.Database
	page        *rod.Page
	seenProfiles map[string]bool // For duplicate detection
}

// NewSearcher creates a new searcher
func NewSearcher(cfg *config.Config, log *logger.Logger, s *stealth.StealthManager, rl *stealth.RateLimiter, db *storage.Database) *Searcher {
	return &Searcher{
		config:       cfg,
		logger:       log.WithModule("search"),
		stealth:      s,
		rateLimiter:  rl,
		db:           db,
		seenProfiles: make(map[string]bool),
	}
}

// SetPage sets the page instance
func (s *Searcher) SetPage(page *rod.Page) {
	s.page = page
}

// Search performs a LinkedIn people search with the given parameters
func (s *Searcher) Search(params SearchParams) ([]*SearchResult, error) {
	s.logger.WithFields(map[string]interface{}{
		"job_title":   params.JobTitle,
		"company":     params.Company,
		"location":    params.Location,
		"keywords":    params.Keywords,
		"max_results": params.MaxResults,
	}).Info("Starting search")

	// Check rate limits
	if !s.rateLimiter.CanPerformAction("search") {
		return nil, fmt.Errorf("search rate limit reached")
	}

	// Build search URL
	searchURL := s.buildSearchURL(params)
	s.logger.WithField("url", searchURL).Debug("Search URL built")

	// Navigate to search page
	err := s.page.Navigate(searchURL)
	if err != nil {
		return nil, fmt.Errorf("failed to navigate to search page: %w", err)
	}

	s.stealth.PageLoadDelay()
	err = s.page.WaitLoad()
	if err != nil {
		return nil, fmt.Errorf("failed to load search page: %w", err)
	}

	// Apply fingerprint masking
	s.stealth.ApplyFingerprintMasking(s.page)

	// Random behavior before parsing results
	s.stealth.RandomMouseWander(s.page)
	s.stealth.ThinkingDelay()

	// Collect results with pagination
	results, err := s.collectResults(params.MaxResults)
	if err != nil {
		return nil, fmt.Errorf("failed to collect results: %w", err)
	}

	// Record action
	s.rateLimiter.RecordAction("search")

	// Save search history
	s.db.SaveSearchHistory(
		searchURL,
		params.JobTitle,
		params.Company,
		params.Location,
		params.Keywords,
		len(results),
	)

	s.logger.Infof("Search completed, found %d unique profiles", len(results))
	return results, nil
}

// buildSearchURL constructs the LinkedIn search URL with parameters
func (s *Searcher) buildSearchURL(params SearchParams) string {
	baseURL := LinkedInPeopleSearchURL
	queryParams := url.Values{}

	// Build keywords from all search terms
	var keywords []string
	if params.JobTitle != "" {
		keywords = append(keywords, params.JobTitle)
	}
	if len(params.Keywords) > 0 {
		keywords = append(keywords, params.Keywords...)
	}

	if len(keywords) > 0 {
		queryParams.Set("keywords", strings.Join(keywords, " "))
	}

	// Company filter
	if params.Company != "" {
		// LinkedIn uses company IDs, but we can use keyword search
		// For proper filtering, would need to resolve company to ID
		if queryParams.Get("keywords") != "" {
			queryParams.Set("keywords", queryParams.Get("keywords")+" "+params.Company)
		} else {
			queryParams.Set("keywords", params.Company)
		}
	}

	// Location filter
	if params.Location != "" {
		// LinkedIn uses geoUrn for location filtering
		// For simplicity, adding to keywords
		if queryParams.Get("keywords") != "" {
			queryParams.Set("keywords", queryParams.Get("keywords")+" "+params.Location)
		} else {
			queryParams.Set("keywords", params.Location)
		}
	}

	// Network filter (connection degree)
	if len(params.Network) > 0 {
		// LinkedIn uses network=["F","S","O"] for 1st, 2nd, 3rd+
		networkFilter := []string{}
		for _, n := range params.Network {
			switch n {
			case "1st", "1":
				networkFilter = append(networkFilter, "F")
			case "2nd", "2":
				networkFilter = append(networkFilter, "S")
			case "3rd", "3", "3rd+":
				networkFilter = append(networkFilter, "O")
			}
		}
		if len(networkFilter) > 0 {
			queryParams.Set("network", `["`+strings.Join(networkFilter, `","`)+`"]`)
		}
	}

	// Origin parameter
	queryParams.Set("origin", "GLOBAL_SEARCH_HEADER")

	if len(queryParams) > 0 {
		return baseURL + "?" + queryParams.Encode()
	}
	return baseURL
}

// collectResults collects search results with pagination
func (s *Searcher) collectResults(maxResults int) ([]*SearchResult, error) {
	var allResults []*SearchResult
	currentPage := 1
	// LinkedIn typically shows 10 results per page

	for len(allResults) < maxResults {
		s.logger.WithField("page", currentPage).Debug("Processing search results page")

		// Wait for results to load
		err := s.waitForResults()
		if err != nil {
			s.logger.WithError(err).Warn("Failed to wait for results")
			break
		}

		// Parse results on current page
		pageResults, err := s.parseSearchResults()
		if err != nil {
			s.logger.WithError(err).Warn("Failed to parse results")
			break
		}

		if len(pageResults) == 0 {
			s.logger.Debug("No more results found")
			break
		}

		// Filter duplicates
		for _, result := range pageResults {
			if !s.isDuplicate(result.ProfileURL) {
				allResults = append(allResults, result)
				s.markAsSeen(result.ProfileURL)

				if len(allResults) >= maxResults {
					break
				}
			}
		}

		s.logger.Infof("Collected %d profiles so far", len(allResults))

		// Check if we have enough
		if len(allResults) >= maxResults {
			break
		}

		// Try to go to next page
		hasNextPage, err := s.goToNextPage()
		if err != nil || !hasNextPage {
			s.logger.Debug("No more pages available")
			break
		}

		currentPage++

		// Rate limiting between pages
		s.rateLimiter.WaitForNextAction()

		// Natural scrolling and delay
		s.stealth.HumanScroll(s.page, "up", 200) // Scroll back up
		s.stealth.ThinkingDelay()
	}

	return allResults, nil
}

// waitForResults waits for search results to load
func (s *Searcher) waitForResults() error {
	// Wait for search results container
	_, err := s.page.Timeout(10 * time.Second).Element(".search-results-container, .reusable-search__entity-result-list")
	if err != nil {
		// Try alternative selectors
		_, err = s.page.Timeout(5 * time.Second).Element("[data-chameleon-result-urn]")
		if err != nil {
			return fmt.Errorf("search results not found: %w", err)
		}
	}

	// Small delay for full render
	time.Sleep(500 * time.Millisecond)
	return nil
}

// parseSearchResults parses the current page of search results
func (s *Searcher) parseSearchResults() ([]*SearchResult, error) {
	var results []*SearchResult

	// Find all result cards
	resultCards, err := s.page.Elements(".reusable-search__result-container, [data-chameleon-result-urn], .entity-result")
	if err != nil {
		return nil, fmt.Errorf("failed to find result cards: %w", err)
	}

	s.logger.Debugf("Found %d result cards on page", len(resultCards))

	for i, card := range resultCards {
		result, err := s.parseResultCard(card)
		if err != nil {
			s.logger.WithError(err).Debugf("Failed to parse result card %d", i)
			continue
		}

		if result != nil && result.ProfileURL != "" {
			results = append(results, result)

			// Scroll to card to simulate reading
			if i%3 == 0 {
				s.stealth.HumanScroll(s.page, "down", 100+i*20)
			}
		}
	}

	return results, nil
}

// parseResultCard extracts profile information from a result card
func (s *Searcher) parseResultCard(card *rod.Element) (*SearchResult, error) {
	result := &SearchResult{}

	// Get profile URL
	linkEl, err := card.Element("a.app-aware-link[href*='/in/']")
	if err != nil {
		// Try alternative selectors
		linkEl, err = card.Element("span.entity-result__title-text a")
		if err != nil {
			return nil, fmt.Errorf("profile link not found")
		}
	}

	href, err := linkEl.Attribute("href")
	if err != nil || href == nil {
		return nil, fmt.Errorf("failed to get profile URL")
	}

	// Clean up URL (remove tracking parameters)
	result.ProfileURL = s.cleanProfileURL(*href)

	// Get name
	nameEl, err := card.Element(".entity-result__title-text a span[aria-hidden='true'], .entity-result__title-line span[dir='ltr']")
	if err == nil && nameEl != nil {
		name, _ := nameEl.Text()
		result.Name = strings.TrimSpace(name)
		result.FirstName, result.LastName = s.splitName(result.Name)
	}

	// Get headline (primary subtitle)
	headlineEl, err := card.Element(".entity-result__primary-subtitle, .entity-result__summary")
	if err == nil && headlineEl != nil {
		headline, _ := headlineEl.Text()
		result.Headline = strings.TrimSpace(headline)
		// Try to extract company from headline
		result.Company = s.extractCompanyFromHeadline(result.Headline)
	}

	// Get location (secondary subtitle)
	locationEl, err := card.Element(".entity-result__secondary-subtitle")
	if err == nil && locationEl != nil {
		location, _ := locationEl.Text()
		result.Location = strings.TrimSpace(location)
	}

	// Get connection degree
	connectionEl, err := card.Element(".entity-result__badge-text, .distance-badge")
	if err == nil && connectionEl != nil {
		connection, _ := connectionEl.Text()
		result.Connection = strings.TrimSpace(connection)
	}

	// Get mutual connections count
	mutualEl, err := card.Element(".member-insights__connection-count, .reusable-search-simple-insight__text")
	if err == nil && mutualEl != nil {
		mutualText, _ := mutualEl.Text()
		result.MutualConns = s.extractMutualCount(mutualText)
	}

	return result, nil
}

// cleanProfileURL removes tracking parameters from profile URL
func (s *Searcher) cleanProfileURL(rawURL string) string {
	// Parse URL
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	// Remove query parameters
	parsed.RawQuery = ""
	parsed.Fragment = ""

	// Ensure it's a valid LinkedIn profile URL
	cleanURL := parsed.String()
	if !strings.HasPrefix(cleanURL, "https://www.linkedin.com/in/") {
		if strings.Contains(cleanURL, "/in/") {
			// Fix relative URLs
			parts := strings.Split(cleanURL, "/in/")
			if len(parts) > 1 {
				profileSlug := strings.Split(parts[1], "/")[0]
				cleanURL = "https://www.linkedin.com/in/" + profileSlug + "/"
			}
		}
	}

	// Ensure trailing slash
	if !strings.HasSuffix(cleanURL, "/") {
		cleanURL += "/"
	}

	return cleanURL
}

// splitName splits a full name into first and last name
func (s *Searcher) splitName(fullName string) (string, string) {
	parts := strings.Fields(fullName)
	if len(parts) == 0 {
		return "", ""
	}
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], strings.Join(parts[1:], " ")
}

// extractCompanyFromHeadline tries to extract company name from headline
func (s *Searcher) extractCompanyFromHeadline(headline string) string {
	// Common patterns: "Role at Company", "Role | Company"
	patterns := []string{
		` at `,
		` @ `,
		` | `,
		` - `,
	}

	for _, pattern := range patterns {
		if idx := strings.LastIndex(headline, pattern); idx != -1 {
			company := strings.TrimSpace(headline[idx+len(pattern):])
			// Clean up company name
			if pipeIdx := strings.Index(company, "|"); pipeIdx != -1 {
				company = strings.TrimSpace(company[:pipeIdx])
			}
			return company
		}
	}

	return ""
}

// extractMutualCount extracts the number of mutual connections from text
func (s *Searcher) extractMutualCount(text string) int {
	// Look for number in text like "5 mutual connections"
	re := regexp.MustCompile(`(\d+)\s*mutual`)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		var count int
		fmt.Sscanf(matches[1], "%d", &count)
		return count
	}
	return 0
}

// goToNextPage attempts to navigate to the next page of results
func (s *Searcher) goToNextPage() (bool, error) {
	// Scroll to bottom to ensure pagination is visible
	s.stealth.HumanScroll(s.page, "down", 500)
	time.Sleep(500 * time.Millisecond)

	// Look for next page button
	nextButton, err := s.page.Element("button.artdeco-pagination__button--next:not([disabled])")
	if err != nil {
		// Try alternative selector
		nextButton, err = s.page.Element("button[aria-label='Next']:not([disabled])")
		if err != nil {
			return false, nil
		}
	}

	// Check if button is disabled
	disabled, _ := nextButton.Attribute("disabled")
	if disabled != nil {
		return false, nil
	}

	// Human-like click on next button
	err = s.stealth.ClickElement(s.page, nextButton)
	if err != nil {
		return false, fmt.Errorf("failed to click next button: %w", err)
	}

	// Wait for page to load
	s.stealth.PageLoadDelay()
	time.Sleep(time.Second)

	return true, nil
}

// isDuplicate checks if a profile URL has already been seen
func (s *Searcher) isDuplicate(profileURL string) bool {
	// Check in-memory cache
	if s.seenProfiles[profileURL] {
		return true
	}

	// Check database
	exists, err := s.db.ProfileExists(profileURL)
	if err == nil && exists {
		s.seenProfiles[profileURL] = true
		return true
	}

	// Check if connection request was already sent
	hasSent, err := s.db.HasSentConnectionRequest(profileURL)
	if err == nil && hasSent {
		s.seenProfiles[profileURL] = true
		return true
	}

	return false
}

// markAsSeen marks a profile URL as seen
func (s *Searcher) markAsSeen(profileURL string) {
	s.seenProfiles[profileURL] = true
}

// SaveProfile saves a search result as a profile in the database
func (s *Searcher) SaveProfile(result *SearchResult) (int64, error) {
	profile := &storage.Profile{
		ProfileURL:       result.ProfileURL,
		Name:             result.Name,
		FirstName:        result.FirstName,
		LastName:         result.LastName,
		Headline:         result.Headline,
		Company:          result.Company,
		Location:         result.Location,
		ConnectionDegree: result.Connection,
	}

	return s.db.SaveProfile(profile)
}

// ClearSeenProfiles clears the in-memory seen profiles cache
func (s *Searcher) ClearSeenProfiles() {
	s.seenProfiles = make(map[string]bool)
}
