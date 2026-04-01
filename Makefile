SHELL := /bin/sh

BINARY := ipk-L4-scan
GO ?= go
CGO_ENABLED ?= 0

.PHONY: all NixDevShellName clean

all: $(BINARY)

NixDevShellName:
	@printf '%s\n' go

$(BINARY):
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build -o $(BINARY) .
	chmod +x $(BINARY)

clean:
	rm -f $(BINARY)
