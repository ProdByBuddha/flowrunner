#!/bin/bash

# Test DynamoDB provider with HTTP service
echo "Testing DynamoDB provider with HTTP service..."

# Load environment variables from .env file
if [ -f .env ]; then
    echo "Loading environment variables from .env file..."
    export $(grep -v '^#' .env | xargs)
fi

# Check if DynamoDB Local is running
if ! curl -s http://localhost:8000/shell > /dev/null; then
    echo "DynamoDB Local is not running. Please start DynamoDB Local and try again."
    echo "You can start it with: docker run -p 8000:8000 amazon/dynamodb-local"
    exit 1
fi

# Build the server
echo "Building server..."
go build -o flowrunner ./cmd/flowrunner

# Start the server with DynamoDB configuration
echo "Starting server with DynamoDB configuration..."
FLOWRUNNER_SERVER_PORT=8082 \
FLOWRUNNER_STORAGE_TYPE=dynamodb \
./flowrunner &

SERVER_PID=$!

# Wait for the server to start
sleep 2

# Check if the server is running
if ! ps -p $SERVER_PID > /dev/null; then
    echo "Failed to start server"
    exit 1
fi

echo "Server started successfully with PID $SERVER_PID"

# Test the API
echo "Testing API..."

# Create an account with unique username
USERNAME="dynamouser-$(date +%s)"
echo "Creating account with username: $USERNAME"
ACCOUNT_RESPONSE=$(curl -s -X POST http://localhost:8082/api/v1/accounts \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"$USERNAME\",\"password\":\"dynamopassword\"}")

echo "Account response: $ACCOUNT_RESPONSE"

# Wait for DynamoDB eventual consistency
echo "Waiting for DynamoDB eventual consistency..."
sleep 3

# Login to get JWT token
echo "Logging in..."
LOGIN_RESPONSE=$(curl -s -X POST http://localhost:8082/api/v1/login \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"$USERNAME\",\"password\":\"dynamopassword\"}")

echo "Login response: $LOGIN_RESPONSE"

# Extract token
TOKEN=$(echo $LOGIN_RESPONSE | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
    echo "Failed to get token"
    kill $SERVER_PID
    exit 1
fi

echo "Got token: $TOKEN"

# Create a flow
echo "Creating flow..."
FLOW_RESPONSE=$(curl -s -X POST http://localhost:8082/api/v1/flows \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"Test Flow","content":"metadata:\n  name: Test Flow\n  description: A test flow\n  version: 1.0.0\nnodes:\n  start:\n    type: test\n    params:\n      foo: bar"}')

echo "Flow response: $FLOW_RESPONSE"

# Extract flow ID
FLOW_ID=$(echo $FLOW_RESPONSE | grep -o '"id":"[^"]*"' | cut -d'"' -f4)

if [ -z "$FLOW_ID" ]; then
    echo "Failed to get flow ID"
    kill $SERVER_PID
    exit 1
fi

echo "Got flow ID: $FLOW_ID"

# List flows
echo "Listing flows..."
FLOWS_RESPONSE=$(curl -s -X GET http://localhost:8082/api/v1/flows \
  -H "Authorization: Bearer $TOKEN")

echo "Flows response: $FLOWS_RESPONSE"

# Get flow
echo "Getting flow..."
FLOW_GET_RESPONSE=$(curl -s -X GET http://localhost:8082/api/v1/flows/$FLOW_ID \
  -H "Authorization: Bearer $TOKEN")

echo "Flow get response: $FLOW_GET_RESPONSE"

# Update flow
echo "Updating flow..."
UPDATE_RESPONSE=$(curl -s -X PUT http://localhost:8082/api/v1/flows/$FLOW_ID \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"content":"metadata:\n  name: Updated Test Flow\n  description: An updated test flow\n  version: 1.1.0\nnodes:\n  start:\n    type: test\n    params:\n      foo: updated\n      bar: baz"}')

echo "Update response: $UPDATE_RESPONSE"

# Delete flow
echo "Deleting flow..."
DELETE_RESPONSE=$(curl -s -X DELETE http://localhost:8082/api/v1/flows/$FLOW_ID \
  -H "Authorization: Bearer $TOKEN")

echo "Delete response: $DELETE_RESPONSE"

# Kill the server
echo "Stopping server..."
kill $SERVER_PID

# Clean up
echo "Cleaning up..."
rm -f flowrunner

echo "Test completed successfully!"