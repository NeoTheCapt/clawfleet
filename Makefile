.PHONY: build build-all build-server build-node build-fleetctl server agent web clean test run-server run-agent

OUT_DIR := dist/bin
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  := $(shell git rev-parse HEAD 2>/dev/null)
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
VERSION_PKG := github.com/NeoTheCapt/clawfleet/internal/version
LDFLAGS := -X $(VERSION_PKG).Version=$(VERSION) -X $(VERSION_PKG).Commit=$(COMMIT) -X $(VERSION_PKG).BuildTime=$(BUILD_TIME)

build: build-all

build-all: build-server build-node build-fleetctl

build-server:
	mkdir -p $(OUT_DIR)
	go build -ldflags "$(LDFLAGS) -X $(VERSION_PKG).Component=server" -o $(OUT_DIR)/clawfleet-server ./cmd/server

build-node:
	mkdir -p $(OUT_DIR)
	go build -ldflags "$(LDFLAGS) -X $(VERSION_PKG).Component=node" -o $(OUT_DIR)/clawfleet-node ./cmd/agent

build-fleetctl:
	mkdir -p $(OUT_DIR)
	go build -ldflags "$(LDFLAGS) -X $(VERSION_PKG).Component=fleetctl" -o $(OUT_DIR)/fleetctl ./cmd/fleetctl

# Legacy aliases
server: build-server
agent: build-node

web:
	cd web && npm install && npm run build

clean:
	find dist -type f -exec unlink {} \; 2>/dev/null || true
	find dist -depth -type d -exec rmdir {} \; 2>/dev/null || true
	find bin -type f -exec unlink {} \; 2>/dev/null || true
	find bin -depth -type d -exec rmdir {} \; 2>/dev/null || true
	unlink clawfleet.db 2>/dev/null || true
	unlink clawfleet.db-shm 2>/dev/null || true
	unlink clawfleet.db-wal 2>/dev/null || true
	unlink clawfleet-node 2>/dev/null || true
	unlink clawfleet-server 2>/dev/null || true
	unlink fleetctl 2>/dev/null || true

test:
	go test ./...

run-server:
	go run ./cmd/server

run-agent:
	go run ./cmd/agent --server http://localhost:8090
