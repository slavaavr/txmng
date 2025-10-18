.PHONY: setup test itest lint mocks

setup:
	@which golangci-lint >/dev/null || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@which mockery >/dev/null || go install github.com/vektra/mockery/v3

test:
	go test -count=1 ./... -covermode=atomic -race

itest:
	@echo "Running integration tests using Docker Compose..."
	@./scripts/run-int-tests.sh

lint: setup
	golangci-lint run -c golangci.yml ./...

mocks: setup
	mockery --config .mockery.yml && mockery --config .mockery-pub.yml