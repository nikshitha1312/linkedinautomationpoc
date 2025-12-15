// Package logger provides structured logging for the LinkedIn automation tool.
// It supports multiple log levels, output formats, and contextual information.
package logger

import (
	"io"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

// Logger wraps logrus to provide structured logging
type Logger struct {
	*logrus.Logger
	fields logrus.Fields
}

// Config holds logger configuration
type Config struct {
	Level      string
	Format     string
	OutputFile string
}

// New creates a new logger instance with the given configuration
func New(cfg Config) (*Logger, error) {
	log := logrus.New()

	// Set log level
	level, err := logrus.ParseLevel(cfg.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	log.SetLevel(level)

	// Set output format
	if cfg.Format == "json" {
		log.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime:  "timestamp",
				logrus.FieldKeyLevel: "level",
				logrus.FieldKeyMsg:   "message",
			},
		})
	} else {
		log.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
			FullTimestamp:   true,
			ForceColors:     true,
		})
	}

	// Set up output
	writers := []io.Writer{os.Stdout}

	if cfg.OutputFile != "" {
		// Create log directory if it doesn't exist
		logDir := filepath.Dir(cfg.OutputFile)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return nil, err
		}

		// Open log file
		file, err := os.OpenFile(cfg.OutputFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, err
		}
		writers = append(writers, file)
	}

	log.SetOutput(io.MultiWriter(writers...))

	return &Logger{
		Logger: log,
		fields: make(logrus.Fields),
	}, nil
}

// WithField returns a new logger with the given field added
func (l *Logger) WithField(key string, value interface{}) *Logger {
	newFields := make(logrus.Fields)
	for k, v := range l.fields {
		newFields[k] = v
	}
	newFields[key] = value

	return &Logger{
		Logger: l.Logger,
		fields: newFields,
	}
}

// WithFields returns a new logger with multiple fields added
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	newFields := make(logrus.Fields)
	for k, v := range l.fields {
		newFields[k] = v
	}
	for k, v := range fields {
		newFields[k] = v
	}

	return &Logger{
		Logger: l.Logger,
		fields: newFields,
	}
}

// WithModule returns a new logger with the module field set
func (l *Logger) WithModule(module string) *Logger {
	return l.WithField("module", module)
}

// WithAction returns a new logger with the action field set
func (l *Logger) WithAction(action string) *Logger {
	return l.WithField("action", action)
}

// Debug logs a debug message with context fields
func (l *Logger) Debug(msg string) {
	l.Logger.WithFields(l.fields).Debug(msg)
}

// Debugf logs a formatted debug message
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.Logger.WithFields(l.fields).Debugf(format, args...)
}

// Info logs an info message with context fields
func (l *Logger) Info(msg string) {
	l.Logger.WithFields(l.fields).Info(msg)
}

// Infof logs a formatted info message
func (l *Logger) Infof(format string, args ...interface{}) {
	l.Logger.WithFields(l.fields).Infof(format, args...)
}

// Warn logs a warning message with context fields
func (l *Logger) Warn(msg string) {
	l.Logger.WithFields(l.fields).Warn(msg)
}

// Warnf logs a formatted warning message
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.Logger.WithFields(l.fields).Warnf(format, args...)
}

// Error logs an error message with context fields
func (l *Logger) Error(msg string) {
	l.Logger.WithFields(l.fields).Error(msg)
}

// Errorf logs a formatted error message
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.Logger.WithFields(l.fields).Errorf(format, args...)
}

// WithError returns a new logger with error field added
func (l *Logger) WithError(err error) *Logger {
	return l.WithField("error", err.Error())
}

// StealthAction logs a stealth action with details
func (l *Logger) StealthAction(action string, details map[string]interface{}) {
	fields := make(map[string]interface{})
	fields["stealth_action"] = action
	for k, v := range details {
		fields[k] = v
	}
	l.WithFields(fields).Debug("Stealth action performed")
}

// BrowserAction logs a browser action
func (l *Logger) BrowserAction(action string, url string) {
	l.WithFields(map[string]interface{}{
		"browser_action": action,
		"url":            url,
	}).Info("Browser action")
}

// ConnectionRequest logs a connection request attempt
func (l *Logger) ConnectionRequest(profileURL string, status string, note string) {
	l.WithFields(map[string]interface{}{
		"profile_url": profileURL,
		"status":      status,
		"note_length": len(note),
	}).Info("Connection request")
}

// Message logs a message action
func (l *Logger) Message(recipientURL string, status string, templateUsed string) {
	l.WithFields(map[string]interface{}{
		"recipient_url": recipientURL,
		"status":        status,
		"template":      templateUsed,
	}).Info("Message sent")
}

// RateLimit logs rate limit events
func (l *Logger) RateLimit(limitType string, current int, max int) {
	l.WithFields(map[string]interface{}{
		"limit_type": limitType,
		"current":    current,
		"max":        max,
	}).Warn("Rate limit status")
}

// SecurityEvent logs security-related events (2FA, captcha, etc.)
func (l *Logger) SecurityEvent(eventType string, details string) {
	l.WithFields(map[string]interface{}{
		"security_event": eventType,
		"details":        details,
	}).Warn("Security event detected")
}
