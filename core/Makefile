SHELL := /bin/bash

default: lint

.PHONY: setup
setup:
	go mod download
	go mod tidy

.PHONY: lint
lint:
	golangci-lint run .

full-test:
	SKIP_INTEGRATION_TESTS=0 $(MAKE) .test.log

test:
	SKIP_INTEGRATION_TESTS=1 $(MAKE) .test.log

.test.log: go.* *.go
	trap "rm .test.log" EXIT && \
		set -o pipefail && \
		go test | tee .test.log

coverage.html: go.* *.go
	go test -covermode=count -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

coverage_badge_report.out: go.* *.go
	go test -covermode=count -coverprofile=coverage.out
	go tool cover -func=coverage.out -o=coverage_badge_report.out

.PHONY: coverage
coverage: coverage.html
	xdg-open coverage.html
