SHELL := /bin/bash

CGO_ENABLED ?= 0

default: lint

.PHONY: setup
setup:
	$(MAKE) -C ./cli setup && \
	$(MAKE) -C ./core setup

.PHONY: build
build: go-imapgrab

go-imapgrab: */*.go
	CGO_ENABLED=$(CGO_ENABLED) $(MAKE) -C cli build

.PHONY: lint
lint:
	$(MAKE) -C ./cli lint && \
	$(MAKE) -C ./core lint

test: .test.log

.test.log: */go.* */*.go
	$(MAKE) -C ./cli test && \
	$(MAKE) -C ./core test && \
	cat */test.log > test.log

