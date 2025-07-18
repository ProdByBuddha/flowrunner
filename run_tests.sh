#!/bin/bash

# Load environment variables
export $(grep -v '^#' .env | xargs)

# Run tests
go test -v ./pkg/storage/...