package docs

import (
	"encoding/json"
	"net/http"
	"time"
)

// APIDocumentation represents the complete API documentation
type APIDocumentation struct {
	OpenAPI    string             `json:"openapi"`
	Info       APIInfo            `json:"info"`
	Servers    []APIServer        `json:"servers"`
	Paths      map[string]APIPath `json:"paths"`
	Components APIComponents      `json:"components"`
	Tags       []APITag           `json:"tags"`
}

// APIInfo represents API information
type APIInfo struct {
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Version     string     `json:"version"`
	Contact     APIContact `json:"contact"`
	License     APILicense `json:"license"`
}

// APIContact represents API contact information
type APIContact struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	URL   string `json:"url"`
}

// APILicense represents API license information
type APILicense struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// APIServer represents API server information
type APIServer struct {
	URL         string `json:"url"`
	Description string `json:"description"`
}

// APIPath represents an API path
type APIPath struct {
	Get    *APIOperation `json:"get,omitempty"`
	Post   *APIOperation `json:"post,omitempty"`
	Put    *APIOperation `json:"put,omitempty"`
	Delete *APIOperation `json:"delete,omitempty"`
	Patch  *APIOperation `json:"patch,omitempty"`
}

// APIOperation represents an API operation
type APIOperation struct {
	Tags        []string               `json:"tags"`
	Summary     string                 `json:"summary"`
	Description string                 `json:"description"`
	OperationID string                 `json:"operationId"`
	Parameters  []APIParameter         `json:"parameters,omitempty"`
	RequestBody *APIRequestBody        `json:"requestBody,omitempty"`
	Responses   map[string]APIResponse `json:"responses"`
	Security    []map[string][]string  `json:"security,omitempty"`
}

// APIParameter represents an API parameter
type APIParameter struct {
	Name        string      `json:"name"`
	In          string      `json:"in"`
	Description string      `json:"description"`
	Required    bool        `json:"required"`
	Schema      *APISchema  `json:"schema"`
	Example     interface{} `json:"example,omitempty"`
}

// APIRequestBody represents an API request body
type APIRequestBody struct {
	Description string                `json:"description"`
	Content     map[string]APIContent `json:"content"`
	Required    bool                  `json:"required"`
}

// APIResponse represents an API response
type APIResponse struct {
	Description string                `json:"description"`
	Content     map[string]APIContent `json:"content,omitempty"`
}

// APIContent represents API content
type APIContent struct {
	Schema *APISchema `json:"schema"`
}

// APISchema represents an API schema
type APISchema struct {
	Type        string                `json:"type,omitempty"`
	Format      string                `json:"format,omitempty"`
	Description string                `json:"description,omitempty"`
	Ref         string                `json:"$ref,omitempty"`
	Properties  map[string]*APISchema `json:"properties,omitempty"`
	Items       *APISchema            `json:"items,omitempty"`
	Required    []string              `json:"required,omitempty"`
	Example     interface{}           `json:"example,omitempty"`
	Enum        []interface{}         `json:"enum,omitempty"`
	MinLength   int                   `json:"minLength,omitempty"`
	MaxLength   int                   `json:"maxLength,omitempty"`
	Minimum     float64               `json:"minimum,omitempty"`
	Maximum     float64               `json:"maximum,omitempty"`
	Pattern     string                `json:"pattern,omitempty"`
}

// APIComponents represents API components
type APIComponents struct {
	Schemas         map[string]*APISchema  `json:"schemas"`
	SecuritySchemes map[string]interface{} `json:"securitySchemes"`
}

// APITag represents an API tag
type APITag struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// GenerateAPIDocumentation generates comprehensive API documentation
func GenerateAPIDocumentation() *APIDocumentation {
	return &APIDocumentation{
		OpenAPI: "3.0.3",
		Info: APIInfo{
			Title:       "AI Styler API",
			Description: "AI-powered image styling and conversion service",
			Version:     "1.0.0",
			Contact: APIContact{
				Name:  "AI Styler Team",
				Email: "support@aistyler.com",
				URL:   "https://aistyler.com",
			},
			License: APILicense{
				Name: "MIT",
				URL:  "https://opensource.org/licenses/MIT",
			},
		},
		Servers: []APIServer{
			{
				URL:         "https://api.aistyler.com",
				Description: "Production server",
			},
			{
				URL:         "https://staging-api.aistyler.com",
				Description: "Staging server",
			},
			{
				URL:         "http://localhost:8080",
				Description: "Development server",
			},
		},
		Paths: generateAPIPaths(),
		Components: APIComponents{
			Schemas: generateAPISchemas(),
			SecuritySchemes: map[string]interface{}{
				"BearerAuth": map[string]interface{}{
					"type":         "http",
					"scheme":       "bearer",
					"bearerFormat": "JWT",
				},
			},
		},
		Tags: generateAPITags(),
	}
}

// generateAPIPaths generates all API paths
func generateAPIPaths() map[string]APIPath {
	return map[string]APIPath{
		"/api/health": {
			Get: &APIOperation{
				Tags:        []string{"Health"},
				Summary:     "Health Check",
				Description: "Check the health status of the API",
				OperationID: "healthCheck",
				Responses: map[string]APIResponse{
					"200": {
						Description: "Service is healthy",
						Content: map[string]APIContent{
							"application/json": {
								Schema: &APISchema{
									Type: "object",
									Properties: map[string]*APISchema{
										"status": {
											Type:    "string",
											Example: "healthy",
										},
										"timestamp": {
											Type:    "string",
											Format:  "date-time",
											Example: time.Now().Format(time.RFC3339),
										},
									},
								},
							},
						},
					},
				},
			},
		},
		"/api/auth/send-otp": {
			Post: &APIOperation{
				Tags:        []string{"Authentication"},
				Summary:     "Send OTP",
				Description: "Send OTP to user's phone number",
				OperationID: "sendOTP",
				RequestBody: &APIRequestBody{
					Description: "OTP request details",
					Content: map[string]APIContent{
						"application/json": {
							Schema: &APISchema{
								Ref: "#/components/schemas/SendOTPRequest",
							},
						},
					},
					Required: true,
				},
				Responses: map[string]APIResponse{
					"200": {
						Description: "OTP sent successfully",
						Content: map[string]APIContent{
							"application/json": {
								Schema: &APISchema{
									Ref: "#/components/schemas/SendOTPResponse",
								},
							},
						},
					},
					"400": {
						Description: "Bad request",
						Content: map[string]APIContent{
							"application/json": {
								Schema: &APISchema{
									Ref: "#/components/schemas/ErrorResponse",
								},
							},
						},
					},
					"429": {
						Description: "Rate limit exceeded",
						Content: map[string]APIContent{
							"application/json": {
								Schema: &APISchema{
									Ref: "#/components/schemas/ErrorResponse",
								},
							},
						},
					},
				},
			},
		},
		"/api/auth/verify-otp": {
			Post: &APIOperation{
				Tags:        []string{"Authentication"},
				Summary:     "Verify OTP",
				Description: "Verify OTP code",
				OperationID: "verifyOTP",
				RequestBody: &APIRequestBody{
					Description: "OTP verification details",
					Content: map[string]APIContent{
						"application/json": {
							Schema: &APISchema{
								Ref: "#/components/schemas/VerifyOTPRequest",
							},
						},
					},
					Required: true,
				},
				Responses: map[string]APIResponse{
					"200": {
						Description: "OTP verified successfully",
						Content: map[string]APIContent{
							"application/json": {
								Schema: &APISchema{
									Ref: "#/components/schemas/VerifyOTPResponse",
								},
							},
						},
					},
					"400": {
						Description: "Bad request",
						Content: map[string]APIContent{
							"application/json": {
								Schema: &APISchema{
									Ref: "#/components/schemas/ErrorResponse",
								},
							},
						},
					},
				},
			},
		},
		"/api/auth/register": {
			Post: &APIOperation{
				Tags:        []string{"Authentication"},
				Summary:     "Register User",
				Description: "Register a new user account",
				OperationID: "registerUser",
				RequestBody: &APIRequestBody{
					Description: "User registration details",
					Content: map[string]APIContent{
						"application/json": {
							Schema: &APISchema{
								Ref: "#/components/schemas/RegisterRequest",
							},
						},
					},
					Required: true,
				},
				Responses: map[string]APIResponse{
					"201": {
						Description: "User registered successfully",
						Content: map[string]APIContent{
							"application/json": {
								Schema: &APISchema{
									Ref: "#/components/schemas/RegisterResponse",
								},
							},
						},
					},
					"400": {
						Description: "Bad request",
						Content: map[string]APIContent{
							"application/json": {
								Schema: &APISchema{
									Ref: "#/components/schemas/ErrorResponse",
								},
							},
						},
					},
					"409": {
						Description: "User already exists",
						Content: map[string]APIContent{
							"application/json": {
								Schema: &APISchema{
									Ref: "#/components/schemas/ErrorResponse",
								},
							},
						},
					},
				},
			},
		},
		"/api/auth/login": {
			Post: &APIOperation{
				Tags:        []string{"Authentication"},
				Summary:     "Login User",
				Description: "Authenticate user and return access tokens",
				OperationID: "loginUser",
				RequestBody: &APIRequestBody{
					Description: "User login credentials",
					Content: map[string]APIContent{
						"application/json": {
							Schema: &APISchema{
								Ref: "#/components/schemas/LoginRequest",
							},
						},
					},
					Required: true,
				},
				Responses: map[string]APIResponse{
					"200": {
						Description: "Login successful",
						Content: map[string]APIContent{
							"application/json": {
								Schema: &APISchema{
									Ref: "#/components/schemas/LoginResponse",
								},
							},
						},
					},
					"401": {
						Description: "Invalid credentials",
						Content: map[string]APIContent{
							"application/json": {
								Schema: &APISchema{
									Ref: "#/components/schemas/ErrorResponse",
								},
							},
						},
					},
				},
			},
		},
		"/api/conversions": {
			Post: &APIOperation{
				Tags:        []string{"Conversions"},
				Summary:     "Create Conversion",
				Description: "Create a new image conversion request",
				OperationID: "createConversion",
				Security: []map[string][]string{
					{"BearerAuth": {}},
				},
				RequestBody: &APIRequestBody{
					Description: "Conversion request details",
					Content: map[string]APIContent{
						"application/json": {
							Schema: &APISchema{
								Ref: "#/components/schemas/CreateConversionRequest",
							},
						},
					},
					Required: true,
				},
				Responses: map[string]APIResponse{
					"201": {
						Description: "Conversion created successfully",
						Content: map[string]APIContent{
							"application/json": {
								Schema: &APISchema{
									Ref: "#/components/schemas/ConversionResponse",
								},
							},
						},
					},
					"400": {
						Description: "Bad request",
						Content: map[string]APIContent{
							"application/json": {
								Schema: &APISchema{
									Ref: "#/components/schemas/ErrorResponse",
								},
							},
						},
					},
					"401": {
						Description: "Unauthorized",
						Content: map[string]APIContent{
							"application/json": {
								Schema: &APISchema{
									Ref: "#/components/schemas/ErrorResponse",
								},
							},
						},
					},
					"429": {
						Description: "Quota exceeded",
						Content: map[string]APIContent{
							"application/json": {
								Schema: &APISchema{
									Ref: "#/components/schemas/ErrorResponse",
								},
							},
						},
					},
				},
			},
			Get: &APIOperation{
				Tags:        []string{"Conversions"},
				Summary:     "List Conversions",
				Description: "Get user's conversion history",
				OperationID: "listConversions",
				Security: []map[string][]string{
					{"BearerAuth": {}},
				},
				Parameters: []APIParameter{
					{
						Name:        "page",
						In:          "query",
						Description: "Page number",
						Required:    false,
						Schema: &APISchema{
							Type:    "integer",
							Minimum: 1,
							Example: 1,
						},
					},
					{
						Name:        "page_size",
						In:          "query",
						Description: "Number of items per page",
						Required:    false,
						Schema: &APISchema{
							Type:    "integer",
							Minimum: 1,
							Maximum: 100,
							Example: 20,
						},
					},
				},
				Responses: map[string]APIResponse{
					"200": {
						Description: "Conversions retrieved successfully",
						Content: map[string]APIContent{
							"application/json": {
								Schema: &APISchema{
									Ref: "#/components/schemas/ConversionListResponse",
								},
							},
						},
					},
					"401": {
						Description: "Unauthorized",
						Content: map[string]APIContent{
							"application/json": {
								Schema: &APISchema{
									Ref: "#/components/schemas/ErrorResponse",
								},
							},
						},
					},
				},
			},
		},
	}
}

// generateAPISchemas generates all API schemas
func generateAPISchemas() map[string]*APISchema {
	return map[string]*APISchema{
		"SendOTPRequest": {
			Type: "object",
			Properties: map[string]*APISchema{
				"phone": {
					Type:        "string",
					Description: "Phone number in international format",
					Pattern:     "^\\+[1-9]\\d{1,14}$",
					Example:     "+1234567890",
				},
				"purpose": {
					Type:        "string",
					Description: "Purpose of the OTP",
					Enum:        []interface{}{"phone_verify", "password_reset"},
					Example:     "phone_verify",
				},
			},
			Required: []string{"phone", "purpose"},
		},
		"SendOTPResponse": {
			Type: "object",
			Properties: map[string]*APISchema{
				"sent": {
					Type:        "boolean",
					Description: "Whether OTP was sent successfully",
					Example:     true,
				},
				"expires_in_sec": {
					Type:        "integer",
					Description: "OTP expiration time in seconds",
					Example:     300,
				},
			},
		},
		"VerifyOTPRequest": {
			Type: "object",
			Properties: map[string]*APISchema{
				"phone": {
					Type:        "string",
					Description: "Phone number in international format",
					Pattern:     "^\\+[1-9]\\d{1,14}$",
					Example:     "+1234567890",
				},
				"code": {
					Type:        "string",
					Description: "OTP code",
					Pattern:     "^\\d{6}$",
					Example:     "123456",
				},
				"purpose": {
					Type:        "string",
					Description: "Purpose of the OTP",
					Enum:        []interface{}{"phone_verify", "password_reset"},
					Example:     "phone_verify",
				},
			},
			Required: []string{"phone", "code", "purpose"},
		},
		"VerifyOTPResponse": {
			Type: "object",
			Properties: map[string]*APISchema{
				"verified": {
					Type:        "boolean",
					Description: "Whether OTP was verified successfully",
					Example:     true,
				},
			},
		},
		"RegisterRequest": {
			Type: "object",
			Properties: map[string]*APISchema{
				"phone": {
					Type:        "string",
					Description: "Phone number in international format",
					Pattern:     "^\\+[1-9]\\d{1,14}$",
					Example:     "+1234567890",
				},
				"password": {
					Type:        "string",
					Description: "User password",
					MinLength:   10,
					Example:     "SecurePassword123!",
				},
				"role": {
					Type:        "string",
					Description: "User role",
					Enum:        []interface{}{"user", "vendor"},
					Example:     "user",
				},
				"display_name": {
					Type:        "string",
					Description: "User display name",
					MaxLength:   100,
					Example:     "John Doe",
				},
				"company_name": {
					Type:        "string",
					Description: "Company name (for vendors)",
					MaxLength:   200,
					Example:     "Acme Corp",
				},
				"auto_login": {
					Type:        "boolean",
					Description: "Whether to automatically log in after registration",
					Example:     false,
				},
			},
			Required: []string{"phone", "password", "role"},
		},
		"RegisterResponse": {
			Type: "object",
			Properties: map[string]*APISchema{
				"user_id": {
					Type:        "string",
					Description: "User ID",
					Format:      "uuid",
					Example:     "123e4567-e89b-12d3-a456-426614174000",
				},
				"role": {
					Type:        "string",
					Description: "User role",
					Example:     "user",
				},
				"is_phone_verified": {
					Type:        "boolean",
					Description: "Whether phone is verified",
					Example:     true,
				},
				"access_token": {
					Type:        "string",
					Description: "Access token (if auto_login is true)",
					Example:     "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
				},
				"access_token_expires_in": {
					Type:        "integer",
					Description: "Access token expiration time in seconds",
					Example:     900,
				},
				"refresh_token": {
					Type:        "string",
					Description: "Refresh token (if auto_login is true)",
					Example:     "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
				},
				"refresh_token_expires_at": {
					Type:        "string",
					Description: "Refresh token expiration time",
					Format:      "date-time",
					Example:     "2024-01-01T12:00:00Z",
				},
			},
		},
		"LoginRequest": {
			Type: "object",
			Properties: map[string]*APISchema{
				"phone": {
					Type:        "string",
					Description: "Phone number in international format",
					Pattern:     "^\\+[1-9]\\d{1,14}$",
					Example:     "+1234567890",
				},
				"password": {
					Type:        "string",
					Description: "User password",
					Example:     "SecurePassword123!",
				},
			},
			Required: []string{"phone", "password"},
		},
		"LoginResponse": {
			Type: "object",
			Properties: map[string]*APISchema{
				"access_token": {
					Type:        "string",
					Description: "Access token",
					Example:     "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
				},
				"access_token_expires_in": {
					Type:        "integer",
					Description: "Access token expiration time in seconds",
					Example:     900,
				},
				"refresh_token": {
					Type:        "string",
					Description: "Refresh token",
					Example:     "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
				},
				"refresh_token_expires_at": {
					Type:        "string",
					Description: "Refresh token expiration time",
					Format:      "date-time",
					Example:     "2024-01-01T12:00:00Z",
				},
				"user": {
					Type: "object",
					Properties: map[string]*APISchema{
						"id": {
							Type:        "string",
							Description: "User ID",
							Format:      "uuid",
							Example:     "123e4567-e89b-12d3-a456-426614174000",
						},
						"role": {
							Type:        "string",
							Description: "User role",
							Example:     "user",
						},
						"is_phone_verified": {
							Type:        "boolean",
							Description: "Whether phone is verified",
							Example:     true,
						},
					},
				},
			},
		},
		"CreateConversionRequest": {
			Type: "object",
			Properties: map[string]*APISchema{
				"user_image_id": {
					Type:        "string",
					Description: "User's image ID",
					Format:      "uuid",
					Example:     "123e4567-e89b-12d3-a456-426614174000",
				},
				"cloth_image_id": {
					Type:        "string",
					Description: "Cloth image ID",
					Format:      "uuid",
					Example:     "123e4567-e89b-12d3-a456-426614174001",
				},
			},
			Required: []string{"user_image_id", "cloth_image_id"},
		},
		"ConversionResponse": {
			Type: "object",
			Properties: map[string]*APISchema{
				"id": {
					Type:        "string",
					Description: "Conversion ID",
					Format:      "uuid",
					Example:     "123e4567-e89b-12d3-a456-426614174000",
				},
				"user_id": {
					Type:        "string",
					Description: "User ID",
					Format:      "uuid",
					Example:     "123e4567-e89b-12d3-a456-426614174000",
				},
				"status": {
					Type:        "string",
					Description: "Conversion status",
					Enum:        []interface{}{"pending", "processing", "completed", "failed"},
					Example:     "pending",
				},
				"created_at": {
					Type:        "string",
					Description: "Creation timestamp",
					Format:      "date-time",
					Example:     "2024-01-01T12:00:00Z",
				},
				"completed_at": {
					Type:        "string",
					Description: "Completion timestamp",
					Format:      "date-time",
					Example:     "2024-01-01T12:05:00Z",
				},
				"result_image_id": {
					Type:        "string",
					Description: "Result image ID",
					Format:      "uuid",
					Example:     "123e4567-e89b-12d3-a456-426614174002",
				},
				"processing_time_ms": {
					Type:        "integer",
					Description: "Processing time in milliseconds",
					Example:     5000,
				},
				"error_message": {
					Type:        "string",
					Description: "Error message if conversion failed",
					Example:     "Image processing failed",
				},
			},
		},
		"ConversionListResponse": {
			Type: "object",
			Properties: map[string]*APISchema{
				"conversions": {
					Type: "array",
					Items: &APISchema{
						Ref: "#/components/schemas/ConversionResponse",
					},
				},
				"total": {
					Type:        "integer",
					Description: "Total number of conversions",
					Example:     100,
				},
				"page": {
					Type:        "integer",
					Description: "Current page number",
					Example:     1,
				},
				"page_size": {
					Type:        "integer",
					Description: "Number of items per page",
					Example:     20,
				},
				"total_pages": {
					Type:        "integer",
					Description: "Total number of pages",
					Example:     5,
				},
			},
		},
		"ErrorResponse": {
			Type: "object",
			Properties: map[string]*APISchema{
				"error": {
					Type:        "string",
					Description: "Error type",
					Example:     "validation_error",
				},
				"code": {
					Type:        "string",
					Description: "Error code",
					Example:     "INVALID_PHONE",
				},
				"message": {
					Type:        "string",
					Description: "Error message",
					Example:     "Invalid phone number format",
				},
				"details": {
					Type:        "object",
					Description: "Additional error details",
					Example:     map[string]interface{}{"field": "phone"},
				},
				"timestamp": {
					Type:        "string",
					Description: "Error timestamp",
					Format:      "date-time",
					Example:     "2024-01-01T12:00:00Z",
				},
				"request_id": {
					Type:        "string",
					Description: "Request ID for tracing",
					Example:     "req_123456789",
				},
			},
		},
	}
}

// generateAPITags generates all API tags
func generateAPITags() []APITag {
	return []APITag{
		{
			Name:        "Authentication",
			Description: "User authentication and authorization",
		},
		{
			Name:        "Conversions",
			Description: "Image conversion operations",
		},
		{
			Name:        "Images",
			Description: "Image management operations",
		},
		{
			Name:        "Payments",
			Description: "Payment and subscription operations",
		},
		{
			Name:        "Users",
			Description: "User profile and account management",
		},
		{
			Name:        "Vendors",
			Description: "Vendor profile and gallery management",
		},
		{
			Name:        "Admin",
			Description: "Administrative operations",
		},
		{
			Name:        "Health",
			Description: "Health check and monitoring",
		},
	}
}

// ServeAPIDocumentation serves the API documentation
func ServeAPIDocumentation(w http.ResponseWriter, r *http.Request) {
	doc := GenerateAPIDocumentation()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(doc)
}

// ServeSwaggerUI serves the Swagger UI
func ServeSwaggerUI(w http.ResponseWriter, r *http.Request) {
	swaggerHTML := `
<!DOCTYPE html>
<html>
<head>
    <title>AI Styler API Documentation</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@4.15.5/swagger-ui.css" />
    <style>
        html {
            box-sizing: border-box;
            overflow: -moz-scrollbars-vertical;
            overflow-y: scroll;
        }
        *, *:before, *:after {
            box-sizing: inherit;
        }
        body {
            margin:0;
            background: #fafafa;
        }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@4.15.5/swagger-ui-bundle.js"></script>
    <script src="https://unpkg.com/swagger-ui-dist@4.15.5/swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            const ui = SwaggerUIBundle({
                url: '/api/docs/openapi.json',
                dom_id: '#swagger-ui',
                deepLinking: true,
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                plugins: [
                    SwaggerUIBundle.plugins.DownloadUrl
                ],
                layout: "StandaloneLayout"
            });
        };
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(swaggerHTML))
}
