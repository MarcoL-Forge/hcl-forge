SHELL := /bin/zsh

.PHONY: test test-unit test-integration test-e2e

test: test-unit

test-unit:
	go test ./...

test-integration:
	go test -tags=integration ./...

test-e2e:
	go test -tags=e2e ./cmd/hcl-forge
