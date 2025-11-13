# AI Styler Backend API Documentation

## 📋 فهرست

- [نصب و راه‌اندازی](#نصب-و-راه‌اندازی)
- [Authentication](#authentication)
- [User Management](#user-management)
- [Conversion](#conversion)
- [Images](#images)
- [Vendors](#vendors)
- [Payment](#payment)
- [Share](#share)
- [Notifications](#notifications)
- [Admin](#admin)
- [Health](#health)

---

## نصب و راه‌اندازی

### Import در Postman

1. فایل `postman_collection.json` را در Postman import کنید
2. فایل `postman_environment.json` را به عنوان Environment import کنید
3. Environment را انتخاب کنید و `base_url` را تنظیم کنید (پیش‌فرض: `http://localhost:8080`)

### استفاده از Variables

- `{{base_url}}` - آدرس پایه API (پیش‌فرض: `http://localhost:8080`)
- `{{access_token}}` - توکن دسترسی کاربر
- `{{admin_access_token}}` - توکن دسترسی ادمین
- `{{refresh_token}}` - توکن refresh

---

## Authentication

### Send OTP
```
POST /auth/send-otp
```

**Request Body:**
```json
{
  "phone": "+989123456789",
  "purpose": "phone_verify",
  "channel": "sms"
}
```

**Response:**
```json
{
  "sent": true,
  "expiresInSec": 300,
  "code": "123456"  // Only in development/mock mode
}
```

---

### Verify OTP
```
POST /auth/verify-otp
```

**Request Body:**
```json
{
  "phone": "+989123456789",
  "code": "123456"
}
```

---

### Check User
```
POST /auth/check-user
```

**Request Body:**
```json
{
  "phone": "+989123456789"
}
```

**Response:**
```json
{
  "registered": true
}
```

توضیح: اگر شماره قبلاً به عنوان `user` یا `vendor` ثبت شده باشد مقدار `registered` برابر `true` خواهد بود.

---

### Register
```
POST /auth/register
```

**Request Body:**
```json
{
  "phone": "+989123456789",
  "password": "SecurePassword123!",
  "name": "John Doe",
  "role": "user"
}
```

**Response:**
```json
{
  "userId": "uuid-here",
  "role": "user",
  "isPhoneVerified": true,
  "accessToken": "token-here",  // If autoLogin: true
  "accessTokenExpiresIn": 900,
  "refreshToken": "refresh-token-here",
  "refreshTokenExpires": "2025-11-05T10:00:00Z"
}
```

---

### Login
```
POST /auth/login
```

**Request Body:**
```json
{
  "phone": "+989123456789",
  "password": "SecurePassword123!"
}
```

**Response:**
```json
{
  "accessToken": "token-here",
  "accessTokenExpiresIn": 900,
  "refreshToken": "refresh-token-here",
  "refreshTokenExpiresAt": "2025-11-05T10:00:00Z",
  "user": {
    "id": "uuid-here",
    "role": "user",
    "isPhoneVerified": true
  }
}
```

---

### Refresh Token
```
POST /auth/refresh
```

**Request Body:**
```json
{
  "refreshToken": "refresh-token-here"
}
```

---

### Logout
```
POST /auth/logout
Headers: Authorization: Bearer {access_token}
```

---

### Logout All
```
POST /auth/logout-all
Headers: Authorization: Bearer {access_token}
```

---

## User Management

### Get Profile
```
GET /api/user/profile
Headers: Authorization: Bearer {access_token}
```

**Response:**
```json
{
  "id": "uuid-here",
  "phone": "+989123456789",
  "name": "John Doe",
  "avatarUrl": "https://example.com/avatar.jpg",
  "bio": "Software developer",
  "role": "user",
  "isPhoneVerified": true,
  "isActive": true,
  "freeConversionsUsed": 0,
  "freeConversionsLimit": 2,
  "createdAt": "2025-11-04T10:00:00Z",
  "updatedAt": "2025-11-04T10:00:00Z"
}
```

---

### Update Profile
```
PUT /api/user/profile
Headers: Authorization: Bearer {access_token}
```

**Request Body:**
```json
{
  "name": "John Doe",
  "avatarUrl": "https://example.com/avatar.jpg",
  "bio": "Software developer"
}
```

---

## Conversion

### Create Conversion
```
POST /api/convert
Headers: Authorization: Bearer {access_token}
```

این endpoint کانورژن را ایجاد می‌کند و منتظر می‌ماند تا پردازش کامل شود (long polling). سپس نتیجه کامل کانورژن را برمی‌گرداند.

**Query Parameters (اختیاری):**
- `mock` (optional): اگر `true` باشد، یک پاسخ mock برگردانده می‌شود بدون استفاده از سرویس AI (برای تست)
- `timeout` (optional): حداکثر زمان انتظار به ثانیه (پیش‌فرض: 300 ثانیه، حداکثر: 1800 ثانیه)
- `poll_interval` (optional): فاصله بررسی وضعیت به میلی‌ثانیه (پیش‌فرض: 25ms، حداقل: 10ms، حداکثر: 10000ms)

**Request Body:**
```json
{
  "userImageId": "uuid-here",
  "clothImageId": "uuid-here",
  "styleName": "vintage"
}
```

**Response:**
این endpoint همیشه نتیجه کامل کانورژن را برمی‌گرداند. در صورت موفقیت، `status` برابر `completed` و `resultImageId` شامل شناسه تصویر نتیجه است.

```json
{
  "id": "conversion-uuid",
  "userId": "user-uuid",
  "userImageId": "image-uuid",
  "clothImageId": "cloth-uuid",
  "status": "completed",
  "resultImageId": "result-image-uuid",
  "errorMessage": null,
  "processingTimeMs": 5000,
  "createdAt": "2025-11-04T10:00:00Z",
  "updatedAt": "2025-11-04T10:05:00Z",
  "completedAt": "2025-11-04T10:05:00Z",
  "userImageUrl": "https://example.com/user-image.jpg",
  "clothImageUrl": "https://example.com/cloth-image.jpg",
  "resultImageUrl": "https://example.com/result-image.jpg"
}
```

**نکات:**
- این endpoint همیشه منتظر می‌ماند تا کانورژن کامل شود و نتیجه کامل را برمی‌گرداند
- نیازی به استفاده از endpoint `GET /api/conversion/{id}` نیست
- در صورت خطا، `status` برابر `failed` و `errorMessage` شامل پیام خطا است
- فیلدهای `userImageUrl`, `clothImageUrl`, `resultImageUrl` در صورت وجود URL تصویر نمایش داده می‌شوند
- `status` می‌تواند یکی از مقادیر زیر باشد: `pending`, `processing`, `completed`, `failed`

**حالت Mock (برای تست):**
برای تست بدون استفاده از سرویس AI واقعی، می‌توانید از query parameter `mock=true` استفاده کنید:

```
POST /api/convert?mock=true
```

در این حالت، endpoint فوراً یک پاسخ mock موفق برمی‌گرداند بدون اینکه واقعاً کانورژن را پردازش کند. این برای تست و توسعه مفید است.

---

### Get Quota Status
```
GET /api/convert/quota
Headers: Authorization: Bearer {access_token}
```

**Response:**
```json
{
  "freeConversionsRemaining": 2,
  "paidConversionsRemaining": 0,
  "totalConversionsRemaining": 2,
  "planName": "free"
}
```

---

### Get Conversion Metrics
```
GET /api/convert/metrics
Headers: Authorization: Bearer {access_token}
```

---

### List Conversions
```
GET /api/conversions?page=1&pageSize=20&status=completed
Headers: Authorization: Bearer {access_token}
```

**Query Parameters:**
- `page` (optional, default: 1)
- `pageSize` (optional, default: 20, max: 100)
- `status` (optional): pending, processing, completed, failed, cancelled

**Response:**
```json
{
  "conversions": [
    {
      "id": "conversion-uuid",
      "userId": "user-uuid",
      "userImageId": "image-uuid",
      "clothImageId": "cloth-uuid",
      "status": "completed",
      "resultImageId": "result-image-uuid",
      "errorMessage": null,
      "processingTimeMs": 5000,
      "createdAt": "2025-11-04T10:00:00Z",
      "updatedAt": "2025-11-04T10:05:00Z",
      "completedAt": "2025-11-04T10:05:00Z",
      "userImageUrl": "https://example.com/user-image.jpg",
      "clothImageUrl": "https://example.com/cloth-image.jpg",
      "resultImageUrl": "https://example.com/result-image.jpg"
    }
  ],
  "total": 100,
  "page": 1,
  "pageSize": 20,
  "totalPages": 5
}
```

---

### Get Conversion
```
GET /api/conversion/:id
Headers: Authorization: Bearer {access_token}
```

**Response:**
```json
{
  "id": "conversion-uuid",
  "userId": "user-uuid",
  "userImageId": "image-uuid",
  "clothImageId": "cloth-uuid",
  "status": "completed",
  "resultImageId": "result-image-uuid",
  "errorMessage": null,
  "processingTimeMs": 5000,
  "createdAt": "2025-11-04T10:00:00Z",
  "updatedAt": "2025-11-04T10:05:00Z",
  "completedAt": "2025-11-04T10:05:00Z",
  "userImageUrl": "https://example.com/user-image.jpg",
  "clothImageUrl": "https://example.com/cloth-image.jpg",
  "resultImageUrl": "https://example.com/result-image.jpg"
}
```

---

### Update Conversion
```
PUT /api/conversion/:id
Headers: Authorization: Bearer {access_token}
```

**Request Body:**
```json
{
  "status": "completed",
  "resultImageId": "uuid-here"
}
```

---

### Delete Conversion
```
DELETE /api/conversion/:id
Headers: Authorization: Bearer {access_token}
```

---

### Cancel Conversion
```
POST /api/conversion/:id/cancel
Headers: Authorization: Bearer {access_token}
```

---

### Get Conversion Status
```
GET /api/conversion/:id/status
Headers: Authorization: Bearer {access_token}
```

---

## Images

### Upload Image
```
POST /api/images
Headers: Authorization: Bearer {access_token}
Content-Type: multipart/form-data
```

**Form Data:**
- `file` (file): تصویر
- `type` (text): user, vendor, result

---

### List Images
```
GET /api/images?page=1&pageSize=20&type=user
Headers: Authorization: Bearer {access_token}
```

**Query Parameters:**
- `page` (optional)
- `pageSize` (optional)
- `type` (optional): user, vendor, result

---

### Get Image
```
GET /api/images/:id
Headers: Authorization: Bearer {access_token}
```

---

### Update Image
```
PUT /api/images/:id
Headers: Authorization: Bearer {access_token}
```

**Request Body:**
```json
{
  "tags": ["tag1", "tag2"],
  "isPublic": false
}
```

---

### Delete Image
```
DELETE /api/images/:id
Headers: Authorization: Bearer {access_token}
```

---

### Generate Signed URL
```
POST /api/images/:id/signed-url
Headers: Authorization: Bearer {access_token}
```

**Request Body:**
```json
{
  "expiresIn": 3600
}
```

---

### Get Image Usage History
```
GET /api/images/:id/usage
Headers: Authorization: Bearer {access_token}
```

---

### Get Quota Status
```
GET /api/quota
Headers: Authorization: Bearer {access_token}
```

---

### Get Image Stats
```
GET /api/stats
Headers: Authorization: Bearer {access_token}
```

---

## Vendors

### Get Vendors
```
GET /api/vendors
Headers: Authorization: Bearer {access_token}
```

---

### Get Vendor
```
GET /api/vendors/:id
Headers: Authorization: Bearer {access_token}
```

---

### Create Vendor
```
POST /api/vendors
Headers: Authorization: Bearer {access_token}
```

**Request Body:**
```json
{
  "displayName": "Vendor Name",
  "companyName": "Company Name",
  "status": "active"
}
```

---

### Update Vendor
```
PUT /api/vendors/:id
Headers: Authorization: Bearer {access_token}
```

---

### Delete Vendor
```
DELETE /api/vendors/:id
Headers: Authorization: Bearer {access_token}
```

---

## Payment

### Create Payment
```
POST /api/payments/create
Headers: Authorization: Bearer {access_token}
```

**Request Body:**
```json
{
  "planId": "plan-uuid",
  "amountCents": 99900,
  "currency": "IRR"
}
```

---

### Get Payment Status
```
GET /api/payments/:id/status
Headers: Authorization: Bearer {access_token}
```

---

### Get Payment History
```
GET /api/payments/history?page=1&pageSize=20
Headers: Authorization: Bearer {access_token}
```

---

### Cancel Payment
```
DELETE /api/payments/:id/cancel
Headers: Authorization: Bearer {access_token}
```

---

### Get Plans
```
GET /api/plans/
```

---

### Get User Active Plan
```
GET /api/plans/active
Headers: Authorization: Bearer {access_token}
```

---

## Share

### Create Shared Link
```
POST /api/share/create
Headers: Authorization: Bearer {access_token}
```

**Request Body:**
```json
{
  "conversionId": "conversion-uuid",
  "expiresIn": 300,
  "maxAccessCount": 10
}
```

---

### Access Shared Link
```
GET /api/share/:token
```

---

### Deactivate Shared Link
```
DELETE /api/share/:id
Headers: Authorization: Bearer {access_token}
```

---

### List User Shared Links
```
GET /api/share/
Headers: Authorization: Bearer {access_token}
```

---

## Notifications

### Create Notification
```
POST /api/notifications
Headers: Authorization: Bearer {access_token}
```

**Request Body:**
```json
{
  "type": "info",
  "title": "Notification Title",
  "message": "Notification message",
  "channel": "in_app"
}
```

---

### List Notifications
```
GET /api/notifications?page=1&pageSize=20&unread_only=true
Headers: Authorization: Bearer {access_token}
```

---

### Get Notification
```
GET /api/notifications/:id
Headers: Authorization: Bearer {access_token}
```

---

### Mark Notification as Read
```
PUT /api/notifications/:id/read
Headers: Authorization: Bearer {access_token}
```

---

### Delete Notification
```
DELETE /api/notifications/:id
Headers: Authorization: Bearer {access_token}
```

---

### Get Notification Preferences
```
GET /api/notifications/preferences
Headers: Authorization: Bearer {access_token}
```

---

### Update Notification Preferences
```
PUT /api/notifications/preferences
Headers: Authorization: Bearer {access_token}
```

**Request Body:**
```json
{
  "emailEnabled": true,
  "smsEnabled": false,
  "pushEnabled": true
}
```

---

### Get Notification Stats
```
GET /api/notifications/stats
Headers: Authorization: Bearer {access_token}
```

---

## Admin

**Note:** تمام endpoints زیر نیاز به Admin role دارند.

### Users

- `GET /api/admin/users` - Get all users
- `GET /api/admin/users/:id` - Get user
- `PUT /api/admin/users/:id` - Update user
- `DELETE /api/admin/users/:id` - Delete user
- `POST /api/admin/users/:id/suspend` - Suspend user
- `POST /api/admin/users/:id/activate` - Activate user
- `POST /api/admin/users/:id/revoke-quota` - Revoke user quota
- `POST /api/admin/users/:id/revoke-plan` - Revoke user plan

### Vendors

- `GET /api/admin/vendors` - Get all vendors
- `GET /api/admin/vendors/:id` - Get vendor
- `PUT /api/admin/vendors/:id` - Update vendor
- `DELETE /api/admin/vendors/:id` - Delete vendor
- `POST /api/admin/vendors/:id/suspend` - Suspend vendor
- `POST /api/admin/vendors/:id/activate` - Activate vendor
- `POST /api/admin/vendors/:id/verify` - Verify vendor
- `POST /api/admin/vendors/:id/revoke-quota` - Revoke vendor quota

### Plans

- `GET /api/admin/plans` - Get all plans
- `GET /api/admin/plans/:id` - Get plan
- `POST /api/admin/plans` - Create plan
- `PUT /api/admin/plans/:id` - Update plan
- `DELETE /api/admin/plans/:id` - Delete plan

### Statistics

- `GET /api/admin/stats` - Get system stats
- `GET /api/admin/stats/users` - Get user stats
- `GET /api/admin/stats/vendors` - Get vendor stats
- `GET /api/admin/stats/payments` - Get payment stats
- `GET /api/admin/stats/conversions` - Get conversion stats
- `GET /api/admin/stats/images` - Get image stats

---

## Health

### Health Check
```
GET /api/health/
```

### Readiness Check
```
GET /api/health/ready
```

### Liveness Check
```
GET /api/health/live
```

### System Info
```
GET /api/health/system
```

### Metrics
```
GET /api/health/metrics
```

---

## Error Responses

تمام endpoints در صورت بروز خطا، پاسخ زیر را برمی‌گردانند:

```json
{
  "error": {
    "code": "error_code",
    "message": "Error message",
    "details": {}
  }
}
```

### کدهای خطای رایج:

- `400` - Bad Request
- `401` - Unauthorized
- `403` - Forbidden
- `404` - Not Found
- `429` - Too Many Requests
- `500` - Internal Server Error

---

## Authentication

بیشتر endpoints نیاز به Authentication دارند. برای استفاده:

1. ابتدا با `/auth/login` یا `/auth/register` login کنید
2. `accessToken` را از response دریافت کنید
3. در header همه requestها اضافه کنید:
   ```
   Authorization: Bearer {access_token}
   ```

---

## Rate Limiting

برخی endpoints دارای rate limiting هستند:
- Send OTP: 3 requests per hour per phone
- Create Conversion: 10 requests per hour per user

---

## Notes

- تمام UUID ها باید به صورت معتبر ارسال شوند
- تمام تاریخ‌ها به صورت ISO 8601 (RFC3339) هستند
- Phone numbers باید با فرمت بین‌المللی ارسال شوند (مثال: `+989123456789`)
- File uploads باید از نوع `multipart/form-data` باشند

