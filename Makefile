VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS  = -s -w -X main.version=$(VERSION)
BINARY   = kontango

.PHONY: build build-all test lint clean

## build: Compile for the current platform
build:
	go build -ldflags '$(LDFLAGS)' -o build/$(BINARY) ./cmd/kontango/

## build-all: Cross-compile for all supported platforms
build-all:
	@mkdir -p build
	GOOS=linux   GOARCH=amd64 CGO_ENABLED=0 go build -ldflags '$(LDFLAGS)' -o build/$(BINARY)-linux-amd64       ./cmd/kontango/
	GOOS=linux   GOARCH=arm64 CGO_ENABLED=0 go build -ldflags '$(LDFLAGS)' -o build/$(BINARY)-linux-arm64       ./cmd/kontango/
	GOOS=linux   GOARCH=arm   CGO_ENABLED=0 go build -ldflags '$(LDFLAGS)' -o build/$(BINARY)-linux-arm         ./cmd/kontango/
	GOOS=darwin  GOARCH=amd64 CGO_ENABLED=0 go build -ldflags '$(LDFLAGS)' -o build/$(BINARY)-darwin-amd64      ./cmd/kontango/
	GOOS=darwin  GOARCH=arm64 CGO_ENABLED=0 go build -ldflags '$(LDFLAGS)' -o build/$(BINARY)-darwin-arm64      ./cmd/kontango/
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags '$(LDFLAGS)' -o build/$(BINARY)-windows-amd64.exe ./cmd/kontango/
	@echo "Built $(VERSION) for all platforms"

## test: Run all tests
test:
	go test ./...

## lint: Run go vet
lint:
	go vet ./...

## clean: Remove build artifacts
clean:
	rm -rf build/

## version: Print the version
version:
	@echo $(VERSION)

## help: Show this help
help:
	@grep -E '^## ' Makefile | sed 's/## //' | column -t -s ':'
