.PHONY: build install clean test run

BUILD_DIR=bin
BINARY=arch-agent
GO=go

build:
	@echo "Building $(BINARY)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build -o $(BUILD_DIR)/$(BINARY) ./cmd

install: build
	@echo "Installing $(BINARY)..."
	@install -d ~/.local/bin
	@install -Dm755 $(BUILD_DIR)/$(BINARY) ~/.local/bin/$(BINARY)
	@install -d ~/.config/arch-agent
	@install -Dm644 config.example.yaml ~/.config/arch-agent/config.example.yaml
	@echo "Installed to ~/.local/bin/$(BINARY)"

run:
	$(GO) run ./cmd

test:
	$(GO) test -v ./...

clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)

deps:
	$(GO) mod tidy
	$(GO) mod download
