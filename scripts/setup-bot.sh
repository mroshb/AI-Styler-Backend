#!/bin/bash

# Setup script for Telegram Bot
# This script helps configure and run the Telegram bot

set -e

echo "=========================================="
echo "AI Styler Telegram Bot Setup"
echo "=========================================="
echo ""

# Check if .env.bot exists
if [ ! -f .env.bot ]; then
    echo "Creating .env.bot from template..."
    cp .env.bot.example .env.bot
    
    # Set the bot token
    echo ""
    echo "Setting bot token..."
    sed -i.bak "s/TELEGRAM_BOT_TOKEN=your-bot-token-from-botfather/TELEGRAM_BOT_TOKEN=8578746464:AAGCVHk0NMvy-TXKwTwplgu2iTJxAd9Hhbg/" .env.bot
    rm -f .env.bot.bak
    
    # Set database connection from main .env if it exists
    if [ -f .env ]; then
        echo "Copying database configuration from .env..."
        DB_HOST=$(grep "^DB_HOST=" .env | cut -d '=' -f2)
        DB_PORT=$(grep "^DB_PORT=" .env | cut -d '=' -f2)
        DB_USER=$(grep "^DB_USER=" .env | cut -d '=' -f2)
        DB_PASSWORD=$(grep "^DB_PASSWORD=" .env | cut -d '=' -f2)
        DB_NAME=$(grep "^DB_NAME=" .env | cut -d '=' -f2)
        
        if [ ! -z "$DB_HOST" ]; then
            POSTGRES_DSN="host=${DB_HOST} port=${DB_PORT} user=${DB_USER} password=${DB_PASSWORD} dbname=${DB_NAME} sslmode=disable"
            sed -i.bak "s|POSTGRES_DSN=.*|POSTGRES_DSN=${POSTGRES_DSN}|" .env.bot
            sed -i.bak "s|DB_HOST=.*|DB_HOST=${DB_HOST}|" .env.bot
            sed -i.bak "s|DB_PORT=.*|DB_PORT=${DB_PORT}|" .env.bot
            sed -i.bak "s|DB_USER=.*|DB_USER=${DB_USER}|" .env.bot
            sed -i.bak "s|DB_PASSWORD=.*|DB_PASSWORD=${DB_PASSWORD}|" .env.bot
            sed -i.bak "s|DB_NAME=.*|DB_NAME=${DB_NAME}|" .env.bot
            rm -f .env.bot.bak
        fi
        
        # Copy JWT secret
        JWT_SECRET=$(grep "^JWT_SECRET=" .env | cut -d '=' -f2)
        if [ ! -z "$JWT_SECRET" ]; then
            sed -i.bak "s|JWT_SECRET=.*|JWT_SECRET=${JWT_SECRET}|" .env.bot
            rm -f .env.bot.bak
        fi
    fi
    
    echo "✓ .env.bot created and configured"
else
    echo "✓ .env.bot already exists"
fi

echo ""
echo "=========================================="
echo "Configuration Summary"
echo "=========================================="
echo "Bot Token: $(grep TELEGRAM_BOT_TOKEN .env.bot | cut -d '=' -f2 | cut -c1-20)..."
echo "Bot Environment: $(grep BOT_ENV .env.bot | cut -d '=' -f2)"
echo "API Base URL: $(grep API_BASE_URL .env.bot | cut -d '=' -f2)"
echo "Database: $(grep DB_HOST .env.bot | cut -d '=' -f2)"
echo ""

echo "=========================================="
echo "Next Steps"
echo "=========================================="
echo "1. Review .env.bot and update if needed"
echo "2. Ensure backend API is running on $(grep API_BASE_URL .env.bot | cut -d '=' -f2)"
echo "3. Ensure PostgreSQL and Redis are running"
echo "4. Run the bot with:"
echo "   export \$(cat .env.bot | xargs) && go run cmd/bot/main.go"
echo "   OR"
echo "   make bot-run"
echo ""
echo "5. Test the bot:"
echo "   - Open Telegram and search for @chi_beposham_bot"
echo "   - Send /start command"
echo ""

