# User Service

The User Service manages user profiles, conversion tracking, and subscription plans for the AI Styler application.

## Features

### User Profile Management
- **Profile Information**: Name, phone, avatar, bio
- **Free Quota Tracking**: 2 free conversions per user
- **Profile Updates**: Update name, avatar, and bio

### Conversion Tracking
- **Conversion History**: Track all user conversions
- **Status Management**: Pending, processing, completed, failed
- **File Management**: Input and output file URLs
- **Quota Enforcement**: Free and paid conversion limits

### Subscription Plans
- **Plan Types**: Free, Basic, Premium, Enterprise
- **Usage Tracking**: Monthly conversion limits
- **Plan Management**: Create, update, and cancel plans
- **Billing Integration**: Price tracking and billing cycles

## API Endpoints

### Profile Management
- `GET /user/profile` - Get user profile
- `PUT /user/profile` - Update user profile

### Conversion Management
- `GET /user/conversions` - Get conversion history
- `POST /user/conversions` - Create new conversion
- `GET /user/conversions/:id` - Get specific conversion

### Quota Management
- `GET /user/quota` - Get current quota status

### Plan Management
- `GET /user/plan` - Get current user plan
- `POST /user/plan` - Create new plan
- `PUT /user/plan/:id` - Update plan status

## Database Schema

### Users Table (Extended)
```sql
-- Extended users table with profile fields
ALTER TABLE users ADD COLUMN name TEXT;
ALTER TABLE users ADD COLUMN avatar_url TEXT;
ALTER TABLE users ADD COLUMN bio TEXT;
ALTER TABLE users ADD COLUMN free_conversions_used INTEGER NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN free_conversions_limit INTEGER NOT NULL DEFAULT 2;
```

### User Conversions Table
```sql
CREATE TABLE user_conversions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    conversion_type TEXT NOT NULL CHECK (conversion_type IN ('free', 'paid')),
    input_file_url TEXT NOT NULL,
    output_file_url TEXT,
    style_name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    error_message TEXT,
    processing_time_ms INTEGER,
    file_size_bytes BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);
```

### User Plans Table
```sql
CREATE TABLE user_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plan_name TEXT NOT NULL CHECK (plan_name IN ('free', 'basic', 'premium', 'enterprise')),
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'cancelled', 'expired', 'suspended')),
    monthly_conversions_limit INTEGER NOT NULL DEFAULT 0,
    conversions_used_this_month INTEGER NOT NULL DEFAULT 0,
    price_per_month_cents INTEGER NOT NULL DEFAULT 0,
    billing_cycle_start_date DATE,
    billing_cycle_end_date DATE,
    auto_renew BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ
);
```

## Usage Examples

### Get User Profile
```bash
curl -X GET "http://localhost:8080/user/profile" \
  -H "Authorization: Bearer <access_token>"
```

### Update User Profile
```bash
curl -X PUT "http://localhost:8080/user/profile" \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "John Doe",
    "bio": "AI enthusiast and developer"
  }'
```

### Create Conversion
```bash
curl -X POST "http://localhost:8080/user/conversions" \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "inputFileUrl": "https://example.com/input.jpg",
    "styleName": "vintage",
    "type": "free"
  }'
```

### Get Conversion History
```bash
curl -X GET "http://localhost:8080/user/conversions?page=1&pageSize=20&status=completed" \
  -H "Authorization: Bearer <access_token>"
```

### Get Quota Status
```bash
curl -X GET "http://localhost:8080/user/quota" \
  -H "Authorization: Bearer <access_token>"
```

### Create User Plan
```bash
curl -X POST "http://localhost:8080/user/plan" \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "planName": "basic"
  }'
```

## Configuration

The User Service requires the following configuration:

```go
type DatabaseConfig struct {
    Host     string
    Port     int
    User     string
    Password string
    Name     string
    SSLMode  string
}
```

## Dependencies

- **Database**: PostgreSQL 13+ with pgcrypto extension
- **HTTP Framework**: Gin
- **Authentication**: JWT tokens (handled by auth service)
- **File Storage**: Configurable file storage interface
- **Notifications**: Configurable notification service
- **Rate Limiting**: Configurable rate limiter

## Testing

### Unit Tests
```bash
go test ./internal/user/...
```

### Integration Tests
```bash
go test -v ./internal/user/... -tags=integration
```

### Test Coverage
```bash
go test -cover ./internal/user/...
```

## Error Handling

The service provides comprehensive error handling:

- **Validation Errors**: 400 Bad Request
- **Authentication Errors**: 401 Unauthorized
- **Authorization Errors**: 403 Forbidden
- **Not Found Errors**: 404 Not Found
- **Quota Exceeded**: 429 Too Many Requests
- **Server Errors**: 500 Internal Server Error

## Rate Limiting

- **Conversion Creation**: 10 requests per hour per user
- **Profile Updates**: No specific limit (handled by general rate limiting)
- **Quota Checks**: No specific limit

## Security Considerations

- **Input Validation**: All inputs are validated and sanitized
- **SQL Injection**: Protected using parameterized queries
- **Authorization**: All endpoints require valid authentication
- **Data Privacy**: Sensitive data is properly handled and logged

## Monitoring and Logging

- **Audit Logging**: All user actions are logged
- **Error Tracking**: Comprehensive error logging
- **Performance Metrics**: Conversion processing times tracked
- **Usage Analytics**: Quota usage and plan statistics

## Future Enhancements

- **Advanced Analytics**: Detailed usage analytics and reporting
- **Plan Upgrades**: Seamless plan upgrade/downgrade flows
- **Usage Alerts**: Proactive quota warnings and notifications
- **Batch Operations**: Bulk conversion processing
- **API Versioning**: Versioned API endpoints for backward compatibility
