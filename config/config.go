// Package config provides configuration management for the LinkedIn automation tool.
// It supports YAML configuration files with environment variable overrides.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration settings for the automation tool
type Config struct {
	// LinkedIn credentials
	LinkedIn LinkedInConfig `yaml:"linkedin"`

	// Browser configuration
	Browser BrowserConfig `yaml:"browser"`

	// Stealth settings for anti-detection
	Stealth StealthConfig `yaml:"stealth"`

	// Rate limiting configuration
	RateLimits RateLimitConfig `yaml:"rate_limits"`

	// Search configuration
	Search SearchConfig `yaml:"search"`

	// Messaging configuration
	Messaging MessagingConfig `yaml:"messaging"`

	// Storage configuration
	Storage StorageConfig `yaml:"storage"`

	// Logging configuration
	Logging LoggingConfig `yaml:"logging"`

	// Activity scheduling
	Schedule ScheduleConfig `yaml:"schedule"`
}

// LinkedInConfig holds LinkedIn-specific settings
type LinkedInConfig struct {
	Email    string `yaml:"email"`
	Password string `yaml:"password"`
}

// BrowserConfig holds browser automation settings
type BrowserConfig struct {
	Headless       bool   `yaml:"headless"`
	UserDataDir    string `yaml:"user_data_dir"`
	SlowMotion     int    `yaml:"slow_motion_ms"`
	Timeout        int    `yaml:"timeout_seconds"`
	ViewportWidth  int    `yaml:"viewport_width"`
	ViewportHeight int    `yaml:"viewport_height"`
}

// StealthConfig holds anti-detection settings
type StealthConfig struct {
	// Mouse movement settings
	MouseSpeedMin      float64 `yaml:"mouse_speed_min"`
	MouseSpeedMax      float64 `yaml:"mouse_speed_max"`
	MouseOvershoot     bool    `yaml:"mouse_overshoot"`
	MouseMicroCorrect  bool    `yaml:"mouse_micro_corrections"`

	// Typing settings
	TypingDelayMin     int  `yaml:"typing_delay_min_ms"`
	TypingDelayMax     int  `yaml:"typing_delay_max_ms"`
	TypingMistakeRate  float64 `yaml:"typing_mistake_rate"`

	// Scrolling settings
	ScrollSpeedMin     int  `yaml:"scroll_speed_min"`
	ScrollSpeedMax     int  `yaml:"scroll_speed_max"`
	ScrollBackChance   float64 `yaml:"scroll_back_chance"`

	// Timing settings
	ActionDelayMin     int  `yaml:"action_delay_min_ms"`
	ActionDelayMax     int  `yaml:"action_delay_max_ms"`
	PageLoadWaitMin    int  `yaml:"page_load_wait_min_ms"`
	PageLoadWaitMax    int  `yaml:"page_load_wait_max_ms"`

	// Fingerprint masking
	RandomizeViewport  bool    `yaml:"randomize_viewport"`
	DisableWebdriver   bool    `yaml:"disable_webdriver"`
	RandomUserAgent    bool    `yaml:"random_user_agent"`
}

// RateLimitConfig holds rate limiting settings
type RateLimitConfig struct {
	MaxConnectionsPerDay    int `yaml:"max_connections_per_day"`
	MaxMessagesPerDay       int `yaml:"max_messages_per_day"`
	MaxProfileViewsPerDay   int `yaml:"max_profile_views_per_day"`
	MaxSearchesPerHour      int `yaml:"max_searches_per_hour"`
	CooldownMinutes         int `yaml:"cooldown_minutes"`
	MinDelayBetweenActions  int `yaml:"min_delay_between_actions_ms"`
	MaxDelayBetweenActions  int `yaml:"max_delay_between_actions_ms"`
}

// SearchConfig holds search-related settings
type SearchConfig struct {
	DefaultJobTitle    string   `yaml:"default_job_title"`
	DefaultCompany     string   `yaml:"default_company"`
	DefaultLocation    string   `yaml:"default_location"`
	Keywords           []string `yaml:"keywords"`
	MaxResultsPerSearch int     `yaml:"max_results_per_search"`
}

// MessagingConfig holds messaging settings
type MessagingConfig struct {
	ConnectionNoteTemplate  string `yaml:"connection_note_template"`
	FollowUpMessageTemplate string `yaml:"follow_up_message_template"`
	MaxNoteLength           int    `yaml:"max_note_length"`
	MaxMessageLength        int    `yaml:"max_message_length"`
}

// StorageConfig holds data persistence settings
type StorageConfig struct {
	DatabasePath   string `yaml:"database_path"`
	CookiesPath    string `yaml:"cookies_path"`
	BackupEnabled  bool   `yaml:"backup_enabled"`
	BackupInterval int    `yaml:"backup_interval_hours"`
}

// LoggingConfig holds logging settings
type LoggingConfig struct {
	Level      string `yaml:"level"`
	Format     string `yaml:"format"`
	OutputFile string `yaml:"output_file"`
	MaxSizeMB  int    `yaml:"max_size_mb"`
	MaxBackups int    `yaml:"max_backups"`
}

// ScheduleConfig holds activity scheduling settings
type ScheduleConfig struct {
	Enabled        bool   `yaml:"enabled"`
	StartHour      int    `yaml:"start_hour"`
	EndHour        int    `yaml:"end_hour"`
	WorkDaysOnly   bool   `yaml:"work_days_only"`
	BreakMinMin    int    `yaml:"break_min_minutes"`
	BreakMinMax    int    `yaml:"break_max_minutes"`
	SessionMaxMin  int    `yaml:"session_max_minutes"`
	Timezone       string `yaml:"timezone"`
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		LinkedIn: LinkedInConfig{
			Email:    "",
			Password: "",
		},
		Browser: BrowserConfig{
			Headless:       false,
			UserDataDir:    "./data/browser",
			SlowMotion:     0,
			Timeout:        30,
			ViewportWidth:  1366,
			ViewportHeight: 768,
		},
		Stealth: StealthConfig{
			MouseSpeedMin:      0.5,
			MouseSpeedMax:      2.0,
			MouseOvershoot:     true,
			MouseMicroCorrect:  true,
			TypingDelayMin:     50,
			TypingDelayMax:     200,
			TypingMistakeRate:  0.02,
			ScrollSpeedMin:     100,
			ScrollSpeedMax:     400,
			ScrollBackChance:   0.15,
			ActionDelayMin:     500,
			ActionDelayMax:     2000,
			PageLoadWaitMin:    1000,
			PageLoadWaitMax:    3000,
			RandomizeViewport:  true,
			DisableWebdriver:   true,
			RandomUserAgent:    true,
		},
		RateLimits: RateLimitConfig{
			MaxConnectionsPerDay:   25,
			MaxMessagesPerDay:      50,
			MaxProfileViewsPerDay:  100,
			MaxSearchesPerHour:     10,
			CooldownMinutes:        5,
			MinDelayBetweenActions: 2000,
			MaxDelayBetweenActions: 5000,
		},
		Search: SearchConfig{
			DefaultJobTitle:     "",
			DefaultCompany:      "",
			DefaultLocation:     "",
			Keywords:            []string{},
			MaxResultsPerSearch: 25,
		},
		Messaging: MessagingConfig{
			ConnectionNoteTemplate:  "Hi {{.FirstName}}, I came across your profile and would love to connect!",
			FollowUpMessageTemplate: "Thanks for connecting, {{.FirstName}}! I'd love to learn more about your work at {{.Company}}.",
			MaxNoteLength:           300,
			MaxMessageLength:        8000,
		},
		Storage: StorageConfig{
			DatabasePath:   "./data/linkedin_automation.db",
			CookiesPath:    "./data/cookies.json",
			BackupEnabled:  true,
			BackupInterval: 24,
		},
		Logging: LoggingConfig{
			Level:      "info",
			Format:     "json",
			OutputFile: "./logs/automation.log",
			MaxSizeMB:  100,
			MaxBackups: 5,
		},
		Schedule: ScheduleConfig{
			Enabled:       true,
			StartHour:     9,
			EndHour:       18,
			WorkDaysOnly:  true,
			BreakMinMin:   5,
			BreakMinMax:   15,
			SessionMaxMin: 120,
			Timezone:      "Local",
		},
	}
}

// LoadConfig loads configuration from a YAML file and applies environment variable overrides
func LoadConfig(configPath string) (*Config, error) {
	config := DefaultConfig()

	// Try to load from file if it exists
	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("failed to read config file: %w", err)
			}
			// File doesn't exist, use defaults
		} else {
			if err := yaml.Unmarshal(data, config); err != nil {
				return nil, fmt.Errorf("failed to parse config file: %w", err)
			}
		}
	}

	// Apply environment variable overrides
	config.applyEnvOverrides()

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// applyEnvOverrides applies environment variable overrides to the configuration
func (c *Config) applyEnvOverrides() {
	// LinkedIn credentials (most commonly overridden via env)
	if email := os.Getenv("LINKEDIN_EMAIL"); email != "" {
		c.LinkedIn.Email = email
	}
	if password := os.Getenv("LINKEDIN_PASSWORD"); password != "" {
		c.LinkedIn.Password = password
	}

	// Browser settings
	if headless := os.Getenv("BROWSER_HEADLESS"); headless != "" {
		c.Browser.Headless = headless == "true" || headless == "1"
	}
	if userDataDir := os.Getenv("BROWSER_USER_DATA_DIR"); userDataDir != "" {
		c.Browser.UserDataDir = userDataDir
	}

	// Rate limits
	if maxConn := os.Getenv("MAX_CONNECTIONS_PER_DAY"); maxConn != "" {
		if val, err := strconv.Atoi(maxConn); err == nil {
			c.RateLimits.MaxConnectionsPerDay = val
		}
	}
	if maxMsg := os.Getenv("MAX_MESSAGES_PER_DAY"); maxMsg != "" {
		if val, err := strconv.Atoi(maxMsg); err == nil {
			c.RateLimits.MaxMessagesPerDay = val
		}
	}

	// Logging
	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		c.Logging.Level = logLevel
	}
	if logFormat := os.Getenv("LOG_FORMAT"); logFormat != "" {
		c.Logging.Format = logFormat
	}

	// Storage
	if dbPath := os.Getenv("DATABASE_PATH"); dbPath != "" {
		c.Storage.DatabasePath = dbPath
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Validate required fields
	if c.LinkedIn.Email == "" {
		return fmt.Errorf("LinkedIn email is required (set LINKEDIN_EMAIL env var or in config)")
	}
	if c.LinkedIn.Password == "" {
		return fmt.Errorf("LinkedIn password is required (set LINKEDIN_PASSWORD env var or in config)")
	}

	// Validate rate limits
	if c.RateLimits.MaxConnectionsPerDay < 0 || c.RateLimits.MaxConnectionsPerDay > 100 {
		return fmt.Errorf("max_connections_per_day must be between 0 and 100")
	}
	if c.RateLimits.MaxMessagesPerDay < 0 || c.RateLimits.MaxMessagesPerDay > 150 {
		return fmt.Errorf("max_messages_per_day must be between 0 and 150")
	}

	// Validate schedule
	if c.Schedule.StartHour < 0 || c.Schedule.StartHour > 23 {
		return fmt.Errorf("start_hour must be between 0 and 23")
	}
	if c.Schedule.EndHour < 0 || c.Schedule.EndHour > 23 {
		return fmt.Errorf("end_hour must be between 0 and 23")
	}

	// Validate logging level
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[c.Logging.Level] {
		return fmt.Errorf("invalid log level: %s (must be debug, info, warn, or error)", c.Logging.Level)
	}

	return nil
}

// GetTimeout returns the configured timeout as a time.Duration
func (c *Config) GetTimeout() time.Duration {
	return time.Duration(c.Browser.Timeout) * time.Second
}

// SaveConfig saves the current configuration to a YAML file
func (c *Config) SaveConfig(configPath string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
