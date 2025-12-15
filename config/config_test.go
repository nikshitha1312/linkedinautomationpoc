// Package config - Tests for configuration management
package config

import (
	"os"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig should not return nil")
	}

	// Check default values
	if cfg.Browser.Timeout != 30 {
		t.Errorf("Expected default timeout of 30, got %d", cfg.Browser.Timeout)
	}

	if cfg.RateLimits.MaxConnectionsPerDay != 25 {
		t.Errorf("Expected default max connections of 25, got %d", cfg.RateLimits.MaxConnectionsPerDay)
	}

	if cfg.Stealth.MouseOvershoot != true {
		t.Error("Mouse overshoot should be enabled by default")
	}

	if cfg.Schedule.StartHour != 9 {
		t.Errorf("Expected default start hour of 9, got %d", cfg.Schedule.StartHour)
	}
}

func TestConfigValidation(t *testing.T) {
	cfg := DefaultConfig()

	// Should fail without credentials
	err := cfg.Validate()
	if err == nil {
		t.Error("Validation should fail without credentials")
	}

	// Set credentials
	cfg.LinkedIn.Email = "test@example.com"
	cfg.LinkedIn.Password = "password123"

	// Should pass with credentials
	err = cfg.Validate()
	if err != nil {
		t.Errorf("Validation should pass with credentials: %v", err)
	}

	// Test invalid rate limits
	cfg.RateLimits.MaxConnectionsPerDay = 200
	err = cfg.Validate()
	if err == nil {
		t.Error("Validation should fail with max connections > 100")
	}
	cfg.RateLimits.MaxConnectionsPerDay = 25 // Reset

	// Test invalid log level
	cfg.Logging.Level = "invalid"
	err = cfg.Validate()
	if err == nil {
		t.Error("Validation should fail with invalid log level")
	}
	cfg.Logging.Level = "info" // Reset

	// Test invalid schedule hours
	cfg.Schedule.StartHour = 25
	err = cfg.Validate()
	if err == nil {
		t.Error("Validation should fail with start hour > 23")
	}
}

func TestEnvOverrides(t *testing.T) {
	// Set environment variables
	os.Setenv("LINKEDIN_EMAIL", "env_email@test.com")
	os.Setenv("LINKEDIN_PASSWORD", "env_password")
	os.Setenv("MAX_CONNECTIONS_PER_DAY", "10")
	os.Setenv("LOG_LEVEL", "debug")
	defer func() {
		os.Unsetenv("LINKEDIN_EMAIL")
		os.Unsetenv("LINKEDIN_PASSWORD")
		os.Unsetenv("MAX_CONNECTIONS_PER_DAY")
		os.Unsetenv("LOG_LEVEL")
	}()

	cfg := DefaultConfig()
	cfg.applyEnvOverrides()

	if cfg.LinkedIn.Email != "env_email@test.com" {
		t.Errorf("Email should be overridden from env, got %s", cfg.LinkedIn.Email)
	}

	if cfg.LinkedIn.Password != "env_password" {
		t.Error("Password should be overridden from env")
	}

	if cfg.RateLimits.MaxConnectionsPerDay != 10 {
		t.Errorf("Max connections should be 10 from env, got %d", cfg.RateLimits.MaxConnectionsPerDay)
	}

	if cfg.Logging.Level != "debug" {
		t.Errorf("Log level should be debug from env, got %s", cfg.Logging.Level)
	}
}

func TestGetTimeout(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Browser.Timeout = 60

	timeout := cfg.GetTimeout()
	if timeout.Seconds() != 60 {
		t.Errorf("Expected 60 seconds, got %f", timeout.Seconds())
	}
}

func TestLoadConfigNonExistent(t *testing.T) {
	// Set required env vars
	os.Setenv("LINKEDIN_EMAIL", "test@test.com")
	os.Setenv("LINKEDIN_PASSWORD", "password")
	defer func() {
		os.Unsetenv("LINKEDIN_EMAIL")
		os.Unsetenv("LINKEDIN_PASSWORD")
	}()

	cfg, err := LoadConfig("nonexistent.yaml")
	if err != nil {
		t.Errorf("Should not error for non-existent file: %v", err)
	}

	if cfg == nil {
		t.Error("Config should not be nil")
	}

	// Should have defaults
	if cfg.Browser.Timeout != 30 {
		t.Error("Should have default timeout")
	}
}
