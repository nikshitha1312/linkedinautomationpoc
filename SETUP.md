# Setup Guide - LinkedIn Automation PoC

This guide walks you through setting up and running the LinkedIn Automation PoC project.

## Prerequisites

### 1. Install Go

#### Windows (Recommended Methods)

**Option A: Official Installer (Easiest)**
1. Download Go from [https://go.dev/dl/](https://go.dev/dl/)
2. Download the Windows MSI installer: `go1.23.4.windows-amd64.msi`
3. Run the installer and follow the prompts
4. The installer will set up PATH automatically
5. **Restart your terminal** after installation

**Option B: Using winget**
```powershell
winget install GoLang.Go
```

**Option C: Using Chocolatey**
```powershell
choco install golang
```

**Verify Installation:**
```powershell
go version
# Should output: go version go1.23.x windows/amd64
```

### 2. Install Git (if not already installed)

```powershell
winget install Git.Git
```

### 3. Chrome/Chromium Browser

The project uses Rod for browser automation, which requires Chrome or Chromium. If you have Chrome installed, you're good to go. Otherwise:

```powershell
winget install Google.Chrome
```

## Project Setup

### Step 1: Clone/Download the Project

```powershell
cd C:\Users\niksh\Desktop
git clone <your-repo-url> linkedinautomationpoc
cd linkedinautomationpoc
```

Or if you already have the files, navigate to the project:
```powershell
cd C:\Users\niksh\Desktop\linkedinautomationpoc
```

### Step 2: Configure Credentials

1. Copy the example environment file:
```powershell
copy .env.example .env
```

2. Edit the `.env` file with your LinkedIn credentials:
```powershell
notepad .env
```

Add your credentials:
```
LINKEDIN_EMAIL=your_email@example.com
LINKEDIN_PASSWORD=your_password
```

⚠️ **Never commit your `.env` file to git!**

### Step 3: Install Dependencies

```powershell
go mod download
go mod tidy
```

This will download all required packages:
- `github.com/go-rod/rod` - Browser automation
- `github.com/joho/godotenv` - Environment variable loading
- `github.com/mattn/go-sqlite3` - SQLite database
- `github.com/sirupsen/logrus` - Logging
- `gopkg.in/yaml.v3` - YAML configuration

### Step 4: Build the Application

```powershell
# Create output directories
mkdir -Force data, logs, bin

# Build
go build -o bin/linkedin-automation.exe ./cmd/main.go
```

Or use the quick start script:
```powershell
.\quickstart.bat
```

### Step 5: Run the Application

```powershell
# Interactive mode (opens browser for manual control)
.\bin\linkedin-automation.exe -mode=interactive

# Search mode
.\bin\linkedin-automation.exe -mode=search -search="Software Engineer" -location="San Francisco"

# See all options
.\bin\linkedin-automation.exe -help
```

## Quick Reference

### Command Line Options

| Flag | Description | Example |
|------|-------------|---------|
| `-mode` | Run mode | `-mode=search` |
| `-search` | Search query | `-search="Product Manager"` |
| `-company` | Company filter | `-company="Google"` |
| `-location` | Location filter | `-location="New York"` |
| `-max-results` | Max profiles | `-max-results=50` |
| `-dry-run` | Simulate only | `-dry-run` |
| `-verbose` | Debug logging | `-verbose` |
| `-config` | Config file | `-config=myconfig.yaml` |

### Available Modes

- `interactive` - Open browser for manual interaction
- `search` - Search for LinkedIn profiles
- `connect` - Search and send connection requests
- `message` - Send follow-up messages to new connections
- `full` - Complete automation workflow

### Example Commands

```powershell
# Search for software engineers in NYC
.\bin\linkedin-automation.exe -mode=search -search="Software Engineer" -location="New York"

# Connect with product managers (dry run first!)
.\bin\linkedin-automation.exe -mode=connect -search="Product Manager" -max-results=10 -dry-run

# Send follow-up messages
.\bin\linkedin-automation.exe -mode=message

# Full workflow with verbose logging
.\bin\linkedin-automation.exe -mode=full -verbose
```

## Configuration

### config.yaml

The `config.yaml` file contains all settings. Key sections:

```yaml
# Rate limits - Keep these conservative!
rate_limits:
  max_connections_per_day: 25
  max_messages_per_day: 50

# Stealth settings
stealth:
  mouse_overshoot: true
  typing_mistake_rate: 0.02
  
# Schedule - Only operate during work hours
schedule:
  enabled: true
  start_hour: 9
  end_hour: 18
```

### Message Templates

Customize connection notes and follow-up messages:

```yaml
messaging:
  connection_note_template: "Hi {{.FirstName}}, I came across your profile..."
  follow_up_message_template: "Thanks for connecting, {{.FirstName}}!"
```

Available variables: `{{.FirstName}}`, `{{.LastName}}`, `{{.FullName}}`, `{{.Company}}`, `{{.Headline}}`, `{{.Location}}`

## Troubleshooting

### "go is not recognized"
- Restart your terminal after installing Go
- Check if Go is in PATH: `echo $env:PATH`

### CGO/SQLite errors
SQLite requires CGO. On Windows, you may need:
```powershell
# Install TDM-GCC
winget install TDM-GCC.TDM-GCC
```

Or use a pure Go SQLite alternative by modifying the storage package.

### Browser not launching
- Ensure Chrome is installed
- Check if another automation is running
- Try setting `headless: true` in config.yaml

### Login failures
- Check credentials in `.env`
- LinkedIn may require 2FA - handle manually first
- Wait and retry if rate-limited

### Rate limit warnings
- Reduce `max_connections_per_day` in config
- Increase delays between actions
- Take longer breaks

## Project Structure

```
linkedinautomationpoc/
├── cmd/main.go          # Entry point
├── auth/                # Login & session
├── browser/             # Browser setup
├── config/              # Configuration
├── connection/          # Connect functionality
├── logger/              # Logging
├── messaging/           # Messaging
├── search/              # Search & profiles
├── stealth/             # Anti-detection
├── storage/             # Database
├── config.yaml          # Settings
├── .env                 # Credentials (DO NOT COMMIT)
└── README.md            # Documentation
```

## Running Tests

```powershell
go test ./... -v
```

## Support

This is an educational project. For issues:
1. Check the troubleshooting section
2. Review logs in `./logs/`
3. Run with `-verbose` for debug output

---

⚠️ **Remember**: This is for EDUCATIONAL PURPOSES ONLY. Do not use on production LinkedIn accounts.
