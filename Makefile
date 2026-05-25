BINARY  := a3k
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -X github.com/flysecurity/a3k/cmd.Version=$(VERSION) \
           -X github.com/flysecurity/a3k/cmd.Commit=$(COMMIT) \
           -X github.com/flysecurity/a3k/cmd.Date=$(DATE)
BUILD   := CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)"

.PHONY: build run clean install lint test snapshot release check-goreleaser

## build: compile binary for current platform
build:
	$(BUILD) -o $(BINARY) .

## install: install binary to $GOPATH/bin
install:
	$(BUILD) -o $(GOPATH)/bin/$(BINARY) .

## run: run without compiling (pass args with ARGS=...)
run:
	CGO_ENABLED=0 go run -trimpath -ldflags "$(LDFLAGS)" . $(ARGS)

## test: run all tests with race detector
test:
	go test -v -race ./...

## lint: run golangci-lint
lint:
	golangci-lint run ./...

## vuln: run govulncheck
vuln:
	go install golang.org/x/vuln/cmd/govulncheck@latest && govulncheck ./...

## snapshot: build a local release snapshot (no publish, no tag required)
snapshot:
	goreleaser release --snapshot --clean

## check-goreleaser: validate .goreleaser.yaml without building
check-goreleaser:
	goreleaser check

## release: tag and push to trigger the release pipeline
release: test lint
	@if [ -z "$(TAG)" ]; then echo "Usage: make release TAG=v1.0.0"; exit 1; fi
	git tag -a $(TAG) -m "Release $(TAG)"
	git push origin $(TAG)

## clean: remove compiled binary
clean:
	rm -f $(BINARY)
