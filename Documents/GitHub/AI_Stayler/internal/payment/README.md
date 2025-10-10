# Payment Service

The Payment Service handles subscription plans, payment processing, and quota management for the AI Stayler application.

## Features

- **Payment Plans**: Free, Basic, and Advanced subscription plans
- **Zarinpal Integration**: Complete integration with Zarinpal payment gateway
- **Payment Processing**: Create, verify, and track payments
- **Webhook Support**: Handle payment notifications from Zarinpal
- **Quota Management**: Automatic quota updates based on plan activation
- **User Notifications**: Send payment success/failure notifications

## API Endpoints

### Payment Operations
- `POST /api/payments/create` - Create a new payment
- `GET /api/payments/:id/status` - Get payment status
- `GET /api/payments/history` - Get payment history
- `DELETE /api/payments/:id/cancel` - Cancel a payment

### Plan Operations
- `GET /api/plans/` - Get all available plans
- `GET /api/plans/active` - Get user's active plan

### Webhook Operations
- `POST /api/payments/webhooks/notify` - Handle payment webhooks

## Configuration

Set the following environment variables:

```bash
# Zarinpal Configuration
ZARINPAL_MERCHANT_ID=your_merchant_id
ZARINPAL_BASE_URL=https://gateway.zibal.ir

# Payment URLs
PAYMENT_CALLBACK_URL=https://your-domain.com/api/payments/webhooks/notify?gateway=zarinpal
PAYMENT_RETURN_URL=https://your-domain.com/payment/success

# Payment Settings
PAYMENT_EXPIRY_MINUTES=30
```

## Payment Plans

### Free Plan
- **Price**: 0 Rials
- **Conversions**: 2 per month
- **Features**: Basic support

### Basic Plan
- **Price**: 50,000 Rials/month
- **Conversions**: 20 per month
- **Features**: Email support, Priority processing

### Advanced Plan
- **Price**: 150,000 Rials/month
- **Conversions**: 100 per month
- **Features**: Priority support, Fast processing, Advanced features

## Payment Flow

1. **Create Payment**: User selects a plan and creates a payment
2. **Gateway Redirect**: User is redirected to Zarinpal payment page
3. **Payment Processing**: User completes payment on Zarinpal
4. **Webhook Notification**: Zarinpal sends webhook to our service
5. **Payment Verification**: Service verifies payment with Zarinpal
6. **Plan Activation**: User's plan is activated and quota is updated
7. **Notification**: User receives success notification

## Database Schema

### Tables
- `payment_plans` - Available subscription plans
- `payments` - Payment transactions
- `payment_history` - Payment status changes
- `user_plans` - User subscription plans (extends existing)

### Key Functions
- `create_payment()` - Create a new payment
- `update_payment_status()` - Update payment status
- `get_user_payment_summary()` - Get user payment summary
- `cleanup_expired_payments()` - Clean up expired payments

## Error Handling

The service handles various error scenarios:
- Invalid plan selection
- Payment gateway errors
- Webhook verification failures
- Quota update failures
- Rate limiting

## Security

- All payment operations require user authentication
- Webhook endpoints validate gateway signatures
- Payment amounts are validated against plan prices
- Rate limiting prevents abuse

## Testing

Run tests with:
```bash
go test ./internal/payment/...
```

## Dependencies

- Database (PostgreSQL)
- Zarinpal Gateway
- User Service
- Notification Service
- Quota Service
- Audit Logger
- Rate Limiter
