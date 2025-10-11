# Share Service

The Share Service handles public sharing of conversion results with signed URLs and token-based access control.

## Features

- **Public Sharing**: Users can share conversion results with temporary public links
- **Signed URLs**: Secure, time-limited access to result images
- **Token-based Access**: Unique tokens for each shared link
- **Expiry Control**: Links expire after 1-5 minutes (configurable)
- **Access Tracking**: Monitor link usage and access patterns
- **Rate Limiting**: Prevent abuse with access count limits
- **Audit Logging**: Log all sharing activities
- **Metrics Collection**: Track sharing performance and usage

## API Endpoints

### Share Operations

- `POST /share/create` - Create a new shared link for a conversion result
- `GET /share/{token}` - Access a shared link (public endpoint)
- `DELETE /share/{id}` - Deactivate a shared link
- `GET /share/` - List user's shared links
- `GET /share/stats` - Get sharing statistics
- `POST /share/cleanup` - Cleanup expired links (admin)

## Models

### CreateShareRequest
```json
{
  "conversionId": "uuid",
  "expiryMinutes": 5,
  "maxAccessCount": 10
}
```

### CreateShareResponse
```json
{
  "shareId": "uuid",
  "shareToken": "base64url-token",
  "signedUrl": "signed-url",
  "expiresAt": "2023-01-01T00:05:00Z",
  "publicUrl": "/api/share/token"
}
```

### AccessShareResponse
```json
{
  "success": true,
  "conversionId": "uuid",
  "resultImageUrl": "image-url",
  "accessCount": 5,
  "secondsUntilExpiry": 180
}
```

### SharedLinkStats
```json
{
  "totalLinks": 10,
  "activeLinks": 3,
  "expiredLinks": 7,
  "totalAccessCount": 25,
  "uniqueIpAddresses": 8
}
```

## Database Schema

### shared_links table
- `id` - UUID primary key
- `conversion_id` - Foreign key to conversions table
- `user_id` - Foreign key to users table
- `share_token` - Unique token for access
- `signed_url` - Pre-signed URL for direct access
- `expires_at` - Link expiration time (1-5 minutes)
- `access_count` - Number of times accessed
- `max_access_count` - Optional access limit
- `is_active` - Whether link is active
- `created_at` - Creation timestamp
- `updated_at` - Last update timestamp

### shared_link_access_logs table
- `id` - UUID primary key
- `shared_link_id` - Foreign key to shared_links table
- `ip_address` - Client IP address
- `user_agent` - Client user agent
- `referer` - HTTP referer header
- `access_type` - Type of access (view/download)
- `success` - Whether access was successful
- `error_message` - Error message if failed
- `metadata` - Additional metadata
- `created_at` - Access timestamp

## Security Features

- **Time-limited Access**: Links expire after 1-5 minutes
- **Cryptographically Secure Tokens**: Random tokens generated using crypto/rand
- **Access Tracking**: Monitor all access attempts
- **Rate Limiting**: Optional access count limits
- **Audit Logging**: Complete audit trail of sharing activities
- **IP Tracking**: Track access by IP address for analytics

## Usage Examples

### Create a Shared Link
```bash
curl -X POST /api/share/create \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "conversionId": "uuid",
    "expiryMinutes": 5,
    "maxAccessCount": 10
  }'
```

### Access a Shared Link
```bash
curl -X GET /api/share/{token}?type=view
```

### List User's Shared Links
```bash
curl -X GET /api/share/?limit=20&offset=0 \
  -H "Authorization: Bearer <token>"
```

## Configuration

- **MinExpiryMinutes**: 1 minute minimum
- **MaxExpiryMinutes**: 5 minutes maximum
- **DefaultExpiryMinutes**: 5 minutes default
- **ShareTokenLength**: 32 bytes (base64url encoded)
- **CleanupInterval**: Expired links kept for 1 hour for analytics

## Error Handling

- Invalid conversion ID or ownership
- Conversion not completed
- Expired or inactive links
- Access count limits exceeded
- Invalid share tokens
- Database connection issues

## Monitoring

- Share creation metrics
- Access success/failure rates
- Link expiry patterns
- User sharing behavior
- Storage usage for access logs
