#!/bin/bash
# Quick fix script to create worker_jobs table

echo "Creating worker_jobs table..."
echo ""
echo "Please run this command manually:"
echo "psql -d YOUR_DB_NAME -f scripts/create_worker_table.sql"
echo ""
echo "Or if you have .env file with DB config:"
echo "source .env && psql -h \$DB_HOST -p \$DB_PORT -U \$DB_USER -d \$DB_NAME -f scripts/create_worker_table.sql"
