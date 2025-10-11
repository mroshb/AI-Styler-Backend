#!/bin/bash

# Test script that runs tests without database integration tests

echo "Running tests without database integration tests..."

# Set environment variables to skip database tests
export SKIP_DB_TESTS=true

# Run tests excluding integration tests
go test ./internal/... -v -short

echo "Tests completed!"
