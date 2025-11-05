#!/bin/bash

# Simple conversion test script
# Make sure your server is running on localhost:8080

BASE_URL="http://localhost:8080"

echo "=========================================="
echo "Conversion Endpoint Test"
echo "=========================================="
echo ""

# Check if server is running
echo "Checking if server is running..."
if ! curl -s -f "${BASE_URL}/api/health" > /dev/null 2>&1; then
    echo "✗ Server is not running on ${BASE_URL}"
    echo "Please start the server first!"
    exit 1
fi
echo "✓ Server is running"
echo ""

# Get access token (you need to login first)
echo "To test, you need to login first and get an access token."
echo "Example login command:"
echo ""
echo "curl -X POST ${BASE_URL}/auth/login \\"
echo "  -H 'Content-Type: application/json' \\"
echo "  -d '{\"phone\":\"YOUR_PHONE\",\"password\":\"YOUR_PASSWORD\"}'"
echo ""
echo "Then copy the 'accessToken' from the response and run:"
echo ""
echo "export ACCESS_TOKEN=\"your_access_token_here\""
echo "./test_conversion_simple.sh"
echo ""

if [ -z "$ACCESS_TOKEN" ]; then
    read -p "Enter your access token (or press Ctrl+C to exit): " ACCESS_TOKEN
fi

if [ -z "$ACCESS_TOKEN" ]; then
    echo "No token provided. Exiting."
    exit 1
fi

echo ""
echo "=========================================="
echo "Creating conversion..."
echo "=========================================="
echo ""

# Create conversion
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
echo ""
echo "Response Body:"
if command -v jq &> /dev/null; then
    echo "$BODY" | jq '.'
else
    echo "$BODY"
fi
echo ""

if [ "$HTTP_CODE" -eq 201 ]; then
    echo "✓ Conversion created successfully!"
    echo ""
    
    # Extract conversion ID
    if command -v jq &> /dev/null; then
        CONVERSION_ID=$(echo "$BODY" | jq -r '.data.id // .id // empty')
    else
        CONVERSION_ID=$(echo "$BODY" | grep -oP '"id"\s*:\s*"[^"]*"' | head -1 | grep -oP '"[^"]*"' | head -1 | tr -d '"')
    fi
    
    if [ -n "$CONVERSION_ID" ] && [ "$CONVERSION_ID" != "null" ] && [ "$CONVERSION_ID" != "" ]; then
        echo "Conversion ID: $CONVERSION_ID"
        echo ""
        
        echo "Waiting 5 seconds for worker to process..."
        sleep 5
        
        echo ""
        echo "=========================================="
        echo "Checking conversion status..."
        echo "=========================================="
        echo ""
        
        STATUS_RESPONSE=$(curl -s -w "\n%{http_code}" -X GET "${BASE_URL}/api/conversion/${CONVERSION_ID}/status" \
          -H "Authorization: Bearer ${ACCESS_TOKEN}")
        
        STATUS_CODE=$(echo "$STATUS_RESPONSE" | tail -n1)
        STATUS_BODY=$(echo "$STATUS_RESPONSE" | sed '$d')
        
        echo "Status HTTP Code: $STATUS_CODE"
        echo ""
        if command -v jq &> /dev/null; then
            echo "$STATUS_BODY" | jq '.'
        else
            echo "$STATUS_BODY"
        fi
        
        echo ""
        echo "=========================================="
        echo "Getting full conversion details..."
        echo "=========================================="
        echo ""
        
        DETAILS_RESPONSE=$(curl -s -X GET "${BASE_URL}/api/conversion/${CONVERSION_ID}" \
          -H "Authorization: Bearer ${ACCESS_TOKEN}")
        
        echo "Full Details:"
        if command -v jq &> /dev/null; then
            echo "$DETAILS_RESPONSE" | jq '.'
        else
            echo "$DETAILS_RESPONSE"
        fi
    else
        echo "✗ Could not extract conversion ID from response"
    fi
else
    echo "✗ Conversion creation failed!"
    echo ""
    echo "Common issues:"
    echo "1. Invalid access token - try logging in again"
    echo "2. Image IDs don't exist - make sure the images are uploaded"
    echo "3. Quota exceeded - check your conversion quota"
    echo "4. Server error - check server logs"
fi

echo ""
echo "=========================================="
echo "Test completed!"
echo "=========================================="

