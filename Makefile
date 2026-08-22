# Makefile for Go Excel Library
# Provides common development tasks

# Variables
GOBASE := $(shell pwd)
GOBIN := $(GOBASE)/bin
GOFILES := $(shell find . -name "*.go" -type f ! -path "./vendor/*")

# Colors for output
GREEN := \033[0;32m
YELLOW := \033[0;33m
RED := \033[0;31m
NC := \033[0m # No Color

.PHONY: help
help: ## Display this help message
	@echo "$(GREEN)Available targets:$(NC)"
	@awk 'BEGIN {FS = ":.*##"; printf "\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2 } /^##@/ { printf "\n$(YELLOW)%s$(NC)\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: install
install: ## Install dependencies
	@echo "$(GREEN)Installing dependencies...$(NC)"
	go mod download
	go mod tidy

.PHONY: fmt
fmt: ## Format code using gofmt
	@echo "$(GREEN)Formatting code...$(NC)"
	gofmt -s -w $(GOFILES)

.PHONY: vet
vet: ## Run go vet
	@echo "$(GREEN)Running go vet...$(NC)"
	go vet ./...

.PHONY: lint
lint: ## Run golangci-lint
	@echo "$(GREEN)Running linter...$(NC)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "$(RED)golangci-lint not installed. Install it from https://golangci-lint.run/usage/install/$(NC)"; \
		exit 1; \
	fi

.PHONY: lint-fix
lint-fix: ## Run golangci-lint with auto-fix
	@echo "$(GREEN)Running linter with auto-fix...$(NC)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --fix; \
	else \
		echo "$(RED)golangci-lint not installed. Install it from https://golangci-lint.run/usage/install/$(NC)"; \
		exit 1; \
	fi

.PHONY: vulncheck
vulncheck: ## Run govulncheck against the module and its dependencies
	@echo "$(GREEN)Running govulncheck...$(NC)"
	@if command -v govulncheck >/dev/null 2>&1; then \
		govulncheck ./...; \
	else \
		echo "$(YELLOW)govulncheck not installed. Install: go install golang.org/x/vuln/cmd/govulncheck@latest$(NC)"; \
		exit 1; \
	fi

##@ Testing

.PHONY: test
test: ## Run tests
	@echo "$(GREEN)Running tests...$(NC)"
	go test -v -race -coverprofile=coverage.out ./...

.PHONY: test-short
test-short: ## Run short tests (skip long-running tests)
	@echo "$(GREEN)Running short tests...$(NC)"
	go test -v -short -race ./...

.PHONY: test-coverage
test-coverage: test ## Run tests with coverage report
	@echo "$(GREEN)Generating coverage report...$(NC)"
	go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report: coverage.html$(NC)"

##@ Benchmarking

.PHONY: bench
bench: ## Run all benchmarks
	@echo "$(GREEN)Running benchmarks...$(NC)"
	go test -bench=. -benchmem -run=^$$ ./...

.PHONY: bench-large
bench-large: ## Run the 1M-row benchmarks (stream package only)
	@echo "$(GREEN)Running large dataset (1M row) benchmarks...$(NC)"
	go test -bench=1M -benchmem -run=^$$ ./stream/...

.PHONY: bench-compare
bench-compare: ## Compare benchmark results (requires benchstat)
	@echo "$(GREEN)Comparing benchmarks...$(NC)"
	@if [ ! -f "old.bench" ]; then \
		echo "$(RED)old.bench not found. Run 'make bench > old.bench' first$(NC)"; \
		exit 1; \
	fi
	@go test -bench=. -benchmem -run=^$$ ./... > new.bench
	@if command -v benchstat >/dev/null 2>&1; then \
		benchstat old.bench new.bench; \
	else \
		echo "$(YELLOW)benchstat not installed. Install: go install golang.org/x/perf/cmd/benchstat@latest$(NC)"; \
	fi

##@ Profiling

.PHONY: profile-cpu
profile-cpu: ## Generate CPU profile
	@echo "$(GREEN)Generating CPU profile...$(NC)"
	go test -cpuprofile=cpu.prof -bench=. -run=^$$ ./...
	@echo "$(GREEN)Analyze with: go tool pprof cpu.prof$(NC)"

.PHONY: profile-mem
profile-mem: ## Generate memory profile
	@echo "$(GREEN)Generating memory profile...$(NC)"
	go test -memprofile=mem.prof -bench=. -run=^$$ ./...
	@echo "$(GREEN)Analyze with: go tool pprof mem.prof$(NC)"

.PHONY: profile-trace
profile-trace: ## Generate execution trace
	@echo "$(GREEN)Generating execution trace...$(NC)"
	go test -trace=trace.out -bench=. -run=^$$ ./...
	@echo "$(GREEN)Analyze with: go tool trace trace.out$(NC)"

##@ Code Quality

.PHONY: check
check: fmt vet lint test ## Run all checks (format, vet, lint, test)
	@echo "$(GREEN)All checks passed!$(NC)"

##@ Build

.PHONY: build
build: ## Build library (verify compilation)
	@echo "$(GREEN)Building library...$(NC)"
	go build -v ./...

.PHONY: clean
clean: ## Clean build artifacts and caches (does not touch the shared module cache)
	@echo "$(GREEN)Cleaning...$(NC)"
	go clean -cache -testcache
	rm -f coverage.out coverage.html
	rm -f cpu.prof mem.prof trace.out
	rm -f old.bench new.bench
	rm -rf $(GOBIN)

##@ Examples

.PHONY: run-example-basic-export
run-example-basic-export: ## Run basic export example
	@echo "$(GREEN)Running basic export example...$(NC)"
	go run examples/basic_export/main.go

.PHONY: run-example-basic-import
run-example-basic-import: ## Run basic import example
	@echo "$(GREEN)Running basic import example...$(NC)"
	go run examples/basic_import/main.go

.PHONY: run-example-csv-format
run-example-csv-format: ## Run CSV format example
	@echo "$(GREEN)Running csv format example...$(NC)"
	go run examples/csv_format/main.go

.PHONY: run-example-events
run-example-events: ## Run event hooks example
	@echo "$(GREEN)Running events example...$(NC)"
	go run examples/events/main.go

.PHONY: run-example-merge-cells
run-example-merge-cells: ## Run merge cells example
	@echo "$(GREEN)Running merge cells example...$(NC)"
	go run examples/merge_cells/main.go

.PHONY: run-example-multisheet
run-example-multisheet: ## Run multi-sheet example
	@echo "$(GREEN)Running multisheet example...$(NC)"
	go run examples/multisheet/main.go

.PHONY: run-example-struct-mapping
run-example-struct-mapping: ## Run struct mapping example
	@echo "$(GREEN)Running struct mapping example...$(NC)"
	go run examples/struct_mapping/main.go

.PHONY: run-example-styling
run-example-styling: ## Run styling example
	@echo "$(GREEN)Running styling example...$(NC)"
	go run examples/styling/main.go

.PHONY: run-example-web-export
run-example-web-export: ## Run web export (net/http) example
	@echo "$(GREEN)Running web export example...$(NC)"
	go run examples/web_export/main.go

.PHONY: run-example-stream-export
run-example-stream-export: ## Run stream export example
	@echo "$(GREEN)Running stream export example...$(NC)"
	cd examples/stream_export && go run main.go

.PHONY: run-example-stream-import
run-example-stream-import: run-example-stream-export ## Run stream import example (runs stream export first to produce its input)
	@echo "$(GREEN)Running stream import example...$(NC)"
	cp examples/stream_export/stream_export_output.xlsx examples/stream_import/
	cd examples/stream_import && go run main.go

##@ Documentation

.PHONY: doc
doc: ## Generate and serve documentation
	@echo "$(GREEN)Starting godoc server at http://localhost:6060$(NC)"
	@if command -v godoc >/dev/null 2>&1; then \
		godoc -http=:6060; \
	else \
		echo "$(YELLOW)godoc not installed. Install: go install golang.org/x/tools/cmd/godoc@latest$(NC)"; \
	fi

##@ CI/CD

.PHONY: ci
ci: install check vulncheck ## Run the same checks as .github/workflows/ci.yml locally (excludes bench, which CI does not run)
	@echo "$(GREEN)CI pipeline completed successfully!$(NC)"

.PHONY: pre-commit
pre-commit: fmt lint test-short ## Run pre-commit checks
	@echo "$(GREEN)Pre-commit checks passed!$(NC)"

##@ Utilities

.PHONY: deps-update
deps-update: ## Update dependencies
	@echo "$(GREEN)Updating dependencies...$(NC)"
	go get -u ./...
	go mod tidy

.PHONY: deps-graph
deps-graph: ## Show dependency graph (requires graphviz)
	@echo "$(GREEN)Generating dependency graph...$(NC)"
	@if command -v dot >/dev/null 2>&1; then \
		go mod graph | modgraphviz | dot -Tsvg -o deps.svg; \
		echo "$(GREEN)Dependency graph: deps.svg$(NC)"; \
	else \
		echo "$(YELLOW)graphviz not installed. Install: brew install graphviz$(NC)"; \
	fi

.PHONY: todo
todo: ## List TODO comments in code
	@echo "$(GREEN)Finding TODO comments...$(NC)"
	@grep -rn "TODO" --include="*.go" . || echo "$(GREEN)No TODOs found!$(NC)"

.PHONY: stats
stats: ## Show code statistics
	@echo "$(GREEN)Code Statistics:$(NC)"
	@echo "Files: $$(find . -name "*.go" ! -path "./vendor/*" | wc -l)"
	@echo "Lines: $$(find . -name "*.go" ! -path "./vendor/*" -exec wc -l {} + | tail -1)"
	@echo "Packages: $$(go list ./... | wc -l)"

# Default target
.DEFAULT_GOAL := help
