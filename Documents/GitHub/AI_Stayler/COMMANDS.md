# AI Styler Docker Development Commands

## Quick Start

### Start Development Environment
```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f ai-styler

# Stop services
docker-compose down
```

### Database Management
```bash
# Run migrations
docker-compose exec ai-styler go run scripts/migrate.go up

# Seed database
docker-compose exec ai-styler go run scripts/seed.go seed

# Check migration status
docker-compose exec ai-styler go run scripts/migrate.go status
```

### Testing
```bash
# Start test environment
docker-compose -f docker-compose.test.yml up -d

# Run tests
docker-compose exec ai-styler-test go test -v -tags=integration ./...

# Stop test environment
docker-compose -f docker-compose.test.yml down
```

## Development Commands

### Build and Run Locally
```bash
# Build the application
go build -o ai-styler .

# Run with development config
./ai-styler

# Run with custom config
ENVIRONMENT=development LOG_LEVEL=debug ./ai-styler
```

### Database Operations
```bash
# Connect to database
docker-compose exec postgres psql -U styler_user -d styler

# Backup database
docker-compose exec postgres pg_dump -U styler_user styler > backup.sql

# Restore database
docker-compose exec -T postgres psql -U styler_user styler < backup.sql
```

### Redis Operations
```bash
# Connect to Redis
docker-compose exec redis redis-cli

# Monitor Redis
docker-compose exec redis redis-cli monitor
```

## Health Checks

### Application Health
```bash
# Basic health check
curl http://localhost:8080/health/

# Detailed health status
curl http://localhost:8080/health/metrics

# System information
curl http://localhost:8080/health/system
```

### Service Health
```bash
# Check all services
docker-compose ps

# Check service logs
docker-compose logs postgres
docker-compose logs redis
docker-compose logs ai-styler
```

## Troubleshooting

### Common Issues

1. **Port conflicts:**
   ```bash
   # Check what's using port 8080
   netstat -tulpn | grep :8080
   
   # Use different port
   HTTP_ADDR=:8081 docker-compose up
   ```

2. **Database connection issues:**
   ```bash
   # Check database logs
   docker-compose logs postgres
   
   # Test connection
   docker-compose exec ai-styler go run -c "SELECT 1"
   ```

3. **Permission issues:**
   ```bash
   # Fix upload directory permissions
   sudo chown -R 1001:1001 uploads/
   ```

### Debug Mode
```bash
# Run with debug logging
LOG_LEVEL=debug docker-compose up

# Run single service for debugging
docker-compose run --rm ai-styler bash
```

## Production Deployment

### Build Production Image
```bash
# Build production image
docker build -t ai-styler:latest .

# Run production container
docker run -d \
  --name ai-styler-prod \
  -p 8080:8080 \
  -e ENVIRONMENT=production \
  -e LOG_LEVEL=warn \
  ai-styler:latest
```

### Windows Server Deployment
```powershell
# Run deployment script
.\scripts\deploy-windows.ps1

# Install Windows Service
.\deployment\install-service.ps1

# Check service status
Get-Service "AI Styler Service"
```

## Monitoring

### Application Metrics
```bash
# Health metrics
curl http://localhost:8080/health/metrics | jq

# System metrics
curl http://localhost:8080/health/system | jq
```

### Log Monitoring
```bash
# Follow application logs
docker-compose logs -f ai-styler

# Search logs
docker-compose logs ai-styler | grep ERROR

# Export logs
docker-compose logs ai-styler > app.log
```

## Backup and Recovery

### Database Backup
```bash
# Create backup
docker-compose exec postgres pg_dump -U styler_user styler > backup_$(date +%Y%m%d).sql

# Restore from backup
docker-compose exec -T postgres psql -U styler_user styler < backup_20240115.sql
```

### Application Backup
```bash
# Backup uploads
tar -czf uploads_backup_$(date +%Y%m%d).tar.gz uploads/

# Backup configuration
cp .env .env.backup
cp docker-compose.yml docker-compose.yml.backup
```

This comprehensive command reference covers all the essential operations for developing, testing, and deploying the AI Styler API.
