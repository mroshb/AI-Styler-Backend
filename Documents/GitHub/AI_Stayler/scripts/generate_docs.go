package main

import (
	"AI_Styler/internal/docs"
	"fmt"
	"log"
	"os"
)

func main() {
	// Create api directory if it doesn't exist
	if err := os.MkdirAll("api", 0755); err != nil {
		log.Fatalf("Failed to create api directory: %v", err)
	}

	// Generate OpenAPI specification
	fmt.Println("Generating OpenAPI specification...")
	if err := docs.GenerateOpenAPISpec(); err != nil {
		log.Fatalf("Failed to generate OpenAPI spec: %v", err)
	}
	fmt.Println("✅ OpenAPI specification generated: api/openapi.json")

	// Generate Swagger HTML
	fmt.Println("Generating Swagger HTML documentation...")
	if err := docs.GenerateSwaggerHTML(); err != nil {
		log.Fatalf("Failed to generate Swagger HTML: %v", err)
	}
	fmt.Println("✅ Swagger HTML documentation generated: api/index.html")

	fmt.Println("\n🎉 API documentation generated successfully!")
	fmt.Println("📖 View documentation at: api/index.html")
	fmt.Println("📋 OpenAPI spec at: api/openapi.json")
}
