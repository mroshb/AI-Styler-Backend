#!/bin/bash

# Setup Test Database Script for AI Stayler

echo "Setting up test database for AI Stayler..."

# Database configuration
DB_HOST=${TEST_DB_HOST:-localhost}
DB_PORT=${TEST_DB_PORT:-5432}
DB_USER=${TEST_DB_USER:-postgres}
DB_PASSWORD=${TEST_DB_PASSWORD:-A1212@shb#}
DB_NAME=${TEST_DB_NAME:-styler}

# Check if PostgreSQL is running
if ! pg_isready -h $DB_HOST -p $DB_PORT -U $DB_USER > /dev/null 2>&1; then
    echo "Error: PostgreSQL is not running or not accessible"
    echo "Please make sure PostgreSQL is running on $DB_HOST:$DB_PORT"
    exit 1
fi

# Create test database
echo "Creating test database: $DB_NAME"
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -c "CREATE DATABASE $DB_NAME;" 2>/dev/null || echo "Database $DB_NAME already exists or creation failed"

# Run migrations
echo "Running database migrations..."
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f db/migrations/0001_auth.sql
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f db/migrations/0002_user_service.sql
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f db/migrations/0003_vendor_service.sql
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f db/migrations/0004_image_service.sql
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f db/migrations/0005_conversion_service.sql

echo "Test database setup completed!"
echo "Database: $DB_NAME"
echo "Host: $DB_HOST:$DB_PORT"
echo "User: $DB_USER"
