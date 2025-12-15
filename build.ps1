# LinkedIn Automation PoC - Build Script
# Run this script to build and run the application

param(
    [string]$Action = "build",
    [string]$Mode = "interactive",
    [string]$Search = "",
    [switch]$Verbose
)

$ErrorActionPreference = "Stop"

# Colors for output
function Write-ColorOutput($ForegroundColor) {
    $fc = $host.UI.RawUI.ForegroundColor
    $host.UI.RawUI.ForegroundColor = $ForegroundColor
    if ($args) {
        Write-Output $args
    }
    $host.UI.RawUI.ForegroundColor = $fc
}

function Write-Header($text) {
    Write-Host ""
    Write-Host "========================================" -ForegroundColor Cyan
    Write-Host "  $text" -ForegroundColor Cyan
    Write-Host "========================================" -ForegroundColor Cyan
    Write-Host ""
}

function Check-Go {
    try {
        $goVersion = go version 2>$null
        if ($goVersion) {
            Write-Host "Go is installed: $goVersion" -ForegroundColor Green
            return $true
        }
    } catch {
        return $false
    }
    return $false
}

function Install-Go-Instructions {
    Write-Host ""
    Write-Host "Go is not installed or not in PATH!" -ForegroundColor Red
    Write-Host ""
    Write-Host "Please install Go by following these steps:" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "1. Download Go from: https://go.dev/dl/" -ForegroundColor White
    Write-Host "2. Download the Windows MSI installer (go1.21.x.windows-amd64.msi)"
    Write-Host "3. Run the installer and follow the prompts"
    Write-Host "4. Restart your terminal/PowerShell after installation"
    Write-Host "5. Verify installation by running: go version"
    Write-Host ""
    Write-Host "Alternatively, use winget:" -ForegroundColor Yellow
    Write-Host "  winget install GoLang.Go"
    Write-Host ""
    Write-Host "Or use chocolatey:" -ForegroundColor Yellow
    Write-Host "  choco install golang"
    Write-Host ""
}

function Create-Directories {
    Write-Host "Creating necessary directories..."
    New-Item -ItemType Directory -Force -Path ".\data" | Out-Null
    New-Item -ItemType Directory -Force -Path ".\logs" | Out-Null
    New-Item -ItemType Directory -Force -Path ".\bin" | Out-Null
    Write-Host "Directories created." -ForegroundColor Green
}

function Install-Dependencies {
    Write-Header "Installing Dependencies"
    
    Write-Host "Running go mod download..."
    go mod download
    
    Write-Host "Running go mod tidy..."
    go mod tidy
    
    Write-Host "Dependencies installed successfully!" -ForegroundColor Green
}

function Build-Application {
    Write-Header "Building Application"
    
    Write-Host "Compiling..."
    go build -o .\bin\linkedin-automation.exe .\cmd\main.go
    
    if (Test-Path ".\bin\linkedin-automation.exe") {
        Write-Host "Build successful!" -ForegroundColor Green
        Write-Host "Executable: .\bin\linkedin-automation.exe"
    } else {
        Write-Host "Build failed!" -ForegroundColor Red
        exit 1
    }
}

function Run-Application {
    Write-Header "Running Application"
    
    $args = @("-mode=$Mode")
    
    if ($Search -ne "") {
        $args += "-search=`"$Search`""
    }
    
    if ($Verbose) {
        $args += "-verbose"
    }
    
    Write-Host "Running with arguments: $args" -ForegroundColor Yellow
    
    & go run .\cmd\main.go $args
}

function Show-Help {
    Write-Host ""
    Write-Host "LinkedIn Automation PoC - Build Script" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Usage: .\build.ps1 [-Action <action>] [-Mode <mode>] [-Search <query>] [-Verbose]"
    Write-Host ""
    Write-Host "Actions:" -ForegroundColor Yellow
    Write-Host "  init      - Create directories and check dependencies"
    Write-Host "  deps      - Install Go dependencies"
    Write-Host "  build     - Build the application (default)"
    Write-Host "  run       - Build and run the application"
    Write-Host "  test      - Run tests"
    Write-Host "  clean     - Clean build artifacts"
    Write-Host "  help      - Show this help message"
    Write-Host ""
    Write-Host "Modes (for run action):" -ForegroundColor Yellow
    Write-Host "  interactive - Open browser for manual interaction (default)"
    Write-Host "  search      - Search for profiles"
    Write-Host "  connect     - Search and send connection requests"
    Write-Host "  message     - Send follow-up messages"
    Write-Host "  full        - Complete automation workflow"
    Write-Host ""
    Write-Host "Examples:" -ForegroundColor Yellow
    Write-Host "  .\build.ps1 -Action init"
    Write-Host "  .\build.ps1 -Action build"
    Write-Host "  .\build.ps1 -Action run -Mode search -Search 'Software Engineer'"
    Write-Host "  .\build.ps1 -Action run -Mode interactive -Verbose"
    Write-Host ""
}

function Run-Tests {
    Write-Header "Running Tests"
    go test -v ./...
}

function Clean-Build {
    Write-Header "Cleaning Build Artifacts"
    
    if (Test-Path ".\bin") {
        Remove-Item -Recurse -Force ".\bin"
        Write-Host "Removed bin directory"
    }
    
    if (Test-Path ".\data") {
        Remove-Item -Recurse -Force ".\data"
        Write-Host "Removed data directory"
    }
    
    if (Test-Path ".\logs") {
        Remove-Item -Recurse -Force ".\logs"
        Write-Host "Removed logs directory"
    }
    
    Write-Host "Clean complete!" -ForegroundColor Green
}

# Main script logic
Write-Header "LinkedIn Automation PoC"

switch ($Action.ToLower()) {
    "help" {
        Show-Help
    }
    "init" {
        if (!(Check-Go)) {
            Install-Go-Instructions
            exit 1
        }
        Create-Directories
        Install-Dependencies
    }
    "deps" {
        if (!(Check-Go)) {
            Install-Go-Instructions
            exit 1
        }
        Install-Dependencies
    }
    "build" {
        if (!(Check-Go)) {
            Install-Go-Instructions
            exit 1
        }
        Create-Directories
        Install-Dependencies
        Build-Application
    }
    "run" {
        if (!(Check-Go)) {
            Install-Go-Instructions
            exit 1
        }
        Create-Directories
        Run-Application
    }
    "test" {
        if (!(Check-Go)) {
            Install-Go-Instructions
            exit 1
        }
        Run-Tests
    }
    "clean" {
        Clean-Build
    }
    default {
        Write-Host "Unknown action: $Action" -ForegroundColor Red
        Show-Help
        exit 1
    }
}

Write-Host ""
Write-Host "Done!" -ForegroundColor Green
