.PHONY: build test install

build:
	go build -o tc ./cmd/tc

test:
	go test ./...

install: build
	mkdir -p $(HOME)/bin
	cp tc $(HOME)/bin/tc
