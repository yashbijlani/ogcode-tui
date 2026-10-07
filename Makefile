# Optional gitignored .env (e.g. OGX_APP_SECRET=...). Loaded so a local build
# bakes in the same OGX client-assertion secret the official release does; a
# missing .env is fine (the binary then signs nothing rather than failing).
-include .env
export OGX_APP_SECRET

.PHONY: dev build clean install dev-blog build-blog build-tui install-tui

dev:
	@echo "Starting Go server on :9595..."
	go run . serve --port 9595

dev-web:
	@echo "Starting Vite dev server on :5173..."
	cd web && npm run dev

build-web:
	cd web && npm install --legacy-peer-deps --cache /tmp/npm-cache && npm run build

dev-blog:
	@echo "Starting Astro dev server on :4321 (blog at /blog/)..."
	cd blog && npm install --cache /tmp/npm-cache && npm run dev

build-blog:
	cd blog && npm install --cache /tmp/npm-cache && npm run build

build-server:
	$(eval VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || node -p "require('./web/package.json').version" 2>/dev/null || echo "dev"))
	CGO_ENABLED=1 go build \
		-ldflags "-X github.com/prasenjeet-symon/ogcode/internal/version.Version=$(VERSION) -X github.com/prasenjeet-symon/ogcode/internal/provider.OGXAppSecret=$(OGX_APP_SECRET)" \
		-o ogcode .

# Standalone in-process TUI (cmd/ogcode-tui). Bakes in the same VERSION and
# OGX client-assertion secret as build-server — set OGX_APP_SECRET in the
# environment, in the gitignored .env loaded above, or on the command line.
# The recipe is quiet so the secret in the ldflags never echoes to the log.
build-tui:
	$(eval VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || node -p "require('./web/package.json').version" 2>/dev/null || echo "dev"))
	@CGO_ENABLED=1 go build \
		-ldflags "-X github.com/prasenjeet-symon/ogcode/internal/version.Version=$(VERSION) -X github.com/prasenjeet-symon/ogcode/internal/provider.OGXAppSecret=$(OGX_APP_SECRET)" \
		-o ogcode-tui ./cmd/ogcode-tui
	@echo "Build complete: ./ogcode-tui"

# Same macOS stale-code-signature workaround as install.
install-tui: build-tui
	mkdir -p $(HOME)/.local/bin
	rm -f $(HOME)/.local/bin/ogcode-tui
	cp ogcode-tui $(HOME)/.local/bin/ogcode-tui
	@if [ "$$(uname -s)" = "Darwin" ] && command -v codesign >/dev/null 2>&1; then \
		codesign --force --sign - $(HOME)/.local/bin/ogcode-tui && echo "Re-signed ogcode-tui (macOS ad-hoc)" \
			|| echo "warning: codesign failed; if ogcode-tui is Killed:9, run: codesign --force --sign - $(HOME)/.local/bin/ogcode-tui"; \
	fi
	@echo "Installed to $(HOME)/.local/bin/ogcode-tui"

build: build-web build-server
	@echo "Build complete: ./ogcode"

# On macOS, replacing the binary in place leaves the kernel with a stale cached
# code signature for the path and it SIGKILLs the new process ("Killed: 9").
# Removing the old file first (fresh inode) and re-signing ad-hoc avoids it.
install: build
	mkdir -p $(HOME)/.local/bin
	rm -f $(HOME)/.local/bin/ogcode
	cp ogcode $(HOME)/.local/bin/ogcode
	@if [ "$$(uname -s)" = "Darwin" ] && command -v codesign >/dev/null 2>&1; then \
		codesign --force --sign - $(HOME)/.local/bin/ogcode && echo "Re-signed ogcode (macOS ad-hoc)" \
			|| echo "warning: codesign failed; if ogcode is Killed:9, run: codesign --force --sign - $(HOME)/.local/bin/ogcode"; \
	fi
	@echo "Installed to $(HOME)/.local/bin/ogcode"
	@echo "Web search is built in — no Node.js or Chromium needed."

clean:
	rm -f ogcode ogcode-tui
	rm -rf web/dist web/node_modules web/.solid
	rm -rf blog/node_modules blog/.astro docs/blog
	rm -rf .ogcode

test:
	go test ./...