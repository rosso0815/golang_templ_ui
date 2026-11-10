GOBIN ?= $(CURDIR)/.gobin
export GOBIN
export PATH := $(GOBIN):$(PATH)
TEMPL_VERSION := $(shell go list -m -f '{{.Version}}' github.com/a-h/templ)

.PHONY: tools generate build test vet run-example

tools:
	go install github.com/a-h/templ/cmd/templ@$(TEMPL_VERSION)

generate: tools
	templ generate ./...

build: generate
	go build ./...

test: generate
	go test ./...

vet: generate
	go vet ./...

run-example: generate
	go run ./examples/basic
