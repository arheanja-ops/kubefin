BINARY := kubefin
PKG := ./cmd/kubefin
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X main.version=$(VERSION)

.PHONY: build test lint run tidy clean

build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) $(PKG)

test:
	go test ./...

lint:
	golangci-lint run

run: build
	./bin/$(BINARY)

tidy:
	go mod tidy

clean:
	rm -rf bin/
