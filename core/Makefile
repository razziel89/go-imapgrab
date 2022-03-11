SHELL := /bin/bash

default: lint

.PHONY: setup
setup:
	go mod download
	go mod tidy

.PHONY: lint
lint:
	golangci-lint run .

test: .test.log

.test.log: go.* *.go
	set -o pipefail && \
		go test ./... | tee .test.log || \
		rm .test.log

coverage.html: go.* *.go
	go test -covermode=count -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

coverage_badge_report.out: go.* *.go
	go test -covermode=count -coverprofile=coverage.out
	go tool cover -func=coverage.out -o=coverage_badge_report.out

.PHONY: coverage
coverage: coverage.html
	xdg-open coverage.html
