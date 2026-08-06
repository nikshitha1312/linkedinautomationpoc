# LinkedIn Automation PoC

A comprehensive technical proof-of-concept demonstrating advanced browser automation, anti-detection techniques, and clean Go architecture. This project showcases sophisticated automation tools while implementing human-like behavior patterns and stealth mechanisms.


## 🎯 Project Overview

This Go-based LinkedIn automation tool using the [Rod library](https://github.com/go-rod/rod) demonstrates:

- **Advanced browser automation capabilities**
- **Human-like behavior simulation**
- **Sophisticated anti-bot detection techniques**
- **Clean, modular Go architecture**

---

## 📁 Project Structure

```
linkedinautomationpoc/
├── cmd/
│   └── main.go              # Main application entry point
├── auth/
│   └── auth.go              # Authentication system
├── browser/
│   └── browser.go           # Browser management with stealth
├── config/
│   └── config.go            # Configuration management
├── connection/
│   └── connection.go        # Connection request handling
├── logger/
│   └── logger.go            # Structured logging
├── messaging/
│   └── messaging.go         # Messaging system
├── search/
│   └── search.go            # Search & targeting
├── stealth/
│   └── stealth.go           # Anti-detection techniques
├── storage/
│   └── database.go          # SQLite persistence
├── config.yaml              # Configuration file
├── .env.example             # Environment template
├── go.mod                   # Go module definition
└── README.md               # This file
```

---

## 🛡️ Implemented Anti-Detection Techniques

This project implements **8 stealth techniques** to simulate authentic human behavior:

### 1. Human-like Mouse Movement (MANDATORY ✓)
- **Bézier curves** for natural curved paths
- **Variable speed** with acceleration/deceleration
- **Natural overshoot** past targets with correction
- **Micro-corrections** near the destination

```go
// Example: Mouse moves in curved paths, not straight lines
stealth.MoveMouse(page, targetX, targetY)
```

### 2. Randomized Timing Patterns (MANDATORY ✓)
- Realistic delays between actions
- Variable think time (1-5 seconds)
- Page load wait variations
- Cognitive processing simulation

```go
stealth.ThinkingDelay()  // Simulates human reading/thinking
stealth.ActionDelay()    // Random delay between actions
```

### 3. Browser Fingerprint Masking (MANDATORY ✓)
- Removes `navigator.webdriver` flag
- Spoofs browser plugins
- Overrides languages and permissions
- Masks automation properties

```go
stealth.ApplyFingerprintMasking(page)
```

### 4. Random Scrolling Behavior
- Variable scroll speeds
- Natural acceleration/deceleration
- Occasional scroll-back movements (15% chance)
- Viewport-aware scrolling

### 5. Realistic Typing Simulation
- Variable keystroke intervals (50-200ms)
- Occasional typos with corrections (2% rate)
- Adjacent key mistakes (QWERTY-aware)
- Human typing rhythm variations

### 6. Mouse Hovering & Movement
- Random hover events over elements
- Natural cursor wandering
- Realistic movement patterns during idle

### 7. Activity Scheduling
- Operates only during business hours (9 AM - 6 PM)
- Work days only (Monday-Friday)
- Realistic break patterns (5-15 minutes)
- Session duration limits (2 hours)

### 8. Rate Limiting & Throttling
- Daily connection limits (25/day)
- Message limits (50/day)
- Profile view limits (100/day)
- Cooldown periods between actions

---

## 🚀 Getting Started

### Prerequisites

- Go 1.21 or higher
- Chrome/Chromium browser installed
- Git

### Installation

1. **Clone the repository**
```bash
git clone https://github.com/yourusername/linkedin-automation-poc.git
cd linkedin-automation-poc
```

2. **Install dependencies**
```bash
go mod download
```

3. **Configure environment**
```bash
# Copy the example environment file
cp .env.example .env

# Edit .env with your credentials
# LINKEDIN_EMAIL=your_email@example.com
# LINKEDIN_PASSWORD=your_password
```

4. **Customize configuration (optional)**
```bash
# Edit config.yaml to customize behavior
```

### Building

```bash
# Build the application
go build -o linkedin-automation ./cmd/main.go

# Or run directly
go run ./cmd/main.go
```

### Running

```bash
# Interactive mode (opens browser for manual interaction)
./linkedin-automation -mode=interactive

# Search mode (search for profiles)
./linkedin-automation -mode=search -search="Software Engineer" -location="San Francisco"

# Connect mode (search and send connection requests)
./linkedin-automation -mode=connect -search="Product Manager" -max-results=10

# Message mode (check new connections and send follow-ups)
./linkedin-automation -mode=message

# Full workflow (complete automation cycle)
./linkedin-automation -mode=full

# Dry run (no actual actions)
./linkedin-automation -mode=connect -search="Developer" -dry-run

# Verbose logging
./linkedin-automation -mode=search -verbose
```

### Command Line Flags

| Flag | Description | Default |
|------|-------------|---------|
| `-config` | Path to configuration file | `config.yaml` |
| `-mode` | Run mode: interactive, search, connect, message, full | `interactive` |
| `-search` | Search query (job title, keywords) | - |
| `-company` | Company filter | - |
| `-location` | Location filter | - |
| `-max-results` | Maximum search results | `25` |
| `-dry-run` | Simulate without actions | `false` |
| `-verbose` | Enable debug logging | `false` |

---

## ⚙️ Configuration

### Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `LINKEDIN_EMAIL` | LinkedIn login email | ✓ |
| `LINKEDIN_PASSWORD` | LinkedIn password | ✓ |
| `BROWSER_HEADLESS` | Run browser in headless mode | |
| `LOG_LEVEL` | Logging level (debug/info/warn/error) | |
| `MAX_CONNECTIONS_PER_DAY` | Daily connection limit | |

### YAML Configuration

See `config.yaml` for full configuration options including:

- Browser settings
- Stealth parameters
- Rate limits
- Search defaults
- Message templates
- Scheduling options

### Message Templates

Templates support dynamic variables:

```yaml
connection_note_template: "Hi {{.FirstName}}, I came across your profile and was impressed by your work at {{.Company}}!"
follow_up_message_template: "Thanks for connecting, {{.FirstName}}! I'd love to hear about your experience."
```

Available variables: `{{.FirstName}}`, `{{.LastName}}`, `{{.FullName}}`, `{{.Company}}`, `{{.Headline}}`, `{{.Location}}`

---

## 💾 Data Persistence

The tool uses SQLite for state persistence:

- **Profiles**: Stores discovered LinkedIn profiles
- **Connection Requests**: Tracks sent requests and their status
- **Messages**: Records sent messages
- **Daily Stats**: Activity statistics
- **Session Cookies**: For session restoration

Database location: `./data/linkedin_automation.db`

---

## 📊 Code Quality Standards

### Modular Architecture
- Clear separation of concerns
- Well-defined interfaces
- Logical package organization

### Error Handling
- Comprehensive error detection
- Graceful degradation
- Retry mechanisms
- Detailed error logging

### Structured Logging
- Leveled logging (debug, info, warn, error)
- Contextual information
- JSON and text formats
- File and console output

---

## 🔒 Security Considerations

1. **Never commit `.env` file** - It contains credentials
2. **Use environment variables** - For sensitive configuration
3. **Review rate limits** - Avoid triggering security measures
4. **Monitor for 2FA/Captcha** - Handle security checkpoints

---

## 📝 Sample Output

```
╔══════════════════════════════════════════════════════════════════╗
║           LinkedIn Automation PoC - Educational Only             ║
╚══════════════════════════════════════════════════════════════════╝

INFO[2024-12-15 10:30:00] LinkedIn Automation PoC starting...
INFO[2024-12-15 10:30:00] Mode: search
INFO[2024-12-15 10:30:02] Browser launched successfully
INFO[2024-12-15 10:30:05] Authentication successful!
INFO[2024-12-15 10:30:05] Logged in as: John Doe
INFO[2024-12-15 10:30:05] === Today's Activity ===
INFO[2024-12-15 10:30:05]   Connections Sent: 5 / 25
INFO[2024-12-15 10:30:05]   Messages Sent: 2 / 50
INFO[2024-12-15 10:30:10] Starting search: Software Engineer
INFO[2024-12-15 10:30:25] Found 25 profiles
```

---

## 🧪 Testing

```bash
# Run tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test ./stealth/...
```

---

## 🤝 Contributing

This is an educational project. Contributions should focus on:

- Improving anti-detection techniques
- Better error handling
- Code quality improvements
- Documentation

---

## 📄 License

This project is for educational purposes only. No license is granted for commercial or production use.

---

## 📚 Resources

- [Rod Browser Automation](https://github.com/go-rod/rod)
- [Go Documentation](https://golang.org/doc/)
- [LinkedIn Terms of Service](https://www.linkedin.com/legal/user-agreement)

---

## ✉️ Submission

**Repository**: Include all source code with proper Go module configuration
**Environment Template**: `.env.example` with documented variables
**Documentation**: This README with setup instructions

---

*Built for technical demonstration purposes only.*
