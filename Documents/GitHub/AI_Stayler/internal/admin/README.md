# Admin Service

The Admin Service provides comprehensive administrative functionality for managing users, vendors, plans, payments, conversions, and system monitoring.

## Features

### User Management
- **List Users**: Get paginated list of users with filtering by role, search, and active status
- **Get User**: Retrieve detailed user information by ID
- **Update User**: Modify user profile, role, verification status, and quota limits
- **Delete User**: Remove user accounts
- **Suspend/Activate User**: Control user account status
- **Revoke Quota**: Remove user conversion quotas with reason tracking

### Vendor Management
- **List Vendors**: Get paginated list of vendors with filtering by verification status, search, and active status
- **Get Vendor**: Retrieve detailed vendor information by ID
- **Update Vendor**: Modify vendor profile, business information, and settings
- **Delete Vendor**: Remove vendor accounts
- **Suspend/Activate Vendor**: Control vendor account status
- **Verify Vendor**: Mark vendors as verified
- **Revoke Quota**: Remove vendor image quotas with reason tracking

### Plan Management
- **List Plans**: Get paginated list of subscription plans
- **Get Plan**: Retrieve detailed plan information by ID
- **Create Plan**: Create new subscription plans with pricing and features
- **Update Plan**: Modify existing plans
- **Delete Plan**: Remove subscription plans
- **Revoke User Plan**: Cancel user subscriptions with reason tracking

### Payment Management
- **List Payments**: Get paginated list of payments with filtering by status, user, plan, and date range
- **Get Payment**: Retrieve detailed payment information by ID
- **Payment Statistics**: View payment totals and revenue

### Conversion Management
- **List Conversions**: Get paginated list of conversions with filtering by status, user, type, and date range
- **Get Conversion**: Retrieve detailed conversion information by ID
- **Conversion Statistics**: View conversion totals, pending, and failed counts

### Image Management
- **List Images**: Get paginated list of vendor images with filtering by vendor, visibility, type, and date range
- **Get Image**: Retrieve detailed image information by ID
- **Image Statistics**: View total image counts

### Audit Trail
- **List Audit Logs**: Get paginated list of system audit logs with filtering by user, action, resource, and date range
- **Action Logging**: Automatic logging of all admin actions with metadata

### Statistics & Monitoring
- **System Stats**: Comprehensive system-wide statistics
- **User Stats**: User count and activity metrics
- **Vendor Stats**: Vendor count and activity metrics
- **Payment Stats**: Payment totals and revenue metrics
- **Conversion Stats**: Conversion totals and status metrics
- **Image Stats**: Image count metrics

## API Endpoints

### User Management
```
GET    /admin/users                    # List users
GET    /admin/users/:id                # Get user
PUT    /admin/users/:id                # Update user
DELETE /admin/users/:id                # Delete user
POST   /admin/users/:id/suspend        # Suspend user
POST   /admin/users/:id/activate       # Activate user
POST   /admin/users/:id/revoke-quota   # Revoke user quota
POST   /admin/users/:id/revoke-plan    # Revoke user plan
```

### Vendor Management
```
GET    /admin/vendors                    # List vendors
GET    /admin/vendors/:id                # Get vendor
PUT    /admin/vendors/:id                # Update vendor
DELETE /admin/vendors/:id                # Delete vendor
POST   /admin/vendors/:id/suspend        # Suspend vendor
POST   /admin/vendors/:id/activate       # Activate vendor
POST   /admin/vendors/:id/verify         # Verify vendor
POST   /admin/vendors/:id/revoke-quota   # Revoke vendor quota
```

### Plan Management
```
GET    /admin/plans        # List plans
GET    /admin/plans/:id    # Get plan
POST   /admin/plans        # Create plan
PUT    /admin/plans/:id    # Update plan
DELETE /admin/plans/:id    # Delete plan
```

### Payment Management
```
GET    /admin/payments        # List payments
GET    /admin/payments/:id    # Get payment
```

### Conversion Management
```
GET    /admin/conversions        # List conversions
GET    /admin/conversions/:id    # Get conversion
```

### Image Management
```
GET    /admin/images        # List images
GET    /admin/images/:id    # Get image
```

### Audit Trail
```
GET    /admin/audit-logs    # List audit logs
```

### Statistics
```
GET    /admin/stats              # System stats
GET    /admin/stats/users        # User stats
GET    /admin/stats/vendors      # Vendor stats
GET    /admin/stats/payments     # Payment stats
GET    /admin/stats/conversions  # Conversion stats
GET    /admin/stats/images       # Image stats
```

## Authentication & Authorization

All admin endpoints require:
1. **Authentication**: Valid JWT token
2. **Authorization**: User must have `admin` role
3. **Rate Limiting**: Admin-specific rate limits
4. **Audit Logging**: All actions are logged

## Data Models

### AdminUser
```go
type AdminUser struct {
    ID                   string     `json:"id"`
    Phone                string     `json:"phone"`
    Name                 *string    `json:"name,omitempty"`
    AvatarURL            *string    `json:"avatarUrl,omitempty"`
    Bio                  *string    `json:"bio,omitempty"`
    Role                 string     `json:"role"`
    IsPhoneVerified      bool       `json:"isPhoneVerified"`
    FreeConversionsUsed  int        `json:"freeConversionsUsed"`
    FreeConversionsLimit int        `json:"freeConversionsLimit"`
    CreatedAt            time.Time  `json:"createdAt"`
    UpdatedAt            time.Time  `json:"updatedAt"`
    LastLoginAt          *time.Time `json:"lastLoginAt,omitempty"`
    IsActive             bool       `json:"isActive"`
}
```

### AdminVendor
```go
type AdminVendor struct {
    ID              string      `json:"id"`
    UserID          string      `json:"userId"`
    BusinessName    string      `json:"businessName"`
    AvatarURL       *string     `json:"avatarUrl,omitempty"`
    Bio             *string     `json:"bio,omitempty"`
    ContactInfo     ContactInfo `json:"contactInfo"`
    SocialLinks     SocialLinks `json:"socialLinks"`
    IsVerified      bool        `json:"isVerified"`
    IsActive        bool        `json:"isActive"`
    FreeImagesUsed  int         `json:"freeImagesUsed"`
    FreeImagesLimit int         `json:"freeImagesLimit"`
    CreatedAt       time.Time   `json:"createdAt"`
    UpdatedAt       time.Time   `json:"updatedAt"`
    LastLoginAt     *time.Time  `json:"lastLoginAt,omitempty"`
}
```

### AdminPlan
```go
type AdminPlan struct {
    ID                      string    `json:"id"`
    Name                    string    `json:"name"`
    DisplayName             string    `json:"displayName"`
    Description             string    `json:"description"`
    PricePerMonthCents      int64     `json:"pricePerMonthCents"`
    MonthlyConversionsLimit int       `json:"monthlyConversionsLimit"`
    Features                []string  `json:"features"`
    IsActive                bool      `json:"isActive"`
    CreatedAt               time.Time `json:"createdAt"`
    UpdatedAt               time.Time `json:"updatedAt"`
    SubscriberCount         int       `json:"subscriberCount"`
}
```

### AdminStats
```go
type AdminStats struct {
    TotalUsers         int   `json:"totalUsers"`
    ActiveUsers        int   `json:"activeUsers"`
    TotalVendors       int   `json:"totalVendors"`
    ActiveVendors      int   `json:"activeVendors"`
    TotalConversions   int   `json:"totalConversions"`
    TotalPayments      int   `json:"totalPayments"`
    TotalRevenue       int64 `json:"totalRevenue"`
    TotalImages        int   `json:"totalImages"`
    PendingConversions int   `json:"pendingConversions"`
    FailedConversions  int   `json:"failedConversions"`
}
```

## Usage Examples

### List Users with Filtering
```bash
curl -H "Authorization: Bearer <admin_token>" \
     "https://api.example.com/admin/users?page=1&pageSize=20&role=user&search=john"
```

### Update User
```bash
curl -X PUT \
     -H "Authorization: Bearer <admin_token>" \
     -H "Content-Type: application/json" \
     -d '{"name": "John Doe", "role": "admin"}' \
     "https://api.example.com/admin/users/user123"
```

### Suspend User
```bash
curl -X POST \
     -H "Authorization: Bearer <admin_token>" \
     -H "Content-Type: application/json" \
     -d '{"reason": "Violation of terms of service"}' \
     "https://api.example.com/admin/users/user123/suspend"
```

### Create Plan
```bash
curl -X POST \
     -H "Authorization: Bearer <admin_token>" \
     -H "Content-Type: application/json" \
     -d '{
       "name": "premium",
       "displayName": "Premium Plan",
       "description": "Premium subscription with unlimited conversions",
       "pricePerMonthCents": 150000,
       "monthlyConversionsLimit": 100,
       "features": ["unlimited_conversions", "priority_support"],
       "isActive": true
     }' \
     "https://api.example.com/admin/plans"
```

### Revoke User Quota
```bash
curl -X POST \
     -H "Authorization: Bearer <admin_token>" \
     -H "Content-Type: application/json" \
     -d '{
       "quotaType": "free",
       "amount": 5,
       "reason": "Abuse of free quota"
     }' \
     "https://api.example.com/admin/users/user123/revoke-quota"
```

### Get System Statistics
```bash
curl -H "Authorization: Bearer <admin_token>" \
     "https://api.example.com/admin/stats"
```

## Error Handling

The admin service returns appropriate HTTP status codes:

- `200 OK`: Successful operation
- `201 Created`: Resource created successfully
- `400 Bad Request`: Invalid request data
- `401 Unauthorized`: Missing or invalid authentication
- `403 Forbidden`: Insufficient permissions
- `404 Not Found`: Resource not found
- `500 Internal Server Error`: Server error

## Rate Limiting

Admin endpoints have specific rate limits:
- **Read operations**: 1000 requests per hour
- **Write operations**: 100 requests per hour
- **Bulk operations**: 10 requests per hour

## Audit Trail

All admin actions are automatically logged with:
- **Actor**: Admin user ID
- **Action**: Type of action performed
- **Resource**: Resource type and ID
- **Metadata**: Additional context and changes
- **Timestamp**: When the action occurred

## Testing

Run the test suite:

```bash
go test ./internal/admin/...
```

Run tests with coverage:

```bash
go test -cover ./internal/admin/...
```

## Dependencies

- **Database**: PostgreSQL with existing schema
- **Authentication**: JWT-based authentication
- **Notifications**: Email and SMS notification service
- **Audit Logging**: Centralized audit logging system

## Security Considerations

1. **Admin-only access**: All endpoints require admin role
2. **Input validation**: All inputs are validated and sanitized
3. **Audit logging**: All actions are logged for compliance
4. **Rate limiting**: Prevents abuse and DoS attacks
5. **Data encryption**: Sensitive data is encrypted at rest
6. **Secure headers**: Security headers are applied to all responses
