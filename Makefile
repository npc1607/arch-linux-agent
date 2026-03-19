.PHONY: build install clean test run dev

BUILD_DIR=bin
BINARY=arch-agent
GO=go

build:
	@echo "Building $(BINARY)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build -o $(BUILD_DIR)/$(BINARY) .
	@echo "✓ Built $(BUILD_DIR)/$(BINARY)"

install: build
	@echo "Installing $(BINARY)..."
	@install -d ~/.local/bin
	@install -Dm755 $(BUILD_DIR)/$(BINARY) ~/.local/bin/$(BINARY)
	@install -d ~/.config/arch-agent
	@if [ ! -f ~/.config/arch-agent/config.yaml ]; then \
		install -Dm644 config.example.yaml ~/.config/arch-agent/config.example.yaml; \
		echo "✓ Config template installed to ~/.config/arch-agent/config.example.yaml"; \
	fi
	@echo "✓ Installed to ~/.local/bin/$(BINARY)"

uninstall:
	@echo "Uninstalling $(BINARY)..."
	@rm -f ~/.local/bin/$(BINARY)
	@echo "✓ Uninstalled"

run:
	$(GO) run .

dev: build
	@./$(BUILD_DIR)/$(BINARY) --help

test:
	$(GO) test -v ./...

test-chat:
	@echo "Testing chat command (requires OPENAI_API_KEY)..."
	@if [ -z "$$OPENAI_API_KEY" ]; then \
		echo "Error: OPENAI_API_KEY not set"; \
		exit 1; \
	fi
	@echo "test" | timeout 5 ./$(BUILD_DIR)/$(BINARY) chat || true

clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)

deps:
	$(GO) mod tidy
	$(GO) mod download
	@echo "✓ Dependencies updated"

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

lint: fmt vet
	@echo "✓ Linting complete"

check: lint test
	@echo "✓ All checks passed"
