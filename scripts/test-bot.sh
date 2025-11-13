#!/bin/bash

# Test script for Telegram bot
# This script helps diagnose bot startup issues

echo "=== Telegram Bot Diagnostic Tool ==="
echo ""

# Check if .env.bot exists
if [ ! -f ".env.bot" ]; then
    echo "❌ .env.bot file not found!"
    echo "Creating from example..."
    if [ -f "configs/example.env" ]; then
        cp configs/example.env .env.bot
        echo "✅ Created .env.bot from example"
        echo "⚠️  Please edit .env.bot and set your TELEGRAM_BOT_TOKEN"
        exit 1
    else
        echo "❌ configs/example.env not found either!"
        exit 1
    fi
fi

# Load environment variables
export $(cat .env.bot | grep -v '^#' | xargs)

# Check TELEGRAM_BOT_TOKEN
if [ -z "$TELEGRAM_BOT_TOKEN" ]; then
    echo "❌ TELEGRAM_BOT_TOKEN is not set in .env.bot"
    exit 1
fi

echo "✅ TELEGRAM_BOT_TOKEN is set: ${TELEGRAM_BOT_TOKEN:0:10}..."
echo ""

# Test Telegram API connection
echo "Testing Telegram API connection..."
RESPONSE=$(curl -s "https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/getMe")
if echo "$RESPONSE" | grep -q '"ok":true'; then
    BOT_USERNAME=$(echo "$RESPONSE" | grep -o '"username":"[^"]*' | cut -d'"' -f4)
    BOT_ID=$(echo "$RESPONSE" | grep -o '"id":[0-9]*' | cut -d':' -f2)
    echo "✅ Bot authenticated successfully!"
    echo "   Username: @${BOT_USERNAME}"
    echo "   ID: ${BOT_ID}"
else
    echo "❌ Failed to authenticate with Telegram API"
    echo "   Response: $RESPONSE"
    echo "   Please check your TELEGRAM_BOT_TOKEN"
    exit 1
fi

echo ""

# Check database connection (if configured)
if [ ! -z "$POSTGRES_DSN" ]; then
    echo "Checking PostgreSQL connection..."
    # Extract connection details (simplified check)
    if echo "$POSTGRES_DSN" | grep -q "host="; then
        echo "✅ POSTGRES_DSN is configured"
        echo "   Note: Actual connection test requires running the bot"
    else
        echo "⚠️  POSTGRES_DSN format may be incorrect"
    fi
else
    echo "⚠️  POSTGRES_DSN not configured (bot may fail to start)"
fi

echo ""

# Check Redis connection (optional)
if [ ! -z "$REDIS_URL" ] || [ ! -z "$REDIS_HOST" ]; then
    echo "✅ Redis is configured (optional, bot will continue without it)"
else
    echo "⚠️  Redis not configured (optional, bot will continue without it)"
fi

echo ""
echo "=== Summary ==="
echo "✅ Configuration file: .env.bot exists"
echo "✅ Bot token: Configured and valid"
echo "✅ Telegram API: Connection successful"
echo ""
echo "You can now start the bot with:"
echo "  export \$(cat .env.bot | xargs)"
echo "  go run cmd/bot/main.go"
echo ""
echo "Or use Make:"
echo "  make bot-run"
echo ""

