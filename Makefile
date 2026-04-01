SHELL := /bin/sh

BINARY := ipk-omega-l4
GO ?= go
CGO_ENABLED ?= 0

.PHONY: all NixDevShellName test clean

all: $(BINARY)

NixDevShellName:
	@printf '%s\n' go

$(BINARY):
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build -o $(BINARY) .
	chmod +x $(BINARY)

test: $(BINARY)
	$(GO) test .

clean:
	rm -f $(BINARY)
