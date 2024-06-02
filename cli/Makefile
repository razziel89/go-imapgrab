SHELL := /bin/bash

CGO_ENABLED ?= 0

default: lint

.PHONY: setup
setup:
	go mod download
	go mod tidy

.PHONY: update-deps
update-deps:
	go get -t -u
	$(MAKE) setup

.PHONY: build
build: go-imapgrab

go-imapgrab: *.go go.*
	CGO_ENABLED=$(CGO_ENABLED) go build -o go-imapgrab ./...

.PHONY: lint
lint:
	golangci-lint run .

test: .test.log

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
