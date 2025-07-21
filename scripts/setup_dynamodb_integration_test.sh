#!/bin/bash

# Setup script for DynamoDB integration tests
# This script helps set up the local DynamoDB environment for testing

set -e

echo "🚀 Setting up DynamoDB Integration Test Environment"

# Default values
DYNAMODB_ENDPOINT=${DYNAMODB_ENDPOINT:-http://localhost:8000}
DYNAMODB_REGION=${DYNAMODB_REGION:-us-east-1}
DYNAMODB_TABLE_PREFIX=${DYNAMODB_TABLE_PREFIX:-flowrunner_test_}

# Check if AWS CLI is installed
if ! command -v aws &> /dev/null; then
    echo "❌ AWS CLI is not installed. Please install it to continue."
    echo "   Visit: https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html"
    exit 1
fi

# Check if local DynamoDB is running
echo "🔍 Checking if local DynamoDB is running at $DYNAMODB_ENDPOINT..."
if ! aws dynamodb list-tables --endpoint-url $DYNAMODB_ENDPOINT --region $DYNAMODB_REGION &> /dev/null; then
    echo "❌ Local DynamoDB is not running at $DYNAMODB_ENDPOINT"
    echo "   Please start local DynamoDB using one of these methods:"
    echo ""
    echo "   🐳 Using Docker:"
    echo "      docker run -p 8000:8000 amazon/dynamodb-local"
    echo ""
    echo "   ☕ Using Java:"
    echo "      java -Djava.library.path=./DynamoDBLocal_lib -jar DynamoDBLocal.jar -sharedDb"
    echo ""
    exit 1
fi

echo "✅ Local DynamoDB is running at $DYNAMODB_ENDPOINT"

# Create required tables if they don't exist
echo "🗄️  Setting up DynamoDB tables with prefix: $DYNAMODB_TABLE_PREFIX"

# Account table
ACCOUNT_TABLE="${DYNAMODB_TABLE_PREFIX}accounts"
echo "   Creating $ACCOUNT_TABLE table if it doesn't exist..."
aws dynamodb create-table \
    --table-name $ACCOUNT_TABLE \
    --attribute-definitions AttributeName=id,AttributeType=S AttributeName=username,AttributeType=S \
    --key-schema AttributeName=id,KeyType=HASH \
    --global-secondary-indexes \
        "IndexName=username-index,KeySchema=[{AttributeName=username,KeyType=HASH}],Projection={ProjectionType=ALL}" \
    --billing-mode PAY_PER_REQUEST \
    --endpoint-url $DYNAMODB_ENDPOINT \
    --region $DYNAMODB_REGION &> /dev/null || echo "   Table $ACCOUNT_TABLE already exists"

# Flow table
FLOW_TABLE="${DYNAMODB_TABLE_PREFIX}flows"
echo "   Creating $FLOW_TABLE table if it doesn't exist..."
aws dynamodb create-table \
    --table-name $FLOW_TABLE \
    --attribute-definitions AttributeName=id,AttributeType=S AttributeName=account_id,AttributeType=S \
    --key-schema AttributeName=id,KeyType=HASH \
    --global-secondary-indexes \
        "IndexName=account-index,KeySchema=[{AttributeName=account_id,KeyType=HASH}],Projection={ProjectionType=ALL}" \
    --billing-mode PAY_PER_REQUEST \
    --endpoint-url $DYNAMODB_ENDPOINT \
    --region $DYNAMODB_REGION &> /dev/null || echo "   Table $FLOW_TABLE already exists"

# Execution table
EXECUTION_TABLE="${DYNAMODB_TABLE_PREFIX}executions"
echo "   Creating $EXECUTION_TABLE table if it doesn't exist..."
aws dynamodb create-table \
    --table-name $EXECUTION_TABLE \
    --attribute-definitions AttributeName=id,AttributeType=S AttributeName=flow_id,AttributeType=S AttributeName=account_id,AttributeType=S \
    --key-schema AttributeName=id,KeyType=HASH \
    --global-secondary-indexes \
        "IndexName=flow-index,KeySchema=[{AttributeName=flow_id,KeyType=HASH}],Projection={ProjectionType=ALL}" \
        "IndexName=account-index,KeySchema=[{AttributeName=account_id,KeyType=HASH}],Projection={ProjectionType=ALL}" \
    --billing-mode PAY_PER_REQUEST \
    --endpoint-url $DYNAMODB_ENDPOINT \
    --region $DYNAMODB_REGION &> /dev/null || echo "   Table $EXECUTION_TABLE already exists"

# Secret table
SECRET_TABLE="${DYNAMODB_TABLE_PREFIX}secrets"
echo "   Creating $SECRET_TABLE table if it doesn't exist..."
aws dynamodb create-table \
    --table-name $SECRET_TABLE \
    --attribute-definitions AttributeName=id,AttributeType=S AttributeName=account_id,AttributeType=S \
    --key-schema AttributeName=id,KeyType=HASH \
    --global-secondary-indexes \
        "IndexName=account-index,KeySchema=[{AttributeName=account_id,KeyType=HASH}],Projection={ProjectionType=ALL}" \
    --billing-mode PAY_PER_REQUEST \
    --endpoint-url $DYNAMODB_ENDPOINT \
    --region $DYNAMODB_REGION &> /dev/null || echo "   Table $SECRET_TABLE already exists"

echo "✅ DynamoDB tables are ready"

# Create .env file if it doesn't exist
if [ ! -f ".env" ]; then
    echo "📝 Creating .env file with DynamoDB configuration..."
    cat > .env << EOF
# DynamoDB Configuration
FLOWRUNNER_DYNAMODB_ENDPOINT=$DYNAMODB_ENDPOINT
FLOWRUNNER_DYNAMODB_REGION=$DYNAMODB_REGION
FLOWRUNNER_DYNAMODB_TABLE_PREFIX=$DYNAMODB_TABLE_PREFIX
EOF
    echo "✅ Created .env file"
else
    echo "⚠️  .env file already exists, not overwriting"
    echo "   Make sure it contains the correct DynamoDB configuration"
fi

echo ""
echo "🎉 DynamoDB Integration Test Environment is ready!"
echo ""
echo "📋 Configuration Summary:"
echo "   Endpoint: $DYNAMODB_ENDPOINT"
echo "   Region: $DYNAMODB_REGION"
echo "   Table Prefix: $DYNAMODB_TABLE_PREFIX"
echo ""
echo "🧪 To run the tests, use:"
echo "   ./scripts/test_dynamodb_integration.sh"
echo ""