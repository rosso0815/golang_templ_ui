.PHONY: generate build test vet run-example

generate:
	templ generate ./...

build: generate
	go build ./...

test: generate
	go test ./...

vet: generate
	go vet ./...

run-example: generate
	go run ./examples/basic
