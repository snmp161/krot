VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
SOURCE_DATE_EPOCH ?= $(shell git show -s --format=%ct HEAD 2>/dev/null || echo 0)
DATE    ?= $(shell date -u -d "@$(SOURCE_DATE_EPOCH)" +%Y-%m-%dT%H:%M:%SZ)
export SOURCE_DATE_EPOCH
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

GO_VERSION := $(shell awk '$$1 == "go" { print $$2 }' go.mod)
export GOTOOLCHAIN := go$(GO_VERSION)
GOFMT := $(shell GOTOOLCHAIN=$(GOTOOLCHAIN) go env GOROOT)/bin/gofmt
# koalaman/shellcheck v0.11.0.
SHELLCHECK_IMAGE := koalaman/shellcheck@sha256:61862eba1fcf09a484ebcc6feea46f1782532571a34ed51fedf90dd25f925a8d

.PHONY: all build build-arm64 fmt-check vet mod-check test fuzz-smoke vulncheck gosec staticcheck shellcheck actionlint changelog changelog-check repro-check check integration package-preflight

all: check build

build:
	CGO_ENABLED=0 go build -buildvcs=false -trimpath -ldflags "$(LDFLAGS)" -o bin/krot ./cmd/krot

build-arm64:
	CGO_ENABLED=0 GOARCH=arm64 go build -buildvcs=false -trimpath -ldflags "$(LDFLAGS)" -o bin/krot-arm64 ./cmd/krot

fmt-check:
	@out="$$($(GOFMT) -l .)"; if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi

vet:
	go vet ./...
	go vet -tags integration ./...

mod-check:
	go mod tidy -diff
	go mod verify

# -race требует cgo, поэтому CGO_ENABLED здесь не отключается.
test:
	go test -race -count=1 -shuffle=on -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
	@total="$$(go tool cover -func=coverage.out | awk '/^total:/ {gsub(/%/, "", $$3); print $$3}')"; \
		echo "total coverage: $${total}%"; \
		awk -v t="$$total" 'BEGIN { exit !(t >= 90.0) }'

fuzz-smoke:
	./scripts/fuzz-smoke.sh

vulncheck:
	go install golang.org/x/vuln/cmd/govulncheck@v1.9.0
	"$$(go env GOPATH)/bin/govulncheck" ./...

gosec:
	go install github.com/securego/gosec/v2/cmd/gosec@v2.30.0
	"$$(go env GOPATH)/bin/gosec" ./...

staticcheck:
	go install honnef.co/go/tools/cmd/staticcheck@v0.8.2
	"$$(go env GOPATH)/bin/staticcheck" ./...

shellcheck:
	docker run --rm -v "$(CURDIR):/repo:ro" -w /repo "$(SHELLCHECK_IMAGE)" scripts/*.sh

actionlint:
	go install github.com/rhysd/actionlint/cmd/actionlint@v1.7.12
	"$$(go env GOPATH)/bin/actionlint" .github/workflows/*.yml

changelog:
	go install github.com/goreleaser/chglog/cmd/chglog@v0.7.6
	./scripts/release-notes.sh

changelog-check:
	go install github.com/goreleaser/chglog/cmd/chglog@v0.7.6
	./scripts/release-notes.sh --check

repro-check:
	go install github.com/goreleaser/nfpm/v2/cmd/nfpm@v2.48.0
	./scripts/reproducible-check.sh

check: fmt-check vet mod-check test vulncheck fuzz-smoke gosec staticcheck shellcheck actionlint changelog-check repro-check

integration:
	./scripts/integration-docker.sh

package-preflight:
	go install github.com/goreleaser/nfpm/v2/cmd/nfpm@v2.48.0
	./scripts/package-preflight.sh
