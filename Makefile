# ==================================================================================================
# long shell commands
# ==================================================================================================

GOLANGCI_LINT := go run github.com/golangci/golangci-lint/cmd/golangci-lint

# ==================================================================================================
# all
# ==================================================================================================

.PHONY: all
all: install fix lint test build

# ==================================================================================================
# fix, lint and test
# ==================================================================================================

PACKAGES_PREFIX := github.com/mikaelmello/pingo

.PHONY: install
install:
	go get ./...

.PHONY: fix
fix:
	go mod tidy

.PHONY: lint
lint:
	$(GOLANGCI_LINT) run

.PHONY: test
test: 
	go test ./... -cover

.PHONY: build
build:
	go build