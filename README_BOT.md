# Telegram Bot Quick Start

## Bot Information
- **Bot Username**: [@chi_beposham_bot](https://t.me/chi_beposham_bot)
- **Token**: Configured in `.env.bot`

## Quick Start

### 1. Configure Environment

Copy the example file and update with your values:

```bash
cp .env.bot.example .env.bot
```

Or use the provided `.env.bot` file which already has the bot token configured.

### 2. Run the Bot

#### Option A: Direct Run
```bash
# Load environment variables
export $(cat .env.bot | xargs)

# Run the bot
go run cmd/bot/main.go
```

#### Option B: Using Make
```bash
# Set environment variables first
export $(cat .env.bot | xargs)

# Run with make
make bot-run
```

#### Option C: Docker Compose
```bash
# Update docker-compose.bot.yml with your environment variables
# Then run:
docker-compose -f docker-compose.bot.yml up
```

### 3. Test the Bot

1. Open Telegram and search for [@chi_beposham_bot](https://t.me/chi_beposham_bot)
2. Send `/start` command
3. Follow the Persian instructions to:
   - Authenticate with phone number
   - Upload images
   - Create conversions
   - View conversion history

## Environment Variables

The bot requires these environment variables (see `.env.bot`):

- `TELEGRAM_BOT_TOKEN` - Your bot token (already configured)
- `API_BASE_URL` - Backend API URL (default: http://localhost:8080)
- `POSTGRES_DSN` - Database connection string
- `REDIS_URL` - Redis connection URL
- `JWT_SECRET` - JWT secret (must match backend)

## Health Check

Once running, check bot health:

```bash
curl http://localhost:8081/health
```

## Monitoring

- Health endpoint: http://localhost:8081/health
- Metrics endpoint: http://localhost:8081/metrics
- Readiness check: http://localhost:8081/health/ready
- Liveness check: http://localhost:8081/health/live

## Troubleshooting

### Bot not responding
1. Check bot token is correct
2. Verify backend API is running and accessible
3. Check database and Redis connections
4. View logs for errors

### Authentication issues
1. Ensure backend API is running
2. Verify OTP service is configured
3. Check database connection

### Rate limiting
1. Check Redis connection
2. Verify rate limit configuration

## Production Deployment

For production, set:
- `BOT_ENV=production`
- `WEBHOOK_URL=https://yourdomain.com`
- Configure reverse proxy (Nginx) for webhook endpoint

See `docs/TELEGRAM_BOT.md` for detailed deployment instructions.

