# Telegram Bot Documentation

## Overview

The AI Styler Telegram Bot is a production-ready bot service that integrates with the AI Styler Backend API. It provides Persian-language UI for users to upload images, create conversions, and manage their conversion history.

## Features

- **Persian Language UI**: All user-facing messages are in Persian (Farsi)
- **OTP-based Authentication**: Secure phone number verification
- **Image Conversion**: Upload images and create AI-powered style conversions
- **Conversion Management**: View and manage conversion history
- **Rate Limiting**: Redis-based rate limiting to prevent abuse
- **Monitoring**: Prometheus metrics and health check endpoints
- **Webhook & Polling**: Supports both webhook (production) and polling (development) modes

## Architecture

```
┌─────────────┐
│ Telegram    │
│ Users       │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ Telegram    │
│ Bot Service │
└──────┬──────┘
       │
       ├──► PostgreSQL (Sessions)
       ├──► Redis (Tokens, Rate Limiting)
       └──► Backend API (Conversions, Images, Auth)
```

## Prerequisites

- Go 1.21+
- PostgreSQL 15+
- Redis 7+
- Docker & Docker Compose (optional)
- Telegram Bot Token from [@BotFather](https://t.me/BotFather)

## Quick Start

### 1. Get Bot Token

1. Open Telegram and search for [@BotFather](https://t.me/BotFather)
2. Send `/newbot` command
3. Follow instructions to create a bot
4. Copy the bot token (format: `123456789:ABCdefGHIjklMNOpqrsTUVwxyz`)

### 2. Configure Environment

Copy the example environment file:

```bash
cp configs/example.env .env
```

Edit `.env` and set required variables:

```bash
TELEGRAM_BOT_TOKEN=your-bot-token-here
API_BASE_URL=http://localhost:8080
POSTGRES_DSN=host=localhost port=5432 user=postgres password=yourpassword dbname=styler sslmode=disable
REDIS_URL=redis://localhost:6379/0
```

### 3. Run with Docker Compose

```bash
docker-compose -f docker-compose.bot.yml up -d
```

### 4. Run Locally

```bash
# Install dependencies
go mod download

# Run the bot
go run cmd/bot/main.go
```

## Configuration

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `TELEGRAM_BOT_TOKEN` | Bot token from BotFather | - | ✅ |
| `API_BASE_URL` | Backend API base URL | `http://localhost:8080` | ✅ |
| `API_KEY_FOR_BOT` | Optional API key for bot-to-API auth | - | ❌ |
| `POSTGRES_DSN` | PostgreSQL connection string | - | ✅ |
| `REDIS_URL` | Redis connection URL | - | ✅ |
| `BOT_ENV` | Environment: `development` or `production` | `development` | ❌ |
| `MAX_UPLOAD_SIZE` | Maximum upload size | `10MB` | ❌ |
| `RATE_LIMIT_MESSAGES` | Messages per minute per user | `10` | ❌ |
| `RATE_LIMIT_CONVERSIONS` | Conversions per hour per user | `5` | ❌ |
| `WEBHOOK_URL` | Webhook URL (production) | - | ❌ |
| `WEBHOOK_PORT` | Webhook server port | `8443` | ❌ |
| `HEALTH_PORT` | Health check server port | `8081` | ❌ |

## Deployment

### Development Mode (Polling)

The bot runs in polling mode by default when `BOT_ENV=development`. This is suitable for local development.

```bash
BOT_ENV=development go run cmd/bot/main.go
```

### Production Mode (Webhook)

For production, use webhook mode for better performance and reliability.

#### 1. Set Webhook URL

```bash
export WEBHOOK_URL=https://yourdomain.com
export BOT_ENV=production
```

#### 2. Configure Reverse Proxy (Nginx)

```nginx
server {
    listen 443 ssl;
    server_name yourdomain.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location /webhook {
        proxy_pass http://localhost:8443;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

#### 3. Run Bot

```bash
BOT_ENV=production WEBHOOK_URL=https://yourdomain.com go run cmd/bot/main.go
```

### Docker Deployment

```bash
# Build image
docker build -f Dockerfile.bot -t ai-styler-bot .

# Run container
docker run -d \
  --name ai-styler-bot \
  -e TELEGRAM_BOT_TOKEN=your-token \
  -e API_BASE_URL=https://api.yourdomain.com \
  -e POSTGRES_DSN="host=db user=postgres password=pass dbname=styler" \
  -e REDIS_URL=redis://redis:6379/0 \
  -p 8081:8081 \
  -p 8443:8443 \
  ai-styler-bot
```

### Kubernetes Deployment

See `deploy/bot-deploy.yml` for Kubernetes deployment example.

## Database Schema

The bot creates a `telegram_sessions` table automatically on startup:

```sql
CREATE TABLE telegram_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    telegram_user_id BIGINT UNIQUE NOT NULL,
    backend_user_id UUID,
    phone VARCHAR(20),
    access_token TEXT,
    refresh_token TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

## User Flow

### 1. Start Command

User sends `/start` → Bot shows welcome message with main menu

### 2. Authentication

- User clicks "شروع تبدیل تصویر" (Start Conversion)
- Bot requests phone number
- User enters phone number
- Bot sends OTP via backend API
- User enters OTP code
- Bot verifies OTP and creates/updates user session

### 3. Image Conversion

- User uploads their photo
- Bot uploads to backend API
- User uploads clothing/garment image
- Bot uploads to backend API
- User selects style from inline keyboard
- User confirms conversion
- Bot creates conversion via API
- Bot polls conversion status and shows progress
- Bot delivers result image when completed

### 4. My Conversions

- User clicks "تبدیل‌های من" (My Conversions)
- Bot fetches conversions from API
- Bot displays list with pagination
- User can view individual conversions

## Monitoring

### Health Endpoints

- `GET /health` - General health check
- `GET /health/ready` - Readiness check (checks DB and Redis)
- `GET /health/live` - Liveness check
- `GET /metrics` - Prometheus metrics

### Prometheus Metrics

- `telegram_updates_total` - Total updates received
- `telegram_processing_duration_seconds` - Processing time
- `telegram_errors_total` - Error count by type
- `telegram_active_users` - Active user count
- `telegram_conversions_total` - Conversion count by status
- `telegram_api_requests_total` - API request count
- `telegram_rate_limit_hits_total` - Rate limit hits

### Example Queries

```promql
# Requests per minute
rate(telegram_updates_total[1m])

# Error rate
rate(telegram_errors_total[5m])

# Average processing time
rate(telegram_processing_duration_seconds_sum[5m]) / rate(telegram_processing_duration_seconds_count[5m])
```

## Security

### Best Practices

1. **Never commit tokens**: Always use environment variables
2. **Use HTTPS**: Always use HTTPS for webhook URLs in production
3. **Rate Limiting**: Configure appropriate rate limits
4. **Input Validation**: All user inputs are validated
5. **Token Storage**: Tokens stored securely in Redis with TTL
6. **Error Handling**: No sensitive data in error messages

### Security Checklist

- [ ] Bot token stored in environment variable
- [ ] Webhook URL uses HTTPS
- [ ] Database credentials secured
- [ ] Redis password set (if required)
- [ ] Rate limiting configured
- [ ] Health endpoints protected (if exposed)
- [ ] Logs don't contain sensitive data
- [ ] Regular security updates

## Troubleshooting

### Bot Not Responding

1. Check bot token is correct
2. Verify bot is running: `curl http://localhost:8081/health`
3. Check logs for errors
4. Verify database and Redis connections

### Webhook Not Working

1. Verify webhook URL is accessible
2. Check SSL certificate is valid
3. Verify reverse proxy configuration
4. Check bot logs for webhook errors

### Authentication Issues

1. Verify backend API is accessible
2. Check OTP service is working
3. Verify database connection
4. Check session storage

### Rate Limiting Issues

1. Check Redis connection
2. Verify rate limit configuration
3. Check rate limit metrics

## Development

### Running Tests

```bash
# Unit tests
go test ./tests/telegram/...

# Integration tests
go test -tags=integration ./tests/telegram/...
```

### Code Structure

```
cmd/bot/
  main.go                    # Entry point
internal/telegram/
  bot.go                     # Bot service
  handlers.go                # Message handlers
  keyboards.go               # Inline keyboards
  messages.go                # Persian messages
  api_client.go              # Backend API client
  storage.go                 # Database storage
  session.go                 # Session management
  rate_limiter.go            # Rate limiting
  metrics.go                 # Prometheus metrics
  health.go                  # Health endpoints
  config.go                  # Configuration
```

## Support

For issues and questions:

- GitHub Issues: [Create an issue](https://github.com/your-org/ai-styler-backend/issues)
- Documentation: See main README.md
- API Documentation: See API_DOCUMENTATION.md

## License

See LICENSE file for details.

