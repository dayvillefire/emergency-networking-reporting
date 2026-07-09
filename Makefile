BINARY := en-report
RESPONDER_BINARY := en-responder-stats
SHIFT_BINARY := en-shift-report
MCP_BINARY := en-mcp
WINDOWS_BINARY := $(BINARY).exe
PKG := ./cmd/report
RESPONDER_PKG := ./cmd/responder-stats
SHIFT_PKG := ./cmd/shift-report
MCP_PKG := ./cmd/mcp

.PHONY: all build build-responder build-shift-report build-mcp windows clean

all: build build-responder build-shift-report build-mcp windows

build:
	go build -o $(BINARY) $(PKG)

build-responder:
	go build -o $(RESPONDER_BINARY) $(RESPONDER_PKG)

build-shift-report:
	go build -o $(SHIFT_BINARY) $(SHIFT_PKG)

build-mcp:
	go build -o $(MCP_BINARY) $(MCP_PKG)

windows:
	GOOS=windows GOARCH=amd64 go build -o $(WINDOWS_BINARY) $(PKG)

clean:
	rm -f $(BINARY) $(RESPONDER_BINARY) $(SHIFT_BINARY) $(MCP_BINARY) $(WINDOWS_BINARY)
