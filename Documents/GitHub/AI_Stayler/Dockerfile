# Multi-stage build for AI Styler Backend

# Build stage
FROM golang:1.24 AS builder

# Install build dependencies
RUN apt-get update && apt-get install -y git ca-certificates tzdata && rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ai-styler main.go

# Production stage
FROM debian:bookworm-slim

# Install runtime dependencies
RUN apt-get update && apt-get install -y ca-certificates tzdata curl && rm -rf /var/lib/apt/lists/*

# Create non-root user
RUN groupadd -g 1001 styler && \
    useradd -u 1001 -g styler -s /bin/bash styler

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/ai-styler .

# Copy configuration files
COPY --from=builder /app/.env.example .env

# Create necessary directories
RUN mkdir -p uploads logs && \
    chown -R styler:styler /app

# Switch to non-root user
USER styler

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/api/health || exit 1

# Run the application
CMD ["./ai-styler"]