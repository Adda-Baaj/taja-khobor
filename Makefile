.PHONY: build run test tidy

build:
	go build ./...

run:
	go run ./cmd/collector

test:
	go test ./...

tidy:
	go mod tidy
