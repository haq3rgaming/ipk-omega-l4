SHELL := /bin/sh

BINARY := ipk-L4-scan
GO ?= go
CGO_ENABLED ?= 0
XLOGIN = "xpuskap00"
PACKED_FILES := LICENSE README.md CHANGELOG.md Makefile go.mod
PACKAGE_FILES := $(shell find . -type f -name "*.go")
FILES_TO_PACK := $(PACKED_FILES) $(PACKAGE_FILES)

.PHONY: all NixDevShellName test clean

all: $(BINARY)

NixDevShellName:
    @echo "go"

$(BINARY):
	CGO_ENABLED=$(CGO_ENABLED) $(GO) build -o $(BINARY) .
	chmod +x $(BINARY)

test: $(BINARY)
	$(GO) test .

clean:
	rm -f $(BINARY)

pack: clean
    zip -r $(XLOGIN).zip $(FILES_TO_PACK)