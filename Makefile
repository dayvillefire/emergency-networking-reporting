BINARY := en-report
RESPONDER_BINARY := en-responder-stats
WINDOWS_BINARY := $(BINARY).exe
PKG := ./cmd/report
RESPONDER_PKG := ./cmd/responder-stats

.PHONY: all build build-responder windows clean

all: build build-responder windows

build:
	go build -o $(BINARY) $(PKG)

build-responder:
	go build -o $(RESPONDER_BINARY) $(RESPONDER_PKG)

windows:
	GOOS=windows GOARCH=amd64 go build -o $(WINDOWS_BINARY) $(PKG)

clean:
	rm -f $(BINARY) $(RESPONDER_BINARY) $(WINDOWS_BINARY)
