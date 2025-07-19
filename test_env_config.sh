#!/bin/bash

# Test environment variable configuration
echo "Testing environment variable configuration..."

# Create a temporary .env file
cat > .env.test << EOF
FLOWRUNNER_SERVER_HOST=127.0.0.1
FLOWRUNNER_SERVER_PORT=8085
FLOWRUNNER_STORAGE_TYPE=dynamodb
FLOWRUNNER_DYNAMODB_REGION=us-east-1
FLOWRUNNER_DYNAMODB_ENDPOINT=http://localhost:8000
FLOWRUNNER_DYNAMODB_TABLE_PREFIX=test_
FLOWRUNNER_JWT_SECRET=test-jwt-secret
FLOWRUNNER_TOKEN_EXPIRATION=12
EOF

# Run the server with the test environment variables
echo "Starting server with test environment variables..."
FLOWRUNNER_SERVER_HOST=127.0.0.1 \
FLOWRUNNER_SERVER_PORT=8085 \
FLOWRUNNER_STORAGE_TYPE=dynamodb \
FLOWRUNNER_DYNAMODB_REGION=us-east-1 \
FLOWRUNNER_DYNAMODB_ENDPOINT=http://localhost:8000 \
FLOWRUNNER_DYNAMODB_TABLE_PREFIX=test_ \
FLOWRUNNER_JWT_SECRET=test-jwt-secret \
FLOWRUNNER_TOKEN_EXPIRATION=12 \
./flowrunner &

SERVER_PID=$!

# Wait for the server to start
sleep 2

# Check if the server is running
if ps -p $SERVER_PID > /dev/null; then
    echo "Server started successfully with PID $SERVER_PID"
else
    echo "Failed to start server"
    exit 1
fi

# Kill the server
echo "Stopping server..."
kill $SERVER_PID

# Clean up
rm .env.test

echo "Test completed successfully!"