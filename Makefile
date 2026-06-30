BINARY     := arrange
BIN_DIR    := ./bin
MODULE     := github.com/braswelljr/arrange
VERSION    := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS    := -ldflags "-X $(MODULE)/internal/common.Version=$(VERSION)"

HASGOCILINT := $(shell which golangci-lint 2>/dev/null)
ifdef HASGOCILINT
    GOLINT = golangci-lint
else
    GOLINT = bin/golangci-lint
endif

.PHONY: all
all: build

# ── Build ─────────────────────────────────────────────────────────────────────

.PHONY: build
build:
	go build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY) .

.PHONY: install
install:
	go install $(LDFLAGS) ./...

# ── Development ───────────────────────────────────────────────────────────────

.PHONY: run
run:
	go run $(LDFLAGS) main.go

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: download
download:
	go mod download

# ── Quality ───────────────────────────────────────────────────────────────────

.PHONY: test
test:
	go test -race ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: lint
lint:
	$(GOLINT) run

.PHONY: fix
fix:
	gofmt -s -w .
	goimports -w $(shell find . -type f -name '*.go' -not -path '*/vendor/*')

# ── Clean ─────────────────────────────────────────────────────────────────────

.PHONY: clean
clean:
	rm -rf $(BIN_DIR)
