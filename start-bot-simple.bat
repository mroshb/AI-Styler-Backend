@echo off
REM Simple Telegram Bot Startup Script for Windows
REM This script sets environment variables directly

echo ========================================
echo Starting Telegram Bot
echo ========================================
echo.

REM Set environment variables
set TELEGRAM_BOT_TOKEN=8578746464:AAGCVHk0NMvy-TXKwTwplgu2iTJxAd9Hhbg
set BOT_ENV=development
set API_BASE_URL=http://localhost:8080
set POSTGRES_DSN=host=185.202.113.229 port=5432 user=postgres password=A1212A1212a dbname=styler sslmode=disable
set MAX_UPLOAD_SIZE=10MB
set HEALTH_PORT=8081
set RATE_LIMIT_MESSAGES=10
set RATE_LIMIT_CONVERSIONS=5
set RATE_LIMIT_WINDOW=1m
set JWT_SECRET=95286739ac9475a2aac66036e01f18d34f18def61241df5f0aee472dfa3fdbc6c7522fe670226ed1910099bf59ecbedfce465677d15cfcda3558d6e7e9fd2c11

echo [INFO] Environment variables set
echo [INFO] Starting bot...
echo.

REM Start the bot
go run cmd/bot/main.go

pause

