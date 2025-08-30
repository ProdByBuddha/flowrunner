#!/bin/bash

# Start the server in the background
echo "Starting server..."
./simple-server &
SERVER_PID=$!

# Wait for the server to start
sleep 2

# Test the health endpoint
echo "Testing health endpoint..."
curl -v http://localhost:8080/api/v1/health

# Create a test account
echo "Creating test account..."
ACCOUNT_RESPONSE=$(curl -v -X POST http://localhost:8080/api/v1/accounts \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"testpassword"}')

echo "Account response:"
echo $ACCOUNT_RESPONSE

# Get account info with basic auth
echo "Getting account info with basic auth..."
curl -v -X GET http://localhost:8080/api/v1/accounts/me \
  -u testuser:testpassword

# Clean up
echo "Cleaning up..."
kill $SERVER_PID

echo "Test completed successfully!"