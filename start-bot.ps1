# Telegram Bot Startup Script for Windows PowerShell

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Starting Telegram Bot" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Check if .env.bot exists
if (-not (Test-Path ".env.bot")) {
    Write-Host "[ERROR] .env.bot file not found!" -ForegroundColor Red
    Write-Host "Creating from example..."
    if (Test-Path "configs\example.env") {
        Copy-Item "configs\example.env" ".env.bot"
        Write-Host "[WARNING] Please edit .env.bot and set your TELEGRAM_BOT_TOKEN" -ForegroundColor Yellow
        Read-Host "Press Enter to exit"
        exit 1
    } else {
        Write-Host "[ERROR] configs\example.env not found!" -ForegroundColor Red
        Read-Host "Press Enter to exit"
        exit 1
    }
}

Write-Host "[INFO] Loading environment variables from .env.bot..." -ForegroundColor Green
Write-Host ""

# Load environment variables from .env.bot
Get-Content ".env.bot" | ForEach-Object {
    $line = $_.Trim()
    if ($line -and -not $line.StartsWith("#")) {
        $parts = $line -split "=", 2
        if ($parts.Length -eq 2) {
            $key = $parts[0].Trim()
            $value = $parts[1].Trim()
            [Environment]::SetEnvironmentVariable($key, $value, "Process")
        }
    }
}

Write-Host "[INFO] Starting bot..." -ForegroundColor Green
Write-Host ""

# Start the bot
go run cmd/bot/main.go

Read-Host "Press Enter to exit"

