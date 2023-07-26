SHELL := /bin/bash

default: lint

.PHONY: setup
setup:
	$(MAKE) -C ./cli setup && \
	$(MAKE) -C ./core setup

.PHONY: build
build: go-imapgrab

.PHONY: build-cross-platform
build-cross-platform:
	cd ./cli && \
	CLIVERSION=local goreleaser build --clean --snapshot

go-imapgrab: */*.go
	$(MAKE) -C cli build

.PHONY: lint
lint:
	$(MAKE) -C ./cli lint && \
	$(MAKE) -C ./core lint

test: .test.log

.test.log: */go.* */*.go
	$(MAKE) -C ./cli test && \
	$(MAKE) -C ./core test

