.PHONY: setup fmt lint test build check

## setup: install git hooks (run once after cloning)
setup:
	git config core.hooksPath .githooks
	@echo "git hooks installed (.githooks)"

## fmt: format all Go files
fmt:
	goimports -w .
	gofmt -w .

## lint: run golangci-lint over the module
lint:
	golangci-lint run ./...

## test: run the hermetic test suite
test:
	go test ./... -count=1

## build: build the server binary
build:
	go build -o bin/orcadub-mcp-server ./cmd

## check: everything CI runs
check: fmt lint test
	go vet ./...
	go build ./...
	node --check npm/install.js
	node --check npm/bin/run.js
