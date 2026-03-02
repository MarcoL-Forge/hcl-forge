BINARY     := hclforge
MODULE     := github.com/Marc0l95/hclforge
MAIN       := ./cmd/hclforge
COVERAGE   := coverage.txt

.PHONY: all build test lint fmt vet tidy coverage clean

all: fmt vet lint test build

## build: compile the binary
build:
	go build -o $(BINARY) $(MAIN)

## test: run all unit tests
test:
	go test ./... -v -race

## coverage: run tests with coverage report
coverage:
	go test ./... -coverprofile=$(COVERAGE) -covermode=atomic
	go tool cover -func=$(COVERAGE)

## lint: run golangci-lint (must be installed: https://golangci-lint.run/usage/install/)
lint:
	golangci-lint run ./...

## fmt: format all Go source files
fmt:
	gofmt -s -w .
	goimports -w .

## vet: run go vet
vet:
	go vet ./...

## tidy: tidy and verify go.mod / go.sum
tidy:
	go mod tidy
	go mod verify

## clean: remove build artifacts
clean:
	rm -f $(BINARY) $(COVERAGE)