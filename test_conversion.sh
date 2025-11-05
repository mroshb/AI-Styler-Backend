#!/bin/bash

# Test script for conversion endpoint
# Usage: ./test_conversion.sh <access_token>

BASE_URL="http://localhost:8080"
ACCESS_TOKEN="${1:-YOUR_ACCESS_TOKEN_HERE}"

echo "Testing conversion endpoint..."
echo "=================================="
echo ""

# Test conversion creation
echo "Creating conversion request..."
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

echo "HTTP Status Code: $HTTP_CODE"
echo "Response Body:"
echo "$BODY" | jq '.' 2>/dev/null || echo "$BODY"
echo ""

if [ "$HTTP_CODE" -eq 201 ]; then
    CONVERSION_ID=$(echo "$BODY" | jq -r '.data.id' 2>/dev/null || echo "$BODY" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
    
    if [ -n "$CONVERSION_ID" ] && [ "$CONVERSION_ID" != "null" ]; then
        echo "✓ Conversion created successfully!"
        echo "Conversion ID: $CONVERSION_ID"
        echo ""
        
        # Wait a bit for processing
        echo "Waiting 5 seconds for processing to start..."
        sleep 5
        
        # Check conversion status
        echo "Checking conversion status..."
        STATUS_RESPONSE=$(curl -s -w "\n%{http_code}" -X GET "${BASE_URL}/api/conversion/${CONVERSION_ID}/status" \
          -H "Authorization: Bearer ${ACCESS_TOKEN}")
        
        STATUS_CODE=$(echo "$STATUS_RESPONSE" | tail -n1)
        STATUS_BODY=$(echo "$STATUS_RESPONSE" | sed '$d')
        
        echo "Status HTTP Code: $STATUS_CODE"
        echo "Status Response:"
        echo "$STATUS_BODY" | jq '.' 2>/dev/null || echo "$STATUS_BODY"
        echo ""
        
        # Get full conversion details
        echo "Getting full conversion details..."
        DETAILS_RESPONSE=$(curl -s -w "\n%{http_code}" -X GET "${BASE_URL}/api/conversion/${CONVERSION_ID}" \
          -H "Authorization: Bearer ${ACCESS_TOKEN}")
        
        DETAILS_CODE=$(echo "$DETAILS_RESPONSE" | tail -n1)
        DETAILS_BODY=$(echo "$DETAILS_RESPONSE" | sed '$d')
        
        echo "Details HTTP Code: $DETAILS_CODE"
        echo "Details Response:"
        echo "$DETAILS_BODY" | jq '.' 2>/dev/null || echo "$DETAILS_BODY"
    else
        echo "✗ Failed to extract conversion ID from response"
    fi
else
    echo "✗ Conversion creation failed!"
    echo "Check the error message above"
fi

echo ""
echo "=================================="
echo "Test completed!"

