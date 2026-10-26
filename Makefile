BINARY := keimling
PKG    := ./src
LDFLAGS := -s -w
ARGS   ?=

UNAME_S := $(shell uname -s 2>/dev/null || echo unknown)

# Linux: pick WebKitGTK via pkg-config. macOS/Windows: CGO webview, no tag.
WEBKIT_TAG := $(shell \
	if [ "$(UNAME_S)" != "Linux" ]; then echo ""; \
	elif pkg-config --exists webkit2gtk-4.1 2>/dev/null; then echo webkitgtk_41; \
	elif pkg-config --exists webkitgtk-6.0 2>/dev/null; then echo webkitgtk_60; \
	elif pkg-config --exists webkit2gtk-4.0 2>/dev/null; then echo webkitgtk_40; \
	else echo ""; fi)

ifeq ($(UNAME_S),Linux)
  ifneq ($(WEBKIT_TAG),)
    ifeq ($(WEBKIT_TAG),webkitgtk_40)
      # Upstream fork has no 4.0 tag yet — build headless until patched.
      CGO_BUILD := 0
      BUILD_TAGS := nogui
    else
      CGO_BUILD := 1
      BUILD_TAGS := $(WEBKIT_TAG)
    endif
  else
    CGO_BUILD := 0
    BUILD_TAGS := nogui
  endif
else
  CGO_BUILD := 1
  BUILD_TAGS :=
endif

TAGS_FLAG := $(if $(BUILD_TAGS),-tags $(BUILD_TAGS),)

.DEFAULT_GOAL := build
.PHONY: build build-nogui test vet fmt check run clean detect

detect:
	@echo "UNAME=$(UNAME_S) CGO_BUILD=$(CGO_BUILD) TAGS=$(BUILD_TAGS)"

build:
	@echo "build CGO_ENABLED=$(CGO_BUILD) $(TAGS_FLAG)"
	cd $(PKG) && CGO_ENABLED=$(CGO_BUILD) go build -trimpath -ldflags='$(LDFLAGS)' $(TAGS_FLAG) -o ../$(BINARY) .

build-nogui:
	cd $(PKG) && CGO_ENABLED=0 go build -trimpath -ldflags='$(LDFLAGS)' -tags nogui -o ../$(BINARY) .

test:
	cd $(PKG) && CGO_ENABLED=0 go test -tags nogui ./...

vet:
	cd $(PKG) && CGO_ENABLED=0 go vet -tags nogui ./...

fmt:
	cd $(PKG) && gofmt -l -w .

check: fmt vet test

run: build
	./$(BINARY) $(ARGS)

clean:
	rm -f $(BINARY)
	cd $(PKG) && go clean -testcache >/dev/null 2>&1 || true
