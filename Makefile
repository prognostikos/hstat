PREFIX ?= $(HOME)/.local
BINDIR ?= $(PREFIX)/bin

VERSION := $(shell grep 'const version' main.go | cut -d'"' -f2)
LDFLAGS := -s -w

.PHONY: build install uninstall test clean

build:
ifeq ($(RUNNING_IN_DEVCONTAINER),)
	@if command -v go >/dev/null 2>&1; then \
		go build -ldflags "$(LDFLAGS)" -o hstat .; \
	else \
		HOST_OS=$$(uname -s | tr '[:upper:]' '[:lower:]'); \
		HOST_ARCH=$$(uname -m); \
		case "$$HOST_ARCH" in x86_64) HOST_ARCH=amd64;; aarch64|arm64) HOST_ARCH=arm64;; esac; \
		devcontainer exec --workspace-folder . env GOOS=$$HOST_OS GOARCH=$$HOST_ARCH go build -ldflags "$(LDFLAGS)" -o hstat . < /dev/null; \
	fi
else
	go build -ldflags "$(LDFLAGS)" -o hstat .
endif

install: build
	@mkdir -p $(BINDIR)
	cp hstat $(BINDIR)/hstat
	@echo "Installed hstat v$(VERSION) to $(BINDIR)/hstat"

uninstall:
	rm -f $(BINDIR)/hstat
	@echo "Removed $(BINDIR)/hstat"

test:
	go test ./...

clean:
	rm -f hstat
