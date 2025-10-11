# 🎨 AI Styler Backend - Production-Ready GoLang API

[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org/)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED.svg)](https://www.docker.com/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15+-316192.svg)](https://www.postgresql.org/)
[![Redis](https://img.shields.io/badge/Redis-7+-DC382D.svg)](https://redis.io/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

A production-ready, scalable GoLang backend API for AI-powered image styling and conversion services. Built with enterprise-grade security, monitoring, and performance optimizations.

## 🚀 **Features**

### 🔐 **Security**
- **Production JWT Implementation** with proper token rotation and session management
- **Argon2 & BCrypt** password hashing with configurable parameters
- **Redis-based Rate Limiting** with sliding window algorithm
- **Circuit Breaker Pattern** for external service resilience
- **Comprehensive Input Validation** and sanitization
- **Security Headers** and CORS protection
- **Audit Logging** for all user actions

### 📊 **Monitoring & Observability**
- **Distributed Tracing** with OpenTelemetry
- **Structured Logging** with multiple output formats
- **Prometheus Metrics** collection
- **Grafana Dashboards** for visualization
- **Health Checks** for all services
- **Sentry Integration** for error tracking
- **Telegram Alerts** for critical issues

### ⚡ **Performance**
- **Connection Pooling** with optimized settings
- **Redis Caching** for session and rate limiting
- **Prepared Statements** for database queries
- **Goroutine-safe** implementations
- **Memory-efficient** data structures
- **Horizontal Scaling** support

### 🏗️ **Architecture**
- **Clean Architecture** with clear separation of concerns
- **Dependency Injection** with proper interfaces
- **Microservices-ready** design
- **Event-driven** patterns
- **Comprehensive Error Handling**
- **Graceful Degradation**

## 📋 **Prerequisites**

- **Go 1.21+**
- **Docker & Docker Compose**
- **PostgreSQL 15+**
- **Redis 7+**
- **Make** (optional, for development)

## 🛠️ **Quick Start**

### 1. **Clone Repository**
```bash
git clone https://github.com/your-org/ai-styler-backend.git
cd ai-styler-backend
```

### 2. **Environment Setup**
```bash
# Copy environment template
cp .env.example .env

# Edit configuration
nano .env
```

### 3. **Development Mode**
```bash
# Start development environment
docker-compose up -d

# Run migrations
make migrate

# Start application
go run main.go
```

### 4. **Production Deployment**
```bash
# Make deployment script executable
chmod +x scripts/deploy-prod.sh

# Deploy to production
./scripts/deploy-prod.sh
```

## 🔧 **Configuration**

### **Environment Variables**

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `DB_HOST` | Database host | `localhost` | ✅ |
| `DB_PASSWORD` | Database password | - | ✅ |
| `JWT_SECRET` | JWT signing secret (min 32 chars) | - | ✅ |
| `REDIS_PASSWORD` | Redis password | - | ✅ |
| `GEMINI_API_KEY` | Gemini AI API key | - | ✅ |
| `SMS_API_KEY` | SMS service API key | - | ✅ |
| `SENTRY_DSN` | Sentry error tracking DSN | - | ❌ |

### **Security Configuration**
```bash
# Password Hashing
BCRYPT_COST=12
ARGON2_MEMORY=65536
ARGON2_ITERATIONS=3
ARGON2_PARALLELISM=2

# Rate Limiting
RATE_LIMIT_OTP_PER_PHONE=3
RATE_LIMIT_OTP_PER_IP=100
RATE_LIMIT_WINDOW=1h

# Session Management
SESSION_TIMEOUT=24h
MAX_LOGIN_ATTEMPTS=5
LOCKOUT_DURATION=15m
```

## 📚 **API Documentation**

### **Interactive Documentation**
- **Swagger UI**: `http://localhost:8080/api/docs`
- **OpenAPI Spec**: `http://localhost:8080/api/docs/openapi.json`

### **Core Endpoints**

#### **Authentication**
```bash
POST /api/auth/send-otp      # Send OTP to phone
POST /api/auth/verify-otp    # Verify OTP code
POST /api/auth/register      # Register new user
POST /api/auth/login         # User login
POST /api/auth/refresh       # Refresh access token
POST /api/auth/logout        # Logout user
```

#### **Conversions**
```bash
POST   /api/conversions      # Create conversion
GET    /api/conversions      # List conversions
GET    /api/conversions/:id  # Get conversion details
PUT    /api/conversions/:id  # Update conversion
DELETE /api/conversions/:id  # Delete conversion
```

#### **Health & Monitoring**
```bash
GET /api/health              # Health check
GET /api/metrics             # Prometheus metrics
GET /api/status              # Service status
```

## 🧪 **Testing**

### **Run Tests**
```bash
# Unit tests
go test ./...

# Integration tests
go test -tags=integration ./...

# Coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### **Test Categories**
- **Unit Tests**: Individual component testing
- **Integration Tests**: Service integration testing
- **Performance Tests**: Load and stress testing
- **Security Tests**: Security vulnerability testing

## 📊 **Monitoring**

### **Grafana Dashboards**
- **System Overview**: CPU, Memory, Disk usage
- **API Metrics**: Request rates, response times, error rates
- **Database Metrics**: Connection pools, query performance
- **Business Metrics**: User registrations, conversions, revenue

### **Prometheus Metrics**
```bash
# Application metrics
http_requests_total
http_request_duration_seconds
database_queries_total
database_query_duration_seconds

# Business metrics
user_registrations_total
conversions_total
payments_total
```

### **Log Aggregation**
- **Structured Logs**: JSON format with correlation IDs
- **Log Levels**: DEBUG, INFO, WARN, ERROR, FATAL
- **Log Retention**: Configurable retention period
- **Log Shipping**: Automatic log forwarding to Loki

## 🚀 **Deployment**

### **Production Deployment**
```bash
# Automated deployment
./scripts/deploy-prod.sh

# Manual deployment
docker-compose -f docker-compose.prod.yml up -d
```

### **Scaling**
```bash
# Scale application instances
docker-compose -f docker-compose.prod.yml up -d --scale app=3

# Scale with load balancer
docker-compose -f docker-compose.prod.yml up -d --scale app=5
```

### **High Availability**
- **Database**: PostgreSQL with replication
- **Cache**: Redis Cluster
- **Application**: Multiple instances with load balancing
- **Monitoring**: Redundant monitoring services

## 🔒 **Security**

### **Authentication & Authorization**
- **JWT Tokens**: Secure token-based authentication
- **Session Management**: Redis-backed session storage
- **Role-based Access**: User and admin role separation
- **Token Rotation**: Automatic refresh token rotation

### **Data Protection**
- **Password Hashing**: Argon2 and BCrypt with salt
- **Input Validation**: Comprehensive input sanitization
- **SQL Injection**: Prepared statements and parameterized queries
- **XSS Protection**: Content Security Policy headers

### **Network Security**
- **HTTPS**: TLS encryption for all communications
- **CORS**: Configurable cross-origin resource sharing
- **Rate Limiting**: Protection against brute force attacks
- **IP Whitelisting**: Optional IP-based access control

## 📈 **Performance**

### **Optimization Features**
- **Connection Pooling**: Optimized database connections
- **Redis Caching**: High-performance caching layer
- **Goroutine Management**: Efficient concurrent processing
- **Memory Management**: Optimized memory usage patterns

### **Benchmarks**
```bash
# Run performance benchmarks
go test -bench=. -benchmem ./...

# Load testing
go test -tags=loadtest ./...
```

### **Performance Targets**
- **Response Time**: < 200ms for 95th percentile
- **Throughput**: > 1000 requests/second
- **Memory Usage**: < 512MB per instance
- **CPU Usage**: < 80% under normal load

## 🛠️ **Development**

### **Project Structure**
```
├── cmd/                    # Application entry points
├── internal/               # Private application code
│   ├── auth/              # Authentication service
│   ├── common/            # Shared utilities
│   ├── config/            # Configuration management
│   ├── docs/              # API documentation
│   ├── monitoring/        # Monitoring and tracing
│   ├── security/          # Security utilities
│   └── test/              # Test utilities
├── db/                    # Database migrations
├── scripts/               # Deployment scripts
├── monitoring/            # Monitoring configuration
└── docker-compose.yml     # Development environment
```

### **Code Quality**
- **Linting**: `golangci-lint` with strict rules
- **Formatting**: `gofmt` and `goimports`
- **Testing**: Comprehensive test coverage
- **Documentation**: GoDoc comments for all public APIs

### **Development Workflow**
```bash
# Install dependencies
go mod download

# Run linter
golangci-lint run

# Format code
go fmt ./...
goimports -w .

# Run tests
go test ./...

# Build application
go build -o ai-styler main.go
```

## 🤝 **Contributing**

### **Contributing Guidelines**
1. **Fork** the repository
2. **Create** a feature branch
3. **Write** tests for new functionality
4. **Ensure** all tests pass
5. **Submit** a pull request

### **Code Standards**
- **Go 1.21+** features and idioms
- **Clean Architecture** principles
- **Comprehensive Testing** (min 80% coverage)
- **Documentation** for all public APIs
- **Security** best practices

## 📄 **License**

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🆘 **Support**

### **Documentation**
- **API Docs**: `/api/docs`
- **Architecture**: [ARCHITECTURE.md](docs/ARCHITECTURE.md)
- **Deployment**: [DEPLOYMENT.md](docs/DEPLOYMENT.md)
- **Security**: [SECURITY.md](docs/SECURITY.md)

### **Community**
- **Issues**: [GitHub Issues](https://github.com/your-org/ai-styler-backend/issues)
- **Discussions**: [GitHub Discussions](https://github.com/your-org/ai-styler-backend/discussions)
- **Discord**: [Community Discord](https://discord.gg/your-discord)

### **Professional Support**
- **Email**: support@aistyler.com
- **Slack**: [Enterprise Slack](https://your-slack.com)
- **Consulting**: [Professional Services](https://aistyler.com/services)

---

## 🎯 **Roadmap**

### **Q1 2024**
- [ ] **Microservices Architecture** migration
- [ ] **Event Sourcing** implementation
- [ ] **Advanced Caching** strategies
- [ ] **Multi-region** deployment support

### **Q2 2024**
- [ ] **GraphQL API** implementation
- [ ] **Real-time** WebSocket support
- [ ] **Advanced Analytics** dashboard
- [ ] **AI Model** optimization

### **Q3 2024**
- [ ] **Kubernetes** deployment support
- [ ] **Service Mesh** integration
- [ ] **Advanced Security** features
- [ ] **Performance** optimizations

---

**Built with ❤️ by the AI Styler Team**

*For more information, visit [aistyler.com](https://aistyler.com)*