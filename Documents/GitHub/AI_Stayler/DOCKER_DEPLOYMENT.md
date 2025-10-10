# 🚀 AI Styler Docker Deployment Guide

This guide provides step-by-step instructions for deploying the AI Styler backend using Docker Compose in production mode with full monitoring stack.

## 📋 Prerequisites

- **Docker** 20.10+ and **Docker Compose** 2.0+
- **Minimum 4GB RAM** and **2 CPU cores**
- **SSL certificates** for HTTPS (Let's Encrypt recommended)
- **Domain name** pointing to your server

## 🔧 Quick Setup

### 1. Clone and Prepare

```bash
# Clone the repository
git clone <your-repo-url>
cd AI_Stayler

# Copy environment template
cp env.example .env
```

### 2. Configure Environment Variables

Edit `.env` file with your production values:

```bash
# Required: Database
DB_PASSWORD=your-secure-database-password

# Required: Redis
REDIS_PASSWORD=your-secure-redis-password

# Required: JWT Secret (generate with: openssl rand -base64 32)
JWT_SECRET=your-super-secret-jwt-key-minimum-32-characters

# Required: Gemini AI API Key
GEMINI_API_KEY=your-gemini-api-key

# Required: SMS API Key
SMS_API_KEY=your-sms-api-key

# Required: Payment Merchant ID
ZARINPAL_MERCHANT_ID=your-zarinpal-merchant-id

# Optional: Monitoring
TELEGRAM_BOT_TOKEN=your-telegram-bot-token
TELEGRAM_CHAT_ID=your-telegram-chat-id
SENTRY_DSN=your-sentry-dsn
GRAFANA_PASSWORD=your-grafana-password
```

### 3. SSL Certificates Setup

```bash
# Create SSL directory
mkdir -p ssl

# Option A: Let's Encrypt (Recommended)
certbot certonly --standalone -d api.yourdomain.com
cp /etc/letsencrypt/live/api.yourdomain.com/fullchain.pem ssl/cert.pem
cp /etc/letsencrypt/live/api.yourdomain.com/privkey.pem ssl/key.pem

# Option B: Self-signed (Development only)
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout ssl/key.pem -out ssl/cert.pem \
  -subj "/C=US/ST=State/L=City/O=Organization/CN=api.yourdomain.com"
```

### 4. Create Required Directories

```bash
# Create directories for persistent data
mkdir -p uploads logs backups ssl
chmod 755 uploads logs backups
chmod 600 ssl/*
```

## 🚀 Production Deployment

### Start All Services

```bash
# Start production environment
docker-compose -f docker-compose.prod.yml up -d

# Check service status
docker-compose -f docker-compose.prod.yml ps
```

### Verify Deployment

```bash
# Check application health
curl -k https://localhost/api/health

# Check database connection
docker-compose -f docker-compose.prod.yml exec postgres pg_isready -U styler_user -d styler

# Check Redis connection
docker-compose -f docker-compose.prod.yml exec redis redis-cli ping
```

## 📊 Monitoring Setup

### Access Monitoring Dashboards

- **Grafana**: https://yourdomain.com:3000 (admin / your-grafana-password)
- **Prometheus**: https://yourdomain.com:9090
- **Application Metrics**: https://yourdomain.com/api/metrics

### Key Metrics to Monitor

1. **Application Health**
   - Request rate: `rate(http_requests_total[5m])`
   - Response time: `histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))`
   - Error rate: `rate(http_requests_total{status=~"5.."}[5m])`

2. **Database Performance**
   - Connection pool usage
   - Query performance
   - Lock contention

3. **System Resources**
   - CPU usage
   - Memory consumption
   - Disk I/O

## 🔍 Health Checks

### Application Health Endpoints

```bash
# Basic health check
curl https://api.yourdomain.com/api/health

# Detailed metrics
curl https://api.yourdomain.com/api/metrics

# System information
curl https://api.yourdomain.com/api/health/system
```

### Service Health Commands

```bash
# Check all containers
docker-compose -f docker-compose.prod.yml ps

# View application logs
docker-compose -f docker-compose.prod.yml logs -f app

# Check database logs
docker-compose -f docker-compose.prod.yml logs postgres

# Check Redis logs
docker-compose -f docker-compose.prod.yml logs redis
```

## 🛠️ Maintenance Operations

### Database Operations

```bash
# Backup database
docker-compose -f docker-compose.prod.yml exec postgres \
  pg_dump -U styler_user styler > backup_$(date +%Y%m%d).sql

# Restore database
docker-compose -f docker-compose.prod.yml exec -T postgres \
  psql -U styler_user styler < backup_20240115.sql

# Run migrations
docker-compose -f docker-compose.prod.yml exec app \
  go run scripts/migrate.go up
```

### Application Updates

```bash
# Pull latest changes
git pull origin main

# Rebuild and restart application
docker-compose -f docker-compose.prod.yml up -d --build app

# Check deployment status
docker-compose -f docker-compose.prod.yml ps app
```

### Log Management

```bash
# View recent logs
docker-compose -f docker-compose.prod.yml logs --tail=100 app

# Follow logs in real-time
docker-compose -f docker-compose.prod.yml logs -f app

# Export logs
docker-compose -f docker-compose.prod.yml logs app > app.log
```

## 🔧 Troubleshooting

### Common Issues

#### 1. Application Won't Start

```bash
# Check application logs
docker-compose -f docker-compose.prod.yml logs app

# Common causes:
# - Missing environment variables
# - Database connection issues
# - Port conflicts
```

#### 2. Database Connection Issues

```bash
# Test database connectivity
docker-compose -f docker-compose.prod.yml exec app \
  go run -c "SELECT 1"

# Check database logs
docker-compose -f docker-compose.prod.yml logs postgres

# Verify database is running
docker-compose -f docker-compose.prod.yml exec postgres \
  pg_isready -U styler_user -d styler
```

#### 3. SSL Certificate Issues

```bash
# Check certificate validity
openssl x509 -in ssl/cert.pem -text -noout

# Test SSL connection
openssl s_client -connect api.yourdomain.com:443

# Renew Let's Encrypt certificate
certbot renew --dry-run
```

#### 4. High Memory Usage

```bash
# Check container resource usage
docker stats

# Restart services if needed
docker-compose -f docker-compose.prod.yml restart app

# Check for memory leaks in logs
docker-compose -f docker-compose.prod.yml logs app | grep -i memory
```

### Performance Optimization

#### 1. Database Tuning

```bash
# Connect to database
docker-compose -f docker-compose.prod.yml exec postgres \
  psql -U styler_user -d styler

# Check slow queries
SELECT query, mean_time, calls 
FROM pg_stat_statements 
ORDER BY mean_time DESC 
LIMIT 10;
```

#### 2. Redis Optimization

```bash
# Connect to Redis
docker-compose -f docker-compose.prod.yml exec redis redis-cli

# Check memory usage
INFO memory

# Monitor commands
MONITOR
```

#### 3. Application Scaling

```bash
# Scale application instances
docker-compose -f docker-compose.prod.yml up -d --scale app=3

# Check load distribution
curl https://api.yourdomain.com/api/metrics | grep http_requests_total
```

## 🔒 Security Checklist

- [ ] **Environment Variables**: All secrets properly configured
- [ ] **SSL Certificates**: Valid HTTPS certificates installed
- [ ] **Firewall**: Only necessary ports exposed (80, 443, 22)
- [ ] **Database**: Strong passwords and SSL enabled
- [ ] **Redis**: Password protected and network restricted
- [ ] **Monitoring**: Access restricted to internal network
- [ ] **Logs**: Sensitive data not logged
- [ ] **Updates**: Regular security updates applied

## 📈 Scaling Considerations

### Horizontal Scaling

```bash
# Scale application instances
docker-compose -f docker-compose.prod.yml up -d --scale app=5

# Use external load balancer (HAProxy, Nginx)
# Configure sticky sessions if needed
```

### Database Scaling

```bash
# Set up read replicas
# Configure connection pooling
# Implement database sharding for large datasets
```

### Monitoring Scaling

```bash
# Use external Prometheus/Grafana
# Implement log aggregation (ELK stack)
# Set up alerting rules
```

## 🆘 Support and Maintenance

### Regular Maintenance Tasks

1. **Daily**
   - Check application health
   - Monitor error rates
   - Review logs for issues

2. **Weekly**
   - Update dependencies
   - Backup database
   - Review performance metrics

3. **Monthly**
   - Security updates
   - Certificate renewal
   - Capacity planning

### Emergency Procedures

```bash
# Quick rollback
git checkout previous-stable-tag
docker-compose -f docker-compose.prod.yml up -d --build app

# Emergency stop
docker-compose -f docker-compose.prod.yml down

# Emergency restart
docker-compose -f docker-compose.prod.yml restart app
```

## 📞 Getting Help

- **Documentation**: Check README.md and COMMANDS.md
- **Logs**: Always check logs first for error details
- **Health Checks**: Use provided health endpoints
- **Monitoring**: Check Grafana dashboards for system status

---

**🎉 Congratulations!** Your AI Styler backend is now running in production with full monitoring and security features.
