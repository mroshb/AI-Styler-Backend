#!/bin/bash
# Quick script to create worker_jobs table
# This script will read database config from .env file and create the table

set -e

echo "🔧 Creating worker_jobs table..."

# Check if .env file exists
if [ -f .env ]; then
    echo "📄 Loading configuration from .env file..."
    export $(grep -v '^#' .env | xargs)
fi

# Set defaults if not in .env
DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_USER=${DB_USER:-postgres}
DB_NAME=${DB_NAME:-styler}

# Check if DB_PASSWORD is set
if [ -z "$DB_PASSWORD" ]; then
    echo "⚠️  DB_PASSWORD not found in .env or environment"
    echo "Please enter database password:"
    read -s DB_PASSWORD
fi

echo "🔌 Connecting to database: $DB_NAME@$DB_HOST:$DB_PORT"

# Run the SQL script
PGPASSWORD=$DB_PASSWORD psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f scripts/create_worker_table.sql

if [ $? -eq 0 ]; then
    echo "✅ Successfully created worker_jobs table!"
    echo ""
    echo "You can now restart your application."
else
    echo "❌ Failed to create worker_jobs table"
    exit 1
fi

