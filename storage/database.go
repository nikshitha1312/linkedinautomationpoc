// Package storage provides data persistence using SQLite for the LinkedIn automation tool.
// It tracks connection requests, messages, profiles, and session state.
package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
	"github.com/nikshitha/linkedin-automation-poc/logger"
)

// Database wraps SQLite database operations
type Database struct {
	db     *sql.DB
	logger *logger.Logger
}

// Profile represents a LinkedIn profile
type Profile struct {
	ID          int64     `json:"id"`
	ProfileURL  string    `json:"profile_url"`
	Name        string    `json:"name"`
	FirstName   string    `json:"first_name"`
	LastName    string    `json:"last_name"`
	Headline    string    `json:"headline"`
	Company     string    `json:"company"`
	Location    string    `json:"location"`
	ConnectionDegree string `json:"connection_degree"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ConnectionRequest represents a connection request record
type ConnectionRequest struct {
	ID          int64     `json:"id"`
	ProfileID   int64     `json:"profile_id"`
	ProfileURL  string    `json:"profile_url"`
	Note        string    `json:"note"`
	Status      string    `json:"status"` // pending, accepted, declined
	SentAt      time.Time `json:"sent_at"`
	AcceptedAt  *time.Time `json:"accepted_at,omitempty"`
}

// Message represents a sent message record
type Message struct {
	ID          int64     `json:"id"`
	ProfileID   int64     `json:"profile_id"`
	ProfileURL  string    `json:"profile_url"`
	Content     string    `json:"content"`
	Template    string    `json:"template"`
	MessageType string    `json:"message_type"` // connection_note, follow_up, direct
	SentAt      time.Time `json:"sent_at"`
}

// DailyStats tracks daily activity statistics
type DailyStats struct {
	Date              string `json:"date"`
	ConnectionsSent   int    `json:"connections_sent"`
	ConnectionsAccepted int  `json:"connections_accepted"`
	MessagesSent      int    `json:"messages_sent"`
	ProfilesViewed    int    `json:"profiles_viewed"`
	SearchesPerformed int    `json:"searches_performed"`
}

// SessionCookie represents a stored browser cookie
type SessionCookie struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Domain   string `json:"domain"`
	Path     string `json:"path"`
	Expires  int64  `json:"expires"`
	HTTPOnly bool   `json:"http_only"`
	Secure   bool   `json:"secure"`
}

// NewDatabase creates a new database connection
func NewDatabase(dbPath string, log *logger.Logger) (*Database, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	database := &Database{
		db:     db,
		logger: log.WithModule("storage"),
	}

	// Initialize schema
	if err := database.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	database.logger.Info("Database initialized successfully")
	return database, nil
}

// initSchema creates the database tables if they don't exist
func (d *Database) initSchema() error {
	schema := `
	-- Profiles table
	CREATE TABLE IF NOT EXISTS profiles (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		profile_url TEXT UNIQUE NOT NULL,
		name TEXT,
		first_name TEXT,
		last_name TEXT,
		headline TEXT,
		company TEXT,
		location TEXT,
		connection_degree TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Connection requests table
	CREATE TABLE IF NOT EXISTS connection_requests (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		profile_id INTEGER,
		profile_url TEXT NOT NULL,
		note TEXT,
		status TEXT DEFAULT 'pending',
		sent_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		accepted_at DATETIME,
		FOREIGN KEY (profile_id) REFERENCES profiles(id)
	);

	-- Messages table
	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		profile_id INTEGER,
		profile_url TEXT NOT NULL,
		content TEXT NOT NULL,
		template TEXT,
		message_type TEXT DEFAULT 'direct',
		sent_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (profile_id) REFERENCES profiles(id)
	);

	-- Daily stats table
	CREATE TABLE IF NOT EXISTS daily_stats (
		date TEXT PRIMARY KEY,
		connections_sent INTEGER DEFAULT 0,
		connections_accepted INTEGER DEFAULT 0,
		messages_sent INTEGER DEFAULT 0,
		profiles_viewed INTEGER DEFAULT 0,
		searches_performed INTEGER DEFAULT 0
	);

	-- Session cookies table
	CREATE TABLE IF NOT EXISTS session_cookies (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		value TEXT NOT NULL,
		domain TEXT,
		path TEXT,
		expires INTEGER,
		http_only BOOLEAN,
		secure BOOLEAN,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Search history table
	CREATE TABLE IF NOT EXISTS search_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		query TEXT NOT NULL,
		job_title TEXT,
		company TEXT,
		location TEXT,
		keywords TEXT,
		results_count INTEGER,
		searched_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Create indexes
	CREATE INDEX IF NOT EXISTS idx_profiles_url ON profiles(profile_url);
	CREATE INDEX IF NOT EXISTS idx_connection_requests_status ON connection_requests(status);
	CREATE INDEX IF NOT EXISTS idx_connection_requests_sent_at ON connection_requests(sent_at);
	CREATE INDEX IF NOT EXISTS idx_messages_sent_at ON messages(sent_at);
	`

	_, err := d.db.Exec(schema)
	return err
}

// Close closes the database connection
func (d *Database) Close() error {
	return d.db.Close()
}

// ==============================================================================
// Profile Operations
// ==============================================================================

// SaveProfile saves or updates a profile
func (d *Database) SaveProfile(profile *Profile) (int64, error) {
	query := `
		INSERT INTO profiles (profile_url, name, first_name, last_name, headline, company, location, connection_degree)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(profile_url) DO UPDATE SET
			name = excluded.name,
			first_name = excluded.first_name,
			last_name = excluded.last_name,
			headline = excluded.headline,
			company = excluded.company,
			location = excluded.location,
			connection_degree = excluded.connection_degree,
			updated_at = CURRENT_TIMESTAMP
		RETURNING id
	`

	var id int64
	err := d.db.QueryRow(query,
		profile.ProfileURL, profile.Name, profile.FirstName, profile.LastName,
		profile.Headline, profile.Company, profile.Location, profile.ConnectionDegree,
	).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("failed to save profile: %w", err)
	}

	d.logger.WithField("profile_url", profile.ProfileURL).Debug("Profile saved")
	return id, nil
}

// GetProfile retrieves a profile by URL
func (d *Database) GetProfile(profileURL string) (*Profile, error) {
	query := `SELECT id, profile_url, name, first_name, last_name, headline, company, location, connection_degree, created_at, updated_at FROM profiles WHERE profile_url = ?`

	profile := &Profile{}
	err := d.db.QueryRow(query, profileURL).Scan(
		&profile.ID, &profile.ProfileURL, &profile.Name, &profile.FirstName, &profile.LastName,
		&profile.Headline, &profile.Company, &profile.Location, &profile.ConnectionDegree,
		&profile.CreatedAt, &profile.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}

	return profile, nil
}

// ProfileExists checks if a profile URL already exists
func (d *Database) ProfileExists(profileURL string) (bool, error) {
	query := `SELECT COUNT(*) FROM profiles WHERE profile_url = ?`
	var count int
	err := d.db.QueryRow(query, profileURL).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetAllProfiles retrieves all profiles
func (d *Database) GetAllProfiles() ([]*Profile, error) {
	query := `SELECT id, profile_url, name, first_name, last_name, headline, company, location, connection_degree, created_at, updated_at FROM profiles ORDER BY created_at DESC`

	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var profiles []*Profile
	for rows.Next() {
		profile := &Profile{}
		err := rows.Scan(
			&profile.ID, &profile.ProfileURL, &profile.Name, &profile.FirstName, &profile.LastName,
			&profile.Headline, &profile.Company, &profile.Location, &profile.ConnectionDegree,
			&profile.CreatedAt, &profile.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, profile)
	}

	return profiles, nil
}

// ==============================================================================
// Connection Request Operations
// ==============================================================================

// SaveConnectionRequest saves a connection request
func (d *Database) SaveConnectionRequest(request *ConnectionRequest) (int64, error) {
	query := `
		INSERT INTO connection_requests (profile_id, profile_url, note, status, sent_at)
		VALUES (?, ?, ?, ?, ?)
	`

	result, err := d.db.Exec(query,
		request.ProfileID, request.ProfileURL, request.Note, request.Status, time.Now(),
	)
	if err != nil {
		return 0, fmt.Errorf("failed to save connection request: %w", err)
	}

	id, _ := result.LastInsertId()
	d.logger.WithField("profile_url", request.ProfileURL).Info("Connection request saved")

	// Update daily stats
	d.incrementDailyStat("connections_sent")

	return id, nil
}

// HasSentConnectionRequest checks if a connection request was already sent
func (d *Database) HasSentConnectionRequest(profileURL string) (bool, error) {
	query := `SELECT COUNT(*) FROM connection_requests WHERE profile_url = ?`
	var count int
	err := d.db.QueryRow(query, profileURL).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetPendingConnectionRequests gets all pending connection requests
func (d *Database) GetPendingConnectionRequests() ([]*ConnectionRequest, error) {
	query := `
		SELECT id, profile_id, profile_url, note, status, sent_at, accepted_at
		FROM connection_requests WHERE status = 'pending'
		ORDER BY sent_at DESC
	`

	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []*ConnectionRequest
	for rows.Next() {
		req := &ConnectionRequest{}
		err := rows.Scan(&req.ID, &req.ProfileID, &req.ProfileURL, &req.Note, &req.Status, &req.SentAt, &req.AcceptedAt)
		if err != nil {
			return nil, err
		}
		requests = append(requests, req)
	}

	return requests, nil
}

// UpdateConnectionStatus updates the status of a connection request
func (d *Database) UpdateConnectionStatus(profileURL string, status string) error {
	query := `UPDATE connection_requests SET status = ?, accepted_at = ? WHERE profile_url = ?`

	var acceptedAt interface{}
	if status == "accepted" {
		now := time.Now()
		acceptedAt = &now
		d.incrementDailyStat("connections_accepted")
	}

	_, err := d.db.Exec(query, status, acceptedAt, profileURL)
	if err != nil {
		return fmt.Errorf("failed to update connection status: %w", err)
	}

	d.logger.WithFields(map[string]interface{}{
		"profile_url": profileURL,
		"status":      status,
	}).Info("Connection status updated")

	return nil
}

// GetTodayConnectionCount returns the number of connections sent today
func (d *Database) GetTodayConnectionCount() (int, error) {
	query := `SELECT COUNT(*) FROM connection_requests WHERE DATE(sent_at) = DATE('now')`
	var count int
	err := d.db.QueryRow(query).Scan(&count)
	return count, err
}

// GetRecentlyAcceptedConnections gets connections accepted in the last N days
func (d *Database) GetRecentlyAcceptedConnections(days int) ([]*ConnectionRequest, error) {
	query := `
		SELECT id, profile_id, profile_url, note, status, sent_at, accepted_at
		FROM connection_requests 
		WHERE status = 'accepted' AND accepted_at > datetime('now', ?)
		ORDER BY accepted_at DESC
	`

	rows, err := d.db.Query(query, fmt.Sprintf("-%d days", days))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []*ConnectionRequest
	for rows.Next() {
		req := &ConnectionRequest{}
		err := rows.Scan(&req.ID, &req.ProfileID, &req.ProfileURL, &req.Note, &req.Status, &req.SentAt, &req.AcceptedAt)
		if err != nil {
			return nil, err
		}
		requests = append(requests, req)
	}

	return requests, nil
}

// ==============================================================================
// Message Operations
// ==============================================================================

// SaveMessage saves a sent message
func (d *Database) SaveMessage(message *Message) (int64, error) {
	query := `
		INSERT INTO messages (profile_id, profile_url, content, template, message_type, sent_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	result, err := d.db.Exec(query,
		message.ProfileID, message.ProfileURL, message.Content, message.Template, message.MessageType, time.Now(),
	)
	if err != nil {
		return 0, fmt.Errorf("failed to save message: %w", err)
	}

	id, _ := result.LastInsertId()
	d.logger.WithField("profile_url", message.ProfileURL).Info("Message saved")

	// Update daily stats
	d.incrementDailyStat("messages_sent")

	return id, nil
}

// HasSentFollowUpMessage checks if a follow-up message was already sent
func (d *Database) HasSentFollowUpMessage(profileURL string) (bool, error) {
	query := `SELECT COUNT(*) FROM messages WHERE profile_url = ? AND message_type = 'follow_up'`
	var count int
	err := d.db.QueryRow(query, profileURL).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetTodayMessageCount returns the number of messages sent today
func (d *Database) GetTodayMessageCount() (int, error) {
	query := `SELECT COUNT(*) FROM messages WHERE DATE(sent_at) = DATE('now')`
	var count int
	err := d.db.QueryRow(query).Scan(&count)
	return count, err
}

// GetMessageHistory gets message history for a profile
func (d *Database) GetMessageHistory(profileURL string) ([]*Message, error) {
	query := `
		SELECT id, profile_id, profile_url, content, template, message_type, sent_at
		FROM messages WHERE profile_url = ?
		ORDER BY sent_at DESC
	`

	rows, err := d.db.Query(query, profileURL)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		msg := &Message{}
		err := rows.Scan(&msg.ID, &msg.ProfileID, &msg.ProfileURL, &msg.Content, &msg.Template, &msg.MessageType, &msg.SentAt)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// ==============================================================================
// Daily Stats Operations
// ==============================================================================

// GetTodayStats returns today's activity statistics
func (d *Database) GetTodayStats() (*DailyStats, error) {
	today := time.Now().Format("2006-01-02")
	query := `SELECT date, connections_sent, connections_accepted, messages_sent, profiles_viewed, searches_performed FROM daily_stats WHERE date = ?`

	stats := &DailyStats{Date: today}
	err := d.db.QueryRow(query, today).Scan(
		&stats.Date, &stats.ConnectionsSent, &stats.ConnectionsAccepted,
		&stats.MessagesSent, &stats.ProfilesViewed, &stats.SearchesPerformed,
	)

	if err == sql.ErrNoRows {
		return stats, nil
	}
	if err != nil {
		return nil, err
	}

	return stats, nil
}

// incrementDailyStat increments a daily stat counter
func (d *Database) incrementDailyStat(statName string) error {
	today := time.Now().Format("2006-01-02")

	// Ensure row exists
	insertQuery := `INSERT OR IGNORE INTO daily_stats (date) VALUES (?)`
	d.db.Exec(insertQuery, today)

	// Update the stat
	updateQuery := fmt.Sprintf(`UPDATE daily_stats SET %s = %s + 1 WHERE date = ?`, statName, statName)
	_, err := d.db.Exec(updateQuery, today)
	return err
}

// IncrementProfileViews increments the profile views counter
func (d *Database) IncrementProfileViews() error {
	return d.incrementDailyStat("profiles_viewed")
}

// IncrementSearches increments the searches counter
func (d *Database) IncrementSearches() error {
	return d.incrementDailyStat("searches_performed")
}

// ==============================================================================
// Cookie/Session Operations
// ==============================================================================

// SaveCookies saves session cookies
func (d *Database) SaveCookies(cookies []*SessionCookie) error {
	// Clear existing cookies
	d.db.Exec("DELETE FROM session_cookies")

	query := `INSERT INTO session_cookies (name, value, domain, path, expires, http_only, secure) VALUES (?, ?, ?, ?, ?, ?, ?)`

	for _, cookie := range cookies {
		_, err := d.db.Exec(query, cookie.Name, cookie.Value, cookie.Domain, cookie.Path, cookie.Expires, cookie.HTTPOnly, cookie.Secure)
		if err != nil {
			return fmt.Errorf("failed to save cookie: %w", err)
		}
	}

	d.logger.Infof("Saved %d session cookies", len(cookies))
	return nil
}

// LoadCookies loads session cookies
func (d *Database) LoadCookies() ([]*SessionCookie, error) {
	query := `SELECT name, value, domain, path, expires, http_only, secure FROM session_cookies`

	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cookies []*SessionCookie
	for rows.Next() {
		cookie := &SessionCookie{}
		err := rows.Scan(&cookie.Name, &cookie.Value, &cookie.Domain, &cookie.Path, &cookie.Expires, &cookie.HTTPOnly, &cookie.Secure)
		if err != nil {
			return nil, err
		}
		cookies = append(cookies, cookie)
	}

	d.logger.Infof("Loaded %d session cookies", len(cookies))
	return cookies, nil
}

// SaveCookiesToFile saves cookies to a JSON file
func (d *Database) SaveCookiesToFile(cookies []*SessionCookie, filePath string) error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cookies, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0600)
}

// LoadCookiesFromFile loads cookies from a JSON file
func (d *Database) LoadCookiesFromFile(filePath string) ([]*SessionCookie, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var cookies []*SessionCookie
	if err := json.Unmarshal(data, &cookies); err != nil {
		return nil, err
	}

	return cookies, nil
}

// ==============================================================================
// Search History Operations
// ==============================================================================

// SaveSearchHistory saves a search query
func (d *Database) SaveSearchHistory(query, jobTitle, company, location string, keywords []string, resultsCount int) error {
	keywordsJSON, _ := json.Marshal(keywords)

	insertQuery := `
		INSERT INTO search_history (query, job_title, company, location, keywords, results_count)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := d.db.Exec(insertQuery, query, jobTitle, company, location, string(keywordsJSON), resultsCount)
	if err != nil {
		return err
	}

	d.incrementDailyStat("searches_performed")
	return nil
}
