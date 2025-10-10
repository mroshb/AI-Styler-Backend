package docs

import (
	"encoding/json"
	"fmt"
	"os"
)

// GenerateOpenAPISpec generates OpenAPI 3.0 specification
func GenerateOpenAPISpec() error {
	spec := map[string]interface{}{
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title":       "AI Styler Backend API",
			"description": "AI-powered clothing try-on platform backend API",
			"version":     "1.0.0",
			"contact": map[string]interface{}{
				"name":  "AI Styler Team",
				"email": "support@aistyler.com",
			},
			"license": map[string]interface{}{
				"name": "MIT",
				"url":  "https://opensource.org/licenses/MIT",
			},
		},
		"servers": []map[string]interface{}{
			{
				"url":         "http://localhost:8080",
				"description": "Development server",
			},
			{
				"url":         "https://api.aistyler.com",
				"description": "Production server",
			},
		},
		"security": []map[string]interface{}{
			{
				"bearerAuth": []string{},
			},
		},
		"paths": map[string]interface{}{
			"/api/health": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Health Check",
					"description": "Check if the service is healthy",
					"tags":        []string{"Health"},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Service is healthy",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"status": map[string]interface{}{
												"type":    "string",
												"example": "ok",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			"/api/auth/send-otp": map[string]interface{}{
				"post": map[string]interface{}{
					"summary":     "Send OTP",
					"description": "Send OTP code to phone number for verification",
					"tags":        []string{"Authentication"},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"phone_number": map[string]interface{}{
											"type":    "string",
											"example": "+1234567890",
										},
									},
									"required": []string{"phone_number"},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "OTP sent successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"message": map[string]interface{}{
												"type":    "string",
												"example": "OTP sent successfully",
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "Bad request",
						},
						"429": map[string]interface{}{
							"description": "Rate limit exceeded",
						},
					},
				},
			},
			"/api/auth/verify-otp": map[string]interface{}{
				"post": map[string]interface{}{
					"summary":     "Verify OTP",
					"description": "Verify OTP code and get verification token",
					"tags":        []string{"Authentication"},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"phone_number": map[string]interface{}{
											"type":    "string",
											"example": "+1234567890",
										},
										"otp_code": map[string]interface{}{
											"type":    "string",
											"example": "123456",
										},
									},
									"required": []string{"phone_number", "otp_code"},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "OTP verified successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"verification_token": map[string]interface{}{
												"type":    "string",
												"example": "verification-token-123",
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "Invalid OTP code",
						},
					},
				},
			},
			"/api/auth/register": map[string]interface{}{
				"post": map[string]interface{}{
					"summary":     "Register User",
					"description": "Register a new user or vendor",
					"tags":        []string{"Authentication"},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"phone_number": map[string]interface{}{
											"type":    "string",
											"example": "+1234567890",
										},
										"password": map[string]interface{}{
											"type":    "string",
											"example": "securepassword123",
										},
										"user_type": map[string]interface{}{
											"type":    "string",
											"enum":    []string{"user", "vendor"},
											"example": "user",
										},
										"verification_token": map[string]interface{}{
											"type":    "string",
											"example": "verification-token-123",
										},
									},
									"required": []string{"phone_number", "password", "user_type", "verification_token"},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "User registered successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"user_id": map[string]interface{}{
												"type":    "string",
												"example": "user-123",
											},
											"access_token": map[string]interface{}{
												"type":    "string",
												"example": "jwt-access-token",
											},
											"refresh_token": map[string]interface{}{
												"type":    "string",
												"example": "jwt-refresh-token",
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "Registration failed",
						},
					},
				},
			},
			"/api/auth/login": map[string]interface{}{
				"post": map[string]interface{}{
					"summary":     "Login",
					"description": "Login with phone number and password",
					"tags":        []string{"Authentication"},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"phone_number": map[string]interface{}{
											"type":    "string",
											"example": "+1234567890",
										},
										"password": map[string]interface{}{
											"type":    "string",
											"example": "securepassword123",
										},
									},
									"required": []string{"phone_number", "password"},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Login successful",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"user_id": map[string]interface{}{
												"type":    "string",
												"example": "user-123",
											},
											"access_token": map[string]interface{}{
												"type":    "string",
												"example": "jwt-access-token",
											},
											"refresh_token": map[string]interface{}{
												"type":    "string",
												"example": "jwt-refresh-token",
											},
										},
									},
								},
							},
						},
						"401": map[string]interface{}{
							"description": "Invalid credentials",
						},
					},
				},
			},
			"/api/users/profile": map[string]interface{}{
				"get": map[string]interface{}{
					"summary":     "Get User Profile",
					"description": "Get current user's profile information",
					"tags":        []string{"Users"},
					"security":    []map[string]interface{}{{"bearerAuth": []string{}}},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "User profile retrieved successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"user_id": map[string]interface{}{
												"type":    "string",
												"example": "user-123",
											},
											"phone_number": map[string]interface{}{
												"type":    "string",
												"example": "+1234567890",
											},
											"user_type": map[string]interface{}{
												"type":    "string",
												"example": "user",
											},
											"created_at": map[string]interface{}{
												"type":    "string",
												"format":  "date-time",
												"example": "2023-01-01T00:00:00Z",
											},
										},
									},
								},
							},
						},
						"401": map[string]interface{}{
							"description": "Unauthorized",
						},
					},
				},
			},
			"/api/conversions/": map[string]interface{}{
				"post": map[string]interface{}{
					"summary":     "Create Conversion",
					"description": "Create a new AI clothing try-on conversion",
					"tags":        []string{"Conversions"},
					"security":    []map[string]interface{}{{"bearerAuth": []string{}}},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"user_image_id": map[string]interface{}{
											"type":    "string",
											"example": "image-123",
										},
										"clothing_image_id": map[string]interface{}{
											"type":    "string",
											"example": "clothing-456",
										},
										"style_preference": map[string]interface{}{
											"type":    "string",
											"example": "casual",
										},
									},
									"required": []string{"user_image_id", "clothing_image_id"},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Conversion created successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"conversion_id": map[string]interface{}{
												"type":    "string",
												"example": "conversion-123",
											},
											"status": map[string]interface{}{
												"type":    "string",
												"example": "pending",
											},
										},
									},
								},
							},
						},
						"403": map[string]interface{}{
							"description": "Quota exceeded",
						},
					},
				},
			},
			"/api/images/": map[string]interface{}{
				"post": map[string]interface{}{
					"summary":     "Upload Image",
					"description": "Upload an image file",
					"tags":        []string{"Images"},
					"security":    []map[string]interface{}{{"bearerAuth": []string{}}},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"multipart/form-data": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"file": map[string]interface{}{
											"type":   "string",
											"format": "binary",
										},
									},
									"required": []string{"file"},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Image uploaded successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"image_id": map[string]interface{}{
												"type":    "string",
												"example": "image-123",
											},
											"url": map[string]interface{}{
												"type":    "string",
												"example": "https://api.aistyler.com/images/image-123",
											},
										},
									},
								},
							},
						},
						"400": map[string]interface{}{
							"description": "Invalid file",
						},
					},
				},
			},
			"/api/payments/": map[string]interface{}{
				"post": map[string]interface{}{
					"summary":     "Create Payment",
					"description": "Create a new payment for subscription plan",
					"tags":        []string{"Payments"},
					"security":    []map[string]interface{}{{"bearerAuth": []string{}}},
					"requestBody": map[string]interface{}{
						"required": true,
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"plan_name": map[string]interface{}{
											"type":    "string",
											"example": "basic",
										},
										"amount": map[string]interface{}{
											"type":    "integer",
											"example": 1000,
										},
									},
									"required": []string{"plan_name", "amount"},
								},
							},
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "Payment created successfully",
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"payment_id": map[string]interface{}{
												"type":    "string",
												"example": "payment-123",
											},
											"payment_url": map[string]interface{}{
												"type":    "string",
												"example": "https://zarinpal.com/payment/payment-123",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		"components": map[string]interface{}{
			"securitySchemes": map[string]interface{}{
				"bearerAuth": map[string]interface{}{
					"type":         "http",
					"scheme":       "bearer",
					"bearerFormat": "JWT",
				},
			},
			"schemas": map[string]interface{}{
				"Error": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"error": map[string]interface{}{
							"type":    "string",
							"example": "Error message",
						},
						"code": map[string]interface{}{
							"type":    "integer",
							"example": 400,
						},
					},
				},
				"User": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"user_id": map[string]interface{}{
							"type":    "string",
							"example": "user-123",
						},
						"phone_number": map[string]interface{}{
							"type":    "string",
							"example": "+1234567890",
						},
						"user_type": map[string]interface{}{
							"type":    "string",
							"example": "user",
						},
						"created_at": map[string]interface{}{
							"type":    "string",
							"format":  "date-time",
							"example": "2023-01-01T00:00:00Z",
						},
					},
				},
				"Conversion": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"conversion_id": map[string]interface{}{
							"type":    "string",
							"example": "conversion-123",
						},
						"user_id": map[string]interface{}{
							"type":    "string",
							"example": "user-123",
						},
						"status": map[string]interface{}{
							"type":    "string",
							"example": "completed",
						},
						"result_image_id": map[string]interface{}{
							"type":    "string",
							"example": "result-image-123",
						},
						"created_at": map[string]interface{}{
							"type":    "string",
							"format":  "date-time",
							"example": "2023-01-01T00:00:00Z",
						},
					},
				},
			},
		},
		"tags": []map[string]interface{}{
			{
				"name":        "Health",
				"description": "Health check endpoints",
			},
			{
				"name":        "Authentication",
				"description": "User authentication and authorization",
			},
			{
				"name":        "Users",
				"description": "User profile and account management",
			},
			{
				"name":        "Vendors",
				"description": "Vendor profile and gallery management",
			},
			{
				"name":        "Conversions",
				"description": "AI clothing try-on conversions",
			},
			{
				"name":        "Images",
				"description": "Image upload and management",
			},
			{
				"name":        "Payments",
				"description": "Payment processing and subscription management",
			},
			{
				"name":        "Share",
				"description": "Public sharing of conversion results",
			},
			{
				"name":        "Admin",
				"description": "Administrative functions",
			},
		},
	}

	// Write to file
	file, err := os.Create("api/openapi.json")
	if err != nil {
		return fmt.Errorf("failed to create openapi.json: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(spec)
	if err != nil {
		return fmt.Errorf("failed to encode OpenAPI spec: %w", err)
	}

	return nil
}

// GenerateSwaggerHTML generates HTML documentation for Swagger UI
func GenerateSwaggerHTML() error {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
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
                url: './openapi.json',
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

	file, err := os.Create("api/index.html")
	if err != nil {
		return fmt.Errorf("failed to create index.html: %w", err)
	}
	defer file.Close()

	_, err = file.WriteString(html)
	if err != nil {
		return fmt.Errorf("failed to write HTML: %w", err)
	}

	return nil
}
