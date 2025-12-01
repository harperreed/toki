.PHONY: build test test-race test-coverage install clean

build:
	go build -o toki ./cmd/toki

test:
	go test ./internal/... -v
	go test ./test/... -v

test-race:
	go test -race ./...

test-coverage:
	go test -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html

install:
	go install ./cmd/toki

clean:
	rm -f toki coverage.out coverage.html
	go clean

.DEFAULT_GOAL := build
