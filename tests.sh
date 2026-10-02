#!/bin/sh
log=$(mktemp "${TMPDIR:-/tmp}/gotest.XXXXXX") || exit 1

go test -race -json -coverprofile=coverage.txt -v $(go list ./... | grep -v /examples/) 2>&1 | tee "$log" | gotestfmt
echo "test log: $log"

# generated mocks are not part of the coverage
grep -v '_mock\.go:' coverage.txt > coverage.tmp && mv coverage.tmp coverage.txt

go tool cover -func=coverage.txt
