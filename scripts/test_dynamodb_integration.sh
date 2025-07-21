#!/bin/bash

# Test script for DynamoDB WebSocket Integration Tests
# This script sets up environment variables and runs the DynamoDB integration tests

set -e

echo "🚀 DynamoDB WebSocket Integration Test Runner"
echo "=============================================="

# Check if .env file exists
if [ -f ".env" ]; then
    echo "✅ Found .env file, loading environment variables..."
    source .env
else
    echo "⚠️  No .env file found, using default values..."
fi

# Set default DynamoDB configuration if not already set
export FLOWRUNNER_DYNAMODB_ENDPOINT=${FLOWRUNNER_DYNAMODB_ENDPOINT:-http://localhost:8000}
export FLOWRUNNER_DYNAMODB_REGION=${FLOWRUNNER_DYNAMODB_REGION:-us-east-1}
export FLOWRUNNER_DYNAMODB_TABLE_PREFIX=${FLOWRUNNER_DYNAMODB_TABLE_PREFIX:-flowrunner_test_}

echo "📋 DynamoDB Configuration:"
echo "   Endpoint: $FLOWRUNNER_DYNAMODB_ENDPOINT"
echo "   Region: $FLOWRUNNER_DYNAMODB_REGION"
echo "   Table Prefix: $FLOWRUNNER_DYNAMODB_TABLE_PREFIX"
echo ""

# Check if DynamoDB is accessible
echo "🔍 Checking DynamoDB connectivity..."
if command -v aws >/dev/null 2>&1; then
    if aws dynamodb list-tables --endpoint-url $FLOWRUNNER_DYNAMODB_ENDPOINT --region $FLOWRUNNER_DYNAMODB_REGION >/dev/null 2>&1; then
        echo "✅ DynamoDB is accessible"
    else
        echo "❌ Cannot connect to DynamoDB. Please check your configuration."
        echo "   Make sure DynamoDB is running and endpoint is correct."
        exit 1
    fi
else
    echo "⚠️  AWS CLI not found, skipping connectivity check"
    echo "   Ensure DynamoDB is running at $FLOWRUNNER_DYNAMODB_ENDPOINT"
fi

echo ""
echo "🧪 Running DynamoDB Integration Tests..."
echo "=========================================="

# Run the simple branching test first
echo "1️⃣  Running Simple Branching Test..."
go test -v ./pkg/api -run TestWebSocketDynamoDBIntegration_SimpleBranching -timeout 60s

if [ $? -eq 0 ]; then
    echo "✅ Simple Branching Test passed!"
    echo ""
    
    # Run the complex flow test
    echo "2️⃣  Running Complex Flow Test..."
    go test -v ./pkg/api -run TestWebSocketDynamoDBIntegration_ComplexFlow -timeout 120s
    
    if [ $? -eq 0 ]; then
        echo "✅ Complex Flow Test passed!"
        echo ""
        echo "🎉 All DynamoDB WebSocket Integration Tests passed!"
        echo ""
        echo "📊 Test Summary:"
        echo "   ✅ DynamoDB backend integration"
        echo "   ✅ WebSocket real-time updates"
        echo "   ✅ Complex flow with branching"
        echo "   ✅ Parallel batch processing"
        echo "   ✅ Retry logic with backoff"
        echo "   ✅ Concurrent execution handling"
        echo ""
        echo "🚀 Your FlowRunner system is production-ready with DynamoDB!"
    else
        echo "❌ Complex Flow Test failed"
        exit 1
    fi
else
    echo "❌ Simple Branching Test failed"
    exit 1
fi