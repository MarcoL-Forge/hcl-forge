BINARY   := hclforge
MAIN     := ./cmd/hclforge
COVERAGE := coverage.txt

.PHONY: all build test coverage fmt vet lint tidy clean

all: fmt vet lint test build

build:
	go build -o $(BINARY) $(MAIN)

test:
	go test ./... -v -race

coverage:
	go test ./... -coverprofile=$(COVERAGE) -covermode=atomic
	go tool cover -func=$(COVERAGE)
	rm -f $(COVERAGE)

fmt:
	gofmt -s -w .

vet:
	go vet ./...

lint:
	golangci-lint run ./...

tidy:
	go mod tidy

clean:
	rm -f $(BINARY) $(COVERAGE)