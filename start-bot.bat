@echo off
setlocal enabledelayedexpansion
REM Telegram Bot Startup Script for Windows

echo ========================================
echo Starting Telegram Bot
echo ========================================
echo.

REM Check if .env.bot exists
if not exist .env.bot (
    echo [ERROR] .env.bot file not found!
    echo Creating from example...
    if exist configs\example.env (
        copy configs\example.env .env.bot >nul
        echo [WARNING] Please edit .env.bot and set your TELEGRAM_BOT_TOKEN
        pause
        exit /b 1
    ) else (
        echo [ERROR] configs\example.env not found!
        pause
        exit /b 1
    )
)

echo [INFO] Loading environment variables from .env.bot...
echo.

REM Load environment variables from .env.bot
for /f "usebackq eol=# tokens=1,* delims==" %%a in (".env.bot") do (
    set "key=%%a"
    set "value=%%b"
    if not "!key!"=="" (
        if not "!value!"=="" (
            set "!key!=!value!"
        )
    )
)

REM Start the bot
echo [INFO] Starting bot...
echo.
go run cmd/bot/main.go

pause

