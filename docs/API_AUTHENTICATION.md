# Authentication API Documentation

## 📋 Table of Contents

1. [Overview](#overview)
2. [Authentication Flow](#authentication-flow)
3. [Technical Architecture](#technical-architecture)
4. [Endpoints](#endpoints)
5. [Error Codes](#error-codes)
6. [Rate Limiting](#rate-limiting)
7. [Security Features](#security-features)
8. [Examples](#examples)

---

## Overview

The Authentication API provides secure user authentication and authorization for the AI Styler platform. It supports phone-based authentication with OTP (One-Time Password) verification, password-based login, and token-based session management.

### Key Features

- **Phone-based OTP Verification**: Secure SMS-based verification
- **Multi-role Support**: Users can be `user`, `vendor`, or `admin`
- **JWT Token Management**: Access and refresh token system
- **Session Management**: Multi-device session support with individual logout
- **Rate Limiting**: Protection against brute force attacks
- **Password Security**: Argon2id or BCrypt hashing

### Base URL

```
Production: https://api.example.com
Development: http://localhost:8080
```

### Content Type

All requests and responses use `application/json`.

---

## Authentication Flow

### Registration Flow

```
1. User requests OTP → POST /auth/send-otp
2. User receives SMS with OTP code
3. User verifies OTP → POST /auth/verify-otp
4. Phone number is marked as verified
5. User registers → POST /auth/register
6. (Optional) Auto-login returns access/refresh tokens
```

### Login Flow

```
1. User provides phone and password → POST /auth/login
2. System validates credentials
3. System checks phone verification status
4. System issues access and refresh tokens
5. User can access protected endpoints
```

### Token Refresh Flow

```
1. Access token expires (15 minutes)
2. User sends refresh token → POST /auth/refresh
3. System validates refresh token
4. System revokes old session
5. System issues new access and refresh tokens
6. User continues using application
```

### Logout Flow

```
1. User requests logout → POST /auth/logout (single session)
                    OR → POST /auth/logout-all (all sessions)
2. System revokes session(s) in database
3. Tokens become invalid immediately
4. User must re-authenticate
```

---

## Technical Architecture

### Password Hashing

The system supports two password hashing algorithms:

#### Argon2id (Recommended)
- **Memory**: 65536 KB (64 MB)
- **Iterations**: 3
- **Parallelism**: 2
- **Salt Length**: 16 bytes
- **Key Length**: 32 bytes

#### BCrypt (Fallback)
- **Cost Factor**: 12

**Configuration**: Controlled via environment variables in `config.SecurityConfig`.

### Token Structure

#### Access Token
- **Type**: JWT (JSON Web Token)
- **Lifetime**: 15 minutes (900 seconds)
- **Claims**:
  - `user_id`: UUID of the user
  - `role`: User role (user/vendor/admin)
  - `session_id`: Unique session identifier
  - `exp`: Expiration timestamp
  - `iat`: Issued at timestamp

#### Refresh Token
- **Type**: JWT (JSON Web Token)
- **Lifetime**: 7 days (configurable)
- **Purpose**: Rotate access tokens without re-authentication
- **Storage**: Hashed in database using BCrypt

### Session Management

- Sessions are stored in PostgreSQL `sessions` table
- Each session has:
  - Unique session ID (UUID)
  - User ID reference
  - Hashed refresh token
  - User agent and IP address
  - Expiration timestamp
  - Revocation timestamp

### OTP Generation

- **Length**: 6 digits
- **Lifetime**: 5 minutes
- **Storage**: Hashed in database using BCrypt
- **Purpose**: `phone_verify` (for phone verification)
- **Rate Limits**: 
  - 3 per phone per hour
  - 100 per IP per 24 hours

---

## Endpoints

### 1. Send OTP

Send a one-time password to the user's phone number for verification.

#### Request

```http
POST /auth/send-otp
Content-Type: application/json
```

**Request Body:**

```json
{
  "phone": "+989123456789",
  "purpose": "phone_verify",
  "channel": "sms"
}
```

**Parameters:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `phone` | string | Yes | Phone number in E.164 format (must start with +) |
| `purpose` | string | No | Purpose of OTP (default: "phone_verify") |
| `channel` | string | No | Delivery channel (default: "sms") |

**Phone Number Format:**
- Must start with `+` (e.g., `+989123456789`)
- Must be valid E.164 format
- Spaces and dashes are automatically removed

#### Response

**Success (200 OK):**

```json
{
  "sent": true,
  "expiresInSec": 300,
  "code": "123456",
  "debug": true
}
```

**Note:** `code` and `debug` fields are only returned in development/mock mode.

**Response Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `sent` | boolean | Whether OTP was sent successfully |
| `expiresInSec` | integer | OTP expiration time in seconds (300 = 5 minutes) |
| `code` | string | OTP code (only in mock/dev mode) |
| `debug` | boolean | Indicates debug/mock mode |

**Error Responses:**

| Status | Code | Message | Description |
|--------|------|---------|-------------|
| 400 | `bad_request` | `invalid phone` | Invalid phone number format |
| 429 | `rate_limited` | `too many requests` | Rate limit exceeded |
| 500 | `server_error` | `could not create otp` | Internal server error |

#### Rate Limiting

- **Per Phone**: 3 requests per hour
- **Per IP**: 100 requests per 24 hours

#### Technical Details

1. Phone number is normalized (spaces removed, validated)
2. Rate limits are checked (phone and IP)
3. 6-digit OTP code is generated
4. OTP is hashed using BCrypt and stored in database
5. OTP is sent via SMS provider (SMS.ir or mock)
6. Expiration time is set to 5 minutes

---

### 2. Verify OTP

Verify the OTP code sent to the user's phone.

#### Request

```http
POST /auth/verify-otp
Content-Type: application/json
```

**Request Body:**

```json
{
  "phone": "+989123456789",
  "code": "123456",
  "purpose": "phone_verify"
}
```

**Parameters:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `phone` | string | Yes | Phone number in E.164 format |
| `code` | string | Yes | 6-digit OTP code |
| `purpose` | string | No | Purpose of OTP (default: "phone_verify") |

#### Response

**Success (200 OK):**

```json
{
  "verified": true
}
```

**Response Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `verified` | boolean | Whether OTP was verified successfully |

**Error Responses:**

| Status | Code | Message | Description |
|--------|------|---------|-------------|
| 400 | `bad_request` | `invalid input` | Invalid phone or code format |
| 400 | `invalid_otp` | `invalid or expired otp` | OTP is incorrect or expired |
| 429 | `rate_limited` | `too many requests` | Too many verification attempts |
| 500 | `server_error` | `verification failed` | Internal server error |

#### Technical Details

1. Phone number is normalized
2. OTP code must be exactly 6 digits
3. OTP is looked up in database (hashed comparison)
4. OTP expiration is checked
5. OTP consumption is recorded (can only be used once)
6. Phone number is marked as verified in database
7. Failed attempts are tracked for security

**OTP Validation Rules:**
- OTP must match exactly (case-sensitive)
- OTP must not be expired (5 minutes)
- OTP must not be previously consumed
- Maximum 5 attempts per OTP

---

### 3. Register

Register a new user account. Phone number must be verified first.

#### Request

```http
POST /auth/register
Content-Type: application/json
```

**Request Body:**

```json
{
  "phone": "+989123456789",
  "password": "SecurePassword123!",
  "role": "user",
  "autoLogin": true,
  "displayName": "John Doe",
  "companyName": "Company Inc"
}
```

**Parameters:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `phone` | string | Yes | Verified phone number in E.164 format |
| `password` | string | Yes | Password (minimum 10 characters) |
| `role` | string | Yes | User role: `user` or `vendor` |
| `autoLogin` | boolean | No | Automatically login after registration (default: false) |
| `displayName` | string | No | Display name for the user |
| `companyName` | string | No | Company name (required for vendors) |

**Password Requirements:**
- Minimum 10 characters
- Recommended: Mix of uppercase, lowercase, numbers, and special characters

**Role Types:**
- `user`: Regular user account
- `vendor`: Vendor account (requires additional vendor profile setup)

#### Response

**Success (201 Created):**

```json
{
  "userId": "550e8400-e29b-41d4-a716-446655440000",
  "role": "user",
  "isPhoneVerified": true,
  "accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "accessTokenExpiresIn": 900,
  "refreshToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refreshTokenExpiresAt": "2025-11-11T12:32:04Z"
}
```

**Response Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `userId` | string (UUID) | Unique user identifier |
| `role` | string | User role |
| `isPhoneVerified` | boolean | Phone verification status (always true) |
| `accessToken` | string | JWT access token (only if autoLogin=true) |
| `accessTokenExpiresIn` | integer | Access token lifetime in seconds (900 = 15 min) |
| `refreshToken` | string | JWT refresh token (only if autoLogin=true) |
| `refreshTokenExpiresAt` | string (ISO 8601) | Refresh token expiration timestamp |

**Note:** `accessToken`, `accessTokenExpiresIn`, `refreshToken`, and `refreshTokenExpiresAt` are only included if `autoLogin` is `true`.

**Error Responses:**

| Status | Code | Message | Description |
|--------|------|---------|-------------|
| 400 | `bad_request` | `invalid input` | Invalid phone, password, or role |
| 403 | `unverified` | `phone not verified` | Phone number not verified via OTP |
| 409 | `conflict` | `account exists` | User already exists with this phone |
| 500 | `server_error` | `could not hash password` | Password hashing failed |
| 500 | `server_error` | `could not create user` | User creation failed |

#### Technical Details

1. Phone number is normalized and validated
2. Password length is validated (minimum 10 characters)
3. Role is validated (must be `user` or `vendor`)
4. System checks if user already exists
5. System verifies phone number was verified via OTP
6. Password is hashed using Argon2id or BCrypt
7. User record is created in `users` table
8. If role is `vendor`, vendor record is created in `vendors` table
9. Phone verification status is set to `true`
10. If `autoLogin=true`, tokens are issued and returned

**Database Actions:**
- Insert into `users` table with:
  - `phone` (unique)
  - `password_hash` (hashed)
  - `role`
  - `is_phone_verified = true`
  - `is_active = true`
- If vendor: Insert into `vendors` table with:
  - `user_id` (foreign key)
  - `display_name`
  - `company_name`

---

### 4. Login

Authenticate user with phone number and password.

#### Request

```http
POST /auth/login
Content-Type: application/json
```

**Request Body:**

```json
{
  "phone": "+989123456789",
  "password": "SecurePassword123!"
}
```

**Parameters:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `phone` | string | Yes | Phone number in E.164 format |
| `password` | string | Yes | User password |

#### Response

**Success (200 OK):**

```json
{
  "accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "accessTokenExpiresIn": 900,
  "refreshToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refreshTokenExpiresAt": "2025-11-11T12:32:04Z",
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "role": "user",
    "isPhoneVerified": true
  }
}
```

**Response Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `accessToken` | string | JWT access token |
| `accessTokenExpiresIn` | integer | Access token lifetime in seconds (900 = 15 min) |
| `refreshToken` | string | JWT refresh token |
| `refreshTokenExpiresAt` | string (ISO 8601) | Refresh token expiration timestamp |
| `user.id` | string (UUID) | User identifier |
| `user.role` | string | User role |
| `user.isPhoneVerified` | boolean | Phone verification status |

**Error Responses:**

| Status | Code | Message | Description |
|--------|------|---------|-------------|
| 400 | `bad_request` | `invalid input` | Invalid phone or password format |
| 401 | `unauthorized` | `invalid credentials` | Phone or password is incorrect |
| 403 | `forbidden` | `phone not verified` | Phone number not verified |
| 500 | `server_error` | `could not issue tokens` | Token generation failed |

#### Technical Details

1. Phone number is normalized
2. User is looked up by phone number
3. Password is verified against stored hash (Argon2id or BCrypt)
4. Phone verification status is checked
5. User active status is checked
6. New session is created in database
7. Access and refresh tokens are generated (JWT)
8. Refresh token is hashed and stored in session
9. User agent and IP address are recorded
10. Tokens are returned to client

**Security Checks:**
- User must exist
- Password must match (constant-time comparison)
- Phone must be verified
- User must be active (`is_active = true`)

**Session Creation:**
- New session record in `sessions` table
- Unique session ID (UUID)
- Refresh token hash stored
- Expiration set to 7 days (configurable)
- User agent and IP recorded

---

### 5. Refresh Token

Rotate access and refresh tokens without re-authentication.

#### Request

```http
POST /auth/refresh
Content-Type: application/json
```

**Request Body:**

```json
{
  "refreshToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Parameters:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `refreshToken` | string | Yes | Valid refresh token JWT |

#### Response

**Success (200 OK):**

```json
{
  "accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "accessTokenExpiresIn": 900,
  "refreshToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refreshTokenExpiresAt": "2025-11-11T12:32:04Z"
}
```

**Response Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `accessToken` | string | New JWT access token |
| `accessTokenExpiresIn` | integer | Access token lifetime in seconds (900 = 15 min) |
| `refreshToken` | string | New JWT refresh token |
| `refreshTokenExpiresAt` | string (ISO 8601) | Refresh token expiration timestamp |

**Error Responses:**

| Status | Code | Message | Description |
|--------|------|---------|-------------|
| 400 | `bad_request` | `invalid input` | Missing or invalid refresh token |
| 401 | `unauthorized` | `invalid refresh` | Refresh token is invalid, expired, or revoked |

#### Technical Details

1. Refresh token is validated (JWT signature and expiration)
2. Session is looked up by session ID from token
3. Session revocation status is checked
4. Old session is revoked (marked as revoked)
5. New session is created
6. New access and refresh tokens are generated
7. New refresh token is hashed and stored
8. New tokens are returned to client

**Token Rotation Security:**
- Old session is immediately revoked
- New session is created with new ID
- Refresh token is rotated (cannot reuse old token)
- Prevents token replay attacks

**When to Use:**
- Access token has expired (after 15 minutes)
- Access token is about to expire
- Security rotation (optional periodic refresh)

---

### 6. Logout

Logout from current session only.

#### Request

```http
POST /auth/logout
Content-Type: application/json
Authorization: Bearer <access_token>
```

**Headers:**

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| `Authorization` | string | Yes | Bearer token with access token JWT |

**Request Body:**

None (empty body)

#### Response

**Success (200 OK):**

```json
{
  "success": true
}
```

**Response Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `success` | boolean | Logout success status |

**Error Responses:**

| Status | Code | Message | Description |
|--------|------|---------|-------------|
| 401 | `unauthorized` | `missing token` | Authorization header missing |
| 401 | `unauthorized` | `invalid token` | Access token is invalid or expired |

#### Technical Details

1. Access token is extracted from Authorization header
2. Token is validated (JWT signature and expiration)
3. Session ID is extracted from token claims
4. Session is revoked in database (`revoked_at` is set)
5. Success response is returned

**What Happens:**
- Current session is marked as revoked
- Access token becomes invalid immediately
- Refresh token for this session becomes invalid
- Other sessions remain active
- User must re-authenticate on this device

---

### 7. Logout All

Logout from all sessions (all devices).

#### Request

```http
POST /auth/logout-all
Content-Type: application/json
Authorization: Bearer <access_token>
```

**Headers:**

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| `Authorization` | string | Yes | Bearer token with access token JWT |

**Request Body:**

None (empty body)

#### Response

**Success (200 OK):**

```json
{
  "success": true
}
```

**Response Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `success` | boolean | Logout success status |

**Error Responses:**

| Status | Code | Message | Description |
|--------|------|---------|-------------|
| 401 | `unauthorized` | `missing token` | Authorization header missing |
| 401 | `unauthorized` | `invalid token` | Access token is invalid or expired |

#### Technical Details

1. Access token is extracted from Authorization header
2. Token is validated (JWT signature and expiration)
3. User ID is extracted from token claims
4. All active sessions for this user are revoked (`revoked_at` is set)
5. Success response is returned

**What Happens:**
- All sessions for the user are marked as revoked
- All access tokens become invalid immediately
- All refresh tokens become invalid
- User is logged out from all devices (mobile, web, API clients)
- User must re-authenticate on all devices

**Use Cases:**
- Security breach suspected
- Password change
- Account security enhancement
- User wants to disconnect all devices

---

## Error Codes

### Standard Error Response Format

All error responses follow this structure:

```json
{
  "error": {
    "code": "error_code",
    "message": "Human-readable error message",
    "details": {}
  }
}
```

### Error Code Reference

| HTTP Status | Error Code | Description | Common Causes |
|-------------|-----------|-------------|---------------|
| 400 | `bad_request` | Invalid request format or parameters | Missing required fields, invalid format |
| 401 | `unauthorized` | Authentication required or failed | Missing/invalid token, wrong credentials |
| 403 | `forbidden` | Access denied | Phone not verified, account inactive |
| 409 | `conflict` | Resource conflict | User already exists |
| 429 | `rate_limited` | Too many requests | Rate limit exceeded |
| 500 | `server_error` | Internal server error | Database error, hashing failure |

### Error Handling Best Practices

1. **Always check HTTP status code first**
2. **Parse error response for specific error code**
3. **Handle rate limiting with exponential backoff**
4. **Log errors for debugging**
5. **Show user-friendly messages based on error code**

---

## Rate Limiting

### OTP Endpoints

**Send OTP:**
- **Per Phone**: 3 requests per hour
- **Per IP**: 100 requests per 24 hours

**Verify OTP:**
- **Per Phone**: 5 attempts per OTP code
- **Per IP**: 20 attempts per hour

### Login Endpoints

**Login:**
- **Per Phone**: 5 attempts per 15 minutes
- **Per IP**: 20 attempts per 15 minutes

### Rate Limit Headers

When rate limited, the response includes:

```http
HTTP/1.1 429 Too Many Requests
Retry-After: 3600
X-RateLimit-Limit: 3
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1636142400
```

### Rate Limit Handling

1. Monitor `Retry-After` header for retry timing
2. Implement exponential backoff
3. Cache rate limit state locally
4. Show user-friendly messages

---

## Security Features

### Password Security

- **Hashing**: Argon2id (recommended) or BCrypt
- **Salt**: Random salt per password
- **Verification**: Constant-time comparison
- **Storage**: Never stored in plaintext

### Token Security

- **JWT Signing**: HMAC-SHA256 or RSA
- **Token Storage**: Refresh tokens hashed in database
- **Token Rotation**: Automatic on refresh
- **Token Expiration**: Short-lived access tokens (15 min)

### Session Security

- **Session Tracking**: Database-backed sessions
- **Revocation**: Immediate invalidation
- **Multi-Device**: Support for multiple concurrent sessions
- **Session Metadata**: User agent and IP tracking

### OTP Security

- **Hashing**: BCrypt hashing before storage
- **Single Use**: OTPs consumed after verification
- **Expiration**: 5-minute lifetime
- **Rate Limiting**: Prevents brute force attacks

### HTTPS Requirements

⚠️ **Important**: All authentication endpoints must be accessed over HTTPS in production to protect:
- Passwords
- Tokens
- Session data

---

## Examples

### Complete Registration Flow

```bash
# Step 1: Send OTP
curl -X POST http://localhost:8080/auth/send-otp \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "+989123456789",
    "purpose": "phone_verify",
    "channel": "sms"
  }'

# Response:
# {
#   "sent": true,
#   "expiresInSec": 300
# }

# Step 2: Verify OTP (use code from SMS)
curl -X POST http://localhost:8080/auth/verify-otp \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "+989123456789",
    "code": "123456",
    "purpose": "phone_verify"
  }'

# Response:
# {
#   "verified": true
# }

# Step 3: Register
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "+989123456789",
    "password": "SecurePassword123!",
    "role": "user",
    "autoLogin": true,
    "displayName": "John Doe"
  }'

# Response:
# {
#   "userId": "550e8400-e29b-41d4-a716-446655440000",
#   "role": "user",
#   "isPhoneVerified": true,
#   "accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
#   "accessTokenExpiresIn": 900,
#   "refreshToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
#   "refreshTokenExpiresAt": "2025-11-11T12:32:04Z"
# }
```

### Login and Token Refresh

```bash
# Login
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "+989123456789",
    "password": "SecurePassword123!"
  }'

# Response:
# {
#   "accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
#   "accessTokenExpiresIn": 900,
#   "refreshToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
#   "refreshTokenExpiresAt": "2025-11-11T12:32:04Z",
#   "user": {
#     "id": "550e8400-e29b-41d4-a716-446655440000",
#     "role": "user",
#     "isPhoneVerified": true
#   }
# }

# Use access token for protected endpoints
curl -X GET http://localhost:8080/api/user/profile \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."

# Refresh token when access token expires
curl -X POST http://localhost:8080/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refreshToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  }'

# Response:
# {
#   "accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
#   "accessTokenExpiresIn": 900,
#   "refreshToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
#   "refreshTokenExpiresAt": "2025-11-11T12:32:04Z"
# }
```

### Logout Examples

```bash
# Logout from current session
curl -X POST http://localhost:8080/auth/logout \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."

# Response:
# {
#   "success": true
# }

# Logout from all sessions
curl -X POST http://localhost:8080/auth/logout-all \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."

# Response:
# {
#   "success": true
# }
```

### Error Handling Example

```bash
# Invalid phone format
curl -X POST http://localhost:8080/auth/send-otp \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "989123456789"
  }'

# Response (400 Bad Request):
# {
#   "error": {
#     "code": "bad_request",
#     "message": "invalid phone"
#   }
# }

# Rate limit exceeded
curl -X POST http://localhost:8080/auth/send-otp \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "+989123456789"
  }'

# Response (429 Too Many Requests):
# {
#   "error": {
#     "code": "rate_limited",
#     "message": "too many requests"
#   }
# }
```

---

## Best Practices

### Client Implementation

1. **Store tokens securely**: Use secure storage (Keychain on iOS, Keystore on Android)
2. **Handle token expiration**: Implement automatic token refresh
3. **Handle rate limiting**: Show user-friendly messages and retry with backoff
4. **Validate phone numbers**: Validate format before sending to API
5. **Handle errors gracefully**: Parse error responses and show appropriate messages

### Security Recommendations

1. **Use HTTPS**: Always use HTTPS in production
2. **Token storage**: Never store tokens in localStorage or cookies (use secure storage)
3. **Token rotation**: Refresh tokens before expiration
4. **Logout on app close**: Consider logging out on app termination for security
5. **Monitor sessions**: Allow users to view and manage active sessions

### Performance Tips

1. **Cache tokens**: Store tokens in memory for quick access
2. **Batch requests**: Minimize API calls
3. **Retry logic**: Implement exponential backoff for failed requests
4. **Connection pooling**: Reuse HTTP connections

---

## Configuration

### Environment Variables

```bash
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=password
DB_NAME=styler

# JWT
JWT_SECRET=your-secret-key
JWT_ACCESS_TTL=15m
JWT_REFRESH_TTL=168h

# SMS Provider
SMS_PROVIDER=sms_ir
SMS_API_KEY=your-api-key
SMS_TEMPLATE_ID=723881
SMS_PARAMETER_NAME=Code

# Security
SECURITY_HASHER=argon2
SECURITY_ARGON2_MEMORY=65536
SECURITY_ARGON2_ITERATIONS=3
SECURITY_BCRYPT_COST=12

# Rate Limiting
RATE_LIMIT_OTP_PER_PHONE=3
RATE_LIMIT_OTP_PER_IP=100
RATE_LIMIT_LOGIN_PER_PHONE=5
RATE_LIMIT_LOGIN_PER_IP=20
```

---

## Support

For issues, questions, or feature requests, please contact:
- **Email**: support@example.com
- **Documentation**: https://docs.example.com
- **Status Page**: https://status.example.com

---

## Changelog

### Version 1.0.0 (2025-11-04)
- Initial release
- Phone-based OTP authentication
- JWT token management
- Multi-role support (user, vendor)
- Session management
- Rate limiting

---

**Last Updated**: 2025-11-04  
**API Version**: 1.0.0  
**Documentation Version**: 1.0.0

