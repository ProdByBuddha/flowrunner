#!/bin/bash

# Build the server
echo "Building server..."
go build -o flowrunner ./cmd/flowrunner

# Build the CLI
echo "Building CLI..."
go build -o flowrunner-cli ./cmd/flowrunner-cli

# Start the server in the background
echo "Starting server..."
./flowrunner &
SERVER_PID=$!

# Wait for the server to start
sleep 5

# Create a test account
echo "Creating test account..."
./flowrunner-cli account create --server http://localhost:8080 --username testuser --password testpassword

# Get account info
echo "Getting account info..."
./flowrunner-cli account info --server http://localhost:8080 --username testuser --password testpassword

# Create a test flow
echo "Creating test flow..."
cat > test_flow.yaml << EOF
metadata:
  name: Test Flow
  description: A test flow
  version: 1.0.0
nodes:
  start:
    type: test
    params:
      foo: bar
EOF

./flowrunner-cli flow create TestFlow test_flow.yaml --server http://localhost:8080 --username testuser --password testpassword

# List flows
echo "Listing flows..."
./flowrunner-cli flow list --server http://localhost:8080 --username testuser --password testpassword

# Clean up
echo "Cleaning up..."
kill $SERVER_PID
rm -f flowrunner flowrunner-cli test_flow.yaml

echo "Test completed successfully!"