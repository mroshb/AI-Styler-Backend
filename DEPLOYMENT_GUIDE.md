# AI Styler Backend - Deployment Guide

## Overview

This guide provides comprehensive instructions for deploying the AI Styler backend service, a Go-based AI clothing try-on platform.

## Prerequisites

- Go 1.21 or higher
- PostgreSQL 13 or higher
- Redis 6 or higher
- Docker (optional, for containerized deployment)
- Gemini API key

## Quick Start

### 1. Clone and Setup

```bash
git clone <repository-url>
cd AI_Stayler
go mod download
```

### 2. Environment Configuration

Create a `.env` file based on `.env.example`:

```bash
cp .env.example .env
```

Edit `.env` with your configuration:

```env
# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password_here
DB_NAME=styler
DB_SSLMODE=disable

# Gemini AI Configuration
GEMINI_API_KEY=your_gemini_api_key_here
GEMINI_BASE_URL=https://generativelanguage.googleapis.com
GEMINI_MODEL=gemini-pro-vision
GEMINI_TIMEOUT=300
GEMINI_MAX_RETRIES=3

# JWT Configuration
JWT_SECRET=your-super-secret-jwt-key-change-in-production
JWT_ACCESS_TTL=15m
JWT_REFRESH_TTL=720h

# Redis Configuration
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# Storage Configuration
STORAGE_PATH=./uploads
UPLOAD_MAX_SIZE=10MB
SIGNED_URL_TTL=1h

# Monitoring Configuration
TELEGRAM_BOT_TOKEN=your_telegram_bot_token
TELEGRAM_CHAT_ID=your_telegram_chat_id
SENTRY_DSN=your_sentry_dsn
ENVIRONMENT=production
```

### 3. Database Setup

#### PostgreSQL Installation

**Ubuntu/Debian:**
```bash
sudo apt update
sudo apt install postgresql postgresql-contrib
sudo systemctl start postgresql
sudo systemctl enable postgresql
```

**macOS:**
```bash
brew install postgresql
brew services start postgresql
```

**Windows:**
Download and install from [PostgreSQL official website](https://www.postgresql.org/download/windows/)

#### Create Database and User

```sql
-- Connect to PostgreSQL as superuser
sudo -u postgres psql

-- Create database
CREATE DATABASE styler;

-- Create user
CREATE USER styler_user WITH PASSWORD 'your_password_here';

-- Grant privileges
GRANT ALL PRIVILEGES ON DATABASE styler TO styler_user;

-- Exit
\q
```

#### Run Migrations

```bash
# Run all migrations
go run scripts/migrate_main.go up

# Or run specific migration
go run scripts/migrate_main.go up 0013
```

### 4. Redis Setup

**Ubuntu/Debian:**
```bash
sudo apt install redis-server
sudo systemctl start redis-server
sudo systemctl enable redis-server
```

**macOS:**
```bash
brew install redis
brew services start redis
```

**Windows:**
Download from [Redis official website](https://redis.io/download)

### 5. Build and Run

```bash
# Build the application
go build -o ai-styler main.go

# Run the application
./ai-styler
```

The service will start on `http://localhost:8080`

## Docker Deployment

### 1. Create Dockerfile

```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o ai-styler main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/ai-styler .
COPY --from=builder /app/.env .

EXPOSE 8080
CMD ["./ai-styler"]
```

### 2. Docker Compose Setup

Create `docker-compose.yml`:

```yaml
version: '3.8'

services:
  app:
    build: .
    ports:
      - "8080:8080"
    environment:
      - DB_HOST=postgres
      - REDIS_HOST=redis
    depends_on:
      - postgres
      - redis
    volumes:
      - ./uploads:/app/uploads

  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: styler
      POSTGRES_USER: styler_user
      POSTGRES_PASSWORD: your_password_here
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./db/migrations:/docker-entrypoint-initdb.d
    ports:
      - "5432:5432"

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

volumes:
  postgres_data:
```

### 3. Run with Docker Compose

```bash
# Build and start services
docker-compose up -d

# View logs
docker-compose logs -f app

# Stop services
docker-compose down
```

## Production Deployment

### 1. System Requirements

- **CPU**: 2+ cores
- **RAM**: 4GB+ (8GB recommended)
- **Storage**: 50GB+ SSD
- **OS**: Ubuntu 20.04+ / CentOS 8+ / Windows Server 2019+

### 2. Security Configuration

#### Firewall Setup

**Ubuntu/Debian:**
```bash
sudo ufw allow 22/tcp
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw enable
```

#### SSL/TLS Setup

Use Let's Encrypt with nginx:

```bash
# Install nginx and certbot
sudo apt install nginx certbot python3-certbot-nginx

# Get SSL certificate
sudo certbot --nginx -d yourdomain.com

# Auto-renewal
sudo crontab -e
# Add: 0 12 * * * /usr/bin/certbot renew --quiet
```

#### Nginx Configuration

```nginx
server {
    listen 80;
    server_name yourdomain.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name yourdomain.com;

    ssl_certificate /etc/letsencrypt/live/yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/yourdomain.com/privkey.pem;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # File upload size limit
    client_max_body_size 50M;
}
```

### 3. Process Management

#### Using systemd

Create `/etc/systemd/system/ai-styler.service`:

```ini
[Unit]
Description=AI Styler Backend Service
After=network.target postgresql.service redis.service

[Service]
Type=simple
User=styler
Group=styler
WorkingDirectory=/opt/ai-styler
ExecStart=/opt/ai-styler/ai-styler
Restart=always
RestartSec=5
Environment=ENVIRONMENT=production

[Install]
WantedBy=multi-user.target
```

Enable and start service:

```bash
sudo systemctl daemon-reload
sudo systemctl enable ai-styler
sudo systemctl start ai-styler
sudo systemctl status ai-styler
```

### 4. Monitoring Setup

#### Health Checks

The service provides health endpoints:

- `GET /api/health` - Basic health check
- `GET /api/health/detailed` - Detailed health information

#### Log Management

Configure log rotation:

```bash
sudo nano /etc/logrotate.d/ai-styler
```

Add:
```
/var/log/ai-styler/*.log {
    daily
    missingok
    rotate 30
    compress
    delaycompress
    notifempty
    create 644 styler styler
    postrotate
        systemctl reload ai-styler
    endscript
}
```

#### Monitoring with Prometheus

Add Prometheus metrics endpoint:

```go
// In your service
http.Handle("/metrics", promhttp.Handler())
```

## Backup and Recovery

### 1. Database Backup

```bash
# Create backup
pg_dump -h localhost -U styler_user -d styler > backup_$(date +%Y%m%d_%H%M%S).sql

# Restore backup
psql -h localhost -U styler_user -d styler < backup_file.sql
```

### 2. File Storage Backup

```bash
# Backup uploads directory
tar -czf uploads_backup_$(date +%Y%m%d_%H%M%S).tar.gz ./uploads/

# Restore uploads
tar -xzf uploads_backup_file.tar.gz
```

### 3. Automated Backup Script

Create `backup.sh`:

```bash
#!/bin/bash
BACKUP_DIR="/opt/backups"
DATE=$(date +%Y%m%d_%H%M%S)

# Database backup
pg_dump -h localhost -U styler_user -d styler > $BACKUP_DIR/db_backup_$DATE.sql

# File backup
tar -czf $BACKUP_DIR/uploads_backup_$DATE.tar.gz /opt/ai-styler/uploads/

# Keep only last 7 days
find $BACKUP_DIR -name "*.sql" -mtime +7 -delete
find $BACKUP_DIR -name "*.tar.gz" -mtime +7 -delete
```

Schedule with cron:

```bash
crontab -e
# Add: 0 2 * * * /opt/ai-styler/backup.sh
```

## Troubleshooting

### Common Issues

#### 1. Database Connection Issues

```bash
# Check PostgreSQL status
sudo systemctl status postgresql

# Check connection
psql -h localhost -U styler_user -d styler -c "SELECT 1;"
```

#### 2. Redis Connection Issues

```bash
# Check Redis status
sudo systemctl status redis

# Test connection
redis-cli ping
```

#### 3. File Permission Issues

```bash
# Fix uploads directory permissions
sudo chown -R styler:styler /opt/ai-styler/uploads/
sudo chmod -R 755 /opt/ai-styler/uploads/
```

#### 4. Port Conflicts

```bash
# Check if port 8080 is in use
sudo netstat -tlnp | grep :8080

# Kill process if needed
sudo kill -9 <PID>
```

### Log Analysis

```bash
# View application logs
sudo journalctl -u ai-styler -f

# View nginx logs
sudo tail -f /var/log/nginx/access.log
sudo tail -f /var/log/nginx/error.log
```

## Performance Optimization

### 1. Database Optimization

```sql
-- Add indexes for better performance
CREATE INDEX CONCURRENTLY idx_conversions_user_id ON conversions(user_id);
CREATE INDEX CONCURRENTLY idx_conversions_status ON conversions(status);
CREATE INDEX CONCURRENTLY idx_images_user_id ON images(user_id);
CREATE INDEX CONCURRENTLY idx_images_type ON images(type);
```

### 2. Application Optimization

- Enable connection pooling
- Use Redis for caching
- Implement request rate limiting
- Use CDN for static files

### 3. Monitoring Performance

- Monitor CPU and memory usage
- Track database query performance
- Monitor API response times
- Set up alerts for critical metrics

## Security Best Practices

1. **Environment Variables**: Never commit sensitive data to version control
2. **Database Security**: Use strong passwords and limit access
3. **Network Security**: Use firewalls and VPNs
4. **Regular Updates**: Keep all dependencies updated
5. **Backup Security**: Encrypt backup files
6. **Access Control**: Implement proper user roles and permissions

## Support

For issues and questions:

1. Check the logs first
2. Review this deployment guide
3. Check GitHub issues
4. Contact the development team

## API Documentation

Once deployed, API documentation is available at:
- Swagger UI: `https://yourdomain.com/api/docs`
- OpenAPI Spec: `https://yourdomain.com/api/openapi.json`

## Environment Variables Reference

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `DB_HOST` | Database host | localhost | Yes |
| `DB_PORT` | Database port | 5432 | Yes |
| `DB_USER` | Database user | - | Yes |
| `DB_PASSWORD` | Database password | - | Yes |
| `DB_NAME` | Database name | styler | Yes |
| `GEMINI_API_KEY` | Gemini API key | - | Yes |
| `JWT_SECRET` | JWT signing secret | - | Yes |
| `REDIS_HOST` | Redis host | localhost | Yes |
| `REDIS_PORT` | Redis port | 6379 | Yes |
| `STORAGE_PATH` | File storage path | ./uploads | Yes |
| `ENVIRONMENT` | Environment (dev/prod) | development | Yes |
| `TELEGRAM_BOT_TOKEN` | Telegram bot token | - | No |
| `TELEGRAM_CHAT_ID` | Telegram chat ID | - | No |
| `SENTRY_DSN` | Sentry DSN | - | No |