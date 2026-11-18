# The playhold version this SDK is built for, the same v0.X.0 as the SDK's own: go.mod requires playhold's wire module at
# it, for the frames, and testserver/go.mod requires playhold at it, for the tests' server. make pin checks both.
PLAYHOLD_VERSION := v0.4.0

GOVULNCHECK_VERSION := v1.9.0

.DEFAULT_GOAL := help

.PHONY: help check build vet test lint vuln pin fmt tidy

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

pin: ## Check go.mod requires playhold/wire, and testserver/go.mod playhold, at PLAYHOLD_VERSION
	@got=$$(go list -m -f '{{.Version}}' github.com/ny4rl4th0t3p/playhold/wire); \
	if [ "$$got" != "$(PLAYHOLD_VERSION)" ]; then \
		echo "go.mod requires playhold/wire $$got, but PLAYHOLD_VERSION is $(PLAYHOLD_VERSION)"; exit 1; \
	fi
	@got=$$(cd testserver && go list -m -f '{{.Version}}' github.com/ny4rl4th0t3p/playhold); \
	if [ "$$got" != "$(PLAYHOLD_VERSION)" ]; then \
		echo "testserver/go.mod requires playhold $$got, but PLAYHOLD_VERSION is $(PLAYHOLD_VERSION)"; exit 1; \
	fi

fmt: ## Format the code with golangci-lint's formatters
	golangci-lint fmt ./...
	cd testserver && golangci-lint fmt --config ../.golangci.yml ./...

tidy: ## Tidy both modules
	go mod tidy
	cd testserver && go mod tidy