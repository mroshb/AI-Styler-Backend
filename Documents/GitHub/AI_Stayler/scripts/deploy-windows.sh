#!/bin/bash

# AI Styler Deployment Script for Windows Server
# This script automates the deployment process on Windows Server with IIS

set -e

# Configuration
APP_NAME="AI Styler"
APP_DIR="C:\inetpub\wwwroot\ai-styler"
SERVICE_NAME="AI Styler Service"
BACKUP_DIR="C:\backups\ai-styler"
LOG_FILE="C:\logs\ai-styler-deployment.log"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging function
log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1" | tee -a "$LOG_FILE"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1" | tee -a "$LOG_FILE"
    exit 1
}

warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1" | tee -a "$LOG_FILE"
}

# Check if running as administrator
check_admin() {
    if ! net session >nul 2>&1; then
        error "This script must be run as Administrator"
    fi
}

# Create necessary directories
create_directories() {
    log "Creating necessary directories..."
    
    mkdir -p "$APP_DIR" 2>/dev/null || true
    mkdir -p "$BACKUP_DIR" 2>/dev/null || true
    mkdir -p "C:\logs" 2>/dev/null || true
    mkdir -p "$APP_DIR\uploads" 2>/dev/null || true
    mkdir -p "$APP_DIR\logs" 2>/dev/null || true
}

# Backup current version
backup_current() {
    if [ -d "$APP_DIR" ]; then
        log "Backing up current version..."
        timestamp=$(date +%Y%m%d_%H%M%S)
        backup_path="$BACKUP_DIR\backup_$timestamp"
        
        robocopy "$APP_DIR" "$backup_path" /MIR /R:3 /W:10 /NP /NFL /NDL
        log "Backup completed: $backup_path"
    fi
}

# Stop services
stop_services() {
    log "Stopping services..."
    
    # Stop Windows Service
    sc query "$SERVICE_NAME" >nul 2>&1 && sc stop "$SERVICE_NAME" || true
    
    # Stop IIS Application Pool
    appcmd stop apppool "AI_Styler_Pool" 2>/dev/null || true
    
    # Wait for services to stop
    sleep 5
}

# Deploy new version
deploy_files() {
    log "Deploying new version..."
    
    # Copy new files (assuming they're in current directory)
    if [ -f "ai-styler.exe" ]; then
        copy "ai-styler.exe" "$APP_DIR\ai-styler.exe"
        log "Copied ai-styler.exe"
    else
        error "ai-styler.exe not found in current directory"
    fi
    
    if [ -f ".env" ]; then
        copy ".env" "$APP_DIR\.env"
        log "Copied .env file"
    fi
    
    if [ -f "web.config" ]; then
        copy "web.config" "$APP_DIR\web.config"
        log "Copied web.config"
    fi
}

# Set permissions
set_permissions() {
    log "Setting permissions..."
    
    # Grant IIS_IUSRS full control
    icacls "$APP_DIR" /grant "IIS_IUSRS:(OI)(CI)F" /T
    
    # Grant application pool identity permissions
    icacls "$APP_DIR" /grant "IIS AppPool\AI_Styler_Pool:(OI)(CI)F" /T
    
    log "Permissions set successfully"
}

# Run database migrations
run_migrations() {
    log "Running database migrations..."
    
    cd "$APP_DIR"
    .\ai-styler.exe migrate up
    
    if [ $? -eq 0 ]; then
        log "Database migrations completed successfully"
    else
        error "Database migrations failed"
    fi
}

# Start services
start_services() {
    log "Starting services..."
    
    # Start Windows Service
    sc start "$SERVICE_NAME"
    
    # Start IIS Application Pool
    appcmd start apppool "AI_Styler_Pool"
    
    # Wait for services to start
    sleep 10
}

# Health check
health_check() {
    log "Performing health check..."
    
    # Wait for service to be ready
    sleep 30
    
    # Check if service is running
    if sc query "$SERVICE_NAME" | grep -q "RUNNING"; then
        log "Windows Service is running"
    else
        error "Windows Service failed to start"
    fi
    
    # Check health endpoint
    response=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/health/ || echo "000")
    
    if [ "$response" = "200" ]; then
        log "Health check passed - Application is responding"
    else
        warning "Health check failed - HTTP Status: $response"
        warning "Please check application logs for issues"
    fi
}

# Cleanup old backups
cleanup_backups() {
    log "Cleaning up old backups..."
    
    # Keep only last 5 backups
    cd "$BACKUP_DIR"
    ls -t | tail -n +6 | xargs -r rm -rf
}

# Main deployment function
main() {
    log "Starting AI Styler deployment..."
    
    check_admin
    create_directories
    backup_current
    stop_services
    deploy_files
    set_permissions
    run_migrations
    start_services
    health_check
    cleanup_backups
    
    log "Deployment completed successfully!"
    log "Application is available at: http://localhost:8080"
    log "Health check: http://localhost:8080/health/"
}

# Run main function
main "$@"
