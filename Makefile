.PHONY: build build-universal build-linux-amd64 build-linux-arm64 build-tray-linux build-app-macos build-snap build-aur-srcinfo test lint clean install

BINARY := dotctl
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)
LDFLAGS := -ldflags "-X github.com/felipe-veas/dotctl/internal/version.Version=$(VERSION)"

build:
	go build $(LDFLAGS) -o bin/$(BINARY) ./cmd/dotctl

build-universal:
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY)-arm64 ./cmd/dotctl
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY)-amd64 ./cmd/dotctl
	lipo -create bin/$(BINARY)-arm64 bin/$(BINARY)-amd64 -output bin/$(BINARY)-universal
	rm bin/$(BINARY)-arm64 bin/$(BINARY)-amd64

build-linux-amd64:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY)-linux-amd64 ./cmd/dotctl

build-linux-arm64:
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY)-linux-arm64 ./cmd/dotctl

build-tray-linux:
	./scripts/build-tray-linux.sh

build-app-macos:
	./scripts/build-app-macos.sh

build-snap:
	./scripts/build-snap.sh

build-aur-srcinfo:
	./scripts/generate-aur-srcinfo.sh

test:
	go test ./... -race -count=1

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/

install: build
	cp bin/$(BINARY) /usr/local/bin/$(BINARY)
