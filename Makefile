## Draconiforge (forge) Makefile
## Simple binary you can run globally: `make install` -> `forge prework` (also `draconiforge`)

BIN       := draconiforge
BIN_SHORT := forge
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
	@echo " -> ./$(BIN) ready (./$(BIN) --help ; forge --help)"

DATADIR ?= $(PREFIX)/share
SHAREDIR ?= $(DATADIR)/draconiforge/scripts

install: build ## Install draconiforge globally to $(BINDIR) (default: ~/.local/bin)
	@echo "Installing $(BIN) to $(DESTDIR)$(BINDIR)/$(BIN)..."
	@mkdir -p $(DESTDIR)$(BINDIR)
	@install -m 0755 $(BIN) $(DESTDIR)$(BINDIR)/$(BIN)
	@ln -sf $(BIN) $(DESTDIR)$(BINDIR)/$(BIN_SHORT) 2>/dev/null || cp $(DESTDIR)$(BINDIR)/$(BIN) $(DESTDIR)$(BINDIR)/$(BIN_SHORT)
	@echo " -> installed: $(DESTDIR)$(BINDIR)/$(BIN) + symlink $(BIN_SHORT)"
	@if [ -d scripts ]; then \
		echo "Installing scripts to $(DESTDIR)$(SHAREDIR)..."; \
		mkdir -p $(DESTDIR)$(SHAREDIR); \
		install -m 0755 scripts/*.sh $(DESTDIR)$(SHAREDIR)/ 2>/dev/null || true; \
		install -m 0755 scripts/*.ps1 $(DESTDIR)$(SHAREDIR)/ 2>/dev/null || true; \
		echo " -> scripts: $$(ls -1 $(DESTDIR)$(SHAREDIR) 2>/dev/null | tr '\n' ' ')"; \
	fi
	@echo " -> version: $($(DESTDIR)$(BINDIR)/$(BIN) version 2>/dev/null || $(DESTDIR)$(BINDIR)/$(BIN) --help | head -1)"
	@if ! echo "$$PATH" | tr ':' '\n' | grep -qx "$(BINDIR)"; then \
		echo ""; echo "⚠  $(BINDIR) not in PATH. Add to your shell:"; \
		echo "   echo 'export PATH=\"\$$HOME/.local/bin:\$$PATH\"' >> ~/.zshrc  # or ~/.bashrc"; \
	fi

uninstall: ## Remove installed binary
	rm -f $(DESTDIR)$(BINDIR)/$(BIN) $(DESTDIR)$(BINDIR)/$(BIN_SHORT)
	rm -rf $(DESTDIR)$(SHAREDIR)
	@echo "Removed $(DESTDIR)$(BINDIR)/$(BIN) and $(BIN_SHORT) and $(DESTDIR)$(SHAREDIR)"

clean: ## Remove built binaries
	rm -f $(BIN) $(BIN_SHORT) df draconiforge

vet: ## Run go vet
	go vet ./...

fmt: ## Format code
	go fmt ./...

test: ## Run tests (if any)
	go test ./...

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-12s\033[0m %s\n",$$1,$$2}'
