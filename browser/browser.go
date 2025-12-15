// Package browser provides browser automation setup and management using Rod.
// It handles browser initialization, stealth configuration, and page management.
package browser

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/nikshitha/linkedin-automation-poc/config"
	"github.com/nikshitha/linkedin-automation-poc/logger"
	"github.com/nikshitha/linkedin-automation-poc/stealth"
)

// Browser wraps the Rod browser with additional functionality
type Browser struct {
	config  *config.Config
	logger  *logger.Logger
	stealth *stealth.StealthManager
	browser *rod.Browser
	page    *rod.Page
}

// NewBrowser creates a new browser instance
func NewBrowser(cfg *config.Config, log *logger.Logger, s *stealth.StealthManager) *Browser {
	return &Browser{
		config:  cfg,
		logger:  log.WithModule("browser"),
		stealth: s,
	}
}

// Launch initializes and launches the browser with stealth settings
func (b *Browser) Launch() error {
	b.logger.Info("Launching browser")

	// Ensure user data directory exists
	if b.config.Browser.UserDataDir != "" {
		absPath, err := filepath.Abs(b.config.Browser.UserDataDir)
		if err != nil {
			return fmt.Errorf("failed to get absolute path for user data dir: %w", err)
		}
		if err := os.MkdirAll(absPath, 0755); err != nil {
			return fmt.Errorf("failed to create user data directory: %w", err)
		}
		b.config.Browser.UserDataDir = absPath
	}

	// Configure launcher with stealth options
	l := launcher.New().
		Headless(b.config.Browser.Headless).
		Set("disable-blink-features", "AutomationControlled").
		Set("disable-infobars").
		Set("disable-dev-shm-usage").
		Set("no-first-run").
		Set("no-default-browser-check").
		Set("disable-background-networking").
		Set("disable-sync").
		Set("disable-translate").
		Set("disable-extensions").
		Set("disable-popup-blocking").
		Set("metrics-recording-only").
		Set("safebrowsing-disable-auto-update")

	// Set user data directory for session persistence
	if b.config.Browser.UserDataDir != "" {
		l = l.UserDataDir(b.config.Browser.UserDataDir)
	}

	// Get random or configured viewport
	var viewportWidth, viewportHeight int
	if b.config.Stealth.RandomizeViewport {
		viewportWidth, viewportHeight = b.stealth.GetRandomViewport()
	} else {
		viewportWidth = b.config.Browser.ViewportWidth
		viewportHeight = b.config.Browser.ViewportHeight
	}

	// Set window size
	l = l.Set("window-size", fmt.Sprintf("%d,%d", viewportWidth, viewportHeight))

	// Launch browser
	url, err := l.Launch()
	if err != nil {
		return fmt.Errorf("failed to launch browser: %w", err)
	}

	// Connect to browser
	b.browser = rod.New().
		ControlURL(url).
		Timeout(time.Duration(b.config.Browser.Timeout) * time.Second)

	if b.config.Browser.SlowMotion > 0 {
		b.browser = b.browser.SlowMotion(time.Duration(b.config.Browser.SlowMotion) * time.Millisecond)
	}

	err = b.browser.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to browser: %w", err)
	}

	b.logger.Info("Browser launched successfully")

	// Create initial page
	return b.createPage(viewportWidth, viewportHeight)
}

// createPage creates a new page with stealth settings
func (b *Browser) createPage(width, height int) error {
	var err error
	b.page, err = b.browser.Page(proto.TargetCreateTarget{URL: "about:blank"})
	if err != nil {
		return fmt.Errorf("failed to create page: %w", err)
	}

	// Set viewport
	err = b.page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
		Width:             width,
		Height:            height,
		DeviceScaleFactor: 1,
		Mobile:            false,
	})
	if err != nil {
		b.logger.WithError(err).Warn("Failed to set viewport")
	}

	// Set user agent if configured
	if b.config.Stealth.RandomUserAgent {
		userAgent := b.stealth.GetRandomUserAgent()
		err = b.page.SetUserAgent(&proto.NetworkSetUserAgentOverride{
			UserAgent: userAgent,
		})
		if err != nil {
			b.logger.WithError(err).Warn("Failed to set user agent")
		} else {
			b.logger.WithField("user_agent", userAgent).Debug("User agent set")
		}
	}

	// Apply fingerprint masking on page load
	b.page.EvalOnNewDocument(b.getStealthScript())

	b.logger.Info("Page created with stealth settings")
	return nil
}

// getStealthScript returns JavaScript to inject for anti-detection
func (b *Browser) getStealthScript() string {
	return `
		// Remove webdriver property
		Object.defineProperty(navigator, 'webdriver', {
			get: () => undefined
		});

		// Overwrite the 'plugins' property to use a custom getter
		Object.defineProperty(navigator, 'plugins', {
			get: () => [
				{ name: 'Chrome PDF Plugin', filename: 'internal-pdf-viewer' },
				{ name: 'Chrome PDF Viewer', filename: 'mhjfbmdgcfjbbpaeojofohoefgiehjai' },
				{ name: 'Native Client', filename: 'internal-nacl-plugin' }
			]
		});

		// Overwrite the 'languages' property
		Object.defineProperty(navigator, 'languages', {
			get: () => ['en-US', 'en']
		});

		// Fix permissions
		const originalQuery = window.navigator.permissions.query;
		window.navigator.permissions.query = (parameters) => (
			parameters.name === 'notifications'
				? Promise.resolve({ state: Notification.permission })
				: originalQuery(parameters)
		);

		// Mock chrome object
		window.chrome = {
			runtime: {},
			loadTimes: function() {},
			csi: function() {},
			app: {}
		};

		// Fix broken image rendering
		const getContext = HTMLCanvasElement.prototype.getContext;
		HTMLCanvasElement.prototype.getContext = function(type, attributes) {
			if (type === 'webgl' || type === 'webgl2') {
				attributes = Object.assign({}, attributes, {
					preserveDrawingBuffer: true
				});
			}
			return getContext.call(this, type, attributes);
		};

		// Add realistic screen properties
		Object.defineProperty(screen, 'availWidth', { get: () => screen.width });
		Object.defineProperty(screen, 'availHeight', { get: () => screen.height - 40 });

		// Fix toStringing
		const oldToString = Function.prototype.toString;
		Function.prototype.toString = function() {
			if (this === window.navigator.permissions.query) {
				return 'function query() { [native code] }';
			}
			return oldToString.call(this);
		};
	`
}

// GetPage returns the current page
func (b *Browser) GetPage() *rod.Page {
	return b.page
}

// GetBrowser returns the browser instance
func (b *Browser) GetBrowser() *rod.Browser {
	return b.browser
}

// Navigate navigates to a URL with stealth measures
func (b *Browser) Navigate(url string) error {
	b.logger.BrowserAction("navigate", url)

	err := b.page.Navigate(url)
	if err != nil {
		return fmt.Errorf("navigation failed: %w", err)
	}

	b.stealth.PageLoadDelay()

	err = b.page.WaitLoad()
	if err != nil {
		return fmt.Errorf("page load failed: %w", err)
	}

	// Apply fingerprint masking after navigation
	b.stealth.ApplyFingerprintMasking(b.page)

	return nil
}

// TakeScreenshot takes a screenshot of the current page
func (b *Browser) TakeScreenshot(filename string) error {
	data, err := b.page.Screenshot(true, nil)
	if err != nil {
		return fmt.Errorf("screenshot failed: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create screenshot directory: %w", err)
	}

	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to save screenshot: %w", err)
	}

	b.logger.WithField("filename", filename).Info("Screenshot saved")
	return nil
}

// NewTab creates a new tab
func (b *Browser) NewTab() (*rod.Page, error) {
	page, err := b.browser.Page(proto.TargetCreateTarget{URL: "about:blank"})
	if err != nil {
		return nil, err
	}

	// Apply stealth settings to new tab
	page.EvalOnNewDocument(b.getStealthScript())

	return page, nil
}

// Close closes the browser
func (b *Browser) Close() error {
	b.logger.Info("Closing browser")

	if b.page != nil {
		b.page.Close()
	}

	if b.browser != nil {
		return b.browser.Close()
	}

	return nil
}

// WaitForSelector waits for an element to appear
func (b *Browser) WaitForSelector(selector string, timeout time.Duration) (*rod.Element, error) {
	return b.page.Timeout(timeout).Element(selector)
}

// GetCurrentURL returns the current page URL
func (b *Browser) GetCurrentURL() string {
	return b.page.MustInfo().URL
}

// Reload reloads the current page
func (b *Browser) Reload() error {
	return b.page.Reload()
}

// GoBack navigates back in history
func (b *Browser) GoBack() error {
	return b.page.NavigateBack()
}

// GoForward navigates forward in history
func (b *Browser) GoForward() error {
	return b.page.NavigateForward()
}

// GetHTML returns the page HTML
func (b *Browser) GetHTML() (string, error) {
	return b.page.HTML()
}

// IsElementPresent checks if an element is present on the page
func (b *Browser) IsElementPresent(selector string) bool {
	_, err := b.page.Timeout(2 * time.Second).Element(selector)
	return err == nil
}

// WaitForNavigation waits for navigation to complete
func (b *Browser) WaitForNavigation() error {
	wait := b.page.WaitNavigation(proto.PageLifecycleEventNameNetworkIdle)
	wait()
	return nil
}
