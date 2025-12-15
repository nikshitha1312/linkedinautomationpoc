@echo off
REM LinkedIn Automation PoC - Quick Start Script

echo.
echo ========================================
echo   LinkedIn Automation PoC - Quick Start
echo ========================================
echo.

REM Check if Go is installed
where go >nul 2>nul
if %ERRORLEVEL% NEQ 0 (
    echo ERROR: Go is not installed!
    echo.
    echo Please install Go from: https://go.dev/dl/
    echo Or run: winget install GoLang.Go
    echo.
    pause
    exit /b 1
)

REM Show Go version
echo Go version:
go version
echo.

REM Create directories
if not exist "data" mkdir data
if not exist "logs" mkdir logs
if not exist "bin" mkdir bin

REM Check if .env exists
if not exist ".env" (
    echo WARNING: .env file not found!
    echo Copying .env.example to .env...
    copy .env.example .env
    echo.
    echo Please edit .env and add your LinkedIn credentials.
    echo.
    notepad .env
)

REM Install dependencies
echo Installing dependencies...
go mod download
go mod tidy

REM Build
echo.
echo Building application...
go build -o bin\linkedin-automation.exe cmd\main.go

if exist "bin\linkedin-automation.exe" (
    echo.
    echo Build successful!
    echo.
    echo Run the application with:
    echo   bin\linkedin-automation.exe -mode=interactive
    echo   bin\linkedin-automation.exe -mode=search -search="Software Engineer"
    echo   bin\linkedin-automation.exe -help
    echo.
) else (
    echo.
    echo Build failed!
    echo.
)

pause
