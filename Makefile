SHELL := /bin/bash

default: lint

.PHONY: setup
setup:
	$(MAKE) -C ./core setup && \
	$(MAKE) -C ./cli setup

.PHONY: update-deps
update-deps:
	$(MAKE) -C ./core update-deps && \
	$(MAKE) -C ./cli update-deps

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
	$(MAKE) -C ./core lint && \
	$(MAKE) -C ./cli lint

test: .test.log

.test.log: */go.* */*.go
	$(MAKE) -C ./core test && \
	$(MAKE) -C ./cli test

