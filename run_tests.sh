#!/bin/bash

# Load environment variables (if present)
if [ -f .env ]; then
  export $(grep -v '^#' .env | xargs)
fi

# Run core package tests (storage)
go test -count=1 -v ./pkg/storage/...

# Run API package tests, including E2E tool-calls and email summaries
# Requires network and these env vars: OPENAI_API_KEY, GMAIL_USERNAME, GMAIL_PASSWORD, EMAIL_RECIPIENT
go test -count=1 -v ./pkg/api
