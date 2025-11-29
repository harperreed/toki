.PHONY: build test install clean

build:
	go build -o toki ./cmd/toki

test:
	go test ./internal/... -v
	go test ./test/... -v

install:
	go install ./cmd/toki

clean:
	rm -f toki
	go clean

.DEFAULT_GOAL := build
