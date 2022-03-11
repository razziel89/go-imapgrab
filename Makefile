SHELL := /bin/bash

default: lint

.PHONY: setup
setup:
	$(MAKE) -C ./cli setup && \
	$(MAKE) -C ./core setup

build: imapgrab

imapgrab: */*.go
	$(MAKE) -C cli build

.PHONY: lint
lint:
	$(MAKE) -C ./cli lint && \
	$(MAKE) -C ./core lint

test: .test.log

.test.log: */go.* */*.go
	$(MAKE) -C ./cli test && \
	$(MAKE) -C ./core test && \
	cat */test.log > test.log

