#!/bin/bash

# Test script for PostgreSQL WebSocket Integration Tests
# This script sets up environment variables and runs the PostgreSQL integration tests

set -e

echo "🚀 PostgreSQL WebSocket Integration Test Runner"
echo "=============================================="

# Check if .env file exists
if [ -f ".env" ]; then
    echo "✅ Found .env file, loading environment variables..."
    source .env
else
    echo "⚠️  No .env file found, using default values..."
fi

# Set default PostgreSQL configuration if not already set
export FLOWRUNNER_POSTGRES_HOST=${FLOWRUNNER_POSTGRES_HOST:-localhost}
export FLOWRUNNER_POSTGRES_PORT=${FLOWRUNNER_POSTGRES_PORT:-5432}
export FLOWRUNNER_POSTGRES_USER=${FLOWRUNNER_POSTGRES_USER:-postgres}
export FLOWRUNNER_POSTGRES_PASSWORD=${FLOWRUNNER_POSTGRES_PASSWORD:-postgres}
export FLOWRUNNER_POSTGRES_DATABASE=${FLOWRUNNER_POSTGRES_DATABASE:-flowrunner_test}
export FLOWRUNNER_POSTGRES_SSL_MODE=${FLOWRUNNER_POSTGRES_SSL_MODE:-disable}

echo "📋 PostgreSQL Configuration:"
echo "   Host: $FLOWRUNNER_POSTGRES_HOST"
echo "   Port: $FLOWRUNNER_POSTGRES_PORT"
echo "   User: $FLOWRUNNER_POSTGRES_USER"
echo "   Database: $FLOWRUNNER_POSTGRES_DATABASE"
echo "   SSL Mode: $FLOWRUNNER_POSTGRES_SSL_MODE"
echo ""

# Check if PostgreSQL is accessible
echo "🔍 Checking PostgreSQL connectivity..."
if command -v psql >/dev/null 2>&1; then
    if PGPASSWORD=$FLOWRUNNER_POSTGRES_PASSWORD psql -h $FLOWRUNNER_POSTGRES_HOST -p $FLOWRUNNER_POSTGRES_PORT -U $FLOWRUNNER_POSTGRES_USER -d postgres -c "SELECT 1;" >/dev/null 2>&1; then
        echo "✅ PostgreSQL is accessible"
    else
        echo "❌ Cannot connect to PostgreSQL. Please check your configuration."
        echo "   Make sure PostgreSQL is running and credentials are correct."
        exit 1
    fi
else
    echo "⚠️  psql not found, skipping connectivity check"
fi

# Create test database if it doesn't exist
echo "🗄️  Setting up test database..."
if command -v psql >/dev/null 2>&1; then
    PGPASSWORD=$FLOWRUNNER_POSTGRES_PASSWORD psql -h $FLOWRUNNER_POSTGRES_HOST -p $FLOWRUNNER_POSTGRES_PORT -U $FLOWRUNNER_POSTGRES_USER -d postgres -c "CREATE DATABASE $FLOWRUNNER_POSTGRES_DATABASE;" 2>/dev/null || echo "   Database already exists or creation failed (this is usually OK)"
fi

echo ""
echo "🧪 Running PostgreSQL Integration Tests..."
echo "=========================================="

# Run the simple branching test first
echo "1️⃣  Running Simple Branching Test..."
go test -v ./pkg/api -run TestWebSocketPostgreSQLIntegration_SimpleBranching -timeout 60s

if [ $? -eq 0 ]; then
    echo "✅ Simple Branching Test passed!"
    echo ""
    
    # Run the complex flow test
    echo "2️⃣  Running Complex Flow Test..."
    go test -v ./pkg/api -run TestWebSocketPostgreSQLIntegration_ComplexFlow -timeout 120s
    
    if [ $? -eq 0 ]; then
        echo "✅ Complex Flow Test passed!"
        echo ""
        echo "🎉 All PostgreSQL WebSocket Integration Tests passed!"
        echo ""
        echo "📊 Test Summary:"
        echo "   ✅ PostgreSQL backend integration"
        echo "   ✅ WebSocket real-time updates"
        echo "   ✅ Complex flow with branching"
        echo "   ✅ Parallel batch processing"
        echo "   ✅ Retry logic with backoff"
        echo "   ✅ Concurrent execution handling"
        echo ""
        echo "🚀 Your FlowRunner system is production-ready with PostgreSQL!"
    else
        echo "❌ Complex Flow Test failed"
        exit 1
    fi
else
    echo "❌ Simple Branching Test failed"
    exit 1
fi