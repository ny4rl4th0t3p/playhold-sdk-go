# The playhold version this SDK is built for: its wire code comes from that version's proto, and its tests run against a
# server on that version. testserver/go.mod must require the same one; make pin checks it.
PLAYHOLD_VERSION := v0.3.0

GOVULNCHECK_VERSION := v1.9.0
BUF_VERSION := v1.75.0
BUF := go run github.com/bufbuild/buf/cmd/buf@$(BUF_VERSION)
PROTO := https://github.com/ny4rl4th0t3p/playhold.git\#tag=$(PLAYHOLD_VERSION),subdir=proto

.DEFAULT_GOAL := help

.PHONY: help check build vet test lint vuln pin gen fmt tidy

help: ## Show the available targets
	@awk 'BEGIN {FS = ":.*## "} /^[a-z]+:.*## / {printf "  %-6s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

check: build vet test lint vuln pin ## Run every check

build: ## Build the SDK and the test server
	go build ./...
	cd testserver && go build -o /dev/null . # a single main package: without -o, go build would write the binary here

vet: ## Run go vet on both modules
	go vet ./...
	cd testserver && go vet ./...

test: ## Run the tests with the race detector, against a test server they build
	go test -count=1 -race ./...

lint: ## Run golangci-lint on both modules
	golangci-lint cache clean
	golangci-lint run ./...
	cd testserver && golangci-lint run --config ../.golangci.yml ./...

vuln: ## Check dependencies and the standard library for known vulnerabilities
	go run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) ./...

pin: ## Check testserver/go.mod requires PLAYHOLD_VERSION
	@got=$$(cd testserver && go list -m -f '{{.Version}}' github.com/ny4rl4th0t3p/playhold); \
	if [ "$$got" != "$(PLAYHOLD_VERSION)" ]; then \
		echo "testserver/go.mod requires playhold $$got, but PLAYHOLD_VERSION is $(PLAYHOLD_VERSION)"; exit 1; \
	fi

gen: ## Generate the wire code from playhold's proto at PLAYHOLD_VERSION
	$(BUF) generate '$(PROTO)' --path playhold/v1/frames.proto

fmt: ## Format the code with golangci-lint's formatters
	golangci-lint fmt ./...
	cd testserver && golangci-lint fmt --config ../.golangci.yml ./...

tidy: ## Tidy both modules
	go mod tidy
	cd testserver && go mod tidy