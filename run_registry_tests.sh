#!/bin/bash
go test -v ./pkg/registry/... > test_results.txt 2>&1
cat test_results.txt
