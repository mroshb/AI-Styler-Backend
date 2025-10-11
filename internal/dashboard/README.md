# Dashboard Service

The Dashboard Service provides comprehensive dashboard functionality for the AI Styler application, including quota management, conversion history, vendor gallery, and plan status information.

## Features

### 🎯 Core Dashboard Features
- **Comprehensive Dashboard Data**: Aggregates user information, quota status, conversion history, vendor gallery, and plan information
- **Quota Management**: Real-time quota checking with upgrade recommendations
- **Conversion History**: User's conversion history with pagination and filtering
- **Vendor Gallery**: Featured albums and recent images from vendors
- **Plan Status**: Current subscription plan information and available upgrades
- **Statistics**: User conversion statistics and trends
- **Recent Activity**: Recent user activity tracking

### 📊 Quota Management
- **Real-time Quota Checking**: Check remaining conversions and quota status
- **Upgrade Prompts**: Intelligent upgrade recommendations based on usage
- **Quota Exceeded Handling**: Proper error responses when quota is exceeded
- **Usage Percentage**: Calculate and display usage percentages

### 🎨 Vendor Gallery Integration
- **Featured Albums**: Display popular vendor albums
- **Recent Images**: Show latest vendor images
- **Popular Styles**: Track and display popular style trends
- **Public Gallery**: Public access to vendor content

### 📈 Analytics & Statistics
- **Conversion Statistics**: Success rates, processing times, and trends
- **Usage Analytics**: Track user behavior and preferences
- **Trend Analysis**: Historical data and usage patterns
- **Performance Metrics**: Processing time and success rate tracking

## API Endpoints

### Dashboard Endpoints

#### Get Dashboard Data
```http
GET /api/v1/dashboard
```
Retrieves comprehensive dashboard data including all user information.

**Query Parameters:**
- `includeHistory` (bool): Include conversion history (default: true)
- `includeGallery` (bool): Include vendor gallery (default: true)
- `includeStatistics` (bool): Include statistics (default: true)
- `historyLimit` (int): Limit for conversion history (default: 10, max: 50)
- `galleryLimit` (int): Limit for gallery items (default: 20, max: 100)

#### Get Quota Status
```http
GET /api/v1/dashboard/quota
```
Retrieves current quota status and upgrade recommendations.

#### Check Quota Exceeded
```http
POST /api/v1/dashboard/quota/check
```
Checks if user has exceeded their conversion quota.

#### Get Conversion History
```http
GET /api/v1/dashboard/conversions
```
Retrieves user's conversion history with pagination.

#### Get Vendor Gallery
```http
GET /api/v1/dashboard/gallery
```
Retrieves featured vendor albums and recent images.

#### Get Plan Status
```http
GET /api/v1/dashboard/plan
```
Retrieves current subscription plan information and available upgrades.

#### Get Statistics
```http
GET /api/v1/dashboard/statistics
```
Retrieves user's conversion statistics and trends.

#### Get Recent Activity
```http
GET /api/v1/dashboard/activity
```
Retrieves recent user activity including conversions, logins, and plan changes.

#### Invalidate Cache
```http
POST /api/v1/dashboard/cache/invalidate
```
Invalidates cached dashboard data for the current user.

### Public Endpoints

#### Get Public Gallery
```http
GET /api/v1/public/gallery
```
Retrieves public vendor gallery information (no authentication required).

## Data Models

### DashboardData
Complete dashboard information including all user data.

### QuotaStatus
Current quota information with usage statistics.

### ConversionHistory
User's conversion history with statistics.

### VendorGallery
Vendor gallery information including albums and images.

### PlanStatus
Current plan information and available upgrades.

### UpgradePrompt
Intelligent upgrade recommendations based on usage.

### DashboardStatistics
User statistics and conversion trends.

### RecentActivity
Recent user activity tracking.

## Business Logic

### Quota Management
- **Free Conversions**: Users get 2 free conversions permanently
- **Paid Conversions**: Based on subscription plan limits
- **Usage Tracking**: Real-time tracking of conversion usage
- **Upgrade Recommendations**: Intelligent prompts based on usage patterns

### Upgrade Prompts
- **Urgency Levels**: Low, Medium, High based on remaining quota
- **Smart Recommendations**: Suggest appropriate plans based on usage
- **Contextual Messages**: Different messages for different scenarios

### Caching Strategy
- **Dashboard Data**: Cached for 5 minutes to improve performance
- **Cache Invalidation**: Manual invalidation when data changes
- **Cache Keys**: User-specific cache keys for data isolation

### Error Handling
- **Quota Exceeded**: Returns 403 Forbidden with upgrade information
- **Service Errors**: Proper error responses with meaningful messages
- **Validation**: Input validation with appropriate error messages

## Integration

### Dependencies
- **User Service**: User profile and quota information
- **Conversion Service**: Conversion history and statistics
- **Vendor Service**: Gallery and album information
- **Payment Service**: Plan and payment information
- **Cache Service**: Data caching for performance
- **Metrics Service**: Analytics and monitoring
- **Audit Service**: Activity logging and compliance

### Database Functions
- `get_user_quota_status(user_id)`: Get current quota status
- `can_user_convert(user_id, conversion_type)`: Check if user can convert
- `record_conversion(...)`: Record conversion and update quotas

## Testing

### Test Coverage
- **Unit Tests**: Service layer business logic
- **Handler Tests**: HTTP endpoint testing
- **Integration Tests**: Database and service integration
- **Mock Testing**: Comprehensive mock implementations

### Test Scenarios
- **Dashboard Data Retrieval**: Complete dashboard data aggregation
- **Quota Management**: Quota checking and upgrade prompts
- **Error Handling**: Various error scenarios and responses
- **Caching**: Cache hit/miss scenarios
- **Validation**: Input validation and edge cases

## Performance Considerations

### Optimization
- **Caching**: Dashboard data cached for 5 minutes
- **Database Queries**: Optimized queries with proper indexing
- **Pagination**: Proper pagination for large datasets
- **Lazy Loading**: Optional data loading based on request parameters

### Monitoring
- **Metrics Collection**: Dashboard view and quota check metrics
- **Audit Logging**: User activity and access logging
- **Performance Tracking**: Response time and error rate monitoring

## Security

### Authentication
- **JWT Tokens**: Bearer token authentication for protected endpoints
- **User Context**: User ID extracted from authenticated context
- **Authorization**: Role-based access control

### Data Protection
- **User Isolation**: User-specific data access only
- **Input Validation**: Proper input sanitization and validation
- **Audit Trail**: Complete audit logging for compliance

## Configuration

### Environment Variables
- `DASHBOARD_CACHE_TTL`: Cache TTL in seconds (default: 300)
- `DASHBOARD_MAX_HISTORY_LIMIT`: Maximum history limit (default: 50)
- `DASHBOARD_MAX_GALLERY_LIMIT`: Maximum gallery limit (default: 100)

### Default Values
- **History Limit**: 10 conversions
- **Gallery Limit**: 20 items
- **Cache TTL**: 5 minutes
- **Activity Limit**: 10 activities

## Future Enhancements

### Planned Features
- **Real-time Updates**: WebSocket support for real-time dashboard updates
- **Advanced Analytics**: More detailed analytics and reporting
- **Custom Dashboards**: User-customizable dashboard layouts
- **Notification Integration**: Real-time notifications and alerts
- **Export Functionality**: Data export capabilities
- **Mobile Optimization**: Mobile-specific dashboard views
