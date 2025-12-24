PREFIX ?= $(HOME)/.local
BINDIR ?= $(PREFIX)/bin

VERSION := $(shell grep 'const version' main.go | cut -d'"' -f2)
LDFLAGS := -s -w

.PHONY: build install uninstall test clean

build:
	go build -ldflags "$(LDFLAGS)" -o hstat .

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
