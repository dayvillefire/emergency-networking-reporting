BINARY := en-report
RESPONDER_BINARY := en-responder-stats
SHIFT_BINARY := en-shift-report
WINDOWS_BINARY := $(BINARY).exe
PKG := ./cmd/report
RESPONDER_PKG := ./cmd/responder-stats
SHIFT_PKG := ./cmd/shift-report

.PHONY: all build build-responder build-shift-report windows clean

all: build build-responder build-shift-report windows

build:
	go build -o $(BINARY) $(PKG)

build-responder:
	go build -o $(RESPONDER_BINARY) $(RESPONDER_PKG)

build-shift-report:
	go build -o $(SHIFT_BINARY) $(SHIFT_PKG)

windows:
	GOOS=windows GOARCH=amd64 go build -o $(WINDOWS_BINARY) $(PKG)

clean:
	rm -f $(BINARY) $(RESPONDER_BINARY) $(SHIFT_BINARY) $(WINDOWS_BINARY)
