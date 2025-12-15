// LinkedIn Automation PoC - Main Application
// This is a proof-of-concept demonstrating browser automation techniques.
// FOR EDUCATIONAL PURPOSES ONLY - Do not use on production LinkedIn accounts.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/nikshitha/linkedin-automation-poc/auth"
	"github.com/nikshitha/linkedin-automation-poc/browser"
	"github.com/nikshitha/linkedin-automation-poc/config"
	"github.com/nikshitha/linkedin-automation-poc/connection"
	"github.com/nikshitha/linkedin-automation-poc/logger"
	"github.com/nikshitha/linkedin-automation-poc/messaging"
	"github.com/nikshitha/linkedin-automation-poc/search"
	"github.com/nikshitha/linkedin-automation-poc/stealth"
	"github.com/nikshitha/linkedin-automation-poc/storage"
)

// Application holds all components of the automation tool
type Application struct {
	config      *config.Config
	logger      *logger.Logger
	browser     *browser.Browser
	stealth     *stealth.StealthManager
	rateLimiter *stealth.RateLimiter
	scheduler   *stealth.Scheduler
	db          *storage.Database
	auth        *auth.Authenticator
	searcher    *search.Searcher
	connector   *connection.ConnectionManager
	messenger   *messaging.MessagingManager
}

// Command line flags
var (
	configPath  = flag.String("config", "config.yaml", "Path to configuration file")
	mode        = flag.String("mode", "interactive", "Run mode: interactive, search, connect, message, full")
	searchQuery = flag.String("search", "", "Search query (job title, keywords)")
	company     = flag.String("company", "", "Company filter for search")
	location    = flag.String("location", "", "Location filter for search")
	maxResults  = flag.Int("max-results", 25, "Maximum search results")
	dryRun      = flag.Bool("dry-run", false, "Dry run mode - no actual actions")
	verbose     = flag.Bool("verbose", false, "Enable verbose logging")
)

func main() {
	flag.Parse()

	// Print banner
	printBanner()

	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		fmt.Println("Note: No .env file found, using environment variables")
	}

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		fmt.Println("\nPlease ensure you have set LINKEDIN_EMAIL and LINKEDIN_PASSWORD environment variables")
		fmt.Println("or create a .env file with these values.")
		os.Exit(1)
	}

	// Override log level if verbose
	if *verbose {
		cfg.Logging.Level = "debug"
	}

	// Initialize logger
	log, err := logger.New(logger.Config{
		Level:      cfg.Logging.Level,
		Format:     cfg.Logging.Format,
		OutputFile: cfg.Logging.OutputFile,
	})
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	log.Info("LinkedIn Automation PoC starting...")
	log.Infof("Mode: %s", *mode)

	// Create application
	app, err := NewApplication(cfg, log)
	if err != nil {
		log.Errorf("Failed to initialize application: %v", err)
		os.Exit(1)
	}

	// Handle graceful shutdown
	setupGracefulShutdown(app)

	// Run the application
	if err := app.Run(); err != nil {
		log.Errorf("Application error: %v", err)
		os.Exit(1)
	}

	log.Info("Application completed successfully")
}

// NewApplication creates and initializes a new application instance
func NewApplication(cfg *config.Config, log *logger.Logger) (*Application, error) {
	// Initialize database
	db, err := storage.NewDatabase(cfg.Storage.DatabasePath, log)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Initialize stealth manager
	stealthMgr := stealth.NewStealthManager(&cfg.Stealth, log)

	// Initialize rate limiter
	rateLimiter := stealth.NewRateLimiter(&cfg.RateLimits, log)

	// Initialize scheduler
	scheduler := stealth.NewScheduler(&cfg.Schedule, log)

	// Initialize browser
	browserMgr := browser.NewBrowser(cfg, log, stealthMgr)

	// Initialize authenticator
	authMgr := auth.NewAuthenticator(cfg, log, stealthMgr, db)

	// Initialize searcher
	searchMgr := search.NewSearcher(cfg, log, stealthMgr, rateLimiter, db)

	// Initialize connection manager
	connMgr := connection.NewConnectionManager(cfg, log, stealthMgr, rateLimiter, db)

	// Initialize messaging manager
	msgMgr := messaging.NewMessagingManager(cfg, log, stealthMgr, rateLimiter, db)

	return &Application{
		config:      cfg,
		logger:      log,
		browser:     browserMgr,
		stealth:     stealthMgr,
		rateLimiter: rateLimiter,
		scheduler:   scheduler,
		db:          db,
		auth:        authMgr,
		searcher:    searchMgr,
		connector:   connMgr,
		messenger:   msgMgr,
	}, nil
}

// Run executes the application based on the selected mode
func (app *Application) Run() error {
	// Check operating hours if scheduling is enabled
	if app.config.Schedule.Enabled {
		app.scheduler.WaitForOperatingHours()
	}

	// Launch browser
	if err := app.browser.Launch(); err != nil {
		return fmt.Errorf("failed to launch browser: %w", err)
	}
	defer app.Close()

	// Set page references for all managers
	page := app.browser.GetPage()
	app.auth.SetBrowser(app.browser.GetBrowser())
	app.auth.SetPage(page)
	app.searcher.SetPage(page)
	app.connector.SetPage(page)
	app.messenger.SetPage(page)

	// Authenticate
	app.logger.Info("Authenticating with LinkedIn...")
	if err := app.auth.Login(); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}
	app.logger.Info("Authentication successful!")

	// Log current user info
	if user, err := app.auth.GetCurrentUser(); err == nil {
		app.logger.Infof("Logged in as: %s", user["name"])
	}

	// Show daily stats
	app.showDailyStats()

	// Execute based on mode
	switch *mode {
	case "interactive":
		return app.runInteractiveMode()
	case "search":
		return app.runSearchMode()
	case "connect":
		return app.runConnectMode()
	case "message":
		return app.runMessageMode()
	case "full":
		return app.runFullWorkflow()
	default:
		return fmt.Errorf("unknown mode: %s", *mode)
	}
}

// runInteractiveMode runs an interactive session
func (app *Application) runInteractiveMode() error {
	app.logger.Info("Running in interactive mode")
	app.logger.Info("Browser is open. You can interact manually or close to exit.")
	app.logger.Info("Press Ctrl+C to exit")

	// Keep running until interrupted
	select {}
}

// runSearchMode runs search-only mode
func (app *Application) runSearchMode() error {
	app.logger.Info("Running in search mode")

	query := *searchQuery
	if query == "" {
		query = app.config.Search.DefaultJobTitle
	}

	if query == "" {
		return fmt.Errorf("no search query provided (use -search flag or set in config)")
	}

	params := search.SearchParams{
		JobTitle:   query,
		Company:    *company,
		Location:   *location,
		Keywords:   app.config.Search.Keywords,
		MaxResults: *maxResults,
	}

	results, err := app.searcher.Search(params)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	app.logger.Infof("Found %d profiles", len(results))

	// Save profiles to database
	for _, result := range results {
		app.searcher.SaveProfile(result)
		app.logger.Infof("  - %s (%s) - %s", result.Name, result.Connection, result.ProfileURL)
	}

	return nil
}

// runConnectMode runs connection-only mode
func (app *Application) runConnectMode() error {
	app.logger.Info("Running in connect mode")

	// First, do a search
	if err := app.runSearchMode(); err != nil {
		return err
	}

	// Get profiles that haven't been connected
	profiles, err := app.db.GetAllProfiles()
	if err != nil {
		return fmt.Errorf("failed to get profiles: %w", err)
	}

	var toConnect []*search.SearchResult
	for _, p := range profiles {
		hasSent, _ := app.db.HasSentConnectionRequest(p.ProfileURL)
		if !hasSent {
			toConnect = append(toConnect, &search.SearchResult{
				ProfileURL: p.ProfileURL,
				Name:       p.Name,
				FirstName:  p.FirstName,
				LastName:   p.LastName,
				Headline:   p.Headline,
				Company:    p.Company,
				Location:   p.Location,
				Connection: p.ConnectionDegree,
			})
		}
	}

	if len(toConnect) == 0 {
		app.logger.Info("No new profiles to connect with")
		return nil
	}

	app.logger.Infof("Sending connection requests to %d profiles", len(toConnect))

	if *dryRun {
		app.logger.Info("Dry run mode - skipping actual connections")
		return nil
	}

	sent, failed, err := app.connector.SendBulkConnectionRequests(toConnect, "")
	if err != nil {
		return err
	}

	app.logger.Infof("Connection requests: %d sent, %d failed", sent, failed)
	return nil
}

// runMessageMode runs messaging-only mode
func (app *Application) runMessageMode() error {
	app.logger.Info("Running in message mode")

	if *dryRun {
		app.logger.Info("Dry run mode - skipping actual messages")
		return nil
	}

	return app.messenger.ProcessNewConnectionsWorkflow()
}

// runFullWorkflow runs the complete automation workflow
func (app *Application) runFullWorkflow() error {
	app.logger.Info("Running full workflow")
	sessionStart := time.Now()

	for {
		// Check if we should take a break
		if app.scheduler.ShouldTakeBreak(sessionStart) {
			app.scheduler.TakeBreak()
			sessionStart = time.Now()
		}

		// Check operating hours
		if !app.scheduler.IsWithinOperatingHours() {
			app.logger.Info("Outside operating hours, waiting...")
			app.scheduler.WaitForOperatingHours()
		}

		// 1. Check for newly accepted connections and send follow-ups
		app.logger.Info("Step 1: Processing new connections...")
		if err := app.messenger.ProcessNewConnectionsWorkflow(); err != nil {
			app.logger.WithError(err).Warn("Failed to process new connections")
		}

		// 2. Search for new profiles
		app.logger.Info("Step 2: Searching for new profiles...")
		if err := app.runSearchMode(); err != nil {
			app.logger.WithError(err).Warn("Search failed")
		}

		// 3. Send connection requests
		if app.connector.GetRemainingConnections() > 0 {
			app.logger.Info("Step 3: Sending connection requests...")
			if err := app.runConnectMode(); err != nil {
				app.logger.WithError(err).Warn("Failed to send connections")
			}
		} else {
			app.logger.Info("Daily connection limit reached")
		}

		// Show stats
		app.showDailyStats()

		// Cooldown before next cycle
		app.logger.Info("Workflow cycle complete. Starting cooldown...")
		app.rateLimiter.EnforceCooldown()
	}
}

// showDailyStats displays today's activity statistics
func (app *Application) showDailyStats() {
	stats, err := app.db.GetTodayStats()
	if err != nil {
		app.logger.WithError(err).Warn("Failed to get daily stats")
		return
	}

	app.logger.Info("=== Today's Activity ===")
	app.logger.Infof("  Connections Sent: %d / %d", stats.ConnectionsSent, app.config.RateLimits.MaxConnectionsPerDay)
	app.logger.Infof("  Connections Accepted: %d", stats.ConnectionsAccepted)
	app.logger.Infof("  Messages Sent: %d / %d", stats.MessagesSent, app.config.RateLimits.MaxMessagesPerDay)
	app.logger.Infof("  Profiles Viewed: %d / %d", stats.ProfilesViewed, app.config.RateLimits.MaxProfileViewsPerDay)
	app.logger.Infof("  Searches: %d", stats.SearchesPerformed)
	app.logger.Info("========================")
}

// Close cleans up application resources
func (app *Application) Close() {
	app.logger.Info("Shutting down...")

	if app.browser != nil {
		app.browser.Close()
	}

	if app.db != nil {
		app.db.Close()
	}

	app.logger.Info("Cleanup complete")
}

// setupGracefulShutdown handles OS signals for graceful shutdown
func setupGracefulShutdown(app *Application) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		app.logger.Infof("Received signal: %v", sig)
		app.Close()
		os.Exit(0)
	}()
}

// printBanner prints the application banner
func printBanner() {
	banner := `
╔══════════════════════════════════════════════════════════════════╗
║           LinkedIn Automation PoC - Educational Only             ║
╠══════════════════════════════════════════════════════════════════╣
║  ⚠️  WARNING: This tool is for EDUCATIONAL PURPOSES ONLY         ║
║  ⚠️  Using automation on LinkedIn violates their ToS             ║
║  ⚠️  Do NOT use this on production accounts                      ║
╚══════════════════════════════════════════════════════════════════╝
`
	fmt.Println(banner)
}
