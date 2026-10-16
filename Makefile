VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

.PHONY: all build build-arm64 fmt-check vet test check integration

all: check build

build:
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/krot ./cmd/krot

build-arm64:
	CGO_ENABLED=0 GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/krot-arm64 ./cmd/krot

fmt-check:
	@out="$$(gofmt -l .)"; if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi

vet:
	go vet ./...
	go vet -tags integration ./...

# -race требует cgo, поэтому CGO_ENABLED здесь не отключается.
test:
	go test -race ./...

check: fmt-check vet test

integration:
	./scripts/integration-docker.sh
