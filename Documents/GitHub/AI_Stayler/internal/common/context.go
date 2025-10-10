package common

import (
	"context"
)

// Context keys
type contextKey string

const (
	UserIDKey   contextKey = "userID"
	VendorIDKey contextKey = "vendorID"
)

// SetUserIDInContext sets the user ID in the context
func SetUserIDInContext(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

// GetUserIDFromContext gets the user ID from the context
func GetUserIDFromContext(ctx context.Context) string {
	if userID, ok := ctx.Value(UserIDKey).(string); ok {
		return userID
	}
	return ""
}

// SetVendorIDInContext sets the vendor ID in the context
func SetVendorIDInContext(ctx context.Context, vendorID string) context.Context {
	return context.WithValue(ctx, VendorIDKey, vendorID)
}

// GetVendorIDFromContext gets the vendor ID from the context
func GetVendorIDFromContext(ctx context.Context) string {
	if vendorID, ok := ctx.Value(VendorIDKey).(string); ok {
		return vendorID
	}
	return ""
}

// WithVendorID is an alias for SetVendorIDInContext for backward compatibility
func WithVendorID(ctx context.Context, vendorID string) context.Context {
	return SetVendorIDInContext(ctx, vendorID)
}
