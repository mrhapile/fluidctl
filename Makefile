BINARY_NAME=fluidctl
VERSION=v0.1.0
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildDate=$(DATE)"

.PHONY: all build install clean test

all: build

build:
	mkdir -p bin
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./fluidctl/cmd

install: build
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	sudo cp bin/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	@echo "Installation successful. Run '$(BINARY_NAME) version' to verify."

test:
	go test -v ./...

clean:
	rm -rf bin/
