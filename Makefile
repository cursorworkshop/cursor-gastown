.PHONY: build install clean test generate

BINARY := gt
BUILD_DIR := .

# Get version info for ldflags
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -X github.com/steveyegge/gastown/internal/cmd.Version=$(VERSION) \
           -X github.com/steveyegge/gastown/internal/cmd.Commit=$(COMMIT) \
           -X github.com/steveyegge/gastown/internal/cmd.BuildTime=$(BUILD_TIME)

# Disable CGO on macOS to avoid dyld hangs with Security.framework
# See: https://github.com/golang/go/issues/19734
ifeq ($(shell uname),Darwin)
    export CGO_ENABLED=0
endif

generate:
	go generate ./...

build: generate
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY) ./cmd/gt
ifeq ($(shell uname),Darwin)
	@# Sign with linker-signed signature (faster, no keychain access)
	@codesign -s - -f --entitlements /dev/null $(BUILD_DIR)/$(BINARY) 2>/dev/null || \
		codesign -s - -f $(BUILD_DIR)/$(BINARY) 2>/dev/null || true
	@# Remove quarantine attribute if present
	@xattr -d com.apple.quarantine $(BUILD_DIR)/$(BINARY) 2>/dev/null || true
	@echo "Built $(BINARY) for macOS (CGO disabled)"
endif

install: build
	cp $(BUILD_DIR)/$(BINARY) ~/.local/bin/$(BINARY)
	@echo "Installed to ~/.local/bin/$(BINARY)"

clean:
	rm -f $(BUILD_DIR)/$(BINARY)

test:
	go test ./...
