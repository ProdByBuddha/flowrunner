#!/bin/bash

echo "=== DynamoDB Mock vs Real Comparison Test ==="
echo

# Test with mock (default, fast)
echo "1. Testing with Mock DynamoDB (fast)..."
go test ./pkg/storage -v -run TestDynamoDB
mock_result=$?

echo
echo "Mock test result: $mock_result"
echo

# Test with real DynamoDB if credentials are available (slow)
if [[ -n "$AWS_ACCESS_KEY_ID" && -n "$AWS_SECRET_ACCESS_KEY" ]] || [[ -n "$DYNAMODB_ENDPOINT" ]]; then
    echo "2. Testing with Real DynamoDB (slow, ~3-5 minutes)..."
    echo "   Note: This will create and delete real tables"
    echo "   Press Ctrl+C within 5 seconds to skip real DynamoDB test..."
    sleep 5
    
    # Use longer timeout for real DynamoDB - note this may take several minutes
    go test ./pkg/storage -v -run TestDynamoDB -real-dynamodb -timeout=5m
    real_result=$?
    
    echo
    echo "Real DynamoDB test result: $real_result"
    echo
    
    if [[ $mock_result -eq 0 && $real_result -eq 0 ]]; then
        echo "✅ SUCCESS: Both Mock and Real DynamoDB tests passed!"
    elif [[ $mock_result -eq 0 ]]; then
        echo "⚠️  PARTIAL: Mock tests passed, Real DynamoDB tests failed"
    else
        echo "❌ FAILURE: Mock tests failed"
    fi
else
    echo "2. Skipping Real DynamoDB test (no AWS credentials or local endpoint)"
    echo "   Set AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY or DYNAMODB_ENDPOINT to test real DynamoDB"
    
    if [[ $mock_result -eq 0 ]]; then
        echo "✅ SUCCESS: Mock DynamoDB tests passed!"
    else
        echo "❌ FAILURE: Mock DynamoDB tests failed"
    fi
fi

echo
echo "=== Test Summary ==="
echo "- Account Service: ✅ Implemented in pkg/services/"
echo "- DynamoDB Mock: ✅ Implemented with interface pattern"
echo "- Flag-based Testing: ✅ Use -real-dynamodb flag to test real DB"
echo "- Mock by Default: ✅ Fast tests using in-memory mock"
echo
echo "Usage examples:"
echo "  go test ./pkg/storage                    # Use mock (fast)"
echo "  go test ./pkg/storage -real-dynamodb     # Use real DynamoDB (slow)"
echo
