#!/bin/bash

# Run the test and capture output
cd /Users/trevormartin/Projects/flowrunner
go test -count=1 -v ./pkg/registry/ -run TestFlowVersioningAndMetadata > test_output.log 2>&1

# Display the results
echo "Test Results:"
echo "=============="
if grep -q "FAIL" test_output.log; then
    echo "❌ Test FAILED"
    cat test_output.log
else
    echo "✅ Test PASSED"
    grep -A 100 "TestFlowVersioningAndMetadata" test_output.log
fi
