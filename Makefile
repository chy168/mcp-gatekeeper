BINARY  := mcp-gatekeeper
BINARY2 := mcp-gatekeeper-secret
MODULE  := github.com/chy168/mcp-gatekeeper
CMD     := ./cmd/mcp-gatekeeper
CMD2    := ./cmd/mcp-gatekeeper-secret

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

.PHONY: build test clean cross

build:
	go build $(LDFLAGS) -o bin/$(BINARY) $(CMD)
	go build $(LDFLAGS) -o bin/$(BINARY2) $(CMD2)

test:
	go test ./...

cross:
	GOOS=linux   GOARCH=amd64  go build $(LDFLAGS) -o dist/$(BINARY)-linux-amd64   $(CMD)
	GOOS=linux   GOARCH=arm64  go build $(LDFLAGS) -o dist/$(BINARY)-linux-arm64   $(CMD)
	GOOS=darwin  GOARCH=amd64  go build $(LDFLAGS) -o dist/$(BINARY)-darwin-amd64  $(CMD)
	GOOS=darwin  GOARCH=arm64  go build $(LDFLAGS) -o dist/$(BINARY)-darwin-arm64  $(CMD)
	GOOS=windows GOARCH=amd64  go build $(LDFLAGS) -o dist/$(BINARY)-windows-amd64.exe $(CMD)
	GOOS=linux   GOARCH=amd64  go build $(LDFLAGS) -o dist/$(BINARY2)-linux-amd64   $(CMD2)
	GOOS=linux   GOARCH=arm64  go build $(LDFLAGS) -o dist/$(BINARY2)-linux-arm64   $(CMD2)
	GOOS=darwin  GOARCH=amd64  go build $(LDFLAGS) -o dist/$(BINARY2)-darwin-amd64  $(CMD2)
	GOOS=darwin  GOARCH=arm64  go build $(LDFLAGS) -o dist/$(BINARY2)-darwin-arm64  $(CMD2)
	GOOS=windows GOARCH=amd64  go build $(LDFLAGS) -o dist/$(BINARY2)-windows-amd64.exe $(CMD2)

clean:
	rm -rf bin/ dist/
