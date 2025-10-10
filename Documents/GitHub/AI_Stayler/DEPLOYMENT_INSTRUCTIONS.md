# 🚀 AI Styler Docker Deployment Instructions

## Quick Start Commands

### 1. Environment Setup
```bash
# Copy environment template
cp env.example .env

# Edit with your production values
nano .env
```

### 2. SSL Certificates (Required for Production)
```bash
# Create SSL directory
mkdir -p ssl

# Let's Encrypt (Recommended)
certbot certonly --standalone -d api.yourdomain.com
cp /etc/letsencrypt/live/api.yourdomain.com/fullchain.pem ssl/cert.pem
cp /etc/letsencrypt/live/api.yourdomain.com/privkey.pem ssl/key.pem
```

### 3. Create Required Directories
```bash
mkdir -p uploads logs backups ssl
chmod 755 uploads logs backups
chmod 600 ssl/*
```

### 4. Deploy Production Environment
```bash
# Start all services
docker-compose -f docker-compose.prod.yml up -d

# Check status
docker-compose -f docker-compose.prod.yml ps
```

## 🔍 Verification Steps

### 1. Service Health Checks
```bash
# Application health
curl -k https://localhost/api/health

# Database connectivity
docker-compose -f docker-compose.prod.yml exec postgres pg_isready -U styler_user -d styler

# Redis connectivity
docker-compose -f docker-compose.prod.yml exec redis redis-cli ping

# All services status
docker-compose -f docker-compose.prod.yml ps
```

### 2. Monitoring Access
```bash
# Grafana Dashboard
open https://yourdomain.com:3000
# Login: admin / your-grafana-password

# Prometheus Metrics
open https://yourdomain.com:9090

# Application Metrics
curl https://yourdomain.com/api/metrics
```

### 3. API Endpoints Test
```bash
# Health endpoint
curl https://api.yourdomain.com/api/health

# Authentication test
curl -X POST https://api.yourdomain.com/api/auth/send-otp \
  -H "Content-Type: application/json" \
  -d '{"phone": "+1234567890"}'

# Metrics endpoint
curl https://api.yourdomain.com/api/metrics
```

## 🛠️ Essential Commands

### Logs and Debugging
```bash
# Application logs
docker-compose -f docker-compose.prod.yml logs -f app

# Database logs
docker-compose -f docker-compose.prod.yml logs postgres

# All services logs
docker-compose -f docker-compose.prod.yml logs

# Check container resource usage
docker stats
```

### Database Operations
```bash
# Backup database
docker-compose -f docker-compose.prod.yml exec postgres \
  pg_dump -U styler_user styler > backup_$(date +%Y%m%d).sql

# Run migrations
docker-compose -f docker-compose.prod.yml exec app \
  go run scripts/migrate.go up
```

### Updates and Maintenance
```bash
# Update application
git pull origin main
docker-compose -f docker-compose.prod.yml up -d --build app

# Restart specific service
docker-compose -f docker-compose.prod.yml restart app

# Scale application
docker-compose -f docker-compose.prod.yml up -d --scale app=3
```

## 🚨 Troubleshooting

### Common Issues
1. **Port conflicts**: Check if ports 80, 443, 8080 are available
2. **SSL errors**: Verify certificate files exist and are valid
3. **Database connection**: Check DB_PASSWORD in .env file
4. **Memory issues**: Ensure at least 4GB RAM available

### Emergency Commands
```bash
# Stop all services
docker-compose -f docker-compose.prod.yml down

# Emergency restart
docker-compose -f docker-compose.prod.yml restart

# Check service health
docker-compose -f docker-compose.prod.yml ps
```

## 📊 Monitoring URLs

- **Grafana**: https://yourdomain.com:3000
- **Prometheus**: https://yourdomain.com:9090  
- **Application**: https://api.yourdomain.com
- **Health Check**: https://api.yourdomain.com/api/health
- **Metrics**: https://api.yourdomain.com/api/metrics

## ✅ Success Criteria

Your deployment is successful when:
- [ ] All containers are running (`docker-compose ps` shows "Up")
- [ ] Health endpoint returns 200 OK
- [ ] Database and Redis connections work
- [ ] SSL certificates are valid
- [ ] Grafana dashboard is accessible
- [ ] Application metrics are being collected
- [ ] No error logs in application container

---

**🎉 Your AI Styler backend is now running in production!**
