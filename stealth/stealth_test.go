// Package stealth - Tests for anti-detection techniques
package stealth

import (
	"testing"
	"time"

	"github.com/nikshitha/linkedin-automation-poc/config"
	"github.com/nikshitha/linkedin-automation-poc/logger"
)

func TestNewStealthManager(t *testing.T) {
	cfg := &config.StealthConfig{
		MouseSpeedMin:     0.5,
		MouseSpeedMax:     2.0,
		MouseOvershoot:    true,
		MouseMicroCorrect: true,
		TypingDelayMin:    50,
		TypingDelayMax:    200,
		TypingMistakeRate: 0.02,
		ActionDelayMin:    500,
		ActionDelayMax:    2000,
	}

	log, _ := logger.New(logger.Config{Level: "error"})
	sm := NewStealthManager(cfg, log)

	if sm == nil {
		t.Fatal("StealthManager should not be nil")
	}

	if sm.config != cfg {
		t.Error("Config should match")
	}
}

func TestBezierPath(t *testing.T) {
	cfg := &config.StealthConfig{
		MouseSpeedMin:     0.5,
		MouseSpeedMax:     2.0,
		MouseOvershoot:    true,
		MouseMicroCorrect: true,
	}

	log, _ := logger.New(logger.Config{Level: "error"})
	sm := NewStealthManager(cfg, log)

	start := Point{X: 0, Y: 0}
	end := Point{X: 100, Y: 100}

	points := sm.generateBezierPath(start, end)

	if len(points) < 10 {
		t.Errorf("Expected at least 10 points, got %d", len(points))
	}

	// Check that path starts near start point
	firstPoint := points[0]
	if firstPoint.X > 10 || firstPoint.Y > 10 {
		t.Error("First point should be near start")
	}

	// Check that path ends near end point
	lastPoint := points[len(points)-1]
	if lastPoint.X < 90 || lastPoint.Y < 90 {
		t.Error("Last point should be near end")
	}
}

func TestCubicBezier(t *testing.T) {
	cfg := &config.StealthConfig{}
	log, _ := logger.New(logger.Config{Level: "error"})
	sm := NewStealthManager(cfg, log)

	p0 := Point{X: 0, Y: 0}
	p1 := Point{X: 25, Y: 25}
	p2 := Point{X: 75, Y: 75}
	p3 := Point{X: 100, Y: 100}

	// At t=0, should be at p0
	point0 := sm.cubicBezier(0, p0, p1, p2, p3)
	if point0.X != 0 || point0.Y != 0 {
		t.Errorf("At t=0, expected (0,0), got (%f,%f)", point0.X, point0.Y)
	}

	// At t=1, should be at p3
	point1 := sm.cubicBezier(1, p0, p1, p2, p3)
	if point1.X != 100 || point1.Y != 100 {
		t.Errorf("At t=1, expected (100,100), got (%f,%f)", point1.X, point1.Y)
	}
}

func TestRandomDelay(t *testing.T) {
	cfg := &config.StealthConfig{
		ActionDelayMin: 100,
		ActionDelayMax: 200,
	}

	log, _ := logger.New(logger.Config{Level: "error"})
	sm := NewStealthManager(cfg, log)

	start := time.Now()
	sm.RandomDelay(100, 200)
	elapsed := time.Since(start)

	if elapsed < 100*time.Millisecond {
		t.Error("Delay should be at least 100ms")
	}

	if elapsed > 250*time.Millisecond {
		t.Error("Delay should not exceed 200ms significantly")
	}
}

func TestGetAdjacentKey(t *testing.T) {
	cfg := &config.StealthConfig{
		TypingMistakeRate: 1.0, // Force mistakes for testing
	}

	log, _ := logger.New(logger.Config{Level: "error"})
	sm := NewStealthManager(cfg, log)

	// Test that 'a' returns an adjacent key
	adjacent := sm.getAdjacentKey('a')
	validAdjacent := []rune{'s', 'q', 'z'}

	found := false
	for _, v := range validAdjacent {
		if adjacent == v {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Adjacent key for 'a' should be s, q, or z, got %c", adjacent)
	}
}

func TestScheduler(t *testing.T) {
	cfg := &config.ScheduleConfig{
		Enabled:       true,
		StartHour:     0,
		EndHour:       23,
		WorkDaysOnly:  false,
		BreakMinMin:   1,
		BreakMinMax:   2,
		SessionMaxMin: 120,
	}

	log, _ := logger.New(logger.Config{Level: "error"})
	scheduler := NewScheduler(cfg, log)

	// With hours 0-23, should always be within operating hours
	if !scheduler.IsWithinOperatingHours() {
		t.Error("Should be within operating hours with 0-23 range")
	}
}

func TestRateLimiter(t *testing.T) {
	cfg := &config.RateLimitConfig{
		MaxConnectionsPerDay:   5,
		MaxMessagesPerDay:      10,
		MaxProfileViewsPerDay:  20,
		MaxSearchesPerHour:     3,
		CooldownMinutes:        1,
		MinDelayBetweenActions: 100,
		MaxDelayBetweenActions: 200,
	}

	log, _ := logger.New(logger.Config{Level: "error"})
	rl := NewRateLimiter(cfg, log)

	// Should be able to perform initial actions
	if !rl.CanPerformAction("connection") {
		t.Error("Should be able to perform initial connection")
	}

	// Record some actions
	for i := 0; i < 5; i++ {
		rl.RecordAction("connection")
	}

	// Should be at limit
	if rl.CanPerformAction("connection") {
		t.Error("Should not be able to perform connection after limit")
	}

	// Check remaining
	remaining := rl.GetRemainingActions("connection")
	if remaining != 0 {
		t.Errorf("Expected 0 remaining connections, got %d", remaining)
	}

	// Other action types should still be available
	if !rl.CanPerformAction("message") {
		t.Error("Should still be able to send messages")
	}
}

func TestGetRandomUserAgent(t *testing.T) {
	cfg := &config.StealthConfig{
		RandomUserAgent: true,
	}

	log, _ := logger.New(logger.Config{Level: "error"})
	sm := NewStealthManager(cfg, log)

	ua := sm.GetRandomUserAgent()

	if ua == "" {
		t.Error("User agent should not be empty")
	}

	// Should contain typical browser identifiers
	if len(ua) < 50 {
		t.Error("User agent seems too short")
	}
}

func TestGetRandomViewport(t *testing.T) {
	cfg := &config.StealthConfig{
		RandomizeViewport: true,
	}

	log, _ := logger.New(logger.Config{Level: "error"})
	sm := NewStealthManager(cfg, log)

	width, height := sm.GetRandomViewport()

	if width < 1200 || width > 2000 {
		t.Errorf("Width %d seems unreasonable", width)
	}

	if height < 600 || height > 1200 {
		t.Errorf("Height %d seems unreasonable", height)
	}
}

func TestCalculateMovementDelay(t *testing.T) {
	cfg := &config.StealthConfig{
		MouseSpeedMin: 0.5,
		MouseSpeedMax: 2.0,
	}

	log, _ := logger.New(logger.Config{Level: "error"})
	sm := NewStealthManager(cfg, log)

	// Delay at start should be higher than at middle
	delayStart := sm.calculateMovementDelay(0, 100)
	delayMiddle := sm.calculateMovementDelay(50, 100)
	delayEnd := sm.calculateMovementDelay(99, 100)

	if delayMiddle >= delayStart {
		t.Error("Delay in middle should be less than at start (ease-in)")
	}

	if delayMiddle >= delayEnd {
		t.Error("Delay in middle should be less than at end (ease-out)")
	}
}
