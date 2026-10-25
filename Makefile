BINARY := keimling
PKG    := ./src
LDFLAGS := -s -w
ARGS   ?=

.DEFAULT_GOAL := build
.PHONY: build test vet fmt check run clean

build:
	cd $(PKG) && CGO_ENABLED=0 go build -trimpath -ldflags='$(LDFLAGS)' -o ../$(BINARY) .

test:
	cd $(PKG) && go test ./...

vet:
	cd $(PKG) && go vet ./...

fmt:
	cd $(PKG) && gofmt -l -w .

check: fmt vet test

run: build
	./$(BINARY) $(ARGS)

clean:
	rm -f $(BINARY)
	cd $(PKG) && go clean -testcache >/dev/null 2>&1 || true
