BINARY := en-report
WINDOWS_BINARY := $(BINARY).exe
PKG := ./cmd/report

.PHONY: all build windows clean

all: build windows

build:
	go build -o $(BINARY) $(PKG)

windows:
	GOOS=windows GOARCH=amd64 go build -o $(WINDOWS_BINARY) $(PKG)

clean:
	rm -f $(BINARY) $(WINDOWS_BINARY)
