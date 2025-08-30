#!/bin/bash
cd /Users/trevormartin/Projects/flowrunner
go test -v ./pkg/registry/ -run TestFlowVersioning > test_output.txt 2>&1
cat test_output.txt
