.PHONY: help check-format format lint lint-golangci-lint test

help:
	@echo "Available commands:"
	@echo "  make check-format  - Check if the code is formatted correctly"
	@echo "  make format        - Format the code using gofumpt"
	@echo "  make lint          - Run linters on the code"
	@echo "  make test          - Run tests"

check-format:
	gofumpt -d .

format:
	gofumpt -w .

lint: lint-golangci-lint

lint-golangci-lint:
	@if [ -f .golangci.yml ]; then \
		echo "Running golangci-lint..."; \
		golangci-lint run ./...; \
	else \
		echo "Skipping golangci-lint as .golangci-lint.yml is not present"; \
	fi

test:
	go test -v ./...
