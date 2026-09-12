## Draconiforge (df) Makefile
## Simple binary you can run globally: `make install` -> `draconiforge prework` (alias: df)

BIN       := draconiforge
BIN_SHORT := df
PKG       := ./cmd/goforge-cli

# Install location - respects PREFIX/BINDIR, defaults to user-local (no sudo needed)
PREFIX  ?= $(HOME)/.local
BINDIR  ?= $(PREFIX)/bin
DESTDIR ?=

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)

LDFLAGS := -s -w -X github.com/draconis-engineering/draconiforge/internal/cli.version=$(VERSION)

.PHONY: build install uninstall clean vet fmt test help

build: ## Build ./draconiforge binary locally
	@echo "Building $(BIN) $(VERSION) ($(COMMIT))..."
	go build -ldflags "$(LDFLAGS)" -o $(BIN) $(PKG)
	@echo " -> ./$(BIN) ready (./$(BIN) --help ; ./$(BIN) prework --help)"

install: build ## Install draconiforge globally to $(BINDIR) (default: ~/.local/bin)
	@echo "Installing $(BIN) to $(DESTDIR)$(BINDIR)/$(BIN)..."
	@mkdir -p $(DESTDIR)$(BINDIR)
	@install -m 0755 $(BIN) $(DESTDIR)$(BINDIR)/$(BIN)
	@ln -sf $(BIN) $(DESTDIR)$(BINDIR)/$(BIN_SHORT) 2>/dev/null || cp $(DESTDIR)$(BINDIR)/$(BIN) $(DESTDIR)$(BINDIR)/$(BIN_SHORT)
	@echo " -> installed: $(DESTDIR)$(BINDIR)/$(BIN) + symlink $(BIN_SHORT)"
	@echo " -> version: $$($(DESTDIR)$(BINDIR)/$(BIN) version 2>/dev/null || $(DESTDIR)$(BINDIR)/$(BIN) --help | head -1)"
	@if ! echo "$$PATH" | tr ':' '\n' | grep -qx "$(BINDIR)"; then \
		echo ""; echo "⚠  $(BINDIR) not in PATH. Add to your shell:"; \
		echo "   echo 'export PATH=\"\$$HOME/.local/bin:\$$PATH\"' >> ~/.zshrc  # or ~/.bashrc"; \
	fi
	@if which df >/dev/null 2>&1 && [ "$$(which df)" != "$(DESTDIR)$(BINDIR)/$(BIN_SHORT)" ] && [ "$$(which df)" != "$(BINDIR)/$(BIN_SHORT)" ]; then \
		echo ""; echo "⚠  'df' collides with system /usr/bin/df (disk free)."; \
		echo "   Your PATH has $(BINDIR) AFTER /usr/bin, so 'df' runs the system tool."; \
		echo "   Fix: put ~/.local/bin FIRST:"; \
		echo "     export PATH=\"\$$HOME/.local/bin:\$$PATH\""; \
		echo "   Then 'df prework' will work. Until then use 'draconiforge prework' (always works)."; \
		echo "   Current: which df = $$(which df)"; \
		echo "            which draconiforge = $$(which draconiforge 2>/dev/null || echo $(DESTDIR)$(BINDIR)/$(BIN))"; \
	fi

uninstall: ## Remove installed binary
	rm -f $(DESTDIR)$(BINDIR)/$(BIN) $(DESTDIR)$(BINDIR)/$(BIN_SHORT)
	@echo "Removed $(DESTDIR)$(BINDIR)/$(BIN) and $(BIN_SHORT)"

clean: ## Remove built binaries
	rm -f $(BIN) $(BIN_SHORT) df

vet: ## Run go vet
	go vet ./...

fmt: ## Format code
	go fmt ./...

test: ## Run tests (if any)
	go test ./...

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-12s\033[0m %s\n",$$1,$$2}'
