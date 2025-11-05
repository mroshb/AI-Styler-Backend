#!/bin/bash

# Manual test script for conversion endpoint
# Usage: ./test_conversion_manual.sh

BASE_URL="http://localhost:8080"

echo "=========================================="
echo "Testing Conversion Endpoint"
echo "=========================================="
echo ""
echo "Note: You need a valid access token to run this test."
echo "Get your token by logging in first:"
echo "  curl -X POST ${BASE_URL}/auth/login -H 'Content-Type: application/json' -d '{\"phone\":\"YOUR_PHONE\",\"password\":\"YOUR_PASSWORD\"}'"
echo ""
read -p "Enter your access token (or press Enter to skip): " ACCESS_TOKEN

if [ -z "$ACCESS_TOKEN" ]; then
    echo "Skipping test - no token provided"
    exit 0
fi

echo ""
echo "Creating conversion request..."
echo "User Image ID: 93e7c110-35e4-4498-ad22-f3c8171a069d"
echo "Cloth Image ID: ab7e7810-c870-47bc-ae44-476bd2d87168"
echo ""

RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "${BASE_URL}/api/convert" \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "user_image_id": "93e7c110-35e4-4498-ad22-f3c8171a069d",
    "cloth_image_id": "ab7e7810-c870-47bc-ae44-476bd2d87168",
    "style_name": "vintage"
  }')

HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | sed '$d')

echo "Response:"
echo "HTTP Status: $HTTP_CODE"
echo ""

if command -v jq &> /dev/null; then
    echo "$BODY" | jq '.'
else
    echo "$BODY"
fi

echo ""
echo "=========================================="

if [ "$HTTP_CODE" -eq 201 ]; then
    echo "✓ Conversion created successfully!"
    
    if command -v jq &> /dev/null; then
        CONVERSION_ID=$(echo "$BODY" | jq -r '.data.id // .id // empty')
    else
        CONVERSION_ID=$(echo "$BODY" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
    fi
    
    if [ -n "$CONVERSION_ID" ] && [ "$CONVERSION_ID" != "null" ]; then
        echo "Conversion ID: $CONVERSION_ID"
        echo ""
        echo "Waiting 3 seconds, then checking status..."
        sleep 3
        
        echo ""
        echo "Checking conversion status..."
        STATUS_RESPONSE=$(curl -s -w "\n%{http_code}" -X GET "${BASE_URL}/api/conversion/${CONVERSION_ID}/status" \
          -H "Authorization: Bearer ${ACCESS_TOKEN}")
        
        STATUS_CODE=$(echo "$STATUS_RESPONSE" | tail -n1)
        STATUS_BODY=$(echo "$STATUS_RESPONSE" | sed '$d')
        
        echo "Status HTTP: $STATUS_CODE"
        if command -v jq &> /dev/null; then
            echo "$STATUS_BODY" | jq '.'
        else
            echo "$STATUS_BODY"
        fi
    fi
else
    echo "✗ Conversion creation failed!"
    echo "Check the error message above"
fi

echo ""
echo "=========================================="
echo "Test completed!"
echo ""
echo "Check server logs for detailed processing information."

