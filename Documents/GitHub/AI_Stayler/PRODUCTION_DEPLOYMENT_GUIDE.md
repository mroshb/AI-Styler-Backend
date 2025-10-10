# AI Styler Backend - Production Deployment Guide

## Overview

This guide provides step-by-step instructions for deploying the AI Styler backend service to production environments. The system is designed to be scalable, secure, and maintainable.

## Architecture Overview

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Load Balancer │────│   Web Servers   │────│   API Gateway   │
│   (Nginx)       │    │   (Multiple)    │    │   (Optional)    │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │
                ┌───────────────┼───────────────┐
                │               │               │
        ┌───────▼──────┐ ┌──────▼──────┐ ┌──────▼──────┐
        │   Auth       │ │ Conversion  │ │   Payment   │
        │   Service    │ │   Service   │ │   Service   │
        └──────────────┘ └─────────────┘ └─────────────┘
                │               │               │
        ┌───────▼──────┐ ┌──────▼──────┐ ┌──────▼──────┐
        │ PostgreSQL  │ │   Redis     │ │   Gemini    │
        │  Database   │ │   Cache     │ │     AI      │
        └─────────────┘ └─────────────┘ └─────────────┘
```

## Prerequisites

### System Requirements

- **CPU**: 4+ cores (8+ recommended)
- **RAM**: 8GB+ (16GB recommended)
- **Storage**: 100GB+ SSD
- **OS**: Ubuntu 20.04+ / CentOS 8+ / RHEL 8+

### Software Requirements

- Go 1.21+
- PostgreSQL 13+
- Redis 6+
- Nginx 1.18+
- Docker (optional)
- Certbot (for SSL)

### External Services

- Gemini API key
- SMS provider (SMS.ir recommended)
- Payment gateway (Zarinpal)
- Monitoring service (Sentry)
- Telegram bot (for alerts)

## Step 1: Server Setup

### 1.1 Initial Server Configuration

```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install essential packages
sudo apt install -y curl wget git vim htop unzip

# Create application user
sudo useradd -m -s /bin/bash styler
sudo usermod -aG sudo styler

# Switch to application user
sudo su - styler
```

### 1.2 Install Go

```bash
# Download and install Go
wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz

# Add Go to PATH
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
echo 'export GOPATH=$HOME/go' >> ~/.bashrc
echo 'export PATH=$PATH:$GOPATH/bin' >> ~/.bashrc
source ~/.bashrc

# Verify installation
go version
```

### 1.3 Install PostgreSQL

```bash
# Install PostgreSQL
sudo apt install -y postgresql postgresql-contrib

# Start and enable PostgreSQL
sudo systemctl start postgresql
sudo systemctl enable postgresql

# Create database and user
sudo -u postgres psql << EOF
CREATE DATABASE styler;
CREATE USER styler_user WITH PASSWORD 'your_secure_password_here';
GRANT ALL PRIVILEGES ON DATABASE styler TO styler_user;
ALTER USER styler_user CREATEDB;
\q
EOF

# Configure PostgreSQL
sudo nano /etc/postgresql/13/main/postgresql.conf
# Uncomment and set:
# listen_addresses = 'localhost'
# max_connections = 200
# shared_buffers = 256MB
# effective_cache_size = 1GB

sudo nano /etc/postgresql/13/main/pg_hba.conf
# Add:
# local   styler          styler_user                            md5

# Restart PostgreSQL
sudo systemctl restart postgresql
```

### 1.4 Install Redis

```bash
# Install Redis
sudo apt install -y redis-server

# Configure Redis
sudo nano /etc/redis/redis.conf
# Set:
# maxmemory 256mb
# maxmemory-policy allkeys-lru
# requirepass your_redis_password_here

# Start and enable Redis
sudo systemctl start redis-server
sudo systemctl enable redis-server
```

### 1.5 Install Nginx

```bash
# Install Nginx
sudo apt install -y nginx

# Start and enable Nginx
sudo systemctl start nginx
sudo systemctl enable nginx

# Configure firewall
sudo ufw allow 22/tcp
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw enable
```

## Step 2: Application Deployment

### 2.1 Clone and Build Application

```bash
# Clone repository
git clone https://github.com/your-org/ai-styler.git
cd ai-styler

# Build application
go mod download
go build -o ai-styler main.go

# Create application directory
sudo mkdir -p /opt/ai-styler
sudo cp ai-styler /opt/ai-styler/
sudo chown -R styler:styler /opt/ai-styler
```

### 2.2 Create Environment Configuration

```bash
# Create environment file
sudo nano /opt/ai-styler/.env
```

Add the following configuration:

```env
# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=styler_user
DB_PASSWORD=your_secure_password_here
DB_NAME=styler
DB_SSLMODE=require

# Redis Configuration
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=your_redis_password_here
REDIS_DB=0

# Server Configuration
HTTP_ADDR=:8080
GIN_MODE=release

# JWT Configuration
JWT_SECRET=your-super-secret-jwt-key-change-in-production-minimum-32-chars
JWT_ACCESS_TTL=15m
JWT_REFRESH_TTL=720h

# Gemini AI Configuration
GEMINI_API_KEY=your-gemini-api-key-here
GEMINI_BASE_URL=https://generativelanguage.googleapis.com
GEMINI_MODEL=gemini-pro-vision
GEMINI_TIMEOUT=300
GEMINI_MAX_RETRIES=3

# Storage Configuration
STORAGE_PATH=/opt/ai-styler/uploads
UPLOAD_MAX_SIZE=10MB
SIGNED_URL_TTL=1h

# Security Configuration
BCRYPT_COST=12
ARGON2_MEMORY=65536
ARGON2_ITERATIONS=3
ARGON2_PARALLELISM=2

# Rate Limiting Configuration
RATE_LIMIT_OTP_PER_PHONE=3
RATE_LIMIT_OTP_PER_IP=100
RATE_LIMIT_LOGIN_PER_PHONE=5
RATE_LIMIT_LOGIN_PER_IP=10
RATE_LIMIT_WINDOW=1h

# SMS Configuration
SMS_PROVIDER=sms_ir
SMS_API_KEY=your-sms-api-key-here
SMS_TEMPLATE_ID=100000

# Payment Configuration
ZARINPAL_MERCHANT_ID=your-zarinpal-merchant-id
ZARINPAL_SANDBOX=false
ZARINPAL_CALLBACK_URL=https://yourdomain.com/api/v1/payments/callback

# Monitoring Configuration
ENVIRONMENT=production
LOG_LEVEL=info
HEALTH_ENABLED=true
VERSION=1.0.0

# Telegram Bot Configuration
TELEGRAM_BOT_TOKEN=your-telegram-bot-token
TELEGRAM_CHAT_ID=your-telegram-chat-id

# Sentry Configuration
SENTRY_DSN=your-sentry-dsn-here
```

### 2.3 Create Application Directories

```bash
# Create necessary directories
sudo mkdir -p /opt/ai-styler/uploads/{users,vendors,results,thumbnails}
sudo mkdir -p /opt/ai-styler/logs
sudo mkdir -p /opt/ai-styler/backups
sudo chown -R styler:styler /opt/ai-styler
```

### 2.4 Run Database Migrations

```bash
# Run migrations
cd /opt/ai-styler
./ai-styler migrate up
```

## Step 3: SSL/TLS Configuration

### 3.1 Install Certbot

```bash
# Install Certbot
sudo apt install -y certbot python3-certbot-nginx
```

### 3.2 Obtain SSL Certificate

```bash
# Get SSL certificate
sudo certbot --nginx -d yourdomain.com -d api.yourdomain.com

# Test auto-renewal
sudo certbot renew --dry-run
```

### 3.3 Configure Auto-Renewal

```bash
# Add to crontab
sudo crontab -e
# Add:
# 0 12 * * * /usr/bin/certbot renew --quiet
```

## Step 4: Nginx Configuration

### 4.1 Create Nginx Configuration

```bash
sudo nano /etc/nginx/sites-available/ai-styler
```

Add the following configuration:

```nginx
# Rate limiting
limit_req_zone $binary_remote_addr zone=api:10m rate=10r/s;
limit_req_zone $binary_remote_addr zone=auth:10m rate=5r/s;

# Upstream servers
upstream ai_styler_backend {
    server 127.0.0.1:8080;
    # Add more servers for load balancing
    # server 127.0.0.1:8081;
    # server 127.0.0.1:8082;
}

# Main server block
server {
    listen 80;
    server_name yourdomain.com api.yourdomain.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name yourdomain.com api.yourdomain.com;

    # SSL configuration
    ssl_certificate /etc/letsencrypt/live/yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/yourdomain.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-RSA-AES256-GCM-SHA512:DHE-RSA-AES256-GCM-SHA512:ECDHE-RSA-AES256-GCM-SHA384:DHE-RSA-AES256-GCM-SHA384;
    ssl_prefer_server_ciphers off;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;

    # Security headers
    add_header X-Frame-Options DENY;
    add_header X-Content-Type-Options nosniff;
    add_header X-XSS-Protection "1; mode=block";
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;

    # File upload size
    client_max_body_size 50M;
    client_body_timeout 60s;
    client_header_timeout 60s;

    # Gzip compression
    gzip on;
    gzip_vary on;
    gzip_min_length 1024;
    gzip_types text/plain text/css application/json application/javascript text/xml application/xml application/xml+rss text/javascript;

    # API routes
    location /api/ {
        limit_req zone=api burst=20 nodelay;
        
        proxy_pass http://ai_styler_backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # Timeouts
        proxy_connect_timeout 30s;
        proxy_send_timeout 30s;
        proxy_read_timeout 30s;
        
        # Buffer settings
        proxy_buffering on;
        proxy_buffer_size 4k;
        proxy_buffers 8 4k;
    }

    # Auth routes with stricter rate limiting
    location /api/auth/ {
        limit_req zone=auth burst=10 nodelay;
        
        proxy_pass http://ai_styler_backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # Health check endpoint
    location /api/health {
        proxy_pass http://ai_styler_backend;
        access_log off;
    }

    # Static files
    location /api/storage/ {
        alias /opt/ai-styler/uploads/;
        expires 1y;
        add_header Cache-Control "public, immutable";
    }

    # API documentation
    location /api/docs {
        alias /opt/ai-styler/api/;
        try_files $uri $uri/ /api/index.html;
    }
}
```

### 4.2 Enable Site and Test Configuration

```bash
# Enable site
sudo ln -s /etc/nginx/sites-available/ai-styler /etc/nginx/sites-enabled/

# Test configuration
sudo nginx -t

# Reload Nginx
sudo systemctl reload nginx
```

## Step 5: Systemd Service Configuration

### 5.1 Create Systemd Service

```bash
sudo nano /etc/systemd/system/ai-styler.service
```

Add the following configuration:

```ini
[Unit]
Description=AI Styler Backend Service
After=network.target postgresql.service redis.service
Wants=postgresql.service redis.service

[Service]
Type=simple
User=styler
Group=styler
WorkingDirectory=/opt/ai-styler
ExecStart=/opt/ai-styler/ai-styler
Restart=always
RestartSec=5
Environment=ENVIRONMENT=production

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/ai-styler/uploads /opt/ai-styler/logs

# Resource limits
LimitNOFILE=65536
LimitNPROC=4096

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=ai-styler

[Install]
WantedBy=multi-user.target
```

### 5.2 Enable and Start Service

```bash
# Reload systemd
sudo systemctl daemon-reload

# Enable service
sudo systemctl enable ai-styler

# Start service
sudo systemctl start ai-styler

# Check status
sudo systemctl status ai-styler

# View logs
sudo journalctl -u ai-styler -f
```

## Step 6: Monitoring and Logging

### 6.1 Configure Log Rotation

```bash
sudo nano /etc/logrotate.d/ai-styler
```

Add:

```
/opt/ai-styler/logs/*.log {
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

### 6.2 Setup Monitoring Scripts

```bash
# Create monitoring script
sudo nano /opt/ai-styler/monitor.sh
```

Add:

```bash
#!/bin/bash

# Health check script
check_service() {
    local service_name=$1
    local port=$2
    
    if ! systemctl is-active --quiet $service_name; then
        echo "ERROR: $service_name is not running"
        return 1
    fi
    
    if ! curl -f http://localhost:$port/api/health > /dev/null 2>&1; then
        echo "ERROR: $service_name health check failed"
        return 1
    fi
    
    echo "OK: $service_name is healthy"
    return 0
}

# Check all services
check_service ai-styler 8080
check_service postgresql 5432
check_service redis-server 6379
check_service nginx 80

# Check disk space
DISK_USAGE=$(df /opt/ai-styler | tail -1 | awk '{print $5}' | sed 's/%//')
if [ $DISK_USAGE -gt 80 ]; then
    echo "WARNING: Disk usage is ${DISK_USAGE}%"
fi

# Check memory usage
MEM_USAGE=$(free | grep Mem | awk '{printf "%.0f", $3/$2 * 100.0}')
if [ $MEM_USAGE -gt 80 ]; then
    echo "WARNING: Memory usage is ${MEM_USAGE}%"
fi
```

```bash
# Make executable
sudo chmod +x /opt/ai-styler/monitor.sh

# Add to crontab
sudo crontab -e
# Add:
# */5 * * * * /opt/ai-styler/monitor.sh
```

## Step 7: Backup Configuration

### 7.1 Database Backup Script

```bash
sudo nano /opt/ai-styler/backup_db.sh
```

Add:

```bash
#!/bin/bash

BACKUP_DIR="/opt/ai-styler/backups"
DATE=$(date +%Y%m%d_%H%M%S)
DB_NAME="styler"
DB_USER="styler_user"

# Create backup directory
mkdir -p $BACKUP_DIR

# Database backup
pg_dump -h localhost -U $DB_USER -d $DB_NAME > $BACKUP_DIR/db_backup_$DATE.sql

# Compress backup
gzip $BACKUP_DIR/db_backup_$DATE.sql

# Keep only last 7 days
find $BACKUP_DIR -name "db_backup_*.sql.gz" -mtime +7 -delete

echo "Database backup completed: db_backup_$DATE.sql.gz"
```

### 7.2 File Backup Script

```bash
sudo nano /opt/ai-styler/backup_files.sh
```

Add:

```bash
#!/bin/bash

BACKUP_DIR="/opt/ai-styler/backups"
UPLOADS_DIR="/opt/ai-styler/uploads"
DATE=$(date +%Y%m%d_%H%M%S)

# Create backup directory
mkdir -p $BACKUP_DIR

# File backup
tar -czf $BACKUP_DIR/uploads_backup_$DATE.tar.gz -C $UPLOADS_DIR .

# Keep only last 7 days
find $BACKUP_DIR -name "uploads_backup_*.tar.gz" -mtime +7 -delete

echo "File backup completed: uploads_backup_$DATE.tar.gz"
```

### 7.3 Schedule Backups

```bash
# Make scripts executable
sudo chmod +x /opt/ai-styler/backup_*.sh

# Schedule backups
sudo crontab -e
# Add:
# 0 2 * * * /opt/ai-styler/backup_db.sh
# 0 3 * * * /opt/ai-styler/backup_files.sh
```

## Step 8: Security Hardening

### 8.1 Firewall Configuration

```bash
# Configure UFW
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow ssh
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw enable
```

### 8.2 Fail2Ban Configuration

```bash
# Install Fail2Ban
sudo apt install -y fail2ban

# Configure Fail2Ban
sudo nano /etc/fail2ban/jail.local
```

Add:

```ini
[DEFAULT]
bantime = 3600
findtime = 600
maxretry = 3

[sshd]
enabled = true
port = ssh
logpath = /var/log/auth.log

[nginx-http-auth]
enabled = true
filter = nginx-http-auth
port = http,https
logpath = /var/log/nginx/error.log

[nginx-limit-req]
enabled = true
filter = nginx-limit-req
port = http,https
logpath = /var/log/nginx/error.log
maxretry = 10
```

```bash
# Start Fail2Ban
sudo systemctl start fail2ban
sudo systemctl enable fail2ban
```

### 8.3 Application Security

```bash
# Set proper file permissions
sudo chmod 600 /opt/ai-styler/.env
sudo chmod 755 /opt/ai-styler/ai-styler
sudo chmod -R 755 /opt/ai-styler/uploads
```

## Step 9: Performance Optimization

### 9.1 PostgreSQL Optimization

```bash
sudo nano /etc/postgresql/13/main/postgresql.conf
```

Optimize settings:

```conf
# Memory settings
shared_buffers = 256MB
effective_cache_size = 1GB
work_mem = 4MB
maintenance_work_mem = 64MB

# Connection settings
max_connections = 200
listen_addresses = 'localhost'

# Checkpoint settings
checkpoint_completion_target = 0.9
wal_buffers = 16MB

# Query optimization
random_page_cost = 1.1
effective_io_concurrency = 200
```

### 9.2 Redis Optimization

```bash
sudo nano /etc/redis/redis.conf
```

Optimize settings:

```conf
# Memory management
maxmemory 256mb
maxmemory-policy allkeys-lru

# Persistence
save 900 1
save 300 10
save 60 10000

# Network
tcp-keepalive 60
timeout 300
```

## Step 10: Load Balancing (Optional)

### 10.1 Multiple Application Instances

```bash
# Create additional instances
sudo cp /opt/ai-styler/ai-styler /opt/ai-styler/ai-styler-8081
sudo cp /opt/ai-styler/ai-styler /opt/ai-styler/ai-styler-8082

# Create additional systemd services
sudo cp /etc/systemd/system/ai-styler.service /etc/systemd/system/ai-styler-8081.service
sudo cp /etc/systemd/system/ai-styler.service /etc/systemd/system/ai-styler-8082.service

# Modify ports in additional services
sudo sed -i 's/:8080/:8081/g' /etc/systemd/system/ai-styler-8081.service
sudo sed -i 's/:8080/:8082/g' /etc/systemd/system/ai-styler-8082.service

# Start additional instances
sudo systemctl enable ai-styler-8081 ai-styler-8082
sudo systemctl start ai-styler-8081 ai-styler-8082
```

### 10.2 Update Nginx Configuration

Update the upstream block in `/etc/nginx/sites-available/ai-styler`:

```nginx
upstream ai_styler_backend {
    server 127.0.0.1:8080;
    server 127.0.0.1:8081;
    server 127.0.0.1:8082;
}
```

## Step 11: Testing and Validation

### 11.1 Health Checks

```bash
# Test API endpoints
curl -f https://yourdomain.com/api/health
curl -f https://yourdomain.com/api/auth/send-otp -X POST -H "Content-Type: application/json" -d '{"phone_number":"+1234567890"}'
```

### 11.2 Load Testing

```bash
# Install Apache Bench
sudo apt install -y apache2-utils

# Run load test
ab -n 1000 -c 10 https://yourdomain.com/api/health
```

### 11.3 SSL Testing

```bash
# Test SSL configuration
curl -I https://yourdomain.com
openssl s_client -connect yourdomain.com:443 -servername yourdomain.com
```

## Step 12: Maintenance and Updates

### 12.1 Update Procedure

```bash
# Create update script
sudo nano /opt/ai-styler/update.sh
```

Add:

```bash
#!/bin/bash

set -e

BACKUP_DIR="/opt/ai-styler/backups"
DATE=$(date +%Y%m%d_%H%M%S)

echo "Starting update process..."

# Backup current version
cp /opt/ai-styler/ai-styler $BACKUP_DIR/ai-styler_backup_$DATE

# Stop service
sudo systemctl stop ai-styler

# Update application
cd /opt/ai-styler
git pull origin main
go build -o ai-styler main.go

# Run migrations
./ai-styler migrate up

# Start service
sudo systemctl start ai-styler

# Check status
sudo systemctl status ai-styler

echo "Update completed successfully"
```

### 12.2 Monitoring Commands

```bash
# Check service status
sudo systemctl status ai-styler

# View logs
sudo journalctl -u ai-styler -f

# Check resource usage
htop
df -h
free -h

# Check database connections
sudo -u postgres psql -c "SELECT count(*) FROM pg_stat_activity;"

# Check Redis memory usage
redis-cli info memory
```

## Troubleshooting

### Common Issues

1. **Service won't start**
   - Check logs: `sudo journalctl -u ai-styler -f`
   - Verify configuration: `sudo systemctl status ai-styler`
   - Check file permissions

2. **Database connection issues**
   - Verify PostgreSQL is running: `sudo systemctl status postgresql`
   - Check connection: `psql -h localhost -U styler_user -d styler`
   - Verify credentials in `.env`

3. **Redis connection issues**
   - Verify Redis is running: `sudo systemctl status redis-server`
   - Check connection: `redis-cli ping`
   - Verify password in `.env`

4. **Nginx issues**
   - Test configuration: `sudo nginx -t`
   - Check logs: `sudo tail -f /var/log/nginx/error.log`
   - Verify upstream servers

5. **SSL issues**
   - Check certificate: `sudo certbot certificates`
   - Test SSL: `openssl s_client -connect yourdomain.com:443`
   - Verify Nginx SSL configuration

### Performance Issues

1. **High CPU usage**
   - Check for infinite loops in logs
   - Monitor database queries
   - Verify worker processes

2. **High memory usage**
   - Check for memory leaks
   - Monitor Redis memory usage
   - Verify connection pooling

3. **Slow response times**
   - Check database performance
   - Monitor network latency
   - Verify caching effectiveness

## Security Checklist

- [ ] SSL/TLS certificates installed and auto-renewing
- [ ] Firewall configured and enabled
- [ ] Fail2Ban installed and configured
- [ ] Application running as non-root user
- [ ] Database credentials secured
- [ ] API keys stored securely
- [ ] Rate limiting enabled
- [ ] Security headers configured
- [ ] File permissions set correctly
- [ ] Regular security updates applied

## Monitoring Checklist

- [ ] Health checks configured
- [ ] Log rotation set up
- [ ] Backup procedures tested
- [ ] Monitoring scripts running
- [ ] Alert notifications working
- [ ] Performance metrics collected
- [ ] Error tracking configured

## Conclusion

This deployment guide provides a comprehensive setup for the AI Styler backend service. The system is designed to be:

- **Scalable**: Can handle multiple instances and load balancing
- **Secure**: Implements security best practices
- **Maintainable**: Includes monitoring, logging, and backup procedures
- **Reliable**: Includes health checks and failover mechanisms

For additional support or questions, refer to the project documentation or contact the development team.
