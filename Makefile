PREFIX ?= $(HOME)/.local/bin

.PHONY: build install test

build:
	go build -trimpath -o bin/apex ./cmd/apex

install: build
	mkdir -p $(PREFIX)
	cp bin/apex $(PREFIX)/apex

test:
	go test ./...
