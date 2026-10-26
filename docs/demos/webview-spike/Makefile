# Auto-pick Linux WebKitGTK backend; macOS/Windows need no tags.
#
#   make build          # detect + build
#   make run            # build and open WebView
#   make size           # print binary bytes
#   make baseline-size  # plain Go HTTP binary without webview (compare)

ROOT := $(abspath ../../..)
SPIKE_BIN := spike-webview
BASE_BIN := spike-http-only

.PHONY: build run size baseline-size detect clean

detect:
	@if pkg-config --exists webkit2gtk-4.1; then \
		echo "WEBKIT_TAG=webkitgtk_41  (webkit2gtk-4.1)"; \
	elif pkg-config --exists webkit2gtk-4.0; then \
		echo "WEBKIT_TAG=webkitgtk_40  (webkit2gtk-4.0 — needs local patch; see README)"; \
	elif pkg-config --exists webkitgtk-6.0; then \
		echo "WEBKIT_TAG=webkitgtk_60  (webkitgtk-6.0)"; \
	else \
		echo "No WebKitGTK pkg-config module found (ok on macOS/Windows builds)"; \
	fi

# Prefer 4.1, then 4.0 (via replace/patch), then 6.0.
WEBKIT_TAG := $(shell \
	if pkg-config --exists webkit2gtk-4.1 2>/dev/null; then echo webkitgtk_41; \
	elif pkg-config --exists webkit2gtk-4.0 2>/dev/null; then echo webkitgtk_40; \
	elif pkg-config --exists webkitgtk-6.0 2>/dev/null; then echo webkitgtk_60; \
	else echo ""; fi)

TAGS_FLAG := $(if $(WEBKIT_TAG),-tags $(WEBKIT_TAG),)

LDFLAGS ?= -s -w

build:
	@if [ "$(WEBKIT_TAG)" = "webkitgtk_40" ]; then \
		echo "webkit2gtk-4.0 detected, but samcharles93/webview_go has no webkitgtk_40 tag yet."; \
		echo "See patches/webkitgtk_40_unix.go — use a module replace, then: make WEBKIT_TAG=webkitgtk_40 build"; \
		exit 1; \
	fi
	@echo "Building with $(TAGS_FLAG) ldflags='$(LDFLAGS)'"
	CGO_ENABLED=1 go build -trimpath -ldflags='$(LDFLAGS)' $(TAGS_FLAG) -o $(SPIKE_BIN) .
	@ls -la $(SPIKE_BIN)
	@wc -c $(SPIKE_BIN)

run: build
	./$(SPIKE_BIN)

size: build
	@echo -n "spike-webview: "; wc -c < $(SPIKE_BIN)
	@echo -n "keimling now:  "; wc -c < $(ROOT)/keimling 2>/dev/null || echo "(no root binary)"

# Same HTTP page, no webview dependency — size floor for comparison.
baseline-size:
	CGO_ENABLED=0 go build -trimpath -ldflags='$(LDFLAGS)' -o $(BASE_BIN) ./cmd/httponly
	@echo -n "spike-http-only (no webview): "; wc -c < $(BASE_BIN)

clean:
	rm -f $(SPIKE_BIN) $(BASE_BIN)
