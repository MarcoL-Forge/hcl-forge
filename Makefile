SHELL := /bin/zsh

.PHONY: test test-unit test-integration test-e2e test-coverage fmt fmt-check vet lint quality-checks

test: test-unit

test-unit:
	go test ./...

test-integration:
	go test -tags=integration ./...

test-e2e:
	go test -tags=e2e ./cmd/hcl-forge

test-coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out | tail -n 1

fmt:
	gofmt -w .

fmt-check:
	test -z "$(gofmt -l .)"

vet:
	go vet ./...

lint:
	$(go env GOPATH)/bin/golangci-lint run ./...

quality-checks: fmt-check vet lint
