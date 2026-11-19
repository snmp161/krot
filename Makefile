VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

GO_VERSION := $(shell awk '$$1 == "go" { print $$2 }' go.mod)
export GOTOOLCHAIN := go$(GO_VERSION)
GOFMT := $(shell GOTOOLCHAIN=$(GOTOOLCHAIN) go env GOROOT)/bin/gofmt

.PHONY: all build build-arm64 fmt-check vet test gosec staticcheck check integration

all: check build

build:
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/krot ./cmd/krot

build-arm64:
	CGO_ENABLED=0 GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/krot-arm64 ./cmd/krot

fmt-check:
	@out="$$($(GOFMT) -l .)"; if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi

vet:
	go vet ./...
	go vet -tags integration ./...

# -race требует cgo, поэтому CGO_ENABLED здесь не отключается.
test:
	go test -race ./...

gosec:
	go install github.com/securego/gosec/v2/cmd/gosec@v2.30.0
	"$$(go env GOPATH)/bin/gosec" ./...

staticcheck:
	go install honnef.co/go/tools/cmd/staticcheck@v0.8.2
	"$$(go env GOPATH)/bin/staticcheck" ./...

check: fmt-check vet test gosec staticcheck

integration:
	./scripts/integration-docker.sh
