.PHONY: run run-mock build lint tidy

run:
	go run ./cmd/spindle

run-mock:
	go run ./cmd/spindle --mock

build:
	go build -o spindle ./cmd/spindle

lint:
	go vet ./... && staticcheck ./...

tidy:
	go mod tidy
