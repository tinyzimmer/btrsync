SHELL := /bin/bash

BINARY_NAME ?= bin/btrsync

build:
	CGO_ENABLED=0 go build \
		-ldflags "-s -w -X main.version=$(shell git describe --tags --always --dirty)" \
		-o $(BINARY_NAME) \
		cmd/btrsync/main.go

install: build
	install -Dm755 $(BINARY_NAME) "$(shell go env GOPATH)/$(BINARY_NAME)"

generate:
	GO111MODULE=off go get golang.org/x/tools/cmd/stringer
	go generate ./...