VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "v2.8.2")
LDFLAGS = -s -w -X main.Version=$(VERSION)
DIST_DIR = dist

.PHONY: all build static cross-compile package test clean install uninstall

all: build

## build: Build standard local binary
build:
	go build -ldflags="$(LDFLAGS)" -o bush main.go

## static: Build 100% statically linked local binary (zero Cgo/glibc dependency)
static:
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bush main.go

## cross-compile: Cross-compile static binaries for all supported platforms into $(DIST_DIR)
cross-compile:
	@mkdir -p $(DIST_DIR)
	@echo "==> Building Linux amd64..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/bush-linux-amd64 main.go
	@echo "==> Building Linux arm64..."
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/bush-linux-arm64 main.go
	@echo "==> Building Linux armv7..."
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/bush-linux-armv7 main.go
	@echo "==> Building macOS amd64..."
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/bush-darwin-amd64 main.go
	@echo "==> Building macOS arm64..."
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/bush-darwin-arm64 main.go
	@echo "==> Done. Binaries saved in $(DIST_DIR)/"

## package: Build and bundle release tarballs with checksums
package:
	@bash scripts/build-releases.sh

## test: Run all project unit tests
test:
	go test -v ./...

## install: Install Bush to /usr/local/bin (requires write permission or sudo)
install: static
	@if [ -w "/usr/local/bin" ]; then \
		install -m 755 bush /usr/local/bin/bush; \
	else \
		echo "Installing to /usr/local/bin requires sudo..."; \
		sudo install -m 755 bush /usr/local/bin/bush; \
	fi
	@echo "Bush successfully installed to /usr/local/bin/bush"

## uninstall: Remove Bush from /usr/local/bin
uninstall:
	@sudo rm -f /usr/local/bin/bush
	@echo "Bush removed from /usr/local/bin/bush"

## clean: Remove built binaries and dist artifacts
clean:
	rm -rf bush $(DIST_DIR) build-*
