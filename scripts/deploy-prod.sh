#!/bin/bash

# AI Styler Production Deployment Script
# This script deploys the AI Styler backend to production

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
PROJECT_NAME="ai-styler"
COMPOSE_FILE="docker-compose.prod.yml"
ENV_FILE=".env.prod"

# Functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if running as root
check_root() {
    if [[ $EUID -eq 0 ]]; then
        log_error "This script should not be run as root"
        exit 1
    fi
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check if Docker is installed
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed"
        exit 1
    fi
    
    # Check if Docker Compose is installed
    if ! command -v docker-compose &> /dev/null; then
        log_error "Docker Compose is not installed"
        exit 1
    fi
    
    # Check if environment file exists
    if [[ ! -f "$ENV_FILE" ]]; then
        log_error "Environment file $ENV_FILE not found"
        log_info "Please copy .env.prod.example to $ENV_FILE and configure it"
        exit 1
    fi
    
    log_success "Prerequisites check passed"
}

# Validate environment configuration
validate_config() {
    log_info "Validating environment configuration..."
    
    # Source environment file
    source "$ENV_FILE"
    
    # Check required variables
    required_vars=(
        "DB_PASSWORD"
        "REDIS_PASSWORD"
        "JWT_SECRET"
        "GEMINI_API_KEY"
        "SMS_API_KEY"
        "ZARINPAL_MERCHANT_ID"
        "GRAFANA_PASSWORD"
    )
    
    for var in "${required_vars[@]}"; do
        if [[ -z "${!var}" ]]; then
            log_error "Required environment variable $var is not set"
            exit 1
        fi
    done
    
    # Validate JWT secret length
    if [[ ${#JWT_SECRET} -lt 32 ]]; then
        log_error "JWT_SECRET must be at least 32 characters long"
        exit 1
    fi
    
    log_success "Environment configuration validation passed"
}

# Create necessary directories
create_directories() {
    log_info "Creating necessary directories..."
    
    directories=(
        "uploads"
        "logs"
        "backups"
        "ssl"
        "monitoring/grafana/dashboards"
        "monitoring/grafana/datasources"
    )
    
    for dir in "${directories[@]}"; do
        mkdir -p "$dir"
        log_info "Created directory: $dir"
    done
    
    log_success "Directories created successfully"
}

# Generate SSL certificates (self-signed for development)
generate_ssl_certificates() {
    log_info "Generating SSL certificates..."
    
    if [[ ! -f "ssl/cert.pem" ]] || [[ ! -f "ssl/key.pem" ]]; then
        openssl req -x509 -newkey rsa:4096 -keyout ssl/key.pem -out ssl/cert.pem -days 365 -nodes \
            -subj "/C=US/ST=State/L=City/O=Organization/CN=api.aistyler.com"
        log_success "SSL certificates generated"
    else
        log_info "SSL certificates already exist"
    fi
}

# Build and start services
deploy_services() {
    log_info "Deploying services..."
    
    # Pull latest images
    docker-compose -f "$COMPOSE_FILE" pull
    
    # Build application
    docker-compose -f "$COMPOSE_FILE" build --no-cache app
    
    # Start services
    docker-compose -f "$COMPOSE_FILE" up -d
    
    log_success "Services deployed successfully"
}

# Wait for services to be healthy
wait_for_services() {
    log_info "Waiting for services to be healthy..."
    
    services=("postgres" "redis" "app")
    
    for service in "${services[@]}"; do
        log_info "Waiting for $service to be healthy..."
        timeout=300
        while [[ $timeout -gt 0 ]]; do
            if docker-compose -f "$COMPOSE_FILE" ps "$service" | grep -q "healthy"; then
                log_success "$service is healthy"
                break
            fi
            sleep 5
            timeout=$((timeout - 5))
        done
        
        if [[ $timeout -le 0 ]]; then
            log_error "$service failed to become healthy"
            exit 1
        fi
    done
}

# Run database migrations
run_migrations() {
    log_info "Running database migrations..."
    
    # Wait for database to be ready
    sleep 10
    
    # Run migrations
    docker-compose -f "$COMPOSE_FILE" exec -T postgres psql -U styler_user -d styler -f /docker-entrypoint-initdb.d/0009_comprehensive_schema.sql
    
    log_success "Database migrations completed"
}

# Setup monitoring
setup_monitoring() {
    log_info "Setting up monitoring..."
    
    # Wait for monitoring services to be ready
    sleep 30
    
    # Check if Prometheus is accessible
    if curl -f http://localhost:9090/api/v1/status/config &> /dev/null; then
        log_success "Prometheus is accessible"
    else
        log_warning "Prometheus is not accessible"
    fi
    
    # Check if Grafana is accessible
    if curl -f http://localhost:3000/api/health &> /dev/null; then
        log_success "Grafana is accessible"
    else
        log_warning "Grafana is not accessible"
    fi
    
    log_success "Monitoring setup completed"
}

# Run health checks
run_health_checks() {
    log_info "Running health checks..."
    
    # Check API health
    if curl -f http://localhost:8080/api/health &> /dev/null; then
        log_success "API health check passed"
    else
        log_error "API health check failed"
        exit 1
    fi
    
    # Check database connection
    if docker-compose -f "$COMPOSE_FILE" exec -T postgres pg_isready -U styler_user -d styler &> /dev/null; then
        log_success "Database health check passed"
    else
        log_error "Database health check failed"
        exit 1
    fi
    
    # Check Redis connection
    if docker-compose -f "$COMPOSE_FILE" exec -T redis redis-cli ping &> /dev/null; then
        log_success "Redis health check passed"
    else
        log_error "Redis health check failed"
        exit 1
    fi
    
    log_success "All health checks passed"
}

# Display deployment information
display_deployment_info() {
    log_success "Deployment completed successfully!"
    echo
    echo "=== Deployment Information ==="
    echo "Project: $PROJECT_NAME"
    echo "Environment: Production"
    echo "Compose File: $COMPOSE_FILE"
    echo
    echo "=== Service URLs ==="
    echo "API: http://localhost:8080"
    echo "API Documentation: http://localhost:8080/api/docs"
    echo "Prometheus: http://localhost:9090"
    echo "Grafana: http://localhost:3000"
    echo "Loki: http://localhost:3100"
    echo
    echo "=== Default Credentials ==="
    echo "Grafana Admin: admin / $GRAFANA_PASSWORD"
    echo
    echo "=== Useful Commands ==="
    echo "View logs: docker-compose -f $COMPOSE_FILE logs -f"
    echo "Stop services: docker-compose -f $COMPOSE_FILE down"
    echo "Restart services: docker-compose -f $COMPOSE_FILE restart"
    echo "Scale app: docker-compose -f $COMPOSE_FILE up -d --scale app=3"
    echo
}

# Main deployment function
main() {
    log_info "Starting AI Styler production deployment..."
    
    check_root
    check_prerequisites
    validate_config
    create_directories
    generate_ssl_certificates
    deploy_services
    wait_for_services
    run_migrations
    setup_monitoring
    run_health_checks
    display_deployment_info
    
    log_success "Deployment completed successfully!"
}

# Run main function
main "$@"
