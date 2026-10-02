.PHONY: build test install

build:
	go build -o tc ./cmd/tc

test:
	go test ./...

install: build
	mkdir -p $(HOME)/.local/bin
	cp tc $(HOME)/.local/bin/tc
