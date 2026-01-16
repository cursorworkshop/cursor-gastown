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

# macOS: Disable CGO and avoid Security.framework hangs
# The x509usefallbackroots=1 tells Go to use embedded cert roots instead of
# calling Security.framework, which can hang on some macOS systems.
# See: https://github.com/golang/go/issues/19734
ifeq ($(shell uname),Darwin)
    export CGO_ENABLED=0
    export GODEBUG=x509usefallbackroots=1
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
ifeq ($(shell uname),Darwin)
	@# On macOS, install binary as gt.bin and create wrapper script
	@# The wrapper sets GODEBUG=x509usefallbackroots=1 to avoid Security.framework hangs
	cp $(BUILD_DIR)/$(BINARY) ~/.local/bin/$(BINARY).bin
	@echo '#!/bin/bash' > ~/.local/bin/$(BINARY)
	@echo '# Wrapper to avoid macOS Security.framework hangs' >> ~/.local/bin/$(BINARY)
	@echo 'export GODEBUG=x509usefallbackroots=1' >> ~/.local/bin/$(BINARY)
	@echo 'exec ~/.local/bin/$(BINARY).bin "$$@"' >> ~/.local/bin/$(BINARY)
	@chmod +x ~/.local/bin/$(BINARY) ~/.local/bin/$(BINARY).bin
	@echo "Installed to ~/.local/bin/$(BINARY) (with wrapper for macOS)"
else
	cp $(BUILD_DIR)/$(BINARY) ~/.local/bin/$(BINARY)
	@echo "Installed to ~/.local/bin/$(BINARY)"
endif

clean:
	rm -f $(BUILD_DIR)/$(BINARY)

test:
	go test ./...
