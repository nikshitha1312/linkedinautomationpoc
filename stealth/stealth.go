// Package stealth provides anti-bot detection techniques for browser automation.
// It implements human-like behavior patterns to avoid detection by anti-automation systems.
package stealth

import (
	"math"
	"math/rand"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
	"github.com/nikshitha/linkedin-automation-poc/config"
	"github.com/nikshitha/linkedin-automation-poc/logger"
)

// StealthManager handles all anti-detection operations
type StealthManager struct {
	config *config.StealthConfig
	logger *logger.Logger
	rand   *rand.Rand
}

// NewStealthManager creates a new stealth manager
func NewStealthManager(cfg *config.StealthConfig, log *logger.Logger) *StealthManager {
	return &StealthManager{
		config: cfg,
		logger: log.WithModule("stealth"),
		rand:   rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Point represents a 2D coordinate
type Point struct {
	X, Y float64
}

// ==============================================================================
// TECHNIQUE 1: Human-like Mouse Movement (Bézier curves with variable speed)
// ==============================================================================

// MoveMouse moves the mouse from current position to target with human-like motion
func (s *StealthManager) MoveMouse(page *rod.Page, targetX, targetY float64) error {
	// Get current mouse position (approximate from page center if unknown)
	currentX, currentY := s.getApproximateMousePosition(page)

	// Generate Bézier curve path
	points := s.generateBezierPath(
		Point{currentX, currentY},
		Point{targetX, targetY},
	)

	// Add overshoot if enabled
	if s.config.MouseOvershoot && s.rand.Float64() < 0.3 {
		points = s.addOvershoot(points, targetX, targetY)
	}

	// Execute movement with variable speed
	for i, point := range points {
		// Variable delay between movements (faster in middle, slower at ends)
		delay := s.calculateMovementDelay(i, len(points))
		time.Sleep(time.Duration(delay) * time.Millisecond)

		err := page.Mouse.MoveLinear(proto.NewPoint(point.X, point.Y), 1)
		if err != nil {
			return err
		}

		// Micro-corrections at the end
		if s.config.MouseMicroCorrect && i > len(points)-5 {
			s.addMicroCorrection(page, point.X, point.Y)
		}
	}

	s.logger.StealthAction("mouse_move", map[string]interface{}{
		"from_x": currentX, "from_y": currentY,
		"to_x": targetX, "to_y": targetY,
		"steps": len(points),
	})

	return nil
}

// generateBezierPath creates a curved path between two points using cubic Bézier
func (s *StealthManager) generateBezierPath(start, end Point) []Point {
	// Generate random control points for natural curve
	distance := math.Sqrt(math.Pow(end.X-start.X, 2) + math.Pow(end.Y-start.Y, 2))
	numSteps := int(distance/10) + 10 // More steps for longer distances

	// Random control points offset
	offsetRange := distance * 0.3
	ctrl1 := Point{
		X: start.X + (end.X-start.X)*0.25 + (s.rand.Float64()-0.5)*offsetRange,
		Y: start.Y + (end.Y-start.Y)*0.25 + (s.rand.Float64()-0.5)*offsetRange,
	}
	ctrl2 := Point{
		X: start.X + (end.X-start.X)*0.75 + (s.rand.Float64()-0.5)*offsetRange,
		Y: start.Y + (end.Y-start.Y)*0.75 + (s.rand.Float64()-0.5)*offsetRange,
	}

	points := make([]Point, numSteps)
	for i := 0; i < numSteps; i++ {
		t := float64(i) / float64(numSteps-1)
		points[i] = s.cubicBezier(t, start, ctrl1, ctrl2, end)
	}

	return points
}

// cubicBezier calculates a point on a cubic Bézier curve
func (s *StealthManager) cubicBezier(t float64, p0, p1, p2, p3 Point) Point {
	u := 1 - t
	tt := t * t
	uu := u * u
	uuu := uu * u
	ttt := tt * t

	return Point{
		X: uuu*p0.X + 3*uu*t*p1.X + 3*u*tt*p2.X + ttt*p3.X,
		Y: uuu*p0.Y + 3*uu*t*p1.Y + 3*u*tt*p2.Y + ttt*p3.Y,
	}
}

// addOvershoot adds natural overshoot past the target
func (s *StealthManager) addOvershoot(points []Point, targetX, targetY float64) []Point {
	// Overshoot amount (5-15 pixels)
	overshootX := (s.rand.Float64()*10 + 5) * s.randomSign()
	overshootY := (s.rand.Float64()*10 + 5) * s.randomSign()

	// Add overshoot point
	overshootPoint := Point{X: targetX + overshootX, Y: targetY + overshootY}
	points = append(points, overshootPoint)

	// Correct back to target
	correctionSteps := 3 + s.rand.Intn(3)
	for i := 0; i < correctionSteps; i++ {
		t := float64(i+1) / float64(correctionSteps)
		points = append(points, Point{
			X: overshootPoint.X + (targetX-overshootPoint.X)*t,
			Y: overshootPoint.Y + (targetY-overshootPoint.Y)*t,
		})
	}

	return points
}

// addMicroCorrection adds small random movements near the target
func (s *StealthManager) addMicroCorrection(page *rod.Page, x, y float64) {
	microX := x + (s.rand.Float64()-0.5)*2
	microY := y + (s.rand.Float64()-0.5)*2
	time.Sleep(time.Duration(5+s.rand.Intn(10)) * time.Millisecond)
	page.Mouse.MoveLinear(proto.NewPoint(microX, microY), 1)
}

// calculateMovementDelay returns variable delay (ease-in-out effect)
func (s *StealthManager) calculateMovementDelay(step, totalSteps int) int {
	// Sine wave for smooth acceleration/deceleration
	progress := float64(step) / float64(totalSteps)
	easeFactor := math.Sin(progress * math.Pi)

	minDelay := int(s.config.MouseSpeedMin * 5)
	maxDelay := int(s.config.MouseSpeedMax * 15)

	// Slower at start and end, faster in middle
	delay := maxDelay - int(float64(maxDelay-minDelay)*easeFactor)
	return delay + s.rand.Intn(3)
}

// getApproximateMousePosition gets current mouse position
func (s *StealthManager) getApproximateMousePosition(page *rod.Page) (float64, float64) {
	// Default to page center if position unknown
	return 683, 384 // Common half-HD viewport center
}

// ==============================================================================
// TECHNIQUE 2: Randomized Timing Patterns
// ==============================================================================

// RandomDelay adds a randomized delay between min and max milliseconds
func (s *StealthManager) RandomDelay(minMs, maxMs int) {
	delay := minMs + s.rand.Intn(maxMs-minMs+1)
	time.Sleep(time.Duration(delay) * time.Millisecond)
}

// ActionDelay adds human-like delay between actions
func (s *StealthManager) ActionDelay() {
	s.RandomDelay(s.config.ActionDelayMin, s.config.ActionDelayMax)
}

// ThinkingDelay simulates human cognitive processing time
func (s *StealthManager) ThinkingDelay() {
	// Longer, variable delay to simulate reading/thinking
	baseDelay := 1000 + s.rand.Intn(3000)
	// Occasionally add extra "consideration" time
	if s.rand.Float64() < 0.2 {
		baseDelay += 2000 + s.rand.Intn(3000)
	}
	time.Sleep(time.Duration(baseDelay) * time.Millisecond)
	s.logger.StealthAction("thinking_delay", map[string]interface{}{"duration_ms": baseDelay})
}

// PageLoadDelay waits for page to fully load with natural variation
func (s *StealthManager) PageLoadDelay() {
	s.RandomDelay(s.config.PageLoadWaitMin, s.config.PageLoadWaitMax)
}

// ==============================================================================
// TECHNIQUE 3: Browser Fingerprint Masking
// ==============================================================================

// ApplyFingerprintMasking applies various browser fingerprint modifications
func (s *StealthManager) ApplyFingerprintMasking(page *rod.Page) error {
	scripts := []string{}

	// Disable webdriver flag
	if s.config.DisableWebdriver {
		scripts = append(scripts, `
			Object.defineProperty(navigator, 'webdriver', {
				get: () => undefined
			});
			
			// Remove automation-related properties
			delete window.cdc_adoQpoasnfa76pfcZLmcfl_Array;
			delete window.cdc_adoQpoasnfa76pfcZLmcfl_Promise;
			delete window.cdc_adoQpoasnfa76pfcZLmcfl_Symbol;
		`)
	}

	// Mask Chrome automation flags
	scripts = append(scripts, `
		// Override Chrome properties
		Object.defineProperty(navigator, 'plugins', {
			get: () => [
				{name: 'Chrome PDF Plugin', filename: 'internal-pdf-viewer'},
				{name: 'Chrome PDF Viewer', filename: 'mhjfbmdgcfjbbpaeojofohoefgiehjai'},
				{name: 'Native Client', filename: 'internal-nacl-plugin'}
			]
		});

		// Override languages
		Object.defineProperty(navigator, 'languages', {
			get: () => ['en-US', 'en']
		});

		// Override permissions
		const originalQuery = window.navigator.permissions.query;
		window.navigator.permissions.query = (parameters) => (
			parameters.name === 'notifications' ?
				Promise.resolve({ state: Notification.permission }) :
				originalQuery(parameters)
		);

		// Add realistic screen properties
		Object.defineProperty(screen, 'availWidth', { get: () => screen.width });
		Object.defineProperty(screen, 'availHeight', { get: () => screen.height - 40 });
	`)

	// Mask hardware concurrency
	scripts = append(scripts, `
		Object.defineProperty(navigator, 'hardwareConcurrency', {
			get: () => `+s.randomHardwareConcurrency()+`
		});
	`)

	// Execute all masking scripts
	for _, script := range scripts {
		_, err := page.Eval(script)
		if err != nil {
			s.logger.WithError(err).Warn("Failed to apply fingerprint mask")
			// Continue with other scripts even if one fails
		}
	}

	s.logger.StealthAction("fingerprint_mask", map[string]interface{}{"scripts_applied": len(scripts)})
	return nil
}

// GetRandomUserAgent returns a random, realistic user agent string
func (s *StealthManager) GetRandomUserAgent() string {
	userAgents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1 Safari/605.1.15",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0",
	}
	return userAgents[s.rand.Intn(len(userAgents))]
}

// GetRandomViewport returns randomized viewport dimensions
func (s *StealthManager) GetRandomViewport() (int, int) {
	viewports := []struct{ width, height int }{
		{1920, 1080},
		{1366, 768},
		{1536, 864},
		{1440, 900},
		{1280, 720},
		{1600, 900},
	}
	vp := viewports[s.rand.Intn(len(viewports))]
	// Add slight random variation
	return vp.width + s.rand.Intn(20) - 10, vp.height + s.rand.Intn(20) - 10
}

func (s *StealthManager) randomHardwareConcurrency() string {
	cores := []int{4, 8, 12, 16}
	return string(rune('0' + cores[s.rand.Intn(len(cores))]))
}

// ==============================================================================
// TECHNIQUE 4: Random Scrolling Behavior
// ==============================================================================

// HumanScroll performs natural scrolling behavior on the page
func (s *StealthManager) HumanScroll(page *rod.Page, direction string, amount int) error {
	// Vary the scroll amount slightly
	actualAmount := amount + s.rand.Intn(100) - 50

	// Scroll in small increments with variable speed
	scrolled := 0
	for scrolled < actualAmount {
		// Variable increment size
		increment := s.config.ScrollSpeedMin + s.rand.Intn(s.config.ScrollSpeedMax-s.config.ScrollSpeedMin)
		if scrolled+increment > actualAmount {
			increment = actualAmount - scrolled
		}

		// Natural acceleration/deceleration
		progress := float64(scrolled) / float64(actualAmount)
		speedFactor := math.Sin(progress * math.Pi) // Faster in middle
		delay := int(float64(30) / (speedFactor + 0.3))

		// Execute scroll
		deltaY := float64(increment)
		if direction == "up" {
			deltaY = -deltaY
		}

		err := page.Mouse.Scroll(0, deltaY, 1)
		if err != nil {
			return err
		}

		scrolled += increment
		time.Sleep(time.Duration(delay) * time.Millisecond)
	}

	// Occasionally scroll back slightly (reading behavior)
	if s.rand.Float64() < s.config.ScrollBackChance {
		backAmount := 50 + s.rand.Intn(100)
		s.scrollBack(page, direction, backAmount)
	}

	s.logger.StealthAction("scroll", map[string]interface{}{
		"direction": direction,
		"amount":    actualAmount,
	})

	return nil
}

// scrollBack performs a small scroll in the opposite direction
func (s *StealthManager) scrollBack(page *rod.Page, originalDirection string, amount int) {
	time.Sleep(time.Duration(200+s.rand.Intn(300)) * time.Millisecond)

	deltaY := float64(amount)
	if originalDirection == "down" {
		deltaY = -deltaY
	}
	page.Mouse.Scroll(0, deltaY, 5)
}

// ScrollToElement scrolls to bring an element into view with natural motion
func (s *StealthManager) ScrollToElement(page *rod.Page, selector string) error {
	el, err := page.Element(selector)
	if err != nil {
		return err
	}

	// Get element position
	box, err := el.Shape()
	if err != nil {
		return err
	}

	// Calculate scroll needed
	viewportHeight := 768.0 // Default
	if info, err := page.Info(); err == nil && info != nil {
		// Use actual viewport if available
		viewportHeight = 768.0
	}

	quad := box.Quads[0]
	elementY := quad[1]
	targetY := elementY - viewportHeight/3 // Position element in upper third

	if targetY > 0 {
		return s.HumanScroll(page, "down", int(targetY))
	} else if targetY < 0 {
		return s.HumanScroll(page, "up", int(-targetY))
	}

	return nil
}

// ==============================================================================
// TECHNIQUE 5: Realistic Typing Simulation
// ==============================================================================

// HumanType types text with human-like characteristics
func (s *StealthManager) HumanType(page *rod.Page, element *rod.Element, text string) error {
	runes := []rune(text)

	for i := 0; i < len(runes); i++ {
		char := runes[i]

		// Random delay between keystrokes
		delay := s.config.TypingDelayMin + s.rand.Intn(s.config.TypingDelayMax-s.config.TypingDelayMin)

		// Occasionally add extra delay (thinking)
		if s.rand.Float64() < 0.05 {
			delay += 200 + s.rand.Intn(400)
		}

		// Simulate typing mistakes
		if s.config.TypingMistakeRate > 0 && s.rand.Float64() < s.config.TypingMistakeRate {
			// Type wrong character
			wrongChar := s.getAdjacentKey(char)
			err := element.Input(string(wrongChar))
			if err != nil {
				return err
			}
			time.Sleep(time.Duration(100+s.rand.Intn(200)) * time.Millisecond)

			// Delete it using Backspace key
			page.Keyboard.Press(input.Backspace)
			time.Sleep(time.Duration(50+s.rand.Intn(100)) * time.Millisecond)
		}

		// Type the correct character
		err := element.Input(string(char))
		if err != nil {
			return err
		}

		time.Sleep(time.Duration(delay) * time.Millisecond)
	}

	s.logger.StealthAction("typing", map[string]interface{}{
		"length":   len(text),
		"mistakes": int(float64(len(text)) * s.config.TypingMistakeRate),
	})

	return nil
}

// getAdjacentKey returns a key adjacent to the given key on a QWERTY keyboard
func (s *StealthManager) getAdjacentKey(char rune) rune {
	adjacentKeys := map[rune][]rune{
		'a': {'s', 'q', 'z'},
		'b': {'v', 'n', 'g', 'h'},
		'c': {'x', 'v', 'd', 'f'},
		'd': {'s', 'f', 'e', 'r', 'c', 'x'},
		'e': {'w', 'r', 'd', 's'},
		'f': {'d', 'g', 'r', 't', 'v', 'c'},
		'g': {'f', 'h', 't', 'y', 'b', 'v'},
		'h': {'g', 'j', 'y', 'u', 'n', 'b'},
		'i': {'u', 'o', 'k', 'j'},
		'j': {'h', 'k', 'u', 'i', 'm', 'n'},
		'k': {'j', 'l', 'i', 'o', 'm'},
		'l': {'k', 'o', 'p'},
		'm': {'n', 'j', 'k'},
		'n': {'b', 'm', 'h', 'j'},
		'o': {'i', 'p', 'k', 'l'},
		'p': {'o', 'l'},
		'q': {'w', 'a'},
		'r': {'e', 't', 'd', 'f'},
		's': {'a', 'd', 'w', 'e', 'z', 'x'},
		't': {'r', 'y', 'f', 'g'},
		'u': {'y', 'i', 'h', 'j'},
		'v': {'c', 'b', 'f', 'g'},
		'w': {'q', 'e', 'a', 's'},
		'x': {'z', 'c', 's', 'd'},
		'y': {'t', 'u', 'g', 'h'},
		'z': {'a', 'x'},
	}

	lowerChar := char
	if char >= 'A' && char <= 'Z' {
		lowerChar = char + 32
	}

	if adjacent, ok := adjacentKeys[lowerChar]; ok {
		result := adjacent[s.rand.Intn(len(adjacent))]
		if char >= 'A' && char <= 'Z' {
			result -= 32
		}
		return result
	}
	return char
}

// ==============================================================================
// TECHNIQUE 6: Mouse Hovering & Random Movement
// ==============================================================================

// HoverElement hovers over an element naturally
func (s *StealthManager) HoverElement(page *rod.Page, element *rod.Element) error {
	box, err := element.Shape()
	if err != nil {
		return err
	}

	quad := box.Quads[0]
	// Random position within element bounds
	x := quad[0] + (quad[2]-quad[0])*s.rand.Float64()*0.6 + (quad[2]-quad[0])*0.2
	y := quad[1] + (quad[5]-quad[1])*s.rand.Float64()*0.6 + (quad[5]-quad[1])*0.2

	err = s.MoveMouse(page, x, y)
	if err != nil {
		return err
	}

	// Hover for a natural duration
	hoverTime := 200 + s.rand.Intn(500)
	time.Sleep(time.Duration(hoverTime) * time.Millisecond)

	return nil
}

// RandomMouseWander performs random mouse movements to simulate idle behavior
func (s *StealthManager) RandomMouseWander(page *rod.Page) error {
	numMoves := 2 + s.rand.Intn(4)

	for i := 0; i < numMoves; i++ {
		// Random position within viewport
		x := 100 + s.rand.Float64()*1000
		y := 100 + s.rand.Float64()*500

		err := s.MoveMouse(page, x, y)
		if err != nil {
			return err
		}

		// Wait between movements
		time.Sleep(time.Duration(300+s.rand.Intn(700)) * time.Millisecond)
	}

	s.logger.StealthAction("mouse_wander", map[string]interface{}{"movements": numMoves})
	return nil
}

// ==============================================================================
// TECHNIQUE 7: Activity Scheduling
// ==============================================================================

// Scheduler manages activity timing
type Scheduler struct {
	config *config.ScheduleConfig
	logger *logger.Logger
	rand   *rand.Rand
}

// NewScheduler creates a new activity scheduler
func NewScheduler(cfg *config.ScheduleConfig, log *logger.Logger) *Scheduler {
	return &Scheduler{
		config: cfg,
		logger: log.WithModule("scheduler"),
		rand:   rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// IsWithinOperatingHours checks if current time is within allowed hours
func (s *Scheduler) IsWithinOperatingHours() bool {
	if !s.config.Enabled {
		return true
	}

	now := time.Now()

	// Check work days only
	if s.config.WorkDaysOnly {
		weekday := now.Weekday()
		if weekday == time.Saturday || weekday == time.Sunday {
			s.logger.Info("Outside work days - activity paused")
			return false
		}
	}

	// Check hours
	hour := now.Hour()
	if hour < s.config.StartHour || hour >= s.config.EndHour {
		s.logger.Infof("Outside operating hours (%d:00 - %d:00) - current: %d:00",
			s.config.StartHour, s.config.EndHour, hour)
		return false
	}

	return true
}

// WaitForOperatingHours waits until operating hours begin
func (s *Scheduler) WaitForOperatingHours() {
	for !s.IsWithinOperatingHours() {
		s.logger.Info("Waiting for operating hours...")
		time.Sleep(5 * time.Minute)
	}
}

// ShouldTakeBreak determines if it's time for a break
func (s *Scheduler) ShouldTakeBreak(sessionStart time.Time) bool {
	elapsed := time.Since(sessionStart)
	maxSession := time.Duration(s.config.SessionMaxMin) * time.Minute

	if elapsed > maxSession {
		return true
	}

	// Random break chance increases over time
	breakChance := float64(elapsed.Minutes()) / float64(s.config.SessionMaxMin) * 0.3
	return s.rand.Float64() < breakChance
}

// TakeBreak simulates a human taking a break
func (s *Scheduler) TakeBreak() {
	breakDuration := s.config.BreakMinMin + s.rand.Intn(s.config.BreakMinMax-s.config.BreakMinMin+1)
	s.logger.Infof("Taking a break for %d minutes", breakDuration)
	time.Sleep(time.Duration(breakDuration) * time.Minute)
}

// ==============================================================================
// TECHNIQUE 8: Rate Limiting & Throttling
// ==============================================================================

// RateLimiter manages rate limiting for actions
type RateLimiter struct {
	config      *config.RateLimitConfig
	logger      *logger.Logger
	actionCounts map[string]int
	lastReset   time.Time
	lastAction  time.Time
	rand        *rand.Rand
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(cfg *config.RateLimitConfig, log *logger.Logger) *RateLimiter {
	return &RateLimiter{
		config:       cfg,
		logger:       log.WithModule("rate_limiter"),
		actionCounts: make(map[string]int),
		lastReset:    time.Now(),
		lastAction:   time.Now(),
		rand:         rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// CanPerformAction checks if an action can be performed within rate limits
func (r *RateLimiter) CanPerformAction(actionType string) bool {
	r.checkReset()

	var limit int
	switch actionType {
	case "connection":
		limit = r.config.MaxConnectionsPerDay
	case "message":
		limit = r.config.MaxMessagesPerDay
	case "profile_view":
		limit = r.config.MaxProfileViewsPerDay
	case "search":
		limit = r.config.MaxSearchesPerHour
	default:
		return true
	}

	current := r.actionCounts[actionType]
	if current >= limit {
		r.logger.RateLimit(actionType, current, limit)
		return false
	}

	return true
}

// RecordAction records that an action was performed
func (r *RateLimiter) RecordAction(actionType string) {
	r.actionCounts[actionType]++
	r.lastAction = time.Now()

	r.logger.WithFields(map[string]interface{}{
		"action_type": actionType,
		"count":       r.actionCounts[actionType],
	}).Debug("Action recorded")
}

// WaitForNextAction enforces minimum delay between actions
func (r *RateLimiter) WaitForNextAction() {
	elapsed := time.Since(r.lastAction)
	minDelay := time.Duration(r.config.MinDelayBetweenActions) * time.Millisecond
	maxDelay := time.Duration(r.config.MaxDelayBetweenActions) * time.Millisecond

	// Random delay within range
	targetDelay := minDelay + time.Duration(r.rand.Int63n(int64(maxDelay-minDelay)))

	if elapsed < targetDelay {
		sleepTime := targetDelay - elapsed
		time.Sleep(sleepTime)
	}
}

// EnforceCooldown enforces a cooldown period
func (r *RateLimiter) EnforceCooldown() {
	cooldownDuration := time.Duration(r.config.CooldownMinutes) * time.Minute
	r.logger.Infof("Enforcing cooldown for %d minutes", r.config.CooldownMinutes)
	time.Sleep(cooldownDuration)
}

// GetRemainingActions returns how many more actions of a type can be performed
func (r *RateLimiter) GetRemainingActions(actionType string) int {
	r.checkReset()

	var limit int
	switch actionType {
	case "connection":
		limit = r.config.MaxConnectionsPerDay
	case "message":
		limit = r.config.MaxMessagesPerDay
	case "profile_view":
		limit = r.config.MaxProfileViewsPerDay
	case "search":
		limit = r.config.MaxSearchesPerHour
	default:
		return 999
	}

	return limit - r.actionCounts[actionType]
}

// checkReset resets counts if a new day/hour has started
func (r *RateLimiter) checkReset() {
	now := time.Now()

	// Reset daily counts
	if now.YearDay() != r.lastReset.YearDay() || now.Year() != r.lastReset.Year() {
		r.actionCounts["connection"] = 0
		r.actionCounts["message"] = 0
		r.actionCounts["profile_view"] = 0
		r.lastReset = now
		r.logger.Info("Daily rate limits reset")
	}

	// Reset hourly counts
	if now.Hour() != r.lastReset.Hour() {
		r.actionCounts["search"] = 0
	}
}

// ==============================================================================
// Helper functions
// ==============================================================================

func (s *StealthManager) randomSign() float64 {
	if s.rand.Float64() < 0.5 {
		return -1
	}
	return 1
}

// ClickElement performs a human-like click on an element
func (s *StealthManager) ClickElement(page *rod.Page, element *rod.Element) error {
	// First hover over the element
	err := s.HoverElement(page, element)
	if err != nil {
		return err
	}

	// Small delay before clicking
	time.Sleep(time.Duration(50+s.rand.Intn(150)) * time.Millisecond)

	// Click
	err = element.Click(proto.InputMouseButtonLeft, 1)
	if err != nil {
		return err
	}

	// Small delay after clicking
	time.Sleep(time.Duration(100+s.rand.Intn(200)) * time.Millisecond)

	return nil
}
