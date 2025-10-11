# AI Stayler - Setup and Configuration Guide

## 🚀 Quick Start

### Prerequisites
- Go 1.24.4+
- PostgreSQL 13+
- Redis (optional, for caching)

### 1. Database Setup

```bash
# Create database
createdb styler

# Run migrations
psql -d styler -f db/migrations/0001_auth.sql
psql -d styler -f db/migrations/0002_user_service.sql
psql -d styler -f db/migrations/0003_vendor_service.sql
psql -d styler -f db/migrations/0004_image_service.sql
psql -d styler -f db/migrations/0005_conversion_service.sql
psql -d styler -f db/migrations/0006_payment_service.sql
psql -d styler -f db/migrations/0007_admin_service.sql
psql -d styler -f db/migrations/0008_notification_service.sql
psql -d styler -f db/migrations/0009_comprehensive_schema.sql
psql -d styler -f db/migrations/0010_conversions_images_schema.sql
```

### 2. Environment Configuration

Create a `.env` file with the following variables:

```bash
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=styler
DB_SSLMODE=disable

# Server
HTTP_ADDR=:8080
GIN_MODE=debug

# JWT
JWT_SECRET=your-secret-key-change-in-production
JWT_ACCESS_TTL=15m
JWT_REFRESH_TTL=720h

# SMS (for OTP)
SMS_PROVIDER=mock
SMS_API_KEY=your_sms_api_key
SMS_TEMPLATE_ID=100000

# Rate Limiting
RATE_LIMIT_OTP_PER_PHONE=3
RATE_LIMIT_OTP_PER_IP=100
RATE_LIMIT_WINDOW=1h
```

### 3. Build and Run

```bash
# Install dependencies
go mod download

# Build application
go build -o server .

# Run application
./server
```

### 4. Test the Application

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test ./... -cover

# Run specific service tests
go test ./internal/auth -v
go test ./internal/user -v
go test ./internal/vendor -v
```

## 📊 Service Status

| Service | Status | Tests | Coverage |
|---------|--------|-------|----------|
| Auth | ✅ Complete | 18/18 | 66.7% |
| Admin | ✅ Complete | 24/24 | 11.8% |
| User | ✅ Complete | 16/16 | 27.4% |
| Vendor | ✅ Complete | 16/16 | 23.7% |
| Image | ✅ Complete | 4/4 | 10.8% |
| Conversion | ✅ Complete | 3/3 | 6.7% |
| Payment | ✅ Complete | 5/5 | 11.7% |
| Notification | ✅ Complete | 8/8 | 0.3% |
| SMS | ✅ Complete | 7/7 | 86.5% |
| Security | ✅ Complete | 10/10 | 33.1% |
| Worker | ✅ Complete | 6/6 | 25.6% |
| Config | ✅ Complete | 4/4 | 100% |

## 🔧 API Endpoints

### Authentication
- `POST /auth/send-otp` - Send OTP
- `POST /auth/verify-otp` - Verify OTP
- `POST /auth/register` - Register user
- `POST /auth/login` - Login
- `POST /auth/refresh` - Refresh token
- `POST /auth/logout` - Logout

### User Management
- `GET /user/profile` - Get profile
- `PUT /user/profile` - Update profile
- `GET /user/conversions` - Get conversions
- `POST /user/conversions` - Create conversion
- `GET /user/quota` - Get quota status

### Vendor Management
- `GET /vendor/profile` - Get vendor profile
- `POST /vendor/profile` - Create vendor profile
- `PUT /vendor/profile` - Update vendor profile
- `GET /vendor/albums` - Get albums
- `POST /vendor/albums` - Create album
- `GET /vendor/images` - Get images
- `POST /vendor/images` - Upload image

### Admin Panel
- `GET /api/admin/users` - Get all users
- `GET /api/admin/vendors` - Get all vendors
- `GET /api/admin/stats` - Get system statistics
- `POST /api/admin/plans` - Create payment plan

## 🗄️ Database Schema

The application uses PostgreSQL with the following main tables:
- `users` - User accounts and profiles
- `vendors` - Vendor accounts and profiles
- `images` - Image storage and metadata
- `conversions` - Image conversion requests
- `payments` - Payment transactions
- `sessions` - User sessions
- `otps` - OTP verification codes
- `audit_logs` - System audit trail

## 🧪 Testing

All services have comprehensive test coverage:
- **Unit Tests**: Test individual components
- **Integration Tests**: Test service interactions
- **Handler Tests**: Test HTTP endpoints
- **Service Tests**: Test business logic

Some integration tests require a database connection and will be skipped if not available.

## 🔒 Security Features

- JWT-based authentication
- Password hashing with BCrypt/Argon2
- Rate limiting for API endpoints
- CORS protection
- Security headers
- Input validation
- Audit logging

## 📈 Performance

- Database connection pooling
- Efficient query patterns
- Caching support (Redis)
- Rate limiting
- Async processing for heavy operations

## 🚀 Production Deployment

1. Set `GIN_MODE=release`
2. Use strong JWT secrets
3. Enable SSL for database connections
4. Configure proper rate limits
5. Set up monitoring and logging
6. Use environment-specific configurations

## 📞 Support

For issues or questions, please check the service-specific README files in each service directory.
