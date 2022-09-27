NO_COLOR=\033[0m
OK_COLOR=\033[32;01m
ERROR_COLOR=\033[31;01m
WARN_COLOR=\033[33;01m

.PHONY: all test lint

all: lint test

test:
	@echo "$(OK_COLOR)==> Running tests$(NO_COLOR)"
	@go test -cover ./... -coverprofile=coverage.txt -covermode=atomic

lint:
	@echo "$(OK_COLOR)==> Linting with golangci-lint$(NO_COLOR)"
	@docker run --rm -v $(PWD):/app -w /app golangci/golangci-lint:v1.49.0 golangci-lint run -v
