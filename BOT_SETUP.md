# Telegram Bot Setup Guide

## Bot Information
- **Bot Username**: [@chi_beposham_bot](https://t.me/chi_beposham_bot)
- **Token**: `8578746464:AAGCVHk0NMvy-TXKwTwplgu2iTJxAd9Hhbg`

## Quick Setup

### Option 1: Automated Setup (Recommended)

```bash
# Run the setup script
./scripts/setup-bot.sh

# Load environment and run
export $(cat .env.bot | xargs)
go run cmd/bot/main.go
```

### Option 2: Manual Setup

1. **Create `.env.bot` file**:

```bash
cp .env.bot.example .env.bot
```

2. **Update `.env.bot` with your configuration**:

```bash
TELEGRAM_BOT_TOKEN=8578746464:AAGCVHk0NMvy-TXKwTwplgu2iTJxAd9Hhbg
BOT_ENV=development
API_BASE_URL=http://localhost:8080
POSTGRES_DSN=host=185.202.113.229 port=5432 user=postgres password=A1212A1212a dbname=styler sslmode=disable
REDIS_URL=redis://localhost:6379/0
JWT_SECRET=95286739ac9475a2aac66036e01f18d34f18def61241df5f0aee472dfa3fdbc6c7522fe670226ed1910099bf59ecbedfce465677d15cfcda3558d6e7e9fd2c11
```

3. **Run the bot**:

```bash
# Load environment variables
export $(cat .env.bot | xargs)

# Run
go run cmd/bot/main.go
```

## Prerequisites

Before running the bot, ensure:

1. **Backend API is running** on `http://localhost:8080` (or update `API_BASE_URL`)
2. **PostgreSQL is accessible** at `185.202.113.229:5432`
3. **Redis is running** on `localhost:6379`

## Testing the Bot

1. Open Telegram
2. Search for [@chi_beposham_bot](https://t.me/chi_beposham_bot)
3. Click "Start" or send `/start`
4. Follow the Persian instructions to:
   - Enter your phone number
   - Verify OTP code
   - Upload images
   - Create conversions

## Health Check

Once the bot is running, verify it's healthy:

```bash
curl http://localhost:8081/health
```

Expected response:
```json
{
  "status": "ok",
  "timestamp": 1234567890,
  "service": "telegram-bot"
}
```

## Troubleshooting

### Bot not responding
- Check bot token is correct
- Verify backend API is running: `curl http://localhost:8080/api/health`
- Check database connection
- View bot logs for errors

### Database connection issues
- Verify PostgreSQL is accessible: `psql -h 185.202.113.229 -U postgres -d styler`
- Check firewall rules
- Verify credentials in `.env.bot`

### Redis connection issues
- Verify Redis is running: `redis-cli ping`
- Check Redis URL in `.env.bot`

## Production Deployment

For production:

1. Set `BOT_ENV=production`
2. Configure `WEBHOOK_URL` with your domain
3. Set up reverse proxy (Nginx) for webhook endpoint
4. Use environment variables or secrets management

See `docs/TELEGRAM_BOT.md` for detailed deployment instructions.

## Make Commands

```bash
# Run bot
make bot-run

# Build bot
make bot-build

# Run tests
make bot-test

# Docker build
make bot-docker-build

# Docker run
make bot-docker-run
```

## Support

For issues:
- Check logs: `docker-compose -f docker-compose.bot.yml logs -f telegram-bot`
- Review documentation: `docs/TELEGRAM_BOT.md`
- Check health endpoints: `http://localhost:8081/health`

