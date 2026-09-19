# Metering — basic Go workflow + integration tests.
#
# Usage:
#   make build            # compile ./... into bin/metering
#   make test             # unit tests (excludes integration-tagged tests)
#   make test-integration # CRUD integration tests over bufconn gRPC
#   make test-all         # vet + unit + integration with -race
#   make proto            # regenerate protobuf stubs from common/album.proto
#   make help             # list targets (default)

MODULE       := github.com/ravirajsubramanian/metering
BINARY       := bin/metering
PROTO_SRC    := common/album.proto
INTEGRATION  := ./tests/integration/...
GO           := go
GOTESTSUM    :=

.PHONY: all build run vet fmt fmt-check tidy deps test test-integration \
	test-all test-race cover clean proto tools help

all: build ## Build the binary (default).

help: ## Show this help.
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  %-18s %s\n", $$1, $$2}'

build: ## Compile the server binary into $(BINARY).
	@mkdir -p bin
	$(GO) build -o $(BINARY) .

run: ## Run the gRPC server locally (:50051, or PORT=<port>).
	$(GO) run .

vet: ## go vet everything, including integration-tagged files.
	$(GO) vet ./...
	$(GO) vet -tags integration ./...

fmt: ## Format all Go files in place.
	gofmt -w .

fmt-check: ## Fail if any Go file is unformatted.
	@test -z "$$(gofmt -l .)" || (echo "unformatted files:"; gofmt -l .; exit 1)

tidy: ## Tidy go.mod / go.sum.
	$(GO) mod tidy

deps: ## Download module dependencies.
	$(GO) mod download

test: ## Run unit tests (skips integration-tagged tests).
	$(GO) test ./... -count=1

test-integration: ## Run CRUD integration tests (bufconn gRPC, no external deps).
	$(GO) test -tags integration $(INTEGRATION) -v -count=1

test-race: ## Run unit + integration tests with the race detector.
	$(GO) test ./... -count=1 -race
	$(GO) test -tags integration $(INTEGRATION) -count=1 -race

test-all: vet test test-integration ## Vet + unit + integration tests.

cover: ## Unit-test coverage report (coverage.out + HTML).
	$(GO) test ./... -count=1 -coverprofile=coverage.out
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "wrote coverage.out and coverage.html"

proto: ## Regenerate protobuf stubs from $(PROTO_SRC).
	protoc --go_out=. --go_opt=paths=source_relative \
	       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
	       $(PROTO_SRC)

tools: ## Install protoc plugins for code generation.
	$(GO) install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	$(GO) install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

clean: ## Remove build artifacts and coverage files.
	rm -rf bin/ coverage.out coverage.html
